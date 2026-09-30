package handlers

import (
	"coding-platform/config"
	"coding-platform/database"
	"coding-platform/middleware"
	"coding-platform/models"
	"coding-platform/utils"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegisterRequest struct {
	RegdNo   string `json:"regdno" binding:"required,min=3,max=50"`    // Registration/Employee number
	Email    string `json:"email" binding:"required,email,max=255"`    // Email with max length
	Password string `json:"password" binding:"required,min=6,max=128"` // Password: min 6, max 128 (bcrypt DoS prevention)
	Role     string `json:"role"`
	Name     string `json:"name" binding:"required,min=1,max=200"` // Name with max length
}

type LoginRequest struct {
	CollegeID *string `json:"college_id"`
	RegdNo    string  `json:"regdno" binding:"required,min=1,max=255"`
	Password  string  `json:"password" binding:"required,min=1,max=128"`
}

// GetColleges returns all active colleges for the login dropdown
func GetColleges(c *gin.Context) {
	var colleges []models.College
	if err := database.DB.Where("is_active = ?", true).Order("college_name").Find(&colleges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch colleges"})
		return
	}
	c.JSON(http.StatusOK, colleges)
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate password strength
	if err := utils.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default role is student
	if req.Role == "" {
		req.Role = "student"
	}

	// Only allow student or admin roles during registration
	if req.Role != "student" && req.Role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	// Check for existing regdno (including soft-deleted users)
	var existingUser models.User
	err := database.DB.Unscoped().Where("regdno = ?", req.RegdNo).First(&existingUser).Error
	if err == nil {
		// RegdNo exists
		if existingUser.DeletedAt.Valid {
			// User was soft deleted - restore (undelete) the account with new info
			existingUser.DeletedAt.Time = time.Time{} // Reset to zero time
			existingUser.Email = req.Email
			existingUser.Name = req.Name
			existingUser.Role = req.Role
			existingUser.IsActive = true

			if err := existingUser.HashPassword(req.Password); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
				return
			}

			if err := database.DB.Unscoped().Save(&existingUser).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore account"})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"message": "Account restored successfully",
				"user": gin.H{
					"regdno": existingUser.RegdNo,
					"email":  existingUser.Email,
					"name":   existingUser.Name,
					"role":   existingUser.Role,
				},
			})
			return
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "Registration number already exists"})
			return
		}
	}

	// Check for existing email (including soft-deleted users)
	err = database.DB.Unscoped().Where("email = ?", req.Email).First(&existingUser).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	}

	// Create new user
	user := models.User{
		RegdNo: req.RegdNo,
		Email:  req.Email,
		Name:   req.Name,
		Role:   req.Role,
	}

	if err := user.HashPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": gin.H{
			"regdno": user.RegdNo,
			"email":  user.Email,
			"name":   user.Name,
			"role":   user.Role,
		},
	})
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	// Try to find user by regdno or email
	if err := database.DB.Where("regdno = ? OR email = ?", req.RegdNo, req.RegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Fix 1: Reject login for deactivated accounts before any token is issued
	if !user.IsActive {
		log.Printf("Login FAILED (deactivated): regdno=%s", user.RegdNo)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is deactivated. Please contact your administrator"})
		return
	}

	if !user.CheckPassword(req.Password) {
		log.Printf("Login FAILED (password): regdno=%s", user.RegdNo)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	log.Printf("Login attempt: regdno=%s role=%s user.CollegeID=%v req.CollegeID=%v",
		user.RegdNo, user.Role, user.CollegeID, req.CollegeID)

	// Super admin has no college association — skip college validation
	if user.Role == "super_admin" {
		if user.CollegeID != nil {
			log.Printf("Login FAILED (super_admin has college): regdno=%s", user.RegdNo)
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid super_admin account configuration"})
			return
		}
	} else {
		// Regular users must supply a college_id and belong to it
		if req.CollegeID == nil {
			log.Printf("Login FAILED (no college_id sent): regdno=%s", user.RegdNo)
			c.JSON(http.StatusBadRequest, gin.H{"error": "college_id is required"})
			return
		}
		if user.CollegeID == nil || *user.CollegeID != *req.CollegeID {
			log.Printf("Login FAILED (college mismatch): regdno=%s userCollege=%v reqCollege=%v", user.RegdNo, user.CollegeID, *req.CollegeID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "You do not belong to the selected college"})
			return
		}
	}

	// Generate JWT token
	token, err := generateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"regdno":     user.RegdNo,
			"email":      user.Email,
			"name":       user.Name,
			"role":       user.Role,
			"college_id": user.CollegeID,
		},
	})
}

