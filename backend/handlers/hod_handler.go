package handlers

import (
	"coding-platform/database"
	"coding-platform/middleware"
	"coding-platform/models"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetHODDashboardStats returns overall dashboard statistics for HOD
func GetHODDashboardStats(c *gin.Context) {
	var stats struct {
		TotalStudents     int64 `json:"total_students"`
		TotalFaculty      int64 `json:"total_faculty"`
		TotalCourses      int64 `json:"total_courses"`
		TotalSubmissions  int64 `json:"total_submissions"`
		ActiveStudents24h int64 `json:"active_students_24h"`
	}

	collegeID, hasCollege := middleware.GetCurrentUserCollege(c)

	if hasCollege {
		database.DB.Model(&models.User{}).Where("role = ? AND college_id = ?", "student", *collegeID).Count(&stats.TotalStudents)
		database.DB.Model(&models.User{}).Where("role = ? AND college_id = ?", "faculty", *collegeID).Count(&stats.TotalFaculty)
		database.DB.Model(&models.Course{}).Where("college_id = ?", *collegeID).Count(&stats.TotalCourses)
		// Use direct college_id on submissions for efficient filtering (no JOIN needed)
		database.DB.Model(&models.Submission{}).
			Where("college_id = ?", *collegeID).
			Count(&stats.TotalSubmissions)
		database.DB.Model(&models.Submission{}).
			Where("college_id = ? AND created_at >= NOW() - INTERVAL '24 hours'", *collegeID).
			Distinct("user_regd_no").
			Count(&stats.ActiveStudents24h)
	} else {
		database.DB.Model(&models.User{}).Where("role = ?", "student").Count(&stats.TotalStudents)
		database.DB.Model(&models.User{}).Where("role = ?", "faculty").Count(&stats.TotalFaculty)
		database.DB.Model(&models.Course{}).Count(&stats.TotalCourses)
		database.DB.Model(&models.Submission{}).Count(&stats.TotalSubmissions)
		database.DB.Model(&models.Submission{}).
			Where("created_at >= NOW() - INTERVAL '24 hours'").
			Distinct("user_regd_no").
			Count(&stats.ActiveStudents24h)
	}

	c.JSON(http.StatusOK, stats)
}

// GetBranchAnalytics returns detailed branch performance metrics
func GetBranchAnalytics(c *gin.Context) {
	type BranchAnalytics struct {
		TopPerformers        []map[string]interface{} `json:"top_performers"`
		CourseCompletion     []map[string]interface{} `json:"course_completion"`
		FacultyPerformance   []map[string]interface{} `json:"faculty_performance"`
		WeeklySubmissionData []map[string]interface{} `json:"weekly_submission_data"`
		DifficultyStats      []map[string]interface{} `json:"difficulty_stats"`
	}

	// Get top performers (students with most solved problems)
	var topPerformers []struct {
		RegdNo string `json:"regdno"`
		Name   string `json:"name"`
		Solved int    `json:"solved"`
	}
	topQuery := database.DB.Table("users u").
		Select("u.regdno, u.name, COUNT(DISTINCT sp.problem_id) as solved").
		Joins("INNER JOIN user_problem_completions sp ON u.regdno = sp.user_regd_no").
		Where("u.role = ?", "student")

	collegeID, hasCollege := middleware.GetCurrentUserCollege(c)
	if hasCollege {
		topQuery = topQuery.Where("u.college_id = ?", *collegeID)
	}

	topQuery.Group("u.regdno, u.name").
		Order("solved DESC").
		Limit(10).
		Scan(&topPerformers)

	performers := make([]map[string]interface{}, len(topPerformers))
	for i, p := range topPerformers {
		performers[i] = map[string]interface{}{
			"rank":   i + 1,
			"regdno": p.RegdNo,
			"name":   p.Name,
			"solved": p.Solved,
		}
	}

	// Get course completion stats - OPTIMIZED: single query with GROUP BY instead of N+1
	var courses []models.Course
	courseQuery := database.DB.Preload("Lab").Preload("Theory")
	if hasCollege {
		courseQuery = courseQuery.Where("college_id = ?", *collegeID)
	}
	courseQuery.Find(&courses)

	// OPTIMIZED: Get all enrolled counts in a single query
	type CourseEnrollment struct {
		CourseID      uint  `gorm:"column:course_id"`
		EnrolledCount int64 `gorm:"column:enrolled_count"`
	}
	var enrollments []CourseEnrollment
	courseIDs := make([]uint, len(courses))
	for i, c := range courses {
		courseIDs[i] = c.ID
	}
	if len(courseIDs) > 0 {
		database.DB.Raw(`
			SELECT co.course_id, COUNT(DISTINCT e.student_regdno) as enrolled_count
			FROM enrollments e
			INNER JOIN course_offerings co ON co.id = e.course_offering_id
			WHERE co.course_id IN ?
			GROUP BY co.course_id
		`, courseIDs).Scan(&enrollments)
	}

	// Build enrollment map for quick lookup
	enrollmentMap := make(map[uint]int64)
	for _, e := range enrollments {
		enrollmentMap[e.CourseID] = e.EnrolledCount
	}

	courseCompletion := make([]map[string]interface{}, len(courses))
	for i, course := range courses {
		courseCompletion[i] = map[string]interface{}{
			"course_code":    course.CourseCode,
			"course_name":    course.CourseName,
			"total_enrolled": enrollmentMap[course.ID],
			"has_lab":        course.Lab != nil,
			"has_theory":     course.Theory != nil,
		}
	}

	// Get HOD's branch for filtering
	userRegdNo, _ := c.Get("regdno")
	var hod models.Faculty
	if err := database.DB.Where("regd_no = ?", userRegdNo).First(&hod).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch HOD information"})
		return
	}

	// Get faculty performance - only faculty from HOD's branch
	var faculty []models.Faculty
	query := database.DB
	if hod.BranchID > 0 {
		query = query.Where("branch_id = ?", hod.BranchID)
	}
	query.Find(&faculty)
	loadBranchesForFaculty(faculty)
	loadUsersForFaculty(faculty)

	// OPTIMIZED: Get all problem counts in a single query instead of N+1
	type FacultyProblemCount struct {
		RegdNo       string `gorm:"column:created_by"`
		ProblemCount int64  `gorm:"column:problem_count"`
	}
	var problemCounts []FacultyProblemCount
	database.DB.Raw(`
		SELECT created_by, COUNT(*) as problem_count
		FROM problems
		WHERE created_by = ?
		GROUP BY created_by
	`, getFacultyRegdNos(faculty)).Scan(&problemCounts)

	// Build problem count map for quick lookup
	problemCountMap := make(map[string]int64)
	for _, pc := range problemCounts {
		problemCountMap[pc.RegdNo] = pc.ProblemCount
	}

	facultyPerformance := make([]map[string]interface{}, len(faculty))
	for i, f := range faculty {
		name := ""
		if f.User != nil {
			name = f.User.Name
		}

		facultyPerformance[i] = map[string]interface{}{
			"regdno":           f.RegdNo,
			"name":             name,
			"problems_created": problemCountMap[f.RegdNo],
		}
	}

	// NOTE: This query uses parameterized query with ? placeholder for collegeID
	// This is safe from SQL injection
	var weeklyData []struct {
		Week  int `json:"week"`
		Count int `json:"count"`
	}
	if hasCollege {
		database.DB.Raw(`
			SELECT EXTRACT(WEEK FROM s.created_at) as week, COUNT(*) as count
			FROM submissions s
			INNER JOIN users u ON u.regdno = s.user_regd_no
			WHERE s.created_at >= NOW() - INTERVAL '8 weeks' AND u.college_id = ?
			GROUP BY EXTRACT(WEEK FROM s.created_at)
			ORDER BY week
		`, *collegeID).Scan(&weeklyData)
	} else {
		database.DB.Raw(`
			SELECT EXTRACT(WEEK FROM created_at) as week, COUNT(*) as count
			FROM submissions
			WHERE created_at >= NOW() - INTERVAL '8 weeks'
			GROUP BY EXTRACT(WEEK FROM created_at)
			ORDER BY week
		`).Scan(&weeklyData)
	}

	weeklySubmissionData := make([]map[string]interface{}, len(weeklyData))
	for i, d := range weeklyData {
		weeklySubmissionData[i] = map[string]interface{}{
			"week":  d.Week,
			"count": d.Count,
		}
	}

	// Difficulty stats
	var difficultyStats []struct {
		Difficulty string `json:"difficulty"`
		Count      int    `json:"count"`
	}
	diffQuery := database.DB.Model(&models.Problem{}).
		Select("difficulty, COUNT(*) as count")
	if hasCollege {
		diffQuery = diffQuery.Where("college_id = ? OR is_global = ?", *collegeID, true)
	}
	diffQuery.Group("difficulty").
		Scan(&difficultyStats)

	diffStats := make([]map[string]interface{}, len(difficultyStats))
	for i, d := range difficultyStats {
		diffStats[i] = map[string]interface{}{
			"difficulty": d.Difficulty,
			"count":      d.Count,
		}
	}

	analytics := BranchAnalytics{
		TopPerformers:        performers,
		CourseCompletion:     courseCompletion,
		FacultyPerformance:   facultyPerformance,
		WeeklySubmissionData: weeklySubmissionData,
		DifficultyStats:      diffStats,
	}

	c.JSON(http.StatusOK, analytics)
}

