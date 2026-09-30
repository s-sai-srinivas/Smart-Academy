package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireCollege ensures the authenticated user has a college assigned
// Super admin bypasses this check
func RequireCollege() gin.HandlerFunc {
	return func(c *gin.Context) {
		collegeID, exists := c.Get("college_id")
		role, _ := c.Get("role")

		// Super admin bypasses college check
		if role == "super_admin" {
			c.Next()
			return
		}

		if !exists || collegeID == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUserCollege returns the college_id from context
// Use this in handlers instead of repeating the lookup logic
// Returns (collegeID pointer, true) if found, (nil, false) otherwise
func GetCurrentUserCollege(c *gin.Context) (*string, bool) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		return nil, false
	}
	collegeIDStr, ok := collegeID.(*string)
	if !ok {
		return nil, false
	}
	// Return the dereferenced string as a new pointer for consistent usage
	result := *collegeIDStr
	return &result, true
}

// IsSuperAdmin checks if the current user is a super admin
func IsSuperAdmin(c *gin.Context) bool {
	role, exists := c.Get("role")
	return exists && role == "super_admin"
}

// IsCollegeAdmin checks if the current user is a college admin or super admin
func IsCollegeAdmin(c *gin.Context) bool {
	role, exists := c.Get("role")
	return exists && (role == "college_admin" || role == "super_admin")
}

// IsAdmin checks if the current user has any admin role
func IsAdmin(c *gin.Context) bool {
	role, exists := c.Get("role")
	return exists && (role == "admin" || role == "college_admin" || role == "super_admin")
}

// GetCurrentUserRegdNo returns the user's registration number from context
func GetCurrentUserRegdNo(c *gin.Context) (string, bool) {
	regdNo, exists := c.Get("regdno")
	if !exists {
		return "", false
	}
	regdNoStr, ok := regdNo.(string)
	return regdNoStr, ok
}

// GetCurrentUserRole returns the user's role from context
func GetCurrentUserRole(c *gin.Context) (string, bool) {
	role, exists := c.Get("role")
	if !exists {
		return "", false
	}
	roleStr, ok := role.(string)
	return roleStr, ok
}
