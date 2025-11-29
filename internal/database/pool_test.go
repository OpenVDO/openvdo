package database

import (
	"context"
	"testing"
	"time"

	"openvdo/internal/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolManagerCreation(t *testing.T) {
	cfg := config.Database{
		Host:            "localhost",
		Port:            "5432",
		User:            "openvdo",
		Password:        "openvdo",
		Name:            "openvdo",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxTenantPools:  10,
		ConnMaxLifetime: 5 * time.Minute,
		PoolIdleTimeout: 10 * time.Minute,
	}

	pm, err := NewPoolManager(cfg)
	require.NoError(t, err)
	assert.NotNil(t, pm)

	// Test stats
	stats := pm.GetStats()
	assert.Equal(t, 0, stats.TotalTenantPools)
	assert.Equal(t, 10, stats.MaxTenantPools)

	err = pm.Close()
	require.NoError(t, err)
}

func TestTenantConnection(t *testing.T) {
	// This test requires a running database
	cfg := config.Database{
		Host:            "localhost",
		Port:            "5432",
		User:            "openvdo",
		Password:        "openvdo",
		Name:            "openvdo",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxTenantPools:  10,
		ConnMaxLifetime: 5 * time.Minute,
		PoolIdleTimeout: 10 * time.Minute,
	}

	pm, err := NewPoolManager(cfg)
	require.NoError(t, err)
	defer pm.Close()

	// Use the admin user we created earlier
	userID, err := uuid.Parse("USER_ID_FROM_ADMIN_USER") // Replace with actual user ID
	if err != nil {
		t.Skip("No valid user ID available for testing")
		return
	}

	ctx := context.Background()
	conn, err := pm.GetTenantConnection(ctx, userID)
	require.NoError(t, err)
	assert.NotNil(t, conn)

	// Test that the connection works
	err = conn.Ping(ctx)
	assert.NoError(t, err)

	err = conn.Close()
	require.NoError(t, err)

	// Check that a pool was created
	stats := pm.GetStats()
	assert.Equal(t, 1, stats.TotalTenantPools)
}

func TestPoolHealth(t *testing.T) {
	cfg := config.Database{
		Host:            "localhost",
		Port:            "5432",
		User:            "openvdo",
		Password:        "openvdo",
		Name:            "openvdo",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		MaxTenantPools:  10,
		ConnMaxLifetime: 5 * time.Minute,
		PoolIdleTimeout: 10 * time.Minute,
	}

	pm, err := NewPoolManager(cfg)
	require.NoError(t, err)
	defer pm.Close()

	health := pm.GetHealth()
	assert.True(t, health.Healthy)
	assert.True(t, health.MasterHealthy)
	assert.Greater(t, health.Timestamp.Unix(), int64(0))
}

func BenchmarkTenantConnectionCreation(b *testing.B) {
	cfg := config.Database{
		Host:            "localhost",
		Port:            "5432",
		User:            "openvdo",
		Password:        "openvdo",
		Name:            "openvdo",
		SSLMode:         "disable",
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		MaxTenantPools:  100,
		ConnMaxLifetime: 5 * time.Minute,
		PoolIdleTimeout: 10 * time.Minute,
	}

	pm, err := NewPoolManager(cfg)
	require.NoError(b, err)
	defer pm.Close()

	userID := uuid.New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := pm.GetTenantConnection(ctx, userID)
		if err != nil {
			b.Fatal(err)
		}
		conn.Close()
	}
}

func BenchmarkTenantQueryExecution(b *testing.B) {
	cfg := config.Database{
		Host:            "localhost",
		Port:            "5432",
		User:            "openvdo",
		Password:        "openvdo",
		Name:            "openvdo",
		SSLMode:         "disable",
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		MaxTenantPools:  100,
		ConnMaxLifetime: 5 * time.Minute,
		PoolIdleTimeout: 10 * time.Minute,
	}

	pm, err := NewPoolManager(cfg)
	require.NoError(b, err)
	defer pm.Close()

	// Use existing user ID
	userID, err := uuid.Parse("USER_ID_FROM_ADMIN_USER") // Replace with actual user ID
	if err != nil {
		b.Skip("No valid user ID available for benchmarking")
		return
	}

	tenantDB, err := pm.NewTenantDB(context.Background(), userID)
	require.NoError(b, err)
	defer tenantDB.Release()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := tenantDB.QueryContext(ctx, "SELECT COUNT(*) FROM organizations")
		if err != nil {
			b.Fatal(err)
		}
		rows.Close()
	}
}

// Example of how to use the pool manager
func ExamplePoolManager() {
	cfg := config.Database{
		Host:            "localhost",
		Port:            "5432",
		User:            "openvdo",
		Password:        "openvdo",
		Name:            "openvdo",
		SSLMode:         "disable",
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		MaxTenantPools:  50,
		ConnMaxLifetime: 5 * time.Minute,
		PoolIdleTimeout: 10 * time.Minute,
	}

	// Initialize pool manager
	pm, err := NewPoolManager(cfg)
	if err != nil {
		panic(err)
	}
	defer pm.Close()

	// Get a tenant connection
	userID := uuid.New()
	ctx := context.Background()

	tenantDB, err := pm.NewTenantDB(ctx, userID)
	if err != nil {
		panic(err)
	}
	defer tenantDB.Release()

	// Use the tenant database connection
	rows, err := tenantDB.QueryContext(ctx, "SELECT * FROM organizations")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	// Process results...
	_ = rows

	// Get pool statistics
	stats := pm.GetStats()
	_ = stats

	// Check pool health
	health := pm.GetHealth()
	_ = health
}