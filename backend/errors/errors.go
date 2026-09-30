package errors

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// ErrorCode represents a standardized error code for API responses
type ErrorCode string

// Standard error codes
const (
	// Authentication errors
	ErrUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	ErrTokenInvalid       ErrorCode = "TOKEN_INVALID"
	ErrTokenRevoked       ErrorCode = "TOKEN_REVOKED"
	ErrInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"

	// Authorization errors
	ErrForbidden        ErrorCode = "FORBIDDEN"
	ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"

	// Resource errors
	ErrNotFound       ErrorCode = "NOT_FOUND"
	ErrResourceExists ErrorCode = "RESOURCE_EXISTS"

	// Validation errors
	ErrValidation   ErrorCode = "VALIDATION_ERROR"
	ErrInvalidInput ErrorCode = "INVALID_INPUT"
	ErrMissingField ErrorCode = "MISSING_FIELD"

	// System errors
	ErrInternal           ErrorCode = "INTERNAL_ERROR"
	ErrDatabase           ErrorCode = "DATABASE_ERROR"
	ErrExternalService    ErrorCode = "EXTERNAL_SERVICE_ERROR"
	ErrTimeout            ErrorCode = "TIMEOUT"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrTooManyRequests    ErrorCode = "TOO_MANY_REQUESTS"

	// Business logic errors
	ErrBusinessRule ErrorCode = "BUSINESS_RULE_VIOLATION"
	ErrInvalidState ErrorCode = "INVALID_STATE"
)

// ErrorLevel represents the severity level for logging
type ErrorLevel string

const (
	LevelDebug ErrorLevel = "debug"
	LevelInfo  ErrorLevel = "info"
	LevelWarn  ErrorLevel = "warn"
	LevelError ErrorLevel = "error"
	LevelFatal ErrorLevel = "fatal"
)

// AppError is the standardized application error type
type AppError struct {
	Err        error                  `json:"-"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Level      ErrorLevel             `json:"-"`
	Operation  string                 `json:"-"`
	Component  string                 `json:"-"`
	File       string                 `json:"-"`
	HTTPStatus int                    `json:"-"`
	Line       int                    `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// Is checks if the target error is of the same type
func (e *AppError) Is(target error) bool {
	if appErr, ok := target.(*AppError); ok {
		return e.Code == appErr.Code
	}
	return false
}

// WithDetails adds context details to the error
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

// WithDetail adds a single detail to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithLevel sets the logging level
func (e *AppError) WithLevel(level ErrorLevel) *AppError {
	e.Level = level
	return e
}

// WithOperation sets the operation that failed
func (e *AppError) WithOperation(op string) *AppError {
	e.Operation = op
	return e
}

// WithComponent sets the component name
func (e *AppError) WithComponent(comp string) *AppError {
	e.Component = comp
	return e
}

// setCallerInfo captures the caller's file and line
func (e *AppError) setCallerInfo() {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		// Extract just the filename
		parts := strings.Split(file, "/")
		if len(parts) > 0 {
			file = parts[len(parts)-1]
		}
		e.File = file
		e.Line = line
	}
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
	err := &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Level:      LevelError,
		Details:    make(map[string]interface{}),
	}
	err.setCallerInfo()
	return err
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code ErrorCode, message string) *AppError {
	appErr := &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		HTTPStatus: http.StatusInternalServerError,
		Level:      LevelError,
		Details:    make(map[string]interface{}),
	}
	appErr.setCallerInfo()
	return appErr
}

// Wrapf wraps an error with a formatted message
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *AppError {
	appErr := &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		Err:        err,
		HTTPStatus: http.StatusInternalServerError,
		Level:      LevelError,
		Details:    make(map[string]interface{}),
	}
	appErr.setCallerInfo()
	return appErr
}

// Errorf creates a new formatted error
func Errorf(code ErrorCode, format string, args ...interface{}) *AppError {
	err := &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: http.StatusInternalServerError,
		Level:      LevelError,
		Details:    make(map[string]interface{}),
	}
	err.setCallerInfo()
	return err
}

// Factory functions for common errors

// NewUnauthorized creates a 401 Unauthorized error
func NewUnauthorized(message string) *AppError {
	if message == "" {
		message = "Authentication required"
	}
	err := New(ErrUnauthorized, message)
	err.HTTPStatus = http.StatusUnauthorized
	return err
}

// NewTokenExpired creates a 401 Token Expired error
func NewTokenExpired() *AppError {
	err := New(ErrTokenExpired, "Session has expired. Please login again.")
	err.HTTPStatus = http.StatusUnauthorized
	return err
}

