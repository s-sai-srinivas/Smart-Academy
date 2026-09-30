package services

import (
	"coding-platform/models"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

// =====================================================
// Error Definitions
// =====================================================

var (
	ErrUnauthorized       = errors.New("faculty not authorized for this course offering")
	ErrQuizNotFound       = errors.New("quiz not found")
	ErrQuizNotDraft       = errors.New("quiz must be in draft status")
	ErrQuizNotPublished   = errors.New("quiz must be published")
	ErrQuizClosed         = errors.New("quiz is closed")
	ErrMaxAttemptsReached = errors.New("maximum attempts reached")
	ErrQuizNotStarted     = errors.New("quiz has not started yet")
	ErrQuizEnded          = errors.New("quiz has ended")
	ErrAttemptNotFound    = errors.New("attempt not found")
	ErrNotEnrolled        = errors.New("student not enrolled in this course offering")
	ErrInvalidQuestions   = errors.New("quiz must have at least one valid question")
)

// =====================================================
// Quiz Service
// =====================================================

// QuizService provides business logic for quiz operations
type QuizService struct {
	db *gorm.DB
}

// NewQuizService creates a new QuizService
func NewQuizService(db *gorm.DB) *QuizService {
	return &QuizService{db: db}
}

// =====================================================
// Faculty Authorization Helpers
// =====================================================

// ValidateFacultyOwnership checks if faculty has an active assignment for the course offering
func (s *QuizService) ValidateFacultyOwnership(facultyRegdNo string, courseOfferingID uint) error {
	var count int64
	err := s.db.Table("faculty_assignments").
		Where("faculty_regd_no = ? AND course_offering_id = ? AND is_active = ?", facultyRegdNo, courseOfferingID, true).
		Count(&count).Error

	if err != nil {
		return err
	}

	if count == 0 {
		return ErrUnauthorized
	}

	return nil
}

// GetFacultyOfferings returns all course offerings assigned to a faculty member (theory/integrated only for quizzes)
func (s *QuizService) GetFacultyOfferings(facultyRegdNo string) ([]models.CourseOffering, error) {
	var offerings []models.CourseOffering

	err := s.db.
		Joins("JOIN faculty_assignments ON faculty_assignments.course_offering_id = course_offerings.id").
		Joins("JOIN courses ON courses.id = course_offerings.course_id").
		Preload("Course").
		Preload("Section").
		Where("faculty_assignments.faculty_regd_no = ? AND faculty_assignments.is_active = ?", facultyRegdNo, true).
		Where("courses.course_type IN ?", []string{"theory", "integrated"}).
		Where("course_offerings.deleted_at IS NULL").
		Find(&offerings).Error

	return offerings, err
}

// GetTheoryModulesForOffering returns theory modules for a course offering
func (s *QuizService) GetTheoryModulesForOffering(courseOfferingID uint) ([]models.TheoryWeek, error) {
	var weeks []models.TheoryWeek

	err := s.db.
		Joins("JOIN theories ON theories.id = theory_weeks.theory_id").
		Joins("JOIN courses ON courses.id = theories.course_id").
		Joins("JOIN course_offerings ON course_offerings.course_id = courses.id").
		Where("course_offerings.id = ?", courseOfferingID).
		Preload("Modules").
		Order("theory_weeks.week_order ASC").
		Find(&weeks).Error

	return weeks, err
}

// =====================================================
// Quiz CRUD Operations
// =====================================================

// CreateQuiz creates a new quiz
func (s *QuizService) CreateQuiz(facultyRegdNo string, quiz *models.Quiz, theorySourceIDs []uint) error {
	// Validate faculty ownership
	if err := s.ValidateFacultyOwnership(facultyRegdNo, quiz.CourseOfferingID); err != nil {
		return err
	}

	// Set creator
	quiz.CreatedBy = facultyRegdNo

	// Create quiz with transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(quiz).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Record theory sources if provided
	if len(theorySourceIDs) > 0 {
		for _, moduleID := range theorySourceIDs {
			// Get theory week ID for the module
			var module models.TheoryModule
			if err := tx.First(&module, moduleID).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("theory module %d not found: %w", moduleID, err)
			}

			source := models.QuizTheorySource{
				QuizID:         quiz.ID,
				TheoryModuleID: moduleID,
				TheoryWeekID:   module.TheoryWeekID,
			}

			if err := tx.Create(&source).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

// UpdateQuiz updates an existing quiz
func (s *QuizService) UpdateQuiz(facultyRegdNo string, quizID uint, updates map[string]interface{}) (*models.Quiz, error) {
	// Get quiz and validate ownership
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return nil, ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return nil, ErrUnauthorized
	}

	// Only draft quizzes can be updated
	if quiz.Status != models.QuizStatusDraft {
		return nil, ErrQuizNotDraft
	}

	// Remove protected fields from updates
	delete(updates, "id")
	delete(updates, "created_by")
	delete(updates, "course_offering_id")
	delete(updates, "created_at")

	// Update quiz
	if err := s.db.Model(&quiz).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &quiz, nil
}

// DeleteQuiz soft deletes a quiz
func (s *QuizService) DeleteQuiz(facultyRegdNo string, quizID uint) error {
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return ErrUnauthorized
	}

	// Delete (soft delete via gorm.DeletedAt)
	return s.db.Delete(&quiz).Error
}

// GetQuiz retrieves a quiz with questions and options
func (s *QuizService) GetQuiz(facultyRegdNo string, quizID uint) (*models.Quiz, error) {
	var quiz models.Quiz

	err := s.db.
		Preload("Questions.Options").
		Preload("TheorySources").
		First(&quiz, quizID).Error

	if err != nil {
		return nil, ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return nil, ErrUnauthorized
	}

	return &quiz, nil
}

// ListFacultyQuizzes returns all quizzes for a faculty member's course offering
func (s *QuizService) ListFacultyQuizzes(facultyRegdNo string, courseOfferingID uint) ([]models.Quiz, error) {
	// Validate faculty ownership of the offering
	if err := s.ValidateFacultyOwnership(facultyRegdNo, courseOfferingID); err != nil {
		return nil, err
	}

	var quizzes []models.Quiz
	err := s.db.
		Where("course_offering_id = ? AND created_by = ?", courseOfferingID, facultyRegdNo).
		Order("created_at DESC").
		Find(&quizzes).Error

	return quizzes, err
}

// =====================================================
// Quiz Lifecycle
// =====================================================

// PublishQuiz publishes a quiz (makes it available to students)
func (s *QuizService) PublishQuiz(facultyRegdNo string, quizID uint) error {
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return ErrUnauthorized
	}

	if quiz.Status != models.QuizStatusDraft {
		return ErrQuizNotDraft
	}

	// Validate quiz has questions with correct answers
	var questionCount int64
	s.db.Model(&models.QuizQuestion{}).Where("quiz_id = ?", quizID).Count(&questionCount)

	if questionCount == 0 {
		return ErrInvalidQuestions
	}

	// Validate MCQ questions have options with correct answers
	var invalidMCQs int64
	err := s.db.Table("quiz_questions qq").
		Joins("LEFT JOIN quiz_question_options qo ON qo.question_id = qq.id AND qo.is_correct = ?", true).
		Where("qq.quiz_id = ? AND qq.question_type IN (?, ?, ?)", quizID, "mcq", "multi_select", "true_false").
		Where("qo.id IS NULL").
		Count(&invalidMCQs).Error

	if err != nil {
		return err
	}

	if invalidMCQs > 0 {
		return fmt.Errorf("%d questions have no correct answer marked", invalidMCQs)
	}

	// Update status
	quiz.Status = models.QuizStatusPublished
	return s.db.Save(&quiz).Error
}

