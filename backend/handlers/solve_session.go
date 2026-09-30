package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// StartSolveSession begins tracking a user's solve session for a problem
func StartSolveSession(c *gin.Context) {
	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get problem ID from URL
	problemID := c.Param("id")
	if problemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Problem ID is required"})
		return
	}

	// Check if user already has an active session for this problem (within last 24 hours)
	var existingSession models.SolveSession
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	err := database.DB.Where(
		"user_regd_no = ? AND problem_id = ? AND started_at > ? AND ended_at IS NULL",
		userRegdNo, problemID, oneDayAgo,
	).First(&existingSession).Error

	if err == nil {
		// Active session exists, return it
		c.JSON(http.StatusOK, gin.H{
			"session_id": existingSession.ID,
			"started_at": existingSession.StartedAt,
			"existing":   true,
		})
		return
	}

	// Create new session
	session := models.SolveSession{
		UserRegdNo: userRegdNo.(string),
		ProblemID:  parseUint(problemID),
		StartedAt:  time.Now(),
	}

	if err := database.DB.Create(&session).Error; err != nil {
		log.Printf("Failed to create solve session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": session.ID,
		"started_at": session.StartedAt,
		"existing":   false,
	})
}

// EndSolveSession marks a solve session as ended
func EndSolveSession(c *gin.Context) {
	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	// Update session
	now := time.Now()
	result := database.DB.Model(&models.SolveSession{}).
		Where("id = ? AND user_regd_no = ?", sessionID, userRegdNo).
		Updates(map[string]interface{}{
			"ended_at":  &now,
			"completed": c.Query("completed") == "true",
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to end session"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session ended"})
}

// GetActiveSolveSession returns the active session for a user and problem
func GetActiveSolveSession(c *gin.Context) {
	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	problemID := c.Param("id")
	if problemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Problem ID is required"})
		return
	}

	// Find active session (within last 24 hours, not ended)
	var session models.SolveSession
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	err := database.DB.Where(
		"user_regd_no = ? AND problem_id = ? AND started_at > ? AND ended_at IS NULL",
		userRegdNo, problemID, oneDayAgo,
	).First(&session).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"active":     false,
			"session_id": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active":          true,
		"session_id":      session.ID,
		"started_at":      session.StartedAt,
		"elapsed_seconds": int(time.Since(session.StartedAt).Seconds()),
	})
}

// Helper function to parse uint from string
func parseUint(s string) uint {
	var result uint
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			result = result*10 + uint(ch-'0')
		}
	}
	return result
}
