package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetCourses returns courses based on user role (batch-based)
// For students: filters by enrollments AND validates branch/cohort/program match
func GetCourses(c *gin.Context) {
	var courses []models.Course

	// Check if user is authenticated
	userRegdNo, userExists := c.Get("regdno")
	role, roleExists := c.Get("role")

	// If authenticated and is a student, return only enrolled courses that match their branch/cohort/program
	if userExists && roleExists && role == "student" {
		// Get student's branch, cohort, and program info for validation
		type StudentInfo struct {
			BranchID   uint
			CohortYear int
			ProgramID  uint
		}
		var studentInfo StudentInfo
		if err := database.DB.Raw(`
			SELECT st.branch_id, st.cohort_year, b.program_id
			FROM students st
			INNER JOIN branches b ON b.branch_id = st.branch_id
			WHERE st.regd_no = ?
		`, userRegdNo).First(&studentInfo).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get student information"})
			return
		}

		// Get enrolled courses that match student's branch, cohort year, and program
		// Join with sections to validate branch/cohort/program
		type EnrolledCourse struct {
			CourseCode string
			CourseName string
			CourseType string
			CourseID   uint
			Credits    int
		}

		var enrolledCourses []EnrolledCourse
		result := database.DB.Raw(`
			SELECT DISTINCT
				c.id as course_id,
				c.course_code,
				c.course_name,
				c.course_type,
				c.credits
			FROM enrollments e
			INNER JOIN course_offerings co ON co.id = e.course_offering_id
			INNER JOIN sections s ON s.section_id = co.section_id
			INNER JOIN branches b ON b.branch_id = s.branch_id
			INNER JOIN courses c ON c.id = co.course_id
			WHERE e.student_regdno = ?
			  AND e.status IN ('enrolled', 'completed')
			  AND co.is_active = true
			  AND co.deleted_at IS NULL
			  AND s.branch_id = ?
			  AND s.cohort_year = ?
			  AND b.program_id = ?
			ORDER BY c.course_code
		`, userRegdNo, studentInfo.BranchID, studentInfo.CohortYear, studentInfo.ProgramID).Scan(&enrolledCourses)

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch enrolled courses"})
			return
		}

		for _, ec := range enrolledCourses {
			course := models.Course{
				ID:         ec.CourseID,
				CourseCode: ec.CourseCode,
				CourseName: ec.CourseName,
				CourseType: ec.CourseType,
				Credits:    ec.Credits,
				IsActive:   true,
			}
			courses = append(courses, course)
		}

	} else if userExists && roleExists && role == "faculty" {
		// Faculty: get courses they are assigned to via FacultyAssignment
		// Also filter by faculty's branch AND college
		type FacultyInfo struct {
			CollegeID string
			BranchID  uint
		}
		var facultyInfo FacultyInfo
		database.DB.Raw(`
			SELECT f.branch_id, u.college_id
			FROM faculties f
			INNER JOIN users u ON u.regdno = f.regd_no
			WHERE f.regd_no = ?
		`, userRegdNo).First(&facultyInfo)

		type AssignedCourse struct {
			CourseCode  string
			CourseName  string
			CourseType  string
			SectionName string
			CourseID    uint
			Credits     int
			CohortYear  int
		}

		var assignedCourses []AssignedCourse
		database.DB.Raw(`
			SELECT DISTINCT
				co.course_id,
				c.course_code,
				c.course_name,
				c.course_type,
				c.credits,
				s.section_name,
				s.cohort_year
			FROM faculty_assignments fa
			INNER JOIN course_offerings co ON co.id = fa.course_offering_id
			INNER JOIN sections s ON s.section_id = co.section_id
			INNER JOIN courses c ON c.id = co.course_id
			WHERE fa.faculty_regd_no = ?
			  AND fa.is_active = true
			  AND s.branch_id = ?
			  AND c.college_id = ?
			ORDER BY c.course_code, s.section_name
		`, userRegdNo, facultyInfo.BranchID, facultyInfo.CollegeID).Scan(&assignedCourses)

		for _, ac := range assignedCourses {
			course := models.Course{
				ID:         ac.CourseID,
				CourseCode: ac.CourseCode,
				CourseName: ac.CourseName,
				CourseType: ac.CourseType,
				Credits:    ac.Credits,
				IsActive:   true,
			}
			courses = append(courses, course)
		}
	} else if userExists && roleExists && (role == "hod" || role == "college_admin" || role == "admin") {
		// HOD/Admin: get only courses from their college
		var user models.User
		database.DB.Where("regdno = ?", userRegdNo).First(&user)

		if user.CollegeID != nil {
			database.DB.Where("is_active = ? AND college_id = ?", true, *user.CollegeID).
				Preload("Lab").
				Preload("Theory").
				Find(&courses)
		} else {
			// If no college assigned, return empty
			courses = []models.Course{}
		}
	} else {
		// For unauthenticated users, return empty (or could return all if public access is allowed)
		courses = []models.Course{}
	}

	c.JSON(http.StatusOK, courses)
}

