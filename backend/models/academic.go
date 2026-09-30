package models

import (
	"time"

	"gorm.io/gorm"
)

// =====================================================
// LEVEL 1: Independent Tables (No foreign keys)
// =====================================================

// Role defines user roles in the system
type Role struct {
	RoleName string `gorm:"size:50;unique;not null" json:"role_name"`
	RoleID   uint   `gorm:"primaryKey;autoIncrement" json:"role_id"`
}

// Permission defines system permissions
type Permission struct {
	PermissionName string `gorm:"size:100;unique;not null" json:"permission_name"`
	PermissionID   uint   `gorm:"primaryKey;autoIncrement" json:"permission_id"`
}

// RolePermission maps roles to permissions (many-to-many)
type RolePermission struct {
	Role         *Role       `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE" json:"-"`
	Permission   *Permission `gorm:"foreignKey:PermissionID;constraint:OnDelete:CASCADE" json:"-"`
	RoleID       uint        `gorm:"primaryKey" json:"role_id"`
	PermissionID uint        `gorm:"primaryKey" json:"permission_id"`
}

// College represents an educational institution
type College struct {
	CreatedAt   time.Time `json:"created_at"`
	CollegeID   string    `gorm:"primaryKey;size:50" json:"college_id"`
	CollegeName string    `gorm:"size:200;not null" json:"college_name"`
	ShortName   string    `gorm:"size:50;not null;unique" json:"short_name"`
	Address     string    `gorm:"type:text" json:"address"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
}

// Program represents academic programs like BTech, MTech, MBA
type Program struct {
	ProgramCode   string `gorm:"size:20;not null;unique" json:"program_code"`
	ProgramName   string `gorm:"size:100;not null" json:"program_name"`
	ProgramID     uint   `gorm:"primaryKey;autoIncrement" json:"program_id"`
	DurationYears int    `gorm:"default:4" json:"duration_years"`
}

// AcademicYear represents an academic year (calendar context)
type AcademicYear struct {
	StartDate      time.Time `gorm:"not null" json:"start_date"`
	EndDate        time.Time `gorm:"not null" json:"end_date"`
	Name           string    `gorm:"size:20;not null;unique" json:"name"`
	AcademicYearID uint      `gorm:"primaryKey;autoIncrement" json:"academic_year_id"`
	IsCurrent      bool      `gorm:"default:false" json:"is_current"`
}

// =====================================================
// LEVEL 2: Academic Structure (Batch-Based, No Semester)
// =====================================================

// Branch represents departments within a program
type Branch struct {
	College    *College `gorm:"foreignKey:CollegeID;references:CollegeID;constraint:OnDelete:CASCADE" json:"college,omitempty"`
	Program    *Program `gorm:"foreignKey:ProgramID;constraint:OnDelete:CASCADE" json:"program,omitempty"`
	BranchName string   `gorm:"size:100;not null" json:"branch_name"`
	ShortName  string   `gorm:"size:20;not null" json:"short_name"`
	CollegeID  string   `gorm:"size:50;not null;index" json:"college_id"`
	BranchID   uint     `gorm:"primaryKey;autoIncrement" json:"branch_id"`
	ProgramID  uint     `gorm:"not null" json:"program_id"`
}

// Regulation represents a curriculum version
// Applies to a set of cohorts (e.g., R2021 for cohorts 2021-2024)
type Regulation struct {
	Code                string `gorm:"size:20;not null;unique" json:"code"`
	Description         string `gorm:"type:text" json:"description"`
	RegulationID        uint   `gorm:"primaryKey;autoIncrement" json:"regulation_id"`
	EffectiveFromCohort int    `gorm:"not null" json:"effective_from_cohort"`
	IsActive            bool   `gorm:"default:true" json:"is_active"`
}

// Curriculum represents the complete academic structure for a program-regulation combination
type Curriculum struct {
	CurriculumID  uint `gorm:"primaryKey;autoIncrement" json:"curriculum_id"`
	ProgramID     uint `gorm:"not null" json:"program_id"`
	RegulationID  uint `gorm:"not null;index" json:"regulation_id"` // Index for queries, but no FK constraint
	DurationYears int  `gorm:"not null;default:4" json:"duration_years"`
	IsActive      bool `gorm:"default:true" json:"is_active"`

	// No struct pointers to avoid GORM inference issues
	// Load via joins in queries
}

// CurriculumCourse maps courses to a specific study year within a curriculum
type CurriculumCourse struct {
	Curriculum     *Curriculum `gorm:"foreignKey:CurriculumID;constraint:OnDelete:CASCADE" json:"curriculum,omitempty"`
	Course         *Course     `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE" json:"course,omitempty"`
	Category       string      `gorm:"size:50" json:"category"`
	ID             uint        `gorm:"primaryKey;autoIncrement" json:"id"`
	CurriculumID   uint        `gorm:"not null" json:"curriculum_id"`
	CourseID       uint        `gorm:"not null" json:"course_id"`
	SequenceInYear int         `gorm:"default:0" json:"sequence_in_year"`
	IsMandatory    bool        `gorm:"default:true" json:"is_mandatory"`
}

// Section represents class sections for a specific batch and academic year
// NO SEMESTER DEPENDENCY - uses CohortYear instead
type Section struct {
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	SectionName    string         `gorm:"size:10;not null" json:"section_name"`
	SectionID      uint           `gorm:"primaryKey;autoIncrement" json:"section_id"`
	BranchID       uint           `gorm:"not null" json:"branch_id"`
	CohortYear     int            `gorm:"not null" json:"cohort_year"`
	AcademicYearID uint           `gorm:"not null" json:"academic_year_id"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
}
