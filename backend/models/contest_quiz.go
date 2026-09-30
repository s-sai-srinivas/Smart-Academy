package models

import (
	"time"
)

// ContestQuiz represents an exam-style quiz embedded within a contest section
// Separate from faculty course quizzes (quiz.go) to prevent mixing
type ContestQuiz struct {
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	Section         *ContestSection       `gorm:"foreignKey:SectionID;constraint:OnDelete:CASCADE" json:"section,omitempty"`
	SectionID       *uint                 `gorm:"uniqueIndex" json:"section_id,omitempty"`
	Contest         *Contest              `gorm:"foreignKey:ContestID;references:ContestID;constraint:OnDelete:CASCADE" json:"contest,omitempty"`
	Title           string                `gorm:"size:300;not null" json:"title"`
	Instructions    string                `gorm:"type:text" json:"instructions"`
	Questions       []ContestQuizQuestion `gorm:"foreignKey:ContestQuizID" json:"questions,omitempty"`
	PassingMarks    int                   `gorm:"default:0" json:"passing_marks"`
	DurationMinutes int                   `gorm:"default:0" json:"duration_minutes"`
	TotalMarks      int                   `gorm:"default:0" json:"total_marks"`
	ID              uint                  `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID       uint                  `gorm:"index;not null" json:"contest_id"`
}

func (ContestQuiz) TableName() string { return "contest_quizzes" }

// ContestQuizQuestion stores individual questions in a contest quiz
type ContestQuizQuestion struct {
	CreatedAt     time.Time           `json:"created_at"`
	ContestQuiz   *ContestQuiz        `gorm:"foreignKey:ContestQuizID;constraint:OnDelete:CASCADE" json:"contest_quiz,omitempty"`
	QuestionType  string              `gorm:"size:20;not null" json:"question_type"`
	QuestionText  string              `gorm:"type:text;not null" json:"question_text"`
	Explanation   string              `gorm:"type:text" json:"explanation"`
	Options       []ContestQuizOption `gorm:"foreignKey:QuestionID" json:"options,omitempty"`
	ID            uint                `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestQuizID uint                `gorm:"not null;index" json:"contest_quiz_id"`
	Marks         int                 `gorm:"default:1" json:"marks"`
	OrderIndex    int                 `gorm:"default:0" json:"order_index"`
}

func (ContestQuizQuestion) TableName() string { return "contest_quiz_questions" }

// ContestQuizOption stores answer options for MCQ/TF questions
type ContestQuizOption struct {
	CreatedAt  time.Time            `json:"created_at"`
	Question   *ContestQuizQuestion `gorm:"foreignKey:QuestionID;constraint:OnDelete:CASCADE" json:"question,omitempty"`
	OptionText string               `gorm:"type:text;not null" json:"option_text"`
	ID         uint                 `gorm:"primaryKey;autoIncrement" json:"id"`
	QuestionID uint                 `gorm:"not null;index" json:"question_id"`
	OrderIndex int                  `gorm:"default:0" json:"order_index"`
	IsCorrect  bool                 `gorm:"default:false" json:"is_correct"`
}

func (ContestQuizOption) TableName() string { return "contest_quiz_options" }

// ContestQuizAttempt tracks a student's quiz attempt in a contest
type ContestQuizAttempt struct {
	CreatedAt     time.Time           `json:"created_at"`
	StartedAt     time.Time           `gorm:"not null" json:"started_at"`
	User          *User               `gorm:"foreignKey:UserRegdNo;references:RegdNo;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	ContestQuiz   *ContestQuiz        `gorm:"foreignKey:ContestQuizID;constraint:OnDelete:CASCADE" json:"contest_quiz,omitempty"`
	Contest       *Contest            `gorm:"foreignKey:ContestID;references:ContestID;constraint:OnDelete:CASCADE" json:"contest,omitempty"`
	SubmittedAt   *time.Time          `json:"submitted_at,omitempty"`
	UserRegdNo    string              `gorm:"size:50;not null;index" json:"user_regdno"`
	Answers       []ContestQuizAnswer `gorm:"foreignKey:AttemptID" json:"answers,omitempty"`
	MaxScore      int                 `gorm:"default:0" json:"max_score"`
	TotalScore    int                 `gorm:"default:0" json:"total_score"`
	ID            uint                `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID     uint                `gorm:"not null;index" json:"contest_id"`
	ContestQuizID uint                `gorm:"not null;index" json:"contest_quiz_id"`
	Passed        bool                `gorm:"default:false" json:"passed"`
}

func (ContestQuizAttempt) TableName() string { return "contest_quiz_attempts" }

// ContestQuizAnswer stores individual answers in a quiz attempt
type ContestQuizAnswer struct {
	SelectedOptionID *uint               `gorm:"index" json:"selected_option_id,omitempty"`
	Attempt          *ContestQuizAttempt `gorm:"foreignKey:AttemptID;constraint:OnDelete:CASCADE" json:"attempt,omitempty"`
	ID               uint                `gorm:"primaryKey;autoIncrement" json:"id"`
	AttemptID        uint                `gorm:"not null;index" json:"attempt_id"`
	QuestionID       uint                `gorm:"not null" json:"question_id"`
	MarksObtained    int                 `gorm:"default:0" json:"marks_obtained"`
	IsCorrect        bool                `gorm:"default:false" json:"is_correct"`
}

func (ContestQuizAnswer) TableName() string { return "contest_quiz_answers" }
