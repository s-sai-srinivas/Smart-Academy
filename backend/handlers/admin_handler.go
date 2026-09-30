package handlers

import (
	"coding-platform/database"
	"coding-platform/middleware"
	"coding-platform/models"
	"coding-platform/services"
	"coding-platform/utils"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAdminStats returns dashboard statistics for admin
func GetAdminStats(c *gin.Context) {
	var totalCourses int64
	var totalLabSessions int64
	var totalProblems int64
	var totalStudents int64

	if middleware.IsSuperAdmin(c) {
		database.DB.Model(&models.Course{}).Count(&totalCourses)
		database.DB.Model(&models.LabSession{}).Count(&totalLabSessions)
		database.DB.Model(&models.Problem{}).Count(&totalProblems)
		database.DB.Model(&models.User{}).Where("role = ?", "student").Count(&totalStudents)
	} else {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
			return
		}
		database.DB.Model(&models.Course{}).Where("college_id = ?", *collegeID).Count(&totalCourses)
		database.DB.Model(&models.LabSession{}).
			Joins("INNER JOIN labs ON labs.id = lab_sessions.lab_id").
			Joins("INNER JOIN courses ON courses.id = labs.course_id").
			Where("courses.college_id = ?", *collegeID).
			Count(&totalLabSessions)
		database.DB.Model(&models.Problem{}).Where("college_id = ? OR is_global = ?", *collegeID, true).Count(&totalProblems)
		database.DB.Model(&models.User{}).Where("role = ? AND college_id = ?", "student", *collegeID).Count(&totalStudents)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_courses":      totalCourses,
		"total_lab_sessions": totalLabSessions,
		"total_problems":     totalProblems,
		"total_students":     totalStudents,
	})
}

// CreatePrincipal creates a principal user (admin only)
func CreatePrincipal(c *gin.Context) {
	var req struct {
		RegdNo   string `json:"regdno" binding:"required"`
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6,max=128"` // Password: min 6, max 128
		Phone    string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate password strength
	if err := utils.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get admin's college
	adminCollegeID, _ := c.Get("college_id")
	if adminCollegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Admin has no college assigned"})
		return
	}
	collegeIDPtr := adminCollegeID.(*string)

	// Check if a principal already exists for this college
	var existingPrincipal models.User
	err := database.DB.Where("role = ? AND college_id = ? AND is_active = true", "principal", *collegeIDPtr).First(&existingPrincipal).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A principal already exists for this college: " + existingPrincipal.Name})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := database.DB.Where("regdno = ? OR email = ?", req.RegdNo, req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this regdno or email already exists"})
		return
	}

	user := models.User{
		RegdNo:    req.RegdNo,
		Name:      req.Name,
		Email:     req.Email,
		Role:      "principal",
		CollegeID: collegeIDPtr,
		Phone:     req.Phone,
		IsActive:  true,
	}

	if err := user.HashPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create principal: " + err.Error()})
		return
	}

	log.Printf("Principal created: %s (%s) for college_id=%s", user.Name, user.RegdNo, *collegeIDPtr)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Principal created successfully",
		"user": gin.H{
			"regdno": user.RegdNo,
			"name":   user.Name,
			"email":  user.Email,
			"role":   user.Role,
		},
	})
}

