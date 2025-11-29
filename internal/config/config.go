package config

import (
	"fmt"
	"os"
	"time"
)

type Database struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string

	// Connection Pool Configuration
	MaxOpenConns    int           `default:"100"`
	MaxIdleConns    int           `default:"10"`
	ConnMaxLifetime time.Duration `default:"5m"`
	ConnMaxIdleTime time.Duration `default:"30s"`

	// Tenant Pool Configuration
	MaxTenantPools  int           `default:"50"`
	PoolIdleTimeout time.Duration `default:"10m"`
}

type Redis struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type Config struct {
	Database Database
	Redis    Redis
}

func Load() *Config {
	return &Config{
		Database: Database{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "openvdo"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),

			// Connection Pool Configuration
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 30*time.Second),

			// Tenant Pool Configuration
			MaxTenantPools:  getEnvAsInt("DB_MAX_TENANT_POOLS", 50),
			PoolIdleTimeout: getEnvAsDuration("DB_POOL_IDLE_TIMEOUT", 10*time.Minute),
		},
		Redis: Redis{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
	}
}

func (d *Database) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode)
}

func (r *Redis) Address() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue := parseInt(value); intValue != 0 {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func parseInt(s string) int {
	var result int
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			return 0
		}
	}
	return result
}