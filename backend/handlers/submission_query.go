package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetSubmissions returns all submissions for a user or problem
// SECURITY: Requires authentication and filters by college
func GetSubmissions(c *gin.Context) {
	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
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

	var submissions []models.Submission
	query := database.DB.Preload("User").Preload("Problem")

	// Join with problems table to filter by college
	query = query.Joins("JOIN problems ON submissions.problem_id = problems.id").
		Where("problems.college_id = ?", *user.CollegeID)

	// Filter by user_regdno if provided (only for same user or admin)
	role, _ := c.Get("role")
	if filterUserRegdNo := c.Query("user_regdno"); filterUserRegdNo != "" {
		if role != "admin" && filterUserRegdNo != userRegdNo.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view other users' submissions"})
			return
		}
		query = query.Where("user_regd_no = ?", filterUserRegdNo)
	}
	// Also support legacy user_id parameter for backward compatibility
	if userID := c.Query("user_id"); userID != "" {
		if role != "admin" && userID != userRegdNo.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view other users' submissions"})
			return
		}
		query = query.Where("user_regd_no = ?", userID)
	}

	// Filter by problem_id if provided
	if problemID := c.Query("problem_id"); problemID != "" {
		query = query.Where("problem_id = ?", problemID)
	}

	// Order by most recent first
	query = query.Order("submitted_at DESC")

	// Limit results
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	query = query.Limit(limit)

	if err := query.Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"submissions": submissions})
}

// GetSubmission returns a single submission by ID
// SECURITY: Requires authentication and validates college access
func GetSubmission(c *gin.Context) {
	id := c.Param("id")

	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
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

	// Get submission with college validation
	var submission models.Submission
	err := database.DB.
		Preload("User").
		Preload("Problem").
		Joins("JOIN problems ON submissions.problem_id = problems.id").
		Where("submissions.id = ? AND problems.college_id = ?", id, *user.CollegeID).
		First(&submission).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	// Only allow users to view their own submissions (or admins)
	role, _ := c.Get("role")
	if role != "admin" && submission.UserRegdNo != userRegdNo.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view other users' submissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"submission": submission})
}

// GetUserSubmissions returns all submissions for the authenticated user
func GetUserSubmissions(c *gin.Context) {
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var submissions []models.Submission
	query := database.DB.Preload("Problem").Where("user_regd_no = ?", userRegdNo)

	// Filter by problem_id if provided
	if problemID := c.Query("problem_id"); problemID != "" {
		query = query.Where("problem_id = ?", problemID)
	}

	// Order by most recent first
	query = query.Order("submitted_at DESC").Limit(50)

	if err := query.Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"submissions": submissions})
}

// GetProblemSubmissions returns paginated submissions for a specific problem
func GetProblemSubmissions(c *gin.Context) {
	problemID := c.Param("id")

	// Parse pagination parameters
	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	offset := (page - 1) * limit

	// Get total count
	var total int64
	countQuery := database.DB.Model(&models.Submission{}).Where("problem_id = ?", problemID)

	// Users can only see their own submissions, admins can see all
	role, _ := c.Get("role")
	userRegdNo, userExists := c.Get("regdno")

	if role != "admin" {
		if !userExists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}
		countQuery = countQuery.Where("user_regd_no = ?", userRegdNo)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count submissions"})
		return
	}

	// Fetch submissions
	var submissions []models.Submission
	query := database.DB.Select("id, user_regd_no, problem_id, language_id, passed, total_tests, passed_tests, score, max_score, execution_time, memory_used, submitted_at").
		Where("problem_id = ?", problemID)

	if role != "admin" {
		query = query.Where("user_regd_no = ?", userRegdNo)
	}

	query = query.Order("submitted_at DESC").
		Limit(limit).
		Offset(offset)

	if err := query.Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submissions"})
		return
	}

	// Calculate pagination info
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	})
}