// NewTokenInvalid creates a 401 Invalid Token error
func NewTokenInvalid(message string) *AppError {
	if message == "" {
		message = "Invalid authentication token"
	}
	err := New(ErrTokenInvalid, message)
	err.HTTPStatus = http.StatusUnauthorized
	return err
}

// NewForbidden creates a 403 Forbidden error
func NewForbidden(message string) *AppError {
	if message == "" {
		message = "You do not have permission to access this resource"
	}
	err := New(ErrForbidden, message)
	err.HTTPStatus = http.StatusForbidden
	return err
}

// NewNotFound creates a 404 Not Found error
func NewNotFound(resource string) *AppError {
	if resource == "" {
		resource = "Resource"
	}
	err := New(ErrNotFound, fmt.Sprintf("%s not found", resource))
	err.HTTPStatus = http.StatusNotFound
	err.Level = LevelWarn
	return err
}

// NewValidation creates a 400 Validation Error
func NewValidation(message string) *AppError {
	if message == "" {
		message = "Invalid input"
	}
	err := New(ErrValidation, message)
	err.HTTPStatus = http.StatusBadRequest
	err.Level = LevelWarn
	return err
}

// NewInternal creates a 500 Internal Server Error
func NewInternal(message string) *AppError {
	if message == "" {
		message = "An unexpected error occurred"
	}
	err := New(ErrInternal, message)
	err.HTTPStatus = http.StatusInternalServerError
	return err
}

// NewDatabase creates a 500 Database Error (safe message)
func NewDatabase(operation string) *AppError {
	err := New(ErrDatabase, "Failed to process your request")
	err.HTTPStatus = http.StatusInternalServerError
	err.Operation = operation
	return err
}

// NewExternalService creates a 502 External Service Error
func NewExternalService(serviceName string) *AppError {
	err := Newf(ErrExternalService, "%s service is currently unavailable", serviceName)
	err.HTTPStatus = http.StatusBadGateway
	err.Level = LevelError
	return err
}

// NewTimeout creates a 504 Timeout Error
func NewTimeout(operation string) *AppError {
	err := Errorf(ErrTimeout, "Operation timed out: %s", operation)
	err.HTTPStatus = http.StatusGatewayTimeout
	return err
}

// NewConflict creates a 409 Conflict Error
func NewConflict(message string) *AppError {
	if message == "" {
		message = "Resource conflict"
	}
	err := New(ErrResourceExists, message)
	err.HTTPStatus = http.StatusConflict
	err.Level = LevelWarn
	return err
}

// NewServiceUnavailable creates a 503 Service Unavailable error
func NewServiceUnavailable(message string) *AppError {
	if message == "" {
		message = "Service is temporarily unavailable"
	}
	err := New(ErrServiceUnavailable, message)
	err.HTTPStatus = http.StatusServiceUnavailable
	err.Level = LevelError
	return err
}

// NewTooManyRequests creates a 429 Too Many Requests error
func NewTooManyRequests(message string) *AppError {
	if message == "" {
		message = "Too many requests. Please try again later."
	}
	err := New(ErrTooManyRequests, message)
	err.HTTPStatus = http.StatusTooManyRequests
	err.Level = LevelWarn
	return err
}

// Is checks if an error is of a specific error code
func Is(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}

// GetCode extracts the error code from an AppError
func GetCode(err error) ErrorCode {
	if err == nil {
		return ""
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrInternal
}

// GetHTTPStatus extracts the HTTP status code from an error
func GetHTTPStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}
	if appErr, ok := err.(*AppError); ok {
		if appErr.HTTPStatus > 0 {
			return appErr.HTTPStatus
		}
	}
	return http.StatusInternalServerError
}

// GetUserMessage returns the safe, user-friendly error message
func GetUserMessage(err error) string {
	if err == nil {
		return ""
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr.Message
	}
	// For non-AppErrors, return a generic message
	return "An unexpected error occurred"
}

// GetSafeResponse returns a safe error response for API consumers
func GetSafeResponse(err error) (code ErrorCode, message string, status int) {
	if err == nil {
		return "", "", http.StatusOK
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code, appErr.Message, appErr.HTTPStatus
	}
	// For unknown errors, return generic internal error
	return ErrInternal, "An unexpected error occurred", http.StatusInternalServerError
}

// Errorf is alias for New with formatting
func Newf(code ErrorCode, format string, args ...interface{}) *AppError {
	return Errorf(code, format, args...)
}
