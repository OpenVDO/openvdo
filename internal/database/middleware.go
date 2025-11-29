package database

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ContextKey is used for storing values in request context
type ContextKey string

const (
	UserIDKey  ContextKey = "user_id"
	OrgIDKey   ContextKey = "org_id"
	RoleKey    ContextKey = "user_role"
	DBKey      ContextKey = "tenant_db"
	StatelessDBKey ContextKey = "stateless_tenant_db"
	PoolKey    ContextKey = "pool_manager"
)

// StatelessDatabaseMiddleware provides stateless tenant database connection middleware
func StatelessDatabaseMiddleware(spm *StatelessPoolManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store pool manager in context
		c.Set(string(PoolKey), spm)

		// Extract user ID from header or JWT token
		userID, err := extractUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user identification"})
			c.Abort()
			return
		}

		// Create stateless tenant database connection
		tenantDB, err := spm.NewTenantDB(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
			c.Abort()
			return
		}

		// Store tenant DB in context
		c.Set(string(StatelessDBKey), tenantDB)

		// Set up cleanup when request is done
		c.Writer.Header().Set("X-Tenant-ID", userID.String())
		c.Writer.Header().Set("X-Pool-Type", "stateless")

		c.Next()

		// Release connection back to pool
		if tenantDB != nil {
			if err := tenantDB.Release(); err != nil {
				// Log error but don't fail the request
			}
		}
	}
}

// extractUserID extracts user ID from request headers or JWT token
func extractUserID(c *gin.Context) (uuid.UUID, error) {
	// Try to get from X-User-ID header first
	if userIDHeader := c.GetHeader("X-User-ID"); userIDHeader != "" {
		userID, err := uuid.Parse(userIDHeader)
		if err != nil {
			return uuid.Nil, err
		}
		return userID, nil
	}

	// Try to get from Authorization header (JWT would be parsed here)
	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			// TODO: Parse JWT token and extract user ID
			// For now, this is a placeholder
			return uuid.Nil, fmt.Errorf("JWT token parsing not implemented")
		}
	}

	return uuid.Nil, fmt.Errorf("no user identification found")
}

// GetTenantDBFromContext retrieves tenant database from context
func GetTenantDBFromContext(c *gin.Context) (*TenantDB, bool) {
	if db, exists := c.Get(string(DBKey)); exists {
		if tenantDB, ok := db.(*TenantDB); ok {
			return tenantDB, true
		}
	}
	return nil, false
}

// GetStatelessTenantDBFromContext retrieves stateless tenant database from context
func GetStatelessTenantDBFromContext(c *gin.Context) (*StatelessTenantDB, bool) {
	if db, exists := c.Get(string(StatelessDBKey)); exists {
		if tenantDB, ok := db.(*StatelessTenantDB); ok {
			return tenantDB, true
		}
	}
	return nil, false
}

// GetPoolManagerFromContext retrieves stateful pool manager from context
func GetPoolManagerFromContext(c *gin.Context) (*PoolManager, bool) {
	if pm, exists := c.Get(string(PoolKey)); exists {
		if pool, ok := pm.(*PoolManager); ok {
			return pool, true
		}
	}
	return nil, false
}

// GetStatelessPoolManagerFromContext retrieves stateless pool manager from context
func GetStatelessPoolManagerFromContext(c *gin.Context) (*StatelessPoolManager, bool) {
	if pm, exists := c.Get(string(PoolKey)); exists {
		if pool, ok := pm.(*StatelessPoolManager); ok {
			return pool, true
		}
	}
	return nil, false
}

// StatelessRequireAuth middleware that ensures user is authenticated
func StatelessRequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := extractUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Store user ID in context
		c.Set(string(UserIDKey), userID)
		c.Next()
	}
}

// StatelessRequireRole middleware that ensures user has specific role in organization
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

		// Parse organization ID from parameter
		orgID, err := uuid.Parse(c.Param(orgIDParam))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID"})
			c.Abort()
			return
		}

		// Check if user has required role using stateless operations
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

		// Store org and role in context
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
				"status":      "healthy",
				"message":     "Stateless database pools are healthy",
				"pool_type":   "stateless",
				"data":        health,
			})
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":      "unhealthy",
				"message":     "Stateless database pools unhealthy",
				"pool_type":   "stateless",
				"data":        health,
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
			"status":      "success",
			"message":     "Stateless database pool metrics",
			"pool_type":   "stateless",
			"data":        metrics,
		})
	}
}

// Legacy middleware for backward compatibility
func RequireAuth() gin.HandlerFunc {
	return StatelessRequireAuth()
}

func RequireRole(orgIDParam string, requiredRole string) gin.HandlerFunc {
	return StatelessRequireRole(orgIDParam, requiredRole)
}

// OptionalAuth middleware that tries to authenticate but doesn't require it
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID, err := extractUserID(c); err == nil {
			c.Set(string(UserIDKey), userID)
		}
		c.Next()
	}
}

