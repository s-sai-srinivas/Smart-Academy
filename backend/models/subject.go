package models

import (
	"time"

	"gorm.io/gorm"
)

// Subject represents a subject/domain (e.g., DSA, OS, DBMS)
// Created and managed by super admins. Topics belong to subjects.
type Subject struct {
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:100;unique;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	CreatedBy   string         `gorm:"size:50;not null" json:"created_by"`
	Topics      []Topic        `gorm:"foreignKey:SubjectID" json:"topics,omitempty"`
	ID          uint           `gorm:"primaryKey" json:"id"`
}
