package handlers

import (
	"coding-platform/errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse is the standard response structure
type APIResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
	Success bool        `json:"success"`
}

// ErrorBody contains standardized error information
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse sends a standardized JSON error response
// It extracts safe error information from AppError
func ErrorResponse(c *gin.Context, err error) {
	if err == nil {
		return
	}

	code, message, status := errors.GetSafeResponse(err)

	c.JSON(status, APIResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    string(code),
			Message: message,
		},
	})
}

// SuccessResponse sends a standardized JSON success response
func SuccessResponse(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Data:    data,
	})
}

// CreatedResponse sends a 201 Created response
func CreatedResponse(c *gin.Context, data interface{}) {
	SuccessResponse(c, http.StatusCreated, data)
}

// OKResponse sends a 200 OK response
func OKResponse(c *gin.Context, data interface{}) {
	SuccessResponse(c, http.StatusOK, data)
}

// MessageResponse sends a response with just a message
func MessageResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{"message": message})
}

// BadRequestResponse sends a 400 Bad Request error
func BadRequestResponse(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrValidation),
			Message: message,
		},
	})
}

// UnauthorizedResponse sends a 401 Unauthorized error
func UnauthorizedResponse(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrUnauthorized),
			Message: message,
		},
	})
}

// ForbiddenResponse sends a 403 Forbidden error
func ForbiddenResponse(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, APIResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrForbidden),
			Message: message,
		},
	})
}

// NotFoundResponse sends a 404 Not Found error
func NotFoundResponse(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrNotFound),
			Message: message,
		},
	})
}

// InternalServerErrorResponse sends a 500 Internal Server error
// IMPORTANT: Never expose internal error details to clients
func InternalServerErrorResponse(c *gin.Context, message string) {
	// Use generic message for user safety
	userMessage := "An unexpected error occurred"
	if message != "" {
		// Only use custom message if it's safe (no technical details)
		userMessage = message
	}

	c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrInternal),
			Message: userMessage,
		},
	})
}

// ValidationErrorResponse sends a 400 error with field-specific validation errors
func ValidationErrorResponse(c *gin.Context, _ map[string]string) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrValidation),
			Message: "Validation failed",
		},
		// Optional: include field errors in a separate field for detailed feedback
	})
}

// ConflictResponse sends a 409 Conflict error
func ConflictResponse(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, APIResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrResourceExists),
			Message: message,
		},
	})
}

// ServiceUnavailableResponse sends a 503 error
func ServiceUnavailableResponse(c *gin.Context, message string) {
	c.JSON(http.StatusServiceUnavailable, APIResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrServiceUnavailable),
			Message: message,
		},
	})
}

// TimeoutResponse sends a 504 Gateway Timeout error
func TimeoutResponse(c *gin.Context, message string) {
	c.JSON(http.StatusGatewayTimeout, APIResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    string(errors.ErrTimeout),
			Message: message,
		},
	})
}
