package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// StatelessTenantDB represents a database connection with tenant context (stateless version)
type StatelessTenantDB struct {
	conn     *sql.Conn
	userID   uuid.UUID
	pool     *StatelessPoolManager
	released bool
}

// NewTenantDB creates a new tenant-aware database connection (stateless version)
func (spm *StatelessPoolManager) NewTenantDB(ctx context.Context, userID uuid.UUID) (*StatelessTenantDB, error) {
	conn, err := spm.GetTenantConnection(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &StatelessTenantDB{
		conn:   conn,
		userID: userID,
		pool:   spm,
	}, nil
}

// ExecContext executes a query without returning rows
func (t *StatelessTenantDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if t.released {
		return nil, fmt.Errorf("connection has been released")
	}
	return t.conn.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows
func (t *StatelessTenantDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if t.released {
		return nil, fmt.Errorf("connection has been released")
	}
	return t.conn.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns a single row
func (t *StatelessTenantDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if t.released {
		// Return a row that will error on any operation
		return &sql.Row{}
	}
	return t.conn.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a transaction with tenant context
func (t *StatelessTenantDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if t.released {
		return nil, fmt.Errorf("connection has been released")
	}
	return t.conn.BeginTx(ctx, opts)
}

// Ping checks if the database connection is alive
func (t *StatelessTenantDB) Ping(ctx context.Context) error {
	if t.released {
		return fmt.Errorf("connection has been released")
	}
	// Use the underlying database to ping
	return t.pool.masterDB.PingContext(ctx)
}

// Release returns the connection to the pool with context cleanup
func (t *StatelessTenantDB) Release() error {
	if t.released {
		return nil
	}

	t.released = true
	return t.pool.ReleaseConnection(t.conn)
}

// GetUserID returns the user ID for this tenant connection
func (t *StatelessTenantDB) GetUserID() uuid.UUID {
	return t.userID
}

// GetUserSession returns cached user session information
func (t *StatelessTenantDB) GetUserSession(ctx context.Context) (*UserSession, error) {
	return t.pool.GetUserSession(ctx, t.userID)
}

// WithTransaction executes a function within a transaction
func (t *StatelessTenantDB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
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

// StatelessTenantOperations provides high-level operations for tenant data
type StatelessTenantOperations struct {
	spm *StatelessPoolManager
}

// NewStatelessTenantOperations creates a new stateless tenant operations instance
func NewStatelessTenantOperations(spm *StatelessPoolManager) *StatelessTenantOperations {
	return &StatelessTenantOperations{spm: spm}
}

// HasRole checks if a user has a specific role in an organization
func (sto *StatelessTenantOperations) HasRole(ctx context.Context, userID, orgID uuid.UUID, role string) (bool, error) {
	session, err := sto.spm.GetUserSession(ctx, userID)
	if err != nil {
		return false, err
	}

	// Check if session is valid
	if time.Now().After(session.ExpiresAt) {
		// Invalidate expired session
		sto.spm.InvalidateUserSession(ctx, userID)
		return false, fmt.Errorf("session expired")
	}

	// Check organization match
	if session.OrgID != orgID {
		return false, nil
	}

	// Check role (if specified)
	if role != "" && session.Role != role {
		return false, nil
	}

	return true, nil
}

// GetUserOrganizations returns all organizations for a user
func (sto *StatelessTenantOperations) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]OrganizationInfo, error) {
	conn, err := sto.spm.GetTenantConnection(ctx, userID)
	if err != nil {
		return nil, err
	}
	defer sto.spm.ReleaseConnection(conn)

	query := `
		SELECT o.id, o.name, o.description, o.created_at, o.updated_at, uor.role
		FROM organizations o
		JOIN user_org_roles uor ON o.id = uor.organization_id
		WHERE uor.user_id = $1
		ORDER BY o.created_at DESC
	`

	rows, err := conn.QueryContext(ctx, query, userID)
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

// InvalidateUserSession removes cached session for a user (useful after role changes)
func (sto *StatelessTenantOperations) InvalidateUserSession(ctx context.Context, userID uuid.UUID) error {
	return sto.spm.InvalidateUserSession(ctx, userID)
}

// PreloadUserSession preloads session data for active users
func (sto *StatelessTenantOperations) PreloadUserSession(ctx context.Context, userID uuid.UUID) error {
	// Force session loading by calling GetUserSession
	_, err := sto.spm.GetUserSession(ctx, userID)
	return err
}

// BatchPreloadUserSessions preloads sessions for multiple users efficiently
func (sto *StatelessTenantOperations) BatchPreloadUserSessions(ctx context.Context, userIDs []uuid.UUID) error {
	log.Printf("INFO: Preloading sessions for %d users", len(userIDs))

	successCount := 0
	for _, userID := range userIDs {
		if err := sto.PreloadUserSession(ctx, userID); err != nil {
			log.Printf("WARN: Failed to preload session for user %s: %v", userID, err)
		} else {
			successCount++
		}
	}

	log.Printf("INFO: Successfully preloaded %d out of %d user sessions", successCount, len(userIDs))
	return nil
}