// loadBranchesForFaculty bulk-loads Branch records for a slice of Faculty and populates Faculty.Branch.
// Avoids Preload("Branch") which confuses GORM due to Faculty's string primary key.
func loadBranchesForFaculty(faculty []models.Faculty) {
	if len(faculty) == 0 {
		return
	}
	seen := make(map[uint]bool)
	branchIDs := make([]uint, 0, len(faculty))
	for _, f := range faculty {
		if !seen[f.BranchID] {
			branchIDs = append(branchIDs, f.BranchID)
			seen[f.BranchID] = true
		}
	}
	var branches []models.Branch
	database.DB.Where("branch_id IN ?", branchIDs).Find(&branches)
	branchMap := make(map[uint]*models.Branch, len(branches))
	for i := range branches {
		branchMap[branches[i].BranchID] = &branches[i]
	}
	for i := range faculty {
		if b, ok := branchMap[faculty[i].BranchID]; ok {
			faculty[i].Branch = b
		}
	}
}

// loadUsersForFaculty bulk-loads User records for a slice of Faculty and populates Faculty.User.
// This is needed because Faculty.User is tagged gorm:"-" to prevent GORM from creating backwards FK constraints.
func loadUsersForFaculty(faculty []models.Faculty) {
	if len(faculty) == 0 {
		return
	}
	regdNos := make([]string, len(faculty))
	for i, f := range faculty {
		regdNos[i] = f.RegdNo
	}
	var users []models.User
	database.DB.Where("regdno IN ?", regdNos).Find(&users)
	userMap := make(map[string]*models.User, len(users))
	for i := range users {
		userMap[users[i].RegdNo] = &users[i]
	}
	for i := range faculty {
		if u, ok := userMap[faculty[i].RegdNo]; ok {
			faculty[i].User = u
		}
	}
}

