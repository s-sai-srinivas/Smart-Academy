package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// =====================================================
// Input Size Limits Configuration
// =====================================================

// Input size limits to prevent DoS attacks and memory exhaustion
const (
	// MaxRequestBodySize is the maximum size for request bodies (1MB)
	MaxRequestBodySize int64 = 1 << 20 // 1MB

	// MaxLessonPlanUploadSize is the maximum size for lesson plan uploads (50MB)
	MaxLessonPlanUploadSize int64 = 50 << 20 // 50MB

	// MaxCodeSize is the maximum size for source code submissions (100KB)
	MaxCodeSize int = 100 * 1024

	// MaxTestCaseInputSize is the maximum size for test case input (10KB)
	MaxTestCaseInputSize int = 10 * 1024

	// MaxTestCaseOutputSize is the maximum size for test case expected output (10KB)
	MaxTestCaseOutputSize int = 10 * 1024

	// MaxProblemDescriptionSize is the maximum size for problem descriptions (50KB)
	MaxProblemDescriptionSize int = 50 * 1024

	// MaxTitleLength is the maximum length for titles (300 chars)
	MaxTitleLength int = 300

	// MaxDescriptionLength is the maximum length for short descriptions (5000 chars)
	MaxDescriptionLength int = 5000

	// MaxNameLength is the maximum length for names (200 chars)
	MaxNameLength int = 200

	// MaxEmailLength is the maximum length for emails (255 chars)
	MaxEmailLength int = 255

	// MaxPasswordLength is the maximum length for passwords (128 chars - prevents DoS from bcrypt)
	MaxPasswordLength int = 128

	// MaxTimeTakenSeconds is the maximum time taken that can be reported (24 hours)
	MaxTimeTakenSeconds float64 = 86400
)

// =====================================================
// Request Body Size Middleware
// =====================================================

// RequestBodyLimitMiddleware limits the size of request bodies to prevent memory exhaustion
func RequestBodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Limit request body size
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

		// Check content-length header if present
		if c.Request.ContentLength > maxBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("Request body too large. Maximum allowed: %d bytes (%.1f MB)", maxBytes, float64(maxBytes)/(1024*1024)),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// DefaultRequestBodyLimitMiddleware applies the default 1MB limit
func DefaultRequestBodyLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if this is a route that needs larger upload limits
		// Lesson plan uploads need 50MB
		if c.FullPath() == "/api/admin/lesson-plans/upload" {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxLessonPlanUploadSize)
			if c.Request.ContentLength > MaxLessonPlanUploadSize {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": fmt.Sprintf("Request body too large. Maximum allowed: %d bytes (%.1f MB)", MaxLessonPlanUploadSize, float64(MaxLessonPlanUploadSize)/(1024*1024)),
				})
				c.Abort()
				return
			}
		} else {
			// Default 1MB limit for all other routes
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRequestBodySize)
			if c.Request.ContentLength > MaxRequestBodySize {
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": fmt.Sprintf("Request body too large. Maximum allowed: %d bytes (%.1f MB)", MaxRequestBodySize, float64(MaxRequestBodySize)/(1024*1024)),
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// LessonPlanUploadLimitMiddleware applies the 50MB limit for lesson plan uploads
func LessonPlanUploadLimitMiddleware() gin.HandlerFunc {
	return RequestBodyLimitMiddleware(MaxLessonPlanUploadSize)
}

// =====================================================
// Custom Validators
// =====================================================

// RegisterCustomValidators registers custom validators with gin
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// Register custom validator for code size (used with struct tag)
		_ = v.RegisterValidation("codesize", func(fl validator.FieldLevel) bool {
			code := fl.Field().String()
			return len(code) <= MaxCodeSize
		})

		// Register custom validator for test case input size
		_ = v.RegisterValidation("tcinputsize", func(fl validator.FieldLevel) bool {
			input := fl.Field().String()
			return len(input) <= MaxTestCaseInputSize
		})

		// Register custom validator for test case output size
		_ = v.RegisterValidation("tcoutputsize", func(fl validator.FieldLevel) bool {
			output := fl.Field().String()
			return len(output) <= MaxTestCaseOutputSize
		})

		// Register custom validator for time taken
		_ = v.RegisterValidation("timetaken", func(fl validator.FieldLevel) bool {
			timeTaken := fl.Field().Float()
			return timeTaken >= 0 && timeTaken <= MaxTimeTakenSeconds
		})

		// Register custom validator for problem description size
		_ = v.RegisterValidation("problemdesc", func(fl validator.FieldLevel) bool {
			desc := fl.Field().String()
			return len(desc) <= MaxProblemDescriptionSize
		})
	}
}

// =====================================================
// Validation Helper Functions
// =====================================================

// ValidateCodeSize checks if code size is within limits
func ValidateCodeSize(code string) error {
	if len(code) == 0 {
		return fmt.Errorf("source code is required")
	}
	if len(code) > MaxCodeSize {
		return fmt.Errorf("code size exceeds limit. Maximum allowed: %d KB", MaxCodeSize/1024)
	}
	return nil
}

// ValidateTimeTaken checks if time taken is within reasonable bounds
func ValidateTimeTaken(timeTaken float64) error {
	if timeTaken < 0 {
		return fmt.Errorf("time taken cannot be negative")
	}
	if timeTaken > MaxTimeTakenSeconds {
		return fmt.Errorf("time taken exceeds maximum allowed (24 hours)")
	}
	return nil
}

// ValidateTitle checks if title is within limits
func ValidateTitle(title string) error {
	if len(title) == 0 {
		return fmt.Errorf("title is required")
	}
	if len(title) > MaxTitleLength {
		return fmt.Errorf("title exceeds maximum length of %d characters", MaxTitleLength)
	}
	return nil
}

// ValidatePassword checks if password meets requirements
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("password exceeds maximum length of %d characters", MaxPasswordLength)
	}
	return nil
}
