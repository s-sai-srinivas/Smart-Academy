package models

import (
	"time"

	"gorm.io/datatypes"
)

// ContestLeaderboardSnapshot stores a finalized leaderboard for ended contests.
type ContestLeaderboardSnapshot struct {
	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	Data      datatypes.JSON `gorm:"type:jsonb;not null" json:"data"`
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID uint           `gorm:"uniqueIndex;not null" json:"contest_id"`
}

// ContestNonParticipantsSnapshot stores non-joined student data for ended contests.
type ContestNonParticipantsSnapshot struct {
	CreatedAt time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	Data      datatypes.JSON `gorm:"type:jsonb;not null" json:"data"`
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID uint           `gorm:"uniqueIndex;not null" json:"contest_id"`
}
