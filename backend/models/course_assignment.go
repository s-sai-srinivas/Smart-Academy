package models

import "gorm.io/gorm"

// CourseAssignment is a simplified faculty-to-course assignment for HOD management
type CourseAssignment struct {
	Faculty *User   `gorm:"foreignKey:FacultyRegdNo;references:RegdNo;constraint:OnDelete:CASCADE" json:"faculty,omitempty"`
	Course  *Course `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE" json:"course,omitempty"`
	gorm.Model
	FacultyRegdNo string `gorm:"index;not null;size:50" json:"faculty_regdno"`
	Section       string `gorm:"size:10;not null;default:'A'" json:"section"`
	CourseID      uint   `gorm:"index;not null" json:"course_id"`
	Year          int    `gorm:"not null;default:1" json:"year"`
	Semester      int    `gorm:"not null;default:1" json:"semester"`
}
