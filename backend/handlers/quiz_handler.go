package handlers

import (
	"coding-platform/config"
	"coding-platform/database"
	"coding-platform/models"
	"coding-platform/services"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// =====================================================
// Response Types
// =====================================================

// FacultyOfferingResponse represents a course offering for faculty dropdown
type FacultyOfferingResponse struct {
	CourseName       string `json:"course_name"`
	SectionName      string `json:"section_name"`
	BranchName       string `json:"branch_name"`
	DisplayName      string `json:"display_name"`
	ID               uint   `json:"id"`
	CourseOfferingID uint   `json:"course_offering_id"`
	CourseID         uint   `json:"course_id"`
	CohortYear       int    `json:"cohort_year"`
}

// QuizListResponse represents a quiz in list view
type QuizListResponse struct {
	CreatedAt        time.Time         `json:"created_at"`
	ScheduledStart   *time.Time        `json:"scheduled_start"`
	ScheduledEnd     *time.Time        `json:"scheduled_end"`
	AttemptCount     *int64            `json:"attempt_count,omitempty"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	QuizType         models.QuizType   `json:"quiz_type"`
	Status           models.QuizStatus `json:"status"`
	ID               uint              `json:"id"`
	TimeLimitMinutes int               `json:"time_limit_minutes"`
	TotalMarks       int               `json:"total_marks"`
	PassingMarks     int               `json:"passing_marks"`
}

// QuizDetailResponse represents a quiz with full details
type QuizDetailResponse struct {
	UpdatedAt        time.Time               `json:"updated_at"`
	CreatedAt        time.Time               `json:"created_at"`
	ScheduledStart   *time.Time              `json:"scheduled_start"`
	ScheduledEnd     *time.Time              `json:"scheduled_end"`
	AIPrompt         string                  `json:"ai_prompt,omitempty"`
	Title            string                  `json:"title"`
	Status           models.QuizStatus       `json:"status"`
	AIModelUsed      string                  `json:"ai_model_used,omitempty"`
	Description      string                  `json:"description"`
	QuizType         models.QuizType         `json:"quiz_type"`
	ShowResultsAfter models.ShowResultsAfter `json:"show_results_after"`
	Questions        []QuizQuestionResponse  `json:"questions,omitempty"`
	TheorySources    []TheorySourceResponse  `json:"theory_sources,omitempty"`
	MaxAttempts      int                     `json:"max_attempts"`
	PassingMarks     int                     `json:"passing_marks"`
	ID               uint                    `json:"id"`
	TotalMarks       int                     `json:"total_marks"`
	TimeLimitMinutes int                     `json:"time_limit_minutes"`
	CourseOfferingID uint                    `json:"course_offering_id"`
	ShuffleOptions   bool                    `json:"shuffle_options"`
	ShuffleQuestions bool                    `json:"shuffle_questions"`
}

// QuizQuestionResponse represents a question with options
type QuizQuestionResponse struct {
	SourceModuleID *uint                     `json:"source_module_id,omitempty"`
	QuestionType   models.QuestionType       `json:"question_type"`
	QuestionText   string                    `json:"question_text"`
	Explanation    string                    `json:"explanation"`
	Difficulty     models.QuestionDifficulty `json:"difficulty"`
	BloomLevel     models.BloomLevel         `json:"bloom_level"`
	Options        []QuizOptionResponse      `json:"options,omitempty"`
	ID             uint                      `json:"id"`
	QuizID         uint                      `json:"quiz_id"`
	Marks          int                       `json:"marks"`
	OrderIndex     int                       `json:"order_index"`
	IsAIGenerated  bool                      `json:"is_ai_generated"`
}

// QuizOptionResponse represents a question option
type QuizOptionResponse struct {
	OptionText string `json:"option_text"`
	ID         uint   `json:"id"`
	QuestionID uint   `json:"question_id"`
	OrderIndex int    `json:"order_index"`
	IsCorrect  bool   `json:"is_correct"`
}

// TheorySourceResponse represents a theory source
type TheorySourceResponse struct {
	ModuleName     string `json:"module_name,omitempty"`
	WeekName       string `json:"week_name,omitempty"`
	ID             uint   `json:"id"`
	QuizID         uint   `json:"quiz_id"`
	TheoryModuleID uint   `json:"theory_module_id"`
	TheoryWeekID   uint   `json:"theory_week_id"`
}

// TheoryModuleTreeResponse represents theory weeks with modules
type TheoryModuleTreeResponse struct {
	WeekName  string                 `json:"week_name"`
	Modules   []TheoryModuleResponse `json:"modules"`
	ID        uint                   `json:"id"`
	WeekOrder int                    `json:"week_order"`
}

// TheoryModuleResponse represents a theory module
type TheoryModuleResponse struct {
	ModuleName  string `json:"module_name"`
	Description string `json:"description"`
	ID          uint   `json:"id"`
}

// GenerateQuizRequest represents the AI generation request
type GenerateQuizRequest struct {
	DifficultyMix    map[string]int `json:"difficulty_mix"`
	TeacherPrompt    string         `json:"teacher_prompt"`
	TheoryModuleIDs  []uint         `json:"theory_module_ids"`
	QuestionTypes    []string       `json:"question_types"`
	CourseOfferingID uint           `json:"course_offering_id"`
	QuestionCount    int            `json:"question_count"`
}

// GenerateQuizPreviewResponse represents the AI generation preview
type GenerateQuizPreviewResponse struct {
	Warning   string                       `json:"warning,omitempty"`
	Questions []services.GeneratedQuestion `json:"questions"`
}

// CreateQuizRequest represents the quiz creation request
type CreateQuizRequest struct {
	ScheduledStart   *time.Time              `json:"scheduled_start"`
	ScheduledEnd     *time.Time              `json:"scheduled_end"`
	ShowResultsAfter models.ShowResultsAfter `json:"show_results_after"`
	QuizType         models.QuizType         `json:"quiz_type"`
	Description      string                  `json:"description"`
	Title            string                  `json:"title"`
	AIPrompt         string                  `json:"ai_prompt"`
	TheoryModuleIDs  []uint                  `json:"theory_module_ids"`
	TimeLimitMinutes int                     `json:"time_limit_minutes"`
	PassingMarks     int                     `json:"passing_marks"`
	MaxAttempts      int                     `json:"max_attempts"`
	CourseOfferingID uint                    `json:"course_offering_id"`
	ShuffleQuestions bool                    `json:"shuffle_questions"`
	ShuffleOptions   bool                    `json:"shuffle_options"`
}

// SubmitQuizAttemptRequest represents student answer submission
type SubmitQuizAttemptRequest struct {
	Answers []AnswerSubmissionRequest `json:"answers"`
}

// AnswerSubmissionRequest represents a single answer
type AnswerSubmissionRequest struct {
	TextAnswer        string `json:"text_answer"`
	SelectedOptionIDs []uint `json:"selected_option_ids"`
	QuestionID        uint   `json:"question_id"`
}

// StudentQuizResponse represents a quiz for student view (without correct answers)
type StudentQuizResponse struct {
	Title            string                    `json:"title"`
	Description      string                    `json:"description"`
	Questions        []StudentQuestionResponse `json:"questions"`
	ID               uint                      `json:"id"`
	TimeLimitMinutes int                       `json:"time_limit_minutes"`
	TotalMarks       int                       `json:"total_marks"`
	PassingMarks     int                       `json:"passing_marks"`
	ShuffleQuestions bool                      `json:"shuffle_questions"`
	ShuffleOptions   bool                      `json:"shuffle_options"`
}

// StudentQuestionResponse represents a question for student view
type StudentQuestionResponse struct {
	QuestionType models.QuestionType  `json:"question_type"`
	QuestionText string               `json:"question_text"`
	Options      []QuizOptionResponse `json:"options"`
	ID           uint                 `json:"id"`
	Marks        int                  `json:"marks"`
}

// QuizAttemptResponse represents a quiz attempt
type QuizAttemptResponse struct {
	StartedAt     time.Time            `json:"started_at"`
	SubmittedAt   *time.Time           `json:"submitted_at"`
	StudentRegdNo string               `json:"student_regdno"`
	StudentName   string               `json:"student_name,omitempty"`
	Status        models.AttemptStatus `json:"status"`
	ID            uint                 `json:"id"`
	QuizID        uint                 `json:"quiz_id"`
	AttemptNumber int                  `json:"attempt_number"`
	Score         int                  `json:"score"`
	MaxScore      int                  `json:"max_score"`
	Percentage    float64              `json:"percentage"`
}

// QuizResultResponse represents quiz result for student
type QuizResultResponse struct {
	Attempt    *QuizAttemptResponse    `json:"attempt"`
	Quiz       *QuizDetailResponse     `json:"quiz,omitempty"`
	Answers    []StudentAnswerResponse `json:"answers"`
	Percentage float64                 `json:"percentage"`
	Passed     bool                    `json:"passed"`
}

// StudentAnswerResponse represents an answer with explanation
type StudentAnswerResponse struct {
	QuestionText      string               `json:"question_text"`
	QuestionType      models.QuestionType  `json:"question_type"`
	TextAnswer        string               `json:"text_answer"`
	Explanation       string               `json:"explanation,omitempty"`
	SelectedOptionIDs []uint               `json:"selected_option_ids"`
	CorrectOptions    []QuizOptionResponse `json:"correct_options,omitempty"`
	QuestionID        uint                 `json:"question_id"`
	MarksAwarded      int                  `json:"marks_awarded"`
	IsCorrect         bool                 `json:"is_correct"`
}

// =====================================================
// Global quiz service instance
// =====================================================

var quizServiceInstance *services.QuizService

// getQuizService returns the quiz service instance
func getQuizService() *services.QuizService {
	if quizServiceInstance == nil {
		quizServiceInstance = services.NewQuizService(database.DB)
	}
	return quizServiceInstance
}

// =====================================================
// Faculty Routes - Offerings & Theory Modules
// =====================================================

// GetMyOfferings returns course offerings assigned to the faculty
func GetMyOfferings(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	offerings, err := getQuizService().GetFacultyOfferings(facultyRegdNo.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Format response
	response := make([]FacultyOfferingResponse, len(offerings))
	for i, o := range offerings {
		response[i] = FacultyOfferingResponse{
			ID:               o.ID,
			CourseOfferingID: o.ID,
			CourseID:         o.CourseID,
			CourseName:       o.Course.CourseName,
			SectionName:      "",
			BranchName:       "",
			CohortYear:       0,
			DisplayName:      formatOfferingDisplayName(o),
		}

		// Load section info if available
		if o.Section != nil {
			response[i].SectionName = o.Section.SectionName
			response[i].CohortYear = o.Section.CohortYear
		}
	}

	c.JSON(http.StatusOK, gin.H{"offerings": response})
}

// GetTheoryModulesForOffering returns theory modules for a course offering
func GetTheoryModulesForOffering(c *gin.Context) {
	courseOfferingIDStr := c.Param("id")
	courseOfferingID, err := strconv.ParseUint(courseOfferingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course offering ID"})
		return
	}

	weeks, err := getQuizService().GetTheoryModulesForOffering(uint(courseOfferingID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Format response as tree
	response := make([]TheoryModuleTreeResponse, len(weeks))
	for i, week := range weeks {
		modules := make([]TheoryModuleResponse, len(week.Modules))
		for j, module := range week.Modules {
			modules[j] = TheoryModuleResponse{
				ID:          module.ID,
				ModuleName:  module.ModuleName,
				Description: module.Description,
			}
		}

		response[i] = TheoryModuleTreeResponse{
			ID:        week.ID,
			WeekName:  week.WeekName,
			WeekOrder: week.WeekOrder,
			Modules:   modules,
		}
	}

	c.JSON(http.StatusOK, gin.H{"weeks": response})
}

// =====================================================
// Faculty Routes - AI Quiz Generation
// =====================================================

// GenerateAIQuestions generates quiz questions using AI (preview only, not saved)
func GenerateAIQuestions(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req GenerateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := getQuizService().ValidateFacultyOwnership(facultyRegdNo.(string), req.CourseOfferingID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Get theory modules
	weeks, err := getQuizService().GetTheoryModulesForOffering(req.CourseOfferingID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Filter to selected modules and build context
	var selectedModules []services.ModuleContent
	for _, week := range weeks {
		for _, module := range week.Modules {
			// If specific modules requested, filter
			if len(req.TheoryModuleIDs) > 0 {
				found := false
				for _, id := range req.TheoryModuleIDs {
					if module.ID == id {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			selectedModules = append(selectedModules, services.ModuleContent{
				WeekName:       week.WeekName,
				ModuleName:     module.ModuleName,
				Description:    module.Description,
				Content:        module.Content,
				TheoryWeekID:   week.ID,
				TheoryModuleID: module.ID,
			})
		}
	}

	if len(selectedModules) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no theory modules selected"})
		return
	}

	// Check token budget
	totalTokens := services.EstimateTotalTokens(selectedModules, req.TeacherPrompt)
	if totalTokens > 28000 {
		c.JSON(http.StatusOK, gin.H{
			"questions": []services.GeneratedQuestion{},
			"warning":   "Content exceeds token limit. Please select fewer modules.",
		})
		return
	}

	// Get AI provider
	aiProvider, err := services.GetAIProvider(
		config.AppConfig.AIProvider,
		config.AppConfig.AIAPIKey,
		config.AppConfig.AIModel,
		config.AppConfig.AIMaxTokens,
		config.AppConfig.AITemperature,
		config.AppConfig.AIRateLimitRPM,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI provider not configured"})
		return
	}

	// Generate questions
	genReq := services.QuizGenerationRequest{
		TheoryContent: selectedModules,
		TeacherPrompt: req.TeacherPrompt,
		QuestionCount: req.QuestionCount,
		DifficultyMix: req.DifficultyMix,
		QuestionTypes: req.QuestionTypes,
	}

	questions, err := aiProvider.GenerateQuizQuestions(c.Request.Context(), genReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"questions": questions,
	})
}

// =====================================================
// Faculty Routes - Quiz CRUD
// =====================================================

// CreateQuiz creates a new quiz
func CreateQuiz(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quiz := &models.Quiz{
		CourseOfferingID: req.CourseOfferingID,
		Title:            req.Title,
		Description:      req.Description,
		QuizType:         req.QuizType,
		Status:           models.QuizStatusDraft,
		TimeLimitMinutes: req.TimeLimitMinutes,
		PassingMarks:     req.PassingMarks,
		MaxAttempts:      req.MaxAttempts,
		ShuffleQuestions: req.ShuffleQuestions,
		ShuffleOptions:   req.ShuffleOptions,
		ShowResultsAfter: req.ShowResultsAfter,
		ScheduledStart:   req.ScheduledStart,
		ScheduledEnd:     req.ScheduledEnd,
		AIPrompt:         req.AIPrompt,
		AIModelUsed:      config.AppConfig.AIModel,
	}

	err := getQuizService().CreateQuiz(facultyRegdNo.(string), quiz, req.TheoryModuleIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"quiz": quiz, "message": "Quiz created successfully"})
}

// ListMyQuizzes lists all quizzes for a faculty member
func ListMyQuizzes(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	courseOfferingIDStr := c.Query("course_offering_id")
	if courseOfferingIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_offering_id is required"})
		return
	}

	courseOfferingID, err := strconv.ParseUint(courseOfferingIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course offering ID"})
		return
	}

	quizzes, err := getQuizService().ListFacultyQuizzes(facultyRegdNo.(string), uint(courseOfferingID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Format response
	response := make([]QuizListResponse, len(quizzes))
	for i, q := range quizzes {
		response[i] = QuizListResponse{
			ID:               q.ID,
			Title:            q.Title,
			Description:      q.Description,
			QuizType:         q.QuizType,
			Status:           q.Status,
			TimeLimitMinutes: q.TimeLimitMinutes,
			TotalMarks:       q.TotalMarks,
			PassingMarks:     q.PassingMarks,
			ScheduledStart:   q.ScheduledStart,
			ScheduledEnd:     q.ScheduledEnd,
			CreatedAt:        q.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"quizzes": response})
}

// GetQuiz gets a single quiz with details
func GetQuiz(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	quiz, err := getQuizService().GetQuiz(facultyRegdNo.(string), uint(quizID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"quiz": formatQuizDetail(quiz)})
}

// UpdateQuiz updates an existing quiz
func UpdateQuiz(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quiz, err := getQuizService().UpdateQuiz(facultyRegdNo.(string), uint(quizID), updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"quiz": quiz, "message": "Quiz updated successfully"})
}

// DeleteQuiz deletes a quiz
func DeleteQuiz(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	if err := getQuizService().DeleteQuiz(facultyRegdNo.(string), uint(quizID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quiz deleted successfully"})
}

// PublishQuiz publishes a quiz
func PublishQuiz(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	if err := getQuizService().PublishQuiz(facultyRegdNo.(string), uint(quizID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quiz published successfully"})
}

// CloseQuiz closes a quiz
func CloseQuiz(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	if err := getQuizService().CloseQuiz(facultyRegdNo.(string), uint(quizID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quiz closed successfully"})
}

// =====================================================
// Faculty Routes - Question Management
// =====================================================

// SaveQuestions saves questions to a quiz
func SaveQuestions(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	// Accept both { "questions": [...] } wrapper and raw [...] array
	type incomingOption struct {
		ID         interface{} `json:"id"`
		OptionText string      `json:"option_text"`
		IsCorrect  bool        `json:"is_correct"`
		OrderIndex int         `json:"order_index"`
	}
	type incomingQuestion struct {
		SourceModuleID   *uint            `json:"source_module_id,omitempty"`
		QuestionType     string           `json:"question_type"`
		QuestionText     string           `json:"question_text"`
		Explanation      string           `json:"explanation"`
		Difficulty       string           `json:"difficulty"`
		BloomLevel       string           `json:"bloom_level"`
		Options          []incomingOption `json:"options"`
		CorrectOptionIDs []interface{}    `json:"correct_option_ids"`
		Marks            int              `json:"marks"`
		OrderIndex       int              `json:"order_index"`
		IsAIGenerated    bool             `json:"is_ai_generated"`
	}

	var incoming []incomingQuestion

	// Read raw body to try multiple formats
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Try wrapped format first: { "questions": [...] }
	var wrapped struct {
		Questions []incomingQuestion `json:"questions"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.Questions) > 0 {
		incoming = wrapped.Questions
	} else {
		// Try raw array format: [...]
		if err := json.Unmarshal(body, &incoming); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question format: " + err.Error()})
			return
		}
	}

	if len(incoming) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one question is required"})
		return
	}

	// Convert incoming questions to model format
	questions := make([]models.QuizQuestion, 0, len(incoming))
	for i, iq := range incoming {
		q := models.QuizQuestion{
			QuestionType:   models.QuestionType(iq.QuestionType),
			QuestionText:   iq.QuestionText,
			Explanation:    iq.Explanation,
			Marks:          iq.Marks,
			Difficulty:     models.QuestionDifficulty(iq.Difficulty),
			BloomLevel:     models.BloomLevel(iq.BloomLevel),
			OrderIndex:     i,
			IsAIGenerated:  iq.IsAIGenerated,
			SourceModuleID: iq.SourceModuleID,
		}

		if q.Marks == 0 {
			q.Marks = 1
		}

		// Build correct option IDs set for matching
		correctIDs := make(map[interface{}]bool)
		for _, cid := range iq.CorrectOptionIDs {
			correctIDs[cid] = true
		}

		// Convert options
		for j, opt := range iq.Options {
			isCorrect := opt.IsCorrect
			// Also check correct_option_ids array if present
			if len(correctIDs) > 0 {
				isCorrect = correctIDs[opt.ID]
			}
			q.Options = append(q.Options, models.QuizQuestionOption{
				OptionText: opt.OptionText,
				IsCorrect:  isCorrect,
				OrderIndex: j,
			})
		}

		questions = append(questions, q)
	}

	if err := getQuizService().SaveQuestions(facultyRegdNo.(string), uint(quizID), questions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Questions saved successfully"})
}