// CreateCourse creates a new course (admin only)
// Batch-based: Course is abstract, no semester/batch fields
// Automatically creates entries in course_offerings AND curriculum_courses
// Uses transaction to ensure atomicity
func CreateCourse(c *gin.Context) {
	var req struct {
		CourseCode     string   `json:"course_code" binding:"required,min=2,max=20"`
		CourseName     string   `json:"course_name" binding:"required,min=2,max=200"`
		CourseType     string   `json:"course_type"`
		CourseCategory string   `json:"course_category"`
		Description    string   `json:"description"`
		Syllabus       string   `json:"syllabus"`
		Branches       []string `json:"branches"`
		Batches        []int    `json:"batches"`
		BranchIDs      []uint   `json:"branch_ids"`
		Credits        int      `json:"credits"`
		SemesterID     int      `json:"semester_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate course_type
	if req.CourseType == "" {
		req.CourseType = req.CourseCategory
	}
	if req.CourseType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_type (or course_category) is required"})
		return
	}
	if req.CourseType != "theory" && req.CourseType != "lab" && req.CourseType != "integrated" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "course_type must be 'theory', 'lab', or 'integrated'"})
		return
	}

	// Transform branch_ids to branches (short names) if provided
	if len(req.Branches) == 0 && len(req.BranchIDs) > 0 {
		var branches []models.Branch
		if err := database.DB.Where("branch_id IN ?", req.BranchIDs).Find(&branches).Error; err == nil {
			for _, b := range branches {
				req.Branches = append(req.Branches, b.ShortName)
			}
		}
	}

	// Transform semester_id to batches (cohort years) if provided
	// Use deterministic batch calculation based on academic year, not current time
	if len(req.Batches) == 0 && req.SemesterID > 0 {
		// Get current academic year for deterministic batch calculation
		var academicYear models.AcademicYear
		if err := database.DB.Where("is_current = ?", true).First(&academicYear).Error; err != nil {
			database.DB.Order("start_date DESC").First(&academicYear)
		}

		// Extract year from academic year name (e.g., "2024-25" -> 2024)
		// Or use the start_date year
		var baseYear int
		if academicYear.StartDate.Year() > 0 {
			baseYear = academicYear.StartDate.Year()
		} else {
			baseYear = time.Now().Year()
			if time.Now().Month() < 6 {
				baseYear--
			}
		}

		// Calculate offset based on semester
		// Semester 1-2 → current batch, 3-4 → current-1, 5-6 → current-2, 7-8 → current-3
		offset := (req.SemesterID - 1) / 2
		if offset > 3 {
			offset = 3
		}
		// Add current batch and one year before/after for flexibility
		for i := offset - 1; i <= offset+1; i++ {
			if i >= 0 && i <= 4 {
				req.Batches = append(req.Batches, baseYear-i)
			}
		}
	}

	userRegdNo, _ := c.Get("regdno")

	// Get admin's college for college-specific courses
	var adminUser models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&adminUser).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "User not found"})
		return
	}

	if adminUser.CollegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Admin must be assigned to a college first"})
		return
	}

	// Validate: Check for duplicate course_code within the college
	var existingCourse models.Course
	if err := database.DB.Where("course_code = ? AND college_id = ?", req.CourseCode, *adminUser.CollegeID).
		First(&existingCourse).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Course with code '%s' already exists in this college", req.CourseCode)})
		return
	}

	// Set default credits if not provided
	credits := req.Credits
	if credits == 0 {
		credits = 3
	}

	// Start transaction for atomic course creation
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Use defer to ensure rollback on error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	course := models.Course{
		CourseCode:  req.CourseCode,
		CourseName:  req.CourseName,
		CourseType:  req.CourseType,
		Credits:     credits,
		Description: req.Description,
		Syllabus:    req.Syllabus,
		CollegeID:   *adminUser.CollegeID,
		CreatedBy:   userRegdNo.(string),
		IsActive:    true,
	}

	if err := tx.Create(&course).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create course"})
		return
	}

	// Auto-create Lab or Theory record based on course_type
	if req.CourseType == "lab" {
		lab := models.Lab{
			CourseID: course.ID,
			LabName:  req.CourseName,
			LabCode:  req.CourseCode,
		}
		if err := tx.Create(&lab).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize lab"})
			return
		}
		tx.Preload("Lab").First(&course, course.ID)
	} else if req.CourseType == "theory" {
		theory := models.Theory{
			CourseID:   course.ID,
			TheoryName: req.CourseName,
			TheoryCode: req.CourseCode,
		}
		if err := tx.Create(&theory).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize theory"})
			return
		}
		tx.Preload("Theory").First(&course, course.ID)
	}

	// -------------------------------------------------------
	// Auto-create CourseOfferings for matching sections
	// -------------------------------------------------------
	offeringsCreated := 0
	offeringWarning := ""

	// Get current (or latest) academic year; auto-create one if none exists
	var academicYear models.AcademicYear
	if err := tx.Where("is_current = ?", true).First(&academicYear).Error; err != nil {
		tx.Order("start_date DESC").First(&academicYear)
	}
	if academicYear.AcademicYearID == 0 {
		// Auto-create a default academic year so offerings can be created
		now := time.Now()
		startYear := now.Year()
		if now.Month() < 6 {
			startYear--
		}
		academicYear = models.AcademicYear{
			Name:      fmt.Sprintf("%d-%02d", startYear, (startYear+1)%100),
			StartDate: time.Date(startYear, 6, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(startYear+1, 5, 31, 0, 0, 0, 0, time.UTC),
			IsCurrent: true,
		}
		if err := tx.Create(&academicYear).Error; err != nil {
			log.Printf("WARNING: Failed to auto-create academic year: %v", err)
		}
	}

	// SectionResult structure - declared outside the if block for wider scope
	type SectionResult struct {
		SectionID uint
		BranchID  uint
		ProgramID uint
	}
	var sectionResults []SectionResult

	if academicYear.AcademicYearID > 0 {
		// Find all matching sections
		query := tx.Table("sections s").
			Select("s.section_id, b.branch_id, b.program_id").
			Joins("INNER JOIN branches b ON b.branch_id = s.branch_id").
			Where("s.deleted_at IS NULL")

		if len(req.Branches) > 0 {
			query = query.Where("b.short_name IN ?", req.Branches)
		}
		if len(req.Batches) > 0 {
			query = query.Where("s.cohort_year IN ?", req.Batches)
		}
		query.Scan(&sectionResults)

		for _, sr := range sectionResults {
			// Skip if offering already exists for this course+section+year
			var existing models.CourseOffering
			err := tx.Where(
				"course_id = ? AND section_id = ? AND academic_year_id = ?",
				course.ID, sr.SectionID, academicYear.AcademicYearID,
			).First(&existing).Error
			if err != nil {
				offering := models.CourseOffering{
					CourseID:       course.ID,
					AcademicYearID: academicYear.AcademicYearID,
					SectionID:      sr.SectionID,
					IsActive:       true,
				}
				if createErr := tx.Create(&offering).Error; createErr == nil {
					offeringsCreated++

					// Auto-enroll all active students in this section using UPSERT
					// This prevents race conditions from concurrent requests
					tx.Exec(`
						INSERT INTO enrollments (student_regdno, course_offering_id, college_id, type, status, attempt_no, enrolled_at)
						SELECT s.regd_no, ?, u.college_id, 'Regular', 'enrolled', 1, NOW()
						FROM students s
						INNER JOIN users u ON u.regdno = s.regd_no
						WHERE s.section_id = ? AND s.status = 'active' AND u.college_id IS NOT NULL
						ON CONFLICT (student_regdno, course_offering_id) DO NOTHING
					`, offering.ID, sr.SectionID)
				} else {
					log.Printf("ERROR creating CourseOffering for section %d: %v", sr.SectionID, createErr)
				}
			}
		}
	} else {
		offeringWarning = "No academic year found; course offerings were not created automatically."
	}

	// Fallback: If no offerings were created (no matching sections), create a generic offering
	// This ensures the course is available for faculty assignment even without sections
	if offeringsCreated == 0 && academicYear.AcademicYearID > 0 {
		// Try to find any section in the college for a default offering
		var firstSection models.Section
		if err := tx.Where("branch_id IN (SELECT branch_id FROM branches WHERE college_id = ?)", *adminUser.CollegeID).
			Order("cohort_year DESC").
			First(&firstSection).Error; err == nil {
			offering := models.CourseOffering{
				CourseID:       course.ID,
				AcademicYearID: academicYear.AcademicYearID,
				SectionID:      firstSection.SectionID,
				IsActive:       true,
			}
			if err := tx.Create(&offering).Error; err == nil {
				offeringsCreated++
				log.Printf("Created fallback CourseOffering for course %d, section %d", course.ID, firstSection.SectionID)
			}
		}
	}

	// -------------------------------------------------------
	// Auto-create CurriculumCourses for matching curriculums
	// -------------------------------------------------------
	curriculumCoursesCreated := 0

	if academicYear.AcademicYearID > 0 && len(sectionResults) > 0 {
		// Get unique program IDs from the sections
		programIDs := make(map[uint]bool)
		for _, sr := range sectionResults {
			programIDs[sr.ProgramID] = true
		}

		// Find all active curriculums for these programs
		var curriculums []models.Curriculum
		tx.Where("program_id IN ? AND is_active = true",
			getKeysFromMap(programIDs)).Find(&curriculums)

		// Create curriculum_courses for each matching curriculum
		for _, curr := range curriculums {
			// Check if already exists
			var existingCC models.CurriculumCourse
			err := tx.Where(
				"curriculum_id = ? AND course_id = ?",
				curr.CurriculumID, course.ID,
			).First(&existingCC).Error

			if err != nil {
				// Set default category based on course_type
				category := "Core"
				if req.CourseType == "lab" {
					category = "Lab"
				} else if req.CourseType == "theory" {
					category = "Core"
				}

				cc := models.CurriculumCourse{
					CurriculumID:   curr.CurriculumID,
					CourseID:       course.ID,
					Category:       category,
					IsMandatory:    true,
					SequenceInYear: 0,
				}
				if err := tx.Create(&cc).Error; err == nil {
					curriculumCoursesCreated++
				}
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Build response message
	msg := fmt.Sprintf("Course created successfully. %d offering(s), %d curriculum mapping(s) created.", offeringsCreated, curriculumCoursesCreated)
	if offeringsCreated == 0 {
		msg = "Course created successfully. WARNING: No course offerings were created. Manually assign this course to a section before faculty can be assigned."
	}
	if offeringWarning != "" {
		msg = "Course created successfully. " + offeringWarning
	}

	c.JSON(http.StatusCreated, gin.H{
		"course":            course,
		"offerings_created": offeringsCreated,
		"message":           msg,
	})
}

// GetAdminCourses returns all courses for admin management
func GetAdminCourses(c *gin.Context) {
	var courses []models.Course
	query := database.DB.Preload("Lab").Preload("Theory")

	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
			return
		}
		query = query.Where("college_id = ?", *collegeID)
	}

	if err := query.Order("created_at DESC").Find(&courses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch courses"})
		return
	}

	c.JSON(http.StatusOK, courses)
}

// UpdateCourse updates an existing course
func UpdateCourse(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Credits    *int   `json:"credits"`
		IsActive   *bool  `json:"is_active"`
		CourseName string `json:"course_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var course models.Course
	if err := database.DB.First(&course, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Verify course belongs to admin's college
	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok || course.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: course belongs to another college"})
			return
		}
	}

	// Update fields
	if req.CourseName != "" {
		course.CourseName = req.CourseName
	}
	if req.Credits != nil {
		course.Credits = *req.Credits
	}
	if req.IsActive != nil {
		course.IsActive = *req.IsActive
	}

	if err := database.DB.Save(&course).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update course"})
		return
	}

	c.JSON(http.StatusOK, course)
}

