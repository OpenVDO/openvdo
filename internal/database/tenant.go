package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"openvdo/pkg/logger"

	"github.com/google/uuid"
)

// TenantDB represents a database connection with tenant context
type TenantDB struct {
	conn     *sql.Conn
	userID   uuid.UUID
	pool     *PoolManager
	released bool
}

// NewTenantDB creates a new tenant-aware database connection
func (pm *PoolManager) NewTenantDB(ctx context.Context, userID uuid.UUID) (*TenantDB, error) {
	conn, err := pm.GetTenantConnection(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &TenantDB{
		conn:   conn,
		userID: userID,
		pool:   pm,
	}, nil
}

// ExecContext executes a query without returning rows
func (t *TenantDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if t.released {
		return nil, fmt.Errorf("connection has been released")
	}
	return t.conn.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows
func (t *TenantDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if t.released {
		return nil, fmt.Errorf("connection has been released")
	}
	return t.conn.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns a single row
func (t *TenantDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if t.released {
		// Return a row that will error on any operation
		return &sql.Row{}
	}
	return t.conn.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a transaction with tenant context
func (t *TenantDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if t.released {
		return nil, fmt.Errorf("connection has been released")
	}
	return t.conn.BeginTx(ctx, opts)
}

// Ping checks if the database connection is alive
func (t *TenantDB) Ping(ctx context.Context) error {
	if t.released {
		return fmt.Errorf("connection has been released")
	}
	// Use the underlying database to ping
	return t.pool.masterDB.PingContext(ctx)
}

// Release returns the connection to the pool
func (t *TenantDB) Release() error {
	if t.released {
		return nil
	}

	t.released = true
	return t.conn.Close()
}

// GetUserID returns the user ID for this tenant connection
func (t *TenantDB) GetUserID() uuid.UUID {
	return t.userID
}

// WithTransaction executes a function within a transaction
func (t *TenantDB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := t.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// TenantQueryBuilder helps build tenant-aware queries
type TenantQueryBuilder struct {
	baseQuery string
	args      []interface{}
}

// NewTenantQueryBuilder creates a new query builder
func NewTenantQueryBuilder(baseQuery string) *TenantQueryBuilder {
	return &TenantQueryBuilder{
		baseQuery: baseQuery,
		args:      make([]interface{}, 0),
	}
}

// Where adds a WHERE clause
func (tqb *TenantQueryBuilder) Where(condition string, args ...interface{}) *TenantQueryBuilder {
	if len(tqb.args) == 0 {
		tqb.baseQuery += " WHERE " + condition
	} else {
		tqb.baseQuery += " AND " + condition
	}
	tqb.args = append(tqb.args, args...)
	return tqb
}

// OrderBy adds an ORDER BY clause
func (tqb *TenantQueryBuilder) OrderBy(orderBy string) *TenantQueryBuilder {
	tqb.baseQuery += " ORDER BY " + orderBy
	return tqb
}

// Limit adds a LIMIT clause
func (tqb *TenantQueryBuilder) Limit(limit int) *TenantQueryBuilder {
	tqb.baseQuery += " LIMIT " + fmt.Sprintf("%d", limit)
	return tqb
}

// Offset adds an OFFSET clause
func (tqb *TenantQueryBuilder) Offset(offset int) *TenantQueryBuilder {
	tqb.baseQuery += " OFFSET " + fmt.Sprintf("%d", offset)
	return tqb
}

// Build returns the final query and arguments
func (tqb *TenantQueryBuilder) Build() (string, []interface{}) {
	return tqb.baseQuery, tqb.args
}

// TenantOperations provides high-level operations for tenant data
type TenantOperations struct {
	pm *PoolManager
}

// NewTenantOperations creates a new tenant operations instance
func NewTenantOperations(pm *PoolManager) *TenantOperations {
	return &TenantOperations{pm: pm}
}

// CreateUserOrganization creates a new user-organization relationship
func (to *TenantOperations) CreateUserOrganization(ctx context.Context, userID, orgID uuid.UUID, role string) error {
	query := `
		INSERT INTO user_org_roles (user_id, organization_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, organization_id)
		DO UPDATE SET role = $3, updated_at = NOW()
	`

	_, err := to.pm.masterDB.ExecContext(ctx, query, userID, orgID, role)
	return err
}

// HasRole checks if a user has a specific role in an organization
func (to *TenantOperations) HasRole(ctx context.Context, userID, orgID uuid.UUID, role string) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM user_org_roles
		WHERE user_id = $1 AND organization_id = $2
	`

	args := []interface{}{userID, orgID}
	if role != "" {
		query += " AND role = $3"
		args = append(args, role)
	}

	err := to.pm.masterDB.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetUserOrganizations returns all organizations for a user
func (to *TenantOperations) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]OrganizationInfo, error) {
	query := `
		SELECT o.id, o.name, o.description, o.created_at, o.updated_at, uor.role
		FROM organizations o
		JOIN user_org_roles uor ON o.id = uor.organization_id
		WHERE uor.user_id = $1
		ORDER BY o.created_at DESC
	`

	rows, err := to.pm.masterDB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []OrganizationInfo
	for rows.Next() {
		var org OrganizationInfo
		if err := rows.Scan(
			&org.ID, &org.Name, &org.Description,
			&org.CreatedAt, &org.UpdatedAt, &org.Role,
		); err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}

	return orgs, rows.Err()
}

// OrganizationInfo represents organization information with user role
type OrganizationInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Role        string    `json:"role"`
}

// InvalidateUserPools removes all pools for a specific user (useful after role changes)
func (pm *PoolManager) InvalidateUserPools(userID uuid.UUID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pool, exists := pm.tenantPools[userID.String()]; exists {
		if err := pool.DB.Close(); err != nil {
			logger.Error("Failed to close user pool during invalidation: %v", err)
		}
		delete(pm.tenantPools, userID.String())
		logger.Info("Invalidated pools for user %s due to role change", userID)
	}
}

// PreloadTenantPools preloads pools for active users (useful during startup)
func (pm *PoolManager) PreloadTenantPools(ctx context.Context, userIDs []uuid.UUID) error {
	successCount := 0
	for _, userID := range userIDs {
		if _, err := pm.createTenantPool(ctx, userID); err != nil {
			// Failed to preload pool for user
		} else {
			successCount++
		}
	}

	return nil
}