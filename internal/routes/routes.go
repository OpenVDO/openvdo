package routes

import (
	"openvdo/internal/database"
	"openvdo/internal/handlers"
	"openvdo/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "openvdo/docs" // swagger docs
)

type Server struct {
	router       *gin.Engine
	poolManager  *database.StatelessPoolManager
	redisClient  *redis.Client
}

func Setup(router *gin.Engine, poolManager *database.StatelessPoolManager, redisClient *redis.Client) {
	server := &Server{
		router:      router,
		poolManager: poolManager,
		redisClient: redisClient,
	}

	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())

	// Health check endpoints (no authentication required)
	router.GET("/health", handlers.HealthCheck)
	router.GET("/health/db", database.StatelessHealthCheckHandler(server.poolManager))
	router.GET("/stats/db", database.StatelessMetricsHandler(server.poolManager))

	// Swagger documentation (no authentication required)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API endpoints with tenant database access
	api := router.Group("/api/v1")
	{
		// Apply database middleware only to API routes
		api.Use(database.StatelessDatabaseMiddleware(server.poolManager))

		// Organizations endpoints (require authentication)
		orgs := api.Group("/organizations")
		orgs.Use(database.StatelessRequireAuth())
		{
			orgs.GET("", handlers.StatelessGetOrganizations)
			orgs.POST("", handlers.StatelessCreateOrganization)
		}

		// Session management endpoints (require authentication)
		sessions := api.Group("/sessions")
		sessions.Use(database.StatelessRequireAuth())
		{
			sessions.GET("", handlers.StatelessGetUserSession)
			sessions.DELETE("", handlers.StatelessInvalidateSession)
		}
	}
}
