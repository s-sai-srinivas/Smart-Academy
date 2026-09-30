package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ==================== PRINCIPAL DASHBOARD ====================

// GetPrincipalStats returns dashboard statistics for principal
func GetPrincipalStats(c *gin.Context) {
	collegeID, _ := c.Get("college_id")

	var totalFaculty int64
	var totalHODs int64
	var totalDepartments int64
	var totalStudents int64

	if collegeID != nil {
		collegeIDPtr := collegeID.(*string)

		// Count faculty in college
		database.DB.Model(&models.User{}).
			Where("role IN (?, ?) AND college_id = ? AND is_active = true", "faculty", "hod", *collegeIDPtr).
			Count(&totalFaculty)

		// Count HODs in college
		database.DB.Model(&models.User{}).
			Where("role = ? AND college_id = ? AND is_active = true", "hod", *collegeIDPtr).
			Count(&totalHODs)

		// Count distinct branches (departments) in college
		database.DB.Model(&models.Branch{}).
			Where("college_id = ?", *collegeIDPtr).
			Count(&totalDepartments)

		// Count students
		database.DB.Model(&models.User{}).
			Where("role = ? AND college_id = ? AND is_active = true", "student", *collegeIDPtr).
			Count(&totalStudents)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_faculty":     totalFaculty,
		"total_hods":        totalHODs,
		"total_departments": totalDepartments,
		"total_students":    totalStudents,
	})
}

// ==================== HOD MANAGEMENT ====================

// DepartmentHODInfo represents a department with its current HOD and available faculty
type DepartmentHODInfo struct {
	CurrentHOD *struct {
		RegdNo string `json:"regdno"`
		Name   string `json:"name"`
		Email  string `json:"email"`
	} `json:"current_hod"`
	BranchName       string          `json:"branch_name"`
	ShortName        string          `json:"short_name"`
	ProgramName      string          `json:"program_name"`
	ProgramCode      string          `json:"program_code"`
	AvailableFaculty []FacultyOption `json:"available_faculty"`
	BranchID         uint            `json:"branch_id"`
	ProgramID        uint            `json:"program_id"`
}

