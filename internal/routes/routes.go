package routes

import (
	"database/sql"

	"openvdo/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "openvdo/docs" // swagger docs
)

type Server struct {
	router      *gin.Engine
	db          *sql.DB
	redisClient *redis.Client
}

func Setup(router *gin.Engine, db *sql.DB, redisClient *redis.Client) {
	server := &Server{
		router:      router,
		db:          db,
		redisClient: redisClient,
	}

	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())

	router.GET("/health", server.healthCheck)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// HealthCheck godoc
// @Summary Health Check
// @Description Check if the server is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"message": "OpenVDO server is running",
	})
}
