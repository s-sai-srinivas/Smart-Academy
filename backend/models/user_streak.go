package models

import (
	"time"

	"gorm.io/gorm"
)

// UserStreak tracks user submission streaks
type UserStreak struct {
	LastActivityDate time.Time      `json:"last_activity_date"`
	User             *User          `gorm:"foreignKey:UserRegdNo;references:RegdNo" json:"user,omitempty"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	UserRegdNo       string         `gorm:"uniqueIndex;size:50" json:"user_regdno"`
	ID               uint           `gorm:"primaryKey" json:"id"`
	CurrentStreak    int            `gorm:"default:0" json:"current_streak"`
	LongestStreak    int            `gorm:"default:0" json:"longest_streak"`
	TotalPoints      int            `gorm:"default:0" json:"total_points"`
}

// UserActivity tracks daily activity for heatmap
type UserActivity struct {
	Date       time.Time      `gorm:"index:idx_user_date,unique;type:date" json:"date"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	UserRegdNo string         `gorm:"index:idx_user_date,unique;size:50" json:"user_regdno"`
	ID         uint           `gorm:"primaryKey" json:"id"`
	Count      int            `gorm:"default:0" json:"count"`
}