// GetSubmissionStats returns statistics for a user or problem
// SECURITY: Requires authentication and filters by college
func GetSubmissionStats(c *gin.Context) {
	// Require authentication
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
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

	type Stats struct {
		TotalSubmissions  int64   `json:"total_submissions"`
		PassedSubmissions int64   `json:"passed_submissions"`
		FailedSubmissions int64   `json:"failed_submissions"`
		SuccessRate       float64 `json:"success_rate"`
	}

	var stats Stats
	query := database.DB.Model(&models.Submission{}).
		Joins("JOIN problems ON submissions.problem_id = problems.id").
		Where("problems.college_id = ?", *user.CollegeID)

	// Filter by user_regdno if provided
	role, _ := c.Get("role")
	if filterUserRegdNo := c.Query("user_regdno"); filterUserRegdNo != "" {
		if role != "admin" && filterUserRegdNo != userRegdNo.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view other users' statistics"})
			return
		}
		query = query.Where("user_regd_no = ?", filterUserRegdNo)
	}
	// Also support legacy user_id parameter for backward compatibility
	if userID := c.Query("user_id"); userID != "" {
		if role != "admin" && userID != userRegdNo.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view other users' statistics"})
			return
		}
		query = query.Where("user_regd_no = ?", userID)
	}

	// Filter by problem_id if provided
	if problemID := c.Query("problem_id"); problemID != "" {
		query = query.Where("problem_id = ?", problemID)
	}

	// Get total count
	query.Count(&stats.TotalSubmissions)

	// Get passed count (same filters)
	passedQuery := database.DB.Model(&models.Submission{}).
		Joins("JOIN problems ON submissions.problem_id = problems.id").
		Where("problems.college_id = ?", *user.CollegeID)

	if filterUserRegdNo := c.Query("user_regdno"); filterUserRegdNo != "" {
		if role == "admin" || filterUserRegdNo == userRegdNo.(string) {
			passedQuery = passedQuery.Where("user_regd_no = ?", filterUserRegdNo)
		}
	}
	if userID := c.Query("user_id"); userID != "" {
		if role == "admin" || userID == userRegdNo.(string) {
			passedQuery = passedQuery.Where("user_regd_no = ?", userID)
		}
	}
	if problemID := c.Query("problem_id"); problemID != "" {
		passedQuery = passedQuery.Where("problem_id = ?", problemID)
	}

	passedQuery.Where("passed = ?", true).Count(&stats.PassedSubmissions)

	stats.FailedSubmissions = stats.TotalSubmissions - stats.PassedSubmissions

	if stats.TotalSubmissions > 0 {
		stats.SuccessRate = float64(stats.PassedSubmissions) / float64(stats.TotalSubmissions) * 100
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserCompletedProblems returns all problem IDs that the authenticated user has completed
func GetUserCompletedProblems(c *gin.Context) {
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var completions []models.UserProblemCompletion
	if err := database.DB.Where("user_regd_no = ?", userRegdNo).Find(&completions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch completed problems"})
		return
	}

	// Extract problem IDs
	problemIDs := make([]uint, len(completions))
	for i, completion := range completions {
		problemIDs[i] = completion.ProblemID
	}

	c.JSON(http.StatusOK, gin.H{
		"completed_problem_ids": problemIDs,
		"total_completed":       len(problemIDs),
	})
}

// CheckProblemCompletion checks if the authenticated user has completed a specific problem
func CheckProblemCompletion(c *gin.Context) {
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	problemID := c.Param("id")
	if problemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Problem ID is required"})
		return
	}

	var completion models.UserProblemCompletion
	err := database.DB.Where("user_regd_no = ? AND problem_id = ?", userRegdNo, problemID).First(&completion).Error

	if err != nil {
		// No completion record found
		c.JSON(http.StatusOK, gin.H{
			"completed":    false,
			"completed_at": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"completed":          true,
		"completed_at":       completion.CompletedAt,
		"time_taken_seconds": completion.TimeTakenSeconds,
	})
}

// GetSubmissionCode returns the source code for a specific submission
// This is used for lazy loading to avoid fetching all code in the list
func GetSubmissionCode(c *gin.Context) {
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	submissionID := c.Param("id")
	if submissionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Submission ID is required"})
		return
	}

	// Get submission with college validation
	var submission models.Submission
	err := database.DB.
		Select("id, source_code, user_regd_no, problem_id, language_id").
		Joins("JOIN problems ON submissions.problem_id = problems.id").
		Where("submissions.id = ?", submissionID).
		First(&submission).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	// Only allow users to view their own submissions (or admins)
	role, _ := c.Get("role")
	if role != "admin" && submission.UserRegdNo != userRegdNo.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view other users' code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          submission.ID,
		"source_code": submission.SourceCode,
		"language_id": submission.LanguageID,
	})
}
