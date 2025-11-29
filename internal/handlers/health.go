package handlers

import (
	"net/http"

	"openvdo/internal/database"

	"github.com/gin-gonic/gin"
)

// HealthCheck godoc
// @Summary Basic health check
// @Description Checks if the server is running and responds with basic status
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{} "Server is healthy"
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Server is healthy",
		"data":    gin.H{},
	})
}

// DatabaseHealthCheck godoc
// @Summary Database pool health check
// @Description Checks the health of database connection pools
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{} "Database pools are healthy"
// @Failure 503 {object} map[string]string "Database pool not available"
// @Router /health/db [get]
func DatabaseHealthCheck(c *gin.Context) {
	pm := database.GetPoolManager()
	if pm == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database pool not available"})
		return
	}

	health := pm.GetHealth()
	if health.Healthy {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"message": "Database pools are healthy",
			"data":    health,
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"message": "Database pools unhealthy",
			"data":    health,
		})
	}
}

// DatabaseStats godoc
// @Summary Database pool statistics
// @Description Returns detailed statistics about database connection pools
// @Tags stats
// @Produce json
// @Success 200 {object} map[string]interface{} "Database pool statistics"
// @Failure 503 {object} map[string]string "Database pool not available"
// @Router /stats/db [get]
func DatabaseStats(c *gin.Context) {
	pm := database.GetPoolManager()
	if pm == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database pool not available"})
		return
	}

	metrics := pm.GetMetrics()
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Database pool statistics",
		"data":    metrics,
	})
}