// FacultyOption represents a faculty member available for HOD assignment
type FacultyOption struct {
	RegdNo      string `json:"regdno"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Designation string `json:"designation"`
}

// GetDepartmentsWithHOD returns all departments with their current HOD and available faculty
func GetDepartmentsWithHOD(c *gin.Context) {
	collegeID, _ := c.Get("college_id")

	if collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Principal has no college assigned"})
		return
	}
	collegeIDPtr := collegeID.(*string)

	// Get all branches (departments) in this college, with program info
	var branches []models.Branch
	if err := database.DB.Preload("Program").Where("college_id = ?", *collegeIDPtr).Find(&branches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch departments"})
		return
	}

	var departments []DepartmentHODInfo

	for _, branch := range branches {
		dept := DepartmentHODInfo{
			BranchID:   branch.BranchID,
			BranchName: branch.BranchName,
			ShortName:  branch.ShortName,
			ProgramID:  branch.ProgramID,
		}
		if branch.Program != nil {
			dept.ProgramName = branch.Program.ProgramName
			dept.ProgramCode = branch.Program.ProgramCode
		}

		// Find current HOD for this branch
		var hodUser models.User
		err := database.DB.Where("role = ? AND branch_id = ? AND college_id = ? AND is_active = true",
			"hod", branch.BranchID, *collegeIDPtr).First(&hodUser).Error
		if err == nil {
			dept.CurrentHOD = &struct {
				RegdNo string `json:"regdno"`
				Name   string `json:"name"`
				Email  string `json:"email"`
			}{
				RegdNo: hodUser.RegdNo,
				Name:   hodUser.Name,
				Email:  hodUser.Email,
			}
		}

		// Get available faculty for this branch (faculty role only, not already HOD)
		var facultyUsers []models.User
		database.DB.Where("role = ? AND branch_id = ? AND college_id = ? AND is_active = true",
			"faculty", branch.BranchID, *collegeIDPtr).Find(&facultyUsers)

		// Also get designation from Faculty table
		dept.AvailableFaculty = make([]FacultyOption, 0)
		for _, fu := range facultyUsers {
			option := FacultyOption{
				RegdNo: fu.RegdNo,
				Name:   fu.Name,
				Email:  fu.Email,
			}
			// Try to get designation from Faculty extension table
			var faculty models.Faculty
			if err := database.DB.Where("regd_no = ?", fu.RegdNo).First(&faculty).Error; err == nil {
				option.Designation = faculty.Designation
			}
			dept.AvailableFaculty = append(dept.AvailableFaculty, option)
		}

		departments = append(departments, dept)
	}

	c.JSON(http.StatusOK, departments)
}

// AssignHODRequest represents the request body for HOD assignment
type AssignHODRequest struct {
	FacultyRegdNo string `json:"faculty_regdno" binding:"required"`
	BranchID      uint   `json:"branch_id" binding:"required"`
}

// AssignHOD assigns or reassigns an HOD for a department
func AssignHOD(c *gin.Context) {
	var req AssignHODRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "branch_id and faculty_regdno are required"})
		return
	}

	collegeID, _ := c.Get("college_id")
	if collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Principal has no college assigned"})
		return
	}
	collegeIDPtr := collegeID.(*string)

	// Verify the faculty member exists and belongs to the correct branch and college
	var facultyUser models.User
	err := database.DB.Where("regdno = ? AND college_id = ? AND is_active = true",
		req.FacultyRegdNo, *collegeIDPtr).First(&facultyUser).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Faculty member not found in your college"})
		return
	}

	// Verify faculty role (must be 'faculty', not already HOD of another dept)
	if facultyUser.Role != "faculty" && facultyUser.Role != "hod" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User is not a faculty member (current role: " + facultyUser.Role + ")"})
		return
	}

	// If faculty is already HOD of a different branch, reject
	if facultyUser.Role == "hod" && facultyUser.BranchID != nil && *facultyUser.BranchID != req.BranchID {
		c.JSON(http.StatusConflict, gin.H{"error": "This faculty is already HOD of another department"})
		return
	}

	// Verify faculty's branch matches the target department
	if facultyUser.BranchID == nil || *facultyUser.BranchID != req.BranchID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Faculty's branch does not match the target department"})
		return
	}

	// Verify the branch belongs to this college
	var branch models.Branch
	err = database.DB.Where("branch_id = ? AND college_id = ?", req.BranchID, *collegeIDPtr).First(&branch).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found in your college"})
		return
	}

	// Start transaction
	tx := database.DB.Begin()

	// Step 1: Demote current HOD (if any) back to faculty
	var currentHOD models.User
	err = tx.Where("role = ? AND branch_id = ? AND college_id = ? AND is_active = true",
		"hod", req.BranchID, *collegeIDPtr).First(&currentHOD).Error
	if err == nil && currentHOD.RegdNo != req.FacultyRegdNo {
		// Demote previous HOD to faculty and invalidate their session
		if err := tx.Model(&currentHOD).Updates(map[string]interface{}{
			"role":          "faculty",
			"token_version": gorm.Expr("token_version + 1"),
		}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to demote previous HOD"})
			return
		}
		log.Printf("Demoted previous HOD %s (%s) to faculty for branch %s",
			currentHOD.Name, currentHOD.RegdNo, branch.ShortName)
	}

	// Step 2: Promote faculty to HOD and invalidate their session
	if err := tx.Model(&facultyUser).Updates(map[string]interface{}{
		"role":          "hod",
		"branch_id":     req.BranchID,
		"token_version": gorm.Expr("token_version + 1"),
	}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign HOD role"})
		return
	}

	tx.Commit()

	log.Printf("Assigned %s (%s) as HOD for %s", facultyUser.Name, facultyUser.RegdNo, branch.ShortName)

	c.JSON(http.StatusOK, gin.H{
		"message":    "HOD assigned successfully",
		"hod_regdno": facultyUser.RegdNo,
		"hod_name":   facultyUser.Name,
		"department": branch.BranchName,
		"previous_hod": func() string {
			if currentHOD.RegdNo != "" && currentHOD.RegdNo != req.FacultyRegdNo {
				return currentHOD.Name + " (demoted to faculty)"
			}
			return "none"
		}(),
	})
}

// RemoveHOD removes HOD assignment, demoting them back to faculty
func RemoveHOD(c *gin.Context) {
	var req struct {
		BranchID uint `json:"branch_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "branch_id is required"})
		return
	}

	collegeID, _ := c.Get("college_id")
	if collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Principal has no college assigned"})
		return
	}
	collegeIDPtr := collegeID.(*string)

	// Find current HOD
	var currentHOD models.User
	err := database.DB.Where("role = ? AND branch_id = ? AND college_id = ? AND is_active = true",
		"hod", req.BranchID, *collegeIDPtr).First(&currentHOD).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No HOD found for this department"})
		return
	}

	// Demote to faculty and invalidate their session
	if err := database.DB.Model(&currentHOD).Updates(map[string]interface{}{
		"role":          "faculty",
		"token_version": gorm.Expr("token_version + 1"),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove HOD"})
		return
	}

	log.Printf("Removed HOD %s (%s) - demoted to faculty", currentHOD.Name, currentHOD.RegdNo)

	c.JSON(http.StatusOK, gin.H{
		"message":  "HOD removed successfully",
		"demoted":  currentHOD.Name,
		"new_role": "faculty",
	})
}