// CloseQuiz closes a quiz (prevents new attempts)
func (s *QuizService) CloseQuiz(facultyRegdNo string, quizID uint) error {
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return ErrUnauthorized
	}

	if quiz.Status != models.QuizStatusPublished {
		return errors.New("quiz must be published to close")
	}

	quiz.Status = models.QuizStatusClosed
	return s.db.Save(&quiz).Error
}

// =====================================================
// Question Management
// =====================================================

// SaveQuestions saves questions to a quiz
func (s *QuizService) SaveQuestions(facultyRegdNo string, quizID uint, questions []models.QuizQuestion) error {
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return ErrUnauthorized
	}

	if quiz.Status != models.QuizStatusDraft {
		return ErrQuizNotDraft
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Calculate total marks
	totalMarks := 0
	for i := range questions {
		questions[i].QuizID = quizID
		totalMarks += questions[i].Marks

		// Save options
		options := questions[i].Options
		questions[i].Options = nil // Clear to avoid GORM confusion

		if err := tx.Create(&questions[i]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("error saving question %d: %w", i, err)
		}

		// Save options for this question
		for j := range options {
			options[j].QuestionID = questions[i].ID
			if err := tx.Create(&options[j]).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("error saving option %d for question %d: %w", j, questions[i].ID, err)
			}
		}
	}

	// Update quiz total marks
	quiz.TotalMarks = totalMarks
	if err := tx.Save(&quiz).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// EditQuestion updates a single question
func (s *QuizService) EditQuestion(facultyRegdNo string, quizID, questionID uint, updates map[string]interface{}) error {
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return ErrUnauthorized
	}

	if quiz.Status != models.QuizStatusDraft {
		return ErrQuizNotDraft
	}

	// Verify question belongs to quiz
	var question models.QuizQuestion
	if err := s.db.First(&question, questionID).Error; err != nil {
		return errors.New("question not found")
	}

	if question.QuizID != quizID {
		return errors.New("question does not belong to this quiz")
	}

	delete(updates, "id")
	delete(updates, "quiz_id")

	return s.db.Model(&question).Updates(updates).Error
}

// DeleteQuestion removes a question from a quiz
func (s *QuizService) DeleteQuestion(facultyRegdNo string, quizID, questionID uint) error {
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return ErrUnauthorized
	}

	if quiz.Status != models.QuizStatusDraft {
		return ErrQuizNotDraft
	}

	// Verify question belongs to quiz
	var question models.QuizQuestion
	if err := s.db.First(&question, questionID).Error; err != nil {
		return errors.New("question not found")
	}

	if question.QuizID != quizID {
		return errors.New("question does not belong to this quiz")
	}

	return s.db.Delete(&question).Error
}

