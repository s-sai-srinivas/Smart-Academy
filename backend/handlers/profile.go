package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ProfileResponse contains all profile data for a user
type ProfileResponse struct {
	Student *StudentProfileData `json:"student,omitempty"`
	Faculty *FacultyProfileData `json:"faculty,omitempty"`
	Stats   *StatsData          `json:"stats,omitempty"`
	User    models.User         `json:"user"`
}

type StudentProfileData struct {
	BranchName    string `json:"branch_name"`
	SectionName   string `json:"section_name,omitempty"`
	Status        string `json:"status"`
	CohortYear    int    `json:"cohort_year"`
	AdmissionYear int    `json:"admission_year"`
	ProgressIndex int    `json:"progress_index"`
}

type FacultyProfileData struct {
	BranchName  string `json:"branch_name"`
	Designation string `json:"designation"`
	JoiningDate string `json:"joining_date"`
	IsActive    bool   `json:"is_active"`
}

// GetProfile returns the authenticated user's complete profile
func GetProfile(c *gin.Context) {
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	regdNoStr := userRegdNo.(string)

	// Get user basic info
	var user models.User
	if err := database.DB.Where("regdno = ?", regdNoStr).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	response := ProfileResponse{
		User: user,
	}

	// Get role-specific data
	role, _ := c.Get("role")
	if role != nil {
		userRole := role.(string)

		if userRole == "student" {
			// Get student details
			var student models.Student
			if err := database.DB.Where("regd_no = ?", regdNoStr).First(&student).Error; err == nil {
				// Get related data
				var branch models.Branch
				database.DB.First(&branch, student.BranchID)

				var section models.Section
				if student.SectionID != nil {
					database.DB.First(&section, *student.SectionID)
				}

				studentData := &StudentProfileData{
					BranchName:    branch.BranchName,
					CohortYear:    student.CohortYear,
					AdmissionYear: student.AdmissionYear,
					ProgressIndex: student.ProgressIndex,
					Status:        student.Status,
				}
				if section.SectionID != 0 {
					studentData.SectionName = section.SectionName
				}
				response.Student = studentData

				// Get student stats
				stats := getStats(regdNoStr)
				response.Stats = &stats
			}
		} else if userRole == "faculty" {
			// Get faculty details
			var faculty models.Faculty
			if err := database.DB.Where("regd_no = ?", regdNoStr).First(&faculty).Error; err == nil {
				var branch models.Branch
				database.DB.First(&branch, faculty.BranchID)

				response.Faculty = &FacultyProfileData{
					BranchName:  branch.BranchName,
					Designation: faculty.Designation,
					JoiningDate: faculty.JoiningDate.Format("2006-01-02"),
					IsActive:    faculty.IsActive,
				}
			}
		}
	}

	c.JSON(http.StatusOK, response)
}