// getFacultyRegdNos extracts registration numbers from a faculty slice for IN queries
func getFacultyRegdNos(faculty []models.Faculty) []string {
	regdNos := make([]string, len(faculty))
	for i, f := range faculty {
		regdNos[i] = f.RegdNo
	}
	return regdNos
}

// GetHODFaculty returns faculty for HOD management (filtered by HOD's branch)
func GetHODFaculty(c *gin.Context) {
	hodRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get HOD's branch information
	var hodUser models.User
	if err := database.DB.Where("regdno = ?", hodRegdNo).First(&hodUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get HOD information"})
		return
	}

	if hodUser.BranchID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HOD must be assigned to a branch first"})
		return
	}

	// Get the branch to find the program
	var hodBranch models.Branch
	if err := database.DB.Preload("Program").First(&hodBranch, hodUser.BranchID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get HOD branch information"})
		return
	}

	// Fetch faculty with the same branch as HOD
	var faculty []models.Faculty
	if err := database.DB.Where("branch_id = ?", hodUser.BranchID).Find(&faculty).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch faculty: " + err.Error()})
		return
	}
	loadBranchesForFaculty(faculty)
	loadUsersForFaculty(faculty)

	c.JSON(http.StatusOK, faculty)
}

// UpdateFacultyRole updates a faculty member's role
func UpdateFacultyRole(c *gin.Context) {
	var req struct {
		RegdNo  string `json:"regdno" binding:"required"`
		NewRole string `json:"new_role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var faculty models.Faculty
	if err := database.DB.Where("regdno = ?", req.RegdNo).First(&faculty).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Faculty not found"})
		return
	}

	// Update User role and invalidate their session
	var user models.User
	if err := database.DB.Where("regdno = ?", req.RegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"role":          req.NewRole,
		"token_version": gorm.Expr("token_version + 1"),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully", "user": user})
}

// GetHODCourses returns all courses from HOD's college (read-only for HOD)
func GetHODCourses(c *gin.Context) {
	userRegdNo, _ := c.Get("regdno")

	// Get HOD's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HOD must be assigned to a college first"})
		return
	}

	var courses []models.Course
	if err := database.DB.Where("college_id = ?", *user.CollegeID).
		Preload("Lab").
		Preload("Theory").
		Order("created_at DESC").
		Find(&courses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch courses"})
		return
	}

	// Add enrolled student counts to each course
	type CourseWithEnrolled struct {
		models.Course
		EnrolledCount int64 `json:"enrolled_count"`
	}

	results := make([]CourseWithEnrolled, len(courses))
	for i, course := range courses {
		var enrolledCount int64
		database.DB.Raw(`
			SELECT COUNT(DISTINCT e.student_regdno)
			FROM enrollments e
			INNER JOIN course_offerings co ON co.id = e.course_offering_id
			WHERE co.course_id = ?
		`, course.ID).Scan(&enrolledCount)

		results[i] = CourseWithEnrolled{
			Course:        course,
			EnrolledCount: enrolledCount,
		}
	}

	c.JSON(http.StatusOK, results)
}

// GetHODCourseOfferings returns all active course offerings with course/section details (filtered by HOD's college)
func GetHODCourseOfferings(c *gin.Context) {
	userRegdNo, _ := c.Get("regdno")

	// Get HOD's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HOD must be assigned to a college first"})
		return
	}

	type OfferingResult struct {
		CourseCode   string `json:"course_code"`
		CourseName   string `json:"course_name"`
		SectionName  string `json:"section_name"`
		AcademicYear string `json:"academic_year"`
		ID           uint   `json:"id"`
		CohortYear   int    `json:"cohort_year"`
	}

	var offerings []OfferingResult
	database.DB.Table("course_offerings co").
		Select(`co.id, c.course_code, c.course_name, s.section_name, s.cohort_year,
		        ay.name as academic_year`).
		Joins("INNER JOIN courses c ON c.id = co.course_id").
		Joins("INNER JOIN sections s ON s.section_id = co.section_id").
		Joins("INNER JOIN academic_years ay ON ay.academic_year_id = co.academic_year_id").
		Joins("INNER JOIN branches b ON b.branch_id = s.branch_id").
		Where("co.is_active = ? AND co.deleted_at IS NULL AND b.college_id = ?", true, *user.CollegeID).
		Order("c.course_code, s.section_name").
		Scan(&offerings)

	c.JSON(http.StatusOK, offerings)
}

// GetFacultyAssignments returns faculty assignments via CourseOffering (filtered by HOD's college)
func GetFacultyAssignments(c *gin.Context) {
	userRegdNo, _ := c.Get("regdno")

	// Get HOD's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HOD must be assigned to a college first"})
		return
	}

	type Assignment struct {
		FacultyRegdNo string `json:"faculty_regdno"`
		FacultyName   string `json:"faculty_name"`
		CourseCode    string `json:"course_code"`
		CourseName    string `json:"course_name"`
		AcademicYear  string `json:"academic_year"`
		SectionName   string `json:"section_name"`
		CohortYear    int    `json:"cohort_year"`
	}

	var assignments []Assignment

	database.DB.Raw(`
		SELECT
			fa.faculty_regd_no,
			u.name as faculty_name,
			c.course_code,
			c.course_name,
			ay.name as academic_year,
			s.section_name,
			s.cohort_year
		FROM faculty_assignments fa
		INNER JOIN users u ON u.regdno = fa.faculty_regd_no
		INNER JOIN course_offerings co ON co.id = fa.course_offering_id
		INNER JOIN courses c ON c.id = co.course_id
		INNER JOIN sections s ON s.section_id = co.section_id
		INNER JOIN academic_years ay ON ay.academic_year_id = co.academic_year_id
		INNER JOIN branches b ON b.branch_id = s.branch_id
		WHERE fa.is_active = true AND b.college_id = ?
		ORDER BY u.name, c.course_code
	`, *user.CollegeID).Scan(&assignments)

	c.JSON(http.StatusOK, assignments)
}

// AssignFacultyToCourseOffering assigns faculty to a CourseOffering
// Validates: HOD.branch == Course.branch AND Faculty.branch == HOD.branch
func AssignFacultyToCourseOffering(c *gin.Context) {
	var req struct {
		FacultyRegdNo    string `json:"faculty_regdno" binding:"required"`
		Role             string `json:"role"`
		CourseOfferingID uint   `json:"course_offering_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify faculty exists
	var faculty models.Faculty
	if err := database.DB.Where("regd_no = ?", req.FacultyRegdNo).First(&faculty).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Faculty not found"})
		return
	}

	// Verify CourseOffering exists
	var offering models.CourseOffering
	if err := database.DB.First(&offering, req.CourseOfferingID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course offering not found"})
		return
	}

	// Validate branch access: HOD.branch == Course.branch AND Faculty.branch == HOD.branch
	if err := validateBranchAccessForAssignment(c, req.FacultyRegdNo, req.CourseOfferingID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Set default role
	role := req.Role
	if role == "" {
		role = "teacher"
	}

	// Check if assignment already exists
	var existingAssignment models.FacultyAssignment
	result := database.DB.Where("faculty_regd_no = ? AND course_offering_id = ?",
		req.FacultyRegdNo, req.CourseOfferingID).First(&existingAssignment)

	if result.Error == nil {
		// Update existing assignment
		existingAssignment.Role = role
		existingAssignment.IsActive = true
		database.DB.Save(&existingAssignment)
		c.JSON(http.StatusOK, gin.H{"message": "Faculty assignment updated", "assignment": existingAssignment})
		return
	}

	// Create new assignment
	assignment := models.FacultyAssignment{
		FacultyRegdNo:    req.FacultyRegdNo,
		CourseOfferingID: req.CourseOfferingID,
		Role:             role,
		IsActive:         true,
	}
	if err := database.DB.Create(&assignment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Faculty assigned to course offering successfully", "assignment": assignment})
}

// AssignFacultyToAllSections assigns a faculty to all sections for a course in current academic year
// Validates: Faculty.branch == HOD.branch, and only assigns to offerings in HOD's branch
func AssignFacultyToAllSections(c *gin.Context) {
	var req struct {
		FacultyRegdNo string `json:"faculty_regdno" binding:"required"`
		CourseID      uint   `json:"course_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get HOD's branch for validation
	hodRegdNo, _ := c.Get("regdno")
	var hodUser models.User
	if err := database.DB.Where("regdno = ?", hodRegdNo).First(&hodUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get HOD information"})
		return
	}
	if hodUser.BranchID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HOD must be assigned to a branch first"})
		return
	}

	// Verify faculty exists AND validate faculty is in HOD's branch
	var faculty models.Faculty
	if err := database.DB.Where("regd_no = ?", req.FacultyRegdNo).First(&faculty).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Faculty not found"})
		return
	}
	if faculty.BranchID != *hodUser.BranchID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Faculty is not in HOD's branch"})
		return
	}

	// Verify course exists
	var course models.Course
	if err := database.DB.First(&course, req.CourseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Get current academic year
	var academicYear models.AcademicYear
	database.DB.Where("is_current = ?", true).First(&academicYear)
	if academicYear.AcademicYearID == 0 {
		database.DB.Order("start_date DESC").First(&academicYear)
	}

	if academicYear.AcademicYearID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No academic year found"})
		return
	}

	// Find all CourseOfferings matching the criteria AND in HOD's branch
	var offerings []models.CourseOffering
	database.DB.
		Joins("INNER JOIN sections s ON s.section_id = course_offerings.section_id").
		Where("course_offerings.course_id = ? AND course_offerings.academic_year_id = ? AND s.branch_id = ?",
			req.CourseID, academicYear.AcademicYearID, *hodUser.BranchID).
		Find(&offerings)

	if len(offerings) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No course offerings found for this course in HOD's branch for the current academic year"})
		return
	}

	// Create assignments for each offering
	assignmentsCreated := 0
	assignmentsSkipped := 0
	for _, offering := range offerings {
		var existingAssignment models.FacultyAssignment
		err := database.DB.Where("faculty_regd_no = ? AND course_offering_id = ?",
			req.FacultyRegdNo, offering.ID).
			First(&existingAssignment).Error
		if err != nil {
			assignment := models.FacultyAssignment{
				FacultyRegdNo:    req.FacultyRegdNo,
				CourseOfferingID: offering.ID,
				Role:             "teacher",
				IsActive:         true,
			}
			if database.DB.Create(&assignment).Error == nil {
				assignmentsCreated++
			}
		} else {
			assignmentsSkipped++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             fmt.Sprintf("Faculty assigned to %d course offering(s) in HOD's branch", assignmentsCreated),
		"assignments_created": assignmentsCreated,
		"assignments_skipped": assignmentsSkipped,
		"total_offerings":     len(offerings),
	})
}

// HOD Course Analytics
type HODCourseAnalyticsResponse struct {
	Course   CourseInfo    `json:"course"`
	Sections []SectionInfo `json:"sections"`
}

// HOD-specific TopicStat for backward compatibility
type TopicStat struct {
	TopicName      string  `json:"topic_name"`
	StudentsSolved int     `json:"students_solved"`
	Percentage     float64 `json:"percentage"`
	TotalProblems  int     `json:"total_problems"`
}

// HOD-specific SectionAnalyticsResponse
type HODSectionAnalyticsResponse struct {
	TopicStats []TopicStat      `json:"topic_stats"`
	Students   []StudentSummary `json:"students"`
	Section    SectionSummary   `json:"section"`
}

// GetHODCourseAnalytics returns all sections for a course with student counts
func GetHODCourseAnalytics(c *gin.Context) {
	courseID := c.Param("id")
	userRegdNo, _ := c.Get("regdno")

	// Verify user is HOD
	var user models.User
	if err := database.DB.First(&user, "regdno = ?", userRegdNo).Error; err != nil || user.Role != "hod" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized - HOD only"})
		return
	}

	// Get course details
	var course models.Course
	if err := database.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Get all sections for this course via CourseOfferings (batch-based)
	// NOTE: This query uses parameterized query with ? placeholder for courseID
	// This is safe from SQL injection
	type SectionResult struct {
		SectionName  string
		AcademicYear string
		SectionID    uint
		CohortYear   int
		StudentCount int64
	}

	var sections []SectionResult
	database.DB.Table("sections s").
		Select(`
			s.section_id,
			s.section_name,
			s.cohort_year,
			COUNT(DISTINCT e.student_regdno) as student_count,
			ay.name as academic_year
		`).
		Joins("INNER JOIN course_offerings co ON co.section_id = s.section_id").
		Joins("INNER JOIN academic_years ay ON ay.academic_year_id = co.academic_year_id").
		Joins("LEFT JOIN enrollments e ON e.course_offering_id = co.id").
		Where("co.course_id = ?", courseID).
		Group("s.section_id, s.section_name, s.cohort_year, ay.name").
		Scan(&sections)

	log.Printf("HOD Course Analytics: Found %d sections for course %d", len(sections), course.ID)

	var sectionInfos []SectionInfo
	for _, s := range sections {
		sectionInfos = append(sectionInfos, SectionInfo{
			SectionID:    s.SectionID,
			SectionName:  s.SectionName,
			StudentCount: int(s.StudentCount),
			CohortYear:   s.CohortYear,
			AcademicYear: s.AcademicYear,
		})
	}

	response := HODCourseAnalyticsResponse{
		Course: CourseInfo{
			ID:         course.ID,
			CourseCode: course.CourseCode,
			CourseName: course.CourseName,
		},
		Sections: sectionInfos,
	}

	c.JSON(http.StatusOK, response)
}

// GetHODSectionAnalytics returns detailed analytics for a section
func GetHODSectionAnalytics(c *gin.Context) {
	sectionID := c.Param("sectionId")
	userRegdNo, _ := c.Get("regdno")

	// Verify user is HOD
	var user models.User
	if err := database.DB.First(&user, "regdno = ?", userRegdNo).Error; err != nil || user.Role != "hod" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized - HOD only"})
		return
	}

	// Verify section exists
	var section models.Section
	if err := database.DB.First(&section, sectionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Section not found"})
		return
	}

	// Count total students in section (via Student table)
	var totalStudents int64
	database.DB.Model(&models.Student{}).
		Where("section_id = ?", sectionID).
		Count(&totalStudents)

	// Get topic-wise stats using problem tags
	// NOTE: This query uses parameterized query with ? placeholder for sectionID
	// This is safe from SQL injection
	type TopicResult struct {
		Tag          string
		StudentCount int64
		ProblemCount int64
	}

	var topicResults []TopicResult
	database.DB.Raw(`
		SELECT
			TRIM(p.tags) as tag,
			COUNT(DISTINCT upc.user_regd_no) as student_count,
			COUNT(DISTINCT p.id) as problem_count
		FROM problems p
		LEFT JOIN user_problem_completions upc ON upc.problem_id = p.id
		LEFT JOIN students st ON st.regdno = upc.user_regd_no
		WHERE st.section_id = ? AND p.tags IS NOT NULL AND p.tags != ''
		GROUP BY p.tags
	`, sectionID).Scan(&topicResults)

	var topicStats []TopicStat
	for _, tr := range topicResults {
		percentage := 0.0
		if totalStudents > 0 {
			percentage = (float64(tr.StudentCount) / float64(totalStudents)) * 100
		}

		topicStats = append(topicStats, TopicStat{
			TopicName:      tr.Tag,
			StudentsSolved: int(tr.StudentCount),
			Percentage:     percentage,
			TotalProblems:  int(tr.ProblemCount),
		})
	}

	// Get student list with summary stats
	// NOTE: This query uses parameterized query with ? placeholder for sectionID
	// This is safe from SQL injection
	type StudentResult struct {
		LastActive     *time.Time `gorm:"column:last_active"`
		RegdNo         string     `gorm:"column:regd_no"`
		Name           string     `gorm:"column:name"`
		ProblemsSolved int64      `gorm:"column:problems_solved"`
		TimeSpent      float64    `gorm:"column:time_spent"`
		CurrentStreak  int        `gorm:"column:current_streak"`
	}

	var studentResults []StudentResult
	database.DB.Raw(`
		SELECT
			u.regdno as regd_no,
			u.name as name,
			COUNT(DISTINCT upc.problem_id) as problems_solved,
			COALESCE(SUM(s.time_spent), 0) as time_spent,
			COALESCE(us.current_streak, 0) as current_streak,
			MAX(ua.date) as last_active
		FROM users u
		JOIN students st ON st.regd_no = u.regdno
		LEFT JOIN user_problem_completions upc ON upc.user_regd_no = u.regdno
		LEFT JOIN submissions s ON s.id = upc.first_submission_id
		LEFT JOIN user_streaks us ON us.user_regd_no = u.regdno
		LEFT JOIN user_activities ua ON ua.user_regd_no = u.regdno
		WHERE st.section_id = ?
		GROUP BY u.regdno, u.name, us.current_streak
		ORDER BY problems_solved DESC
	`, sectionID).Scan(&studentResults)

	var students []StudentSummary
	for _, sr := range studentResults {
		lastActive := time.Time{}
		if sr.LastActive != nil {
			lastActive = *sr.LastActive
		}

		students = append(students, StudentSummary{
			RegdNo:         sr.RegdNo,
			Name:           sr.Name,
			ProblemsSolved: int(sr.ProblemsSolved),
			TimeSpent:      sr.TimeSpent,
			CurrentStreak:  sr.CurrentStreak,
			LastActive:     lastActive,
		})
	}

	response := HODSectionAnalyticsResponse{
		Section: SectionSummary{
			SectionID:     section.SectionID,
			SectionName:   section.SectionName,
			TotalStudents: int(totalStudents),
			CohortYear:    section.CohortYear,
		},
		TopicStats: topicStats,
		Students:   students,
	}

	c.JSON(http.StatusOK, response)
}

// GetCourseAssignments returns faculty course assignments (wrapper for GetFacultyAssignments)
func GetCourseAssignments(c *gin.Context) {
	GetFacultyAssignments(c)
}

// AssignFacultyToCourse assigns faculty to all active CourseOfferings for a given course
// GetSectionsForCourse returns sections that have active offerings for a given course
// Now updated to return ALL sections in HOD's branch, with a flag indicating if offering exists
func GetSectionsForCourse(c *gin.Context) {
	courseID := c.Param("courseId")

	// Get HOD's branch
	hodRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "User not authenticated"})
		return
	}

	var hodUser models.User
	if err := database.DB.Where("regdno = ?", hodRegdNo).First(&hodUser).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Failed to get HOD information"})
		return
	}
	if hodUser.BranchID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "HOD must be assigned to a branch first"})
		return
	}

	type SectionResult struct {
		OfferingID  *uint  `json:"offering_id"`
		SectionName string `json:"section_name"`
		BranchName  string `json:"branch_name"`
		SectionID   uint   `json:"section_id"`
		CohortYear  int    `json:"cohort_year"`
		HasOffering bool   `json:"has_offering"`
	}

	// Get ALL sections in HOD's branch (not just ones with offerings)
	var sections []SectionResult
	database.DB.Table("sections s").
		Select(`
			s.section_id,
			s.section_name,
			s.cohort_year,
			b.branch_name,
			co.id as offering_id
		`).
		Joins("INNER JOIN branches b ON b.branch_id = s.branch_id").
		Joins("LEFT JOIN course_offerings co ON co.section_id = s.section_id AND co.course_id = ? AND co.is_active = ? AND co.deleted_at IS NULL", courseID, true).
		Where("s.branch_id = ? AND s.is_active = ? AND s.deleted_at IS NULL", *hodUser.BranchID, true).
		Order("s.cohort_year DESC, s.section_name").
		Scan(&sections)

	// Convert to response format with HasOffering flag
	var result []SectionResult
	for _, s := range sections {
		result = append(result, SectionResult{
			OfferingID:  s.OfferingID,
			SectionID:   s.SectionID,
			SectionName: s.SectionName,
			CohortYear:  s.CohortYear,
			BranchName:  s.BranchName,
			HasOffering: s.OfferingID != nil,
		})
	}

	c.JSON(http.StatusOK, result)
}

func AssignFacultyToCourse(c *gin.Context) {
	var req struct {
		FacultyRegdNo string `json:"faculty_regdno" binding:"required"`
		SectionIDs    []uint `json:"section_ids" binding:"required,min=1"`
		CourseID      uint   `json:"course_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify faculty exists
	var faculty models.Faculty
	if err := database.DB.Where("regd_no = ?", req.FacultyRegdNo).First(&faculty).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Faculty not found"})
		return
	}

	// Verify course exists
	var course models.Course
	if err := database.DB.First(&course, req.CourseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Get HOD's branch for validation
	hodRegdNo, _ := c.Get("regdno")
	var hodUser models.User
	if err := database.DB.Where("regdno = ?", hodRegdNo).First(&hodUser).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Failed to get HOD information"})
		return
	}
	if hodUser.BranchID == nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "HOD must be assigned to a branch first"})
		return
	}

	// Verify all requested sections exist and belong to HOD's branch
	var requestedSections []models.Section
	if err := database.DB.Where("section_id IN ? AND branch_id = ? AND deleted_at IS NULL", req.SectionIDs, *hodUser.BranchID).Find(&requestedSections).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify sections"})
		return
	}
	if len(requestedSections) != len(req.SectionIDs) {
		foundIDs := make(map[uint]bool)
		for _, s := range requestedSections {
			foundIDs[s.SectionID] = true
		}
		var missing []uint
		for _, id := range req.SectionIDs {
			if !foundIDs[id] {
				missing = append(missing, id)
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Some sections do not exist or do not belong to your branch: %v", missing)})
		return
	}

	// Get current (or latest) academic year
	var academicYear models.AcademicYear
	if err := database.DB.Where("is_current = ?", true).First(&academicYear).Error; err != nil {
		database.DB.Order("start_date DESC").First(&academicYear)
	}

	offeringsAutoCreated := 0
	sectionsAssigned := 0
	var assignedSectionNames []string

	for _, sec := range requestedSections {
		// Find or create offering for this course + section
		var offering models.CourseOffering
		err := database.DB.Preload("Section").Where("course_id = ? AND section_id = ? AND is_active = ? AND deleted_at IS NULL",
			req.CourseID, sec.SectionID, true).First(&offering).Error

		if err != nil || offering.Section == nil {
			// Check if offering already exists for this academic year
			var existingOffering models.CourseOffering
			existingErr := database.DB.Where(
				"course_id = ? AND section_id = ? AND academic_year_id = ?",
				req.CourseID, sec.SectionID, academicYear.AcademicYearID,
			).First(&existingOffering).Error
			if existingErr != nil {
				// Create new offering
				newOffering := models.CourseOffering{
					CourseID:       req.CourseID,
					AcademicYearID: academicYear.AcademicYearID,
					SectionID:      sec.SectionID,
					IsActive:       true,
				}
				if err := database.DB.Create(&newOffering).Error; err == nil {
					offeringsAutoCreated++
				}
			}

			// Retry finding the offering
			if err := database.DB.Preload("Section").Where("course_id = ? AND section_id = ? AND is_active = ? AND deleted_at IS NULL",
				req.CourseID, sec.SectionID, true).First(&offering).Error; err != nil {
				continue // Skip this section if we still can't find it
			}
			if offering.Section == nil {
				continue
			}
		}

		// Validate branch access
		if err := validateBranchAccessForAssignment(c, req.FacultyRegdNo, offering.ID); err != nil {
			continue // Skip sections where branch access fails
		}

		// Check if assignment already exists
		var existing models.FacultyAssignment
		err = database.DB.Where("faculty_regd_no = ? AND course_offering_id = ?",
			req.FacultyRegdNo, offering.ID).First(&existing).Error
		if err != nil {
			database.DB.Create(&models.FacultyAssignment{
				FacultyRegdNo:    req.FacultyRegdNo,
				CourseOfferingID: offering.ID,
				Role:             "teacher",
				IsActive:         true,
			})
		} else {
			existing.IsActive = true
			database.DB.Save(&existing)
		}

		sectionsAssigned++
		assignedSectionNames = append(assignedSectionNames, sec.SectionName)
	}

	message := fmt.Sprintf("Faculty assigned to %d section(s) for course %s", sectionsAssigned, course.CourseCode)
	if len(assignedSectionNames) > 0 {
		message = fmt.Sprintf("Faculty assigned to section(s) %s for course %s", strings.Join(assignedSectionNames, ", "), course.CourseCode)
	}
	if offeringsAutoCreated > 0 {
		message = fmt.Sprintf("%s (Auto-created %d course offerings)", message, offeringsAutoCreated)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                message,
		"sections_assigned":      sectionsAssigned,
		"offerings_auto_created": offeringsAutoCreated,
	})
}

