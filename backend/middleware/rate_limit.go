package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// =====================================================
// Rate Limiter for Code Execution
// =====================================================

// RateLimitEntry tracks rate limit state for a single client
type RateLimitEntry struct {
	ResetAt   time.Time
	BlockedAt time.Time
	Count     int
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	requests map[string]*RateLimitEntry
	mu       sync.RWMutex
	limit    int           // Max requests allowed
	window   time.Duration // Time window
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		requests: make(map[string]*RateLimitEntry),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine to remove old entries
	go limiter.cleanup()

	return limiter
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.requests[key]

	if !exists || now.After(entry.ResetAt) {
		// New window or expired entry
		rl.requests[key] = &RateLimitEntry{
			Count:   1,
			ResetAt: now.Add(rl.window),
		}
		return true
	}

	// Within current window
	if entry.Count >= rl.limit {
		// Rate limit exceeded
		return false
	}

	entry.Count++
	return true
}

// Remaining returns the number of requests remaining in the current window
func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	entry, exists := rl.requests[key]
	if !exists || time.Now().After(entry.ResetAt) {
		return rl.limit
	}

	remaining := rl.limit - entry.Count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ResetAt returns when the rate limit will reset for a key
func (rl *RateLimiter) ResetAt(key string) time.Time {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	entry, exists := rl.requests[key]
	if !exists {
		return time.Now()
	}
	return entry.ResetAt
}

// Limit returns the maximum requests allowed
func (rl *RateLimiter) Limit() int {
	return rl.limit
}

// cleanup periodically removes expired entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.requests {
			if now.After(entry.ResetAt) {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}

// =====================================================
// Rate Limit Middleware
// =====================================================

// Global rate limiters for different endpoints
var (
	// LoginLimiter limits login attempts (IP-based enforcement currently disabled)
	LoginLimiter = NewRateLimiter(5, time.Minute)
	// GlobalLimiter limits all requests globally (IP-based enforcement currently disabled)
	GlobalLimiter = NewRateLimiter(100, time.Minute)
	// SubmitCodeLimiter limits code submission rate (10 per minute)
	SubmitCodeLimiter = NewRateLimiter(10, time.Minute)
	// RunCodeLimiter limits code execution rate (20 per minute)
	RunCodeLimiter = NewRateLimiter(20, time.Minute)
	// ContestSubmitLimiter limits contest submission rate (5 per minute)
	ContestSubmitLimiter = NewRateLimiter(5, time.Minute)
	// PasswordResetLimiter limits password reset attempts (3 per minute)
	PasswordResetLimiter = NewRateLimiter(3, time.Minute)
)

// RateLimitMiddleware creates a gin middleware for rate limiting
// Uses user's regd_no if authenticated, otherwise falls back to IP
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use user's regd_no as key if authenticated
		var key string
		regdno, exists := c.Get("regdno")
		if exists {
			key = regdno.(string)
		} else {
			// Fallback to client IP for unauthenticated requests
			key = c.ClientIP()
		}

		if !limiter.Allow(key) {
			resetAt := limiter.ResetAt(key)
			retryAfter := int(time.Until(resetAt).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}

			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.Limit()))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded. Please wait before trying again.",
				"retry_after": retryAfter,
				"reset_at":    resetAt.Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		// Add rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.Limit()))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(limiter.Remaining(key)))

		c.Next()
	}
}

// IPRateLimitMiddleware is intentionally disabled.
// Shared NAT/proxy IPs caused users to block each other, so IP-based throttling is bypassed.
func IPRateLimitMiddleware(_ *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// LoginRateLimitMiddleware uses IP middleware.
// IP middleware is currently bypassed to avoid shared-IP throttling.
func LoginRateLimitMiddleware() gin.HandlerFunc {
	return IPRateLimitMiddleware(LoginLimiter)
}

// GlobalRateLimitMiddleware uses IP middleware.
// IP middleware is currently bypassed to avoid shared-IP throttling.
func GlobalRateLimitMiddleware() gin.HandlerFunc {
	return IPRateLimitMiddleware(GlobalLimiter)
}

// SubmitRateLimitMiddleware is a convenience middleware for submit endpoint
func SubmitRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(SubmitCodeLimiter)
}

// RunRateLimitMiddleware is a convenience middleware for run endpoint
func RunRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(RunCodeLimiter)
}

// ContestRateLimitMiddleware is a convenience middleware for contest submissions
func ContestRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(ContestSubmitLimiter)
}

// PasswordResetRateLimitMiddleware is a strict rate limiter for password reset
func PasswordResetRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(PasswordResetLimiter)
}

// CombinedRateLimitMiddleware applies multiple rate limiters in sequence
// Stops at the first one that fails
func CombinedRateLimitMiddleware(limiters ...*RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var key string
		regdno, exists := c.Get("regdno")
		if exists {
			key = regdno.(string)
		} else {
			key = c.ClientIP()
		}

		for _, limiter := range limiters {
			if !limiter.Allow(key) {
				resetAt := limiter.ResetAt(key)
				retryAfter := int(time.Until(resetAt).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}

				c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.Limit()))
				c.Header("X-RateLimit-Remaining", "0")
				c.Header("Retry-After", strconv.Itoa(retryAfter))
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":       "Rate limit exceeded. Please wait before trying again.",
					"retry_after": retryAfter,
					"reset_at":    resetAt.Format(time.RFC3339),
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// GetRateLimitStats returns current rate limit statistics (for monitoring)
func GetRateLimitStats() map[string]interface{} {
	return map[string]interface{}{
		"login_requests":          len(LoginLimiter.requests),
		"global_requests":         len(GlobalLimiter.requests),
		"submit_requests":         len(SubmitCodeLimiter.requests),
		"run_requests":            len(RunCodeLimiter.requests),
		"contest_requests":        len(ContestSubmitLimiter.requests),
		"password_reset_requests": len(PasswordResetLimiter.requests),
	}
}

// init logs rate limiter initialization
func init() {
	// Log initialization
	fmt.Printf("[RateLimiter] Initialized rate limiters:\n")
	fmt.Printf("  - Login: 5 requests/minute (IP enforcement disabled)\n")
	fmt.Printf("  - Global: 100 requests/minute (IP enforcement disabled)\n")
	fmt.Printf("  - Submit: 10 requests/minute per user\n")
	fmt.Printf("  - Run: 20 requests/minute per user\n")
	fmt.Printf("  - Contest: 5 requests/minute per user\n")
	fmt.Printf("  - Password Reset: 3 requests/minute per user\n")
}
