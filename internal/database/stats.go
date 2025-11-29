package database

import (
	"time"

	"github.com/google/uuid"
)

// PoolStats contains comprehensive statistics about the connection pool manager
type PoolStats struct {
	TotalTenantPools int                `json:"total_tenant_pools"`
	MaxTenantPools   int                `json:"max_tenant_pools"`
	MasterStats      ConnectionStats    `json:"master_stats"`
	TenantStats      []TenantPoolStats  `json:"tenant_stats"`
	LastCleanup      time.Time          `json:"last_cleanup"`
}

// TenantPoolStats contains statistics for a specific tenant pool
type TenantPoolStats struct {
	UserID    uuid.UUID       `json:"user_id"`
	OrgID     uuid.UUID       `json:"org_id"`
	Role      string          `json:"role"`
	CreatedAt time.Time       `json:"created_at"`
	LastUsed  time.Time       `json:"last_used"`
	Stats     ConnectionStats `json:"stats"`
}

// ConnectionStats contains database connection statistics
type ConnectionStats struct {
	OpenConnections     int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}

// HealthStatus represents the health status of the pool manager
type HealthStatus struct {
	Healthy           bool          `json:"healthy"`
	MasterHealthy     bool          `json:"master_healthy"`
	RedisHealthy      bool          `json:"redis_healthy"`
	TotalConnections  int           `json:"total_connections"`
	TenantPoolsActive int           `json:"tenant_pools_active"`
	Timestamp         time.Time     `json:"timestamp"`
	Errors            []string      `json:"errors,omitempty"`
	LastCheck         time.Time     `json:"last_check"`
	CheckInterval     time.Duration `json:"check_interval"`
	PoolType          string        `json:"pool_type"` // "stateless" or "stateful"
}

// GetHealth returns the current health status of the pool manager
func (pm *PoolManager) GetHealth() HealthStatus {
	status := HealthStatus{
		Timestamp:     time.Now(),
		CheckInterval: 30 * time.Second,
	}

	// Check master database health
	if err := pm.masterDB.Ping(); err != nil {
		status.MasterHealthy = false
		status.Healthy = false
		status.Errors = append(status.Errors, "Master database ping failed: "+err.Error())
	} else {
		status.MasterHealthy = true
	}

	// Count total connections
	pm.mu.RLock()
	status.TenantPoolsActive = len(pm.tenantPools)
	for _, pool := range pm.tenantPools {
		stats := pool.DB.Stats()
		status.TotalConnections += stats.OpenConnections
	}
	pm.mu.RUnlock()

	// Add master connections
	masterStats := pm.masterDB.Stats()
	status.TotalConnections += masterStats.OpenConnections

	// Consider unhealthy if too many connections
	if status.TotalConnections > pm.config.MaxOpenConns+pm.config.MaxTenantPools*10 {
		status.Healthy = false
		status.Errors = append(status.Errors, "Too many open connections")
	}

	return status
}