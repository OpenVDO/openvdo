package handlers

import (
	"net/http"
	"strconv"

	"openvdo/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StatelessGetOrganizations godoc
// @Summary Get user organizations
// @Description Retrieves all organizations for the authenticated user using stateless connection pooling with RLS filtering
// @Tags organizations
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "Organizations retrieved successfully"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/organizations [get]
func StatelessGetOrganizations(c *gin.Context) {
	tenantDB, exists := database.GetStatelessTenantDBFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}

	// Build query with pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	query := `
		SELECT id, name, description, created_at, updated_at
		FROM organizations
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := tenantDB.QueryContext(c.Request.Context(), query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query organizations"})
		return
	}
	defer rows.Close()

	var organizations []gin.H
	for rows.Next() {
		var org struct {
			ID          uuid.UUID `json:"id"`
			Name        string    `json:"name"`
			Description string    `json:"description"`
			CreatedAt   string    `json:"created_at"`
			UpdatedAt   string    `json:"updated_at"`
		}

		if err := rows.Scan(&org.ID, &org.Name, &org.Description, &org.CreatedAt, &org.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan organization"})
			return
		}

		organizations = append(organizations, gin.H{
			"id":          org.ID,
			"name":        org.Name,
			"description": org.Description,
			"created_at":  org.CreatedAt,
			"updated_at":  org.UpdatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error processing organization results"})
		return
	}

	// Get total count for pagination
	var total int
	countQuery := "SELECT COUNT(*) FROM organizations"
	if err := tenantDB.QueryRowContext(c.Request.Context(), countQuery).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Organizations retrieved successfully (stateless)",
		"data": gin.H{
			"organizations": organizations,
			"pagination": gin.H{
				"page":  page,
				"limit": limit,
				"total": total,
			},
			"pool_type": "stateless",
		},
	})
}

// StatelessCreateOrganization godoc
// @Summary Create organization
// @Description Creates a new organization using stateless connection pooling
// @Tags organizations
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param name body string true "Organization name"
// @Param description body string false "Organization description"
// @Success 201 {object} map[string]interface{} "Organization created successfully"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/organizations [post]
func StatelessCreateOrganization(c *gin.Context) {
	tenantDB, exists := database.GetStatelessTenantDBFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not available"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Simple query for now
	query := `
		INSERT INTO organizations (name, description)
		VALUES ($1, $2)
		RETURNING id, created_at
	`

	var newID uuid.UUID
	var createdAt string
	err := tenantDB.QueryRowContext(c.Request.Context(), query, req.Name, req.Description).Scan(&newID, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create organization"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Organization created successfully (stateless)",
		"data": gin.H{
			"id":        newID,
			"name":      req.Name,
			"created_at": createdAt,
			"pool_type": "stateless",
		},
	})
}

// StatelessGetUserSession godoc
// @Summary Get user session
// @Description Retrieves the current user's session information including organization and role details
// @Tags sessions
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "User session retrieved"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Session not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/sessions [get]
func StatelessGetUserSession(c *gin.Context) {
	spm, exists := database.GetStatelessPoolManagerFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pool manager not available"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		// Try to extract from context middleware
		if tenantDB, hasDB := database.GetStatelessTenantDBFromContext(c); hasDB {
			userID = tenantDB.GetUserID()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}
	}

	session, err := spm.GetUserSession(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User session not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User session retrieved",
		"data":    session,
	})
}

// StatelessInvalidateSession godoc
// @Summary Invalidate user session
// @Description Invalidates the current user's session
// @Tags sessions
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "Session invalidated"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/sessions [delete]
func StatelessInvalidateSession(c *gin.Context) {
	spm, exists := database.GetStatelessPoolManagerFromContext(c)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Pool manager not available"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		if tenantDB, hasDB := database.GetStatelessTenantDBFromContext(c); hasDB {
			userID = tenantDB.GetUserID()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}
	}

	err := spm.InvalidateUserSession(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to invalidate session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "User session invalidated",
	})
}