// DeleteCourse permanently deletes a course and all related entities
func DeleteCourse(c *gin.Context) {
	id := c.Param("id")

	var course models.Course
	if err := database.DB.Unscoped().First(&course, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Verify course belongs to admin's college
	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok || course.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: course belongs to another college"})
			return
		}
	}

	// Use a transaction to cascade-delete all related data
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// 1. Delete Lab hierarchy: TestCases → Problems → LabSessions → Lab
	var lab models.Lab
	if err := tx.Unscoped().Where("course_id = ?", course.ID).First(&lab).Error; err == nil {
		// Get all lab session IDs
		var sessionIDs []uint
		tx.Unscoped().Model(&models.LabSession{}).Where("lab_id = ?", lab.ID).Pluck("id", &sessionIDs)

		if len(sessionIDs) > 0 {
			// Get all problem IDs in these sessions
			var problemIDs []uint
			tx.Unscoped().Model(&models.Problem{}).Where("lab_session_id IN ?", sessionIDs).Pluck("id", &problemIDs)

			if len(problemIDs) > 0 {
				// Delete test cases for these problems
				if err := tx.Unscoped().Where("problem_id IN ?", problemIDs).Delete(&models.TestCase{}).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete test cases"})
					return
				}
				// Delete problems
				if err := tx.Unscoped().Where("id IN ?", problemIDs).Delete(&models.Problem{}).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete problems"})
					return
				}
			}
			// Delete lab sessions
			if err := tx.Unscoped().Where("id IN ?", sessionIDs).Delete(&models.LabSession{}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete lab sessions"})
				return
			}
		}
		// Delete lab
		if err := tx.Unscoped().Delete(&lab).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete lab"})
			return
		}
	}

	// 2. Delete Theory hierarchy: TheoryModules → TheoryWeeks → Theory
	var theory models.Theory
	if err := tx.Unscoped().Where("course_id = ?", course.ID).First(&theory).Error; err == nil {
		var weekIDs []uint
		tx.Unscoped().Model(&models.TheoryWeek{}).Where("theory_id = ?", theory.ID).Pluck("id", &weekIDs)

		if len(weekIDs) > 0 {
			// Delete theory modules
			if err := tx.Unscoped().Where("theory_week_id IN ?", weekIDs).Delete(&models.TheoryModule{}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete theory modules"})
				return
			}
			// Delete theory weeks
			if err := tx.Unscoped().Where("id IN ?", weekIDs).Delete(&models.TheoryWeek{}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete theory weeks"})
				return
			}
		}
		// Delete theory
		if err := tx.Unscoped().Delete(&theory).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete theory"})
			return
		}
	}

	// 3. Delete CourseOfferings and their Enrollments
	var offeringIDs []uint
	tx.Unscoped().Model(&models.CourseOffering{}).Where("course_id = ?", course.ID).Pluck("id", &offeringIDs)
	if len(offeringIDs) > 0 {
		// Delete faculty assignments for these offerings
		if err := tx.Unscoped().Where("course_offering_id IN ?", offeringIDs).Delete(&models.FacultyAssignment{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete faculty assignments"})
			return
		}
		if err := tx.Unscoped().Where("course_offering_id IN ?", offeringIDs).Delete(&models.Enrollment{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete enrollments"})
			return
		}
		if err := tx.Unscoped().Where("id IN ?", offeringIDs).Delete(&models.CourseOffering{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete course offerings"})
			return
		}
	}

	// 4. Delete CurriculumCourses
	if err := tx.Unscoped().Where("course_id = ?", course.ID).Delete(&models.CurriculumCourse{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete curriculum courses"})
		return
	}

	// 5. Delete the course itself
	if err := tx.Unscoped().Delete(&course).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete course"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Course and all related data deleted successfully"})
}

// AssignCourseBranches creates course offerings for the given branches/batches
// This replaces any existing offerings for the course
func AssignCourseBranches(c *gin.Context) {
	courseID := c.Param("id")

	var req struct {
		Branches []string `json:"branches"` // branch short names
		Batches  []int    `json:"batches"`  // cohort years
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var course models.Course
	if err := database.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Verify course belongs to admin's college
	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok || course.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: course belongs to another college"})
			return
		}
	}

	// Get current academic year
	var academicYear models.AcademicYear
	if err := database.DB.Where("is_current = ?", true).First(&academicYear).Error; err != nil {
		database.DB.Order("start_date DESC").First(&academicYear)
	}
	if academicYear.AcademicYearID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No academic year found"})
		return
	}

	// Delete old offerings (and their enrollments) for this course
	var oldOfferingIDs []uint
	database.DB.Table("course_offerings").
		Select("id").
		Where("course_id = ? AND deleted_at IS NULL", course.ID).
		Pluck("id", &oldOfferingIDs)

	if len(oldOfferingIDs) > 0 {
		database.DB.Where("course_offering_id IN ?", oldOfferingIDs).Delete(&models.Enrollment{})
		database.DB.Where("course_offering_id IN ?", oldOfferingIDs).Delete(&models.FacultyAssignment{})
		database.DB.Where("course_id = ?", course.ID).Delete(&models.CourseOffering{})
	}

	// Find matching sections
	type SectionResult struct {
		SectionID uint
	}
	var sectionResults []SectionResult

	query := database.DB.Table("sections s").
		Select("s.section_id").
		Joins("INNER JOIN branches b ON b.branch_id = s.branch_id").
		Where("s.deleted_at IS NULL")

	// Restrict to admin's college branches
	if !middleware.IsSuperAdmin(c) {
		query = query.Where("b.college_id = ?", course.CollegeID)
	}

	if len(req.Branches) > 0 {
		query = query.Where("b.short_name IN ?", req.Branches)
	}
	if len(req.Batches) > 0 {
		query = query.Where("s.cohort_year IN ?", req.Batches)
	}
	query.Scan(&sectionResults)

	// Create offerings and enroll students
	offeringsCreated := 0
	enrollmentsCreated := 0
	for _, sr := range sectionResults {
		offering := models.CourseOffering{
			CourseID:       course.ID,
			AcademicYearID: academicYear.AcademicYearID,
			SectionID:      sr.SectionID,
			IsActive:       true,
		}
		if err := database.DB.Create(&offering).Error; err == nil {
			offeringsCreated++

			// Auto-enroll active students in this section
			var studentRegdNos []string
			database.DB.Table("students").
				Select("regd_no").
				Where("section_id = ? AND status = ?", sr.SectionID, "active").
				Pluck("regd_no", &studentRegdNos)

			for _, regdNo := range studentRegdNos {
				database.DB.Create(&models.Enrollment{
					StudentRegdNo:    regdNo,
					CourseOfferingID: offering.ID,
					Type:             "Regular",
					Status:           "enrolled",
					AttemptNo:        1,
				})
				enrollmentsCreated++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             fmt.Sprintf("Assigned %d section(s), enrolled %d student(s)", offeringsCreated, enrollmentsCreated),
		"offerings_created":   offeringsCreated,
		"enrollments_created": enrollmentsCreated,
	})
}

// GetCourseBranches returns the branches a course is currently offered to
func GetCourseBranches(c *gin.Context) {
	courseID := c.Param("id")

	type BranchInfo struct {
		ShortName  string `json:"short_name"`
		BranchName string `json:"branch_name"`
		BranchID   uint   `json:"branch_id"`
		CohortYear int    `json:"cohort_year"`
	}

	var branches []BranchInfo
	database.DB.Raw(`
		SELECT DISTINCT b.branch_id, b.short_name, b.branch_name, s.cohort_year
		FROM course_offerings co
		INNER JOIN sections s ON s.section_id = co.section_id
		INNER JOIN branches b ON b.branch_id = s.branch_id
		WHERE co.course_id = ? AND co.is_active = true AND co.deleted_at IS NULL
		ORDER BY b.short_name, s.cohort_year
	`, courseID).Scan(&branches)

	c.JSON(http.StatusOK, branches)
}

// CreateLab creates a new lab for a course
func CreateLab(c *gin.Context) {
	var req struct {
		LabName  string `json:"lab_name" binding:"required"`
		LabCode  string `json:"lab_code" binding:"required"`
		CourseID uint   `json:"course_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify course exists and is a lab course
	var course models.Course
	if err := database.DB.First(&course, req.CourseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Verify course type is 'lab'
	if course.CourseType != "lab" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only create lab for courses with type 'lab'"})
		return
	}

	// Verify course belongs to admin's college
	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok || course.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: course belongs to another college"})
			return
		}
	}

	// Check if lab already exists for this course
	var count int64
	database.DB.Model(&models.Lab{}).Where("course_id = ?", req.CourseID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lab already exists for this course"})
		return
	}

	// Check for duplicate lab_code
	var existingLab models.Lab
	if err := database.DB.Where("lab_code = ?", req.LabCode).First(&existingLab).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Lab code '%s' already exists", req.LabCode)})
		return
	}

	lab := models.Lab{
		CourseID: req.CourseID,
		LabName:  req.LabName,
		LabCode:  req.LabCode,
	}

	if err := database.DB.Create(&lab).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create lab"})
		return
	}

	c.JSON(http.StatusCreated, lab)
}

// CreateLabSession creates a new session within a lab
func CreateLabSession(c *gin.Context) {
	var req struct {
		SessionName  string `json:"session_name" binding:"required"`
		TopicName    string `json:"topic_name"`
		LabID        uint   `json:"lab_id" binding:"required"`
		SessionOrder int    `json:"session_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify lab exists
	var lab models.Lab
	if err := database.DB.First(&lab, req.LabID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lab not found"})
		return
	}

	// Verify lab's course belongs to admin's college
	if !middleware.IsSuperAdmin(c) {
		var course models.Course
		if err := database.DB.First(&course, lab.CourseID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify lab ownership"})
			return
		}
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok || course.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: lab belongs to another college"})
			return
		}
	}

	// Validate topic exists in the canonical topics table (if provided)
	topicName := strings.TrimSpace(req.TopicName)
	if topicName != "" {
		var topic models.Topic
		if err := database.DB.Where("LOWER(name) = LOWER(?)", topicName).First(&topic).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Topic '%s' does not exist. Please select a valid topic from the list or contact super admin to create it.", topicName),
			})
			return
		}
		// Normalize to the canonical topic name
		topicName = topic.Name
	}

	session := models.LabSession{
		LabID:        req.LabID,
		SessionName:  req.SessionName,
		SessionOrder: req.SessionOrder,
		TopicName:    topicName,
	}

	if err := database.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create lab session"})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// GetCourseLabSessions returns sessions for a specific lab
