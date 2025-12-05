package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"openvdo/internal/config"
	"openvdo/pkg/logger"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var PoolManagerInstance *StatelessPoolManager

func InitPoolManager(dbConfig config.Database, redisConfig config.Redis) error {
	pm, err := NewStatelessPoolManager(dbConfig, ConnectRedis(redisConfig))
	if err != nil {
		return fmt.Errorf("failed to initialize stateless pool manager: %w", err)
	}
	PoolManagerInstance = pm
	return nil
}

func ClosePoolManager() error {
	if PoolManagerInstance != nil {
		return PoolManagerInstance.Close()
	}
	return nil
}

func GetPoolManager() *StatelessPoolManager {
	return PoolManagerInstance
}

func Connect(cfg config.Database) (*sql.DB, error) {
	dsn := cfg.DSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	return db, nil
}

func Close(db *sql.DB) {
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Error("Error closing database connection: %v", err)
		}
	}
}

func ConnectRedis(cfg config.Redis) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Address(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		PoolSize:     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect to Redis: %v", err)
	} else {
		logger.Info("Redis connection established")
	}

	return client
}

func CloseRedis(client *redis.Client) {
	if client != nil {
		if err := client.Close(); err != nil {
			logger.Error("Error closing Redis connection: %v", err)
		}
	}
}

func GetTenantDB(ctx context.Context, userID string) (*StatelessTenantDB, error) {
	if PoolManagerInstance == nil {
		return nil, fmt.Errorf("pool manager not initialized")
	}

	userUUID, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return PoolManagerInstance.NewTenantDB(ctx, userUUID)
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
