package database

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ContextKey string

const (
	UserIDKey      ContextKey = "user_id"
	OrgIDKey       ContextKey = "org_id"
	RoleKey        ContextKey = "user_role"
	DBKey          ContextKey = "tenant_db"
	StatelessDBKey ContextKey = "stateless_tenant_db"
	PoolKey        ContextKey = "pool_manager"
)

func StatelessDatabaseMiddleware(spm *StatelessPoolManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(string(PoolKey), spm)

		userID, err := extractUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user identification"})
			c.Abort()
			return
		}

		tenantDB, err := spm.NewTenantDB(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
			c.Abort()
			return
		}

		c.Set(string(StatelessDBKey), tenantDB)

		c.Writer.Header().Set("X-Tenant-ID", userID.String())
		c.Writer.Header().Set("X-Pool-Type", "stateless")

		c.Next()

		if tenantDB != nil {
			if err := tenantDB.Release(); err != nil {
				// Log error but don't fail the request
			}
		}
	}
}

func extractUserID(c *gin.Context) (uuid.UUID, error) {
	if userIDHeader := c.GetHeader("X-User-ID"); userIDHeader != "" {
		userID, err := uuid.Parse(userIDHeader)
		if err != nil {
			return uuid.Nil, err
		}
		return userID, nil
	}

	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			// TODO: Parse JWT token and extract user ID
			return uuid.Nil, fmt.Errorf("JWT token parsing not implemented")
		}
	}

	return uuid.Nil, fmt.Errorf("no user identification found")
}

func GetTenantDBFromContext(c *gin.Context) (*TenantDB, bool) {
	if db, exists := c.Get(string(DBKey)); exists {
		if tenantDB, ok := db.(*TenantDB); ok {
			return tenantDB, true
		}
	}
	return nil, false
}

func GetStatelessTenantDBFromContext(c *gin.Context) (*StatelessTenantDB, bool) {
	if db, exists := c.Get(string(StatelessDBKey)); exists {
		if tenantDB, ok := db.(*StatelessTenantDB); ok {
			return tenantDB, true
		}
	}
	return nil, false
}

func GetPoolManagerFromContext(c *gin.Context) (*PoolManager, bool) {
	if pm, exists := c.Get(string(PoolKey)); exists {
		if pool, ok := pm.(*PoolManager); ok {
			return pool, true
		}
	}
	return nil, false
}

func GetStatelessPoolManagerFromContext(c *gin.Context) (*StatelessPoolManager, bool) {
	if pm, exists := c.Get(string(PoolKey)); exists {
		if pool, ok := pm.(*StatelessPoolManager); ok {
			return pool, true
		}
	}
	return nil, false
}

func StatelessRequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := extractUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		c.Set(string(UserIDKey), userID)
		c.Next()
	}
}

func StatelessRequireRole(orgIDParam string, requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		spm, exists := GetStatelessPoolManagerFromContext(c)
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database pool not available"})
			c.Abort()
			return
		}

		userID, exists := c.Get(string(UserIDKey))
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		orgID, err := uuid.Parse(c.Param(orgIDParam))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
			c.Abort()
			return
		}

		hasRole, err := NewStatelessTenantOperations(spm).HasRole(
			c.Request.Context(),
			userID.(uuid.UUID),
			orgID,
			requiredRole,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authorization check failed"})
			c.Abort()
			return
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Set(string(OrgIDKey), orgID)
		c.Set(string(RoleKey), requiredRole)
		c.Next()
	}
}

// StatelessHealthCheckHandler godoc
// @Summary Stateless database pool health check
// @Description Checks the health of the stateless database connection pool
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{} "Stateless database pools are healthy"
// @Failure 503 {object} map[string]interface{} "Stateless database pools unhealthy"
// @Router /health/db [get]
func StatelessHealthCheckHandler(spm *StatelessPoolManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		health := spm.GetHealth()

		if health.Healthy {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"message":   "Stateless database pools are healthy",
				"pool_type": "stateless",
				"data":      health,
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"message":   "Stateless database pools unhealthy",
				"pool_type": "stateless",
				"data":      health,
			})
		}
	}
}

// StatelessMetricsHandler godoc
// @Summary Stateless database pool statistics
// @Description Returns detailed statistics about the stateless database connection pool
// @Tags stats
// @Produce json
// @Success 200 {object} map[string]interface{} "Stateless database pool metrics"
// @Router /stats/db [get]
func StatelessMetricsHandler(spm *StatelessPoolManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics := spm.GetMetrics()
		c.JSON(http.StatusOK, gin.H{
			"status":    "success",
			"message":   "Stateless database pool metrics",
			"pool_type": "stateless",
			"data":      metrics,
		})
	}
}

func RequireAuth() gin.HandlerFunc {
	return StatelessRequireAuth()
}

func RequireRole(orgIDParam string, requiredRole string) gin.HandlerFunc {
	return StatelessRequireRole(orgIDParam, requiredRole)
}

func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID, err := extractUserID(c); err == nil {
			c.Set(string(UserIDKey), userID)
		}
		c.Next()
	}
}
