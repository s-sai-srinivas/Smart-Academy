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
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var lessonPlanService *services.GenerationJobService

// initLessonPlanService initializes the lesson plan service
func initLessonPlanService() {
	if lessonPlanService == nil {
		// Get AI provider from config
		aiProvider := services.NewGeminiProvider(
			config.AppConfig.AIAPIKey,
			config.AppConfig.AIModel,
			config.AppConfig.AIMaxTokens,
			config.AppConfig.AITemperature,
			config.AppConfig.AIRateLimitRPM,
		)
		lessonPlanService = services.NewGenerationJobService(aiProvider)
	}
}

// UploadLessonPlan handles lesson plan file upload and AI processing
func UploadLessonPlan(c *gin.Context) {
	initLessonPlanService()

	// Check if user is admin
	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		// Verify admin role
		userRegdNo, _ := c.Get("regdno")
		var user models.User
		if err := database.DB.Where("regdno = ? AND college_id = ?", userRegdNo, *collegeID).First(&user).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		if user.Role != "admin" && user.Role != "college_admin" && user.Role != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
	}

	// Parse multipart form using the middleware-defined limit for consistency
	if err := c.Request.ParseMultipartForm(middleware.MaxLessonPlanUploadSize); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File too large (max %d MB)", middleware.MaxLessonPlanUploadSize/(1024*1024))})
		return
	}

	file, header, err := c.Request.FormFile("lesson_plan")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lesson plan file required"})
		return
	}
	defer file.Close()

	// Validate file
	allowedTypes := []string{"pdf", "docx", "txt"}
	_ = allowedTypes // suppress unused variable warning until ValidateFileForUpload accepts it
	if err := services.ValidateFileForUpload(header); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get course_id from form
	courseIDStr := c.Request.FormValue("course_id")
	if courseIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Course ID required"})
		return
	}

	var courseID uint
	if _, err := fmt.Sscanf(courseIDStr, "%d", &courseID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course ID"})
		return
	}

	// Verify course exists and belongs to admin's college
	var course models.Course
	if err := database.DB.First(&course, courseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check college ownership if not super admin
	if !middleware.IsSuperAdmin(c) {
		collegeID, _ := middleware.GetCurrentUserCollege(c)
		if course.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: course belongs to another college"})
			return
		}
	}

	// Save uploaded file
	filePath, err := services.SaveUploadedFile(header, "lesson-plans")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Get admin regdno from JWT
	adminRegdNo, _ := c.Get("regdno")
	adminRegdNoStr, ok := adminRegdNo.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication"})
		return
	}

	// Check if theory exists for this course
	var theoryID *uint
	var theory models.Theory
	if err := database.DB.Where("course_id = ?", courseID).First(&theory).Error; err == nil {
		theoryID = &theory.ID
	}

	// Create generation job
	job, err := lessonPlanService.CreateJob(
		adminRegdNoStr,
		&courseID,
		theoryID,
		models.GenerationJobLessonPlanParse,
		filePath,
	)
	if err != nil {
		os.Remove(filePath) // Cleanup
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create generation job"})
		return
	}

	// Start async processing
	go func(jobID uint) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		_ = lessonPlanService.ProcessJob(ctx, jobID)
	}(job.ID)

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":         job.ID,
		"status":         job.Status,
		"message":        "Lesson plan uploaded. Processing started.",
		"estimated_time": "2-5 minutes",
	})
}

