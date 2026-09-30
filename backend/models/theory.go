package models

import (
	"time"

	"gorm.io/gorm"
)

// Theory represents a theory course extension
// Theory is a 1-to-1 extension for theory-type courses (where course_category = 'theory')
// Each theory course has exactly one Theory record with additional theory-specific details
type Theory struct {
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Course     *Course        `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE" json:"course,omitempty"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	TheoryName string         `gorm:"size:200;not null" json:"theory_name"`
	TheoryCode string         `gorm:"size:20;unique;not null" json:"theory_code"`
	Weeks      []TheoryWeek   `gorm:"foreignKey:TheoryID" json:"weeks,omitempty"`
	ID         uint           `gorm:"primaryKey" json:"id"`
	CourseID   uint           `gorm:"unique;not null" json:"course_id"`
}

// TheoryWeek represents a week within a theory course
// TheoryWeeks belong to Theories, which belong to theory-type Courses
type TheoryWeek struct {
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Theory    *Theory        `gorm:"foreignKey:TheoryID;constraint:OnDelete:CASCADE" json:"theory,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	WeekName  string         `gorm:"size:200;not null" json:"week_name"`
	Modules   []TheoryModule `gorm:"foreignKey:TheoryWeekID" json:"modules,omitempty"`
	ID        uint           `gorm:"primaryKey" json:"id"`
	TheoryID  uint           `gorm:"not null" json:"theory_id"`
	WeekOrder int            `gorm:"default:0" json:"week_order"`
}

// TheoryModule represents a module within a week
type TheoryModule struct {
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	TheoryWeek   *TheoryWeek    `gorm:"foreignKey:TheoryWeekID;constraint:OnDelete:CASCADE" json:"theory_week,omitempty"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	ModuleName   string         `gorm:"size:200;not null" json:"module_name"`
	Description  string         `gorm:"type:text" json:"description"`
	Content      string         `gorm:"type:text" json:"content"`
	ID           uint           `gorm:"primaryKey" json:"id"`
	TheoryWeekID uint           `gorm:"not null" json:"theory_week_id"`
}

// TheoryPDF represents a PDF file attached to a theory course, optionally linked to a module
type TheoryPDF struct {
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	TheoryModuleID *uint          `gorm:"index" json:"theory_module_id,omitempty"`
	Theory         *Theory        `gorm:"foreignKey:TheoryID;constraint:OnDelete:CASCADE" json:"theory,omitempty"`
	TheoryModule   *TheoryModule  `gorm:"foreignKey:TheoryModuleID;constraint:OnDelete:SET NULL" json:"theory_module,omitempty"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	FileName       string         `gorm:"size:500;not null" json:"file_name"`
	DisplayName    string         `gorm:"size:500" json:"display_name"`
	FilePath       string         `gorm:"size:1000;not null" json:"-"`
	ID             uint           `gorm:"primaryKey" json:"id"`
	TheoryID       uint           `gorm:"not null;index" json:"theory_id"`
	FileSize       int64          `gorm:"default:0" json:"file_size"`
}