func GetCourseLabSessions(c *gin.Context) {
	labID := c.Param("id")

	var sessions []models.LabSession
	if err := database.DB.Where("lab_id = ?", labID).
		Preload("Problems").
		Order("session_order").
		Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sessions"})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// TestCaseRequest represents a test case for problem creation
type TestCaseRequest struct {
	Input          string `json:"input" binding:"required"`           // Size validated in handler
	ExpectedOutput string `json:"expected_output" binding:"required"` // Size validated in handler
	IsSample       bool   `json:"is_sample"`
	Points         int    `json:"points"`
}

// CreateAdminProblem creates a new problem with test cases
func CreateAdminProblem(c *gin.Context) {
	var req struct {
		LabSessionID *uint  `json:"lab_session_id"`
		Title        string `json:"title" binding:"required,min=3,max=300"`
		Description  string `json:"description" binding:"required"`
		Difficulty   string `json:"difficulty" binding:"required,oneof=easy medium hard"`
		// Tags removed for lab-scoped problem creation (handled for global problems elsewhere)
		TestCases   []TestCaseRequest `json:"test_cases" binding:"required,min=1,max=50"`
		TimeLimit   int               `json:"time_limit"`
		MemoryLimit int               `json:"memory_limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate sizes
	const maxTestCaseInputSize = 10 * 1024  // 10KB
	const maxTestCaseOutputSize = 10 * 1024 // 10KB
	const maxDescriptionSize = 50 * 1024    // 50KB

	if len(req.Description) > maxDescriptionSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Description exceeds maximum size of 50KB"})
		return
	}

	for i, tc := range req.TestCases {
		if len(tc.Input) > maxTestCaseInputSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Test case %d input exceeds maximum size of 10KB", i+1)})
			return
		}
		if len(tc.ExpectedOutput) > maxTestCaseOutputSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Test case %d expected output exceeds maximum size of 10KB", i+1)})
			return
		}
	}

	userRegdNo, _ := c.Get("regdno")
	userCollegeID, _ := c.Get("college_id")

	// Type assertion for college_id (now *string from context)
	var collegeIDPtr *string
	if userCollegeID != nil {
		if idPtr, ok := userCollegeID.(*string); ok {
			collegeIDPtr = idPtr
		}
	}

	// Create problem
	problem := models.Problem{
		Title:       req.Title,
		Description: req.Description,
		Difficulty:  req.Difficulty,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		// Do not set Tags for lab problems (UI doesn't send tags for lab problems)
		Tags:         "",
		LabSessionID: req.LabSessionID,
		CollegeID:    collegeIDPtr,
		CreatedBy:    userRegdNo.(string),
		IsGlobal:     false,
	}

	if err := database.DB.Create(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create problem"})
		return
	}

	// Create test cases
	for _, tc := range req.TestCases {
		testCase := models.TestCase{
			ProblemID:      problem.ID,
			Input:          tc.Input,
			ExpectedOutput: tc.ExpectedOutput,
			IsSample:       tc.IsSample,
			Points:         tc.Points,
		}
		if err := database.DB.Create(&testCase).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create test case"})
			return
		}
	}

	database.DB.Preload("TestCases").First(&problem, problem.ID)
	c.JSON(http.StatusCreated, problem)
}

// GetAdminProblems returns problems for admin management
func GetAdminProblems(c *gin.Context) {
	var problems []models.Problem
	query := database.DB.Preload("TestCases").Preload("LabSession")

	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
			return
		}
		query = query.Where("college_id = ? OR is_global = ?", *collegeID, true)
	}

	if err := query.Order("created_at DESC").Find(&problems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch problems"})
		return
	}

	c.JSON(http.StatusOK, problems)
}

// AddProblemToSession adds a problem to a lab session context
func AddProblemToSession(c *gin.Context) {
	var req struct {
		ProblemID uint `json:"problem_id" binding:"required"`
		SessionID uint `json:"session_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var problem models.Problem
	if err := database.DB.First(&problem, req.ProblemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	var session models.LabSession
	if err := database.DB.First(&session, req.SessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Verify session's lab -> course belongs to admin's college
	if !middleware.IsSuperAdmin(c) {
		// LabID is now required (non-nullable), so we always check
		var lab models.Lab
		if err := database.DB.First(&lab, session.LabID).Error; err == nil {
			var course models.Course
			if err := database.DB.First(&course, lab.CourseID).Error; err == nil {
				collegeID, ok := middleware.GetCurrentUserCollege(c)
				if !ok || course.CollegeID != *collegeID {
					c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: session belongs to another college"})
					return
				}
			}
		}
	}

	// Update problem
	problem.LabSessionID = &req.SessionID
	if err := database.DB.Save(&problem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link problem to session"})
		return
	}

	c.JSON(http.StatusOK, problem)
}

// GetRegulations returns all regulations
func GetRegulations(c *gin.Context) {
	var regulations []models.Regulation
	database.DB.Find(&regulations)
	c.JSON(http.StatusOK, regulations)
}

// GetPrograms returns all programs
func GetPrograms(c *gin.Context) {
	var programs []models.Program
	database.DB.Find(&programs)
	c.JSON(http.StatusOK, programs)
}

// GetCurriculums returns all curriculums
func GetCurriculums(c *gin.Context) {
	var curriculums []models.Curriculum
	database.DB.Preload("Program").Preload("Regulation").Find(&curriculums)
	c.JSON(http.StatusOK, curriculums)
}

// GetBranches returns all branches
func GetBranches(c *gin.Context) {
	var branches []models.Branch
	query := database.DB.Preload("Program").Preload("College")

	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
			return
		}
		query = query.Where("college_id = ?", *collegeID)
	}

	query.Find(&branches)
	c.JSON(http.StatusOK, branches)
}

// GetSections returns all sections (batch-based)
func GetSections(c *gin.Context) {
	var sections []models.Section
	query := database.DB

	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
			return
		}
		query = query.Joins("INNER JOIN branches ON branches.branch_id = sections.branch_id").
			Where("branches.college_id = ?", *collegeID)
	}

	query.Preload("Branch").Preload("AcademicYear").Find(&sections)
	c.JSON(http.StatusOK, sections)
}

// GetSemesters returns all academic years (equivalent to semesters in batch-based system)
func GetSemesters(c *gin.Context) {
	var academicYears []models.AcademicYear
	database.DB.Find(&academicYears)
	c.JSON(http.StatusOK, academicYears)
}

// GetBatches returns distinct cohort years from sections
func GetBatches(c *gin.Context) {
	var cohortYears []int
	query := database.DB.Table("sections").
		Where("deleted_at IS NULL")

	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
			return
		}
		query = query.Joins("INNER JOIN branches ON branches.branch_id = sections.branch_id").
			Where("branches.college_id = ?", *collegeID)
	}

	query.Distinct("cohort_year").
		Order("cohort_year DESC").
		Pluck("cohort_year", &cohortYears)
	c.JSON(http.StatusOK, cohortYears)
}

// GetCourseLab returns the lab details for a specific course
func GetCourseLab(c *gin.Context) {
	courseID := c.Param("id")

	// Verify course belongs to admin's college
	if !middleware.IsSuperAdmin(c) {
		var course models.Course
		if err := database.DB.First(&course, courseID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
			return
		}
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok || course.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: course belongs to another college"})
			return
		}
	}

	var lab models.Lab
	if err := database.DB.Where("course_id = ?", courseID).First(&lab).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lab not found for this course"})
		return
	}

	c.JSON(http.StatusOK, lab)
}

// CreateTheory creates a new theory for a course
func CreateTheory(c *gin.Context) {
	var req struct {
		TheoryName string `json:"theory_name" binding:"required"`
		TheoryCode string `json:"theory_code" binding:"required"`
		CourseID   uint   `json:"course_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify course exists and is a theory course
	var course models.Course
	if err := database.DB.First(&course, req.CourseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Verify course type is 'theory'
	if course.CourseType != "theory" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only create theory for courses with type 'theory'"})
		return
	}

	// Verify course belongs to admin's college
	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok || course.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: course belongs to another college"})
			return
		}
	}

	// Check if theory already exists for this course
	var count int64
	database.DB.Model(&models.Theory{}).Where("course_id = ?", req.CourseID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Theory already exists for this course"})
		return
	}

	// Check for duplicate theory_code
	var existingTheory models.Theory
	if err := database.DB.Where("theory_code = ?", req.TheoryCode).First(&existingTheory).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Theory code '%s' already exists", req.TheoryCode)})
		return
	}

	theory := models.Theory{
		CourseID:   req.CourseID,
		TheoryName: req.TheoryName,
		TheoryCode: req.TheoryCode,
	}

	if err := database.DB.Create(&theory).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create theory"})
		return
	}

	c.JSON(http.StatusCreated, theory)
}

// GetCourseTheory returns the theory details for a specific course
// Returns 404 if theory doesn't exist - use POST /courses/:id/theory to create
func GetCourseTheory(c *gin.Context) {
	courseID := c.Param("id")

	// Verify course belongs to admin's college
	if !middleware.IsSuperAdmin(c) {
		var courseCheck models.Course
		if err := database.DB.First(&courseCheck, courseID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
			return
		}
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok || courseCheck.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: course belongs to another college"})
			return
		}
	}

	var theory models.Theory
	if err := database.DB.Where("course_id = ?", courseID).
		Preload("Weeks", func(db *gorm.DB) *gorm.DB {
			return db.Order("week_order ASC")
		}).
		Preload("Weeks.Modules").
		First(&theory).Error; err != nil {
		// Theory not found - return 404 instead of auto-creating
		c.JSON(http.StatusNotFound, gin.H{"error": "Theory not found for this course. Create it using POST /courses/:id/theory"})
		return
	}

	c.JSON(http.StatusOK, theory)
}

// CreateTheoryWeek adds a week to a theory
func CreateTheoryWeek(c *gin.Context) {
	var req struct {
		WeekName  string `json:"week_name" binding:"required"`
		TheoryID  uint   `json:"theory_id" binding:"required"`
		WeekOrder int    `json:"week_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify theory exists
	var theory models.Theory
	if err := database.DB.First(&theory, req.TheoryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Theory not found"})
		return
	}

	week := models.TheoryWeek{
		TheoryID:  req.TheoryID,
		WeekName:  req.WeekName,
		WeekOrder: req.WeekOrder,
	}

	if err := database.DB.Create(&week).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create theory week"})
		return
	}

	c.JSON(http.StatusCreated, week)
}

