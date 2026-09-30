package models

import (
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// =====================================================
// Practice Quiz System Models
// =====================================================
// Embedded practice quizzes within theory course content
// Separate from formal quiz system (quiz.go)
// Practice quizzes are for learning/self-assessment
// =====================================================

// PracticeQuiz - Embedded practice quiz at end of a week's modules
type PracticeQuiz struct {
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	TheoryWeek   *TheoryWeek            `gorm:"foreignKey:TheoryWeekID;constraint:OnDelete:CASCADE" json:"theory_week,omitempty"`
	DeletedAt    gorm.DeletedAt         `gorm:"index" json:"-"`
	Title        string                 `gorm:"size:300;not null" json:"title"`
	Instructions string                 `gorm:"type:text" json:"instructions"`
	Questions    []PracticeQuizQuestion `gorm:"foreignKey:PracticeQuizID" json:"questions,omitempty"`
	ID           uint                   `gorm:"primaryKey;autoIncrement" json:"id"`
	TheoryWeekID uint                   `gorm:"not null;index:idx_theory_week" json:"theory_week_id"`
	TotalMarks   int                    `gorm:"default:0" json:"total_marks"`
	PassingMarks int                    `gorm:"default:0" json:"passing_marks"`
}

// PracticeQuizQuestion - Individual practice questions
type PracticeQuizQuestion struct {
	UpdatedAt      time.Time            `json:"updated_at"`
	CreatedAt      time.Time            `json:"created_at"`
	SourceModuleID *uint                `gorm:"index:idx_source_module" json:"source_module_id,omitempty"`
	SourceModule   *TheoryModule        `gorm:"foreignKey:ID;references:SourceModuleID;constraint:OnDelete:SET NULL" json:"source_module,omitempty"`
	PracticeQuiz   *PracticeQuiz        `gorm:"foreignKey:PracticeQuizID;constraint:OnDelete:CASCADE" json:"practice_quiz,omitempty"`
	QuestionText   string               `gorm:"type:text;not null" json:"question_text"`
	Difficulty     QuestionDifficulty   `gorm:"type:varchar(10);not null" json:"difficulty"`
	Explanation    string               `gorm:"type:text" json:"explanation"`
	QuestionType   QuestionType         `gorm:"type:varchar(20);not null" json:"question_type"`
	Options        []PracticeQuizOption `gorm:"foreignKey:QuestionID" json:"options,omitempty"`
	OrderIndex     int                  `gorm:"default:0" json:"order_index"`
	Marks          int                  `gorm:"default:1" json:"marks"`
	ID             uint                 `gorm:"primaryKey;autoIncrement" json:"id"`
	PracticeQuizID uint                 `gorm:"not null;index:idx_practice_quiz" json:"practice_quiz_id"`
}

// PracticeQuizOption - MCQ/TF options
type PracticeQuizOption struct {
	CreatedAt  time.Time             `json:"created_at"`
	Question   *PracticeQuizQuestion `gorm:"foreignKey:QuestionID;constraint:OnDelete:CASCADE" json:"question,omitempty"`
	OptionText string                `gorm:"type:text;not null" json:"option_text"`
	ID         uint                  `gorm:"primaryKey;autoIncrement" json:"id"`
	QuestionID uint                  `gorm:"not null;index:idx_question_id" json:"question_id"`
	OrderIndex int                   `gorm:"default:0" json:"order_index"`
	IsCorrect  bool                  `gorm:"default:false" json:"is_correct"`
}

// =====================================================
// Generation Job Models
// =====================================================
// Tracks async AI generation jobs for lesson plan processing
// =====================================================

// GenerationJobStatus represents the status of a generation job
type GenerationJobStatus string

const (
	GenerationJobPending    GenerationJobStatus = "pending"
	GenerationJobProcessing GenerationJobStatus = "processing"
	GenerationJobCompleted  GenerationJobStatus = "completed"
	GenerationJobFailed     GenerationJobStatus = "failed"
)

// GenerationJobType represents the type of generation job
type GenerationJobType string

const (
	GenerationJobLessonPlanParse GenerationJobType = "lesson_plan_parse"
	GenerationJobQuizGenerate    GenerationJobType = "quiz_generate"
	GenerationJobContestGenerate GenerationJobType = "contest_generate"
)

// GenerationJob - Tracks async AI generation jobs
type GenerationJob struct {
	CreatedAt     time.Time           `json:"created_at"`
	TheoryID      *uint               `gorm:"index:idx_theory" json:"theory_id,omitempty"`
	CourseID      *uint               `gorm:"index:idx_course" json:"course_id,omitempty"`
	CompletedAt   *time.Time          `json:"completed_at,omitempty"`
	Admin         *User               `gorm:"foreignKey:AdminRegdNo;references:RegdNo;constraint:OnDelete:CASCADE" json:"admin,omitempty"`
	Course        *Course             `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE" json:"course,omitempty"`
	Theory        *Theory             `gorm:"foreignKey:TheoryID;constraint:OnDelete:CASCADE" json:"theory,omitempty"`
	JobType       GenerationJobType   `gorm:"type:varchar(30);not null" json:"job_type"`
	Status        GenerationJobStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	InputFilePath string              `gorm:"size:500" json:"input_file_path"`
	ErrorMessage  string              `gorm:"type:text" json:"error_message,omitempty"`
	AdminRegdNo   string              `gorm:"size:50;not null;index:idx_admin" json:"admin_regdno"`
	GeneratedData datatypes.JSON      `gorm:"type:jsonb" json:"generated_data,omitempty"`
	ID            uint                `gorm:"primaryKey;autoIncrement" json:"id"`
}

// LessonPlanStructure - AI-parsed lesson plan structure (full content)
type LessonPlanStructure struct {
	CourseName string     `json:"course_name"`
	Weeks      []WeekData `json:"weeks"`
	TotalWeeks int        `json:"total_weeks"`
}

// LessonPlanStructureLite - Phase 1 extracted structure (lightweight, no content)
type LessonPlanStructureLite struct {
	CourseName string         `json:"course_name"`
	Weeks      []WeekDataLite `json:"weeks"`
	TotalWeeks int            `json:"total_weeks"`
}

// WeekDataLite - Phase 1 week data (names only, no content)
type WeekDataLite struct {
	WeekName    string   `json:"week_name"`
	DateRange   string   `json:"date_range,omitempty"`
	ModuleNames []string `json:"module_names"`
	WeekNumber  int      `json:"week_number"`
}

// WeekData - Data for a single week
type WeekData struct {
	WeekName   string       `json:"week_name"`
	Modules    []ModuleData `json:"modules"`
	WeekNumber int          `json:"week_number"`
}

// ModuleData - Data for a single module
type ModuleData struct {
	ModuleName  string `json:"module_name"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

// ConvertLiteToFull converts lite structure to full structure with generated content
func (lite *LessonPlanStructureLite) ConvertLiteToFull(modulesContent map[string]ModuleData) *LessonPlanStructure {
	full := &LessonPlanStructure{
		CourseName: lite.CourseName,
		TotalWeeks: lite.TotalWeeks,
		Weeks:      make([]WeekData, len(lite.Weeks)),
	}

	for i, weekLite := range lite.Weeks {
		full.Weeks[i] = WeekData{
			WeekNumber: weekLite.WeekNumber,
			WeekName:   weekLite.WeekName,
			Modules:    make([]ModuleData, len(weekLite.ModuleNames)),
		}
		for j, moduleName := range weekLite.ModuleNames {
			if content, ok := modulesContent[moduleName]; ok {
				full.Weeks[i].Modules[j] = content
			} else {
				full.Weeks[i].Modules[j] = ModuleData{
					ModuleName:  moduleName,
					Description: fmt.Sprintf("Content for %s", moduleName),
					Content:     "",
				}
			}
		}
	}
	return full
}

// PracticeQuizData - AI-generated practice quiz data
type PracticeQuizData struct {
	Title        string                     `json:"title"`
	Instructions string                     `json:"instructions"`
	Questions    []PracticeQuizQuestionData `json:"questions"`
}

// PracticeQuizQuestionData - AI-generated practice question data
type PracticeQuizQuestionData struct {
	QuestionText     string               `json:"question_text"`
	QuestionType     string               `json:"question_type"`
	Explanation      string               `json:"explanation"`
	Difficulty       string               `json:"difficulty"`
	SourceModuleName string               `json:"source_module_name"`
	Options          []PracticeOptionData `json:"options"`
}

// PracticeOptionData - AI-generated practice option data
type PracticeOptionData struct {
	OptionText string `json:"option_text"`
	IsCorrect  bool   `json:"is_correct"`
	OrderIndex int    `json:"order_index"`
}

// =====================================================
// Helper Methods
// =====================================================

// TableName specifies custom table names for practice quiz models
func (PracticeQuiz) TableName() string {
	return "practice_quizzes"
}

func (PracticeQuizQuestion) TableName() string {
	return "practice_quiz_questions"
}

func (PracticeQuizOption) TableName() string {
	return "practice_quiz_options"
}

func (GenerationJob) TableName() string {
	return "generation_jobs"
}