// GetCourse returns a single course by ID
// For students: validates enrollment AND branch/cohort/program match
func GetCourse(c *gin.Context) {
	id := c.Param("id")

	// For students, validate access before showing course details
	role, roleExists := c.Get("role")
	if roleExists && role == "student" {
		userRegdNo, _ := c.Get("regdno")

		// Get student's info including program
		type StudentInfo struct {
			BranchID   uint
			CohortYear int
			ProgramID  uint
		}
		var studentInfo StudentInfo
		if err := database.DB.Raw(`
			SELECT st.branch_id, st.cohort_year, b.program_id
			FROM students st
			INNER JOIN branches b ON b.branch_id = st.branch_id
			WHERE st.regd_no = ?
		`, userRegdNo).First(&studentInfo).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Student information not found"})
			return
		}

		// Check if student is enrolled in this course AND branch/cohort/program match
		var enrollmentCount int64
		database.DB.Raw(`
			SELECT COUNT(*)
			FROM enrollments e
			INNER JOIN course_offerings co ON co.id = e.course_offering_id
			INNER JOIN sections s ON s.section_id = co.section_id
			INNER JOIN branches b ON b.branch_id = s.branch_id
			WHERE e.student_regdno = ?
			  AND co.course_id = ?
			  AND e.status IN ('enrolled', 'completed')
			  AND co.is_active = true
			  AND co.deleted_at IS NULL
			  AND s.branch_id = ?
			  AND s.cohort_year = ?
			  AND b.program_id = ?
		`, userRegdNo, id, studentInfo.BranchID, studentInfo.CohortYear, studentInfo.ProgramID).Count(&enrollmentCount)

		if enrollmentCount == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not enrolled in this course or access denied"})
			return
		}
	}

	var course models.Course
	if err := database.DB.Preload("Lab").Preload("Theory").Preload("Theory.Weeks", func(db *gorm.DB) *gorm.DB {
		return db.Order("theory_weeks.week_order ASC")
	}).Preload("Theory.Weeks.Modules").First(&course, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Fetch Lab Sessions if Lab exists
	var labSessions []models.LabSession
	if course.Lab != nil {
		database.DB.Where("lab_id = ?", course.Lab.ID).Order("session_order").Find(&labSessions)
	}

	// Extract theory weeks if Theory exists
	var theoryWeeks []models.TheoryWeek
	if course.Theory != nil && len(course.Theory.Weeks) > 0 {
		theoryWeeks = course.Theory.Weeks
	}

	// Build response — include theory_id so students can fetch PDFs
	response := gin.H{
		"course":       course,
		"lab_sessions": labSessions,
		"theory_weeks": theoryWeeks,
	}
	if course.Theory != nil {
		response["theory_id"] = course.Theory.ID
	}

	c.JSON(http.StatusOK, response)
}

// GetLabTopics -> renamed/mapped to GetLabSessions
func GetLabTopics(c *gin.Context) {
	// Re-route to GetCourseLabSessions logic or return error
	courseID := c.Param("id")

	var lab models.Lab
	if err := database.DB.Where("course_id = ?", courseID).First(&lab).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lab not found"})
		return
	}

	var sessions []models.LabSession
	database.DB.Where("lab_id = ?", lab.ID).Order("session_order").Find(&sessions)

	c.JSON(http.StatusOK, sessions)
}

