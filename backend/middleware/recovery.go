package middleware

import (
	"coding-platform/logger"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// RecoveryMiddleware returns a middleware that recovers from panics
// and logs the full stack trace before returning 500
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Capture stack trace
				stack := debug.Stack()

				// Get logger
				log := logger.GetGlobal()

				// Log panic with full stack trace
				log.LogPanic(err, string(stack))

				// Get user info for context
				userRegdNo, _ := c.Get("regdno")
				if userRegdNo == nil {
					userRegdNo = "anonymous"
				}

				// Log additional context
				log.Error("Panic details", fmt.Errorf("%v", err),
					logger.KV("path", c.Request.URL.Path),
					logger.KV("method", c.Request.Method),
					logger.KV("user", userRegdNo),
					logger.KV("client_ip", c.ClientIP()),
				)

				// Return generic error to user (never expose internal details)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}

// RequestLoggerMiddleware returns a middleware that logs request details
// with structured logging and configurable detail level
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get client IP
		clientIP := c.ClientIP()

		// Get status code
		statusCode := c.Writer.Status()

		// Get request method
		method := c.Request.Method

		// Get user info if authenticated
		userRegdNo, exists := c.Get("regdno")
		userStr := "-"
		if exists && userRegdNo != nil {
			userStr = userRegdNo.(string)
		}

		// Get logger
		log := logger.GetGlobal()

		// Log request with structured logging
		log.LogRequest(method, path, clientIP, statusCode, latency, userStr)

		// Log errors with context
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				log.Error("Request error", e.Err,
					logger.KV("path", path),
					logger.KV("method", method),
					logger.KV("status", statusCode),
					logger.KV("user", userStr),
				)
			}
		}

		// Log slow requests (over 1 second)
		if latency > time.Second {
			log.Warn("Slow request detected",
				logger.KV("path", path),
				logger.KV("method", method),
				logger.KV("latency_ms", latency.Milliseconds()),
				logger.KV("user", userStr),
			)
		}
	}
}

// RecoveryMiddlewareWithWriter is a legacy function for compatibility
// Deprecated: Use RecoveryMiddleware instead
func RecoveryMiddlewareWithWriter() gin.HandlerFunc {
	return RecoveryMiddleware()
}

// RequestLoggerMiddlewareWithWriter is a legacy function for compatibility
// Deprecated: Use RequestLoggerMiddleware instead
func RequestLoggerMiddlewareWithWriter() gin.HandlerFunc {
	return RequestLoggerMiddleware()
}

// LogRequestBody logs the request body for debugging
// WARNING: Do not use in production for sensitive data
func LogRequestBody(c *gin.Context, bodyBytes []byte) {
	if len(bodyBytes) > 0 {
		log := logger.GetGlobal()
		log.Debug("Request body",
			logger.KV("path", c.Request.URL.Path),
			logger.KV("body", string(bodyBytes)),
		)
	}
}

// LogResponse logs response details
func LogResponse(c *gin.Context, statusCode int, responseBody []byte) {
	log := logger.GetGlobal()

	if statusCode >= 500 {
		log.Error("Server error response", nil,
			logger.KV("path", c.Request.URL.Path),
			logger.KV("status", statusCode),
			logger.KV("response", string(responseBody)),
		)
	} else if statusCode >= 400 {
		log.Warn("Client error response",
			logger.KV("path", c.Request.URL.Path),
			logger.KV("status", statusCode),
		)
	}
}

// BuildLogContext creates a logger with common request context
func BuildLogContext(c *gin.Context) *logger.Logger {
	log := logger.GetGlobal()

	// Add request context
	userRegdNo, exists := c.Get("regdno")
	if exists && userRegdNo != nil {
		log = log.With("user", userRegdNo)
	}

	log = log.With("path", c.Request.URL.Path)
	log = log.With("method", c.Request.Method)
	log = log.With("client_ip", c.ClientIP())

	return log
}
