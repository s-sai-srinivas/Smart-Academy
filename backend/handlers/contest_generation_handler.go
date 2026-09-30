package handlers

import (
	"coding-platform/config"
	"coding-platform/database"
	"coding-platform/middleware"
	"coding-platform/models"
	"coding-platform/services"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var contestGenerationService *services.ContestGenerationService
var generationJobService *services.GenerationJobService
var agentOrchestrator *services.AgentOrchestrator

// initContestGenerationService initializes the contest generation service
func initContestGenerationService() {
	if contestGenerationService == nil {
		aiProvider := services.NewGeminiProvider(
			config.AppConfig.AIAPIKey,
			config.AppConfig.AIModel,
			config.AppConfig.AIMaxTokens,
			config.AppConfig.AITemperature,
			config.AppConfig.AIRateLimitRPM,
		)
		// Initialize orchestrator first (needed by contest service)
		agentOrchestrator = services.NewAgentOrchestrator(aiProvider)
		// Initialize generation job service
		generationJobService = services.NewGenerationJobService(aiProvider)
		// Initialize contest generation service with orchestrator
		contestGenerationService = services.NewContestGenerationService(aiProvider, agentOrchestrator)
	}
}

// =====================================================
// Request/Response Types
// =====================================================

type GenerateContestRequest struct {
	CourseID        *uint    `json:"course_id,omitempty"`
	ContestID       *uint    `json:"contest_id,omitempty"`
	DifficultyLevel string   `json:"difficulty_level"`
	StartTime       string   `json:"start_time,omitempty"`
	EndTime         string   `json:"end_time,omitempty"`
	GenerationType  string   `json:"generation_type"`
	Topics          []string `json:"topics" binding:"required,min=1"`
	RequestedCount  int      `json:"requested_count" binding:"required,min=1,max=10"`
}

type GeneratedProblemsResponse struct {
	CreatedAt *time.Time                      `json:"created_at,omitempty"`
	Status    string                          `json:"status"`
	Problems  []models.GeneratedProblemReview `json:"problems"`
	JobID     uint                            `json:"job_id"`
}

type ApproveProblemsRequest struct {
	ContestID              *uint  `json:"contest_id,omitempty"`
	CreateContest          *bool  `json:"create_contest,omitempty"`
	ContestTitle           string `json:"contest_title,omitempty"`
	ContestDescription     string `json:"contest_description,omitempty"`
	StartTime              string `json:"start_time,omitempty"`
	EndTime                string `json:"end_time,omitempty"`
	ApprovedProblemIndices []int  `json:"approved_problem_indices"`
}

// =====================================================
// Handlers
// =====================================================

// GenerateContestProblems starts async generation of contest problems
func GenerateContestProblems(c *gin.Context) {
	initContestGenerationService()

	// Get admin info
	userRegdNo, _ := c.Get("regdno")
	adminRegdNo := userRegdNo.(string)

	var req GenerateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get college ID from admin user (needed for contest scoping)
	var collegeID *string
	if !middleware.IsSuperAdmin(c) {
		cid, ok := middleware.GetCurrentUserCollege(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "College access required"})
			return
		}
		collegeID = cid
	}

	// Optionally validate course if provided
	if req.CourseID != nil {
		var course models.Course
		if err := database.DB.First(&course, *req.CourseID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify course"})
			}
			return
		}
		if !middleware.IsSuperAdmin(c) && collegeID != nil && course.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied - course not in your college"})
			return
		}
		if collegeID == nil {
			collegeID = &course.CollegeID
		}
	}

	if collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not determined. Provide a course_id or ensure your account has a college."})
		return
	}

	// Store college ID in job data
	_ = collegeID // used later in approval flow

	// Create generation job
	job, err := generationJobService.CreateJob(
		adminRegdNo,
		req.CourseID,
		nil, // theory_id
		models.GenerationJobType("contest_generate"),
		"", // file path not needed for contest generation
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create generation job"})
		log.Printf("Failed to create generation job: %v", err)
		return
	}

	// Store request data in job for processing
	jobData := map[string]interface{}{
		"topics":           req.Topics,
		"requested_count":  req.RequestedCount,
		"difficulty_level": req.DifficultyLevel,
		"start_time":       req.StartTime,
		"end_time":         req.EndTime,
		"generation_type":  req.GenerationType,
	}
	if collegeID != nil {
		jobData["college_id"] = *collegeID
	}
	jobDataJSON, _ := json.Marshal(jobData)
	job.GeneratedData = jobDataJSON
	database.DB.Save(job)

	// Start async processing
	go func(jobID uint) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		if err := generationJobService.ProcessJob(ctx, jobID); err != nil {
			log.Printf("Contest generation job %d failed: %v", jobID, err)
		}
	}(job.ID)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":         job.ID,
		"status":         job.Status,
		"message":        "Contest problem generation started",
		"estimated_time": "3-5 minutes",
	})
}