// EditQuestion edits a single question
func EditQuestion(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	questionID, err := strconv.ParseUint(c.Param("qid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := getQuizService().EditQuestion(facultyRegdNo.(string), uint(quizID), uint(questionID), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Question updated successfully"})
}

// DeleteQuestion deletes a question
func DeleteQuestion(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	questionID, err := strconv.ParseUint(c.Param("qid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question ID"})
		return
	}

	if err := getQuizService().DeleteQuestion(facultyRegdNo.(string), uint(quizID), uint(questionID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Question deleted successfully"})
}

// RegenerateQuestions regenerates quiz questions using AI
func RegenerateQuestions(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	var req GenerateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate faculty ownership
	if err := getQuizService().ValidateFacultyOwnership(facultyRegdNo.(string), req.CourseOfferingID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Get theory modules and generate questions (similar to GenerateAIQuestions)
	// For now, return placeholder
	_ = quizID
	c.JSON(http.StatusOK, gin.H{"message": "Questions regeneration - implementation pending"})
}

// GetQuizAnalytics returns analytics for a quiz
func GetQuizAnalytics(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	analytics, err := getQuizService().GetQuizAnalytics(facultyRegdNo.(string), uint(quizID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Load quiz for total_marks
	var quiz models.Quiz
	if err := database.DB.First(&quiz, quizID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load quiz"})
		return
	}

	// Calculate average time taken
	var avgTimeTaken float64
	database.DB.Model(&models.QuizAttempt{}).
		Where("quiz_id = ? AND status IN (?, ?) AND submitted_at IS NOT NULL",
			quizID, models.AttemptSubmitted, models.AttemptGraded).
		Select("COALESCE(AVG(EXTRACT(EPOCH FROM (submitted_at - started_at)) / 60), 0)").
		Scan(&avgTimeTaken)

	// Build attempt_stats in the format frontend expects
	attemptStats := gin.H{
		"total_attempts":       analytics.TotalAttempts,
		"pass_rate":            analytics.PassRate,
		"average_score":        analytics.AverageScore,
		"average_percentage":   analytics.AveragePercentage,
		"highest_score":        analytics.HighestScore,
		"lowest_score":         analytics.LowestScore,
		"pass_count":           analytics.PassCount,
		"fail_count":           analytics.FailCount,
		"completion_rate":      analytics.CompletionRate,
		"average_time_minutes": math.Round(avgTimeTaken*10) / 10,
	}

	// Build question_stats in the format frontend expects
	questionStats := make([]gin.H, 0, len(analytics.QuestionsAnalytics))
	for _, qa := range analytics.QuestionsAnalytics {
		questionStats = append(questionStats, gin.H{
			"question_id":        qa.QuestionID,
			"question_text":      qa.QuestionText,
			"question_type":      qa.QuestionType,
			"difficulty":         qa.Difficulty,
			"bloom_level":        qa.BloomLevel,
			"marks":              qa.Marks,
			"total_attempts":     qa.TotalAttempts,
			"correct_count":      qa.CorrectCount,
			"incorrect_count":    qa.IncorrectCount,
			"correct_percentage": math.Round(qa.AccuracyRate*10) / 10,
		})
	}

	// Build difficulty_results in the format frontend expects
	difficultyResults := gin.H{
		"easy": gin.H{
			"correct":            analytics.DifficultyResults.Easy.CorrectCount,
			"total":              analytics.DifficultyResults.Easy.TotalAttempts,
			"correct_percentage": math.Round(analytics.DifficultyResults.Easy.AccuracyRate*10) / 10,
			"question_count":     analytics.DifficultyResults.Easy.QuestionCount,
		},
		"medium": gin.H{
			"correct":            analytics.DifficultyResults.Medium.CorrectCount,
			"total":              analytics.DifficultyResults.Medium.TotalAttempts,
			"correct_percentage": math.Round(analytics.DifficultyResults.Medium.AccuracyRate*10) / 10,
			"question_count":     analytics.DifficultyResults.Medium.QuestionCount,
		},
		"hard": gin.H{
			"correct":            analytics.DifficultyResults.Hard.CorrectCount,
			"total":              analytics.DifficultyResults.Hard.TotalAttempts,
			"correct_percentage": math.Round(analytics.DifficultyResults.Hard.AccuracyRate*10) / 10,
			"question_count":     analytics.DifficultyResults.Hard.QuestionCount,
		},
	}

	// Get top performers (top 10 by score)
	var topAttempts []models.QuizAttempt
	database.DB.
		Where("quiz_id = ? AND status IN (?, ?)", quizID, models.AttemptSubmitted, models.AttemptGraded).
		Preload("Student").
		Order("score DESC, percentage DESC").
		Limit(10).
		Find(&topAttempts)

	topPerformers := make([]gin.H, 0, len(topAttempts))
	for _, a := range topAttempts {
		name := a.StudentRegdNo
		if a.Student != nil && a.Student.Name != "" {
			name = a.Student.Name
		}
		topPerformers = append(topPerformers, gin.H{
			"student_name":   name,
			"student_regdno": a.StudentRegdNo,
			"marks_obtained": a.Score,
			"percentage":     a.Percentage,
		})
	}

	// Get all attempts for the attempts tab
	var allAttempts []models.QuizAttempt
	database.DB.
		Where("quiz_id = ? AND status IN (?, ?)", quizID, models.AttemptSubmitted, models.AttemptGraded).
		Preload("Student").
		Order("submitted_at DESC").
		Find(&allAttempts)

	var passingThreshold float64
	if quiz.TotalMarks > 0 {
		passingThreshold = float64(quiz.PassingMarks) / float64(quiz.TotalMarks) * 100
	}

	attemptsResponse := make([]gin.H, 0, len(allAttempts))
	for _, a := range allAttempts {
		name := a.StudentRegdNo
		if a.Student != nil && a.Student.Name != "" {
			name = a.Student.Name
		}
		var timeTakenMinutes float64
		if a.SubmittedAt != nil {
			timeTakenMinutes = math.Round(a.SubmittedAt.Sub(a.StartedAt).Minutes()*10) / 10
		}
		status := "failed"
		if a.Percentage >= passingThreshold {
			status = "passed"
		}
		attemptsResponse = append(attemptsResponse, gin.H{
			"student_name":       name,
			"student_regdno":     a.StudentRegdNo,
			"attempt_number":     a.AttemptNumber,
			"marks_obtained":     a.Score,
			"percentage":         a.Percentage,
			"time_taken_minutes": timeTakenMinutes,
			"status":             status,
			"submitted_at":       a.SubmittedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"quiz": gin.H{
			"id":          quiz.ID,
			"title":       quiz.Title,
			"total_marks": quiz.TotalMarks,
		},
		"attempt_stats":      attemptStats,
		"question_stats":     questionStats,
		"difficulty_results": difficultyResults,
		"top_performers":     topPerformers,
		"attempts":           attemptsResponse,
	})
}

// ListStudentAttempts lists all student attempts for a quiz
func ListStudentAttempts(c *gin.Context) {
	facultyRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	attempts, err := getQuizService().ListStudentAttempts(facultyRegdNo.(string), uint(quizID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Format response
	response := make([]QuizAttemptResponse, len(attempts))
	for i, a := range attempts {
		response[i] = QuizAttemptResponse{
			ID:            a.ID,
			QuizID:        a.QuizID,
			StudentRegdNo: a.StudentRegdNo,
			AttemptNumber: a.AttemptNumber,
			StartedAt:     a.StartedAt,
			SubmittedAt:   a.SubmittedAt,
			Score:         a.Score,
			MaxScore:      a.MaxScore,
			Percentage:    a.Percentage,
			Status:        a.Status,
		}
	}

	c.JSON(http.StatusOK, gin.H{"attempts": response})
}

// =====================================================
// Student Routes
// =====================================================

// StudentQuizListItem extends QuizListResponse with course context for the "all quizzes" view
type StudentQuizListItem struct {
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

// ListAvailableQuizzes lists published quizzes for students.
// If course_offering_id is provided, returns quizzes for that offering only.
// If omitted, returns quizzes across ALL enrolled course offerings.
func ListAvailableQuizzes(c *gin.Context) {
	studentRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	courseOfferingIDStr := c.Query("course_offering_id")

	// If a specific course offering is requested, use the single-offering query
	if courseOfferingIDStr != "" {
		courseOfferingID, err := strconv.ParseUint(courseOfferingIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid course offering ID"})
			return
		}

		quizzes, err := getQuizService().ListAvailableQuizzes(studentRegdNo.(string), uint(courseOfferingID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		response := make([]QuizListResponse, len(quizzes))
		for i, q := range quizzes {
			response[i] = QuizListResponse{
				ID:               q.ID,
				Title:            q.Title,
				Description:      q.Description,
				Status:           q.Status,
				TimeLimitMinutes: q.TimeLimitMinutes,
				TotalMarks:       q.TotalMarks,
				PassingMarks:     q.PassingMarks,
				ScheduledStart:   q.ScheduledStart,
				ScheduledEnd:     q.ScheduledEnd,
			}
		}

		c.JSON(http.StatusOK, gin.H{"quizzes": response})
		return
	}

	// No course_offering_id: return quizzes across all enrolled offerings
	quizItems, err := getQuizService().ListAllAvailableQuizzes(studentRegdNo.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"quizzes": quizItems})
}

// GetQuizForStudent gets a quiz for student to take
func GetQuizForStudent(c *gin.Context) {
	studentRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	quiz, err := getQuizService().GetQuizForStudent(studentRegdNo.(string), uint(quizID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Format for student view (no correct answers)
	questions := make([]StudentQuestionResponse, len(quiz.Questions))
	for i, q := range quiz.Questions {
		options := make([]QuizOptionResponse, len(q.Options))
		for j, o := range q.Options {
			options[j] = QuizOptionResponse{
				ID:         o.ID,
				QuestionID: o.QuestionID,
				OptionText: o.OptionText,
				OrderIndex: o.OrderIndex,
				// IsCorrect is already cleared by service
			}
		}

		questions[i] = StudentQuestionResponse{
			ID:           q.ID,
			QuestionType: q.QuestionType,
			QuestionText: q.QuestionText,
			Marks:        q.Marks,
			Options:      options,
		}
	}

	response := StudentQuizResponse{
		ID:               quiz.ID,
		Title:            quiz.Title,
		Description:      quiz.Description,
		TimeLimitMinutes: quiz.TimeLimitMinutes,
		TotalMarks:       quiz.TotalMarks,
		PassingMarks:     quiz.PassingMarks,
		ShuffleQuestions: quiz.ShuffleQuestions,
		ShuffleOptions:   quiz.ShuffleOptions,
		Questions:        questions,
	}

	c.JSON(http.StatusOK, gin.H{"quiz": response})
}

// StartAttempt starts a new quiz attempt
func StartAttempt(c *gin.Context) {
	studentRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	attempt, err := getQuizService().StartAttempt(studentRegdNo.(string), uint(quizID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"attempt": attempt})
}

// SubmitAttempt submits a quiz attempt
func SubmitAttempt(c *gin.Context) {
	studentRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	var req SubmitQuizAttemptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get active attempt
	attempt, err := getQuizService().GetActiveAttempt(studentRegdNo.(string), uint(quizID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active attempt found"})
		return
	}

	// Convert request format
	answers := make([]services.AnswerSubmission, len(req.Answers))
	for i, a := range req.Answers {
		answers[i] = services.AnswerSubmission{
			QuestionID:        a.QuestionID,
			SelectedOptionIDs: a.SelectedOptionIDs,
			TextAnswer:        a.TextAnswer,
		}
	}

	result, err := getQuizService().SubmitAttempt(studentRegdNo.(string), attempt.ID, answers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"attempt":    result.Attempt,
		"percentage": result.Percentage,
		"passed":     result.Passed,
	})
}

// GetQuizResult gets the result of a completed quiz attempt
func GetQuizResult(c *gin.Context) {
	studentRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	quizID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quiz ID"})
		return
	}

	result, err := getQuizService().GetAttemptResult(studentRegdNo.(string), uint(quizID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Load quiz with questions and options for rich result display
	var quiz models.Quiz
	if err := database.DB.Preload("Questions.Options").First(&quiz, quizID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load quiz details"})
		return
	}

	// Build question map for lookup
	questionMap := make(map[uint]models.QuizQuestion)
	for _, q := range quiz.Questions {
		questionMap[q.ID] = q
	}

	// Build answer map for lookup
	answerMap := make(map[uint]models.QuizAnswer)
	for _, a := range result.Answers {
		answerMap[a.QuestionID] = a
	}

	// Calculate time taken in minutes
	var timeTakenMinutes float64
	if result.Attempt.SubmittedAt != nil {
		timeTakenMinutes = result.Attempt.SubmittedAt.Sub(result.Attempt.StartedAt).Minutes()
	}

	// Build question_results with full details
	questionResults := make([]gin.H, 0, len(quiz.Questions))
	for _, q := range quiz.Questions {
		answer, hasAnswer := answerMap[q.ID]

		qResult := gin.H{
			"question_id":   q.ID,
			"question_text": q.QuestionText,
			"question_type": q.QuestionType,
			"explanation":   q.Explanation,
			"marks":         q.Marks,
			"is_correct":    false,
			"marks_awarded": 0,
		}

		if hasAnswer {
			qResult["is_correct"] = answer.IsCorrect
			qResult["marks_awarded"] = answer.MarksAwarded
			qResult["user_selected_option_ids"] = answer.SelectedOptionIDs
			qResult["user_text_answer"] = answer.TextAnswer
		}

		// Add options with is_correct for MCQ/multi_select/true_false
		if q.QuestionType == models.QuestionTypeMCQ || q.QuestionType == models.QuestionTypeMultiSelect || q.QuestionType == models.QuestionTypeTrueFalse {
			options := make([]gin.H, 0, len(q.Options))
			for _, opt := range q.Options {
				options = append(options, gin.H{
					"id":          opt.ID,
					"option_text": opt.OptionText,
					"is_correct":  opt.IsCorrect,
					"order_index": opt.OrderIndex,
				})
			}
			qResult["options"] = options
		}

		// For text-based questions, provide expected answer
		if q.QuestionType == models.QuestionTypeShortAnswer || q.QuestionType == models.QuestionTypeFillBlank {
			for _, opt := range q.Options {
				if opt.IsCorrect {
					qResult["expected_answer"] = opt.OptionText
					break
				}
			}
		}

		questionResults = append(questionResults, qResult)
	}

	// Build attempt response
	attemptResponse := gin.H{
		"id":                 result.Attempt.ID,
		"quiz_id":            result.Attempt.QuizID,
		"attempt_number":     result.Attempt.AttemptNumber,
		"started_at":         result.Attempt.StartedAt,
		"submitted_at":       result.Attempt.SubmittedAt,
		"marks_obtained":     result.Attempt.Score,
		"total_marks":        quiz.TotalMarks,
		"percentage":         result.Attempt.Percentage,
		"status":             result.Attempt.Status,
		"time_taken_minutes": timeTakenMinutes,
	}

	// Determine pass/fail status for frontend
	if result.Passed {
		attemptResponse["status"] = "passed"
	} else {
		attemptResponse["status"] = "failed"
	}

	// Build quiz info
	quizResponse := gin.H{
		"id":            quiz.ID,
		"title":         quiz.Title,
		"total_marks":   quiz.TotalMarks,
		"passing_marks": quiz.PassingMarks,
		"max_attempts":  quiz.MaxAttempts,
	}

	c.JSON(http.StatusOK, gin.H{
		"attempt":          attemptResponse,
		"quiz":             quizResponse,
		"question_results": questionResults,
	})
}

// =====================================================
// Helper Functions
// =====================================================

func formatOfferingDisplayName(o models.CourseOffering) string {
	parts := []string{}

	if o.Course != nil && o.Course.CourseName != "" {
		parts = append(parts, o.Course.CourseName)
	}

	if o.Section != nil {
		if o.Section.SectionName != "" {
			parts = append(parts, o.Section.SectionName)
		}
		if o.Section.CohortYear > 0 {
			parts[len(parts)-1] = parts[len(parts)-1] + ", Batch " + strconv.Itoa(o.Section.CohortYear)
		}
	}

	return strings.Join(parts, " - ")
}

func formatQuizDetail(quiz *models.Quiz) QuizDetailResponse {
	questions := make([]QuizQuestionResponse, len(quiz.Questions))
	for i, q := range quiz.Questions {
		options := make([]QuizOptionResponse, len(q.Options))
		for j, o := range q.Options {
			options[j] = QuizOptionResponse{
				ID:         o.ID,
				QuestionID: o.QuestionID,
				OptionText: o.OptionText,
				IsCorrect:  o.IsCorrect,
				OrderIndex: o.OrderIndex,
			}
		}

		questions[i] = QuizQuestionResponse{
			ID:             q.ID,
			QuizID:         q.QuizID,
			QuestionType:   q.QuestionType,
			QuestionText:   q.QuestionText,
			Explanation:    q.Explanation,
			Marks:          q.Marks,
			Difficulty:     q.Difficulty,
			OrderIndex:     q.OrderIndex,
			SourceModuleID: q.SourceModuleID,
			BloomLevel:     q.BloomLevel,
			IsAIGenerated:  q.IsAIGenerated,
			Options:        options,
		}
	}

	theorySources := make([]TheorySourceResponse, len(quiz.TheorySources))
	for i, ts := range quiz.TheorySources {
		theorySources[i] = TheorySourceResponse{
			ID:             ts.ID,
			QuizID:         ts.QuizID,
			TheoryModuleID: ts.TheoryModuleID,
			TheoryWeekID:   ts.TheoryWeekID,
		}
	}

	return QuizDetailResponse{
		ID:               quiz.ID,
		CourseOfferingID: quiz.CourseOfferingID,
		Title:            quiz.Title,
		Description:      quiz.Description,
		QuizType:         quiz.QuizType,
		Status:           quiz.Status,
		TimeLimitMinutes: quiz.TimeLimitMinutes,
		TotalMarks:       quiz.TotalMarks,
		PassingMarks:     quiz.PassingMarks,
		MaxAttempts:      quiz.MaxAttempts,
		ShuffleQuestions: quiz.ShuffleQuestions,
		ShuffleOptions:   quiz.ShuffleOptions,
		ShowResultsAfter: quiz.ShowResultsAfter,
		ScheduledStart:   quiz.ScheduledStart,
		ScheduledEnd:     quiz.ScheduledEnd,
		AIPrompt:         quiz.AIPrompt,
		AIModelUsed:      quiz.AIModelUsed,
		CreatedAt:        quiz.CreatedAt,
		UpdatedAt:        quiz.UpdatedAt,
		Questions:        questions,
		TheorySources:    theorySources,
	}
}
