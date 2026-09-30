package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CourseAnalyticsResponse contains sections for a course
type CourseAnalyticsResponse struct {
	Course   CourseInfo    `json:"course"`
	Sections []SectionInfo `json:"sections"`
}

type CourseInfo struct {
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	ID         uint   `json:"id"`
}

type SectionInfo struct {
	SectionName  string `json:"section_name"`
	AcademicYear string `json:"academic_year,omitempty"`
	SectionID    uint   `json:"section_id"`
	StudentCount int    `json:"student_count"`
	CohortYear   int    `json:"cohort_year"`
}

// GetCourseAnalytics returns section list for a course with student counts (batch-based)
// Validates: faculty.branch == course.offering.section.branch
func GetCourseAnalytics(c *gin.Context) {
	courseID := c.Param("id")
	userRegdNo, _ := c.Get("regdno")

	// Get faculty's branch for validation
	type FacultyInfo struct {
		BranchID uint
	}
	var facultyInfo FacultyInfo
	if err := database.DB.Table("faculties").
		Select("branch_id").
		Where("regd_no = ?", userRegdNo).
		First(&facultyInfo).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Faculty information not found"})
		return
	}

	// Verify faculty is assigned to this course (via faculty_assignments table)
	// AND course offerings are in faculty's branch
	var assignment models.FacultyAssignment
	if err := database.DB.Raw(`
		SELECT fa.*
		FROM faculty_assignments fa
		INNER JOIN course_offerings co ON co.id = fa.course_offering_id
		INNER JOIN sections s ON s.section_id = co.section_id
		WHERE fa.faculty_regd_no = ?
		  AND co.course_id = ?
		  AND s.branch_id = ?
		  AND fa.is_active = true
		LIMIT 1
	`, userRegdNo, courseID, facultyInfo.BranchID).First(&assignment).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized for this course (not assigned or branch mismatch)"})
		return
	}

	// Get course details
	var course models.Course
	if err := database.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Get all sections for this course with student counts (batch-based)
	// Only sections where faculty is assigned AND in faculty's branch
	type SectionResult struct {
		SectionName  string
		AcademicYear string
		SectionID    uint
		CohortYear   int
		StudentCount int64
	}

	var sections []SectionResult
	database.DB.Raw(`
		SELECT
			s.section_id,
			s.section_name,
			s.cohort_year,
			COUNT(DISTINCT e.student_regdno) as student_count,
			ay.name as academic_year
		FROM sections s
		INNER JOIN course_offerings co ON co.section_id = s.section_id
		INNER JOIN academic_years ay ON ay.academic_year_id = co.academic_year_id
		LEFT JOIN enrollments e ON e.course_offering_id = co.id
		INNER JOIN faculty_assignments fa ON fa.course_offering_id = co.id
		WHERE fa.faculty_regd_no = ?
		  AND co.course_id = ?
		  AND s.branch_id = ?
		  AND fa.is_active = true
		GROUP BY s.section_id, s.section_name, s.cohort_year, ay.name
		ORDER BY s.cohort_year DESC, s.section_name
	`, userRegdNo, courseID, facultyInfo.BranchID).Scan(&sections)

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

	response := CourseAnalyticsResponse{
		Course: CourseInfo{
			ID:         course.ID,
			CourseCode: course.CourseCode,
			CourseName: course.CourseName,
		},
		Sections: sectionInfos,
	}

	c.JSON(http.StatusOK, response)
}

// SectionAnalyticsResponse contains detailed section analytics
type SectionAnalyticsResponse struct {
	CourseCategory string           `json:"course_category"`
	SessionStats   []SessionStat    `json:"session_stats"`
	Students       []StudentSummary `json:"students"`
	Section        SectionSummary   `json:"section"`
}

type SectionSummary struct {
	SectionName   string `json:"section_name"`
	SectionID     uint   `json:"section_id"`
	TotalStudents int    `json:"total_students"`
	CohortYear    int    `json:"cohort_year"`
}

type ProblemStat struct {
	Title          string  `json:"title"`
	Difficulty     string  `json:"difficulty"`
	ID             uint    `json:"id"`
	CompletionRate float64 `json:"completion_rate"`
	AvgAttempts    float64 `json:"avg_attempts"`
	StudentsSolved int     `json:"students_solved"`
}

type SessionStat struct {
	SessionName    string        `json:"session_name"`
	TopicName      string        `json:"topic_name"`
	Problems       []ProblemStat `json:"problems"`
	SessionID      uint          `json:"session_id"`
	SessionOrder   int           `json:"session_order"`
	StudentsSolved int           `json:"students_solved"`
	Percentage     float64       `json:"percentage"`
	TotalProblems  int           `json:"total_problems"`
}

