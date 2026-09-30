package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents a platform user (base table)
type User struct {
	UpdatedAt    time.Time      `json:"updated_at"`
	CreatedAt    time.Time      `json:"created_at"`
	CollegeID    *string        `gorm:"index" json:"college_id,omitempty"`
	BranchID     *uint          `json:"branch_id,omitempty"`
	RoleID       *uint          `json:"role_id,omitempty"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Password     string         `gorm:"column:password;size:255;not null" json:"-"`
	Role         string         `gorm:"size:50;default:student" json:"role"`
	Phone        string         `gorm:"size:15" json:"phone,omitempty"`
	RegdNo       string         `gorm:"column:regdno;size:50;primaryKey" json:"regdno"`
	Email        string         `gorm:"size:100;unique;not null" json:"email"`
	Name         string         `gorm:"column:name;size:200" json:"name"`
	TokenVersion int            `gorm:"default:0" json:"-"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
}

// HashPassword hashes the user's password
func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword compares the provided password with the hashed password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// =====================================================
// Student: Extension for academic identity
// =====================================================

// Student represents academic-specific information for students
// Extends User with batch-based academic identity
type Student struct {
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	SectionID     *uint       `json:"section_id,omitempty"`
	User          *User       `gorm:"-" json:"user,omitempty"`
	Curriculum    *Curriculum `gorm:"-" json:"curriculum,omitempty"`
	Branch        *Branch     `gorm:"-" json:"branch,omitempty"`
	Section       *Section    `gorm:"-" json:"section,omitempty"`
	RegdNo        string      `gorm:"column:regd_no;size:50;primaryKey" json:"regdno"`
	Status        string      `gorm:"size:20;default:active" json:"status"`
	AdmissionYear int         `gorm:"not null" json:"admission_year"`
	EntryLevel    int         `gorm:"default:1" json:"entry_level"`
	BranchID      uint        `gorm:"not null" json:"branch_id"`
	ProgressIndex int         `gorm:"default:1" json:"progress_index"`
	CohortYear    int         `gorm:"not null" json:"cohort_year"`
	CurriculumID  uint        `gorm:"not null" json:"curriculum_id"`
}

// =====================================================
// Faculty: Extension for faculty members
// =====================================================

// Faculty represents employment-specific information for faculty
// Extends User with department and employment details
type Faculty struct {
	JoiningDate time.Time `json:"joining_date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	User        *User     `gorm:"-" json:"user,omitempty"`
	Branch      *Branch   `gorm:"-" json:"branch,omitempty"`
	RegdNo      string    `gorm:"column:regd_no;size:50;primaryKey" json:"regdno"`
	Designation string    `gorm:"size:100" json:"designation"`
	BranchID    uint      `gorm:"not null" json:"branch_id"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
}
