package models

import (
	"time"

	"gorm.io/gorm"
)

// CourseOffering represents a specific instance of a course for an academic context
// This replaces the old Course+Semester+Section combination
type CourseOffering struct {
	UpdatedAt          time.Time           `json:"updated_at"`
	CreatedAt          time.Time           `json:"created_at"`
	Section            *Section            `gorm:"foreignKey:SectionID;references:SectionID" json:"section,omitempty"`
	CurriculumID       *uint               `json:"curriculum_id"`
	AcademicYear       *AcademicYear       `gorm:"foreignKey:AcademicYearID;references:ID" json:"academic_year,omitempty"`
	Curriculum         *Curriculum         `gorm:"foreignKey:CurriculumID;references:ID" json:"curriculum,omitempty"`
	Course             *Course             `gorm:"foreignKey:CourseID;references:ID;constraint:OnDelete:CASCADE" json:"course,omitempty"`
	DeletedAt          gorm.DeletedAt      `gorm:"index" json:"-"`
	Enrollments        []Enrollment        `gorm:"foreignKey:CourseOfferingID" json:"enrollments,omitempty"`
	FacultyAssignments []FacultyAssignment `gorm:"foreignKey:CourseOfferingID" json:"faculty_assignments,omitempty"`
	Exams              []Exam              `gorm:"foreignKey:CourseOfferingID" json:"exams,omitempty"`
	ID                 uint                `gorm:"primaryKey;autoIncrement" json:"id"`
	SectionID          uint                `gorm:"not null" json:"section_id"`
	AcademicYearID     uint                `gorm:"not null" json:"academic_year_id"`
	CourseID           uint                `gorm:"not null" json:"course_id"`
	IsActive           bool                `gorm:"default:true" json:"is_active"`
}
