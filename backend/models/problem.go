package models

import (
	"time"

	"gorm.io/gorm"
)

// Problem represents coding problems for labs
type Problem struct {
	UpdatedAt    time.Time      `json:"updated_at"`
	CreatedAt    time.Time      `json:"created_at"`
	CollegeID    *string        `gorm:"index" json:"college_id"`
	Subject      *Subject       `gorm:"foreignKey:SubjectID;constraint:OnDelete:SET NULL" json:"subject,omitempty"`
	LabSession   *LabSession    `gorm:"foreignKey:LabSessionID;constraint:OnDelete:SET NULL" json:"lab_session,omitempty"`
	College      *College       `gorm:"foreignKey:CollegeID" json:"college,omitempty"`
	SubjectID    *uint          `gorm:"index" json:"subject_id"`
	LabSessionID *uint          `index:"idx_problem_lab_session" json:"lab_session_id"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	CreatedBy    string         `json:"created_by"`
	Tags         string         `gorm:"size:500" json:"tags"`
	Difficulty   string         `gorm:"size:20;default:easy" json:"difficulty"`
	Description  string         `gorm:"type:text;not null" json:"description"`
	Title        string         `gorm:"size:200;not null" json:"title"`
	TestCases    []TestCase     `gorm:"constraint:OnDelete:CASCADE;foreignKey:ProblemID" json:"test_cases,omitempty"`
	MemoryLimit  int            `gorm:"default:256000" json:"memory_limit"`
	TimeLimit    int            `gorm:"default:2000" json:"time_limit"`
	Points       int            `gorm:"default:0" json:"points"`
	ID           uint           `gorm:"primaryKey" json:"id"`
	IsGlobal     bool           `gorm:"default:false" json:"is_global"`
}
