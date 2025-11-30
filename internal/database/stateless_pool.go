package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"openvdo/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// StatelessPoolManager manages a single shared connection pool with dynamic context switching
type StatelessPoolManager struct {
	masterDB *sql.DB
	redis    *redis.Client
	config   config.Database
	mu       sync.RWMutex

	// Metrics
	metrics PoolMetrics
}

// PoolMetrics tracks connection pool statistics
type PoolMetrics struct {
	TotalConnections     int64     `json:"total_connections"`
	ActiveConnections    int64     `json:"active_connections"`
	ContextSwitches      int64     `json:"context_switches"`
	RedisCacheHits       int64     `json:"redis_cache_hits"`
	RedisCacheMisses     int64     `json:"redis_cache_misses"`
	AverageResponseTime  time.Duration `json:"average_response_time"`
	LastReset           time.Time `json:"last_reset"`
}

// UserSession represents cached user session data
type UserSession struct {
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewStatelessPoolManager creates a new stateless connection pool manager
func NewStatelessPoolManager(cfg config.Database, redisClient *redis.Client) (*StatelessPoolManager, error) {
	masterDB, err := createMasterConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create master connection: %w", err)
	}

	spm := &StatelessPoolManager{
		masterDB: masterDB,
		redis:    redisClient,
		config:   cfg,
		metrics: PoolMetrics{
			LastReset: time.Now(),
		},
	}

	log.Println("INFO: Stateless connection pool manager initialized")
	return spm, nil
}

// GetTenantConnection returns a database connection with RLS context dynamically set
func (spm *StatelessPoolManager) GetTenantConnection(ctx context.Context, userID uuid.UUID) (*sql.Conn, error) {
	start := time.Now()

	// Get connection from shared pool
	conn, err := spm.masterDB.Conn(ctx)
	if err != nil {
		spm.recordError()
		return nil, fmt.Errorf("failed to get connection from pool: %w", err)
	}

	// Set RLS context dynamically
	if err := spm.setUserContext(ctx, conn, userID); err != nil {
		conn.Close()
		spm.recordError()
		return nil, fmt.Errorf("failed to set user context: %w", err)
	}

	spm.recordMetrics(start)
	return conn, nil
}

// setUserContext sets the PostgreSQL RLS user context for the connection
func (spm *StatelessPoolManager) setUserContext(ctx context.Context, conn *sql.Conn, userID uuid.UUID) error {
	return conn.Raw(func(driverConn interface{}) error {
		if pgConn, ok := driverConn.(interface{
			ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
		}); ok {
			// Set the user context for RLS
			_, err := pgConn.ExecContext(ctx, "SET LOCAL app.current_user_id = $1", userID.String())
			if err != nil {
				return fmt.Errorf("failed to set RLS context: %w", err)
			}

			// Optionally set additional context variables for more granular RLS
			_, err = pgConn.ExecContext(ctx, "SET LOCAL app.request_timestamp = $1", time.Now().Format(time.RFC3339))
			if err != nil {
				// Log error but don't fail the connection setup
			}

			return nil
		}
		return fmt.Errorf("failed to cast connection to PostgreSQL driver")
	})
}

// ReleaseConnection returns connection to shared pool with context cleanup
func (spm *StatelessPoolManager) ReleaseConnection(conn *sql.Conn) error {
	if conn == nil {
		return nil
	}

	// Reset connection context to prevent contamination
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := conn.Raw(func(driverConn interface{}) error {
		if pgConn, ok := driverConn.(interface{
			ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
		}); ok {
			// Reset all session variables
			_, err := pgConn.ExecContext(ctx, "RESET ALL")
			if err != nil {
				log.Printf("WARN: Failed to reset connection context: %v", err)
			}

			// Optionally reset search_path to default
			_, err = pgConn.ExecContext(ctx, "SET search_path TO public")
			if err != nil {
				log.Printf("WARN: Failed to reset search_path: %v", err)
			}

			return nil
		}
		return fmt.Errorf("failed to cast connection to PostgreSQL driver")
	})

	// Close connection to return it to pool
	closeErr := conn.Close()

	if err != nil {
		return fmt.Errorf("context reset error: %w, close error: %w", err, closeErr)
	}

	return closeErr
}

// GetUserSession retrieves user session data from cache or database
func (spm *StatelessPoolManager) GetUserSession(ctx context.Context, userID uuid.UUID) (*UserSession, error) {
	// Try Redis cache first
	if spm.redis != nil {
		cached, err := spm.getUserSessionFromCache(ctx, userID)
		if err == nil && cached != nil {
			spm.metrics.RedisCacheHits++
			return cached, nil
		}
		spm.metrics.RedisCacheMisses++
	}

	// Fallback to database
	session, err := spm.getUserSessionFromDB(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if spm.redis != nil {
		spm.cacheUserSession(ctx, session)
	}

	return session, nil
}

// getUserSessionFromCache retrieves user session from Redis
func (spm *StatelessPoolManager) getUserSessionFromCache(ctx context.Context, userID uuid.UUID) (*UserSession, error) {
	if spm.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	key := fmt.Sprintf("user:session:%s", userID.String())
	data, err := spm.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found in cache")
		}
		return nil, fmt.Errorf("redis error: %w", err)
	}

	var session UserSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		spm.redis.Del(ctx, key)
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// getUserSessionFromDB retrieves user session from database
func (spm *StatelessPoolManager) getUserSessionFromDB(ctx context.Context, userID uuid.UUID) (*UserSession, error) {
	conn, err := spm.masterDB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer spm.ReleaseConnection(conn)

	query := `
		SELECT uor.organization_id, uor.role
		FROM user_org_roles uor
		WHERE uor.user_id = $1
		ORDER BY uor.created_at DESC
		LIMIT 1
	`

	var orgID uuid.UUID
	var role string
	err = conn.QueryRowContext(ctx, query, userID).Scan(&orgID, &role)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found in any organization")
		}
		return nil, fmt.Errorf("failed to query user session: %w", err)
	}

	return &UserSession{
		UserID:    userID,
		OrgID:     orgID,
		Role:      role,
		ExpiresAt: time.Now().Add(30 * time.Minute), // Cache for 30 minutes
	}, nil
}