// GetGenerationJobStatus returns the status of a generation job
func GetGenerationJobStatus(c *gin.Context) {
	initLessonPlanService()

	jobIDStr := c.Param("id")
	var jobID uint
	if _, err := fmt.Sscanf(jobIDStr, "%d", &jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := lessonPlanService.GetJobWithRelations(jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job status"})
		return
	}

	// Verify access (admin only)
	userRegdNo, _ := c.Get("regdno")
	if job.AdminRegdNo != userRegdNo {
		// Check if super admin
		if !middleware.IsSuperAdmin(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	response := gin.H{
		"id":            job.ID,
		"status":        job.Status,
		"job_type":      job.JobType,
		"created_at":    job.CreatedAt,
		"completed_at":  job.CompletedAt,
		"error_message": job.ErrorMessage,
		"course":        job.Course,
	}

	// Include generated data preview if completed
	if job.Status == models.GenerationJobCompleted && job.GeneratedData != nil {
		var structure models.LessonPlanStructure
		if err := json.Unmarshal(job.GeneratedData, &structure); err == nil {
			weeksPreview := make([]gin.H, 0, len(structure.Weeks))
			for _, week := range structure.Weeks {
				weeksPreview = append(weeksPreview, gin.H{
					"week_number":  week.WeekNumber,
					"week_name":    week.WeekName,
					"module_count": len(week.Modules),
				})
			}
			response["preview"] = gin.H{
				"course_name": structure.CourseName,
				"total_weeks": structure.TotalWeeks,
				"weeks":       weeksPreview,
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetGenerationPreview returns a formatted preview of generated content
func GetGenerationPreview(c *gin.Context) {
	initLessonPlanService()

	jobIDStr := c.Param("id")
	var jobID uint
	if _, err := fmt.Sscanf(jobIDStr, "%d", &jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := lessonPlanService.GetJob(jobID)
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
			"error": fmt.Sprintf("Job not completed yet. Current status: %s", job.Status),
		})
		return
	}

	// Parse generated data
	var structure models.LessonPlanStructure
	if err := json.Unmarshal(job.GeneratedData, &structure); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse generated data"})
		return
	}

	// Build detailed preview
	preview := gin.H{
		"course_name": structure.CourseName,
		"total_weeks": structure.TotalWeeks,
		"weeks":       []gin.H{},
	}

	for i, week := range structure.Weeks {
		weekPreview := gin.H{
			"week_number":  week.WeekNumber,
			"week_name":    week.WeekName,
			"module_count": len(week.Modules),
			"modules":      []gin.H{},
			"has_quiz":     true, // Will be generated
		}

		for j, module := range week.Modules {
			modulePreview := gin.H{
				"module_number":  j + 1,
				"module_name":    module.ModuleName,
				"description":    module.Description,
				"content_length": len(module.Content),
				// Don't include full content in preview, just summary
			}
			weekPreview["modules"] = append(weekPreview["modules"].([]gin.H), modulePreview)
		}

		_ = i // suppress unused variable warning
		preview["weeks"] = append(preview["weeks"].([]gin.H), weekPreview)
	}

	c.JSON(http.StatusOK, preview)
}

// ApproveAndSaveGeneration approves AI-generated content and saves to database
func ApproveAndSaveGeneration(c *gin.Context) {
	initLessonPlanService()

	jobIDStr := c.Param("id")
	var jobID uint
	if _, err := fmt.Sscanf(jobIDStr, "%d", &jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := lessonPlanService.GetJob(jobID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get job"})
		return
	}

	// Parse request body
	var req struct {
		Modifications *models.LessonPlanStructure `json:"modifications"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if job.Status != models.GenerationJobCompleted {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Job not completed yet. Current status: %s", job.Status),
		})
		return
	}

	// Save generated content to database
	var modifications *models.LessonPlanStructure
	if req.Modifications != nil {
		modifications = req.Modifications
	}

	if err := lessonPlanService.SaveGeneratedContent(job, modifications); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to save content: %v", err.Error()),
		})
		return
	}

	// Cleanup uploaded file
	if job.InputFilePath != "" {
		os.Remove(job.InputFilePath)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Course content generated successfully",
		"theory_id": job.TheoryID,
	})
}

// GetPracticeQuizForWeek retrieves the practice quiz for a specific week
func GetPracticeQuizForWeek(c *gin.Context) {
	weekID := c.Param("id")

	var practiceQuiz models.PracticeQuiz
	if err := database.DB.
		Where("theory_week_id = ?", weekID).
		Preload("Questions.Options").
		First(&practiceQuiz).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "No practice quiz found for this week"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get practice quiz"})
		return
	}

	c.JSON(http.StatusOK, practiceQuiz)
}

// UpdatePracticeQuiz updates a practice quiz (admin only)
func UpdatePracticeQuiz(c *gin.Context) {
	quizID := c.Param("id")

	var req struct {
		Title        string `json:"title"`
		Instructions string `json:"instructions"`
		PassingMarks int    `json:"passing_marks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Verify admin access
	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok || collegeID == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
	}

	var quiz models.PracticeQuiz
	if err := database.DB.First(&quiz, quizID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Practice quiz not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get practice quiz"})
		return
	}

	// Update fields
	if req.Title != "" {
		quiz.Title = req.Title
	}
	if req.Instructions != "" {
		quiz.Instructions = req.Instructions
	}
	if req.PassingMarks > 0 {
		quiz.PassingMarks = req.PassingMarks
	}

	if err := database.DB.Save(&quiz).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update practice quiz"})
		return
	}

	c.JSON(http.StatusOK, quiz)
}

// RegeneratePracticeQuiz regenerates a practice quiz using AI
func RegeneratePracticeQuiz(c *gin.Context) {
	weekID := c.Param("id")

	// Get week with modules
	var week models.TheoryWeek
	if err := database.DB.
		Preload("Modules").
		First(&week, weekID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Week not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get week"})
		return
	}

	// Get theory to get course info
	var theory models.Theory
	if err := database.DB.First(&theory, week.TheoryID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get theory"})
		return
	}

	// Create AI provider and generate quiz
	aiProvider := services.NewGeminiProvider(
		config.AppConfig.AIAPIKey,
		config.AppConfig.AIModel,
		config.AppConfig.AIMaxTokens,
		config.AppConfig.AITemperature,
		config.AppConfig.AIRateLimitRPM,
	)

	req := services.PracticeQuizRequest{
		WeekData: models.WeekData{
			WeekNumber: week.WeekOrder,
			WeekName:   week.WeekName,
			Modules:    []models.ModuleData{},
		},
		QuestionCount: 5,
		DifficultyMix: map[string]int{"easy": 2, "medium": 2, "hard": 1},
	}

	for _, module := range week.Modules {
		req.WeekData.Modules = append(req.WeekData.Modules, models.ModuleData{
			ModuleName:  module.ModuleName,
			Description: module.Description,
			Content:     module.Content,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	questions, err := aiProvider.GeneratePracticeQuiz(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to generate quiz: %v", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"questions": questions,
		"message":   "Practice quiz regenerated successfully",
	})
}
