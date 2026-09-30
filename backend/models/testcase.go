package models

import "time"

// TestCase represents test cases for problems
type TestCase struct {
	CreatedAt      time.Time `json:"created_at"`
	Problem        *Problem  `gorm:"constraint:OnDelete:CASCADE;foreignKey:ProblemID" json:"-"`
	Input          string    `gorm:"not null;type:text" json:"input"`
	ExpectedOutput string    `gorm:"not null;type:text" json:"expected_output"`
	ID             uint      `gorm:"primaryKey" json:"id"`
	ProblemID      uint      `gorm:"not null;index" json:"problem_id"`
	Points         int       `gorm:"default:10" json:"points"`
	IsSample       bool      `gorm:"default:false" json:"is_sample"`
}
