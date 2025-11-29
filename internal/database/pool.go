package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"openvdo/internal/config"
	"openvdo/pkg/logger"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// TenantPool represents a connection pool for a specific tenant/user
type TenantPool struct {
	UserID    uuid.UUID
	OrgID     uuid.UUID
	Role      string
	DB        *sql.DB
	CreatedAt time.Time
	LastUsed  time.Time
}

// PoolManager manages multiple tenant-specific connection pools
type PoolManager struct {
	config       config.Database
	masterDB     *sql.DB
	tenantPools  map[string]*TenantPool // key: userID
	mu           sync.RWMutex
	cleanupTicker *time.Ticker
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewPoolManager creates a new connection pool manager
func NewPoolManager(cfg config.Database) (*PoolManager, error) {
	masterDB, err := createMasterConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create master connection: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	pm := &PoolManager{
		config:      cfg,
		masterDB:    masterDB,
		tenantPools: make(map[string]*TenantPool),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start cleanup routine
	pm.startCleanupRoutine()

	logger.Info("Connection pool manager initialized with %d max tenant pools", cfg.MaxTenantPools)
	return pm, nil
}

// createMasterConnection creates the master database connection with pool configuration
func createMasterConnection(cfg config.Database) (*sql.DB, error) {
	dsn := cfg.DSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	logger.Info("Master database connection established with pool config: MaxOpen=%d, MaxIdle=%d, Lifetime=%v, IdleTime=%v",
		cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetime, cfg.ConnMaxIdleTime)

	return db, nil
}

// GetTenantConnection returns a database connection with RLS context set for the tenant
func (pm *PoolManager) GetTenantConnection(ctx context.Context, userID uuid.UUID) (*sql.Conn, error) {
	pm.mu.RLock()
	pool, exists := pm.tenantPools[userID.String()]
	pm.mu.RUnlock()

	if !exists {
		// Create new tenant pool
		newPool, err := pm.createTenantPool(ctx, userID)
		if err != nil {
			return nil, err
		}
		pool = newPool
	}

	// Update last used time
	pm.mu.Lock()
	pool.LastUsed = time.Now()
	pm.mu.Unlock()

	// Get connection from tenant pool
	conn, err := pool.DB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection from tenant pool: %w", err)
	}

	// Set RLS context for this connection
	if err := setUserContext(ctx, conn, userID); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set user context: %w", err)
	}

	return conn, nil
}

// createTenantPool creates a new connection pool for a specific tenant
func (pm *PoolManager) createTenantPool(ctx context.Context, userID uuid.UUID) (*TenantPool, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if pool was created while waiting for lock
	if pool, exists := pm.tenantPools[userID.String()]; exists {
		return pool, nil
	}

	// Check pool limit
	if len(pm.tenantPools) >= pm.config.MaxTenantPools {
		return nil, fmt.Errorf("maximum tenant pools (%d) reached", pm.config.MaxTenantPools)
	}

	// Get user organization info
	orgID, role, err := pm.getUserOrgInfo(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user org info: %w", err)
	}

	// Create dedicated connection for tenant
	dsn := pm.config.DSN()
	tenantDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open tenant database: %w", err)
	}

	// Configure tenant-specific pool settings (smaller than master)
	tenantDB.SetMaxOpenConns(10)
	tenantDB.SetMaxIdleConns(2)
	tenantDB.SetConnMaxLifetime(30 * time.Minute)
	tenantDB.SetConnMaxIdleTime(5 * time.Minute)

	pool := &TenantPool{
		UserID:    userID,
		OrgID:     orgID,
		Role:      role,
		DB:        tenantDB,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	pm.tenantPools[userID.String()] = pool
	logger.Info("Created tenant pool for user %s (org: %s, role: %s)", userID, orgID, role)

	return pool, nil
}

