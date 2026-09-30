package models

import (
	"time"

	"gorm.io/gorm"
)

// Course represents an academic course (abstract definition)
// NO SEMESTER, NO BATCH - just the academic blueprint
// Theory and Lab are completely independent courses
// Each course is either a theory course or a lab course (not both)
// Courses are college-specific - each college has its own set of courses
type Course struct {
	UpdatedAt   time.Time      `json:"updated_at"`
	CreatedAt   time.Time      `json:"created_at"`
	Lab         *Lab           `json:"lab,omitempty"`
	College     *College       `gorm:"foreignKey:CollegeID;references:CollegeID;constraint:OnDelete:CASCADE" json:"college,omitempty"`
	Theory      *Theory        `json:"theory,omitempty"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Syllabus    string         `gorm:"type:text" json:"syllabus"`
	Outcomes    string         `gorm:"type:text" json:"outcomes"`
	Description string         `gorm:"type:text" json:"description"`
	CollegeID   string         `gorm:"size:50;not null;uniqueIndex:uni_courses_course_code_college" json:"college_id"`
	CreatedBy   string         `json:"created_by"`
	CourseType  string         `gorm:"size:20;not null" json:"course_type"`
	CourseName  string         `gorm:"size:200;not null" json:"course_name"`
	CourseCode  string         `gorm:"size:30;not null;uniqueIndex:uni_courses_course_code_college" json:"course_code"`
	Credits     int            `gorm:"default:3" json:"credits"`
	ID          uint           `gorm:"primaryKey" json:"id"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
}
