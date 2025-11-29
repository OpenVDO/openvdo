package main

import (
	"log"
	"os"

	"openvdo/internal/config"
	"openvdo/internal/database"
	"openvdo/internal/routes"
	"openvdo/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "openvdo/docs" // swagger docs
)

// @title OpenVDO API
// @version 1.0
// @description A high-performance video streaming backend built with Go, Gin, PostgreSQL, and Redis.

// @host localhost:8080

func main() {
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using environment variables")
	}

	cfg := config.Load()

	// Initialize the stateless connection pool manager
	if err := database.InitPoolManager(cfg.Database, cfg.Redis); err != nil {
		log.Fatal("Failed to initialize stateless pool manager:", err)
	}
	defer database.ClosePoolManager()

	
	if gin.Mode() == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Get pool manager for routes
	poolManager := database.GetPoolManager()
	routes.Setup(r, poolManager, nil) // Redis is managed by pool manager

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}