// GetContestGenerationJobStatus returns the status of a contest generation job
func GetContestGenerationJobStatus(c *gin.Context) {
	initContestGenerationService()

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := generationJobService.GetJob(uint(jobID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job status"})
		return
	}

	// Verify access: user must be the job creator or admin of the college
	userRegdNo, _ := c.Get("regdno")
	if !verifyJobAccess(c, job, userRegdNo.(string)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job_id":        job.ID,
		"status":        job.Status,
		"job_type":      job.JobType,
		"created_at":    job.CreatedAt,
		"completed_at":  job.CompletedAt,
		"error_message": job.ErrorMessage,
	})
}

// GetGeneratedProblems returns generated problems for faculty review
func GetGeneratedProblems(c *gin.Context) {
	initContestGenerationService()

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := generationJobService.GetJob(uint(jobID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	if job.Status != models.GenerationJobCompleted {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Job not completed yet. Status: %s", job.Status),
		})
		return
	}

	// Parse generated problems from job data
	var problems []models.GeneratedProblem
	if err := json.Unmarshal(job.GeneratedData, &problems); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse generated problems"})
		log.Printf("Failed to parse generated problems: %v", err)
		return
	}

	// Convert to review format
	reviewProblems := make([]models.GeneratedProblemReview, len(problems))
	for i, p := range problems {
		reviewProblems[i] = models.GeneratedProblemReview{
			ProblemID:          uint(i),
			Title:              p.Title,
			Difficulty:         p.Difficulty,
			TopicsCovered:      p.TopicsCovered,
			VerificationStatus: p.VerificationStatus,
			TestCasesCount:     len(p.TestCases),
			IsApproved:         false,
			IsRejected:         false,
		}
	}

	c.JSON(http.StatusOK, GeneratedProblemsResponse{
		JobID:     job.ID,
		Status:    string(job.Status),
		Problems:  reviewProblems,
		CreatedAt: &job.CreatedAt,
	})
}

// GetGeneratedProblemDetail returns full details of a specific generated problem
func GetGeneratedProblemDetail(c *gin.Context) {
	initContestGenerationService()

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	problemIndex, err := strconv.ParseUint(c.Param("index"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem index"})
		return
	}

	job, err := generationJobService.GetJob(uint(jobID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	if job.Status != models.GenerationJobCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job not completed yet"})
		return
	}

	// Parse generated problems
	var problems []models.GeneratedProblem
	if err := json.Unmarshal(job.GeneratedData, &problems); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse generated problems"})
		return
	}

	if int(problemIndex) >= len(problems) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	c.JSON(http.StatusOK, problems[problemIndex])
}

// ApproveGeneratedProblems saves approved problems to database
func ApproveGeneratedProblems(c *gin.Context) {
	initContestGenerationService()

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var req ApproveProblemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.ApprovedProblemIndices) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one problem must be approved"})
		return
	}

	job, err := generationJobService.GetJob(uint(jobID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	// Verify access: user must be the job creator or admin of the college
	userRegdNo, _ := c.Get("regdno")
	if !verifyJobAccess(c, job, userRegdNo.(string)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if job.Status != models.GenerationJobCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job not completed yet"})
		return
	}

	// Parse generated problems
	var problems []models.GeneratedProblem
	if err := json.Unmarshal(job.GeneratedData, &problems); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse generated problems"})
		return
	}

	// Get admin info
	adminRegdNo := userRegdNo.(string)

	// Get college ID - first try from job data, then from course, then from user
	var collegeIDStr string
	if job.CourseID != nil {
		var course models.Course
		if err := database.DB.First(&course, *job.CourseID).Error; err == nil {
			collegeIDStr = course.CollegeID
		}
	}
	if collegeIDStr == "" {
		// Try getting from admin user's college
		var adminUser models.User
		if err := database.DB.Where("regdno = ?", adminRegdNo).First(&adminUser).Error; err == nil {
			if adminUser.CollegeID != nil {
				collegeIDStr = *adminUser.CollegeID
			}
		}
	}
	if collegeIDStr == "" && !middleware.IsSuperAdmin(c) {
		cid, ok := middleware.GetCurrentUserCollege(c)
		if ok && cid != nil {
			collegeIDStr = *cid
		}
	}

	// Filter approved problems
	var approvedProblems []models.GeneratedProblem
	for _, idx := range req.ApprovedProblemIndices {
		if idx >= 0 && idx < len(problems) {
			approvedProblems = append(approvedProblems, problems[idx])
		}
	}

	if len(approvedProblems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid problems to approve"})
		return
	}

	// Save approved problems to database
	problemIDs, err := contestGenerationService.SaveApprovedProblems(
		adminRegdNo,
		job.CourseID,
		approvedProblems,
		&collegeIDStr,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save problems: %v", err)})
		log.Printf("Failed to save approved problems: %v", err)
		return
	}

	// Handle contest association
	var contestID uint

	if req.ContestID != nil {
		// Use existing contest
		contestID = *req.ContestID
	} else if req.CreateContest != nil && *req.CreateContest {
		// Create a new contest
		if req.ContestTitle == "" {
			req.ContestTitle = fmt.Sprintf("Contest - %s", time.Now().Format("2006-01-02"))
		}

		// Parse start and end times
		var startTime, endTime time.Time
		if req.StartTime != "" {
			startTime, err = time.Parse(time.RFC3339, req.StartTime)
			if err != nil {
				startTime = time.Now().AddDate(0, 0, 7) // Default: 1 week from now
			}
		} else {
			startTime = time.Now().AddDate(0, 0, 7)
		}

		if req.EndTime != "" {
			endTime, err = time.Parse(time.RFC3339, req.EndTime)
			if err != nil {
				endTime = startTime.Add(4 * time.Hour) // Default: 4 hours duration
			}
		} else {
			endTime = startTime.Add(4 * time.Hour)
		}

		contest := models.Contest{
			CollegeID:   &collegeIDStr,
			Title:       req.ContestTitle,
			Description: req.ContestDescription,
			StartTime:   startTime,
			EndTime:     endTime,
			CreatedBy:   adminRegdNo,
		}

		if err := database.DB.Create(&contest).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create contest: %v", err)})
			log.Printf("Failed to create contest: %v", err)
			return
		}

		contestID = contest.ContestID
		log.Printf("[ContestGeneration] Created new contest %d: %s", contestID, contest.Title)
	}

	// Add problems to contest if contestID is set
	if contestID > 0 {
		for i, problemID := range problemIDs {
			contestProblem := models.ContestProblem{
				ContestID:    contestID,
				ProblemID:    problemID,
				Points:       100,
				ProblemOrder: i, // Set order based on index
			}
			if err := database.DB.Create(&contestProblem).Error; err != nil {
				log.Printf("Warning: Failed to add problem %d to contest %d: %v", problemID, contestID, err)
			}
		}
		log.Printf("[ContestGeneration] Added %d problems to contest %d", len(problemIDs), contestID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     fmt.Sprintf("Approved %d problems", len(problemIDs)),
		"problem_ids": problemIDs,
	})
}

