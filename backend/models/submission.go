package models

import (
	"time"

	"gorm.io/gorm"
)

// Submission represents a code submission
type Submission struct {
	SubmittedAt   time.Time      `json:"submitted_at"`
	Contest       *Contest       `gorm:"foreignKey:ContestID;references:ContestID" json:"contest,omitempty"`
	LabSession    *LabSession    `gorm:"foreignKey:LabSessionID;references:ID" json:"lab_session,omitempty"`
	CollegeID     *string        `gorm:"index;size:50" json:"college_id"`
	CourseID      *uint          `json:"course_id"`
	SectionID     *uint          `json:"section_id"`
	LabSessionID  *uint          `json:"lab_session_id"`
	ContestID     *uint          `json:"contest_id"`
	Section       *Section       `gorm:"foreignKey:SectionID;references:SectionID" json:"section,omitempty"`
	Course        *Course        `gorm:"foreignKey:CourseID;references:ID" json:"course,omitempty"`
	College       *College       `gorm:"foreignKey:CollegeID;references:CollegeID" json:"college,omitempty"`
	Problem       *Problem       `gorm:"foreignKey:ProblemID;references:ID" json:"problem,omitempty"`
	User          *User          `gorm:"foreignKey:UserRegdNo;references:RegdNo" json:"user,omitempty"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	ErrorMessage  string         `gorm:"type:text" json:"error_message,omitempty"`
	UserRegdNo    string         `gorm:"index;size:50" json:"user_regdno"`
	SourceCode    string         `gorm:"type:text" json:"source_code"`
	Status        string         `gorm:"size:50" json:"status"`
	TimeSpent     float64        `json:"time_spent"`
	PassedTests   int            `json:"passed_tests"`
	Score         int            `json:"score"`
	ID            uint           `gorm:"primaryKey" json:"id"`
	TotalTests    int            `json:"total_tests"`
	MemoryUsed    int            `json:"memory_used"`
	ExecutionTime float64        `json:"execution_time"`
	LanguageID    int            `json:"language_id"`
	ProblemID     uint           `gorm:"index" json:"problem_id"`
	MaxScore      int            `json:"max_score"`
	Passed        bool           `gorm:"default:false" json:"passed"`
}
