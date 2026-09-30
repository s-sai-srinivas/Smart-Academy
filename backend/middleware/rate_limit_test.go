package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)

	// Should allow first 3 requests
	for i := 0; i < 3; i++ {
		if !limiter.Allow("test-key") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked
	if limiter.Allow("test-key") {
		t.Error("4th request should be blocked")
	}
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	// Each key should have its own limit
	if !limiter.Allow("key1") {
		t.Error("key1 first request should be allowed")
	}
	if !limiter.Allow("key2") {
		t.Error("key2 first request should be allowed")
	}
	if !limiter.Allow("key1") {
		t.Error("key1 second request should be allowed")
	}
	if !limiter.Allow("key2") {
		t.Error("key2 second request should be allowed")
	}

	// Both should now be blocked
	if limiter.Allow("key1") {
		t.Error("key1 third request should be blocked")
	}
	if limiter.Allow("key2") {
		t.Error("key2 third request should be blocked")
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	// Use a very short window for testing
	limiter := NewRateLimiter(2, 100*time.Millisecond)

	// Use up the limit
	limiter.Allow("test-key")
	limiter.Allow("test-key")

	// Should be blocked
	if limiter.Allow("test-key") {
		t.Error("Should be blocked after using limit")
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow("test-key") {
		t.Error("Should be allowed after window reset")
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)

	// Initially should have full limit remaining
	if rem := limiter.Remaining("test-key"); rem != 5 {
		t.Errorf("Expected 5 remaining, got %d", rem)
	}

	// After one request
	limiter.Allow("test-key")
	if rem := limiter.Remaining("test-key"); rem != 4 {
		t.Errorf("Expected 4 remaining, got %d", rem)
	}

	// After using all
	for i := 0; i < 4; i++ {
		limiter.Allow("test-key")
	}
	if rem := limiter.Remaining("test-key"); rem != 0 {
		t.Errorf("Expected 0 remaining, got %d", rem)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}
}

func TestIPRateLimitMiddleware(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	router := gin.New()
	router.Use(IPRateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	// First two requests from same IP should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}

	// Third request should also succeed because IP limiting is bypassed
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoginRateLimitMiddleware(t *testing.T) {
	router := gin.New()
	router.POST("/login", LoginRateLimitMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "login"})
	})

	// Make 5 login attempts (the limit)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Login attempt %d: Expected status 200, got %d", i+1, w.Code)
		}
	}

	// 6th attempt should also succeed because IP limiting is bypassed
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("6th login attempt: Expected status 200, got %d", w.Code)
	}
}

func TestRateLimitHeaders(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)

	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check headers are set
	if w.Header().Get("X-RateLimit-Limit") != "3" {
		t.Errorf("Expected X-RateLimit-Limit header to be '3', got '%s'", w.Header().Get("X-RateLimit-Limit"))
	}

	if w.Header().Get("X-RateLimit-Remaining") != "2" {
		t.Errorf("Expected X-RateLimit-Remaining header to be '2', got '%s'", w.Header().Get("X-RateLimit-Remaining"))
	}
}

func TestGetRateLimitStats(t *testing.T) {
	// Just verify it doesn't panic and returns a map
	stats := GetRateLimitStats()

	if stats == nil {
		t.Error("GetRateLimitStats should return non-nil map")
	}

	// Should have entries for all limiters
	expectedKeys := []string{
		"login_requests",
		"global_requests",
		"submit_requests",
		"run_requests",
		"contest_requests",
		"password_reset_requests",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stats to have key '%s'", key)
		}
	}
}
