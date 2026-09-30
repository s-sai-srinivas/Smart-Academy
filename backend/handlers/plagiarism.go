package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"coding-platform/services"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CheckSubmissionPlagiarism checks a single submission against all other submissions for the same problem
// Requires admin/faculty role and college isolation
func CheckSubmissionPlagiarism(c *gin.Context) {
	// Get authenticated user's college
	userCollegeID, hasCollege := c.Get("college_id")
	if !hasCollege || userCollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "College assignment required"})
		return
	}

	submissionIDStr := c.Param("id")
	submissionID, err := strconv.ParseUint(submissionIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid submission ID"})
		return
	}

	// Get the submission WITH college validation
	var submission models.Submission
	if err := database.DB.Joins("JOIN problems ON submissions.problem_id = problems.id").
		Where("submissions.id = ? AND problems.college_id = ?", submissionID, *userCollegeID.(*string)).
		First(&submission).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	// Check if language is supported for JPlag
	if !services.IsJPlagLanguageSupported(submission.LanguageID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Language not supported for plagiarism detection"})
		return
	}

	// Get all other submissions for the same problem with the same language
	// Filter by college to prevent cross-college data access
	var otherSubmissions []models.Submission
	if err := database.DB.Joins("JOIN problems ON submissions.problem_id = problems.id").
		Where("submissions.problem_id = ? AND submissions.language_id = ? AND submissions.id != ? AND submissions.passed = ? AND problems.college_id = ?",
			submission.ProblemID, submission.LanguageID, submissionID, true, *userCollegeID.(*string)).
		Find(&otherSubmissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submissions"})
		return
	}

	if len(otherSubmissions) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":    "No other submissions to compare with",
			"results":    []interface{}{},
			"checked_at": time.Now(),
		})
		return
	}

	// Prepare submissions for check
	submissions := []services.SubmissionInfo{
		{
			ID:         submission.ID,
			UserRegdNo: submission.UserRegdNo,
			SourceCode: submission.SourceCode,
			LanguageID: submission.LanguageID,
		},
	}
	for _, sub := range otherSubmissions {
		submissions = append(submissions, services.SubmissionInfo{
			ID:         sub.ID,
			UserRegdNo: sub.UserRegdNo,
			SourceCode: sub.SourceCode,
			LanguageID: sub.LanguageID,
		})
	}

	// Run plagiarism check
	results, err := services.CheckPlagiarism(submission.ProblemID, submissions)
	if err != nil {
		// Log detailed error internally
		log.Printf("[Plagiarism] Check failed: %v", err)
		// Return safe error to user
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Plagiarism check failed. Please try again later.",
		})
		return
	}

	// Filter results to only include current submission
	var filteredResults []services.PlagiarismCheckResult
	for _, r := range results {
		if r.SubmissionID1 == uint(submissionID) || r.SubmissionID2 == uint(submissionID) {
			filteredResults = append(filteredResults, r)
		}
	}

	// Store results in database
	for _, r := range filteredResults {
		plagResult := models.PlagiarismResult{
			SubmissionID1:     r.SubmissionID1,
			SubmissionID2:     r.SubmissionID2,
			SimilarityPercent: r.SimilarityPercent,
			Status:            models.PlagiarismStatus(r.Status),
			CheckedAt:         time.Now(),
		}
		database.DB.Create(&plagResult)
	}

	// Filter out same-user comparisons for the response
	var differentUserResults []services.PlagiarismCheckResult
	userRegdNoMap := make(map[uint]string) // submission_id -> user_regdno
	userRegdNoMap[submission.ID] = submission.UserRegdNo
	for _, sub := range otherSubmissions {
		userRegdNoMap[sub.ID] = sub.UserRegdNo
	}

	for _, r := range filteredResults {
		userRegdNo1 := userRegdNoMap[r.SubmissionID1]
		userRegdNo2 := userRegdNoMap[r.SubmissionID2]
		if userRegdNo1 != userRegdNo2 {
			differentUserResults = append(differentUserResults, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"submission_id":     submissionID,
		"total_comparisons": len(differentUserResults),
		"results":           differentUserResults,
		"checked_at":        time.Now(),
	})
}

