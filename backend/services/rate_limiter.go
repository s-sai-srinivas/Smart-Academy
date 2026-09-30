package services

import (
	"context"
	"log"
	"sync"
	"time"
)

// =====================================================
// API Rate Limiter for Free Tier API
// =====================================================
// Implements token bucket algorithm with:
// - Requests Per Minute (RPM) tracking
// - Tokens Per Minute (TPM) tracking
// - Automatic waiting when limits are approached
// - Thread-safe implementation
// =====================================================

// APIRateLimiterConfig holds configuration for the rate limiter
type APIRateLimiterConfig struct {
	RequestsPerMinute int     // Max requests per minute (Gemini free tier: 15)
	TokensPerMinute   int     // Max tokens per minute (Gemini free tier: 1,000,000)
	SafetyMargin      float64 // Safety margin (0.0-1.0, default 0.9 to use 90% of limits)
}

// DefaultAPIRateLimiterConfig returns default config for Gemini free tier
func DefaultAPIRateLimiterConfig() APIRateLimiterConfig {
	return APIRateLimiterConfig{
		RequestsPerMinute: 500,     // Gemini free tier limit
		TokensPerMinute:   1000000, // Gemini free tier limit
		SafetyMargin:      0.9,     // Use 90% of limits to be safe
	}
}

// APIRateLimiter implements rate limiting for API calls
type APIRateLimiter struct {
	lastResetTime time.Time
	config        APIRateLimiterConfig
	requestCount  int
	tokenCount    int
	totalRequests int
	totalTokens   int
	throttleCount int
	mu            sync.Mutex
}

// NewAPIRateLimiter creates a new rate limiter
func NewAPIRateLimiter(config APIRateLimiterConfig) *APIRateLimiter {
	return &APIRateLimiter{
		config:        config,
		lastResetTime: time.Now(),
	}
}

// WaitForSlot waits until an API call can be made without exceeding limits
// Returns the wait duration (0 if no wait was needed)
func (r *APIRateLimiter) WaitForSlot(ctx context.Context, estimatedTokens int) (time.Duration, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	// Reset counters if a minute has passed
	if now.Sub(r.lastResetTime) >= time.Minute {
		r.requestCount = 0
		r.tokenCount = 0
		r.lastResetTime = now
	}

	// Calculate effective limits with safety margin
	effectiveRPM := int(float64(r.config.RequestsPerMinute) * r.config.SafetyMargin)
	effectiveTPM := int(float64(r.config.TokensPerMinute) * r.config.SafetyMargin)

	// Check if we need to wait
	var waitDuration time.Duration

	if r.requestCount >= effectiveRPM {
		// Exceeded request limit
		waitDuration = time.Minute - now.Sub(r.lastResetTime) + 5*time.Second
		log.Printf("[APIRateLimiter] Request limit reached (%d/%d), waiting %v",
			r.requestCount, effectiveRPM, waitDuration)
		r.throttleCount++
	} else if r.tokenCount+estimatedTokens >= effectiveTPM {
		// Would exceed token limit
		waitDuration = time.Minute - now.Sub(r.lastResetTime) + 5*time.Second
		log.Printf("[APIRateLimiter] Token limit approaching (%d + %d >= %d), waiting %v",
			r.tokenCount, estimatedTokens, effectiveTPM, waitDuration)
		r.throttleCount++
	}

	// Wait if needed
	if waitDuration > 0 {
		r.mu.Unlock() // Unlock before sleeping

		select {
		case <-ctx.Done():
			r.mu.Lock()
			return 0, ctx.Err()
		case <-time.After(waitDuration):
		}

		r.mu.Lock() // Re-lock after sleeping

		// Reset counters after waiting
		r.requestCount = 0
		r.tokenCount = 0
		r.lastResetTime = time.Now()
	}

	// Reserve the slot
	r.requestCount++
	r.tokenCount += estimatedTokens
	r.totalRequests++
	r.totalTokens += estimatedTokens

	log.Printf("[APIRateLimiter] Slot reserved: request %d/%d, tokens %d/%d",
		r.requestCount, effectiveRPM, r.tokenCount, effectiveTPM)

	return waitDuration, nil
}

// RecordActualTokens adjusts the token count if actual usage differs from estimate
func (r *APIRateLimiter) RecordActualTokens(estimatedTokens, actualTokens int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Adjust the current token count
	r.tokenCount += actualTokens - estimatedTokens
	r.totalTokens += actualTokens - estimatedTokens
}

// GetStats returns current rate limiter statistics
func (r *APIRateLimiter) GetStats() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	return map[string]interface{}{
		"current_requests":    r.requestCount,
		"current_tokens":      r.tokenCount,
		"total_requests":      r.totalRequests,
		"total_tokens":        r.totalTokens,
		"throttle_count":      r.throttleCount,
		"requests_per_minute": r.config.RequestsPerMinute,
		"tokens_per_minute":   r.config.TokensPerMinute,
		"time_until_reset_ms": time.Until(r.lastResetTime.Add(time.Minute)).Milliseconds(),
	}
}

// EstimateAPITokens estimates token count for content
// Rough estimate: 4 characters ≈ 1 token
func EstimateAPITokens(content string) int {
	return len(content)/4 + 100 // Add 100 for overhead
}

// =====================================================
// API Retry Helpers (uses judge0.go RetryConfig)
// =====================================================

// APIRetryableError represents an error that can be retried for API calls
type APIRetryableError struct {
	Message    string
	StatusCode int
	RetryAfter time.Duration
}

func (e *APIRetryableError) Error() string {
	return e.Message
}

// IsAPIRetryable checks if an error is retryable for API calls
func IsAPIRetryable(err error) bool {
	if retryErr, ok := err.(*APIRetryableError); ok {
		return retryErr.StatusCode == 429 || retryErr.StatusCode >= 500
	}
	return false
}
