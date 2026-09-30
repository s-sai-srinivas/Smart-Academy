package handlers

import (
	"coding-platform/config"
	"coding-platform/database"
	"coding-platform/models"
	"coding-platform/services"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// =====================================================
// Code Execution Constants
// =====================================================

// MaxCodeSize is the maximum allowed code size (100KB)
const MaxCodeSize = 100 * 1024

// =====================================================
// Request/Response Types
// =====================================================

type SubmitCodeRequest struct {
	SourceCode string  `json:"source_code" binding:"required"`
	ProblemID  uint    `json:"problem_id" binding:"required"`
	LanguageID int     `json:"language_id" binding:"required"`
	TimeTaken  float64 `json:"time_taken" binding:"omitempty,min=0,max=86400"`
}

type SubmissionResponse struct {
	Runtime      *RuntimeStats         `json:"runtime,omitempty"`
	Memory       *MemoryStats          `json:"memory,omitempty"`
	TestResults  []services.TestResult `json:"test_results"`
	SubmissionID uint                  `json:"submission_id"`
	TotalTests   int                   `json:"total_tests"`
	PassedTests  int                   `json:"passed_tests"`
	Score        int                   `json:"score"`
	MaxScore     int                   `json:"max_score"`
	AllPassed    bool                  `json:"all_passed"`
}

type RuntimeStats struct {
	Display    string  `json:"display"`
	FasterThan string  `json:"faster_than"`
	ValueMs    int     `json:"value_ms"`
	Percentile float64 `json:"percentile"`
}

type MemoryStats struct {
	Display    string  `json:"display"`
	LowerThan  string  `json:"lower_than"`
	ValueKB    int     `json:"value_kb"`
	Percentile float64 `json:"percentile"`
}

func SubmitCode(c *gin.Context) {
	var req SubmitCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate code size to prevent memory issues
	if len(req.SourceCode) > MaxCodeSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Code size exceeds limit. Maximum allowed: %d KB", MaxCodeSize/1024),
		})
		return
	}

	// Require authentication for code submissions
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required to submit code"})
		return
	}

	// Get user's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
		return
	}

	// Get problem details WITH college validation
	var problem models.Problem
	err := database.DB.Where("id = ? AND college_id = ?", req.ProblemID, *user.CollegeID).First(&problem).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Get all test cases for this problem
	var testCases []models.TestCase
	if err := database.DB.Where("problem_id = ?", req.ProblemID).Find(&testCases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch test cases"})
		return
	}

	if len(testCases) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No test cases found for this problem"})
		return
	}

	// Check if job queue is enabled - use queue for burst handling
	if config.AppConfig.JobQueueEnabled && config.AppConfig.RedisEnabled {
		// Convert test cases to queue format
		queueTestCases := make([]services.TestCaseData, len(testCases))
		for i, tc := range testCases {
			queueTestCases[i] = services.TestCaseData{
				Input:          tc.Input,
				ExpectedOutput: tc.ExpectedOutput,
				TimeLimitMs:    problem.TimeLimit,
				MemoryLimitKB:  problem.MemoryLimit,
				Points:         tc.Points,
				IsSample:       tc.IsSample,
			}
		}

		// Create job with NORMAL priority (lab/course submission)
		job := services.NewJob(
			services.JobTypeSubmit,
			services.PriorityNormal,
			req.SourceCode,
			req.LanguageID,
			queueTestCases,
			services.ConvertModelsProblem(problem),
			userRegdNo.(string),
			*user.CollegeID,
			nil, // Not a contest submission
		)

		// Enqueue job
		jobID := services.EnqueueJob(job)

		c.JSON(http.StatusAccepted, gin.H{
			"job_id":   jobID,
			"status":   "pending",
			"message":  "Submission queued for processing",
			"poll_url": fmt.Sprintf("/api/jobs/%s", jobID),
		})
		return
	}

	// Fallback: Direct execution when job queue is disabled
	// Run code against all test cases concurrently for better performance
	execResult := ExecuteCodeAgainstTestCases(req.SourceCode, req.LanguageID, problem, testCases)

	if execResult.ErrorMessage != "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to execute code. Please try again later.",
		})
		return
	}

	allPassed := execResult.AllPassed

	// Create submission record
	submission := models.Submission{
		UserRegdNo:    userRegdNo.(string),
		ProblemID:     req.ProblemID,
		CollegeID:     user.CollegeID, // Direct college reference for efficient filtering
		LanguageID:    req.LanguageID,
		SourceCode:    req.SourceCode,
		Status:        "completed",
		Passed:        allPassed,
		TotalTests:    execResult.TotalTests,
		PassedTests:   execResult.PassedCount,
		Score:         execResult.Score,
		MaxScore:      execResult.MaxScore,
		ExecutionTime: execResult.TotalTime,
		MemoryUsed:    execResult.MaxMemory,
		TimeSpent:     req.TimeTaken,
		SubmittedAt:   time.Now(),
	}

	// Save submission
	if err := database.DB.Create(&submission).Error; err != nil {
		log.Printf("Failed to save submission: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save submission"})
		return
	}

	// NOTE: Streak and activity tracking are intentionally NOT updated here.
	// This handler serves lab/course submissions. Streak only applies to
	// practice problems and is updated in practice.go (SubmitPracticeSolution).

	// If all tests passed, mark problem as completed (if not already completed)
	if allPassed {
		var completion models.UserProblemCompletion
		result := database.DB.Where("user_regd_no = ? AND problem_id = ?", userRegdNo, req.ProblemID).First(&completion)

		// Only create completion record if it doesn't exist
		if result.Error != nil {
			completion = models.UserProblemCompletion{
				UserRegdNo:        userRegdNo.(string),
				ProblemID:         req.ProblemID,
				CollegeID:         user.CollegeID,
				CompletedAt:       time.Now(),
				FirstSubmissionID: submission.ID,
				TimeTakenSeconds:  req.TimeTaken,
			}
			if err := database.DB.Create(&completion).Error; err != nil {
				log.Printf("Failed to create completion record: %v", err)
				// Don't fail the request, just log the error
			}
		}
	}

	response := SubmissionResponse{
		SubmissionID: submission.ID,
		AllPassed:    allPassed,
		TotalTests:   execResult.TotalTests,
		PassedTests:  execResult.PassedCount,
		Score:        execResult.Score,
		MaxScore:     execResult.MaxScore,
		TestResults:  execResult.Results,
	}

	// Add runtime and memory stats for accepted submissions
	if allPassed && execResult.TotalTime > 0 {
		response.Runtime = getRuntimeStats(req.ProblemID, execResult.TotalTime)
	}
	if allPassed && execResult.MaxMemory > 0 {
		response.Memory = getMemoryStats(req.ProblemID, execResult.MaxMemory)
	}

	c.JSON(http.StatusOK, response)
}

