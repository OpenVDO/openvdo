package handlers

import (
	"net/http"
	"strconv"

	"openvdo/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetOrganizations retrieves organizations for the authenticated user
func GetOrganizations(c *gin.Context) {
	tenantDB, exists := database.GetTenantDBFromContext(c)
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
		"message": "Organizations retrieved successfully",
		"data": gin.H{
			"organizations": organizations,
			"pagination": gin.H{
				"page":  page,
				"limit": limit,
				"total": total,
			},
		},
	})
}

// CreateOrganization creates a new organization
func CreateOrganization(c *gin.Context) {
	tenantDB, exists := database.GetTenantDBFromContext(c)
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
	`

	_, err := tenantDB.ExecContext(c.Request.Context(), query, req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create organization"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Organization created successfully",
		"data": gin.H{
			"name": req.Name,
			"message": "Organization has been created with RLS policies applied",
		},
	})
}