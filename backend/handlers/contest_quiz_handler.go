package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ==================== ADMIN CONTEST QUIZ HANDLERS ====================

// UpsertContestQuiz creates or updates a quiz for a contest section
func UpsertContestQuiz(c *gin.Context) {
	contestID := c.Param("id")

	var req struct {
		SectionID       *uint                        `json:"section_id"`
		Title           string                       `json:"title"`
		Instructions    string                       `json:"instructions"`
		Questions       []models.ContestQuizQuestion `json:"questions"`
		TotalMarks      int                          `json:"total_marks"`
		PassingMarks    int                          `json:"passing_marks"`
		DurationMinutes int                          `json:"duration_minutes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var contest models.Contest
	if err := database.DB.First(&contest, contestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Find existing quiz by section_id or contest_id (legacy)
	var existingQuiz models.ContestQuiz
	var err error
	if req.SectionID != nil && *req.SectionID > 0 {
		err = database.DB.Where("section_id = ?", *req.SectionID).First(&existingQuiz).Error
	} else {
		// Legacy: find by contest_id without section
		err = database.DB.Where("contest_id = ? AND section_id IS NULL", contestID).First(&existingQuiz).Error
	}

	var quiz models.ContestQuiz
	if err == nil {
		quiz = existingQuiz
	} else {
		cID, _ := strconv.ParseUint(contestID, 10, 64)
		quiz = models.ContestQuiz{ContestID: uint(cID), SectionID: req.SectionID}
	}

	quiz.Title = req.Title
	quiz.Instructions = req.Instructions
	quiz.TotalMarks = req.TotalMarks
	quiz.PassingMarks = req.PassingMarks
	quiz.DurationMinutes = req.DurationMinutes

	if err := database.DB.Save(&quiz).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save quiz"})
		return
	}

	// Update contest's has_quiz flag
	contest.HasQuiz = true
	if req.DurationMinutes > 0 {
		contest.QuizDurationMinutes = req.DurationMinutes
	}
	database.DB.Save(&contest)

	// Replace questions if provided
	if req.Questions != nil {
		database.DB.Where("contest_quiz_id = ?", quiz.ID).Delete(&models.ContestQuizQuestion{})

		for i := range req.Questions {
			q := req.Questions[i]
			q.ContestQuizID = quiz.ID
			q.OrderIndex = i
			if err := database.DB.Create(&q).Error; err != nil {
				continue
			}
			for j := range q.Options {
				q.Options[j].QuestionID = q.ID
				q.Options[j].OrderIndex = j
				database.DB.Create(&q.Options[j])
			}
		}
	}

	database.DB.Preload("Questions.Options").First(&quiz, quiz.ID)
	c.JSON(http.StatusOK, gin.H{"quiz": quiz, "message": "Quiz saved successfully"})
}

// GetContestQuiz returns the quiz for a contest (admin view, includes answers)
// Accepts optional ?section_id= query param
func GetContestQuiz(c *gin.Context) {
	contestID := c.Param("id")

	var quiz models.ContestQuiz
	query := database.DB.Where("contest_id = ?", contestID)
	if sectionIDStr := c.Query("section_id"); sectionIDStr != "" {
		query = query.Where("section_id = ?", sectionIDStr)
	}
	if err := query.
		Preload("Questions", func(db *gorm.DB) *gorm.DB { return db.Order("order_index ASC") }).
		Preload("Questions.Options", func(db *gorm.DB) *gorm.DB { return db.Order("order_index ASC") }).
		First(&quiz).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"quiz": quiz})
}

// DeleteContestQuiz deletes a contest quiz and all its questions
func DeleteContestQuiz(c *gin.Context) {
	contestID := c.Param("id")

	var quiz models.ContestQuiz
	if err := database.DB.Where("contest_id = ?", contestID).First(&quiz).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	if err := database.DB.Delete(&quiz).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete quiz"})
		return
	}

	// Update contest's has_quiz flag
	database.DB.Model(&models.Contest{}).Where("contest_id = ?", contestID).
		Updates(map[string]interface{}{"has_quiz": false, "quiz_duration_minutes": 0})

	c.JSON(http.StatusOK, gin.H{"message": "Quiz deleted successfully"})
}

// ==================== STUDENT CONTEST QUIZ HANDLERS ====================

// GetContestQuizForStudent returns quiz info without revealing correct answers
func GetContestQuizForStudent(c *gin.Context) {
	contestID := c.Param("id")
	userRegdNo, _ := c.Get("regdno")

	var contest models.Contest
	if err := database.DB.First(&contest, contestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	if !contest.HasQuiz {
		c.JSON(http.StatusNotFound, gin.H{"error": "This contest does not include a quiz"})
		return
	}

	var quiz models.ContestQuiz
	if err := database.DB.Where("contest_id = ?", contestID).
		Preload("Questions", func(db *gorm.DB) *gorm.DB { return db.Order("order_index ASC") }).
		Preload("Questions.Options", func(db *gorm.DB) *gorm.DB { return db.Order("order_index ASC") }).
		First(&quiz).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	// Check for existing attempt
	var attempt models.ContestQuizAttempt
	database.DB.Where("contest_quiz_id = ? AND user_regd_no = ?", quiz.ID, userRegdNo).
		Order("id DESC").First(&attempt)

	// Strip correct answers from options
	for i := range quiz.Questions {
		for j := range quiz.Questions[i].Options {
			quiz.Questions[i].Options[j].IsCorrect = false
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"quiz":           quiz,
		"has_attempted":  attempt.ID != 0 && attempt.SubmittedAt != nil,
		"is_in_progress": attempt.ID != 0 && attempt.SubmittedAt == nil,
	})
}

// StartContestQuiz starts a quiz attempt for a student
func StartContestQuiz(c *gin.Context) {
	contestID := c.Param("id")
	userRegdNo, _ := c.Get("regdno")

	var quiz models.ContestQuiz
	if err := database.DB.Where("contest_id = ?", contestID).First(&quiz).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	// Check for existing unsubmitted attempt
	var existing models.ContestQuizAttempt
	err := database.DB.Where("contest_quiz_id = ? AND user_regd_no = ? AND submitted_at IS NULL",
		quiz.ID, userRegdNo).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"attempt": existing, "message": "Attempt already in progress"})
		return
	}

	// Check if already submitted
	err = database.DB.Where("contest_quiz_id = ? AND user_regd_no = ? AND submitted_at IS NOT NULL",
		quiz.ID, userRegdNo).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have already submitted this quiz"})
		return
	}

	attempt := models.ContestQuizAttempt{
		ContestQuizID: quiz.ID,
		ContestID:     uint(getContestID(contestID)),
		UserRegdNo:    userRegdNo.(string),
		StartedAt:     time.Now(),
	}

	if err := database.DB.Create(&attempt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start quiz"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"attempt": attempt, "message": "Quiz started"})
}

// SubmitContestQuiz submits quiz answers and calculates score
func SubmitContestQuiz(c *gin.Context) {
	contestID := c.Param("id")
	userRegdNo, _ := c.Get("regdno")

	var req struct {
		Answers []struct {
			QuestionID       uint `json:"question_id"`
			SelectedOptionID uint `json:"selected_option_id"`
		} `json:"answers"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var quiz models.ContestQuiz
	if err := database.DB.Where("contest_id = ?", contestID).
		Preload("Questions.Options").
		First(&quiz).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	var attempt models.ContestQuizAttempt
	if err := database.DB.Where("contest_quiz_id = ? AND user_regd_no = ? AND submitted_at IS NULL",
		quiz.ID, userRegdNo).First(&attempt).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active quiz attempt found"})
		return
	}

	// Build lookup: question_id -> correct option_id
	correctLookup := make(map[uint]uint)
	marksLookup := make(map[uint]int)
	for _, q := range quiz.Questions {
		marksLookup[q.ID] = q.Marks
		for _, opt := range q.Options {
			if opt.IsCorrect {
				correctLookup[q.ID] = opt.ID
				break
			}
		}
	}

	totalScore := 0
	maxScore := 0
	for _, q := range quiz.Questions {
		maxScore += q.Marks
	}

	for _, ans := range req.Answers {
		correctID, hasCorrect := correctLookup[ans.QuestionID]
		isCorrect := hasCorrect && ans.SelectedOptionID == correctID
		marks := 0
		if isCorrect {
			marks = marksLookup[ans.QuestionID]
		}

		answer := models.ContestQuizAnswer{
			AttemptID:        attempt.ID,
			QuestionID:       ans.QuestionID,
			SelectedOptionID: &ans.SelectedOptionID,
			IsCorrect:        isCorrect,
			MarksObtained:    marks,
		}
		database.DB.Create(&answer)

		totalScore += marks
	}

	now := time.Now()
	attempt.SubmittedAt = &now
	attempt.TotalScore = totalScore
	attempt.MaxScore = maxScore
	if quiz.PassingMarks > 0 {
		attempt.Passed = totalScore >= quiz.PassingMarks
	} else {
		attempt.Passed = totalScore > 0
	}
	database.DB.Save(&attempt)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Quiz submitted successfully",
		"score":     totalScore,
		"max_score": maxScore,
		"passed":    attempt.Passed,
	})
}

// GetContestQuizResult returns the result of a student's quiz attempt
func GetContestQuizResult(c *gin.Context) {
	contestID := c.Param("id")
	userRegdNo, _ := c.Get("regdno")

	var quiz models.ContestQuiz
	if err := database.DB.Where("contest_id = ?", contestID).First(&quiz).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	var attempt models.ContestQuizAttempt
	if err := database.DB.Where("contest_quiz_id = ? AND user_regd_no = ? AND submitted_at IS NOT NULL",
		quiz.ID, userRegdNo).
		Preload("Answers").
		Preload("Answers.Attempt").
		Order("id DESC").First(&attempt).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No completed quiz attempt found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"attempt":   attempt,
		"score":     attempt.TotalScore,
		"max_score": attempt.MaxScore,
		"passed":    attempt.Passed,
	})
}

func getContestID(contestIDStr string) uint64 {
	id, _ := strconv.ParseUint(contestIDStr, 10, 64)
	return id
}
