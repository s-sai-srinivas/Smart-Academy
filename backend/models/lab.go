package models

import (
	"time"

	"gorm.io/gorm"
)

// Lab represents a lab course extension
// Lab is a 1-to-1 extension for lab-type courses (where course_category = 'lab')
// Each lab course has exactly one Lab record with additional lab-specific details
type Lab struct {
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Course    *Course        `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE" json:"course,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	LabName   string         `gorm:"size:200;not null" json:"lab_name"`
	LabCode   string         `gorm:"size:20;unique;not null" json:"lab_code"`
	ID        uint           `gorm:"primaryKey" json:"id"`
	CourseID  uint           `gorm:"unique;not null" json:"course_id"`
}

// LabSession represents a session within a lab course
// LabSessions belong to Labs, which belong to lab-type Courses
type LabSession struct {
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Lab          *Lab           `gorm:"foreignKey:LabID;constraint:OnDelete:CASCADE" json:"lab,omitempty"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	SessionName  string         `gorm:"size:200;not null" json:"session_name"`
	TopicName    string         `gorm:"size:200" json:"topic_name"`
	Problems     []Problem      `gorm:"foreignKey:LabSessionID" json:"problems,omitempty"`
	ID           uint           `gorm:"primaryKey" json:"id"`
	LabID        uint           `gorm:"not null;index" json:"lab_id"`
	SessionOrder int            `gorm:"default:0" json:"session_order"`
}
