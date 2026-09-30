package handlers

import (
	"coding-platform/config"
	"coding-platform/database"
	"coding-platform/models"
	"coding-platform/services"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetPracticeProblems returns all global problems for student practice
// These are problems created by Super Admin that are accessible to all students
func GetPracticeProblems(c *gin.Context) {
	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get user info for completion status
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Get all global problems with sample test cases and subject
	var problems []models.Problem
	if err := database.DB.Where("is_global = ?", true).
		Preload("TestCases", "is_sample = ?", true).
		Preload("Subject").
		Order("created_at DESC").
		Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch practice problems"})
		return
	}

	// Get completion status for each problem
	type ProblemWithStatus struct {
		models.Problem
		IsCompleted bool `json:"is_completed"`
	}

	var problemsWithStatus []ProblemWithStatus
	for _, p := range problems {
		var completion models.UserProblemCompletion
		isCompleted := false
		if err := database.DB.Where("user_regd_no = ? AND problem_id = ?", userRegdNo, p.ID).First(&completion).Error; err == nil {
			isCompleted = true
		}
		problemsWithStatus = append(problemsWithStatus, ProblemWithStatus{
			Problem:     p,
			IsCompleted: isCompleted,
		})
	}

	// Get all subjects for filtering
	var subjects []models.Subject
	database.DB.Order("name ASC").Find(&subjects)

	c.JSON(http.StatusOK, gin.H{
		"problems": problemsWithStatus,
		"subjects": subjects,
	})
}

// GetPracticeProblem returns a single global problem for practice
func GetPracticeProblem(c *gin.Context) {
	id := c.Param("id")

	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get the problem - must be global
	var problem models.Problem
	err := database.DB.Where("id = ? AND is_global = ?", id, true).
		Preload("TestCases", "is_sample = ?", true).
		First(&problem).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Practice problem not found"})
		return
	}

	// Check completion status
	var completion models.UserProblemCompletion
	isCompleted := false
	if err := database.DB.Where("user_regd_no = ? AND problem_id = ?", userRegdNo, problem.ID).First(&completion).Error; err == nil {
		isCompleted = true
	}

	c.JSON(http.StatusOK, gin.H{
		"problem":      problem,
		"is_completed": isCompleted,
	})
}