// getUserOrgInfo retrieves user's organization and role information
func (pm *PoolManager) getUserOrgInfo(ctx context.Context, userID uuid.UUID) (uuid.UUID, string, error) {
	var orgID uuid.UUID
	var role string

	query := `
		SELECT organization_id, role
		FROM user_org_roles
		WHERE user_id = $1
		LIMIT 1
	`

	err := pm.masterDB.QueryRowContext(ctx, query, userID).Scan(&orgID, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return uuid.Nil, "", fmt.Errorf("user not found in any organization")
		}
		return uuid.Nil, "", fmt.Errorf("failed to query user org info: %w", err)
	}

	return orgID, role, nil
}

// setUserContext sets the PostgreSQL RLS user context for the connection
func setUserContext(ctx context.Context, conn *sql.Conn, userID uuid.UUID) error {
	return conn.Raw(func(driverConn interface{}) error {
		if pgConn, ok := driverConn.(interface{ ExecContext(context.Context, string, ...interface{}) (sql.Result, error) }); ok {
			_, err := pgConn.ExecContext(ctx, "SET LOCAL app.current_user_id = $1", userID.String())
			return err
		}
		return fmt.Errorf("failed to cast connection to PostgreSQL driver")
	})
}

// GetMasterConnection returns a connection to the master database (for admin operations)
func (pm *PoolManager) GetMasterConnection() *sql.DB {
	return pm.masterDB
}

// Close closes all connection pools
func (pm *PoolManager) Close() error {
	pm.cancel()

	if pm.cleanupTicker != nil {
		pm.cleanupTicker.Stop()
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	var lastErr error
	for _, pool := range pm.tenantPools {
		if err := pool.DB.Close(); err != nil {
			logger.Error("Failed to close tenant pool: %v", err)
			lastErr = err
		}
	}

	if err := pm.masterDB.Close(); err != nil {
		logger.Error("Failed to close master database: %v", err)
		lastErr = err
	}

	pm.tenantPools = make(map[string]*TenantPool)
	return lastErr
}

// GetStats returns pool statistics
func (pm *PoolManager) GetStats() PoolStats {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := PoolStats{
		TotalTenantPools: len(pm.tenantPools),
		MaxTenantPools:   pm.config.MaxTenantPools,
		MasterStats:      getConnectionStats(pm.masterDB),
	}

	for _, pool := range pm.tenantPools {
		stats.TenantStats = append(stats.TenantStats, TenantPoolStats{
			UserID:    pool.UserID,
			OrgID:     pool.OrgID,
			Role:      pool.Role,
			CreatedAt: pool.CreatedAt,
			LastUsed:  pool.LastUsed,
			Stats:     getConnectionStats(pool.DB),
		})
	}

	return stats
}

// getConnectionStats extracts database statistics
func getConnectionStats(db *sql.DB) ConnectionStats {
	dbStats := db.Stats()
	return ConnectionStats{
		OpenConnections: dbStats.OpenConnections,
		InUse:          dbStats.InUse,
		Idle:           dbStats.Idle,
		WaitCount:      dbStats.WaitCount,
		WaitDuration:   dbStats.WaitDuration,
		MaxIdleClosed:  dbStats.MaxIdleClosed,
		MaxLifetimeClosed: dbStats.MaxLifetimeClosed,
	}
}

// startCleanupRoutine starts a routine to clean up idle tenant pools
func (pm *PoolManager) startCleanupRoutine() {
	pm.cleanupTicker = time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <-pm.ctx.Done():
				return
			case <-pm.cleanupTicker.C:
				pm.cleanupIdlePools()
			}
		}
	}()

	logger.Info("Tenant pool cleanup routine started")
}

// cleanupIdlePools removes tenant pools that haven't been used recently
func (pm *PoolManager) cleanupIdlePools() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	now := time.Now()
	var removedCount int

	for userID, pool := range pm.tenantPools {
		if now.Sub(pool.LastUsed) > pm.config.PoolIdleTimeout {
			if err := pool.DB.Close(); err != nil {
				logger.Error("Failed to close idle tenant pool for user %s: %v", userID, err)
			}
			delete(pm.tenantPools, userID)
			removedCount++
			logger.Debug("Cleaned up idle tenant pool for user %s", userID)
		}
	}

	if removedCount > 0 {
		logger.Info("Cleaned up %d idle tenant pools", removedCount)
	}
}