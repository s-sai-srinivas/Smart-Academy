package models

import (
	"time"

	"gorm.io/gorm"
)

// Topic represents a canonical topic/tag for coding problems and lab sessions
// Created and managed by super admins, belongs to a Subject.
// Used by college admins for lab sessions and problem tagging to ensure consistency.
type Topic struct {
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Subject     *Subject       `gorm:"foreignKey:SubjectID;constraint:OnDelete:CASCADE" json:"subject,omitempty"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	CreatedBy   string         `gorm:"size:50;not null" json:"created_by"`
	ID          uint           `gorm:"primaryKey" json:"id"`
	SubjectID   uint           `gorm:"not null;index" json:"subject_id"`
}
