package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"coding-platform/config"
	"coding-platform/models"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// =====================================================
// Job Queue Types and Constants
// =====================================================

// JobType defines the type of job
type JobType string

const (
	JobTypeRun    JobType = "RUN"    // Run code against sample test cases
	JobTypeSubmit JobType = "SUBMIT" // Submit code against all test cases
)

// Priority defines job priority for queue ordering
type Priority int

const (
	PriorityLow    Priority = 0 // Run code (sample tests)
	PriorityNormal Priority = 1 // Lab/course submissions, Practice
	PriorityHigh   Priority = 2 // Contest submissions (highest)
)

// JobStatus defines the current status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// Redis queue keys
const (
	QueueKeyHigh   = "queue:high"   // Contest submissions
	QueueKeyNormal = "queue:normal" // Lab/practice submissions
	QueueKeyLow    = "queue:low"    // Run code
	JobKeyPrefix   = "job:"         // Individual job data
)

// =====================================================
// Job Data Structures
// =====================================================

// TestCaseData represents test case data for a job
type TestCaseData struct {
	Input          string
	ExpectedOutput string
	TimeLimitMs    int
	MemoryLimitKB  int
	Points         int
	IsSample       bool
}

// ProblemData represents problem data for a job
type ProblemData struct {
	ID          uint
	TimeLimit   int
	MemoryLimit int
}

// Job represents a code execution job in the queue
type Job struct {
	CreatedAt    time.Time
	ContestID    *uint
	CompletedAt  *time.Time
	StartedAt    *time.Time
	Result       *JobResult
	CollegeID    string
	UserRegdNo   string
	ID           string
	Status       JobStatus
	ErrorMessage string
	SourceCode   string
	Type         JobType
	TestCases    []TestCaseData
	Problem      ProblemData
	Progress     int
	LanguageID   int
	RetryCount   int
	Priority     Priority
}

// JobResult represents the execution result of a job
type JobResult struct {
	ErrorMessage string
	TestResults  []TestResult
	PassedCount  int
	TotalTests   int
	Score        int
	MaxScore     int
	TotalTime    float64
	MaxMemory    int
	AllPassed    bool
}