// CheckProblemPlagiarism checks all passing submissions for a problem
// Requires admin/faculty role and college isolation
func CheckProblemPlagiarism(c *gin.Context) {
	// Get authenticated user's college
	userCollegeID, hasCollege := c.Get("college_id")
	if !hasCollege || userCollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "College assignment required"})
		return
	}

	problemIDStr := c.Param("id")
	problemID, err := strconv.ParseUint(problemIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID"})
		return
	}

	// Validate problem belongs to user's college
	var problem models.Problem
	if err := database.DB.Where("id = ? AND college_id = ?", problemID, *userCollegeID.(*string)).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Get optional language filter
	languageIDStr := c.Query("language_id")
	var languageFilter *int
	if languageIDStr != "" {
		langID, err := strconv.Atoi(languageIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language_id"})
			return
		}
		if !services.IsJPlagLanguageSupported(langID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Language not supported for plagiarism detection"})
			return
		}
		languageFilter = &langID
	}

	// Get all passing submissions for this problem
	// JOIN with problems to ensure college isolation
	query := database.DB.Table("submissions").
		Joins("JOIN problems ON submissions.problem_id = problems.id").
		Where("submissions.problem_id = ? AND submissions.passed = ? AND problems.college_id = ?", problemID, true, *userCollegeID.(*string))
	if languageFilter != nil {
		query = query.Where("submissions.language_id = ?", *languageFilter)
	}

	var submissions []models.Submission
	if err := query.Preload("User").Order("submitted_at ASC").Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submissions"})
		return
	}

	if len(submissions) < 2 {
		log.Printf("[Plagiarism] Problem %d: Only %d passing submissions found (need at least 2)", problemID, len(submissions))
		c.JSON(http.StatusOK, gin.H{
			"problem_id":        problemID,
			"message":           "Not enough submissions for plagiarism detection",
			"total_submissions": len(submissions),
			"results":           []interface{}{},
		})
		return
	}

	log.Printf("[Plagiarism] Problem %d: Found %d passing submissions", problemID, len(submissions))
	for _, sub := range submissions {
		log.Printf("[Plagiarism]   - Submission %d: UserRegdNo=%s, LanguageID=%d, CodeLen=%d",
			sub.ID, sub.UserRegdNo, sub.LanguageID, len(sub.SourceCode))
	}

	// Group submissions by language for JPlag
	submissionsByLang := make(map[int][]services.SubmissionInfo)
	for _, sub := range submissions {
		if services.IsJPlagLanguageSupported(sub.LanguageID) {
			submissionsByLang[sub.LanguageID] = append(submissionsByLang[sub.LanguageID], services.SubmissionInfo{
				ID:         sub.ID,
				UserRegdNo: sub.UserRegdNo,
				Username:   sub.User.Name,
				SourceCode: sub.SourceCode,
				LanguageID: sub.LanguageID,
			})
		}
	}

	// Run plagiarism check for each language group
	log.Printf("[Plagiarism] Problem %d: %d language groups to check", problemID, len(submissionsByLang))
	var allResults []services.PlagiarismCheckResult
	for langID, subs := range submissionsByLang {
		log.Printf("[Plagiarism] Language %d: %d submissions", langID, len(subs))
		if len(subs) < 2 {
			log.Printf("[Plagiarism] Skipping language %d (need at least 2 submissions)", langID)
			continue
		}

		results, err := services.CheckPlagiarism(uint(problemID), subs)
		if err != nil {
			// Log detailed error internally
			log.Printf("[Plagiarism] Language %d check failed: %v", langID, err)
			// Return safe error to user
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Plagiarism check failed. Please try again later.",
			})
			return
		}
		allResults = append(allResults, results...)
	}

	// Store results in database
	for _, r := range allResults {
		plagResult := models.PlagiarismResult{
			SubmissionID1:     r.SubmissionID1,
			SubmissionID2:     r.SubmissionID2,
			SimilarityPercent: r.SimilarityPercent,
			Status:            models.PlagiarismStatus(r.Status),
			CheckedAt:         time.Now(),
		}
		database.DB.Create(&plagResult)
	}

	// Build user regdno map for filtering same-user comparisons
	userRegdNoMap := make(map[uint]string) // submission_id -> user_regdno
	for _, sub := range submissions {
		userRegdNoMap[sub.ID] = sub.UserRegdNo
	}

	// Filter out same-user comparisons
	var differentUserResults []services.PlagiarismCheckResult
	for _, r := range allResults {
		userRegdNo1 := userRegdNoMap[r.SubmissionID1]
		userRegdNo2 := userRegdNoMap[r.SubmissionID2]
		if userRegdNo1 != userRegdNo2 {
			differentUserResults = append(differentUserResults, r)
		}
	}

	// Count flagged submissions
	flaggedCount := 0
	for _, r := range differentUserResults {
		if r.Status == "SUSPICIOUS" || r.Status == "PLAGIARIZED" {
			flaggedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"problem_id":        problemID,
		"total_submissions": len(submissions),
		"total_comparisons": len(differentUserResults),
		"flagged_count":     flaggedCount,
		"results":           differentUserResults,
		"checked_at":        time.Now(),
	})
}

// GetPlagiarismResults returns stored plagiarism results for a problem
// Requires admin/faculty role and college isolation
func GetPlagiarismResults(c *gin.Context) {
	// Get authenticated user's college
	userCollegeID, hasCollege := c.Get("college_id")
	if !hasCollege || userCollegeID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "College assignment required"})
		return
	}

	problemIDStr := c.Param("problem_id")
	problemID, err := strconv.ParseUint(problemIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid problem ID"})
		return
	}

	// Validate problem belongs to user's college
	var problem models.Problem
	if err := database.DB.Where("id = ? AND college_id = ?", problemID, *userCollegeID.(*string)).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Get all submissions for this problem (already college-validated)
	var submissions []models.Submission
	if err := database.DB.Where("problem_id = ?", problemID).Select("id").Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch submissions"})
		return
	}

	if len(submissions) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"problem_id": problemID,
			"results":    []interface{}{},
		})
		return
	}

	// Get submission IDs
	var submissionIDs []uint
	for _, sub := range submissions {
		submissionIDs = append(submissionIDs, sub.ID)
	}

	// Get plagiarism results
	var results []models.PlagiarismResult
	if err := database.DB.Where("submission_id_1 IN ? OR submission_id_2 IN ?", submissionIDs, submissionIDs).
		Preload("Submission1.User").
		Preload("Submission2.User").
		Order("similarity_percent DESC").
		Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch plagiarism results"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"problem_id": problemID,
		"results":    results,
	})
}
