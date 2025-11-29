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

	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close(db)

	redisClient := database.ConnectRedis(cfg.Redis)
	defer database.CloseRedis(redisClient)

	if gin.Mode() == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	routes.Setup(r, db, redisClient)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}