// validateBranchAccessForAssignment validates that HOD, faculty, and course offering are all in the same branch
// Validation rules:
// - HOD.branch == Course.branch (via CourseOffering -> Section -> Branch)
// - Faculty.branch == HOD.branch
func validateBranchAccessForAssignment(c *gin.Context, facultyRegdNo string, offeringID uint) error {
	// Get HOD's branch from context
	hodRegdNo, exists := c.Get("regdno")
	if !exists {
		return fmt.Errorf("user not authenticated")
	}

	var hodUser models.User
	if err := database.DB.Where("regdno = ?", hodRegdNo).First(&hodUser).Error; err != nil {
		return fmt.Errorf("failed to get HOD information")
	}

	if hodUser.BranchID == nil {
		return fmt.Errorf("HOD must be assigned to a branch first")
	}

	// Get CourseOffering with Section to find the course's branch
	var offering models.CourseOffering
	if err := database.DB.Preload("Section").First(&offering, offeringID).Error; err != nil {
		return fmt.Errorf("course offering not found")
	}

	if offering.Section == nil {
		return fmt.Errorf("course offering has no associated section")
	}

	// Get Faculty's branch
	var faculty models.Faculty
	if err := database.DB.Where("regd_no = ?", facultyRegdNo).First(&faculty).Error; err != nil {
		return fmt.Errorf("faculty not found")
	}

	// Validate: Course offering's section must be in HOD's branch
	if offering.Section.BranchID != *hodUser.BranchID {
		return fmt.Errorf("course offering is not in HOD's branch")
	}

	// Validate: Faculty must be in HOD's branch
	if faculty.BranchID != *hodUser.BranchID {
		return fmt.Errorf("faculty is not in HOD's branch")
	}

	return nil
}