// SubmitPracticeSolution handles code submission for practice problems
// This is similar to regular submission but allows global problems without college restriction
func SubmitPracticeSolution(c *gin.Context) {
	var req SubmitCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate code size
	if len(req.SourceCode) > MaxCodeSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Code size exceeds limit",
		})
		return
	}

	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get user
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Get the problem - must be global
	var problem models.Problem
	err := database.DB.Where("id = ? AND is_global = ?", req.ProblemID, true).First(&problem).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Practice problem not found"})
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

		// Create job with NORMAL priority (practice submission)
		job := services.NewJob(
			services.JobTypeSubmit,
			services.PriorityNormal,
			req.SourceCode,
			req.LanguageID,
			queueTestCases,
			services.ConvertModelsProblem(problem),
			userRegdNo.(string),
			"", // Practice problems are global, no college restriction
			nil,
		)

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
		CollegeID:     user.CollegeID, // Store user's college for filtering
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
		SubmittedAt:   GetCurrentTime(),
	}

	// Save submission
	if err := database.DB.Create(&submission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save submission"})
		return
	}

	// Only update streak and activity when all tests pass.
	// Streak rules (practice problems only):
	//  - Only the FIRST passing solve of the day advances the streak.
	//  - Re-solving an already-completed problem never advances the streak again.
	//  - Failed submissions are completely ignored for streak purposes.
	log.Printf("[SubmitPracticeSolution] allPassed=%v for user=%s, problem=%d", allPassed, userRegdNo, req.ProblemID)

	if allPassed {
		var completion models.UserProblemCompletion
		result := database.DB.Where("user_regd_no = ? AND problem_id = ?", userRegdNo, req.ProblemID).First(&completion)

		// isFirstSolve is true when no prior completion record exists
		isFirstSolve := result.Error != nil

		log.Printf("[SubmitPracticeSolution] isFirstSolve=%v for user=%s, problem=%d (err=%v)", isFirstSolve, userRegdNo, req.ProblemID, result.Error)

		if isFirstSolve {
			// Create completion record for this newly solved problem
			completion = models.UserProblemCompletion{
				UserRegdNo:        userRegdNo.(string),
				ProblemID:         req.ProblemID,
				CollegeID:         user.CollegeID,
				CompletedAt:       GetCurrentTime(),
				FirstSubmissionID: submission.ID,
				TimeTakenSeconds:  req.TimeTaken,
			}
			if err := database.DB.Create(&completion).Error; err != nil {
				log.Printf("[SubmitPracticeSolution] ERROR creating completion: %v", err)
			}

			// Update activity heatmap and streak ONLY on first-ever solve of this problem.
			// The streak SQL handles same-day deduplication atomically, so calling this
			// once per new problem solved is the correct trigger point.
			log.Printf("[SubmitPracticeSolution] Calling UpdateUserActivity and UpdateUserStreak for user=%s", userRegdNo)
			UpdateUserActivity(userRegdNo.(string))
			UpdateUserStreak(userRegdNo.(string))
			log.Printf("[SubmitPracticeSolution] Finished updating streak for user=%s", userRegdNo)
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

// RunPracticeCode runs code against sample test cases for practice problems
func RunPracticeCode(c *gin.Context) {
	var req struct {
		SourceCode string `json:"source_code" binding:"required"`
		ProblemID  uint   `json:"problem_id" binding:"required"`
		LanguageID int    `json:"language_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate code size
	if len(req.SourceCode) > MaxCodeSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Code size exceeds limit",
		})
		return
	}

	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Get user for college ID
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Get the problem - must be global
	var problem models.Problem
	err := database.DB.Where("id = ? AND is_global = ?", req.ProblemID, true).First(&problem).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Practice problem not found"})
		return
	}

	// Get ONLY sample test cases
	var testCases []models.TestCase
	if err := database.DB.Where("problem_id = ? AND is_sample = ?", req.ProblemID, true).Find(&testCases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch test cases"})
		return
	}

	if len(testCases) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No sample test cases found"})
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
				IsSample:       true,
			}
		}

		job := services.NewJob(
			services.JobTypeRun,
			services.PriorityLow,
			req.SourceCode,
			req.LanguageID,
			queueTestCases,
			services.ConvertModelsProblem(problem),
			userRegdNo.(string),
			"",
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

	// Fallback: Direct execution
	execResult := ExecuteCodeAgainstTestCases(req.SourceCode, req.LanguageID, problem, testCases)

	if execResult.ErrorMessage != "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to execute code. Please try again later.",
		})
		return
	}

	response := SubmissionResponse{
		SubmissionID: 0,
		AllPassed:    execResult.AllPassed,
		TotalTests:   execResult.TotalTests,
		PassedTests:  execResult.PassedCount,
		Score:        execResult.Score,
		MaxScore:     execResult.MaxScore,
		TestResults:  execResult.Results,
	}

	c.JSON(http.StatusOK, response)
}

// GetPracticeProblemSubmissions returns submissions for a practice problem
func GetPracticeProblemSubmissions(c *gin.Context) {
	problemID := c.Param("id")

	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Verify problem is global
	var problem models.Problem
	if err := database.DB.Where("id = ? AND is_global = ?", problemID, true).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Practice problem not found"})
		return
	}

	page := 1
	limit := 20

	var submissions []models.Submission
	if err := database.DB.Where("user_regd_no = ? AND problem_id = ?", userRegdNo, problemID).
		Order("submitted_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"submissions": submissions})
}

// GetCurrentTime returns current time - extracted for testability
func GetCurrentTime() time.Time {
	return time.Now()
}