// GetTopicProblems -> GetSessionProblems
func GetTopicProblems(c *gin.Context) {
	sessionID := c.Param("id")

	var session models.LabSession
	if err := database.DB.First(&session, sessionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	var problems []models.Problem
	database.DB.Where("lab_session_id = ?", sessionID).Find(&problems)

	// Get user's completed problems if authenticated
	userRegdNo, exists := c.Get("regdno")
	completedProblemIDs := make(map[uint]bool)
	problemTimeTaken := make(map[uint]float64)

	if exists && userRegdNo != nil {
		var completions []models.UserProblemCompletion
		database.DB.Where("user_regd_no = ?", userRegdNo).Find(&completions)
		for _, comp := range completions {
			completedProblemIDs[comp.ProblemID] = true
			problemTimeTaken[comp.ProblemID] = comp.TimeTakenSeconds
		}
	}

	// Create response with completion status
	type ProblemWithStatus struct {
		models.Problem
		IsCompleted      bool    `json:"is_completed"`
		TimeTakenSeconds float64 `json:"time_taken_seconds,omitempty"`
	}

	problemsWithStatus := make([]ProblemWithStatus, len(problems))
	for i, p := range problems {
		problemsWithStatus[i] = ProblemWithStatus{
			Problem:          p,
			IsCompleted:      completedProblemIDs[p.ID],
			TimeTakenSeconds: problemTimeTaken[p.ID],
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"session":  session,
		"problems": problemsWithStatus,
	})
}

// TopicsOverviewSession represents a lab session with progress info
type TopicsOverviewSession struct {
	SessionName   string `json:"session_name"`
	ID            uint   `json:"id"`
	SessionOrder  int    `json:"session_order"`
	TotalProblems int    `json:"total_problems"`
	Solved        int    `json:"solved"`
	EasyTotal     int    `json:"easy_total"`
	EasySolved    int    `json:"easy_solved"`
	MediumTotal   int    `json:"medium_total"`
	MediumSolved  int    `json:"medium_solved"`
	HardTotal     int    `json:"hard_total"`
	HardSolved    int    `json:"hard_solved"`
	Progress      int    `json:"progress"`
}

// TopicsOverviewCourse represents a course with its lab sessions
type TopicsOverviewCourse struct {
	CourseCode  string                  `json:"course_code"`
	CourseName  string                  `json:"course_name"`
	CourseType  string                  `json:"course_type"`
	LabSessions []TopicsOverviewSession `json:"lab_sessions"`
	ID          uint                    `json:"id"`
	Credits     int                     `json:"credits"`
}

// GetTopicsOverview returns all enrolled courses with lab session progress for the student
func GetTopicsOverview(c *gin.Context) {
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	role, _ := c.Get("role")
	if role != "student" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only students can access this endpoint"})
		return
	}

	regdNo := userRegdNo.(string)

	// Get student info
	type StudentInfo struct {
		BranchID   uint
		CohortYear int
		ProgramID  uint
	}
	var studentInfo StudentInfo
	if err := database.DB.Raw(`
		SELECT st.branch_id, st.cohort_year, b.program_id
		FROM students st
		INNER JOIN branches b ON b.branch_id = st.branch_id
		WHERE st.regd_no = ?
	`, regdNo).First(&studentInfo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get student information"})
		return
	}

	// Get enrolled courses
	type EnrolledCourse struct {
		CourseCode string
		CourseName string
		CourseType string
		CourseID   uint
		Credits    int
	}

	var enrolledCourses []EnrolledCourse
	database.DB.Raw(`
		SELECT DISTINCT
			c.id as course_id,
			c.course_code,
			c.course_name,
			c.course_type,
			c.credits
		FROM enrollments e
		INNER JOIN course_offerings co ON co.id = e.course_offering_id
		INNER JOIN sections s ON s.section_id = co.section_id
		INNER JOIN branches b ON b.branch_id = s.branch_id
		INNER JOIN courses c ON c.id = co.course_id
		WHERE e.student_regdno = ?
		  AND e.status IN ('enrolled', 'completed')
		  AND co.is_active = true
		  AND co.deleted_at IS NULL
		  AND s.branch_id = ?
		  AND s.cohort_year = ?
		  AND b.program_id = ?
		ORDER BY c.course_code
	`, regdNo, studentInfo.BranchID, studentInfo.CohortYear, studentInfo.ProgramID).Scan(&enrolledCourses)

	// Build response
	var result []TopicsOverviewCourse

	for _, ec := range enrolledCourses {
		course := TopicsOverviewCourse{
			ID:         ec.CourseID,
			CourseCode: ec.CourseCode,
			CourseName: ec.CourseName,
			CourseType: ec.CourseType,
			Credits:    ec.Credits,
		}

		// Get lab for this course
		var lab models.Lab
		if err := database.DB.Where("course_id = ?", ec.CourseID).First(&lab).Error; err != nil {
			// No lab for this course (theory-only), still add with empty sessions
			course.LabSessions = []TopicsOverviewSession{}
			result = append(result, course)
			continue
		}

		// Get lab sessions with problem stats in one query
		type SessionStats struct {
			SessionName   string
			SessionID     uint
			SessionOrder  int
			TotalProblems int
			EasyTotal     int
			MediumTotal   int
			HardTotal     int
		}

		var sessionStats []SessionStats
		database.DB.Raw(`
			SELECT
				ls.id as session_id,
				ls.session_name,
				ls.session_order,
				COUNT(p.id) as total_problems,
				SUM(CASE WHEN p.difficulty = 'easy' THEN 1 ELSE 0 END) as easy_total,
				SUM(CASE WHEN p.difficulty = 'medium' THEN 1 ELSE 0 END) as medium_total,
				SUM(CASE WHEN p.difficulty = 'hard' THEN 1 ELSE 0 END) as hard_total
			FROM lab_sessions ls
			LEFT JOIN problems p ON p.lab_session_id = ls.id AND p.deleted_at IS NULL
			WHERE ls.lab_id = ? AND ls.deleted_at IS NULL
			GROUP BY ls.id, ls.session_name, ls.session_order
			ORDER BY ls.session_order
		`, lab.ID).Scan(&sessionStats)

		// Get solved counts per session for this student
		type SolvedStats struct {
			LabSessionID uint
			Solved       int
			EasySolved   int
			MediumSolved int
			HardSolved   int
		}

		var solvedStats []SolvedStats
		database.DB.Raw(`
			SELECT
				p.lab_session_id,
				COUNT(upc.id) as solved,
				SUM(CASE WHEN p.difficulty = 'easy' THEN 1 ELSE 0 END) as easy_solved,
				SUM(CASE WHEN p.difficulty = 'medium' THEN 1 ELSE 0 END) as medium_solved,
				SUM(CASE WHEN p.difficulty = 'hard' THEN 1 ELSE 0 END) as hard_solved
			FROM user_problem_completions upc
			INNER JOIN problems p ON p.id = upc.problem_id AND p.deleted_at IS NULL
			INNER JOIN lab_sessions ls ON ls.id = p.lab_session_id AND ls.deleted_at IS NULL
			WHERE upc.user_regd_no = ? AND ls.lab_id = ?
			GROUP BY p.lab_session_id
		`, regdNo, lab.ID).Scan(&solvedStats)

		// Build solved map
		solvedMap := make(map[uint]SolvedStats)
		for _, s := range solvedStats {
			solvedMap[s.LabSessionID] = s
		}

		// Combine into sessions
		var sessions []TopicsOverviewSession
		for _, ss := range sessionStats {
			solved := solvedMap[ss.SessionID]
			progress := 0
			if ss.TotalProblems > 0 {
				progress = solved.Solved * 100 / ss.TotalProblems
			}

			sessions = append(sessions, TopicsOverviewSession{
				ID:            ss.SessionID,
				SessionName:   ss.SessionName,
				SessionOrder:  ss.SessionOrder,
				TotalProblems: ss.TotalProblems,
				Solved:        solved.Solved,
				EasyTotal:     ss.EasyTotal,
				EasySolved:    solved.EasySolved,
				MediumTotal:   ss.MediumTotal,
				MediumSolved:  solved.MediumSolved,
				HardTotal:     ss.HardTotal,
				HardSolved:    solved.HardSolved,
				Progress:      progress,
			})
		}

		if sessions == nil {
			sessions = []TopicsOverviewSession{}
		}
		course.LabSessions = sessions
		result = append(result, course)
	}

	if result == nil {
		result = []TopicsOverviewCourse{}
	}

	c.JSON(http.StatusOK, result)
}