// generateToken creates a signed JWT for the given user.
// Fix 4: Each token gets a unique jti (UUID) so it can be individually revoked via Logout.
// TokenVersion is included to invalidate all tokens when role changes.
func generateToken(user *models.User) (string, error) {
	claims := middleware.Claims{
		RegdNo:       user.RegdNo,
		Role:         user.Role,
		CollegeID:    user.CollegeID,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    config.AppConfig.JWTIssuer,
			Subject:   user.RegdNo,
			Audience:  []string{config.AppConfig.JWTAudience},
			ID:        uuid.NewString(), // jti — unique per token, used for blacklisting
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

// Logout invalidates the caller's current JWT token immediately.
// Fix 4: The token's jti is added to the in-memory blacklist so subsequent
// requests with the same token are rejected even before natural expiry.
func Logout(c *gin.Context) {
	jti, _ := c.Get("token_jti")
	exp, _ := c.Get("token_exp")

	if jti != nil && jti != "" {
		jtiStr, ok := jti.(string)
		if ok && jtiStr != "" {
			var expTime time.Time
			if numericDate, ok := exp.(*jwt.NumericDate); ok && numericDate != nil {
				expTime = numericDate.Time
			} else {
				expTime = time.Now().Add(24 * time.Hour) // safe fallback
			}
			middleware.BlacklistToken(jtiStr, expTime)
			log.Printf("Logout: blacklisted token jti=%s", jtiStr)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetMe returns basic user info including streak
func GetMe(c *gin.Context) {
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get user info
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Get streak info
	response := gin.H{
		"regdno":     user.RegdNo,
		"email":      user.Email,
		"name":       user.Name,
		"role":       user.Role,
		"college_id": user.CollegeID,
		"streak":     0,
	}

	// Try to get streak from UserStreak table
	var streak models.UserStreak
	if err := database.DB.Where("user_regd_no = ?", userRegdNo).First(&streak).Error; err == nil {
		// Stale-streak guard: if the user's last activity was before yesterday,
		// their streak has lapsed. Show 0 immediately without waiting for the
		// next submission to trigger the SQL reset.
		yesterday := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -1)
		lastActivity := streak.LastActivityDate.Truncate(24 * time.Hour)
		if !lastActivity.IsZero() && lastActivity.Before(yesterday) {
			response["streak"] = 0
		} else {
			response["streak"] = streak.CurrentStreak
		}
	}

	c.JSON(http.StatusOK, response)
}

// SeedSuperAdmin creates or updates the super_admin account using env-based credentials.
// Called automatically on server start — not exposed as an HTTP route.
func SeedSuperAdmin() error {
	cfg := config.AppConfig

	regdNo := cfg.SuperAdminRegdNo
	email := cfg.SuperAdminEmail
	password := cfg.SuperAdminPassword
	if regdNo == "" || email == "" || password == "" {
		return errors.New("SeedSuperAdmin: SUPER_ADMIN_REGDNO, SUPER_ADMIN_EMAIL, and SUPER_ADMIN_PASSWORD are required")
	}

	log.Printf("SeedSuperAdmin: bootstrapping regdno=%s email=%s", regdNo, email)

	var existing models.User
	err := database.DB.Unscoped().Where("regdno = ?", regdNo).First(&existing).Error

	if err == nil {
		// Already exists — ensure role is super_admin and CollegeID is nil
		existing.Role = "super_admin"
		existing.Email = email
		existing.CollegeID = nil
		existing.IsActive = true
		if err := existing.HashPassword(password); err != nil {
			return err
		}
		if err := database.DB.Unscoped().Save(&existing).Error; err != nil {
			return err
		}
		log.Printf("SeedSuperAdmin: updated existing super_admin user regdno=%s", regdNo)
		return nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// Create new super_admin
	user := models.User{
		RegdNo:    regdNo,
		Email:     email,
		Name:      "Super Admin",
		Role:      "super_admin",
		CollegeID: nil,
		IsActive:  true,
	}
	if err := user.HashPassword(password); err != nil {
		return err
	}
	if err := database.DB.Create(&user).Error; err != nil {
		return err
	}
	log.Printf("SeedSuperAdmin: created super_admin user regdno=%s", regdNo)
	return nil
}

type SeedFacultyRequest struct {
	RegdNo   string `json:"regdno" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=6,max=128"`
	Name     string `json:"name" binding:"required,min=1,max=200"`
}

// SeedFacultyUser creates or updates a faculty user with the calling admin's college_id.
// Fix 3: college_id is now always derived from the authenticated admin context,
// preventing orphaned faculty accounts that break college-scoped queries.
// Fix 9: This endpoint is now protected (requires auth + AdminOnly middleware in router).
func SeedFacultyUser(c *gin.Context) {
	var req SeedFacultyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate password strength
	if err := utils.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Derive college_id from the authenticated admin's context
	collegeIDVal, exists := c.Get("college_id")
	if !exists || collegeIDVal == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Admin account has no college assigned. Cannot create faculty without a college"})
		return
	}
	collegeIDPtr, ok := collegeIDVal.(*string)
	if !ok || collegeIDPtr == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid college context"})
		return
	}
	collegeID := *collegeIDPtr

	// Verify the college exists and is active
	var college models.College
	if err := database.DB.Where("college_id = ? AND is_active = ?", collegeID, true).First(&college).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not found or inactive"})
		return
	}

	var existingUser models.User
	err := database.DB.Unscoped().Where("regdno = ?", req.RegdNo).First(&existingUser).Error

	if err == nil {
		// User exists — update with new credentials and set/update college_id
		existingUser.Email = req.Email
		existingUser.Name = req.Name
		existingUser.Role = "faculty"
		existingUser.IsActive = true
		existingUser.CollegeID = &collegeID // Fix 3: assign college

		if err := existingUser.HashPassword(req.Password); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		if err := database.DB.Unscoped().Save(&existingUser).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update faculty user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Faculty user updated successfully",
			"user": gin.H{
				"regdno":     existingUser.RegdNo,
				"email":      existingUser.Email,
				"name":       existingUser.Name,
				"role":       existingUser.Role,
				"college_id": existingUser.CollegeID,
			},
		})
		return
	}

	// Create new faculty user with college_id from admin context
	user := models.User{
		RegdNo:    req.RegdNo,
		Email:     req.Email,
		Name:      req.Name,
		Role:      "faculty",
		IsActive:  true,
		CollegeID: &collegeID, // Fix 3: always set college_id
	}

	if err := user.HashPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create faculty user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Faculty user created successfully",
		"user": gin.H{
			"regdno":     user.RegdNo,
			"email":      user.Email,
			"name":       user.Name,
			"role":       user.Role,
			"college_id": user.CollegeID,
		},
	})
}

