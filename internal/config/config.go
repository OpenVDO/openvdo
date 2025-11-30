package config

import (
	"fmt"
	"os"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
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
	k := koanf.New(".")

	// Load configuration file if exists
	if _, err := os.Stat("config.yaml"); err == nil {
		if err := k.Load(file.Provider("config.yaml"), yaml.Parser()); err != nil {
			fmt.Printf("Warning: Could not load config.yaml: %v\n", err)
		}
	}

	// Load environment variables
	if err := k.Load(env.Provider("", ".", nil), nil); err != nil {
		fmt.Printf("Warning: Could not load environment variables: %v\n", err)
	}

	// Build config using koanf with fallback to environment variables and defaults
	return &Config{
		Database: Database{
			Host:     getEnvWithKoanf(k, "DB_HOST", "DB_HOST", "localhost"),
			Port:     getEnvWithKoanf(k, "DB_PORT", "DB_PORT", "5432"),
			User:     getEnvWithKoanf(k, "DB_USER", "DB_USER", "postgres"),
			Password: getEnvWithKoanf(k, "DB_PASSWORD", "DB_PASSWORD", ""),
			Name:     getEnvWithKoanf(k, "DB_NAME", "DB_NAME", "openvdo"),
			SSLMode:  getEnvWithKoanf(k, "DB_SSLMODE", "DB_SSLMODE", "disable"),

			// Connection Pool Configuration
			MaxOpenConns:    getIntWithKoanf(k, "DB_MAX_OPEN_CONNS", "DB_MAX_OPEN_CONNS", 100),
			MaxIdleConns:    getIntWithKoanf(k, "DB_MAX_IDLE_CONNS", "DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getDurationWithKoanf(k, "DB_CONN_MAX_LIFETIME", "DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getDurationWithKoanf(k, "DB_CONN_MAX_IDLE_TIME", "DB_CONN_MAX_IDLE_TIME", 30*time.Second),

			// Tenant Pool Configuration
			MaxTenantPools:  getIntWithKoanf(k, "DB_MAX_TENANT_POOLS", "DB_MAX_TENANT_POOLS", 50),
			PoolIdleTimeout: getDurationWithKoanf(k, "DB_POOL_IDLE_TIMEOUT", "DB_POOL_IDLE_TIMEOUT", 10*time.Minute),
		},
		Redis: Redis{
			Host:     getEnvWithKoanf(k, "REDIS_HOST", "REDIS_HOST", "localhost"),
			Port:     getEnvWithKoanf(k, "REDIS_PORT", "REDIS_PORT", "6379"),
			Password: getEnvWithKoanf(k, "REDIS_PASSWORD", "REDIS_PASSWORD", ""),
			DB:       getIntWithKoanf(k, "REDIS_DB", "REDIS_DB", 0),
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

func getEnvWithKoanf(k *koanf.Koanf, envKey, koanfKey, defaultValue string) string {
	// Try koanf first (config file or environment variable)
	if value := k.String(koanfKey); value != "" {
		return value
	}
	// Fallback to direct environment variable
	return getEnv(envKey, defaultValue)
}

func getIntWithKoanf(k *koanf.Koanf, envKey, koanfKey string, defaultValue int) int {
	// Try koanf first
	if value := k.Int(koanfKey); value != 0 {
		return value
	}
	// Fallback to environment variable
	return getEnvAsInt(envKey, defaultValue)
}

func getDurationWithKoanf(k *koanf.Koanf, envKey, koanfKey string, defaultValue time.Duration) time.Duration {
	// Try koanf first
	if value := k.Duration(koanfKey); value != 0 {
		return value
	}
	// Fallback to environment variable
	return getEnvAsDuration(envKey, defaultValue)
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
