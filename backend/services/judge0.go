package services

import (
	"bytes"
	"coding-platform/config"
	"coding-platform/errors"
	"coding-platform/logger"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// =====================================================
// Circuit Breaker for Judge0
// =====================================================

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	MaxFailures   int           // Number of failures before opening circuit
	Timeout       time.Duration // Duration circuit stays open
	HalfOpenLimit int           // Max requests in half-open state
}

// DefaultCircuitBreakerConfig returns default configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:   5,
		Timeout:       30 * time.Second,
		HalfOpenLimit: 3,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu              sync.RWMutex
	state           CircuitState
	failures        int
	lastFailureTime time.Time
	halfOpenCount   int
	config          CircuitBreakerConfig
	name            string
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:  CircuitClosed,
		config: config,
		name:   name,
	}
}

// AllowRequest checks if a request should be allowed
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenCount = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		if cb.halfOpenCount < cb.config.HalfOpenLimit {
			cb.halfOpenCount++
			return true
		}
		return false
	}
	return false
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitHalfOpen:
		cb.state = CircuitClosed
		cb.failures = 0
	case CircuitClosed:
		// Reset failures on success
		cb.failures = 0
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.config.MaxFailures {
			cb.state = CircuitOpen
			logger.GetGlobal().Warn("Circuit breaker opened",
				logger.KV("name", cb.name),
				logger.KV("failures", cb.failures),
			)
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		logger.GetGlobal().Warn("Circuit breaker re-opened",
			logger.KV("name", cb.name),
		)
	}
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// StateString returns human-readable state
func (cb *CircuitBreaker) StateString() string {
	switch cb.State() {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF-OPEN"
	}
	return "UNKNOWN"
}

// Global circuit breaker for Judge0
var judge0CircuitBreaker = NewCircuitBreaker("judge0", DefaultCircuitBreakerConfig())

// =====================================================
// Retry Configuration
// =====================================================

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries     int           // Maximum number of retry attempts
	InitialDelay   time.Duration // Initial delay between retries
	MaxDelay       time.Duration // Maximum delay between retries
	Multiplier     float64       // Delay multiplier for exponential backoff
	RetryableCodes []int         // HTTP status codes that trigger retry
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       5 * time.Second,
		Multiplier:     2.0,
		RetryableCodes: []int{408, 425, 429, 500, 502, 503, 504},
	}
}

// Global retry config for Judge0
var judge0RetryConfig = DefaultRetryConfig()

// =====================================================
// HTTP Client with Timeout
// =====================================================

// judge0HTTPClient is a shared HTTP client with configured timeouts
var judge0HTTPClient = &http.Client{
	Timeout: 60 * time.Second, // Total request timeout
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	},
}

// =====================================================
// Judge0 Service Types
// =====================================================

type Judge0Submission struct {
	SourceCode     string `json:"source_code"`
	LanguageID     int    `json:"language_id"`
	Stdin          string `json:"stdin,omitempty"`
	ExpectedOutput string `json:"expected_output,omitempty"`
	CPUTimeLimit   string `json:"cpu_time_limit,omitempty"`
	MemoryLimit    int    `json:"memory_limit,omitempty"`
	WallTimeLimit  string `json:"wall_time_limit,omitempty"`
}

type Judge0Response struct {
	Token  string `json:"token"`
	Status struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
	} `json:"status"`
	Stdout        *string `json:"stdout"`
	Stderr        *string `json:"stderr"`
	CompileOutput *string `json:"compile_output"`
	Message       *string `json:"message"`
	Time          *string `json:"time"`
	Memory        *int    `json:"memory"`
}

type TestResult struct {
	Passed         bool    `json:"passed"`
	StatusID       int     `json:"status_id"`
	Status         string  `json:"status"`
	Time           float64 `json:"time"`
	Memory         int     `json:"memory"`
	Stdout         string  `json:"stdout"`
	Stderr         string  `json:"stderr"`
	CompileOutput  string  `json:"compile_output"`
	ExpectedOutput string  `json:"expected_output"`
	Input          string  `json:"input"`
	Points         int     `json:"points"`
	EarnedPoints   int     `json:"earned_points"`
	IsHidden       bool    `json:"is_hidden"`
}

// =====================================================
// Judge0 Service Functions
// =====================================================