// ChangePasswordRequest is the payload for changing your own password while authenticated.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=1,max=128"`
	NewPassword     string `json:"new_password" binding:"required,min=6,max=128"`
}

// ChangePassword allows an authenticated user to change their own password.
// Fix 5: Self-service password change (when you know your current password).
func ChangePassword(c *gin.Context) {
	regdNoVal, _ := c.Get("regdno")
	regdNo, _ := regdNoVal.(string)

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate new password strength
	if err := utils.ValidatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Where("regdno = ?", regdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if !user.CheckPassword(req.CurrentPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	if err := user.HashPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash new password"})
		return
	}

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	log.Printf("Password changed by user: regdno=%s", regdNo)
	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// AdminResetPasswordRequest is the payload for an admin resetting another user's password.
type AdminResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
}

// AdminResetPassword allows an admin to reset the password for any user within their college.
// Fix 5: Admin-assisted password recovery (used when the user cannot remember their password).
func AdminResetPassword(c *gin.Context) {
	targetRegdNo := c.Param("regdno")
	if targetRegdNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target user regdno is required"})
		return
	}

	var req AdminResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate new password strength
	if err := utils.ValidatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get admin's college to enforce college isolation
	adminCollegeIDVal, _ := c.Get("college_id")
	adminCollegeIDPtr, _ := adminCollegeIDVal.(*string)

	// Find target user
	var targetUser models.User
	if err := database.DB.Where("regdno = ?", targetRegdNo).First(&targetUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find user"})
		}
		return
	}

	// Super admin can reset anyone; regular admin must stay within their college
	if !middleware.IsSuperAdmin(c) {
		if adminCollegeIDPtr == nil || targetUser.CollegeID == nil || *adminCollegeIDPtr != *targetUser.CollegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only reset passwords for users in your college"})
			return
		}
	}

	// Prevent admin from resetting super_admin passwords (escalation guard)
	if targetUser.Role == "super_admin" && !middleware.IsSuperAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot reset super_admin password"})
		return
	}

	if err := targetUser.HashPassword(req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := database.DB.Save(&targetUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	adminRegdNo, _ := c.Get("regdno")
	log.Printf("Password reset by admin=%s for user=%s", adminRegdNo, targetRegdNo)
	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully for user " + targetRegdNo})
}
