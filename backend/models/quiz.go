package models

import (
	"time"

	"gorm.io/gorm"
)

// =====================================================
// Quiz System Models
// =====================================================
// Scalable quiz generation and management system scoped to
// College → Branch → Section (CohortYear=batch) → CourseOffering
// =====================================================

// QuizType represents the type of quiz creation
type QuizType string

const (
	QuizTypeAIGenerated QuizType = "ai_generated"
	QuizTypeManual      QuizType = "manual"
	QuizTypeHybrid      QuizType = "hybrid"
)

// QuizStatus represents the lifecycle status of a quiz
type QuizStatus string

const (
	QuizStatusDraft     QuizStatus = "draft"
	QuizStatusPublished QuizStatus = "published"
	QuizStatusClosed    QuizStatus = "closed"
	QuizStatusArchived  QuizStatus = "archived"
)

// ShowResultsAfter determines when students can see results
type ShowResultsAfter string

const (
	ShowResultsImmediately ShowResultsAfter = "immediately"
	ShowResultsAfterClose  ShowResultsAfter = "after_close"
	ShowResultsNever       ShowResultsAfter = "never"
)

// QuestionType represents the type of quiz question
type QuestionType string

const (
	QuestionTypeMCQ         QuestionType = "mcq"
	QuestionTypeMultiSelect QuestionType = "multi_select"
	QuestionTypeTrueFalse   QuestionType = "true_false"
	QuestionTypeShortAnswer QuestionType = "short_answer"
	QuestionTypeFillBlank   QuestionType = "fill_blank"
)

// QuestionDifficulty represents the difficulty level of a question
type QuestionDifficulty string

const (
	DifficultyEasy   QuestionDifficulty = "easy"
	DifficultyMedium QuestionDifficulty = "medium"
	DifficultyHard   QuestionDifficulty = "hard"
)

// BloomLevel represents Bloom's taxonomy cognitive levels
type BloomLevel string

const (
	BloomRemember   BloomLevel = "remember"
	BloomUnderstand BloomLevel = "understand"
	BloomApply      BloomLevel = "apply"
	BloomAnalyze    BloomLevel = "analyze"
	BloomEvaluate   BloomLevel = "evaluate"
	BloomCreate     BloomLevel = "create"
)

// AttemptStatus represents the status of a quiz attempt
type AttemptStatus string

const (
	AttemptInProgress AttemptStatus = "in_progress"
	AttemptSubmitted  AttemptStatus = "submitted"
	AttemptGraded     AttemptStatus = "graded"
	AttemptTimedOut   AttemptStatus = "timed_out"
)

// =====================================================
// Quiz - Main quiz metadata, scoped to a CourseOffering
// =====================================================
type Quiz struct {
	UpdatedAt        time.Time          `json:"updated_at"`
	CreatedAt        time.Time          `json:"created_at"`
	ScheduledStart   *time.Time         `gorm:"index:idx_schedule" json:"scheduled_start"`
	Creator          *User              `gorm:"foreignKey:CreatedBy;references:RegdNo;constraint:OnDelete:RESTRICT" json:"creator,omitempty"`
	CourseOffering   *CourseOffering    `gorm:"foreignKey:CourseOfferingID;constraint:OnDelete:CASCADE" json:"course_offering,omitempty"`
	ScheduledEnd     *time.Time         `gorm:"index:idx_schedule" json:"scheduled_end"`
	DeletedAt        gorm.DeletedAt     `gorm:"index" json:"-"`
	ShowResultsAfter ShowResultsAfter   `gorm:"type:varchar(20);default:'immediately'" json:"show_results_after"`
	Description      string             `gorm:"type:text" json:"description"`
	Title            string             `gorm:"size:300;not null" json:"title"`
	QuizType         QuizType           `gorm:"type:varchar(30);not null;default:'ai_generated'" json:"quiz_type"`
	Status           QuizStatus         `gorm:"type:varchar(20);not null;default:'draft'" json:"status"`
	AIModelUsed      string             `gorm:"size:50" json:"ai_model_used"`
	AIPrompt         string             `gorm:"type:text" json:"ai_prompt"`
	CreatedBy        string             `gorm:"size:50;not null;index:idx_created_by" json:"created_by"`
	Questions        []QuizQuestion     `gorm:"foreignKey:QuizID" json:"questions,omitempty"`
	Attempts         []QuizAttempt      `gorm:"foreignKey:QuizID" json:"attempts,omitempty"`
	TheorySources    []QuizTheorySource `gorm:"foreignKey:QuizID" json:"theory_sources,omitempty"`
	TimeLimitMinutes int                `gorm:"default:0" json:"time_limit_minutes"`
	TotalMarks       int                `gorm:"default:0" json:"total_marks"`
	ID               uint               `gorm:"primaryKey;autoIncrement" json:"id"`
	PassingMarks     int                `gorm:"default:0" json:"passing_marks"`
	MaxAttempts      int                `gorm:"default:1" json:"max_attempts"`
	CourseOfferingID uint               `gorm:"not null;index:idx_course_offering" json:"course_offering_id"`
	ShuffleOptions   bool               `gorm:"default:false" json:"shuffle_options"`
	ShuffleQuestions bool               `gorm:"default:false" json:"shuffle_questions"`
}