// CreateTheoryModule adds a module to a week
func CreateTheoryModule(c *gin.Context) {
	var req struct {
		ModuleName   string `json:"module_name" binding:"required"`
		Description  string `json:"description"`
		Content      string `json:"content"`
		TheoryWeekID uint   `json:"theory_week_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify week exists
	var week models.TheoryWeek
	if err := database.DB.First(&week, req.TheoryWeekID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Theory week not found"})
		return
	}

	module := models.TheoryModule{
		TheoryWeekID: req.TheoryWeekID,
		ModuleName:   req.ModuleName,
		Description:  req.Description,
		Content:      req.Content,
	}

	if err := database.DB.Create(&module).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create theory module"})
		return
	}

	c.JSON(http.StatusCreated, module)
}

// UpdateTheoryModule updates an existing theory module
func UpdateTheoryModule(c *gin.Context) {
	moduleID := c.Param("id")

	var existing models.TheoryModule
	if err := database.DB.First(&existing, moduleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	var req struct {
		ModuleName  string `json:"module_name"`
		Description string `json:"description"`
		Content     string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ModuleName != "" {
		existing.ModuleName = req.ModuleName
	}
	existing.Description = req.Description
	existing.Content = req.Content

	if err := database.DB.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update module"})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteTheoryWeek deletes a week and all its modules
func DeleteTheoryWeek(c *gin.Context) {
	weekID := c.Param("id")

	// Verify week exists and get modules count
	var week models.TheoryWeek
	if err := database.DB.First(&week, weekID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Week not found"})
		return
	}

	// Delete the week (modules will be cascade deleted)
	if err := database.DB.Delete(&week).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete week"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Week deleted successfully"})
}

// DeleteTheoryModule deletes a module
func DeleteTheoryModule(c *gin.Context) {
	moduleID := c.Param("id")

	// Verify module exists
	var module models.TheoryModule
	if err := database.DB.First(&module, moduleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Module not found"})
		return
	}

	// Delete the module
	if err := database.DB.Delete(&module).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete module"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Module deleted successfully"})
}

// ==================== THEORY PDF MANAGEMENT ====================

// UploadTheoryPDF uploads a PDF file for a theory course
func UploadTheoryPDF(c *gin.Context) {
	theoryIDStr := c.PostForm("theory_id")
	if theoryIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "theory_id is required"})
		return
	}
	theoryID, err := strconv.ParseUint(theoryIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theory_id"})
		return
	}

	var moduleID *uint
	if moduleIDStr := c.PostForm("theory_module_id"); moduleIDStr != "" {
		mid, err := strconv.ParseUint(moduleIDStr, 10, 64)
		if err == nil {
			midUint := uint(mid)
			moduleID = &midUint
		}
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF files are allowed"})
		return
	}

	const maxSize = 50 * 1024 * 1024
	if header.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large. Maximum size is 50MB"})
		return
	}

	filePath, err := services.SaveUploadedFile(header, "theory-pdfs")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	displayName := c.PostForm("display_name")
	if displayName == "" {
		displayName = header.Filename
	}

	pdf := models.TheoryPDF{
		TheoryID:       uint(theoryID),
		TheoryModuleID: moduleID,
		FileName:       header.Filename,
		DisplayName:    displayName,
		FilePath:       filePath,
		FileSize:       header.Size,
	}

	if err := database.DB.Create(&pdf).Error; err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save PDF record"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"pdf": pdf, "message": "PDF uploaded successfully"})
}

// GetTheoryPDFs returns all PDFs for a theory course
func GetTheoryPDFs(c *gin.Context) {
	theoryID := c.Param("id")

	var pdfs []models.TheoryPDF
	if err := database.DB.Where("theory_id = ?", theoryID).
		Preload("TheoryModule").
		Order("created_at DESC").
		Find(&pdfs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch PDFs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pdfs": pdfs})
}

// DeleteTheoryPDF deletes a theory PDF record and its file
func DeleteTheoryPDF(c *gin.Context) {
	pdfID := c.Param("id")

	var pdf models.TheoryPDF
	if err := database.DB.First(&pdf, pdfID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PDF not found"})
		return
	}

	if err := database.DB.Delete(&pdf).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete PDF record"})
		return
	}

	if pdf.FilePath != "" {
		os.Remove(pdf.FilePath)
	}

	c.JSON(http.StatusOK, gin.H{"message": "PDF deleted successfully"})
}

// ServeTheoryPDF serves a theory PDF file for download
func ServeTheoryPDF(c *gin.Context) {
	pdfID := c.Param("id")

	var pdf models.TheoryPDF
	if err := database.DB.First(&pdf, pdfID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PDF not found"})
		return
	}

	if _, err := os.Stat(pdf.FilePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "PDF file not found on disk"})
		return
	}

	c.Header("Content-Disposition", "inline; filename=\""+pdf.FileName+"\"")
	c.Header("Content-Type", "application/pdf")
	c.File(pdf.FilePath)
}

// ==================== CURRICULUM MANAGEMENT ====================

// CreateCurriculumRequest creates a new curriculum
type CreateCurriculumRequest struct {
	RegulationCode string `json:"regulation_code" binding:"required"`
	ProgramID      uint   `json:"program_id" binding:"required"`
	DurationYears  int    `json:"duration_years"`
}

// CreateCurriculum creates a new curriculum with regulation
func CreateCurriculum(c *gin.Context) {
	var req CreateCurriculumRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify program exists
	var program models.Program
	if err := database.DB.First(&program, req.ProgramID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Program not found"})
		return
	}

	// Set default duration
	durationYears := req.DurationYears
	if durationYears == 0 {
		durationYears = 4
	}

	// Create or get Regulation
	var regulation models.Regulation
	database.DB.Where("code = ?", req.RegulationCode).FirstOrCreate(&regulation, models.Regulation{
		Code:                req.RegulationCode,
		EffectiveFromCohort: 2024, // Default, can be updated
		IsActive:            true,
	})

	// Create Curriculum
	var curriculum models.Curriculum
	err := database.DB.Where("program_id = ? AND regulation_id = ?", req.ProgramID, regulation.RegulationID).
		First(&curriculum).Error

	if err != nil {
		// Create new curriculum
		curriculum = models.Curriculum{
			ProgramID:     req.ProgramID,
			RegulationID:  regulation.RegulationID,
			DurationYears: durationYears,
			IsActive:      true,
		}
		if err := database.DB.Create(&curriculum).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create curriculum"})
			return
		}
	}

	database.DB.Preload("Program").Preload("Regulation").First(&curriculum, curriculum.CurriculumID)
	c.JSON(http.StatusCreated, curriculum)
}

// AddCourseToCurriculum adds a course to a curriculum
type AddCourseToCurriculumRequest struct {
	Category       string `json:"category"`
	CurriculumID   uint   `json:"curriculum_id" binding:"required"`
	CourseID       uint   `json:"course_id" binding:"required"`
	SequenceInYear int    `json:"sequence_in_year"`
	IsMandatory    bool   `json:"is_mandatory"`
}

func AddCourseToCurriculum(c *gin.Context) {
	var req AddCourseToCurriculumRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify curriculum and course exist
	var curriculum models.Curriculum
	if err := database.DB.First(&curriculum, req.CurriculumID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Curriculum not found"})
		return
	}

	var course models.Course
	if err := database.DB.First(&course, req.CourseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Create mapping
	cc := models.CurriculumCourse{
		CurriculumID:   req.CurriculumID,
		CourseID:       req.CourseID,
		Category:       req.Category,
		IsMandatory:    req.IsMandatory,
		SequenceInYear: req.SequenceInYear,
	}

	if err := database.DB.Create(&cc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add course to curriculum"})
		return
	}

	c.JSON(http.StatusCreated, cc)
}

// GetCurriculumCourses returns all courses for a curriculum
func GetCurriculumCourses(c *gin.Context) {
	curriculumID := c.Param("id")

	var courses []models.CurriculumCourse
	database.DB.Preload("Course").
		Where("curriculum_id = ?", curriculumID).
		Order("sequence_in_year").
		Find(&courses)

	c.JSON(http.StatusOK, courses)
}

// RemoveCourseFromCurriculum removes a course from a curriculum
func RemoveCourseFromCurriculum(c *gin.Context) {
	curriculumID := c.Param("id")
	courseID := c.Param("courseId")

	// Verify curriculum exists
	var curriculum models.Curriculum
	if err := database.DB.First(&curriculum, curriculumID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Curriculum not found"})
		return
	}

	// Delete the curriculum_course mapping
	result := database.DB.Where("curriculum_id = ? AND course_id = ?", curriculumID, courseID).
		Delete(&models.CurriculumCourse{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove course from curriculum"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found in this curriculum"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Course removed from curriculum successfully"})
}

// ==================== COURSE OFFERING MANAGEMENT ====================

// CreateCourseOfferingRequest creates a new course offering
type CreateCourseOfferingRequest struct {
	CourseID       uint `json:"course_id" binding:"required"`
	CurriculumID   uint `json:"curriculum_id"`
	AcademicYearID uint `json:"academic_year_id" binding:"required"`
	SectionID      uint `json:"section_id" binding:"required"`
}

// CreateCourseOffering creates a new course offering
func CreateCourseOffering(c *gin.Context) {
	var req CreateCourseOfferingRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify all related records exist
	var course models.Course
	if err := database.DB.First(&course, req.CourseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Verify course belongs to admin's college
	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok || course.CollegeID != *collegeID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: course belongs to another college"})
			return
		}
	}

	var curriculum models.Curriculum
	if err := database.DB.First(&curriculum, req.CurriculumID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Curriculum not found"})
		return
	}

	var academicYear models.AcademicYear
	if err := database.DB.First(&academicYear, req.AcademicYearID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Academic year not found"})
		return
	}

	var section models.Section
	if err := database.DB.First(&section, req.SectionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Section not found"})
		return
	}

	// Create CourseOffering
	var curriculumIDPtr *uint
	if req.CurriculumID != 0 {
		curriculumIDPtr = &req.CurriculumID
	}
	offering := models.CourseOffering{
		CourseID:       req.CourseID,
		CurriculumID:   curriculumIDPtr,
		AcademicYearID: req.AcademicYearID,
		SectionID:      req.SectionID,
		IsActive:       true,
	}

	if err := database.DB.Create(&offering).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create course offering"})
		return
	}

	database.DB.Preload("Course").Preload("Curriculum").Preload("AcademicYear").Preload("Section").
		First(&offering, offering.ID)

	c.JSON(http.StatusCreated, offering)
}

// GetCourseOfferings returns all course offerings
func GetCourseOfferings(c *gin.Context) {
	var offerings []models.CourseOffering

	// Optional filters
	curriculumID := c.Query("curriculum_id")
	academicYearID := c.Query("academic_year_id")
	sectionID := c.Query("section_id")

	query := database.DB.Preload("Course").Preload("Curriculum").Preload("AcademicYear").Preload("Section")

	// Restrict to admin's college
	if !middleware.IsSuperAdmin(c) {
		collegeID, ok := middleware.GetCurrentUserCollege(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "No college assigned"})
			return
		}
		query = query.Joins("INNER JOIN courses ON courses.id = course_offerings.course_id").
			Where("courses.college_id = ?", *collegeID)
	}

	if curriculumID != "" {
		query = query.Where("course_offerings.curriculum_id = ?", curriculumID)
	}
	if academicYearID != "" {
		query = query.Where("course_offerings.academic_year_id = ?", academicYearID)
	}
	if sectionID != "" {
		query = query.Where("course_offerings.section_id = ?", sectionID)
	}

	if err := query.Order("course_offerings.created_at DESC").Find(&offerings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch course offerings"})
		return
	}

	c.JSON(http.StatusOK, offerings)
}

// ==================== STUDENT ONBOARDING (Batch-Based) ====================

// StudentBulkUploadResult represents the result of bulk student upload
type StudentBulkUploadResult struct {
	Errors    []StudentUploadError `json:"errors,omitempty"`
	TotalRows int                  `json:"total_rows"`
	Created   int                  `json:"created"`
	Updated   int                  `json:"updated"`
	Failed    int                  `json:"failed"`
}

// StudentUploadError represents an error for a specific row
type StudentUploadError struct {
	RegdNo string `json:"regdno"`
	Error  string `json:"error"`
	Row    int    `json:"row"`
}

// StudentCSVRow represents a row from the uploaded CSV (batch-based)
type StudentCSVRow struct {
	RegdNo        string `csv:"regdno"`
	Name          string `csv:"name"`
	Email         string `csv:"email"`
	PhoneNumber   string `csv:"phone_number"`
	Program       string `csv:"program"`
	Branch        string `csv:"branch"`
	SectionName   string `csv:"section"`
	Role          string `csv:"role"`
	CohortYear    int    `csv:"cohort_year"`
	AdmissionYear int    `csv:"admission_year"`
}

// DownloadStudentTemplate returns a CSV template for student onboarding (batch-based)
func DownloadStudentTemplate(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=student_onboarding_template.csv")

	// CSV headers (batch-based, no semester)
	headers := []string{
		"regdno", "name", "email", "phone_number",
		"program", "branch", "cohort_year", "admission_year",
		"section", "role",
	}

	// Sample data rows
	sampleData := [][]string{
		headers,
		{"24CSE001", "John Doe", "john@example.com", "9876543210", "B.Tech", "CSE", "2024", "2024", "A", "student"},
		{"24CSE002", "Jane Smith", "jane@example.com", "9876543211", "B.Tech", "CSE", "2024", "2024", "B", "student"},
	}

	// Build CSV content
	csvContent := ""
	for _, row := range sampleData {
		csvContent += "\"" + row[0] + "\""
		for i := 1; i < len(row); i++ {
			csvContent += ",\"" + row[i] + "\""
		}
		csvContent += "\r\n"
	}

	c.String(http.StatusOK, csvContent)
}

// BulkUploadStudents handles bulk student upload via CSV (batch-based)
func BulkUploadStudents(c *gin.Context) {
	// Get admin's college for validation
	adminCollegeID, _ := c.Get("college_id")

	// Parse multipart form
	file, err := c.FormFile("csv_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	// Parse CSV
	rows, err := parseStudentCSV(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse CSV: " + err.Error()})
		return
	}

	// Get or create default Regulation and Curriculum for new students
	var regulation models.Regulation
	database.DB.Where("code = ?", "R2024").FirstOrCreate(&regulation, models.Regulation{
		Code:                "R2024",
		EffectiveFromCohort: 2024,
		IsActive:            true,
	})

	// Process each row
	result := StudentBulkUploadResult{
		TotalRows: len(rows),
		Errors:    []StudentUploadError{},
	}

	// First pass: Check for duplicate emails within the CSV file
	emailToRow := make(map[string]int) // email -> first row number where it appeared
	for i, row := range rows {
		if row.Email != "" {
			emailLower := strings.ToLower(row.Email)
			if firstRow, exists := emailToRow[emailLower]; exists {
				result.Failed++
				result.Errors = append(result.Errors, StudentUploadError{
					Row:    i + 2,
					RegdNo: row.RegdNo,
					Error:  fmt.Sprintf("duplicate email '%s' (first appeared in row %d)", row.Email, firstRow),
				})
			} else {
				emailToRow[emailLower] = i + 2
			}
		}
	}

	// Second pass: Validate and process each row
	for i, row := range rows {
		rowNum := i + 2 // +2 because header is row 1

		// Skip if already marked as failed due to duplicate email
		isDuplicate := false
		for _, err := range result.Errors {
			if err.Row == rowNum {
				isDuplicate = true
				break
			}
		}
		if isDuplicate {
			continue
		}

		// Validate required fields
		if row.RegdNo == "" {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "regdno is required"})
			continue
		}
		if row.Name == "" {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "name is required"})
			continue
		}
		if row.Email == "" {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "email is required"})
			continue
		}
		if row.Program == "" {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "program is required"})
			continue
		}
		if row.Branch == "" {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "branch is required"})
			continue
		}
		if row.CohortYear == 0 {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "cohort_year is required"})
			continue
		}
		if row.AdmissionYear == 0 {
			row.AdmissionYear = row.CohortYear // Default to cohort year
		}
		if row.SectionName == "" {
			row.SectionName = "A" // Default section
		}
		if row.Role == "" {
			row.Role = "student"
		}

		// Use admin's college directly
		var college models.College
		if adminCollegeID == nil {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "admin has no college assigned"})
			continue
		}
		adminCollegeIDPtr := adminCollegeID.(*string)
		err := database.DB.First(&college, "college_id = ?", *adminCollegeIDPtr).Error
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "admin's college not found"})
			continue
		}

		// Get or create Program
		var program models.Program
		programCode := getProgramCode(row.Program)
		err = database.DB.Where("program_code = ?", programCode).FirstOrCreate(&program, models.Program{
			ProgramCode:   programCode,
			ProgramName:   row.Program,
			DurationYears: 4,
		}).Error
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "failed to get/create program: " + err.Error()})
			continue
		}
		if program.ProgramID == 0 {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "program not found and could not be created"})
			continue
		}

		// Get or create Branch
		var branch models.Branch
		branchCode := getBranchCode(row.Branch)
		err = database.DB.Where("short_name = ? AND college_id = ? AND program_id = ?", branchCode, college.CollegeID, program.ProgramID).
			FirstOrCreate(&branch, models.Branch{
				BranchName: row.Branch,
				ShortName:  branchCode,
				CollegeID:  college.CollegeID,
				ProgramID:  program.ProgramID,
			}).Error
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "failed to get/create branch: " + err.Error()})
			continue
		}
		if branch.BranchID == 0 {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "branch not found and could not be created"})
			continue
		}

		// Get or create Academic Year
		academicYearName := fmt.Sprintf("%d-%d", row.AdmissionYear, row.AdmissionYear+1)
		var academicYear models.AcademicYear
		err = database.DB.Where("name = ?", academicYearName).
			FirstOrCreate(&academicYear, models.AcademicYear{
				Name:      academicYearName,
				StartDate: time.Date(row.AdmissionYear, time.July, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(row.AdmissionYear+1, time.June, 30, 0, 0, 0, 0, time.UTC),
			}).Error
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "failed to get/create academic year: " + err.Error()})
			continue
		}

		// Get or create Section (batch-based)
		var section models.Section
		err = database.DB.Where("section_name = ? AND branch_id = ? AND cohort_year = ? AND academic_year_id = ?",
			row.SectionName, branch.BranchID, row.CohortYear, academicYear.AcademicYearID).
			FirstOrCreate(&section, models.Section{
				SectionName:    row.SectionName,
				BranchID:       branch.BranchID,
				CohortYear:     row.CohortYear,
				AcademicYearID: academicYear.AcademicYearID,
			}).Error
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "failed to get/create section: " + err.Error()})
			continue
		}
		if section.SectionID == 0 {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "section not found and could not be created"})
			continue
		}

		// Get or create Curriculum for this program
		var curriculum models.Curriculum
		err = database.DB.Where("program_id = ? AND regulation_id = ?", program.ProgramID, regulation.RegulationID).
			FirstOrCreate(&curriculum, models.Curriculum{
				ProgramID:     program.ProgramID,
				RegulationID:  regulation.RegulationID,
				DurationYears: 4,
				IsActive:      true,
			}).Error
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "failed to get/create curriculum: " + err.Error()})
			continue
		}
		if curriculum.CurriculumID == 0 {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "curriculum not found and could not be created"})
			continue
		}

		// Check if student exists (by regdno OR email)
		var existingUser models.User
		err = database.DB.Where("regdno = ? OR email = ?", row.RegdNo, row.Email).First(&existingUser).Error

		// If user found by email but with different regdno, reject the upload
		if err == nil && existingUser.RegdNo != row.RegdNo {
			result.Failed++
			result.Errors = append(result.Errors, StudentUploadError{
				Row:    rowNum,
				RegdNo: row.RegdNo,
				Error:  fmt.Sprintf("email '%s' already belongs to student with regdno '%s'. Each email must be unique.", row.Email, existingUser.RegdNo),
			})
			continue
		}

		password := row.RegdNo

		if err != nil {
			// Create new user
			user := models.User{
				RegdNo:    row.RegdNo,
				Email:     row.Email,
				Password:  password,
				Role:      row.Role,
				CollegeID: &college.CollegeID,
				Phone:     row.PhoneNumber,
				Name:      row.Name,
				IsActive:  true,
			}

			if err := user.HashPassword(password); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "failed to hash password"})
				continue
			}

			if err := database.DB.Create(&user).Error; err != nil {
				result.Failed++
				result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "failed to create user: " + err.Error()})
				continue
			}

			// Create Student record
			student := models.Student{
				RegdNo:        user.RegdNo,
				CurriculumID:  curriculum.CurriculumID,
				CohortYear:    row.CohortYear,
				AdmissionYear: row.AdmissionYear,
				EntryLevel:    1,
				BranchID:      branch.BranchID,
				SectionID:     &section.SectionID,
				Status:        "active",
				ProgressIndex: 1,
			}
			if err := database.DB.Create(&student).Error; err != nil {
				result.Failed++
				result.Errors = append(result.Errors, StudentUploadError{Row: rowNum, RegdNo: row.RegdNo, Error: "failed to create student: " + err.Error()})
				continue
			}

			// Auto-enroll into any existing course offerings for this section
			enrollStudentInSectionOfferings(user.RegdNo, section.SectionID)

			result.Created++
		} else {
			// Update existing user
			updates := map[string]interface{}{
				"name":       row.Name,
				"phone":      row.PhoneNumber,
				"college_id": college.CollegeID,
				"is_active":  true,
				"deleted_at": nil,
			}
			database.DB.Unscoped().Model(&existingUser).Updates(updates)

			// Update or create Student record
			// IMPORTANT: Student table uses column "regd_no" not "regdno"
			var existingStudent models.Student
			err = database.DB.Where("regd_no = ?", existingUser.RegdNo).First(&existingStudent).Error
			if err != nil {
				// Create student record if missing
				newStudent := models.Student{
					RegdNo:        existingUser.RegdNo,
					CurriculumID:  curriculum.CurriculumID,
					CohortYear:    row.CohortYear,
					AdmissionYear: row.AdmissionYear,
					EntryLevel:    1,
					BranchID:      branch.BranchID,
					SectionID:     &section.SectionID,
					Status:        "active",
					ProgressIndex: 1,
				}
				if createErr := database.DB.Create(&newStudent).Error; createErr == nil {
					// Auto-enroll into any existing course offerings for this section
					enrollStudentInSectionOfferings(existingUser.RegdNo, section.SectionID)
				}
			} else {
				// Update existing student
				database.DB.Model(&existingStudent).Updates(map[string]interface{}{
					"curriculum_id":  curriculum.CurriculumID,
					"branch_id":      branch.BranchID,
					"section_id":     section.SectionID,
					"cohort_year":    row.CohortYear,
					"admission_year": row.AdmissionYear,
				})
				// Auto-enroll into any existing course offerings for this section (in case they were missed)
				enrollStudentInSectionOfferings(existingUser.RegdNo, section.SectionID)
			}

			result.Updated++
		}
	}

	c.JSON(http.StatusOK, result)
}

// enrollStudentInSectionOfferings auto-enrolls a student into all active
// course offerings for their section, skipping offerings they are already enrolled in.
// Uses INSERT...ON CONFLICT to prevent race conditions.
func enrollStudentInSectionOfferings(studentRegdNo string, sectionID uint) {
	// Find all active course offerings for this section
	var offeringIDs []uint
	database.DB.Table("course_offerings").
		Select("id").
		Where("section_id = ? AND is_active = true AND deleted_at IS NULL", sectionID).
		Pluck("id", &offeringIDs)

	for _, offeringID := range offeringIDs {
		// Use INSERT...ON CONFLICT DO NOTHING to prevent duplicate enrollments
		// This is atomic and prevents race conditions
		database.DB.Exec(`
			INSERT INTO enrollments (student_regdno, course_offering_id, type, status, attempt_no, created_at, updated_at)
			VALUES (?, ?, 'Regular', 'enrolled', 1, NOW(), NOW())
			ON CONFLICT (student_regdno, course_offering_id) DO NOTHING
		`, studentRegdNo, offeringID)
	}
}

// parseStudentCSV parses CSV file into StudentCSVRow array (batch-based)
func parseStudentCSV(src io.Reader) ([]StudentCSVRow, error) {
	reader := csv.NewReader(src)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %v", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or has no data rows")
	}

	var rows []StudentCSVRow
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 8 {
			continue
		}

		row := StudentCSVRow{
			RegdNo:      strings.TrimSpace(record[0]),
			Name:        strings.TrimSpace(record[1]),
			Email:       strings.TrimSpace(record[2]),
			PhoneNumber: strings.TrimSpace(record[3]),
			Program:     strings.TrimSpace(record[4]),
			Branch:      strings.TrimSpace(record[5]),
		}

		if len(record) > 6 {
			row.CohortYear, _ = strconv.Atoi(strings.TrimSpace(record[6]))
		}
		if len(record) > 7 {
			row.AdmissionYear, _ = strconv.Atoi(strings.TrimSpace(record[7]))
		}
		if len(record) > 8 {
			row.SectionName = strings.TrimSpace(record[8])
		}
		if len(record) > 9 {
			row.Role = strings.TrimSpace(record[9])
		}

		rows = append(rows, row)
	}

	return rows, nil
}

// getProgramCode converts program name to code
func getProgramCode(programName string) string {
	programMap := map[string]string{
		"B.Tech":   "BT",
		"M.Tech":   "MT",
		"MCA":      "MCA",
		"MBA":      "MBA",
		"B.Arch":   "BARCH",
		"M.Arch":   "MARCH",
		"B.Pharma": "BP",
		"M.Pharma": "MP",
		"Ph.D":     "PHD",
	}

	code := programMap[programName]
	if code == "" {
		if len(programName) >= 2 {
			code = strings.ToUpper(programName[:2])
		} else {
			code = strings.ToUpper(programName)
		}
	}
	return code
}

// getBranchCode converts branch name to short code
func getBranchCode(branchName string) string {
	branchMap := map[string]string{
		"Computer Science":       "CSE",
		"Computer Science & Eng": "CSE",
		"CSE":                    "CSE",
		"Electronics":            "ECE",
		"Electronics & Comm":     "ECE",
		"ECE":                    "ECE",
		"Electrical":             "EEE",
		"EEE":                    "EEE",
		"Mechanical":             "ME",
		"ME":                     "ME",
		"Civil":                  "CE",
		"Civil Engineering":      "CE",
		"CIVIL":                  "CE",
		"CE":                     "CE",
		"Information Technology": "IT",
		"IT":                     "IT",
	}

	code := branchMap[branchName]
	if code == "" {
		if len(branchName) >= 3 {
			code = strings.ToUpper(branchName[:3])
		} else {
			code = strings.ToUpper(branchName)
		}
	}
	return code
}

// ==================== FACULTY ONBOARDING (unchanged, uses User table) ====================

// FacultyBulkUploadResult represents the result of bulk faculty upload
type FacultyBulkUploadResult struct {
	Errors    []FacultyUploadError `json:"errors,omitempty"`
	TotalRows int                  `json:"total_rows"`
	Created   int                  `json:"created"`
	Updated   int                  `json:"updated"`
	Failed    int                  `json:"failed"`
}

// FacultyUploadError represents an error for a specific row
type FacultyUploadError struct {
	RegdNo string `json:"regdno,omitempty"`
	Email  string `json:"email,omitempty"`
	Error  string `json:"error"`
	Row    int    `json:"row"`
}

// FacultyCSVRow represents a row from the uploaded CSV
type FacultyCSVRow struct {
	RegdNo      string `csv:"regdno"`
	Name        string `csv:"name"`
	Email       string `csv:"email"`
	PhoneNumber string `csv:"phone_number"`
	Branch      string `csv:"branch"`
}

// DownloadFacultyTemplate returns a CSV template for faculty onboarding
func DownloadFacultyTemplate(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=faculty_onboarding_template.csv")

	headers := []string{
		"regdno", "name", "email", "phone_number", "branch",
	}

	sampleData := [][]string{
		headers,
		{"EMP001", "Dr. Alan Turing", "alan@example.com", "9876543210", "CSE"},
		{"EMP002", "Grace Hopper", "grace@example.com", "9876543211", "ECE"},
	}

	csvContent := ""
	for _, row := range sampleData {
		csvContent += "\"" + row[0] + "\""
		for i := 1; i < len(row); i++ {
			csvContent += ",\"" + row[i] + "\""
		}
		csvContent += "\r\n"
	}

	c.String(http.StatusOK, csvContent)
}

// BulkUploadFaculty handles bulk faculty upload via CSV
func BulkUploadFaculty(c *gin.Context) {
	adminCollegeID, _ := c.Get("college_id")

	file, err := c.FormFile("csv_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	rows, err := parseFacultyCSV(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse CSV: " + err.Error()})
		return
	}

	log.Printf("Faculty CSV parsed: %d rows found", len(rows))

	result := FacultyBulkUploadResult{
		TotalRows: len(rows),
		Errors:    []FacultyUploadError{},
	}

	// First pass: Check for duplicate emails within the CSV file
	emailToRow := make(map[string]int) // email -> first row number where it appeared
	for i, row := range rows {
		if row.Email != "" {
			emailLower := strings.ToLower(row.Email)
			if firstRow, exists := emailToRow[emailLower]; exists {
				result.Failed++
				result.Errors = append(result.Errors, FacultyUploadError{
					Row:    i + 2,
					RegdNo: row.RegdNo,
					Email:  row.Email,
					Error:  fmt.Sprintf("duplicate email '%s' (first appeared in row %d)", row.Email, firstRow),
				})
			} else {
				emailToRow[emailLower] = i + 2
			}
		}
	}

	// Second pass: Validate and process each row
	for i, row := range rows {
		rowNum := i + 2

		// Skip if already marked as failed due to duplicate email
		isDuplicate := false
		for _, err := range result.Errors {
			if err.Row == rowNum {
				isDuplicate = true
				break
			}
		}
		if isDuplicate {
			continue
		}

		if row.Name == "" {
			result.Failed++
			result.Errors = append(result.Errors, FacultyUploadError{Row: rowNum, RegdNo: row.RegdNo, Email: row.Email, Error: "name is required"})
			continue
		}
		if row.Email == "" {
			result.Failed++
			result.Errors = append(result.Errors, FacultyUploadError{Row: rowNum, RegdNo: row.RegdNo, Email: row.Email, Error: "email is required"})
			continue
		}
		if row.Branch == "" {
			result.Failed++
			result.Errors = append(result.Errors, FacultyUploadError{Row: rowNum, RegdNo: row.RegdNo, Email: row.Email, Error: "branch is required"})
			continue
		}

		// Use admin's college directly
		var college models.College
		if adminCollegeID == nil {
			result.Failed++
			result.Errors = append(result.Errors, FacultyUploadError{Row: rowNum, RegdNo: row.RegdNo, Email: row.Email, Error: "admin has no college assigned"})
			continue
		}
		adminCollegeIDPtr := adminCollegeID.(*string)
		err := database.DB.First(&college, "college_id = ?", *adminCollegeIDPtr).Error
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, FacultyUploadError{Row: rowNum, RegdNo: row.RegdNo, Email: row.Email, Error: "admin's college not found"})
			continue
		}

		// Get or create default Program for branch creation (default to B.Tech)
		var program models.Program
		database.DB.Where("program_code = ?", "BT").FirstOrCreate(&program, models.Program{
			ProgramCode:   "BT",
			ProgramName:   "B.Tech",
			DurationYears: 4,
		})

		// Get or create Branch (matching student upload pattern)
		var branch models.Branch
		branchCode := getBranchCode(row.Branch)
		err = database.DB.Where("short_name = ? AND college_id = ? AND program_id = ?", branchCode, college.CollegeID, program.ProgramID).
			FirstOrCreate(&branch, models.Branch{
				BranchName: row.Branch,
				ShortName:  branchCode,
				CollegeID:  college.CollegeID,
				ProgramID:  program.ProgramID,
			}).Error
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, FacultyUploadError{Row: rowNum, RegdNo: row.RegdNo, Email: row.Email, Error: "branch not found and could not be created: " + err.Error()})
			continue
		}
		if branch.BranchID == 0 {
			result.Failed++
			result.Errors = append(result.Errors, FacultyUploadError{Row: rowNum, RegdNo: row.RegdNo, Email: row.Email, Error: "branch not found and could not be created"})
			continue
		}

		var existingUser models.User
		err = database.DB.Where("email = ? OR (regdno = ? AND regdno != '')", row.Email, row.RegdNo).First(&existingUser).Error

		// If user found by email but with different regdno, reject the upload
		if err == nil && existingUser.RegdNo != row.RegdNo {
			result.Failed++
			result.Errors = append(result.Errors, FacultyUploadError{
				Row:    rowNum,
				RegdNo: row.RegdNo,
				Email:  row.Email,
				Error:  fmt.Sprintf("email '%s' already belongs to faculty with regdno '%s'. Each email must be unique.", row.Email, existingUser.RegdNo),
			})
			continue
		}

		// Use regdno as the initial password (common pattern in educational systems)
		// This is secure because regdno is unique per faculty and known only to them
		password := row.RegdNo

		if err != nil {
			user := models.User{
				RegdNo:    row.RegdNo,
				Email:     row.Email,
				Password:  password,
				Role:      "faculty",
				CollegeID: &college.CollegeID,
				BranchID:  &branch.BranchID,
				Phone:     row.PhoneNumber,
				Name:      row.Name,
				IsActive:  true,
			}

			if err := user.HashPassword(password); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, FacultyUploadError{Row: rowNum, Email: row.Email, Error: "failed to hash password"})
				continue
			}

			if err := database.DB.Create(&user).Error; err != nil {
				result.Failed++
				result.Errors = append(result.Errors, FacultyUploadError{Row: rowNum, Email: row.Email, Error: "failed to create user: " + err.Error()})
				continue
			}

			// Create Faculty record
			faculty := models.Faculty{
				RegdNo:      user.RegdNo,
				BranchID:    branch.BranchID,
				Designation: "Faculty",
				JoiningDate: time.Now(),
				IsActive:    true,
			}
			database.DB.Create(&faculty)

			result.Created++
		} else {
			updates := map[string]interface{}{
				"name":       row.Name,
				"phone":      row.PhoneNumber,
				"college_id": college.CollegeID,
				"branch_id":  branch.BranchID,
				"role":       "faculty",
				"is_active":  true,
				"deleted_at": nil,
			}
			if row.RegdNo != "" {
				updates["regdno"] = row.RegdNo
			}

			database.DB.Unscoped().Model(&existingUser).Updates(updates)

			// Update or create Faculty record
			var existingFaculty models.Faculty
			err = database.DB.Where("regdno = ?", existingUser.RegdNo).First(&existingFaculty).Error
			if err != nil {
				faculty := models.Faculty{
					RegdNo:      existingUser.RegdNo,
					BranchID:    branch.BranchID,
					Designation: "Faculty",
					JoiningDate: time.Now(),
					IsActive:    true,
				}
				database.DB.Create(&faculty)
			}

			result.Updated++
		}
	}

	c.JSON(http.StatusOK, result)
}

// parseFacultyCSV parses CSV file into FacultyCSVRow array
func parseFacultyCSV(src io.Reader) ([]FacultyCSVRow, error) {
	reader := csv.NewReader(src)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %v", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or has no data rows")
	}

	var rows []FacultyCSVRow
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 5 {
			continue
		}

		row := FacultyCSVRow{
			RegdNo:      strings.TrimSpace(record[0]),
			Name:        strings.TrimSpace(record[1]),
			Email:       strings.TrimSpace(record[2]),
			PhoneNumber: strings.TrimSpace(record[3]),
			Branch:      strings.TrimSpace(record[4]),
		}

		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("no valid data rows found in CSV")
	}

	return rows, nil
}

// getKeysFromMap extracts keys from a map[uint]bool
func getKeysFromMap(m map[uint]bool) []uint {
	keys := make([]uint, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