// JobStatusResponse is the API response for job status
type JobStatusResponse struct {
	CreatedAt    time.Time    `json:"created_at"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
	StartedAt    *time.Time   `json:"started_at,omitempty"`
	ErrorMessage string       `json:"error_message,omitempty"`
	JobID        string       `json:"job_id"`
	Message      string       `json:"message,omitempty"`
	Status       JobStatus    `json:"status"`
	TestResults  []TestResult `json:"test_results,omitempty"`
	Score        int          `json:"score,omitempty"`
	MaxScore     int          `json:"max_score,omitempty"`
	RetryCount   int          `json:"retry_count"`
	Progress     int          `json:"progress"`
	AllPassed    bool         `json:"all_passed,omitempty"`
}

// =====================================================
// Job Queue Service
// =====================================================

// JobQueueService manages the Redis-based job queue
type JobQueueService struct {
	ctx             context.Context
	redisClient     *redis.Client
	workerSemaphore chan struct{}
	stopChan        chan struct{}
	wg              sync.WaitGroup
	activeWorkers   int32
	initialized     bool
}

// Global job queue instance
var jobQueue *JobQueueService
var jobQueueOnce sync.Once

// InitJobQueue initializes the global job queue service
func InitJobQueue() error {
	var initErr error
	jobQueueOnce.Do(func() {
		cfg := config.AppConfig

		if !cfg.RedisEnabled {
			initErr = fmt.Errorf("Redis must be enabled for job queue")
			return
		}

		ctx := context.Background()
		addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)

		// Use a separate Redis client for job queue (DB 1 to avoid cache conflicts)
		redisClient := redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB + 1, // Use separate DB for queue
		})

		// Test connection
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Printf("Job queue Redis connection failed: %v", err)
			initErr = err
			return
		}

		workerCount := cfg.JobQueueWorkers
		if workerCount < 10 {
			workerCount = 10
		} else if workerCount > 100 {
			workerCount = 100
		}

		jobQueue = &JobQueueService{
			redisClient:     redisClient,
			ctx:             ctx,
			workerSemaphore: make(chan struct{}, workerCount),
			stopChan:        make(chan struct{}),
			initialized:     true,
		}

		log.Printf("Job queue initialized with %d workers", workerCount)
	})

	return initErr
}

// GetJobQueue returns the global job queue instance
func GetJobQueue() *JobQueueService {
	return jobQueue
}

// StartWorkers starts the worker pool
func StartWorkers(numWorkers int) {
	if jobQueue == nil || !jobQueue.initialized {
		log.Println("Job queue not initialized, workers not started")
		return
	}

	for i := 0; i < numWorkers; i++ {
		jobQueue.wg.Add(1)
		go jobQueue.worker(i)
	}

	log.Printf("Started %d job queue workers", numWorkers)
}

// StopWorkers stops all workers gracefully
func StopWorkers() {
	if jobQueue == nil {
		return
	}

	close(jobQueue.stopChan)
	jobQueue.wg.Wait()
	log.Println("All job queue workers stopped")
}

// worker processes jobs from the queue
func (jq *JobQueueService) worker(id int) {
	defer jq.wg.Done()

	for {
		select {
		case <-jq.stopChan:
			log.Printf("Worker %d stopping", id)
			return
		default:
			// Wait for semaphore slot (limits concurrent Judge0 calls)
			jq.workerSemaphore <- struct{}{}

			// Pop job from highest priority queue
			job := jq.popJobFromQueue()

			if job == nil {
				// No jobs available, release slot and sleep briefly
				<-jq.workerSemaphore
				time.Sleep(200 * time.Millisecond)
				continue
			}

			atomic.AddInt32(&jq.activeWorkers, 1)
			jq.processJob(job)
			atomic.AddInt32(&jq.activeWorkers, -1)

			// Release semaphore slot
			<-jq.workerSemaphore
		}
	}
}

// popJobFromQueue pops a job from the highest priority queue with data
func (jq *JobQueueService) popJobFromQueue() *Job {
	// Try queues in priority order: HIGH > NORMAL > LOW
	queues := []string{QueueKeyHigh, QueueKeyNormal, QueueKeyLow}

	for _, queue := range queues {
		result, err := jq.redisClient.LPop(jq.ctx, queue).Result()
		if err == redis.Nil {
			continue // Queue empty, try next
		}
		if err != nil {
			log.Printf("Error popping from queue %s: %v", queue, err)
			continue
		}

		// Get job data from Redis
		jobData, err := jq.redisClient.Get(jq.ctx, JobKeyPrefix+result).Result()
		if err != nil {
			log.Printf("Error getting job %s: %v", result, err)
			continue
		}

		var job Job
		if err := json.Unmarshal([]byte(jobData), &job); err != nil {
			log.Printf("Error unmarshalling job %s: %v", result, err)
			continue
		}

		return &job
	}

	return nil
}

// =====================================================
// Job Creation and Enqueuing
// =====================================================

// NewJob creates a new job with a unique ID
func NewJob(jobType JobType, priority Priority, sourceCode string, languageID int,
	testCases []TestCaseData, problem ProblemData, userRegdNo, collegeID string, contestID *uint) *Job {

	return &Job{
		ID:         uuid.New().String(),
		Type:       jobType,
		Priority:   priority,
		SourceCode: sourceCode,
		LanguageID: languageID,
		TestCases:  testCases,
		Problem:    problem,
		UserRegdNo: userRegdNo,
		CollegeID:  collegeID,
		ContestID:  contestID,
		Status:     JobStatusPending,
		Progress:   0,
		RetryCount: 0,
		CreatedAt:  time.Now(),
	}
}

// EnqueueJob adds a job to the appropriate queue and stores job data
func EnqueueJob(job *Job) string {
	if jobQueue == nil || !jobQueue.initialized {
		log.Println("Job queue not initialized, cannot enqueue")
		return ""
	}

	// Store job data in Redis
	jobData, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error marshalling job: %v", err)
		return ""
	}

	// Set job with TTL (24 hours - jobs shouldn't live longer)
	jq := jobQueue
	jq.redisClient.Set(jq.ctx, JobKeyPrefix+job.ID, jobData, 24*time.Hour)

	// Push to appropriate queue based on priority
	var queueKey string
	switch job.Priority {
	case PriorityHigh:
		queueKey = QueueKeyHigh
	case PriorityNormal:
		queueKey = QueueKeyNormal
	default:
		queueKey = QueueKeyLow
	}

	jq.redisClient.RPush(jq.ctx, queueKey, job.ID)

	log.Printf("Job %s enqueued to %s (priority=%d)", job.ID, queueKey, job.Priority)
	return job.ID
}

// =====================================================
// Job Processing with Retry
// =====================================================

// processJob executes a job with retry logic
func (jq *JobQueueService) processJob(job *Job) {
	// Update job status to running
	now := time.Now()
	job.Status = JobStatusRunning
	job.StartedAt = &now
	jq.saveJob(job)

	maxRetries := config.AppConfig.JobQueueMaxRetries
	baseDelay := time.Duration(config.AppConfig.JobQueueRetryDelay) * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			job.RetryCount = attempt
			jq.saveJob(job)

			// Exponential backoff: 2s, 4s, 8s
			delay := baseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			log.Printf("Job %s retry %d, waiting %v", job.ID, attempt, delay)
			time.Sleep(delay)
		}

		// Execute test cases
		result := jq.executeTestCases(job)

		// Update progress
		job.Progress = 100

		if result.ErrorMessage == "" {
			// Success
			job.Status = JobStatusCompleted
			job.Result = result
			completedAt := time.Now()
			job.CompletedAt = &completedAt
			jq.saveJob(job)

			// Notify via WebSocket if available
			jq.notifyJobComplete(job)

			log.Printf("Job %s completed: passed=%d/%d, score=%d/%d",
				job.ID, result.PassedCount, result.TotalTests, result.Score, result.MaxScore)
			return
		}

		// Check if error is retryable
		if !jq.isRetryableError(result.ErrorMessage) {
			// Non-retryable error (compilation error, etc.)
			job.Status = JobStatusFailed
			job.ErrorMessage = result.ErrorMessage
			job.Result = result
			completedAt := time.Now()
			job.CompletedAt = &completedAt
			jq.saveJob(job)

			jq.notifyJobComplete(job)

			log.Printf("Job %s failed (non-retryable): %s", job.ID, result.ErrorMessage)
			return
		}

		// Retryable error - will retry
		log.Printf("Job %s failed (retryable): %s, will retry", job.ID, result.ErrorMessage)
	}

	// Max retries exceeded
	job.Status = JobStatusFailed
	job.ErrorMessage = "Max retries exceeded: " + job.Result.ErrorMessage
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	jq.saveJob(job)

	jq.notifyJobComplete(job)

	log.Printf("Job %s failed after %d retries", job.ID, maxRetries)
}

// executeTestCases runs code against all test cases
func (jq *JobQueueService) executeTestCases(job *Job) *JobResult {
	results := make([]TestResult, len(job.TestCases))
	var mu sync.Mutex
	var execErrors []error
	passedCount := 0
	totalTime := 0.0
	maxMemory := 0
	score := 0
	maxScore := 0

	// Execute test cases concurrently (limited by worker semaphore already held)
	var wg sync.WaitGroup

	for i, tc := range job.TestCases {
		wg.Add(1)
		go func(idx int, testCase TestCaseData) {
			defer wg.Done()

			// Ensure stdin is never empty
			stdin := testCase.Input
			if stdin == "" {
				stdin = "\n"
			}

			result, err := SubmitCode(
				job.SourceCode,
				job.LanguageID,
				stdin,
				strings.TrimSpace(testCase.ExpectedOutput),
				job.Problem.TimeLimit,
				job.Problem.MemoryLimit,
			)

			if err != nil {
				mu.Lock()
				execErrors = append(execErrors, err)
				mu.Unlock()
				return
			}

			// Assign points
			result.Points = testCase.Points
			if result.Passed {
				result.EarnedPoints = testCase.Points
			} else {
				result.EarnedPoints = 0
			}

			// Mark as hidden for non-sample test cases
			result.IsHidden = !testCase.IsSample
			if !testCase.IsSample {
				result.Input = ""
				result.ExpectedOutput = ""
			}

			mu.Lock()
			results[idx] = *result
			mu.Unlock()
		}(i, tc)
	}

	wg.Wait()

	// Check for execution errors
	if len(execErrors) > 0 {
		return &JobResult{
			ErrorMessage: fmt.Sprintf("Code execution failed: %v", execErrors[0]),
		}
	}

	// Calculate totals
	for _, result := range results {
		maxScore += result.Points
		if result.Passed {
			passedCount++
			score += result.EarnedPoints
		}
		totalTime += result.Time
		if result.Memory > maxMemory {
			maxMemory = result.Memory
		}
	}

	return &JobResult{
		TestResults: results,
		PassedCount: passedCount,
		TotalTests:  len(job.TestCases),
		AllPassed:   passedCount == len(job.TestCases),
		Score:       score,
		MaxScore:    maxScore,
		TotalTime:   totalTime,
		MaxMemory:   maxMemory,
	}
}

// isRetryableError checks if an error should trigger retry
func (jq *JobQueueService) isRetryableError(errMsg string) bool {
	// Retry on timeout, rate limit, and server errors
	retryable := []string{
		"timeout",
		"timed out",
		"rate limit",
		"too many requests",
		"service unavailable",
		"temporarily unavailable",
		"circuit breaker",
		"connection refused",
		"502",
		"503",
		"504",
	}

	errLower := strings.ToLower(errMsg)
	for _, r := range retryable {
		if strings.Contains(errLower, r) {
			return true
		}
	}

	return false
}

// =====================================================
// Job Status and Retrieval
// =====================================================

// GetJobStatus retrieves the current status of a job
func GetJobStatus(jobID string) (*JobStatusResponse, error) {
	if jobQueue == nil || !jobQueue.initialized {
		return nil, fmt.Errorf("Job queue not initialized")
	}

	jq := jobQueue
	jobData, err := jq.redisClient.Get(jq.ctx, JobKeyPrefix+jobID).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("Job not found")
	}
	if err != nil {
		return nil, err
	}

	var job Job
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, err
	}

	// Build response
	response := &JobStatusResponse{
		JobID:        job.ID,
		Status:       job.Status,
		Progress:     job.Progress,
		RetryCount:   job.RetryCount,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAt,
		StartedAt:    job.StartedAt,
		CompletedAt:  job.CompletedAt,
	}

	// Add human-readable message
	switch job.Status {
	case JobStatusPending:
		response.Message = "Waiting in queue..."
	case JobStatusRunning:
		response.Message = fmt.Sprintf("Running tests... (%d/%d)", job.Progress/100*len(job.TestCases), len(job.TestCases))
	case JobStatusCompleted:
		response.Message = "Completed successfully"
		if job.Result != nil {
			response.TestResults = job.Result.TestResults
			response.AllPassed = job.Result.AllPassed
			response.Score = job.Result.Score
			response.MaxScore = job.Result.MaxScore
		}
	case JobStatusFailed:
		response.Message = "Execution failed"
		if job.Result != nil {
			response.TestResults = job.Result.TestResults
		}
	}

	return response, nil
}

// GetQueueStats returns statistics about the job queue
func GetQueueStats() map[string]interface{} {
	if jobQueue == nil || !jobQueue.initialized {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	jq := jobQueue

	highLen, _ := jq.redisClient.LLen(jq.ctx, QueueKeyHigh).Result()
	normalLen, _ := jq.redisClient.LLen(jq.ctx, QueueKeyNormal).Result()
	lowLen, _ := jq.redisClient.LLen(jq.ctx, QueueKeyLow).Result()

	return map[string]interface{}{
		"enabled":        true,
		"queue_high":     highLen,
		"queue_normal":   normalLen,
		"queue_low":      lowLen,
		"queue_total":    highLen + normalLen + lowLen,
		"active_workers": atomic.LoadInt32(&jq.activeWorkers),
		"max_workers":    config.AppConfig.JobQueueWorkers,
	}
}

// =====================================================
// Helper Methods
// =====================================================

// saveJob updates job data in Redis
func (jq *JobQueueService) saveJob(job *Job) {
	jobData, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error marshalling job: %v", err)
		return
	}

	jq.redisClient.Set(jq.ctx, JobKeyPrefix+job.ID, jobData, 24*time.Hour)
}

// notifyJobComplete sends WebSocket notification for job completion
func (jq *JobQueueService) notifyJobComplete(job *Job) {
	// Get WebSocket hub
	hub := GetWebSocketHub()
	if hub == nil {
		return
	}

	// Send notification to user
	status, _ := GetJobStatus(job.ID)
	if status != nil {
		_ = hub.SendToUser(job.UserRegdNo, WebSocketMessage{
			Type: "JOB_COMPLETE",
			Data: status,
		})
	}
}

// ConvertModelsTestCase converts models.TestCase to TestCaseData
func ConvertModelsTestCase(tc models.TestCase) TestCaseData {
	return TestCaseData{
		Input:          tc.Input,
		ExpectedOutput: tc.ExpectedOutput,
		TimeLimitMs:    0, // Will use problem's time limit
		MemoryLimitKB:  0, // Will use problem's memory limit
		Points:         tc.Points,
		IsSample:       tc.IsSample,
	}
}

// ConvertModelsProblem converts models.Problem to ProblemData
func ConvertModelsProblem(p models.Problem) ProblemData {
	return ProblemData{
		ID:          p.ID,
		TimeLimit:   p.TimeLimit,
		MemoryLimit: p.MemoryLimit,
	}
}
