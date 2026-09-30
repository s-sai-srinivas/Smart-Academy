package models

import (
	"time"

	"gorm.io/gorm"
)

// =====================================================
// Contest Generation Request/Response Models
// =====================================================

// ContestGenerationRequest contains parameters for AI contest problem generation
type ContestGenerationRequest struct {
	CourseID           *uint    `json:"course_id,omitempty"`
	ContestID          *uint    `json:"contest_id,omitempty"`
	StartTime          *string  `json:"start_time,omitempty"`
	EndTime            *string  `json:"end_time,omitempty"`
	DifficultyLevel    string   `json:"difficulty_level" binding:"omitempty,oneof=easy medium hard"`
	Topics             []string `json:"topics" binding:"required,min=1"`
	RequestedCount     int      `json:"requested_count" binding:"required,min=1,max=10"`
	VersionsPerProblem int      `json:"versions_per_problem,omitempty"`
}

// ContestGenerationResponse contains the response for a generation request
type ContestGenerationResponse struct {
	Status        GenerationJobStatus `json:"status"`
	Message       string              `json:"message"`
	EstimatedTime string              `json:"estimated_time,omitempty"`
	JobID         uint                `json:"job_id"`
}

// =====================================================
// Generated Problem Models (AI Output)
// =====================================================

// GeneratedProblem represents a single AI-generated contest problem
type GeneratedProblem struct {
	Constraints        map[string]interface{} `json:"constraints"`
	SampleOutput       string                 `json:"sample_output"`
	Difficulty         string                 `json:"difficulty"`
	OutputFormat       string                 `json:"output_format"`
	Description        string                 `json:"description"`
	SampleInput        string                 `json:"sample_input"`
	Title              string                 `json:"title"`
	SampleExplanation  string                 `json:"sample_explanation,omitempty"`
	InputFormat        string                 `json:"input_format"`
	VerificationStatus string                 `json:"verification_status,omitempty"`
	SolutionApproach   string                 `json:"solution_approach,omitempty"`
	ReferenceSolution  string                 `json:"reference_solution,omitempty"`
	SolutionLanguage   string                 `json:"solution_language,omitempty"`
	TestCases          []GeneratedTestCase    `json:"test_cases"`
	TopicsCovered      []string               `json:"topics_covered"`
}

// GeneratedTestCase represents a single test case for a generated problem
type GeneratedTestCase struct {
	Category        string `json:"category"` // sample, edge, boundary, large, random
	Input           string `json:"input"`
	ExpectedOutput  string `json:"expected_output"`
	Description     string `json:"description,omitempty"`
	Points          int    `json:"points,omitempty"`
	IsSampleVisible bool   `json:"is_sample_visible"`
	TimeLimitMS     int    `json:"time_limit_ms,omitempty"`
	MemoryLimitKB   int    `json:"memory_limit_kb,omitempty"`
}

// GeneratedProblemReview is used for faculty review before approval
type GeneratedProblemReview struct {
	Title              string   `json:"title"`
	Difficulty         string   `json:"difficulty"`
	VerificationStatus string   `json:"verification_status"`
	RejectionReason    string   `json:"rejection_reason,omitempty"`
	TopicsCovered      []string `json:"topics_covered"`
	ProblemID          uint     `json:"problem_id"`
	TestCasesCount     int      `json:"test_cases_count"`
	IsApproved         bool     `json:"is_approved"`
	IsRejected         bool     `json:"is_rejected"`
}

// =====================================================
// Contest Editorial Models
// =====================================================

// ContestEditorial stores editorial hints for contest problems (post-contest)
type ContestEditorial struct {
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
	Hint1           string         `gorm:"column:hint_1;type:text;not null" json:"hint_1"`
	Hint2           string         `gorm:"column:hint_2;type:text" json:"hint_2"`
	Hint3           string         `gorm:"column:hint_3;type:text" json:"hint_3"`
	CoreIdea        string         `gorm:"column:core_idea;type:text;not null" json:"core_idea"`
	TimeComplexity  string         `gorm:"column:time_complexity;size:100" json:"time_complexity"`
	SpaceComplexity string         `gorm:"column:space_complexity;size:100" json:"space_complexity"`
	ID              uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID       uint           `gorm:"not null;index:idx_contest_editorial_contest" json:"contest_id"`
	ProblemID       uint           `gorm:"not null;index:idx_contest_editorial_problem" json:"problem_id"`
}

// EditorialRequest contains parameters for generating editorials
type EditorialRequest struct {
	ProblemDescription string `json:"problem_description" binding:"required"`
	ProblemConstraints string `json:"problem_constraints"`
	SampleInput        string `json:"sample_input"`
	SampleOutput       string `json:"sample_output"`
	ContestID          uint   `json:"contest_id" binding:"required"`
	ProblemID          uint   `json:"problem_id" binding:"required"`
}

// EditorialResponse contains generated editorial hints
type EditorialResponse struct {
	Hint1           string `json:"hint_1"`
	Hint2           string `json:"hint_2"`
	Hint3           string `json:"hint_3"`
	CoreIdea        string `json:"core_idea"`
	TimeComplexity  string `json:"time_complexity"`
	SpaceComplexity string `json:"space_complexity"`
}

// =====================================================
// Plagiarism Detection Models (Phase 2)
// =====================================================

// PlagiarismReport stores plagiarism detection results for a contest
type PlagiarismReport struct {
	GeneratedAt time.Time `json:"generated_at"`
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `gorm:"size:20;default:'pending'" json:"status"`
	ReportData  string    `gorm:"type:jsonb" json:"report_data,omitempty"`
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID   uint      `gorm:"not null;index:idx_plagiarism_report_contest" json:"contest_id"`
}

// PlagiarismCase represents a detected plagiarism case between two submissions
type PlagiarismCase struct {
	CreatedAt           time.Time `json:"created_at"`
	MatchedCodeSections string    `gorm:"type:jsonb" json:"matched_code_sections,omitempty"`
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ReportID            uint      `gorm:"not null;index:idx_plagiarism_case_report" json:"report_id"`
	Submission1ID       uint      `gorm:"not null" json:"submission_1_id"`
	Submission2ID       uint      `gorm:"not null" json:"submission_2_id"`
	SimilarityScore     float64   `gorm:"type:decimal(5,2);not null" json:"similarity_score"`
}

// PlagiarismDetectionRequest triggers plagiarism detection for a contest
type PlagiarismDetectionRequest struct {
	ContestID uint    `json:"contest_id" binding:"required"`
	Threshold float64 `json:"threshold,omitempty"` // Similarity threshold (default: 70%)
}

// =====================================================
// Helper Methods
// =====================================================

// TableName specifies custom table name for contest editorials
func (ContestEditorial) TableName() string {
	return "contest_editorials"
}

// TableName specifies custom table name for plagiarism reports
func (PlagiarismReport) TableName() string {
	return "plagiarism_reports"
}

// TableName specifies custom table name for plagiarism cases
func (PlagiarismCase) TableName() string {
	return "plagiarism_cases"
}
