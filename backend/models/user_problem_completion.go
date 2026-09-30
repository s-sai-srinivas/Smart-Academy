package models

import (
	"time"

	"gorm.io/gorm"
)

// UserProblemCompletion tracks which problems each user has completed
type UserProblemCompletion struct {
	CompletedAt       time.Time      `json:"completed_at"`
	User              *User          `gorm:"foreignKey:UserRegdNo;references:RegdNo" json:"user,omitempty"`
	CollegeID         *string        `gorm:"index" json:"college_id"`
	CourseID          *uint          `json:"course_id"`
	Problem           *Problem       `gorm:"foreignKey:ProblemID;references:ID" json:"problem,omitempty"`
	College           *College       `gorm:"foreignKey:CollegeID;references:CollegeID" json:"college,omitempty"`
	Course            *Course        `gorm:"foreignKey:CourseID;references:ID" json:"course,omitempty"`
	Submission        *Submission    `gorm:"foreignKey:FirstSubmissionID;references:ID" json:"submission,omitempty"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
	UserRegdNo        string         `gorm:"index:idx_user_problem,unique;size:50" json:"user_regdno"`
	ProblemID         uint           `gorm:"index:idx_user_problem,unique" json:"problem_id"`
	TimeTakenSeconds  float64        `json:"time_taken_seconds"`
	FirstSubmissionID uint           `json:"first_submission_id"`
	ID                uint           `gorm:"primaryKey" json:"id"`
}
