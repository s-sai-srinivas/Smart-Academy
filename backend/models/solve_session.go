package models

import (
	"time"

	"gorm.io/gorm"
)

// SolveSession tracks when users start working on a problem
// This enables accurate time tracking for analytics and prevents manipulation
type SolveSession struct {
	StartedAt  time.Time      `json:"started_at"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	EndedAt    *time.Time     `json:"ended_at,omitempty"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	UserRegdNo string         `gorm:"index:idx_solve_session_user_problem;size:50" json:"user_regdno"`
	ID         uint           `gorm:"primaryKey" json:"id"`
	ProblemID  uint           `gorm:"index:idx_solve_session_user_problem;index" json:"problem_id"`
	Completed  bool           `gorm:"default:false" json:"completed"`
}

// TableName specifies the table name for SolveSession
func (SolveSession) TableName() string {
	return "solve_sessions"
}