// cacheUserSession caches user session in Redis
func (spm *StatelessPoolManager) cacheUserSession(ctx context.Context, session *UserSession) error {
	if spm.redis == nil {
		return nil
	}

	key := fmt.Sprintf("user:session:%s", session.UserID.String())
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	return spm.redis.Set(ctx, key, data, 30*time.Minute).Err()
}

// InvalidateUserSession removes user session from cache
func (spm *StatelessPoolManager) InvalidateUserSession(ctx context.Context, userID uuid.UUID) error {
	if spm.redis == nil {
		return nil
	}

	key := fmt.Sprintf("user:session:%s", userID.String())
	return spm.redis.Del(ctx, key).Err()
}

// GetMasterConnection returns the master database connection (for admin operations)
func (spm *StatelessPoolManager) GetMasterConnection() *sql.DB {
	return spm.masterDB
}

// GetMetrics returns current pool metrics
func (spm *StatelessPoolManager) GetMetrics() PoolMetrics {
	spm.mu.RLock()
	defer spm.mu.RUnlock()

	// Get current connection stats from the pool
	dbStats := spm.masterDB.Stats()

	metrics := spm.metrics
	metrics.TotalConnections = int64(dbStats.OpenConnections)
	metrics.ActiveConnections = int64(dbStats.InUse)

	return metrics
}

// GetHealth returns the health status of the connection pool
func (spm *StatelessPoolManager) GetHealth() HealthStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := HealthStatus{
		Healthy:     true,
		Timestamp:   time.Now(),
		LastCheck:   time.Now(),
		CheckInterval: 30 * time.Second,
	}

	// Check master database health
	if err := spm.masterDB.PingContext(ctx); err != nil {
		status.MasterHealthy = false
		status.Healthy = false
		status.Errors = append(status.Errors, "Master database ping failed: "+err.Error())
	} else {
		status.MasterHealthy = true
	}

	// Check Redis health if available
	if spm.redis != nil {
		if err := spm.redis.Ping(ctx).Err(); err != nil {
			status.RedisHealthy = false
			status.Healthy = false
			status.Errors = append(status.Errors, "Redis ping failed: "+err.Error())
		} else {
			status.RedisHealthy = true
		}
	} else {
		status.RedisHealthy = false
		status.Errors = append(status.Errors, "Redis client not initialized")
	}

	// Check connection pool health
	metrics := spm.GetMetrics()
	status.TotalConnections = int(metrics.TotalConnections)

	// Consider unhealthy if too many connections
	maxConnections := spm.config.MaxOpenConns
	if status.TotalConnections > maxConnections {
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("Too many open connections: %d > %d", status.TotalConnections, maxConnections))
	}

	return status
}

// Close closes the connection pool and cleans up resources
func (spm *StatelessPoolManager) Close() error {
	spm.mu.Lock()
	defer spm.mu.Unlock()

	var lastErr error

	// Close database connection
	if err := spm.masterDB.Close(); err != nil {
		log.Printf("ERROR: Failed to close database connection: %v", err)
		lastErr = err
	}

	// Close Redis connection if available
	if spm.redis != nil {
		if err := spm.redis.Close(); err != nil {
			log.Printf("ERROR: Failed to close Redis connection: %v", err)
			lastErr = err
		}
	}

	log.Println("INFO: Stateless connection pool manager closed")
	return lastErr
}

// recordMetrics updates performance metrics
func (spm *StatelessPoolManager) recordMetrics(start time.Time) {
	spm.mu.Lock()
	defer spm.mu.Unlock()

	duration := time.Since(start)
	spm.metrics.ContextSwitches++

	// Calculate rolling average
	if spm.metrics.AverageResponseTime == 0 {
		spm.metrics.AverageResponseTime = duration
	} else {
		// Simple moving average
		spm.metrics.AverageResponseTime = (spm.metrics.AverageResponseTime + duration) / 2
	}
}

// recordError records an error occurrence
func (spm *StatelessPoolManager) recordError() {
	spm.mu.Lock()
	defer spm.mu.Unlock()

	// Could add error rate tracking here
	log.Printf("DEBUG: Connection pool error recorded")
}

// ResetMetrics resets all metrics
func (spm *StatelessPoolManager) ResetMetrics() {
	spm.mu.Lock()
	defer spm.mu.Unlock()

	spm.metrics = PoolMetrics{
		LastReset: time.Now(),
	}
}