package models

import "time"

// Enrollment represents a student's enrollment in a course offering
// This REPLACES the old StudentEnrollment model
// Controls visibility of courses to students
type Enrollment struct {
	EnrolledAt       time.Time       `gorm:"default:CURRENT_TIMESTAMP" json:"enrolled_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	CollegeID        *string         `gorm:"size:50;index" json:"college_id"`
	Student          *User           `gorm:"foreignKey:StudentRegdNo;references:RegdNo;constraint:OnDelete:CASCADE" json:"student,omitempty"`
	CourseOffering   *CourseOffering `gorm:"foreignKey:CourseOfferingID;constraint:OnDelete:CASCADE" json:"course_offering,omitempty"`
	College          *College        `gorm:"foreignKey:CollegeID;references:CollegeID" json:"college,omitempty"`
	Type             string          `gorm:"size:20;not null" json:"type"`
	Status           string          `gorm:"size:20;default:enrolled" json:"status"`
	FinalGrade       string          `gorm:"size:10" json:"final_grade"`
	StudentRegdNo    string          `gorm:"column:student_regdno;size:50;not null;index:idx_enrollment_unique,unique:student_offering" json:"student_regdno"`
	CourseOfferingID uint            `gorm:"not null;index:idx_enrollment_unique,unique:student_offering" json:"course_offering_id"`
	AttemptNo        int             `gorm:"default:1" json:"attempt_no"`
	ID               uint            `gorm:"primaryKey;autoIncrement" json:"id"`
}

// FacultyAssignment represents faculty assigned to a course offering
// This REPLACES the old FacultyCourseAssignment and CourseAssignment models
// Faculty teaches an offering instance, not the course directly
type FacultyAssignment struct {
	AssignedAt       time.Time       `gorm:"default:CURRENT_TIMESTAMP" json:"assigned_at"`
	CollegeID        *string         `gorm:"size:50;index" json:"college_id"`
	Faculty          *User           `gorm:"foreignKey:FacultyRegdNo;references:RegdNo;constraint:OnDelete:CASCADE" json:"faculty,omitempty"`
	CourseOffering   *CourseOffering `gorm:"foreignKey:CourseOfferingID;constraint:OnDelete:CASCADE" json:"course_offering,omitempty"`
	College          *College        `gorm:"foreignKey:CollegeID;references:CollegeID" json:"college,omitempty"`
	FacultyRegdNo    string          `gorm:"size:50;not null;index" json:"faculty_regdno"`
	Role             string          `gorm:"size:20;default:teacher" json:"role"`
	ID               uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	CourseOfferingID uint            `gorm:"not null;index" json:"course_offering_id"`
	IsActive         bool            `gorm:"default:true" json:"is_active"`
}