type StudentSummary struct {
	LastActive     time.Time `json:"last_active"`
	RegdNo         string    `json:"regdno,omitempty"`
	Name           string    `json:"name"`
	RollNumber     string    `json:"roll_number"`
	UserID         uint      `json:"user_id"`
	ProblemsSolved int       `json:"problems_solved"`
	TimeSpent      float64   `json:"time_spent_seconds"`
	CurrentStreak  int       `json:"current_streak"`
}

// GetSectionAnalytics returns detailed analytics for a section (batch-based)
// Validates: faculty.branch == section.branch AND faculty is assigned to this offering
func GetSectionAnalytics(c *gin.Context) {
	courseID := c.Param("id")
	sectionID := c.Param("sectionId")
	userRegdNo, _ := c.Get("regdno")

	// Get faculty's branch for validation
	type FacultyInfo struct {
		BranchID uint
	}
	var facultyInfo FacultyInfo
	if err := database.DB.Table("faculties").
		Select("branch_id").
		Where("regd_no = ?", userRegdNo).
		First(&facultyInfo).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Faculty information not found"})
		return
	}

	// Verify faculty is assigned to this course offering
	// AND the section is in faculty's branch
	var assignment models.FacultyAssignment
	if err := database.DB.Raw(`
		SELECT fa.*
		FROM faculty_assignments fa
		INNER JOIN course_offerings co ON co.id = fa.course_offering_id
		INNER JOIN sections s ON s.section_id = co.section_id
		WHERE fa.faculty_regd_no = ?
		  AND co.course_id = ?
		  AND co.section_id = ?
		  AND s.branch_id = ?
		  AND fa.is_active = true
		LIMIT 1
	`, userRegdNo, courseID, sectionID, facultyInfo.BranchID).First(&assignment).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized for this section (not assigned or branch mismatch)"})
		return
	}

	// Get section details
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

	// Get course to determine if it's a lab course
	var course models.Course
	if err := database.DB.First(&course, courseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	// Get lab sessions for this course (if it's a lab course)
	var sessionStats []SessionStat
	if course.CourseType == "lab" {
		// Get lab sessions for this course
		type SessionResult struct {
			SessionName  string `gorm:"column:session_name"`
			TopicName    string `gorm:"column:topic_name"`
			SessionID    uint   `gorm:"column:session_id"`
			SessionOrder int    `gorm:"column:session_order"`
			StudentCount int64  `gorm:"column:student_count"`
			ProblemCount int64  `gorm:"column:problem_count"`
		}

		var sessionResults []SessionResult
		database.DB.Raw(`
			SELECT
				ls.id as session_id,
				ls.session_name,
				ls.session_order,
				COALESCE(ls.topic_name, '') as topic_name,
				COUNT(DISTINCT upc.user_regd_no) as student_count,
				COUNT(DISTINCT p.id) as problem_count
			FROM lab_sessions ls
			INNER JOIN labs l ON l.id = ls.lab_id
			LEFT JOIN problems p ON p.lab_session_id = ls.id
			LEFT JOIN user_problem_completions upc ON upc.problem_id = p.id
				AND upc.user_regd_no IN (SELECT regd_no FROM students WHERE section_id = ?)
			WHERE l.course_id = ?
			GROUP BY ls.id, ls.session_name, ls.session_order, ls.topic_name
			ORDER BY ls.session_order, ls.id
		`, sectionID, courseID).Scan(&sessionResults)

		// Get per-problem stats for each session
		type ProblemResult struct {
			Title          string  `gorm:"column:title"`
			Difficulty     string  `gorm:"column:difficulty"`
			ProblemID      uint    `gorm:"column:problem_id"`
			SessionID      uint    `gorm:"column:session_id"`
			StudentsSolved int64   `gorm:"column:students_solved"`
			AvgAttempts    float64 `gorm:"column:avg_attempts"`
		}

		var problemResults []ProblemResult
		database.DB.Raw(`
			SELECT
				p.id as problem_id,
				p.title,
				p.difficulty,
				p.lab_session_id as session_id,
				COUNT(DISTINCT upc.user_regd_no) as students_solved,
				COALESCE(
					(SELECT AVG(sub_count) FROM (
						SELECT COUNT(*) as sub_count
						FROM submissions s2
						JOIN students st2 ON st2.regd_no = s2.user_regd_no
						WHERE s2.problem_id = p.id AND st2.section_id = ?
						GROUP BY s2.user_regd_no
					) avg_sub), 0
				) as avg_attempts
			FROM problems p
			INNER JOIN lab_sessions ls ON ls.id = p.lab_session_id
			INNER JOIN labs l ON l.id = ls.lab_id
			LEFT JOIN user_problem_completions upc ON upc.problem_id = p.id
				AND upc.user_regd_no IN (SELECT regd_no FROM students WHERE section_id = ?)
			WHERE l.course_id = ?
			GROUP BY p.id, p.title, p.difficulty, p.lab_session_id, ls.session_order
			ORDER BY ls.session_order, p.id
		`, sectionID, sectionID, courseID).Scan(&problemResults)

		// Build a map of session_id -> []ProblemStat
		problemsBySession := make(map[uint][]ProblemStat)
		for _, pr := range problemResults {
			completionRate := 0.0
			if totalStudents > 0 {
				completionRate = (float64(pr.StudentsSolved) / float64(totalStudents)) * 100
			}
			problemsBySession[pr.SessionID] = append(problemsBySession[pr.SessionID], ProblemStat{
				ID:             pr.ProblemID,
				Title:          pr.Title,
				Difficulty:     pr.Difficulty,
				CompletionRate: completionRate,
				AvgAttempts:    pr.AvgAttempts,
				StudentsSolved: int(pr.StudentsSolved),
			})
		}

		for _, sr := range sessionResults {
			percentage := 0.0
			if totalStudents > 0 {
				percentage = (float64(sr.StudentCount) / float64(totalStudents)) * 100
			}

			sessionStats = append(sessionStats, SessionStat{
				SessionID:      sr.SessionID,
				SessionName:    sr.SessionName,
				SessionOrder:   sr.SessionOrder,
				TopicName:      sr.TopicName,
				StudentsSolved: int(sr.StudentCount),
				Percentage:     percentage,
				TotalProblems:  int(sr.ProblemCount),
				Problems:       problemsBySession[sr.SessionID],
			})
		}
	} else {
		// For theory courses, fall back to topic-based grouping
		// This maintains backward compatibility for non-lab courses
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
			LEFT JOIN students st ON st.regd_no = upc.user_regd_no
			WHERE st.section_id = ? AND p.tags IS NOT NULL AND p.tags != '' AND p.course_id = ?
			GROUP BY p.tags
		`, sectionID, courseID).Scan(&topicResults)

		// Convert TopicResults to SessionStats for unified response
		for i, tr := range topicResults {
			percentage := 0.0
			if totalStudents > 0 {
				percentage = (float64(tr.StudentCount) / float64(totalStudents)) * 100
			}

			sessionStats = append(sessionStats, SessionStat{
				SessionID:      uint(i + 1),
				SessionName:    tr.Tag,
				SessionOrder:   i + 1,
				TopicName:      tr.Tag,
				StudentsSolved: int(tr.StudentCount),
				Percentage:     percentage,
				TotalProblems:  int(tr.ProblemCount),
				Problems:       []ProblemStat{}, // Would need additional query for theory course problems
			})
		}
	}

	// Get student list with summary stats - only students in this section
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

	fmt.Printf("DEBUG: Section %s students returned: %d\n", sectionID, len(studentResults))

	var students []StudentSummary
	for _, sr := range studentResults {
		lastActive := time.Time{}
		if sr.LastActive != nil {
			lastActive = *sr.LastActive
		}

		fmt.Printf("DEBUG: Student - RegdNo: '%s', Name: '%s'\n", sr.RegdNo, sr.Name)

		students = append(students, StudentSummary{
			RegdNo:         sr.RegdNo,
			Name:           sr.Name,
			ProblemsSolved: int(sr.ProblemsSolved),
			TimeSpent:      sr.TimeSpent,
			CurrentStreak:  sr.CurrentStreak,
			LastActive:     lastActive,
		})
	}

	fmt.Printf("DEBUG: Final students array length: %d\n", len(students))

	response := SectionAnalyticsResponse{
		Section: SectionSummary{
			SectionID:     section.SectionID,
			SectionName:   section.SectionName,
			TotalStudents: int(totalStudents),
			CohortYear:    section.CohortYear,
		},
		CourseCategory: course.CourseType,
		SessionStats:   sessionStats,
		Students:       students,
	}

	c.JSON(http.StatusOK, response)
}

// StudentAnalyticsResponse contains detailed student performance data
type StudentAnalyticsResponse struct {
	LastActive       time.Time          `json:"last_active"`
	Student          StudentInfo        `json:"student"`
	Activity         []ActivityDay      `json:"activity"`
	TopicPerformance []TopicPerformance `json:"topic_performance"`
	Stats            StudentStats       `json:"stats"`
}

type StudentInfo struct {
	RegdNo string `json:"regdno"`
	Name   string `json:"name"`
}

type StudentStats struct {
	TotalTimeSeconds  float64 `json:"total_time_seconds"`
	ProblemsSolved    int     `json:"problems_solved"`
	CurrentStreak     int     `json:"current_streak"`
	LongestStreak     int     `json:"longest_streak"`
	Accuracy          float64 `json:"accuracy"`
	TotalSubmissions  int     `json:"total_submissions"`
	PassedSubmissions int     `json:"passed_submissions"`
}

type TopicPerformance struct {
	Topic  string `json:"topic"`
	Solved int    `json:"solved"`
	Total  int    `json:"total"`
}

// GetStudentAnalytics returns detailed analytics for a specific student
// Validates: faculty is assigned to a course offering that the student is enrolled in AND branch matches
func GetStudentAnalytics(c *gin.Context) {
	studentID := c.Param("id")
	userRegdNo, _ := c.Get("regdno")

	// Get faculty's branch for validation
	type FacultyInfo struct {
		BranchID uint
	}
	var facultyInfo FacultyInfo
	if err := database.DB.Table("faculties").
		Select("branch_id").
		Where("regd_no = ?", userRegdNo).
		First(&facultyInfo).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Faculty information not found"})
		return
	}

	// Get student details
	var user models.User
	if err := database.DB.First(&user, "regdno = ?", studentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	// Get student details from Student table
	var student models.Student
	database.DB.Where("regd_no = ?", studentID).First(&student)

	// Validate: faculty can only view students from sections in their branch
	// The faculty already has access to the section from GetSectionAnalytics, so we just verify the branch
	if student.SectionID != nil {
		var sectionBranchID uint
		database.DB.Table("sections").Where("section_id = ?", *student.SectionID).Select("branch_id").Scan(&sectionBranchID)

		// Check if student's section is in faculty's branch
		if sectionBranchID != facultyInfo.BranchID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to view this student's data (branch mismatch)"})
			return
		}
	}

	// Calculate stats
	// FIX: Time spent now uses first_submission_id from user_problem_completions
	// to avoid counting multiple submissions for the same problem
	// Only the time from the first passing submission is counted per problem
	var totalTimeSpent float64
	database.DB.Raw(`
		SELECT COALESCE(SUM(s.time_spent), 0)
		FROM submissions s
		INNER JOIN user_problem_completions upc ON s.id = upc.first_submission_id
		WHERE upc.user_regd_no = ?
	`, studentID).Scan(&totalTimeSpent)

	var problemsSolved int64
	database.DB.Model(&models.UserProblemCompletion{}).
		Where("user_regd_no = ?", studentID).
		Count(&problemsSolved)

	var totalSubmissions, passedSubmissions int64
	database.DB.Model(&models.Submission{}).
		Where("user_regd_no = ?", studentID).
		Count(&totalSubmissions)
	database.DB.Model(&models.Submission{}).
		Where("user_regd_no = ? AND passed = ?", studentID, true).
		Count(&passedSubmissions)

	accuracy := 0.0
	if totalSubmissions > 0 {
		accuracy = (float64(passedSubmissions) / float64(totalSubmissions)) * 100
	}

	var streak models.UserStreak
	currentStreak := 0
	longestStreak := 0
	if err := database.DB.Where("user_regd_no = ?", studentID).First(&streak).Error; err == nil {
		currentStreak = streak.CurrentStreak
		longestStreak = streak.LongestStreak
	}

	// Get activity data (reuse from dashboard)
	activity := getActivity(studentID)

	// Get topic performance
	type TopicResult struct {
		Tag    string
		Solved int64
		Total  int64
	}

	var topicResults []TopicResult
	database.DB.Raw(`
		SELECT
			p.tags as tag,
			COUNT(DISTINCT CASE WHEN upc.user_regd_no = ? THEN p.id END) as solved,
			COUNT(DISTINCT p.id) as total
		FROM problems p
		LEFT JOIN user_problem_completions upc ON upc.problem_id = p.id AND upc.user_regd_no = ?
		WHERE p.tags IS NOT NULL AND p.tags != ''
		GROUP BY p.tags
	`, studentID, studentID).Scan(&topicResults)

	var topicPerformance []TopicPerformance
	for _, tr := range topicResults {
		topicPerformance = append(topicPerformance, TopicPerformance{
			Topic:  tr.Tag,
			Solved: int(tr.Solved),
			Total:  int(tr.Total),
		})
	}

	// Get last active date
	var lastActive time.Time
	database.DB.Model(&models.UserActivity{}).
		Where("user_regd_no = ?", studentID).
		Select("MAX(date)").
		Scan(&lastActive)

	response := StudentAnalyticsResponse{
		Student: StudentInfo{
			RegdNo: user.RegdNo,
			Name:   user.Name,
		},
		Stats: StudentStats{
			TotalTimeSeconds:  totalTimeSpent,
			ProblemsSolved:    int(problemsSolved),
			CurrentStreak:     currentStreak,
			LongestStreak:     longestStreak,
			Accuracy:          accuracy,
			TotalSubmissions:  int(totalSubmissions),
			PassedSubmissions: int(passedSubmissions),
		},
		Activity:         activity,
		TopicPerformance: topicPerformance,
		LastActive:       lastActive,
	}

	c.JSON(http.StatusOK, response)
}
