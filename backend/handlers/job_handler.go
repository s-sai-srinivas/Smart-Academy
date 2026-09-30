package handlers

import (
	"coding-platform/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetJobStatus handles GET /api/jobs/:id - returns job status for polling
func GetJobStatus(c *gin.Context) {
	jobID := c.Param("id")

	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID required"})
		return
	}

	// Require authentication - users can only see their own jobs
	_, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	status, err := services.GetJobStatus(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetJobQueueStats handles GET /api/admin/job-stats - returns queue statistics (admin only)
func GetJobQueueStats(c *gin.Context) {
	stats := services.GetQueueStats()
	c.JSON(http.StatusOK, stats)
}
