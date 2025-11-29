package database

import (
	"database/sql"
	"fmt"
	"time"

	"openvdo/internal/config"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

func Connect(cfg config.Database) (*sql.DB, error) {
	dsn := cfg.DSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func Close(db *sql.DB) {
	if db != nil {
		db.Close()
	}
}

func ConnectRedis(cfg config.Redis) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Address(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

func CloseRedis(client *redis.Client) {
	if client != nil {
		client.Close()
	}
}