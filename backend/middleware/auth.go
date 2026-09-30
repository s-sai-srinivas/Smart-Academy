package middleware

import (
	"coding-platform/config"
	"coding-platform/database"
	"coding-platform/models"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload. The ID field (jti) enables token revocation via the blacklist.
// TokenVersion enables invalidation of all tokens when user's role changes.
type Claims struct {
	CollegeID *string `json:"college_id,omitempty"`
	jwt.RegisteredClaims
	RegdNo       string `json:"regdno"`
	Role         string `json:"role"`
	TokenVersion int    `json:"token_version"`
}

// recheckUserActive looks up the user in the DB on every authenticated request to close
// the race condition where a deleted/deactivated user holds a still-valid JWT.
// Also returns the token version for session invalidation check.
func recheckUserActive(c *gin.Context, regdNo string) (int, error) {
	var user models.User
	if err := database.DB.Select("regdno, is_active, token_version").Where("regdno = ?", regdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User account not found"})
		return 0, err
	}
	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account has been deactivated. Please contact your administrator"})
		return 0, &gin.Error{Err: nil, Type: gin.ErrorTypePublic}
	}
	return user.TokenVersion, nil
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		var tokenString string

		if authHeader != "" {
			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
				c.Abort()
				return
			}
			tokenString = parts[1]
		} else {
			// Fallback for cookie-based auth deployments
			cookieToken, err := c.Cookie("authToken")
			if err == nil && strings.TrimSpace(cookieToken) != "" {
				tokenString = cookieToken
			}
		}

		// Browser WebSocket handshakes cannot set custom Authorization headers.
		// For upgrade requests, allow token fallback via query param (?token=...)
		// to support sessionStorage/localStorage token mode.
		if tokenString == "" && strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
			queryToken := strings.TrimSpace(c.Query("token"))
			if queryToken != "" {
				tokenString = queryToken
			}
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Validate signing algorithm to prevent algorithm confusion attacks
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(config.AppConfig.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			// Check if the error is specifically about token expiration
			if errors.Is(err, jwt.ErrTokenExpired) || (err != nil && err.Error() == "token has invalid claims: token is expired") {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Token has expired",
					"code":  "TOKEN_EXPIRED",
				})
				c.Abort()
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Fix 4: Check token blacklist (for explicitly logged-out tokens)
		if claims.ID != "" && IsTokenBlacklisted(claims.ID) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked. Please log in again"})
			c.Abort()
			return
		}

		// Fix 2: Re-check user existence, active status, and token version on every request.
		// This closes the race condition where deleted/deactivated users continue
		// to call the API with a still-valid JWT.
		// Token version check invalidates sessions when role changes.
		dbTokenVersion, err := recheckUserActive(c, claims.RegdNo)
		if err != nil {
			c.Abort()
			return
		}

		// Check token version - if different, user's session was invalidated (e.g., role change)
		if claims.TokenVersion != dbTokenVersion {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session invalidated. Please log in again"})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("regdno", claims.RegdNo)
		c.Set("role", claims.Role)
		c.Set("college_id", claims.CollegeID)
		c.Set("token_jti", claims.ID)        // used by Logout handler
		c.Set("token_exp", claims.ExpiresAt) // used by Logout handler

		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || (role != "admin" && role != "college_admin" && role != "super_admin") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func HODOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || (role != "hod" && role != "college_admin" && role != "super_admin") {
			c.JSON(http.StatusForbidden, gin.H{"error": "HOD access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func PrincipalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || (role != "principal" && role != "super_admin") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Principal access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// FacultyOnly allows faculty and all higher roles to access faculty routes.
// Fix 6: Role hierarchy is now consistent with all other middlewares:
//
//	faculty < hod < college_admin/principal < super_admin
//
// Previously only role=="faculty" was permitted, blocking HODs from faculty analytics, etc.
func FacultyOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || (role != "faculty" && role != "hod" && role != "college_admin" && role != "principal" && role != "super_admin") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Faculty access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// SuperAdminOnly restricts access exclusively to super_admin users.
// No other role (college_admin, hod, principal, etc.) can pass this guard.
func SuperAdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Super admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ContestParticipantOnly validates that the user is eligible to participate in a contest.
// Checks: authentication, college assignment. Full eligibility is done in the handler.
func ContestParticipantOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user info from context (set by AuthMiddleware)
		_, exists := c.Get("regdno")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		userCollegeID, hasCollege := c.Get("college_id")
		if !hasCollege || userCollegeID == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "User has no college assigned"})
			c.Abort()
			return
		}

		contestID := c.Param("id")
		if contestID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Contest ID required"})
			c.Abort()
			return
		}

		// Validate contest exists and belongs to user's college
		// Note: Full validation (cohort/branch eligibility) is done in handler
		// to avoid circular imports and keep middleware lightweight
		c.Set("contest_id", contestID)
		c.Set("user_college_id", userCollegeID)
		c.Next()
	}
}