// RunCode validates code against sample test cases only
// Requires authentication - users must be logged in to run code
func RunCode(c *gin.Context) {
	var req struct {
		SourceCode string `json:"source_code" binding:"required"`
		ProblemID  uint   `json:"problem_id" binding:"required"`
		LanguageID int    `json:"language_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate code size to prevent memory issues
	if len(req.SourceCode) > MaxCodeSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Code size exceeds limit. Maximum allowed: %d KB", MaxCodeSize/1024),
		})
		return
	}

	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required to run code"})
		return
	}

	// Get user's college for validation
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Get problem details WITH college validation
	var problem models.Problem
	query := database.DB.Where("id = ?", req.ProblemID)
	if user.CollegeID != nil {
		query = query.Where("college_id = ?", *user.CollegeID)
	}
	if err := query.First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Get ONLY sample test cases for this problem
	var testCases []models.TestCase
	if err := database.DB.Where("problem_id = ? AND is_sample = ?", req.ProblemID, true).Find(&testCases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch test cases"})
		return
	}

	if len(testCases) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No sample test cases found for this problem"})
		return
	}

	// Check if job queue is enabled - use queue with LOW priority for run
	if config.AppConfig.JobQueueEnabled && config.AppConfig.RedisEnabled {
		// Convert test cases to queue format
		queueTestCases := make([]services.TestCaseData, len(testCases))
		for i, tc := range testCases {
			queueTestCases[i] = services.TestCaseData{
				Input:          tc.Input,
				ExpectedOutput: tc.ExpectedOutput,
				TimeLimitMs:    problem.TimeLimit,
				MemoryLimitKB:  problem.MemoryLimit,
				Points:         tc.Points,
				IsSample:       true, // All are sample for run
			}
		}

		// Create job with LOW priority (run code is less important than submit)
		job := services.NewJob(
			services.JobTypeRun,
			services.PriorityLow,
			req.SourceCode,
			req.LanguageID,
			queueTestCases,
			services.ConvertModelsProblem(problem),
			userRegdNo.(string),
			*user.CollegeID,
			nil,
		)

		jobID := services.EnqueueJob(job)

		c.JSON(http.StatusAccepted, gin.H{
			"job_id":   jobID,
			"status":   "pending",
			"message":  "Run request queued for processing",
			"poll_url": fmt.Sprintf("/api/jobs/%s", jobID),
		})
		return
	}

	// Fallback: Direct execution when job queue is disabled
	execResult := ExecuteCodeAgainstTestCases(req.SourceCode, req.LanguageID, problem, testCases)

	if execResult.ErrorMessage != "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to execute code. Please try again later.",
		})
		return
	}

	response := SubmissionResponse{
		SubmissionID: 0, // No submission record for run code
		AllPassed:    execResult.AllPassed,
		TotalTests:   execResult.TotalTests,
		PassedTests:  execResult.PassedCount,
		Score:        execResult.Score,
		MaxScore:     execResult.MaxScore,
		TestResults:  execResult.Results,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateUserActivity updates the user's daily activity record
func UpdateUserActivity(userRegdNo string) {
	today := time.Now().Format("2006-01-02")
	todayTime, _ := time.Parse("2006-01-02", today)

	var activity models.UserActivity
	err := database.DB.Where("user_regd_no = ? AND date = ?", userRegdNo, todayTime).First(&activity).Error

	if err != nil {
		// Create new activity record
		activity = models.UserActivity{
			UserRegdNo: userRegdNo,
			Date:       todayTime,
			Count:      1,
		}
		database.DB.Create(&activity)
	} else {
		// Increment existing activity count
		database.DB.Model(&activity).Update("count", activity.Count+1)
	}
}

// UpdateUserStreak updates the user's submission streak
// Uses GORM's OnConflict clause for atomic upsert
// Streak only increments on consecutive days, not multiple submissions on same day
func UpdateUserStreak(userRegdNo string) {
	today := time.Now().Truncate(24 * time.Hour)

	log.Printf("[UpdateUserStreak] Called for user: %s, today: %v", userRegdNo, today)

	// First, try to find existing streak
	var streak models.UserStreak
	err := database.DB.Where("user_regd_no = ?", userRegdNo).First(&streak).Error

	if err != nil {
		log.Printf("[UpdateUserStreak] No existing streak found for %s, creating new: %v", userRegdNo, err)
		// No existing streak - create new one
		newStreak := models.UserStreak{
			UserRegdNo:       userRegdNo,
			CurrentStreak:    1,
			LongestStreak:    1,
			LastActivityDate: today,
		}
		if err := database.DB.Create(&newStreak).Error; err != nil {
			log.Printf("[UpdateUserStreak] ERROR creating streak for %s: %v", userRegdNo, err)
		} else {
			log.Printf("[UpdateUserStreak] Successfully created streak for %s: current=1, longest=1", userRegdNo)
		}
		return
	}

	log.Printf("[UpdateUserStreak] Found existing streak for %s: current=%d, longest=%d, lastActivity=%v",
		userRegdNo, streak.CurrentStreak, streak.LongestStreak, streak.LastActivityDate)

	// Existing streak found - calculate new values
	lastActivity := streak.LastActivityDate.Truncate(24 * time.Hour)
	daysDiff := int(today.Sub(lastActivity).Hours() / 24)

	log.Printf("[UpdateUserStreak] Days difference for %s: %d (today=%v, last=%v)", userRegdNo, daysDiff, today, lastActivity)

	var newCurrentStreak int
	var newLongestStreak int

	switch daysDiff {
	case 0:
		// Same day - keep current streak
		newCurrentStreak = streak.CurrentStreak
		newLongestStreak = streak.LongestStreak
		log.Printf("[UpdateUserStreak] Same day for %s - keeping streak at %d", userRegdNo, newCurrentStreak)
	case 1:
		// Yesterday - increment streak
		newCurrentStreak = streak.CurrentStreak + 1
		newLongestStreak = streak.LongestStreak
		if newCurrentStreak > newLongestStreak {
			newLongestStreak = newCurrentStreak
		}
		log.Printf("[UpdateUserStreak] Yesterday for %s - incrementing to %d", userRegdNo, newCurrentStreak)
	default:
		// More than 1 day - reset streak
		newCurrentStreak = 1
		if streak.LongestStreak > 1 {
			newLongestStreak = streak.LongestStreak
		} else {
			newLongestStreak = 1
		}
		log.Printf("[UpdateUserStreak] Streak broken for %s - resetting to 1", userRegdNo)
	}

	// Update the streak record
	if err := database.DB.Model(&streak).Updates(map[string]interface{}{
		"current_streak":     newCurrentStreak,
		"longest_streak":     newLongestStreak,
		"last_activity_date": today,
	}).Error; err != nil {
		log.Printf("[UpdateUserStreak] ERROR updating streak for %s: %v", userRegdNo, err)
	} else {
		log.Printf("[UpdateUserStreak] Successfully updated streak for %s: current=%d, longest=%d", userRegdNo, newCurrentStreak, newLongestStreak)
	}
}

// calculateRuntimePercentile calculates the percentile rank of a runtime
// among all accepted submissions for the same problem
// Returns a value from 0 to 100, where higher is better (faster)
func calculateRuntimePercentile(problemID uint, runtime float64) float64 {
	var result struct {
		CountFaster int64
		TotalCount  int64
	}

	// Single query to get both counts
	database.DB.Model(&models.Submission{}).
		Select(`
			COUNT(*) as total_count,
			SUM(CASE WHEN execution_time < ? THEN 1 ELSE 0 END) as count_faster
		`, runtime).
		Where("problem_id = ? AND passed = ?", problemID, true).
		Scan(&result)

	if result.TotalCount == 0 {
		return 0
	}

	// Percentile = (number of submissions with higher runtime / total) * 100
	// Higher percentile = better (faster)
	percentile := (float64(result.CountFaster) / float64(result.TotalCount)) * 100
	return percentile
}

// calculateMemoryPercentile calculates the percentile rank of memory usage
// among all accepted submissions for the same problem
// Returns a value from 0 to 100, where higher is better (less memory)
func calculateMemoryPercentile(problemID uint, memory int) float64 {
	var result struct {
		CountLower int64
		TotalCount int64
	}

	// Single query to get both counts
	database.DB.Model(&models.Submission{}).
		Select(`
			COUNT(*) as total_count,
			SUM(CASE WHEN memory_used < ? THEN 1 ELSE 0 END) as count_lower
		`, memory).
		Where("problem_id = ? AND passed = ?", problemID, true).
		Scan(&result)

	if result.TotalCount == 0 {
		return 0
	}

	// Percentile = (number of submissions using more memory / total) * 100
	percentile := (float64(result.CountLower) / float64(result.TotalCount)) * 100
	return percentile
}

// getRuntimeStats creates runtime stats with display information
func getRuntimeStats(problemID uint, runtimeSeconds float64) *RuntimeStats {
	percentile := calculateRuntimePercentile(problemID, runtimeSeconds)

	// Convert seconds to ms for display
	runtimeMs := runtimeSeconds * 1000
	display := fmt.Sprintf("%.0f ms", runtimeMs)
	fasterThan := ""
	if percentile > 0 {
		fasterThan = fmt.Sprintf("Faster than %.0f%% of submissions", percentile)
	}

	return &RuntimeStats{
		ValueMs:    int(runtimeMs),
		Display:    display,
		Percentile: percentile,
		FasterThan: fasterThan,
	}
}

// getMemoryStats creates memory stats with display information
func getMemoryStats(problemID uint, memoryKB int) *MemoryStats {
	percentile := calculateMemoryPercentile(problemID, memoryKB)

	display := ""
	if memoryKB < 1024 {
		display = fmt.Sprintf("%d KB", memoryKB)
	} else {
		display = fmt.Sprintf("%.1f MB", float64(memoryKB)/1024)
	}

	lowerThan := ""
	if percentile > 0 {
		lowerThan = fmt.Sprintf("Lower than %.0f%% of submissions", percentile)
	}

	return &MemoryStats{
		ValueKB:    memoryKB,
		Display:    display,
		Percentile: percentile,
		LowerThan:  lowerThan,
	}
}

// =====================================================
// Shared Submission Executor
// =====================================================

// ExecutionResult contains the results of running code against test cases
type ExecutionResult struct {
	ErrorMessage string
	Results      []services.TestResult
	PassedCount  int
	TotalTests   int
	Score        int
	MaxScore     int
	TotalTime    float64
	MaxMemory    int
	AllPassed    bool
}

// ExecuteCodeAgainstTestCases runs code against all test cases for a problem
// Test cases are executed concurrently for better performance
func ExecuteCodeAgainstTestCases(sourceCode string, languageID int, problem models.Problem, testCases []models.TestCase) ExecutionResult {
	if len(testCases) == 0 {
		return ExecutionResult{
			TotalTests: 0,
		}
	}

	// Execute test cases concurrently
	results := make([]services.TestResult, len(testCases))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var executionErrors []error

	for i, testCase := range testCases {
		wg.Add(1)
		go func(idx int, tc models.TestCase) {
			defer wg.Done()

			result, err := services.SubmitCode(
				sourceCode,
				languageID,
				tc.Input,
				strings.TrimSpace(tc.ExpectedOutput),
				problem.TimeLimit,
				problem.MemoryLimit,
			)

			if err != nil {
				mu.Lock()
				executionErrors = append(executionErrors, err)
				mu.Unlock()
				return
			}

			// Assign points from the test case
			result.Points = tc.Points
			if result.Passed {
				result.EarnedPoints = tc.Points
			} else {
				result.EarnedPoints = 0
			}

			// Mark as hidden and hide sensitive details for hidden (non-sample) test cases
			result.IsHidden = !tc.IsSample
			if !tc.IsSample {
				result.Input = ""
				result.ExpectedOutput = ""
				// Keep stdout visible so user can see their output even for hidden tests
			}

			mu.Lock()
			results[idx] = *result
			mu.Unlock()
		}(i, testCase)
	}

	wg.Wait()

	// Check for execution errors
	if len(executionErrors) > 0 {
		log.Printf("[ExecuteCodeAgainstTestCases] %d test case(s) failed to execute: %v", len(executionErrors), executionErrors[0])
		return ExecutionResult{
			ErrorMessage: "Code execution failed",
		}
	}

	// Process results
	passedCount := 0
	var totalTime float64
	var maxMemory int
	score := 0
	maxScore := 0

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

	return ExecutionResult{
		Results:     results,
		PassedCount: passedCount,
		TotalTests:  len(testCases),
		AllPassed:   passedCount == len(testCases),
		Score:       score,
		MaxScore:    maxScore,
		TotalTime:   totalTime,
		MaxMemory:   maxMemory,
	}
}