// RegenerateQuestions deletes existing questions and regenerates
func (s *QuizService) RegenerateQuestions(facultyRegdNo string, quizID uint, newQuestions []models.QuizQuestion) error {
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return ErrUnauthorized
	}

	if quiz.Status != models.QuizStatusDraft {
		return ErrQuizNotDraft
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete existing questions (cascade will delete options)
	if err := tx.Where("quiz_id = ?", quizID).Delete(&models.QuizQuestion{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Save new questions
	totalMarks := 0
	for i := range newQuestions {
		newQuestions[i].QuizID = quizID
		totalMarks += newQuestions[i].Marks

		options := newQuestions[i].Options
		newQuestions[i].Options = nil

		if err := tx.Create(&newQuestions[i]).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("error saving question %d: %w", i, err)
		}

		for j := range options {
			options[j].QuestionID = newQuestions[i].ID
			if err := tx.Create(&options[j]).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	// Update quiz total marks
	quiz.TotalMarks = totalMarks
	if err := tx.Save(&quiz).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// CloneQuiz clones a quiz to another course offering
func (s *QuizService) CloneQuiz(facultyRegdNo string, sourceQuizID uint, targetOfferingID uint, newTitle string) (*models.Quiz, error) {
	// Verify faculty ownership of source quiz
	var sourceQuiz models.Quiz
	if err := s.db.First(&sourceQuiz, sourceQuizID).Error; err != nil {
		return nil, ErrQuizNotFound
	}

	if sourceQuiz.CreatedBy != facultyRegdNo {
		return nil, ErrUnauthorized
	}

	// Verify faculty has access to target offering
	if err := s.ValidateFacultyOwnership(facultyRegdNo, targetOfferingID); err != nil {
		return nil, err
	}

	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create cloned quiz
	clonedQuiz := sourceQuiz
	clonedQuiz.ID = 0
	clonedQuiz.CourseOfferingID = targetOfferingID
	clonedQuiz.Title = newTitle
	if newTitle == "" {
		clonedQuiz.Title = sourceQuiz.Title + " (Copy)"
	}
	clonedQuiz.Status = models.QuizStatusDraft
	clonedQuiz.CreatedAt = time.Now()
	clonedQuiz.UpdatedAt = time.Now()

	if err := tx.Create(&clonedQuiz).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Clone questions and options
	var questions []models.QuizQuestion
	if err := tx.Where("quiz_id = ?", sourceQuizID).Find(&questions).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for i := range questions {
		oldID := questions[i].ID
		questions[i].ID = 0
		questions[i].QuizID = clonedQuiz.ID

		// Get options for this question
		var options []models.QuizQuestionOption
		if err := tx.Where("question_id = ?", oldID).Find(&options).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		questions[i].Options = nil
		if err := tx.Create(&questions[i]).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		// Clone options
		for j := range options {
			options[j].ID = 0
			options[j].QuestionID = questions[i].ID
			if err := tx.Create(&options[j]).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &clonedQuiz, nil
}

// =====================================================
// Student Operations
// =====================================================

// ListAvailableQuizzes returns published quizzes for a single enrolled course offering
func (s *QuizService) ListAvailableQuizzes(studentRegdNo string, courseOfferingID uint) ([]models.Quiz, error) {
	// Verify student is enrolled in the offering
	var enrollment models.Enrollment
	err := s.db.
		Where("student_regdno = ? AND course_offering_id = ? AND status = ?", studentRegdNo, courseOfferingID, "enrolled").
		First(&enrollment).Error

	if err != nil {
		return nil, ErrNotEnrolled
	}

	var quizzes []models.Quiz
	now := time.Now()

	err = s.db.
		Where("course_offering_id = ? AND status = ?", courseOfferingID, models.QuizStatusPublished).
		Where("(scheduled_start IS NULL OR scheduled_start <= ?)", now).
		Where("(scheduled_end IS NULL OR scheduled_end > ?)", now).
		Order("scheduled_end ASC").
		Find(&quizzes).Error

	return quizzes, err
}

// StudentQuizItem holds quiz data enriched with course info and attempt counts
type StudentQuizItem struct {
	CreatedAt        time.Time         `json:"created_at"`
	ScheduledStart   *time.Time        `json:"scheduled_start"`
	ScheduledEnd     *time.Time        `json:"scheduled_end"`
	Status           models.QuizStatus `json:"status"`
	QuizType         models.QuizType   `json:"quiz_type"`
	CourseCode       string            `json:"course_code"`
	CourseName       string            `json:"course_name"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	PassingMarks     int               `json:"passing_marks"`
	QuestionCount    int               `json:"question_count"`
	MaxAttempts      int               `json:"max_attempts"`
	ID               uint              `json:"id"`
	CourseOfferingID uint              `json:"course_offering_id"`
	TotalMarks       int               `json:"total_marks"`
	TimeLimitMinutes int               `json:"time_limit_minutes"`
	AttemptsUsed     int64             `json:"attempts_used"`
}

// ListAllAvailableQuizzes returns published quizzes across ALL enrolled course offerings
func (s *QuizService) ListAllAvailableQuizzes(studentRegdNo string) ([]StudentQuizItem, error) {
	now := time.Now()

	// Single efficient query: join quizzes → course_offerings → courses, filtered by enrollment
	type quizRow struct {
		CreatedAt        time.Time
		ScheduledStart   *time.Time
		ScheduledEnd     *time.Time
		Title            string
		Description      string
		QuizType         string
		Status           string
		CourseCode       string
		CourseName       string
		MaxAttempts      int
		ID               uint
		PassingMarks     int
		CourseOfferingID uint
		TotalMarks       int
		TimeLimitMinutes int
		QuestionCount    int
	}

	var rows []quizRow
	err := s.db.Raw(`
		SELECT
			q.id, q.title, q.description, q.quiz_type, q.status,
			q.time_limit_minutes, q.total_marks, q.passing_marks, q.max_attempts,
			q.scheduled_start, q.scheduled_end, q.created_at,
			q.course_offering_id,
			c.course_name, c.course_code,
			(SELECT COUNT(*) FROM quiz_questions qq WHERE qq.quiz_id = q.id) as question_count
		FROM quizzes q
		INNER JOIN enrollments e ON e.course_offering_id = q.course_offering_id
		INNER JOIN course_offerings co ON co.id = q.course_offering_id
		INNER JOIN courses c ON c.id = co.course_id
		WHERE e.student_regdno = ?
		  AND e.status = 'enrolled'
		  AND q.status = ?
		  AND q.deleted_at IS NULL
		  AND (q.scheduled_start IS NULL OR q.scheduled_start <= ?)
		  AND (q.scheduled_end IS NULL OR q.scheduled_end > ?)
		ORDER BY q.scheduled_end ASC NULLS LAST, q.created_at DESC
	`, studentRegdNo, models.QuizStatusPublished, now, now).Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	// Batch-fetch attempt counts for this student on these quizzes
	quizIDs := make([]uint, len(rows))
	for i, r := range rows {
		quizIDs[i] = r.ID
	}

	attemptCounts := make(map[uint]int64)
	if len(quizIDs) > 0 {
		type attemptRow struct {
			QuizID uint
			Cnt    int64
		}
		var attempts []attemptRow
		s.db.Raw(`
			SELECT quiz_id, COUNT(*) as cnt
			FROM quiz_attempts
			WHERE student_regdno = ? AND quiz_id IN ?
			GROUP BY quiz_id
		`, studentRegdNo, quizIDs).Scan(&attempts)
		for _, a := range attempts {
			attemptCounts[a.QuizID] = a.Cnt
		}
	}

	// Build result
	items := make([]StudentQuizItem, len(rows))
	for i, r := range rows {
		items[i] = StudentQuizItem{
			ID:               r.ID,
			Title:            r.Title,
			Description:      r.Description,
			QuizType:         models.QuizType(r.QuizType),
			Status:           models.QuizStatus(r.Status),
			TimeLimitMinutes: r.TimeLimitMinutes,
			TotalMarks:       r.TotalMarks,
			PassingMarks:     r.PassingMarks,
			MaxAttempts:      r.MaxAttempts,
			QuestionCount:    r.QuestionCount,
			ScheduledStart:   r.ScheduledStart,
			ScheduledEnd:     r.ScheduledEnd,
			CreatedAt:        r.CreatedAt,
			CourseOfferingID: r.CourseOfferingID,
			CourseName:       r.CourseName,
			CourseCode:       r.CourseCode,
			AttemptsUsed:     attemptCounts[r.ID],
		}
	}

	return items, nil
}

// GetQuizForStudent returns quiz details without correct answers
func (s *QuizService) GetQuizForStudent(studentRegdNo string, quizID uint) (*models.Quiz, error) {
	// Get quiz
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return nil, ErrQuizNotFound
	}

	if quiz.Status != models.QuizStatusPublished {
		return nil, ErrQuizNotPublished
	}

	// Verify enrollment
	var enrollment models.Enrollment
	err := s.db.
		Where("student_regdno = ? AND course_offering_id = ? AND status = ?", studentRegdNo, quiz.CourseOfferingID, "enrolled").
		First(&enrollment).Error

	if err != nil {
		return nil, ErrNotEnrolled
	}

	// Check schedule
	now := time.Now()
	if quiz.ScheduledStart != nil && now.Before(*quiz.ScheduledStart) {
		return nil, ErrQuizNotStarted
	}

	// Load questions without options' is_correct field
	var questions []models.QuizQuestion
	s.db.Where("quiz_id = ?", quizID).Order("order_index ASC").Find(&questions)

	for i := range questions {
		var options []models.QuizQuestionOption
		s.db.Where("question_id = ?", questions[i].ID).Order("order_index ASC").Find(&options)

		// Clear is_correct for student view
		for j := range options {
			options[j].IsCorrect = false
		}

		questions[i].Options = options
	}

	// Apply shuffling if enabled (uses student-specific seed for consistency across refreshes)
	if quiz.ShuffleQuestions || quiz.ShuffleOptions {
		// Create a deterministic seed based on student and quiz for consistent shuffle across refreshes
		seed := int64(hashString(studentRegdNo)) ^ int64(quizID)
		r := rand.New(rand.NewSource(seed))

		// Shuffle questions if enabled
		if quiz.ShuffleQuestions && len(questions) > 1 {
			r.Shuffle(len(questions), func(i, j int) {
				questions[i], questions[j] = questions[j], questions[i]
			})
		}

		// Shuffle options if enabled
		if quiz.ShuffleOptions {
			for i := range questions {
				if len(questions[i].Options) > 1 {
					r.Shuffle(len(questions[i].Options), func(a, b int) {
						questions[i].Options[a], questions[i].Options[b] = questions[i].Options[b], questions[i].Options[a]
					})
				}
			}
		}
	}

	quiz.Questions = questions

	return &quiz, nil
}

// StartAttempt creates a new quiz attempt for a student
func (s *QuizService) StartAttempt(studentRegdNo string, quizID uint) (*models.QuizAttempt, error) {
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return nil, ErrQuizNotFound
	}

	// Verify enrollment
	var enrollment models.Enrollment
	err := s.db.
		Where("student_regdno = ? AND course_offering_id = ? AND status = ?", studentRegdNo, quiz.CourseOfferingID, "enrolled").
		First(&enrollment).Error

	if err != nil {
		return nil, ErrNotEnrolled
	}

	// Check quiz status
	if quiz.Status != models.QuizStatusPublished {
		return nil, ErrQuizNotPublished
	}

	// Check schedule
	now := time.Now()
	if quiz.ScheduledStart != nil && now.Before(*quiz.ScheduledStart) {
		return nil, ErrQuizNotStarted
	}

	if quiz.ScheduledEnd != nil && now.After(*quiz.ScheduledEnd) {
		return nil, ErrQuizEnded
	}

	// Use PostgreSQL advisory lock to prevent race condition on attempt counting
	// This ensures count + create is atomic across concurrent requests
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Acquire advisory lock for this quiz+student combination
	// Using a hash of quizID and studentRegdNo to generate a unique lock key
	lockKey := int64(quizID)<<32 ^ int64(hashString(studentRegdNo))
	tx.Exec("SELECT pg_advisory_xact_lock(?)", lockKey)

	// Check max attempts within transaction
	var attemptCount int64
	err = tx.Model(&models.QuizAttempt{}).
		Where("quiz_id = ? AND student_regdno = ?", quizID, studentRegdNo).
		Count(&attemptCount).Error

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if attemptCount >= int64(quiz.MaxAttempts) {
		tx.Rollback()
		return nil, ErrMaxAttemptsReached
	}

	// Create attempt
	attempt := models.QuizAttempt{
		QuizID:        quizID,
		StudentRegdNo: studentRegdNo,
		AttemptNumber: int(attemptCount) + 1,
		StartedAt:     now,
		Status:        models.AttemptInProgress,
		MaxScore:      quiz.TotalMarks,
	}

	// Set deadline for timed quizzes
	if quiz.TimeLimitMinutes > 0 {
		deadline := now.Add(time.Duration(quiz.TimeLimitMinutes) * time.Minute)
		attempt.DeadlineAt = &deadline
	}

	if err := tx.Create(&attempt).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return &attempt, nil
}

// hashString generates a simple hash for use in lock key generation
func hashString(s string) uint32 {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}

// GetActiveAttempt returns student's in-progress attempt for a quiz
// Returns error if no active attempt exists or if the attempt has expired
func (s *QuizService) GetActiveAttempt(studentRegdNo string, quizID uint) (*models.QuizAttempt, error) {
	var attempt models.QuizAttempt

	err := s.db.
		Where("quiz_id = ? AND student_regdno = ? AND status = ?", quizID, studentRegdNo, models.AttemptInProgress).
		First(&attempt).Error

	if err != nil {
		return nil, err
	}

	// Check if attempt has expired (server-side timer enforcement)
	if attempt.DeadlineAt != nil && time.Now().After(*attempt.DeadlineAt) {
		// Auto-mark as timed out
		now := time.Now()
		attempt.Status = models.AttemptTimedOut
		attempt.SubmittedAt = &now
		s.db.Save(&attempt)
		return nil, errors.New("attempt has expired")
	}

	return &attempt, nil
}

// =====================================================
// Answer Submission Types
// =====================================================

// AnswerSubmission represents a single question answer
type AnswerSubmission struct {
	TextAnswer        string `json:"text_answer"`
	SelectedOptionIDs []uint `json:"selected_option_ids"`
	QuestionID        uint   `json:"question_id"`
}

// QuizAttemptResult represents the result of a quiz attempt
type QuizAttemptResult struct {
	Attempt    *models.QuizAttempt `json:"attempt"`
	Answers    []models.QuizAnswer `json:"answers"`
	Percentage float64             `json:"percentage"`
	Passed     bool                `json:"passed"`
	TimedOut   bool                `json:"timed_out"` // true if submission was after time limit
}

// =====================================================
// Student Submission and Grading
// =====================================================

// SubmitAttempt submits a student's quiz attempt and auto-grades it
func (s *QuizService) SubmitAttempt(studentRegdNo string, attemptID uint, answers []AnswerSubmission) (*QuizAttemptResult, error) {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get attempt
	var attempt models.QuizAttempt
	if err := tx.First(&attempt, attemptID).Error; err != nil {
		tx.Rollback()
		return nil, ErrAttemptNotFound
	}

	// Verify ownership
	if attempt.StudentRegdNo != studentRegdNo {
		tx.Rollback()
		return nil, ErrUnauthorized
	}

	// Check attempt status
	if attempt.Status != models.AttemptInProgress {
		tx.Rollback()
		return nil, fmt.Errorf("attempt is already %s", attempt.Status)
	}

	// Get quiz
	var quiz models.Quiz
	if err := tx.First(&quiz, attempt.QuizID).Error; err != nil {
		tx.Rollback()
		return nil, ErrQuizNotFound
	}

	// Check timeout BEFORE grading - use server-enforced deadline if available
	now := time.Now()
	isExpired := false

	// Primary check: use DeadlineAt if set (most reliable)
	if attempt.DeadlineAt != nil && now.After(*attempt.DeadlineAt) {
		isExpired = true
	} else if quiz.TimeLimitMinutes > 0 && attempt.DeadlineAt == nil {
		// Fallback: check against TimeLimitMinutes (for attempts created before DeadlineAt was added)
		timeTaken := now.Sub(attempt.StartedAt)
		if timeTaken > time.Duration(quiz.TimeLimitMinutes)*time.Minute {
			isExpired = true
		}
	}

	if isExpired {
		// Mark as timed out without grading
		attempt.Status = models.AttemptTimedOut
		attempt.SubmittedAt = &now
		if err := tx.Save(&attempt).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		tx.Commit()
		return &QuizAttemptResult{
			Attempt:  &attempt,
			TimedOut: true,
		}, nil
	}

	// Get all questions for this quiz
	var questions []models.QuizQuestion
	if err := tx.Where("quiz_id = ?", quiz.ID).Find(&questions).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Build question map for quick lookup
	questionMap := make(map[uint]models.QuizQuestion)
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	// Grade each answer
	score := 0
	quizAnswers := make([]models.QuizAnswer, 0, len(answers))

	for _, ans := range answers {
		question, exists := questionMap[ans.QuestionID]
		if !exists {
			continue // Skip unknown questions
		}

		quizAnswer := models.QuizAnswer{
			AttemptID:         attemptID,
			QuestionID:        ans.QuestionID,
			SelectedOptionIDs: ans.SelectedOptionIDs,
			TextAnswer:        ans.TextAnswer,
		}

		// Grade the answer
		isCorrect, marksAwarded := s.gradeAnswer(question, ans)
		quizAnswer.IsCorrect = isCorrect
		quizAnswer.MarksAwarded = marksAwarded

		if isCorrect {
			score += marksAwarded
		}

		quizAnswers = append(quizAnswers, quizAnswer)
	}

	// Save all answers
	for i := range quizAnswers {
		if err := tx.Create(&quizAnswers[i]).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("error saving answer %d: %w", i, err)
		}
	}

	// Update attempt
	attempt.SubmittedAt = &now
	attempt.Status = models.AttemptGraded
	attempt.Score = score

	// Calculate percentage with guard against division by zero
	if quiz.TotalMarks > 0 {
		attempt.Percentage = float64(score) / float64(quiz.TotalMarks) * 100
	} else {
		attempt.Percentage = 0
	}

	if err := tx.Save(&attempt).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Determine pass/fail with guard against division by zero
	passed := false
	if quiz.TotalMarks > 0 {
		passed = attempt.Percentage >= float64(quiz.PassingMarks)/float64(quiz.TotalMarks)*100
	}

	return &QuizAttemptResult{
		Attempt:    &attempt,
		Answers:    quizAnswers,
		Percentage: attempt.Percentage,
		Passed:     passed,
	}, nil
}

// gradeAnswer grades a single answer and returns correctness and marks
func (s *QuizService) gradeAnswer(question models.QuizQuestion, submission AnswerSubmission) (bool, int) {
	switch question.QuestionType {
	case models.QuestionTypeMCQ, models.QuestionTypeTrueFalse:
		// Get correct options
		var correctOptions []models.QuizQuestionOption
		s.db.Where("question_id = ? AND is_correct = ?", question.ID, true).Find(&correctOptions)

		if len(correctOptions) == 0 || len(submission.SelectedOptionIDs) == 0 {
			return false, 0
		}

		// For MCQ, student should select exactly one option
		// Award marks if that option is among the correct ones (handles data issues with multiple correct options)
		if len(submission.SelectedOptionIDs) == 1 {
			for _, correct := range correctOptions {
				if submission.SelectedOptionIDs[0] == correct.ID {
					return true, question.Marks
				}
			}
		}
		return false, 0

	case models.QuestionTypeMultiSelect:
		// Get all correct option IDs
		var correctOptionIDs []uint
		_ = s.db.Model(&models.QuizQuestionOption{}).
			Where("question_id = ? AND is_correct = ?", question.ID, true).
			Pluck("id", &correctOptionIDs).Error

		if len(correctOptionIDs) == 0 {
			return false, 0
		}

		// Build map of correct options
		correctMap := make(map[uint]bool)
		for _, id := range correctOptionIDs {
			correctMap[id] = true
		}

		// Count correct and incorrect selections
		correctSelected := 0
		incorrectSelected := 0
		for _, selectedID := range submission.SelectedOptionIDs {
			if correctMap[selectedID] {
				correctSelected++
			} else {
				incorrectSelected++
			}
		}

		// Full marks: all correct selected and no incorrect selections
		if correctSelected == len(correctOptionIDs) && incorrectSelected == 0 {
			return true, question.Marks
		}

		// Partial credit: (correct - incorrect) / total_correct * marks
		// This penalizes wrong selections while rewarding correct ones
		if correctSelected > 0 && len(correctOptionIDs) > 0 {
			netCorrect := correctSelected - incorrectSelected
			if netCorrect > 0 {
				partialMarks := float64(netCorrect) / float64(len(correctOptionIDs)) * float64(question.Marks)
				return false, int(partialMarks)
			}
		}

		return false, 0

	case models.QuestionTypeShortAnswer, models.QuestionTypeFillBlank:
		// For text answers, use simple string matching
		if submission.TextAnswer == "" {
			return false, 0
		}

		// Get correct option text
		var correctOptions []models.QuizQuestionOption
		s.db.Where("question_id = ? AND is_correct = ?", question.ID, true).Find(&correctOptions)

		if len(correctOptions) == 0 {
			return false, 0
		}

		// Check against any correct option text
		for _, opt := range correctOptions {
			if normalizeText(submission.TextAnswer) == normalizeText(opt.OptionText) {
				return true, question.Marks
			}
		}

		// Partial credit for containing key terms
		for _, opt := range correctOptions {
			normalizedSubmitted := normalizeText(submission.TextAnswer)
			normalizedCorrect := normalizeText(opt.OptionText)

			words := splitWords(normalizedCorrect)
			allMatch := true
			for _, word := range words {
				if len(word) > 2 && !containsWord(normalizedSubmitted, word) {
					allMatch = false
					break
				}
			}
			if allMatch && len(words) > 0 {
				return true, question.Marks
			}
		}

		return false, 0

	default:
		return false, 0
	}
}

// GetAttemptResult returns a student's attempt result with answers
func (s *QuizService) GetAttemptResult(studentRegdNo string, quizID uint) (*QuizAttemptResult, error) {
	// Get latest submitted attempt
	var attempt models.QuizAttempt
	err := s.db.
		Where("quiz_id = ? AND student_regdno = ? AND status IN (?, ?)", quizID, studentRegdNo, models.AttemptSubmitted, models.AttemptGraded).
		Order("submitted_at DESC").
		First(&attempt).Error

	if err != nil {
		return nil, ErrAttemptNotFound
	}

	// Get answers
	var answers []models.QuizAnswer
	if err := s.db.Where("attempt_id = ?", attempt.ID).Find(&answers).Error; err != nil {
		return nil, err
	}

	// Get quiz for pass/fail calculation
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return nil, ErrQuizNotFound
	}

	// Calculate passed status with guard against division by zero
	passed := false
	if quiz.TotalMarks > 0 {
		passed = attempt.Percentage >= float64(quiz.PassingMarks)/float64(quiz.TotalMarks)*100
	}

	return &QuizAttemptResult{
		Attempt:    &attempt,
		Answers:    answers,
		Percentage: attempt.Percentage,
		Passed:     passed,
	}, nil
}

// ListStudentAttempts returns all attempts for a quiz (for faculty to review)
func (s *QuizService) ListStudentAttempts(facultyRegdNo string, quizID uint) ([]models.QuizAttempt, error) {
	// Verify ownership
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return nil, ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return nil, ErrUnauthorized
	}

	var attempts []models.QuizAttempt
	err := s.db.
		Where("quiz_id = ?", quizID).
		Preload("Student").
		Order("submitted_at DESC").
		Find(&attempts).Error

	return attempts, err
}

// =====================================================
// Quiz Analytics Types
// =====================================================

// QuizAnalyticsResult represents quiz analytics data
type QuizAnalyticsResult struct {
	QuizTitle          string              `json:"quiz_title"`
	QuestionsAnalytics []QuestionAnalytics `json:"questions_analytics"`
	DifficultyResults  DifficultyResults   `json:"difficulty_results"`
	LowestScore        int                 `json:"lowest_score"`
	AveragePercentage  float64             `json:"average_percentage"`
	HighestScore       int                 `json:"highest_score"`
	QuizID             uint                `json:"quiz_id"`
	PassCount          int64               `json:"pass_count"`
	FailCount          int64               `json:"fail_count"`
	PassRate           float64             `json:"pass_rate"`
	CompletionRate     float64             `json:"completion_rate"`
	AverageScore       float64             `json:"average_score"`
	TotalAttempts      int64               `json:"total_attempts"`
}

// QuestionAnalytics represents analytics for a single question
type QuestionAnalytics struct {
	QuestionText   string  `json:"question_text"`
	QuestionType   string  `json:"question_type"`
	Difficulty     string  `json:"difficulty"`
	BloomLevel     string  `json:"bloom_level"`
	QuestionID     uint    `json:"question_id"`
	TotalAttempts  int64   `json:"total_attempts"`
	CorrectCount   int64   `json:"correct_count"`
	IncorrectCount int64   `json:"incorrect_count"`
	AccuracyRate   float64 `json:"accuracy_rate"`
	Marks          int     `json:"marks"`
}

// DifficultyResults represents results grouped by difficulty
type DifficultyResults struct {
	Easy   DifficultyStats `json:"easy"`
	Medium DifficultyStats `json:"medium"`
	Hard   DifficultyStats `json:"hard"`
}

// DifficultyStats represents statistics for a difficulty level
type DifficultyStats struct {
	QuestionCount int64   `json:"question_count"`
	TotalAttempts int64   `json:"total_attempts"`
	CorrectCount  int64   `json:"correct_count"`
	AccuracyRate  float64 `json:"accuracy_rate"`
}

// =====================================================
// Quiz Analytics
// =====================================================

// GetQuizAnalytics returns comprehensive analytics for a quiz
func (s *QuizService) GetQuizAnalytics(facultyRegdNo string, quizID uint) (*QuizAnalyticsResult, error) {
	// Verify ownership
	var quiz models.Quiz
	if err := s.db.First(&quiz, quizID).Error; err != nil {
		return nil, ErrQuizNotFound
	}

	if quiz.CreatedBy != facultyRegdNo {
		return nil, ErrUnauthorized
	}

	result := &QuizAnalyticsResult{
		QuizID:    quizID,
		QuizTitle: quiz.Title,
	}

	// Total attempts
	s.db.Model(&models.QuizAttempt{}).
		Where("quiz_id = ?", quizID).
		Count(&result.TotalAttempts)

	if result.TotalAttempts == 0 {
		return result, nil
	}

	// Score statistics
	var scoreStats struct {
		AvgScore      float64
		AvgPercentage float64
		MaxScore      int
		MinScore      int
	}

	s.db.Model(&models.QuizAttempt{}).
		Where("quiz_id = ? AND status IN (?, ?)", quizID, models.AttemptSubmitted, models.AttemptGraded).
		Select("AVG(score) as avg_score, AVG(percentage) as avg_percentage, MAX(score) as max_score, MIN(score) as min_score").
		Scan(&scoreStats)

	result.AverageScore = scoreStats.AvgScore
	result.AveragePercentage = scoreStats.AvgPercentage
	result.HighestScore = scoreStats.MaxScore
	result.LowestScore = scoreStats.MinScore

	// Pass/fail counts with guard against division by zero
	var passingThreshold float64
	if quiz.TotalMarks > 0 {
		passingThreshold = float64(quiz.PassingMarks) / float64(quiz.TotalMarks) * 100
	}

	s.db.Model(&models.QuizAttempt{}).
		Where("quiz_id = ? AND percentage >= ?", quizID, passingThreshold).
		Count(&result.PassCount)

	result.FailCount = result.TotalAttempts - result.PassCount

	// Calculate pass rate with guard against division by zero
	if result.TotalAttempts > 0 {
		result.PassRate = float64(result.PassCount) / float64(result.TotalAttempts) * 100
	}

	// Completion rate (submitted vs in-progress)
	var completedCount int64
	s.db.Model(&models.QuizAttempt{}).
		Where("quiz_id = ? AND status IN (?, ?)", quizID, models.AttemptSubmitted, models.AttemptGraded).
		Count(&completedCount)

	result.CompletionRate = float64(completedCount) / float64(result.TotalAttempts) * 100

	// Question-level analytics
	result.QuestionsAnalytics = s.getQuestionsAnalytics(quizID)

	// Difficulty-based analytics
	result.DifficultyResults = s.getDifficultyResults(quizID)

	return result, nil
}

// getQuestionsAnalytics returns analytics for each question in a quiz
func (s *QuizService) getQuestionsAnalytics(quizID uint) []QuestionAnalytics {
	var analytics []QuestionAnalytics

	// Get all questions
	var questions []models.QuizQuestion
	s.db.Where("quiz_id = ?", quizID).Find(&questions)

	for _, q := range questions {
		qa := QuestionAnalytics{
			QuestionID:   q.ID,
			QuestionText: q.QuestionText,
			QuestionType: string(q.QuestionType),
			Difficulty:   string(q.Difficulty),
			BloomLevel:   string(q.BloomLevel),
			Marks:        q.Marks,
		}

		// Count total attempts for this question
		s.db.Model(&models.QuizAnswer{}).
			Joins("JOIN quiz_attempts ON quiz_attempts.id = quiz_answers.attempt_id").
			Where("quiz_answers.question_id = ? AND quiz_attempts.quiz_id = ?", q.ID, quizID).
			Count(&qa.TotalAttempts)

		// Count correct answers
		s.db.Model(&models.QuizAnswer{}).
			Joins("JOIN quiz_attempts ON quiz_attempts.id = quiz_answers.attempt_id").
			Where("quiz_answers.question_id = ? AND quiz_attempts.quiz_id = ? AND quiz_answers.is_correct = ?", q.ID, quizID, true).
			Count(&qa.CorrectCount)

		qa.IncorrectCount = qa.TotalAttempts - qa.CorrectCount

		if qa.TotalAttempts > 0 {
			qa.AccuracyRate = float64(qa.CorrectCount) / float64(qa.TotalAttempts) * 100
		}

		analytics = append(analytics, qa)
	}

	return analytics
}

// getDifficultyResults returns analytics grouped by difficulty
func (s *QuizService) getDifficultyResults(quizID uint) DifficultyResults {
	results := DifficultyResults{}

	difficulties := []string{"easy", "medium", "hard"}

	for _, diff := range difficulties {
		stats := DifficultyStats{}

		// Count questions with this difficulty
		s.db.Model(&models.QuizQuestion{}).
			Where("quiz_id = ? AND difficulty = ?", quizID, diff).
			Count(&stats.QuestionCount)

		if stats.QuestionCount == 0 {
			switch diff {
			case "easy":
				results.Easy = stats
			case "medium":
				results.Medium = stats
			case "hard":
				results.Hard = stats
			}
			continue
		}

		// Get question IDs with this difficulty
		var questionIDs []uint
		_ = s.db.Model(&models.QuizQuestion{}).
			Where("quiz_id = ? AND difficulty = ?", quizID, diff).
			Pluck("id", &questionIDs).Error

		if len(questionIDs) == 0 {
			continue
		}

		// Count total attempts for these questions
		s.db.Model(&models.QuizAnswer{}).
			Joins("JOIN quiz_attempts ON quiz_attempts.id = quiz_answers.attempt_id").
			Where("quiz_answers.question_id IN ? AND quiz_attempts.quiz_id = ?", questionIDs, quizID).
			Count(&stats.TotalAttempts)

		// Count correct answers
		s.db.Model(&models.QuizAnswer{}).
			Joins("JOIN quiz_attempts ON quiz_attempts.id = quiz_answers.attempt_id").
			Where("quiz_answers.question_id IN ? AND quiz_attempts.quiz_id = ? AND quiz_answers.is_correct = ?", questionIDs, quizID, true).
			Count(&stats.CorrectCount)

		if stats.TotalAttempts > 0 {
			stats.AccuracyRate = float64(stats.CorrectCount) / float64(stats.TotalAttempts) * 100
		}

		switch diff {
		case "easy":
			results.Easy = stats
		case "medium":
			results.Medium = stats
		case "hard":
			results.Hard = stats
		}
	}

	return results
}

// =====================================================
// Helper Functions
// =====================================================

// normalizeText normalizes text for comparison
func normalizeText(text string) string {
	result := ""
	for _, r := range text {
		if r >= 'A' && r <= 'Z' {
			result += string(r + 32) // Convert to lowercase
		} else {
			result += string(r)
		}
	}
	return result
}

// splitWords splits text into words
func splitWords(text string) []string {
	var words []string
	current := ""
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

// containsWord checks if text contains a word
func containsWord(text, word string) bool {
	if len(word) == 0 {
		return true
	}
	for i := 0; i <= len(text)-len(word); i++ {
		if text[i:i+len(word)] == word {
			return true
		}
	}
	return false
}

// For JSON marshaling of map keys
var _ = json.Marshal
