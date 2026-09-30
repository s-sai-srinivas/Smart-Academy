package models

import (
	"time"

	"gorm.io/gorm"
)

// Exam represents an examination for a course offering
// Attached to CourseOffering, not directly to Course
type Exam struct {
	ScheduledDate    time.Time       `gorm:"not null" json:"scheduled_date"`
	UpdatedAt        time.Time       `json:"updated_at"`
	CreatedAt        time.Time       `json:"created_at"`
	CourseOffering   *CourseOffering `gorm:"foreignKey:CourseOfferingID;constraint:OnDelete:CASCADE" json:"course_offering,omitempty"`
	DeletedAt        gorm.DeletedAt  `gorm:"index" json:"-"`
	Name             string          `gorm:"size:200" json:"name"`
	Description      string          `gorm:"type:text" json:"description"`
	Type             string          `gorm:"size:50;not null" json:"type"`
	Results          []ExamResult    `gorm:"foreignKey:ExamID" json:"results,omitempty"`
	DurationMinutes  int             `gorm:"not null" json:"duration_minutes"`
	TotalMarks       int             `gorm:"not null" json:"total_marks"`
	PassingMarks     int             `gorm:"not null" json:"passing_marks"`
	ID               uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	Sequence         int             `gorm:"default:1" json:"sequence"`
	CourseOfferingID uint            `gorm:"not null" json:"course_offering_id"`
	IsActive         bool            `gorm:"default:true" json:"is_active"`
}

// ExamResult represents a student's result for an exam
// Multiple attempts = multiple ExamResult rows
type ExamResult struct {
	EvaluatedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"evaluated_at"`
	Exam          *Exam     `gorm:"foreignKey:ExamID;constraint:OnDelete:CASCADE" json:"exam,omitempty"`
	Student       *User     `gorm:"foreignKey:StudentRegdNo;references:RegdNo;constraint:OnDelete:CASCADE" json:"student,omitempty"`
	StudentRegdNo string    `gorm:"column:student_regdno;size:50;not null" json:"student_regdno"`
	Grade         string    `gorm:"size:10" json:"grade"`
	Remarks       string    `gorm:"type:text" json:"remarks,omitempty"`
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ExamID        uint      `gorm:"not null" json:"exam_id"`
	MarksObtained int       `gorm:"not null" json:"marks_obtained"`
	AttemptNo     int       `gorm:"default:1" json:"attempt_no"`
}
