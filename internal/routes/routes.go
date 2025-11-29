package routes

import (
	"database/sql"

	"openvdo/internal/handlers"
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

	api := router.Group("/api/v1")
	{
		server.setupUserRoutes(api)
	}
}

func (s *Server) setupUserRoutes(api *gin.RouterGroup) {
	userHandler := handlers.NewUserHandler(s.db, s.redisClient)

	users := api.Group("/users")
	{
		users.POST("/", userHandler.CreateUser)
		users.GET("/", userHandler.GetUsers)
		users.GET("/:id", userHandler.GetUser)
		users.PUT("/:id", userHandler.UpdateUser)
		users.DELETE("/:id", userHandler.DeleteUser)
	}
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"message": "OpenVDO server is running",
	})
}