// SubmitCode submits code to Judge0 with circuit breaker and retry logic
func SubmitCode(sourceCode string, languageID int, stdin, expectedOutput string, timeLimitMs, memoryLimitKB int) (*TestResult, error) {
	log := logger.GetGlobal().With("operation", "judge0_submit")

	// Check circuit breaker
	if !judge0CircuitBreaker.AllowRequest() {
		log.Warn("Judge0 circuit breaker is open",
			logger.KV("state", judge0CircuitBreaker.StateString()),
		)
		return nil, errors.NewServiceUnavailable("Code execution service is temporarily unavailable")
	}

	// Convert time limit from ms to seconds (Judge0 expects seconds as string)
	timeLimitSec := fmt.Sprintf("%.1f", float64(timeLimitMs)/1000.0)

	// CRITICAL FIX: Ensure stdin is never empty to prevent EOFError
	if stdin == "" {
		stdin = "\n"
	}

	submission := Judge0Submission{
		SourceCode:     base64.StdEncoding.EncodeToString([]byte(sourceCode)),
		LanguageID:     languageID,
		Stdin:          base64.StdEncoding.EncodeToString([]byte(stdin)),
		ExpectedOutput: base64.StdEncoding.EncodeToString([]byte(expectedOutput)),
		CPUTimeLimit:   timeLimitSec,
		MemoryLimit:    memoryLimitKB,
		WallTimeLimit:  timeLimitSec,
	}

	jsonData, err := json.Marshal(submission)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "Failed to prepare submission")
	}

	url := fmt.Sprintf("%s/submissions?base64_encoded=true&wait=true", config.AppConfig.Judge0URL)

	// Execute with retry
	var judge0Resp Judge0Response
	var lastErr error

	for attempt := 0; attempt <= judge0RetryConfig.MaxRetries; attempt++ {
		judge0Resp, lastErr = executeJudge0Request(url, jsonData)
		if lastErr == nil {
			judge0CircuitBreaker.RecordSuccess()
			break
		}

		// Check if error is retryable
		if !isRetryableError(lastErr, judge0Resp.Status.ID) {
			break
		}

		// Wait before retry (exponential backoff)
		if attempt < judge0RetryConfig.MaxRetries {
			delay := calculateBackoff(attempt)
			log.Debug("Retrying Judge0 request",
				logger.KV("attempt", attempt+1),
				logger.KV("delay_ms", delay.Milliseconds()),
			)
			time.Sleep(delay)
		}
	}

	if lastErr != nil {
		judge0CircuitBreaker.RecordFailure()
		log.Error("Judge0 request failed after retries", lastErr,
			logger.KV("attempts", judge0RetryConfig.MaxRetries+1),
		)
		return nil, errors.Wrap(lastErr, errors.ErrExternalService, "Code execution failed")
	}

	// Parse result
	result := &TestResult{
		StatusID:       judge0Resp.Status.ID,
		Status:         judge0Resp.Status.Description,
		Input:          stdin,
		ExpectedOutput: expectedOutput,
		IsHidden:       false,
	}

	// Decode base64 outputs
	if judge0Resp.Stdout != nil {
		decoded, _ := base64.StdEncoding.DecodeString(*judge0Resp.Stdout)
		result.Stdout = string(decoded)
	}
	if judge0Resp.Stderr != nil {
		decoded, _ := base64.StdEncoding.DecodeString(*judge0Resp.Stderr)
		result.Stderr = string(decoded)
	}
	if judge0Resp.CompileOutput != nil {
		decoded, _ := base64.StdEncoding.DecodeString(*judge0Resp.CompileOutput)
		result.CompileOutput = string(decoded)
	}

	// Parse time and memory
	if judge0Resp.Time != nil {
		_, _ = fmt.Sscanf(*judge0Resp.Time, "%f", &result.Time)
	}
	if judge0Resp.Memory != nil {
		result.Memory = *judge0Resp.Memory
	}

	// Status ID 3 = Accepted
	result.Passed = expectedOutput != "" && judge0Resp.Status.ID == 3

	return result, nil
}

