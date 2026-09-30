package middleware

import (
	"coding-platform/database"
	"coding-platform/models"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetCurrentUser retrieves the authenticated user from the context
// Returns nil if user is not authenticated or not found
func GetCurrentUser(c *gin.Context) (*models.User, error) {
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		return nil, errors.New("not authenticated")
	}

	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetCurrentUserWithCollege retrieves the authenticated user and validates college assignment
// Returns error if user has no college assigned
func GetCurrentUserWithCollege(c *gin.Context) (*models.User, error) {
	user, err := GetCurrentUser(c)
	if err != nil {
		return nil, err
	}

	if user.CollegeID == nil {
		return nil, errors.New("no college assigned")
	}

	return user, nil
}

// RequireCurrentUser ensures user is authenticated, returns error response if not
func RequireCurrentUser(c *gin.Context) (*models.User, bool) {
	user, err := GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return nil, false
	}
	return user, true
}

// RequireUserWithCollege ensures user is authenticated and has a college
func RequireUserWithCollege(c *gin.Context) (*models.User, bool) {
	user, err := GetCurrentUserWithCollege(c)
	if err != nil {
		if err.Error() == "not authenticated" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
		}
		return nil, false
	}
	return user, true
}