// RegenerateProblem requests regeneration of a specific problem
func RegenerateProblem(c *gin.Context) {
	initContestGenerationService()

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var req struct {
		Reason       string `json:"reason"`
		ProblemIndex int    `json:"problem_index"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get job and current problems
	job, err := generationJobService.GetJob(uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	var problems []models.GeneratedProblem
	if err := json.Unmarshal(job.GeneratedData, &problems); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse problems"})
		return
	}

	if req.ProblemIndex < 0 || req.ProblemIndex >= len(problems) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem index"})
		return
	}

	// In a full implementation, we would call AI to regenerate just this problem
	// For now, just return a message
	c.JSON(http.StatusOK, gin.H{
		"message": "Problem regeneration requested",
		"note":    "Full regeneration logic to be implemented",
	})
}

// GenerateEditorial generates editorial hints for a contest problem (post-contest)
func GenerateEditorial(c *gin.Context) {
	initContestGenerationService()

	var req models.EditorialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	editorial, err := contestGenerationService.GenerateEditorial(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate editorial: %v", err)})
		log.Printf("Editorial generation failed: %v", err)
		return
	}

	// Save editorial to database
	editorialDB := models.ContestEditorial{
		ContestID:       req.ContestID,
		ProblemID:       req.ProblemID,
		Hint1:           editorial.Hint1,
		Hint2:           editorial.Hint2,
		Hint3:           editorial.Hint3,
		CoreIdea:        editorial.CoreIdea,
		TimeComplexity:  editorial.TimeComplexity,
		SpaceComplexity: editorial.SpaceComplexity,
	}

	if err := database.DB.Create(&editorialDB).Error; err != nil {
		log.Printf("Warning: Failed to save editorial: %v", err)
	}

	c.JSON(http.StatusOK, editorial)
}

// GetEditorial retrieves editorial hints for a contest problem
func GetEditorial(c *gin.Context) {
	contestID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contest ID"})
		return
	}

	problemID, err := strconv.ParseUint(c.Param("problem_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID"})
		return
	}

	var editorial models.ContestEditorial
	if err := database.DB.Where("contest_id = ? AND problem_id = ?", contestID, problemID).First(&editorial).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Editorial not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get editorial"})
		return
	}

	c.JSON(http.StatusOK, editorial)
}

// =====================================================
// Helper Functions
// =====================================================

// verifyJobAccess checks if the user has access to the generation job
// Access is granted if:
// 1. User is a super admin
// 2. User created the job (job.AdminRegdNo matches user's regdno)
// 3. User is an admin of the college that owns the course associated with the job
func verifyJobAccess(c *gin.Context, job *models.GenerationJob, userRegdNo string) bool {
	// Super admins have full access
	if middleware.IsSuperAdmin(c) {
		return true
	}

	// Job creator has access
	if job.AdminRegdNo == userRegdNo {
		return true
	}

	// Check if user is admin of the college that owns the course
	var course models.Course
	if err := database.DB.First(&course, job.CourseID).Error; err != nil {
		return false // Course not found, deny access
	}

	userCollegeID, ok := middleware.GetCurrentUserCollege(c)
	if !ok || userCollegeID == nil {
		return false
	}

	// Allow access if user's college matches the course's college
	return course.CollegeID == *userCollegeID
}