// executeJudge0Request executes a single HTTP request to Judge0
func executeJudge0Request(url string, jsonData []byte) (Judge0Response, error) {
	var emptyResp Judge0Response

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return emptyResp, errors.Wrap(err, errors.ErrInternal, "Failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := judge0HTTPClient.Do(req)
	if err != nil {
		return emptyResp, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return emptyResp, errors.Wrap(err, errors.ErrInternal, "Failed to read response")
	}

	// Check for HTTP errors
	if resp.StatusCode >= 500 {
		return emptyResp, errors.Errorf(errors.ErrExternalService, "Judge0 server error: %d", resp.StatusCode)
	}

	if err := json.Unmarshal(body, &emptyResp); err != nil {
		return emptyResp, errors.Wrap(err, errors.ErrInternal, "Failed to parse response")
	}

	// Check for Judge0 error
	if emptyResp.Message != nil && *emptyResp.Message != "" {
		return emptyResp, errors.New(errors.ErrExternalService, *emptyResp.Message)
	}

	return emptyResp, nil
}

// isRetryableError checks if an error should trigger retry
func isRetryableError(err error, statusID int) bool {
	if err == nil {
		return false
	}

	// Check if it's an AppError with retryable status
	if appErr, ok := err.(*errors.AppError); ok {
		for _, code := range judge0RetryConfig.RetryableCodes {
			if appErr.HTTPStatus == code {
				return true
			}
		}
	}

	// Retry on specific Judge0 status codes
	// 1 = In Queue, 2 = Processing (wait longer)
	if statusID == 1 || statusID == 2 {
		return true
	}

	return false
}

// calculateBackoff calculates exponential backoff delay
func calculateBackoff(attempt int) time.Duration {
	delay := float64(judge0RetryConfig.InitialDelay) *
		float64(int(judge0RetryConfig.Multiplier*float64(int(judge0RetryConfig.Multiplier)*attempt)))

	if delay > float64(judge0RetryConfig.MaxDelay) {
		delay = float64(judge0RetryConfig.MaxDelay)
	}

	return time.Duration(delay)
}

// GetSubmission gets submission status with circuit breaker
func GetSubmission(token string) (*Judge0Response, error) {
	// Check circuit breaker
	if !judge0CircuitBreaker.AllowRequest() {
		return nil, errors.NewServiceUnavailable("Code execution service is temporarily unavailable")
	}

	url := fmt.Sprintf("%s/submissions/%s?base64_encoded=true", config.AppConfig.Judge0URL, token)

	resp, err := judge0HTTPClient.Get(url)
	if err != nil {
		judge0CircuitBreaker.RecordFailure()
		return nil, errors.Wrap(err, errors.ErrExternalService, "Failed to get submission status")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "Failed to read response")
	}

	var judge0Resp Judge0Response
	if err := json.Unmarshal(body, &judge0Resp); err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "Failed to parse response")
	}

	judge0CircuitBreaker.RecordSuccess()
	return &judge0Resp, nil
}

// WaitForSubmission waits for submission completion with timeout
func WaitForSubmission(token string, maxWaitSeconds int) (*Judge0Response, error) {
	log := logger.GetGlobal().With("operation", "judge0_wait")

	timeout := time.After(time.Duration(maxWaitSeconds) * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, errors.NewTimeout("waiting for code execution")
		case <-ticker.C:
			resp, err := GetSubmission(token)
			if err != nil {
				return nil, err
			}

			// Status ID 1 = In Queue, 2 = Processing
			if resp.Status.ID != 1 && resp.Status.ID != 2 {
				return resp, nil
			}

			log.Debug("Waiting for submission",
				logger.KV("status", resp.Status.Description),
				logger.KV("status_id", resp.Status.ID),
			)
		}
	}
}

// IsLanguageSupported checks if a language ID is supported by Judge0
func IsLanguageSupported(languageID int) bool {
	// Common supported languages
	supported := map[int]bool{
		45: true, // Assembly (NASM 2.14.02)
		46: true, // Bash (5.0.0)
		47: true, // Basic (FBC 1.07.1)
		48: true, // C (GCC 7.4.0)
		49: true, // C (GCC 8.3.0)
		50: true, // C# (Mono 6.6.0.161)
		51: true, // C++ (GCC 7.4.0)
		52: true, // C++ (GCC 8.3.0)
		53: true, // C++ (GCC 9.2.0)
		54: true, // C++ (Clang 7.0.1)
		55: true, // Common Lisp (SBCL 2.0.0)
		56: true, // D (DMD 2.089.1)
		57: true, // Elixir (1.9.4)
		58: true, // Erlang (OTP 22.2)
		59: true, // Fortran (GFortran 9.2.0)
		60: true, // Go (1.13.5)
		61: true, // Haskell (GHC 8.8.1)
		62: true, // Java (OpenJDK 13.0.1)
		63: true, // JavaScript (Node.js 12.14.0)
		64: true, // Lua (5.3.5)
		68: true, // PHP (7.4.1)
		70: true, // Python (3.8.1)
		71: true, // Python (3.8.1)
		72: true, // Ruby (2.7.0)
		73: true, // Rust (1.40.0)
		74: true, // TypeScript (3.7.4)
		75: true, // C (Clang 7.0.1)
		76: true, // C++ (Clang 10.0.1)
		77: true, // COBOL (GnuCOBOL 2.2)
		78: true, // Kotlin (1.3.70)
		79: true, // Objective-C (Clang 7.0.1)
		80: true, // R (4.0.0)
		81: true, // Scala (2.13.2)
		82: true, // SQL (SQLite 3.27.2)
		83: true, // Swift (5.2.3)
		84: true, // Visual Basic.Net (vbnc 0.0.0.5943)
	}
	return supported[languageID]
}

// GetCircuitBreakerState returns the current circuit breaker state for monitoring
func GetCircuitBreakerState() string {
	return judge0CircuitBreaker.StateString()
}