// =====================================================
// QuizTheorySource - Tracks which theory modules were used as AI context
// =====================================================
type QuizTheorySource struct {
	Quiz           *Quiz         `gorm:"foreignKey:QuizID;constraint:OnDelete:CASCADE" json:"quiz,omitempty"`
	TheoryModule   *TheoryModule `gorm:"foreignKey:ID;references:TheoryModuleID" json:"theory_module,omitempty"`
	TheoryWeek     *TheoryWeek   `gorm:"foreignKey:ID;references:TheoryWeekID" json:"theory_week,omitempty"`
	ID             uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	QuizID         uint          `gorm:"not null;index:idx_quiz_id;uniqueIndex:idx_quiz_module" json:"quiz_id"`
	TheoryModuleID uint          `gorm:"not null;index:idx_module_id;uniqueIndex:idx_quiz_module" json:"theory_module_id"`
	TheoryWeekID   uint          `gorm:"not null;index:idx_week_id" json:"theory_week_id"`
}

// =====================================================
// QuizQuestion - Individual questions within a quiz
// =====================================================
type QuizQuestion struct {
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
	SourceModuleID *uint                `gorm:"index:idx_source_module" json:"source_module_id,omitempty"`
	Quiz           *Quiz                `gorm:"foreignKey:QuizID;constraint:OnDelete:CASCADE" json:"quiz,omitempty"`
	SourceModule   *TheoryModule        `gorm:"foreignKey:ID;references:SourceModuleID;constraint:OnDelete:SET NULL" json:"source_module,omitempty"`
	BloomLevel     BloomLevel           `gorm:"type:varchar(20)" json:"bloom_level"`
	Difficulty     QuestionDifficulty   `gorm:"type:varchar(10);not null" json:"difficulty"`
	QuestionText   string               `gorm:"type:text;not null" json:"question_text"`
	QuestionType   QuestionType         `gorm:"type:varchar(20);not null" json:"question_type"`
	Explanation    string               `gorm:"type:text" json:"explanation"`
	Options        []QuizQuestionOption `gorm:"foreignKey:QuestionID" json:"options,omitempty"`
	OrderIndex     int                  `gorm:"default:0" json:"order_index"`
	ID             uint                 `gorm:"primaryKey;autoIncrement" json:"id"`
	QuizID         uint                 `gorm:"not null;index:idx_quiz_id" json:"quiz_id"`
	Marks          int                  `gorm:"default:1" json:"marks"`
	IsAIGenerated  bool                 `gorm:"default:true" json:"is_ai_generated"`
}

// =====================================================
// QuizQuestionOption - MCQ / multi-select options
// =====================================================
type QuizQuestionOption struct {
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	Question   *QuizQuestion `gorm:"foreignKey:QuestionID;constraint:OnDelete:CASCADE" json:"question,omitempty"`
	OptionText string        `gorm:"type:text;not null" json:"option_text"`
	ID         uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	QuestionID uint          `gorm:"not null;index:idx_question_id" json:"question_id"`
	OrderIndex int           `gorm:"default:0" json:"order_index"`
	IsCorrect  bool          `gorm:"default:false;index:idx_is_correct" json:"is_correct"`
}

// =====================================================
// QuizAttempt - Student attempt tracking
// =====================================================
type QuizAttempt struct {
	StartedAt     time.Time     `gorm:"not null" json:"started_at"`
	Student       *User         `gorm:"foreignKey:StudentRegdNo;references:RegdNo;constraint:OnDelete:CASCADE" json:"student,omitempty"`
	DeadlineAt    *time.Time    `gorm:"index:idx_deadline" json:"deadline_at"`
	SubmittedAt   *time.Time    `gorm:"index:idx_submitted" json:"submitted_at"`
	Quiz          *Quiz         `gorm:"foreignKey:QuizID;constraint:OnDelete:CASCADE" json:"quiz,omitempty"`
	Status        AttemptStatus `gorm:"type:varchar(20);not null;default:'in_progress'" json:"status"`
	StudentRegdNo string        `gorm:"column:student_regdno;size:50;not null;index:idx_student" json:"student_regdno"`
	Answers       []QuizAnswer  `gorm:"foreignKey:AttemptID" json:"answers,omitempty"`
	AttemptNumber int           `gorm:"not null" json:"attempt_number"`
	Percentage    float64       `gorm:"type:decimal(5,2);default:0" json:"percentage"`
	MaxScore      int           `gorm:"default:0" json:"max_score"`
	Score         int           `gorm:"default:0" json:"score"`
	ID            uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	QuizID        uint          `gorm:"not null;index:idx_quiz_id" json:"quiz_id"`
}

// =====================================================
// QuizAnswer - Per-question responses
// =====================================================
type QuizAnswer struct {
	Attempt           *QuizAttempt  `gorm:"foreignKey:AttemptID;constraint:OnDelete:CASCADE" json:"attempt,omitempty"`
	Question          *QuizQuestion `gorm:"foreignKey:QuestionID;constraint:OnDelete:CASCADE" json:"question,omitempty"`
	TextAnswer        string        `gorm:"type:text" json:"text_answer"`
	SelectedOptionIDs []uint        `gorm:"type:jsonb" json:"selected_option_ids"`
	ID                uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	AttemptID         uint          `gorm:"not null;index:idx_attempt_id" json:"attempt_id"`
	QuestionID        uint          `gorm:"not null;index:idx_question_id" json:"question_id"`
	MarksAwarded      int           `gorm:"default:0" json:"marks_awarded"`
	IsCorrect         bool          `gorm:"default:false;index:idx_is_correct" json:"is_correct"`
}

// =====================================================
// Helper Methods
// =====================================================

// TableName specifies custom table names for quiz models
func (Quiz) TableName() string {
	return "quizzes"
}

func (QuizTheorySource) TableName() string {
	return "quiz_theory_sources"
}

func (QuizQuestion) TableName() string {
	return "quiz_questions"
}

func (QuizQuestionOption) TableName() string {
	return "quiz_question_options"
}

func (QuizAttempt) TableName() string {
	return "quiz_attempts"
}

func (QuizAnswer) TableName() string {
	return "quiz_answers"
}
