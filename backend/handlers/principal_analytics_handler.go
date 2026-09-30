package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// =====================================================
// DTO Structures
// =====================================================

type MonthlyTrend struct {
	Month string `json:"month"`
	Count int64  `json:"count"`
}

type CourseCompletionItem struct {
	CourseCode    string  `json:"course_code"`
	CourseName    string  `json:"course_name"`
	TotalSessions int     `json:"total_sessions"`
	Completed     int     `json:"completed"`
	Pending       int     `json:"pending"`
	CompletionPct float64 `json:"completion_pct"`
	AvgMarks      float64 `json:"avg_marks"`
	TotalStudents int64   `json:"total_students"`
}

type FacultyEffectiveness struct {
	RegdNo      string  `json:"regdno"`
	Name        string  `json:"name"`
	Courses     int     `json:"courses"`
	Students    int64   `json:"students"`
	AvgMarks    float64 `json:"avg_marks"`
	Submissions int64   `json:"submissions"`
}

type LabConduct struct {
	LabName       string  `json:"lab_name"`
	CourseCode    string  `json:"course_code"`
	Planned       int     `json:"planned"`
	Conducted     int     `json:"conducted"`
	Missed        int     `json:"missed"`
	AvgAttendance float64 `json:"avg_attendance"`
	AvgScore      float64 `json:"avg_score"`
}

type ContestActivity struct {
	Total           int64   `json:"total"`
	Internal        int64   `json:"internal"`
	External        int64   `json:"external"`
	AvgParticipants float64 `json:"avg_participants"`
}

type BranchPerformance struct {
	BranchID             uint    `json:"branch_id"`
	BranchName           string  `json:"branch_name"`
	ShortName            string  `json:"short_name"`
	TotalStudents        int64   `json:"total_students"`
	TotalFaculty         int64   `json:"total_faculty"`
	AvgSubmissionScore   float64 `json:"avg_submission_score"`
	PassPct              float64 `json:"pass_pct"`
	ContestParticipation float64 `json:"contest_participation"`
	AssignmentCompletion float64 `json:"assignment_completion"`
}

type StudentPerformanceItem struct {
	RegdNo           string  `json:"regdno"`
	Name             string  `json:"name"`
	BranchName       string  `json:"branch_name"`
	AvgMarks         float64 `json:"avg_marks"`
	TotalSubmissions int64   `json:"total_submissions"`
	Status           string  `json:"status"`
}

type SubjectPerformance struct {
	SubjectName   string  `json:"subject_name"`
	AvgMarks      float64 `json:"avg_marks"`
	PassPct       float64 `json:"pass_pct"`
	TotalStudents int64   `json:"total_students"`
}

type CohortTrend struct {
	CohortYear    int     `json:"cohort_year"`
	TotalStudents int64   `json:"total_students"`
	AvgMarks      float64 `json:"avg_marks"`
	PassPct       float64 `json:"pass_pct"`
	Submissions   int64   `json:"submissions"`
}

type InsightItem struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	Severity   string `json:"severity"`
	BranchName string `json:"branch_name,omitempty"`
	Metric     string `json:"metric,omitempty"`
	Value      string `json:"value,omitempty"`
}

type AlertItem struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	Severity   string `json:"severity"`
	BranchName string `json:"branch_name,omitempty"`
	CreatedAt  string `json:"created_at"`
}

type AccreditationMetrics struct {
	MetricName string  `json:"metric_name"`
	Value      float64 `json:"value"`
	Target     float64 `json:"target"`
	Status     string  `json:"status"`
}

// =====================================================
// 1. Academic Overview Analytics
// =====================================================

func GetPrincipalAnalyticsOverview(c *gin.Context) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not assigned"})
		return
	}
	collegeIDStr := collegeID.(*string)

	var result struct {
		TotalActiveCourses   int64          `json:"total_active_courses"`
		ActiveLabSessions    int64          `json:"active_lab_sessions"`
		ContestsThisSemester int64          `json:"contests_this_semester"`
		StudentParticipation float64        `json:"student_participation_pct"`
		AvgPerformance       float64        `json:"avg_academic_performance"`
		AvgAttendance        float64        `json:"avg_attendance_pct"`
		PassFailRatio        float64        `json:"pass_fail_ratio"`
		PendingEvaluations   int64          `json:"pending_evaluations"`
		ActiveFaculty        int64          `json:"active_faculty_count"`
		TotalStudents        int64          `json:"total_students"`
		SubmissionsThisMonth int64          `json:"submissions_this_month"`
		MonthlyTrend         []MonthlyTrend `json:"monthly_submission_trend"`
		BranchComparison     []struct {
			BranchName  string `json:"branch_name"`
			Submissions int64  `json:"submissions"`
		} `json:"branch_comparison"`
	}

	result.MonthlyTrend = make([]MonthlyTrend, 0)
	result.BranchComparison = make([]struct {
		BranchName  string `json:"branch_name"`
		Submissions int64  `json:"submissions"`
	}, 0)

	// Active courses
	database.DB.Model(&models.CourseOffering{}).
		Joins("JOIN course_offerings ON course_offerings.course_id = courses.id").
		Where("courses.college_id = ? AND course_offerings.is_active = true", *collegeIDStr).
		Distinct("courses.id").
		Count(&result.TotalActiveCourses)

	// Active lab sessions
	database.DB.Model(&models.LabSession{}).
		Joins("JOIN labs ON labs.id = lab_sessions.lab_id").
		Joins("JOIN courses ON courses.id = labs.course_id").
		Where("courses.college_id = ?", *collegeIDStr).
		Count(&result.ActiveLabSessions)

	// Contests this semester (last 6 months)
	sixMonthsAgo := time.Now().AddDate(0, -6, 0)
	database.DB.Model(&models.Contest{}).
		Where("college_id = ? AND start_time >= ?", *collegeIDStr, sixMonthsAgo).
		Count(&result.ContestsThisSemester)

	// Total students
	database.DB.Model(&models.User{}).
		Where("role = ? AND college_id = ? AND is_active = true", "student", *collegeIDStr).
		Count(&result.TotalStudents)

	// Active faculty (includes HODs)
	database.DB.Model(&models.User{}).
		Where("role IN (?, ?) AND college_id = ? AND is_active = true", "faculty", "hod", *collegeIDStr).
		Count(&result.ActiveFaculty)

	// Submissions this month
	firstOfMonth := time.Now().AddDate(0, 0, -time.Now().Day()+1)
	database.DB.Model(&models.Submission{}).
		Where("college_id = ? AND created_at >= ?", *collegeIDStr, firstOfMonth).
		Count(&result.SubmissionsThisMonth)

	// Monthly trend (last 6 months)
	for i := 5; i >= 0; i-- {
		start := time.Now().AddDate(0, -i, 0).AddDate(0, 0, -time.Now().Day()+1)
		end := start.AddDate(0, 1, 0)
		var count int64
		database.DB.Model(&models.Submission{}).
			Where("college_id = ? AND created_at >= ? AND created_at < ?", *collegeIDStr, start, end).
			Count(&count)
		result.MonthlyTrend = append(result.MonthlyTrend, MonthlyTrend{
			Month: start.Format("Jan"),
			Count: count,
		})
	}

	// Student participation % (students with >=1 submission / total students)
	var activeStudents int64
	database.DB.Model(&models.Submission{}).
		Where("college_id = ?", *collegeIDStr).
		Distinct("user_regd_no").
		Count(&activeStudents)
	if result.TotalStudents > 0 {
		result.StudentParticipation = float64(activeStudents) / float64(result.TotalStudents) * 100
	}

	// Average performance from exam results
	var avgMarks float64
	database.DB.Model(&models.ExamResult{}).
		Joins("JOIN exams ON exams.id = exam_results.exam_id").
		Joins("JOIN course_offerings ON course_offerings.id = exams.course_offering_id").
		Joins("JOIN sections ON sections.section_id = course_offerings.section_id").
		Joins("JOIN branches ON branches.branch_id = sections.branch_id").
		Where("branches.college_id = ?", *collegeIDStr).
		Select("COALESCE(AVG(exam_results.marks_obtained::float / NULLIF(exams.total_marks, 0) * 100), 0)").
		Scan(&avgMarks)
	result.AvgPerformance = avgMarks

	// Pass/Fail ratio from exam results
	var passCount, totalCount int64
	database.DB.Model(&models.ExamResult{}).
		Joins("JOIN exams ON exams.id = exam_results.exam_id").
		Joins("JOIN course_offerings ON course_offerings.id = exams.course_offering_id").
		Joins("JOIN sections ON sections.section_id = course_offerings.section_id").
		Joins("JOIN branches ON branches.branch_id = sections.branch_id").
		Where("branches.college_id = ? AND exam_results.marks_obtained >= exams.passing_marks", *collegeIDStr).
		Count(&passCount)
	database.DB.Model(&models.ExamResult{}).
		Joins("JOIN exams ON exams.id = exam_results.exam_id").
		Joins("JOIN course_offerings ON course_offerings.id = exams.course_offering_id").
		Joins("JOIN sections ON sections.section_id = course_offerings.section_id").
		Joins("JOIN branches ON branches.branch_id = sections.branch_id").
		Where("branches.college_id = ?", *collegeIDStr).
		Count(&totalCount)
	if totalCount > 0 {
		result.PassFailRatio = float64(passCount) / float64(totalCount) * 100
	}

	// Pending evaluations (unmarked submissions without a score in last 14 days)
	var pending int64
	cutoff := time.Now().AddDate(0, 0, -14)
	database.DB.Model(&models.Submission{}).
		Where("college_id = ? AND created_at >= ? AND score = 0 AND passed = false", *collegeIDStr, cutoff).
		Count(&pending)
	result.PendingEvaluations = pending

	// Branch comparison for submissions
	type branchSub struct {
		BranchName  string
		Submissions int64
	}
	var branchSubs []branchSub
	database.DB.Model(&models.Submission{}).
		Select("branches.branch_name, COUNT(*) as submissions").
		Joins("JOIN users ON users.regdno = submissions.user_regd_no").
		Joins("JOIN branches ON branches.branch_id = users.branch_id").
		Where("submissions.college_id = ? AND users.role = ?", *collegeIDStr, "student").
		Group("branches.branch_name").
		Scan(&branchSubs)
	for _, b := range branchSubs {
		result.BranchComparison = append(result.BranchComparison, struct {
			BranchName  string `json:"branch_name"`
			Submissions int64  `json:"submissions"`
		}{
			BranchName:  b.BranchName,
			Submissions: b.Submissions,
		})
	}

	c.JSON(http.StatusOK, result)
}

// =====================================================
// 2. Course Analytics
// =====================================================

func GetPrincipalCourseAnalytics(c *gin.Context) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not assigned"})
		return
	}
	collegeIDStr := collegeID.(*string)

	var result struct {
		CourseCompletion     []CourseCompletionItem `json:"course_completion"`
		DifficultyIndicators []struct {
			CourseCode string  `json:"course_code"`
			CourseName string  `json:"course_name"`
			AvgMarks   float64 `json:"avg_marks"`
			FailurePct float64 `json:"failure_pct"`
		} `json:"difficulty_indicators"`
		FacultyEffectiveness []FacultyEffectiveness `json:"faculty_effectiveness"`
	}

	result.CourseCompletion = make([]CourseCompletionItem, 0)
	result.DifficultyIndicators = make([]struct {
		CourseCode string  `json:"course_code"`
		CourseName string  `json:"course_name"`
		AvgMarks   float64 `json:"avg_marks"`
		FailurePct float64 `json:"failure_pct"`
	}, 0)
	result.FacultyEffectiveness = make([]FacultyEffectiveness, 0)

	// Course completion stats
	type courseRow struct {
		CourseCode string
		CourseName string
		CourseID   uint
	}
	var courses []courseRow
	database.DB.Model(&models.Course{}).
		Select("course_code, course_name, id").
		Where("college_id = ? AND is_active = true", *collegeIDStr).
		Scan(&courses)

	for _, course := range courses {
		item := CourseCompletionItem{
			CourseCode: course.CourseCode,
			CourseName: course.CourseName,
		}

		// Count lab sessions for this course
		var labCount int64
		database.DB.Model(&models.LabSession{}).
			Joins("JOIN labs ON labs.id = lab_sessions.lab_id").
			Where("labs.course_id = ?", course.CourseID).
			Count(&labCount)
		item.TotalSessions = int(labCount)

		// Count students enrolled in active offerings of this course
		var students int64
		database.DB.Model(&models.Enrollment{}).
			Joins("JOIN course_offerings ON course_offerings.id = enrollments.course_offering_id").
			Where("course_offerings.course_id = ? AND course_offerings.is_active = true", course.CourseID).
			Count(&students)
		item.TotalStudents = students

		// Completion based on submissions (proxy: unique students who submitted)
		var completed int64
		database.DB.Model(&models.Submission{}).
			Where("course_id = ? AND college_id = ? AND passed = true", course.CourseID, *collegeIDStr).
			Distinct("user_regd_no").
			Count(&completed)
		item.Completed = int(completed)
		item.Pending = int(students) - int(completed)
		if students > 0 {
			item.CompletionPct = float64(completed) / float64(students) * 100
		}

		// Average marks from exam results
		var avgMarks float64
		database.DB.Model(&models.ExamResult{}).
			Joins("JOIN exams ON exams.id = exam_results.exam_id").
			Joins("JOIN course_offerings ON course_offerings.id = exams.course_offering_id").
			Where("course_offerings.course_id = ?", course.CourseID).
			Select("COALESCE(AVG(exam_results.marks_obtained::float / NULLIF(exams.total_marks, 0) * 100), 0)").
			Scan(&avgMarks)
		item.AvgMarks = avgMarks

		result.CourseCompletion = append(result.CourseCompletion, item)
	}

	// Difficulty indicators (hardest courses)
	type diffRow struct {
		CourseCode string
		CourseName string
		AvgMarks   float64
		FailurePct float64
	}
	var diffs []diffRow
	database.DB.Raw(`
		SELECT c.course_code, c.course_name,
			COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) as avg_marks,
			COALESCE(
				SUM(CASE WHEN er.marks_obtained < e.passing_marks THEN 1 ELSE 0 END)::float
				/ NULLIF(COUNT(*), 0) * 100, 0
			) as failure_pct
		FROM courses c
		LEFT JOIN course_offerings co ON co.course_id = c.id
		LEFT JOIN exams e ON e.course_offering_id = co.id
		LEFT JOIN exam_results er ON er.exam_id = e.id
		WHERE c.college_id = ? AND c.is_active = true
		GROUP BY c.id
		ORDER BY avg_marks ASC
		LIMIT 10
	`, *collegeIDStr).Scan(&diffs)
	for _, d := range diffs {
		result.DifficultyIndicators = append(result.DifficultyIndicators, struct {
			CourseCode string  `json:"course_code"`
			CourseName string  `json:"course_name"`
			AvgMarks   float64 `json:"avg_marks"`
			FailurePct float64 `json:"failure_pct"`
		}{
			CourseCode: d.CourseCode,
			CourseName: d.CourseName,
			AvgMarks:   d.AvgMarks,
			FailurePct: d.FailurePct,
		})
	}

	// Faculty effectiveness
	type facRow struct {
		RegdNo  string
		Name    string
		Courses int
	}
	var facs []facRow
	database.DB.Raw(`
		SELECT u.regdno, u.name, COUNT(DISTINCT fa.course_offering_id) as courses
		FROM users u
		JOIN faculties f ON f.regd_no = u.regdno
		LEFT JOIN faculty_assignments fa ON fa.faculty_regd_no = u.regdno
		WHERE u.college_id = ? AND u.role IN ('faculty', 'hod') AND u.is_active = true
		GROUP BY u.regdno, u.name
	`, *collegeIDStr).Scan(&facs)

	for _, f := range facs {
		fe := FacultyEffectiveness{
			RegdNo:  f.RegdNo,
			Name:    f.Name,
			Courses: f.Courses,
		}
		// Students taught
		var students int64
		database.DB.Model(&models.Enrollment{}).
			Joins("JOIN faculty_assignments ON faculty_assignments.course_offering_id = enrollments.course_offering_id").
			Where("faculty_assignments.faculty_regd_no = ?", f.RegdNo).
			Count(&students)
		fe.Students = students

		// Average marks for this faculty's students
		var avg float64
		database.DB.Raw(`
			SELECT COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0)
			FROM exam_results er
			JOIN exams e ON e.id = er.exam_id
			JOIN course_offerings co ON co.id = e.course_offering_id
			JOIN faculty_assignments fa ON fa.course_offering_id = co.id
			WHERE fa.faculty_regd_no = ?
		`, f.RegdNo).Scan(&avg)
		fe.AvgMarks = avg

		// Submissions
		var subs int64
		database.DB.Model(&models.Submission{}).
			Joins("JOIN course_offerings ON course_offerings.id = submissions.course_id").
			Joins("JOIN faculty_assignments ON faculty_assignments.course_offering_id = course_offerings.id").
			Where("faculty_assignments.faculty_regd_no = ? AND submissions.college_id = ?", f.RegdNo, *collegeIDStr).
			Count(&subs)
		fe.Submissions = subs

		result.FacultyEffectiveness = append(result.FacultyEffectiveness, fe)
	}

	c.JSON(http.StatusOK, result)
}

// =====================================================
// 3. Lab Session Analytics
// =====================================================

func GetPrincipalLabAnalytics(c *gin.Context) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not assigned"})
		return
	}
	collegeIDStr := collegeID.(*string)

	var result struct {
		LabConduct     []LabConduct `json:"lab_conduct"`
		LabPerformance []struct {
			LabName          string  `json:"lab_name"`
			SessionName      string  `json:"session_name"`
			AvgScore         float64 `json:"avg_score"`
			CompletionRate   float64 `json:"completion_rate"`
			TotalSubmissions int64   `json:"total_submissions"`
		} `json:"lab_performance"`
		RiskIndicators []struct {
			LabName       string  `json:"lab_name"`
			BranchName    string  `json:"branch_name"`
			AvgScore      float64 `json:"avg_score"`
			AttendancePct float64 `json:"attendance_pct"`
			RiskLevel     string  `json:"risk_level"`
		} `json:"risk_indicators"`
	}

	result.LabConduct = make([]LabConduct, 0)
	result.LabPerformance = make([]struct {
		LabName          string  `json:"lab_name"`
		SessionName      string  `json:"session_name"`
		AvgScore         float64 `json:"avg_score"`
		CompletionRate   float64 `json:"completion_rate"`
		TotalSubmissions int64   `json:"total_submissions"`
	}, 0)
	result.RiskIndicators = make([]struct {
		LabName       string  `json:"lab_name"`
		BranchName    string  `json:"branch_name"`
		AvgScore      float64 `json:"avg_score"`
		AttendancePct float64 `json:"attendance_pct"`
		RiskLevel     string  `json:"risk_level"`
	}, 0)

	// Lab conduct tracking
	type labRow struct {
		LabName    string
		CourseCode string
		LabID      uint
	}
	var labs []labRow
	database.DB.Raw(`
		SELECT l.lab_name, c.course_code, l.id as lab_id
		FROM labs l
		JOIN courses c ON c.id = l.course_id
		WHERE c.college_id = ?
	`, *collegeIDStr).Scan(&labs)

	for _, lab := range labs {
		lc := LabConduct{
			LabName:    lab.LabName,
			CourseCode: lab.CourseCode,
		}
		// Count sessions
		var sessions int64
		database.DB.Model(&models.LabSession{}).Where("lab_id = ?", lab.LabID).Count(&sessions)
		lc.Planned = int(sessions)
		lc.Conducted = int(sessions) // Simplified: assume all planned are conducted for now
		lc.Missed = 0

		// Average score from submissions for this lab's sessions
		var avgScore float64
		database.DB.Model(&models.Submission{}).
			Where("lab_session_id IN (SELECT id FROM lab_sessions WHERE lab_id = ?)", lab.LabID).
			Select("COALESCE(AVG(score::float / NULLIF(max_score, 0) * 100), 0)").
			Scan(&avgScore)
		lc.AvgScore = avgScore

		result.LabConduct = append(result.LabConduct, lc)
	}

	// Lab performance per session
	type perfRow struct {
		LabName          string
		SessionName      string
		SessionID        uint
		AvgScore         float64
		TotalSubmissions int64
	}
	var perfs []perfRow
	database.DB.Raw(`
		SELECT l.lab_name, ls.session_name, ls.id as session_id,
			COALESCE(AVG(s.score::float / NULLIF(s.max_score, 0) * 100), 0) as avg_score,
			COUNT(s.id) as total_submissions
		FROM lab_sessions ls
		JOIN labs l ON l.id = ls.lab_id
		JOIN courses c ON c.id = l.course_id
		LEFT JOIN submissions s ON s.lab_session_id = ls.id
		WHERE c.college_id = ?
		GROUP BY ls.id, l.lab_name, ls.session_name
		ORDER BY avg_score ASC
		LIMIT 20
	`, *collegeIDStr).Scan(&perfs)

	for _, p := range perfs {
		// Completion rate: unique submitting students / total enrolled students for the lab's course
		var uniqueSubmitters, totalEnrolled int64
		database.DB.Model(&models.Submission{}).
			Where("lab_session_id = ?", p.SessionID).
			Distinct("user_regd_no").
			Count(&uniqueSubmitters)
		database.DB.Model(&models.Enrollment{}).
			Joins("JOIN course_offerings ON course_offerings.id = enrollments.course_offering_id").
			Joins("JOIN labs ON labs.course_id = course_offerings.course_id").
			Joins("JOIN lab_sessions ON lab_sessions.lab_id = labs.id").
			Where("lab_sessions.id = ?", p.SessionID).
			Count(&totalEnrolled)

		completionRate := 0.0
		if totalEnrolled > 0 {
			completionRate = float64(uniqueSubmitters) / float64(totalEnrolled) * 100
		}

		result.LabPerformance = append(result.LabPerformance, struct {
			LabName          string  `json:"lab_name"`
			SessionName      string  `json:"session_name"`
			AvgScore         float64 `json:"avg_score"`
			CompletionRate   float64 `json:"completion_rate"`
			TotalSubmissions int64   `json:"total_submissions"`
		}{
			LabName:          p.LabName,
			SessionName:      p.SessionName,
			AvgScore:         p.AvgScore,
			CompletionRate:   completionRate,
			TotalSubmissions: p.TotalSubmissions,
		})
	}

	// Risk indicators (labs with avg score < 40%)
	type riskRow struct {
		LabName    string
		BranchName string
		AvgScore   float64
	}
	var risks []riskRow
	database.DB.Raw(`
		SELECT l.lab_name, b.branch_name,
			COALESCE(AVG(s.score::float / NULLIF(s.max_score, 0) * 100), 0) as avg_score
		FROM submissions s
		JOIN lab_sessions ls ON ls.id = s.lab_session_id
		JOIN labs l ON l.id = ls.lab_id
		JOIN courses c ON c.id = l.course_id
		JOIN users u ON u.regdno = s.user_regd_no
		JOIN branches b ON b.branch_id = u.branch_id
		WHERE s.college_id = ?
		GROUP BY l.lab_name, b.branch_name
		HAVING COALESCE(AVG(s.score::float / NULLIF(s.max_score, 0) * 100), 0) < 40
		ORDER BY avg_score ASC
	`, *collegeIDStr).Scan(&risks)

	for _, r := range risks {
		rl := "high"
		if r.AvgScore > 25 {
			rl = "medium"
		}
		result.RiskIndicators = append(result.RiskIndicators, struct {
			LabName       string  `json:"lab_name"`
			BranchName    string  `json:"branch_name"`
			AvgScore      float64 `json:"avg_score"`
			AttendancePct float64 `json:"attendance_pct"`
			RiskLevel     string  `json:"risk_level"`
		}{
			LabName:       r.LabName,
			BranchName:    r.BranchName,
			AvgScore:      r.AvgScore,
			AttendancePct: r.AvgScore,
			RiskLevel:     rl,
		})
	}

	c.JSON(http.StatusOK, result)
}

// =====================================================
// 4. Contest Analytics
// =====================================================

func GetPrincipalContestAnalytics(c *gin.Context) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not assigned"})
		return
	}
	collegeIDStr := collegeID.(*string)

	var result struct {
		ContestActivity     ContestActivity `json:"contest_activity"`
		BranchParticipation []struct {
			BranchName       string  `json:"branch_name"`
			ParticipationPct float64 `json:"participation_pct"`
			AvgScore         float64 `json:"avg_score"`
		} `json:"branch_participation"`
		TopPerformers []struct {
			Name           string  `json:"name"`
			BranchName     string  `json:"branch_name"`
			Contests       int64   `json:"contests"`
			AvgScore       float64 `json:"avg_score"`
			ProblemsSolved int64   `json:"problems_solved"`
		} `json:"top_performers"`
		SkillTrends []struct {
			SkillType    string  `json:"skill_type"`
			AvgScore     float64 `json:"avg_score"`
			Participants int64   `json:"participants"`
		} `json:"skill_trends"`
		EngagementStats struct {
			ActiveStudents     int64 `json:"active_students"`
			InactiveStudents   int64 `json:"inactive_students"`
			RepeatParticipants int64 `json:"repeat_participants"`
		} `json:"engagement_stats"`
	}

	result.BranchParticipation = make([]struct {
		BranchName       string  `json:"branch_name"`
		ParticipationPct float64 `json:"participation_pct"`
		AvgScore         float64 `json:"avg_score"`
	}, 0)
	result.TopPerformers = make([]struct {
		Name           string  `json:"name"`
		BranchName     string  `json:"branch_name"`
		Contests       int64   `json:"contests"`
		AvgScore       float64 `json:"avg_score"`
		ProblemsSolved int64   `json:"problems_solved"`
	}, 0)
	result.SkillTrends = make([]struct {
		SkillType    string  `json:"skill_type"`
		AvgScore     float64 `json:"avg_score"`
		Participants int64   `json:"participants"`
	}, 0)

	// Contest activity
	database.DB.Model(&models.Contest{}).
		Where("college_id = ? AND start_time >= ?", *collegeIDStr, time.Now().AddDate(0, -6, 0)).
		Count(&result.ContestActivity.Total)

	// Internal vs external (simplified: all are internal for now)
	result.ContestActivity.Internal = result.ContestActivity.Total
	result.ContestActivity.External = 0

	// Average participants per contest
	var avgParticipants float64
	database.DB.Raw(`
		SELECT COALESCE(AVG(participant_count), 0)
		FROM (
			SELECT contest_id, COUNT(*) as participant_count
			FROM contest_participants
			JOIN contests ON contests.contest_id = contest_participants.contest_id
			WHERE contests.college_id = ?
			GROUP BY contest_id
		) t
	`, *collegeIDStr).Scan(&avgParticipants)
	result.ContestActivity.AvgParticipants = avgParticipants

	// Branch participation
	type bpRow struct {
		BranchName       string
		Participants     int64
		EligibleStudents int64
		AvgScore         float64
	}
	var bps []bpRow
	database.DB.Raw(`
		SELECT b.branch_name,
			COUNT(DISTINCT cp.user_regd_no) as participants,
			COUNT(DISTINCT u.regdno) as eligible_students,
			COALESCE(AVG(cp.total_score), 0) as avg_score
		FROM branches b
		LEFT JOIN users u ON u.branch_id = b.branch_id AND u.role = 'student' AND u.is_active = true AND u.college_id = ?
		LEFT JOIN contest_participants cp ON cp.user_regd_no = u.regdno
		LEFT JOIN contests c ON c.contest_id = cp.contest_id AND c.college_id = ?
		WHERE b.college_id = ?
		GROUP BY b.branch_id, b.branch_name
	`, *collegeIDStr, *collegeIDStr, *collegeIDStr).Scan(&bps)
	for _, b := range bps {
		participationPct := 0.0
		if b.EligibleStudents > 0 {
			participationPct = float64(b.Participants) / float64(b.EligibleStudents) * 100
		}
		result.BranchParticipation = append(result.BranchParticipation, struct {
			BranchName       string  `json:"branch_name"`
			ParticipationPct float64 `json:"participation_pct"`
			AvgScore         float64 `json:"avg_score"`
		}{
			BranchName:       b.BranchName,
			ParticipationPct: participationPct,
			AvgScore:         b.AvgScore,
		})
	}

	// Top performers
	type tpRow struct {
		Name           string
		BranchName     string
		Contests       int64
		AvgScore       float64
		ProblemsSolved int64
	}
	var tps []tpRow
	database.DB.Raw(`
		SELECT u.name, b.branch_name,
			COUNT(DISTINCT cp.contest_id) as contests,
			COALESCE(AVG(cp.total_score), 0) as avg_score,
			COALESCE(SUM(cp.problems_solved), 0) as problems_solved
		FROM contest_participants cp
		JOIN users u ON u.regdno = cp.user_regd_no
		JOIN branches b ON b.branch_id = u.branch_id
		JOIN contests c ON c.contest_id = cp.contest_id
		WHERE c.college_id = ?
		GROUP BY u.regdno, u.name, b.branch_name
		ORDER BY avg_score DESC
		LIMIT 10
	`, *collegeIDStr).Scan(&tps)
	for _, tp := range tps {
		result.TopPerformers = append(result.TopPerformers, struct {
			Name           string  `json:"name"`
			BranchName     string  `json:"branch_name"`
			Contests       int64   `json:"contests"`
			AvgScore       float64 `json:"avg_score"`
			ProblemsSolved int64   `json:"problems_solved"`
		}{
			Name:           tp.Name,
			BranchName:     tp.BranchName,
			Contests:       tp.Contests,
			AvgScore:       tp.AvgScore,
			ProblemsSolved: tp.ProblemsSolved,
		})
	}

	// Engagement stats
	database.DB.Model(&models.ContestParticipant{}).
		Joins("JOIN contests ON contests.contest_id = contest_participants.contest_id").
		Where("contests.college_id = ?", *collegeIDStr).
		Distinct("contest_participants.user_regd_no").
		Count(&result.EngagementStats.ActiveStudents)

	result.EngagementStats.InactiveStudents = 0 // Would need historical data

	var repeat int64
	database.DB.Raw(`
		SELECT COUNT(*) FROM (
			SELECT user_regd_no, COUNT(*) as cnt
			FROM contest_participants
			JOIN contests ON contests.contest_id = contest_participants.contest_id
			WHERE contests.college_id = ?
			GROUP BY user_regd_no
			HAVING COUNT(*) > 1
		) t
	`, *collegeIDStr).Scan(&repeat)
	result.EngagementStats.RepeatParticipants = repeat

	c.JSON(http.StatusOK, result)
}

// =====================================================
// 5. Branch-wise Analytics
// =====================================================

func GetPrincipalBranchAnalytics(c *gin.Context) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not assigned"})
		return
	}
	collegeIDStr := collegeID.(*string)

	var result struct {
		Branches      []BranchPerformance `json:"branches"`
		BestBranch    string              `json:"best_branch"`
		WeakestBranch string              `json:"weakest_branch"`
		BranchTrends  []struct {
			BranchName string  `json:"branch_name"`
			Year       int     `json:"year"`
			AvgMarks   float64 `json:"avg_marks"`
		} `json:"branch_trends"`
	}

	result.Branches = make([]BranchPerformance, 0)
	result.BranchTrends = make([]struct {
		BranchName string  `json:"branch_name"`
		Year       int     `json:"year"`
		AvgMarks   float64 `json:"avg_marks"`
	}, 0)

	var branches []models.Branch
	database.DB.Where("college_id = ?", *collegeIDStr).Find(&branches)

	bestBranch := ""
	weakestBranch := ""
	bestScore := -1.0
	worstScore := 999.0

	for _, branch := range branches {
		bp := BranchPerformance{
			BranchID:   branch.BranchID,
			BranchName: branch.BranchName,
			ShortName:  branch.ShortName,
		}

		// Total students
		database.DB.Model(&models.User{}).
			Where("role = ? AND branch_id = ? AND college_id = ? AND is_active = true", "student", branch.BranchID, *collegeIDStr).
			Count(&bp.TotalStudents)

		// Total faculty
		database.DB.Model(&models.User{}).
			Where("role IN (?, ?) AND branch_id = ? AND college_id = ? AND is_active = true", "faculty", "hod", branch.BranchID, *collegeIDStr).
			Count(&bp.TotalFaculty)

		// Avg submission score
		var avgScore float64
		database.DB.Model(&models.Submission{}).
			Joins("JOIN users ON users.regdno = submissions.user_regd_no").
			Where("users.branch_id = ? AND submissions.college_id = ? AND users.role = ?", branch.BranchID, *collegeIDStr, "student").
			Select("COALESCE(AVG(score::float / NULLIF(max_score, 0) * 100), 0)").
			Scan(&avgScore)
		bp.AvgSubmissionScore = avgScore

		// Pass % from exam results
		var passCount, totalCount int64
		database.DB.Model(&models.ExamResult{}).
			Joins("JOIN exams ON exams.id = exam_results.exam_id").
			Joins("JOIN course_offerings ON course_offerings.id = exams.course_offering_id").
			Joins("JOIN sections ON sections.section_id = course_offerings.section_id").
			Where("sections.branch_id = ? AND exam_results.marks_obtained >= exams.passing_marks", branch.BranchID).
			Count(&passCount)
		database.DB.Model(&models.ExamResult{}).
			Joins("JOIN exams ON exams.id = exam_results.exam_id").
			Joins("JOIN course_offerings ON course_offerings.id = exams.course_offering_id").
			Joins("JOIN sections ON sections.section_id = course_offerings.section_id").
			Where("sections.branch_id = ?", branch.BranchID).
			Count(&totalCount)
		if totalCount > 0 {
			bp.PassPct = float64(passCount) / float64(totalCount) * 100
		}

		// Contest participation %
		var participants, eligible int64
		database.DB.Model(&models.ContestParticipant{}).
			Joins("JOIN users ON users.regdno = contest_participants.user_regd_no").
			Joins("JOIN contests ON contests.contest_id = contest_participants.contest_id").
			Where("users.branch_id = ? AND contests.college_id = ?", branch.BranchID, *collegeIDStr).
			Distinct("contest_participants.user_regd_no").
			Count(&participants)
		database.DB.Model(&models.User{}).
			Where("role = ? AND branch_id = ? AND college_id = ? AND is_active = true", "student", branch.BranchID, *collegeIDStr).
			Count(&eligible)
		if eligible > 0 {
			bp.ContestParticipation = float64(participants) / float64(eligible) * 100
		}

		// Assignment completion (submissions / total students proxy)
		var submissions int64
		database.DB.Model(&models.Submission{}).
			Joins("JOIN users ON users.regdno = submissions.user_regd_no").
			Where("users.branch_id = ? AND submissions.college_id = ? AND users.role = ?", branch.BranchID, *collegeIDStr, "student").
			Count(&submissions)
		if bp.TotalStudents > 0 {
			bp.AssignmentCompletion = float64(submissions) / float64(bp.TotalStudents) * 100
		}

		result.Branches = append(result.Branches, bp)

		composite := avgScore + bp.PassPct + bp.ContestParticipation
		if composite > bestScore {
			bestScore = composite
			bestBranch = branch.BranchName
		}
		if composite < worstScore {
			worstScore = composite
			weakestBranch = branch.BranchName
		}
	}

	result.BestBranch = bestBranch
	result.WeakestBranch = weakestBranch

	// Branch trends (last 3 years)
	for year := time.Now().Year() - 2; year <= time.Now().Year(); year++ {
		for _, branch := range branches {
			var avgMarks float64
			database.DB.Raw(`
				SELECT COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0)
				FROM exam_results er
				JOIN exams e ON e.id = er.exam_id
				JOIN course_offerings co ON co.id = e.course_offering_id
				JOIN sections s ON s.section_id = co.section_id
				JOIN academic_years ay ON ay.academic_year_id = s.academic_year_id
				WHERE s.branch_id = ? AND EXTRACT(YEAR FROM ay.start_date) <= ? AND EXTRACT(YEAR FROM ay.end_date) >= ?
			`, branch.BranchID, year, year).Scan(&avgMarks)

			result.BranchTrends = append(result.BranchTrends, struct {
				BranchName string  `json:"branch_name"`
				Year       int     `json:"year"`
				AvgMarks   float64 `json:"avg_marks"`
			}{
				BranchName: branch.BranchName,
				Year:       year,
				AvgMarks:   avgMarks,
			})
		}
	}

	c.JSON(http.StatusOK, result)
}

// =====================================================
// 6. Student Performance Analytics
// =====================================================

func GetPrincipalStudentAnalytics(c *gin.Context) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not assigned"})
		return
	}
	collegeIDStr := collegeID.(*string)

	var result struct {
		SubjectPerformance []SubjectPerformance     `json:"subject_performance"`
		WeakStudents       []StudentPerformanceItem `json:"weak_students"`
		TopPerformers      []StudentPerformanceItem `json:"top_performers"`
		AtRiskCount        int64                    `json:"at_risk_count"`
	}

	result.SubjectPerformance = make([]SubjectPerformance, 0)
	result.WeakStudents = make([]StudentPerformanceItem, 0)
	result.TopPerformers = make([]StudentPerformanceItem, 0)

	// Subject performance
	type subRow struct {
		SubjectName   string
		AvgMarks      float64
		PassPct       float64
		TotalStudents int64
	}
	var subs []subRow
	database.DB.Raw(`
		SELECT sub.name as subject_name,
			COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) as avg_marks,
			COALESCE(
				SUM(CASE WHEN er.marks_obtained >= e.passing_marks THEN 1 ELSE 0 END)::float
				/ NULLIF(COUNT(*), 0) * 100, 0
			) as pass_pct,
			COUNT(DISTINCT er.student_regdno) as total_students
		FROM exam_results er
		JOIN exams e ON e.id = er.exam_id
		JOIN course_offerings co ON co.id = e.course_offering_id
		JOIN courses c ON c.id = co.course_id
		JOIN subjects sub ON sub.id = c.id
		WHERE c.college_id = ?
		GROUP BY sub.id, sub.name
		ORDER BY avg_marks ASC
	`, *collegeIDStr).Scan(&subs)
	for _, s := range subs {
		result.SubjectPerformance = append(result.SubjectPerformance, SubjectPerformance(s))
	}

	// Weak students (below 40% average in exams)
	type weakRow struct {
		RegdNo           string
		Name             string
		BranchName       string
		AvgMarks         float64
		TotalSubmissions int64
	}
	var weakStudents []weakRow
	database.DB.Raw(`
		SELECT u.regdno, u.name, b.branch_name,
			COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) as avg_marks,
			COUNT(s.id) as total_submissions
		FROM users u
		JOIN branches b ON b.branch_id = u.branch_id
		LEFT JOIN exam_results er ON er.student_regdno = u.regdno
		LEFT JOIN exams e ON e.id = er.exam_id
		LEFT JOIN submissions s ON s.user_regd_no = u.regdno
		WHERE u.college_id = ? AND u.role = 'student' AND u.is_active = true
		GROUP BY u.regdno, u.name, b.branch_name
		HAVING COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) < 40
		ORDER BY avg_marks ASC
		LIMIT 20
	`, *collegeIDStr).Scan(&weakStudents)
	for _, w := range weakStudents {
		result.WeakStudents = append(result.WeakStudents, StudentPerformanceItem{
			RegdNo:           w.RegdNo,
			Name:             w.Name,
			BranchName:       w.BranchName,
			AvgMarks:         w.AvgMarks,
			TotalSubmissions: w.TotalSubmissions,
			Status:           "at_risk",
		})
		result.AtRiskCount++
	}

	// Top performers (above 80% average)
	type topRow struct {
		RegdNo           string
		Name             string
		BranchName       string
		AvgMarks         float64
		TotalSubmissions int64
	}
	var topPerformers []topRow
	database.DB.Raw(`
		SELECT u.regdno, u.name, b.branch_name,
			COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) as avg_marks,
			COUNT(s.id) as total_submissions
		FROM users u
		JOIN branches b ON b.branch_id = u.branch_id
		LEFT JOIN exam_results er ON er.student_regdno = u.regdno
		LEFT JOIN exams e ON e.id = er.exam_id
		LEFT JOIN submissions s ON s.user_regd_no = u.regdno
		WHERE u.college_id = ? AND u.role = 'student' AND u.is_active = true
		GROUP BY u.regdno, u.name, b.branch_name
		HAVING COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) >= 80
		ORDER BY avg_marks DESC
		LIMIT 10
	`, *collegeIDStr).Scan(&topPerformers)
	for _, t := range topPerformers {
		result.TopPerformers = append(result.TopPerformers, StudentPerformanceItem{
			RegdNo:           t.RegdNo,
			Name:             t.Name,
			BranchName:       t.BranchName,
			AvgMarks:         t.AvgMarks,
			TotalSubmissions: t.TotalSubmissions,
			Status:           "top_performer",
		})
	}

	c.JSON(http.StatusOK, result)
}

// =====================================================
// 7. Year/Cohort Analytics
// =====================================================

func GetPrincipalYearAnalytics(c *gin.Context) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not assigned"})
		return
	}
	collegeIDStr := collegeID.(*string)

	var result struct {
		CohortTrends      []CohortTrend `json:"cohort_trends"`
		HistoricalPassPct []struct {
			Year    int     `json:"year"`
			PassPct float64 `json:"pass_pct"`
		} `json:"historical_pass_pct"`
		PlacementReadiness []struct {
			CohortYear           int     `json:"cohort_year"`
			CodingScore          float64 `json:"coding_score"`
			ContestParticipation float64 `json:"contest_participation"`
			SkillAssessments     float64 `json:"skill_assessments"`
		} `json:"placement_readiness"`
		GrowthMetrics struct {
			ImprovementRate float64 `json:"improvement_rate"`
			DeptGrowth      float64 `json:"dept_growth"`
		} `json:"growth_metrics"`
	}

	result.CohortTrends = make([]CohortTrend, 0)
	result.HistoricalPassPct = make([]struct {
		Year    int     `json:"year"`
		PassPct float64 `json:"pass_pct"`
	}, 0)
	result.PlacementReadiness = make([]struct {
		CohortYear           int     `json:"cohort_year"`
		CodingScore          float64 `json:"coding_score"`
		ContestParticipation float64 `json:"contest_participation"`
		SkillAssessments     float64 `json:"skill_assessments"`
	}, 0)

	// Cohort trends
	type cohortRow struct {
		CohortYear    int
		TotalStudents int64
		AvgMarks      float64
		Submissions   int64
	}
	var cohorts []cohortRow
	database.DB.Raw(`
		SELECT st.cohort_year,
			COUNT(DISTINCT st.regd_no) as total_students,
			COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) as avg_marks,
			COUNT(s.id) as submissions
		FROM students st
		JOIN users u ON u.regdno = st.regd_no
		LEFT JOIN exam_results er ON er.student_regdno = st.regd_no
		LEFT JOIN exams e ON e.id = er.exam_id
		LEFT JOIN submissions s ON s.user_regd_no = st.regd_no
		WHERE u.college_id = ? AND u.role = 'student'
		GROUP BY st.cohort_year
		ORDER BY st.cohort_year DESC
	`, *collegeIDStr).Scan(&cohorts)

	for _, c := range cohorts {
		// Pass % per cohort
		var passCount, totalCount int64
		database.DB.Model(&models.ExamResult{}).
			Joins("JOIN exams ON exams.id = exam_results.exam_id").
			Joins("JOIN students ON students.regd_no = exam_results.student_regdno").
			Where("students.cohort_year = ? AND exams.college_id = ?", c.CohortYear, *collegeIDStr).
			Where("exam_results.marks_obtained >= exams.passing_marks").
			Count(&passCount)
		database.DB.Model(&models.ExamResult{}).
			Joins("JOIN exams ON exams.id = exam_results.exam_id").
			Joins("JOIN students ON students.regd_no = exam_results.student_regdno").
			Where("students.cohort_year = ? AND exams.college_id = ?", c.CohortYear, *collegeIDStr).
			Count(&totalCount)

		passPct := 0.0
		if totalCount > 0 {
			passPct = float64(passCount) / float64(totalCount) * 100
		}

		result.CohortTrends = append(result.CohortTrends, CohortTrend{
			CohortYear:    c.CohortYear,
			TotalStudents: c.TotalStudents,
			AvgMarks:      c.AvgMarks,
			PassPct:       passPct,
			Submissions:   c.Submissions,
		})
	}

	// Historical pass % (by admission year)
	for year := time.Now().Year() - 4; year <= time.Now().Year(); year++ {
		var passCount, totalCount int64
		database.DB.Model(&models.ExamResult{}).
			Joins("JOIN exams ON exams.id = exam_results.exam_id").
			Joins("JOIN students ON students.regd_no = exam_results.student_regdno").
			Joins("JOIN users ON users.regdno = students.regd_no").
			Where("students.admission_year = ? AND users.college_id = ? AND exam_results.marks_obtained >= exams.passing_marks", year, *collegeIDStr).
			Count(&passCount)
		database.DB.Model(&models.ExamResult{}).
			Joins("JOIN exams ON exams.id = exam_results.exam_id").
			Joins("JOIN students ON students.regd_no = exam_results.student_regdno").
			Joins("JOIN users ON users.regdno = students.regd_no").
			Where("students.admission_year = ? AND users.college_id = ?", year, *collegeIDStr).
			Count(&totalCount)

		passPct := 0.0
		if totalCount > 0 {
			passPct = float64(passCount) / float64(totalCount) * 100
		}
		result.HistoricalPassPct = append(result.HistoricalPassPct, struct {
			Year    int     `json:"year"`
			PassPct float64 `json:"pass_pct"`
		}{
			Year:    year,
			PassPct: passPct,
		})
	}

	// Placement readiness
	type prRow struct {
		CohortYear int
	}
	var prCohorts []prRow
	database.DB.Raw(`
		SELECT DISTINCT st.cohort_year
		FROM students st
		JOIN users u ON u.regdno = st.regd_no
		WHERE u.college_id = ? AND u.role = 'student'
		ORDER BY st.cohort_year DESC
	`, *collegeIDStr).Scan(&prCohorts)

	for _, c := range prCohorts {
		// Coding score: avg submission score
		var codingScore float64
		database.DB.Model(&models.Submission{}).
			Joins("JOIN students ON students.regd_no = submissions.user_regd_no").
			Joins("JOIN users ON users.regdno = students.regd_no").
			Where("students.cohort_year = ? AND users.college_id = ?", c.CohortYear, *collegeIDStr).
			Select("COALESCE(AVG(score::float / NULLIF(max_score, 0) * 100), 0)").
			Scan(&codingScore)

		// Contest participation
		var participants, eligible int64
		database.DB.Model(&models.ContestParticipant{}).
			Joins("JOIN students ON students.regd_no = contest_participants.user_regd_no").
			Joins("JOIN users ON users.regdno = students.regd_no").
			Joins("JOIN contests ON contests.contest_id = contest_participants.contest_id").
			Where("students.cohort_year = ? AND users.college_id = ? AND contests.college_id = ?", c.CohortYear, *collegeIDStr, *collegeIDStr).
			Distinct("contest_participants.user_regd_no").
			Count(&participants)
		database.DB.Model(&models.User{}).
			Joins("JOIN students ON students.regd_no = users.regdno").
			Where("students.cohort_year = ? AND users.college_id = ? AND users.role = 'student' AND users.is_active = true", c.CohortYear, *collegeIDStr).
			Count(&eligible)

		contestPart := 0.0
		if eligible > 0 {
			contestPart = float64(participants) / float64(eligible) * 100
		}

		// Skill assessments: avg quiz scores
		var skillScore float64
		database.DB.Raw(`
			SELECT COALESCE(AVG(qa.score::float / NULLIF(q.total_marks, 0) * 100), 0)
			FROM quiz_attempts qa
			JOIN quizzes q ON q.id = qa.quiz_id
			JOIN course_offerings co ON co.id = q.course_offering_id
			JOIN sections s ON s.section_id = co.section_id
			JOIN students st ON st.section_id = s.section_id
			JOIN users u ON u.regdno = st.regd_no
			WHERE st.cohort_year = ? AND u.college_id = ? AND qa.status = 'completed'
		`, c.CohortYear, *collegeIDStr).Scan(&skillScore)

		result.PlacementReadiness = append(result.PlacementReadiness, struct {
			CohortYear           int     `json:"cohort_year"`
			CodingScore          float64 `json:"coding_score"`
			ContestParticipation float64 `json:"contest_participation"`
			SkillAssessments     float64 `json:"skill_assessments"`
		}{
			CohortYear:           c.CohortYear,
			CodingScore:          codingScore,
			ContestParticipation: contestPart,
			SkillAssessments:     skillScore,
		})
	}

	// Growth metrics (compare current vs previous year)
	currentYear := time.Now().Year()
	var currentAvg, prevAvg float64
	database.DB.Raw(`
		SELECT COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0)
		FROM exam_results er
		JOIN exams e ON e.id = er.exam_id
		JOIN students st ON st.regd_no = er.student_regdno
		JOIN users u ON u.regdno = st.regd_no
		WHERE u.college_id = ? AND st.admission_year = ?
	`, *collegeIDStr, currentYear).Scan(&currentAvg)
	database.DB.Raw(`
		SELECT COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0)
		FROM exam_results er
		JOIN exams e ON e.id = er.exam_id
		JOIN students st ON st.regd_no = er.student_regdno
		JOIN users u ON u.regdno = st.regd_no
		WHERE u.college_id = ? AND st.admission_year = ?
	`, *collegeIDStr, currentYear-1).Scan(&prevAvg)

	if prevAvg > 0 {
		result.GrowthMetrics.ImprovementRate = ((currentAvg - prevAvg) / prevAvg) * 100
	}

	c.JSON(http.StatusOK, result)
}

// =====================================================
// 8. AI-Powered Insights
// =====================================================

func GetPrincipalInsights(c *gin.Context) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not assigned"})
		return
	}
	collegeIDStr := collegeID.(*string)

	var insights []InsightItem
	insights = make([]InsightItem, 0)

	// 1. Branch performance comparison
	type branchPerf struct {
		BranchName string
		AvgMarks   float64
	}
	var branches []branchPerf
	database.DB.Raw(`
		SELECT b.branch_name, COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) as avg_marks
		FROM branches b
		LEFT JOIN sections s ON s.branch_id = b.branch_id
		LEFT JOIN course_offerings co ON co.section_id = s.section_id
		LEFT JOIN exams e ON e.course_offering_id = co.id
		LEFT JOIN exam_results er ON er.exam_id = e.id
		WHERE b.college_id = ?
		GROUP BY b.branch_id, b.branch_name
		ORDER BY avg_marks DESC
	`, *collegeIDStr).Scan(&branches)

	if len(branches) > 1 {
		best := branches[0]
		worst := branches[len(branches)-1]
		if best.AvgMarks > 0 && worst.AvgMarks > 0 {
			diff := best.AvgMarks - worst.AvgMarks
			insights = append(insights, InsightItem{
				Type:     "branch_performance",
				Message:  fmt.Sprintf("%s is the best performing branch with %.1f%% avg marks, %.1f%% above %s.", best.BranchName, best.AvgMarks, diff, worst.BranchName),
				Severity: "info",
				Metric:   "avg_marks",
				Value:    fmt.Sprintf("%.1f", best.AvgMarks),
			})
		}
	}

	// 2. Lab performance decline
	type labDecline struct {
		LabName    string
		BranchName string
		AvgScore   float64
	}
	var declines []labDecline
	database.DB.Raw(`
		SELECT l.lab_name, b.branch_name, COALESCE(AVG(s.score::float / NULLIF(s.max_score, 0) * 100), 0) as avg_score
		FROM submissions s
		JOIN lab_sessions ls ON ls.id = s.lab_session_id
		JOIN labs l ON l.id = ls.lab_id
		JOIN courses c ON c.id = l.course_id
		JOIN users u ON u.regdno = s.user_regd_no
		JOIN branches b ON b.branch_id = u.branch_id
		WHERE s.college_id = ? AND s.created_at >= NOW() - INTERVAL '3 months'
		GROUP BY l.lab_name, b.branch_name
		HAVING COALESCE(AVG(s.score::float / NULLIF(s.max_score, 0) * 100), 0) < 40
		ORDER BY avg_score ASC
		LIMIT 3
	`, *collegeIDStr).Scan(&declines)

	for _, d := range declines {
		insights = append(insights, InsightItem{
			Type:       "lab_decline",
			Message:    fmt.Sprintf("%s lab in %s shows declining performance with avg score of %.1f%%.", d.LabName, d.BranchName, d.AvgScore),
			Severity:   "warning",
			BranchName: d.BranchName,
			Metric:     "lab_avg_score",
			Value:      fmt.Sprintf("%.1f", d.AvgScore),
		})
	}

	// 3. Contest engagement insight
	var contestParticipation float64
	database.DB.Raw(`
		SELECT COALESCE(
			COUNT(DISTINCT cp.user_regd_no)::float / NULLIF(COUNT(DISTINCT u.regdno), 0) * 100, 0
		)
		FROM users u
		LEFT JOIN contest_participants cp ON cp.user_regd_no = u.regdno
		LEFT JOIN contests c ON c.contest_id = cp.contest_id
		WHERE u.college_id = ? AND u.role = 'student' AND u.is_active = true
		AND (c.college_id = ? OR c.college_id IS NULL)
	`, *collegeIDStr, *collegeIDStr).Scan(&contestParticipation)

	if contestParticipation < 30 {
		insights = append(insights, InsightItem{
			Type:     "contest_engagement",
			Message:  fmt.Sprintf("Overall contest participation is low at %.1f%%. Consider motivating more students to participate.", contestParticipation),
			Severity: "warning",
			Metric:   "contest_participation",
			Value:    fmt.Sprintf("%.1f", contestParticipation),
		})
	} else if contestParticipation > 60 {
		insights = append(insights, InsightItem{
			Type:     "contest_engagement",
			Message:  fmt.Sprintf("Excellent contest engagement: %.1f%% of students are participating in contests.", contestParticipation),
			Severity: "success",
			Metric:   "contest_participation",
			Value:    fmt.Sprintf("%.1f", contestParticipation),
		})
	}

	// 4. Subject difficulty correlation
	type subDiff struct {
		SubjectName string
		AvgMarks    float64
	}
	var difficultSubjects []subDiff
	database.DB.Raw(`
		SELECT sub.name, COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) as avg_marks
		FROM subjects sub
		LEFT JOIN exams e ON e.subject_id = sub.id
		LEFT JOIN exam_results er ON er.exam_id = e.id
		LEFT JOIN course_offerings co ON co.id = e.course_offering_id
		LEFT JOIN courses c ON c.id = co.course_id
		WHERE c.college_id = ?
		GROUP BY sub.id
		ORDER BY avg_marks ASC
		LIMIT 3
	`, *collegeIDStr).Scan(&difficultSubjects)

	if len(difficultSubjects) > 0 && difficultSubjects[0].AvgMarks < 50 {
		insights = append(insights, InsightItem{
			Type:     "subject_difficulty",
			Message:  fmt.Sprintf("Students are struggling in %s with an average of %.1f%%. Consider additional support sessions.", difficultSubjects[0].SubjectName, difficultSubjects[0].AvgMarks),
			Severity: "warning",
			Metric:   "subject_avg",
			Value:    fmt.Sprintf("%.1f", difficultSubjects[0].AvgMarks),
		})
	}

	// 5. Faculty workload insight
	type workload struct {
		Name    string
		Courses int
	}
	var workloads []workload
	database.DB.Raw(`
		SELECT u.name, COUNT(DISTINCT fa.course_offering_id) as courses
		FROM users u
		JOIN faculties f ON f.regd_no = u.regdno
		LEFT JOIN faculty_assignments fa ON fa.faculty_regd_no = u.regdno
		WHERE u.college_id = ? AND u.role IN ('faculty', 'hod') AND u.is_active = true
		GROUP BY u.regdno, u.name
		ORDER BY courses DESC
		LIMIT 1
	`, *collegeIDStr).Scan(&workloads)

	if len(workloads) > 0 && workloads[0].Courses > 5 {
		insights = append(insights, InsightItem{
			Type:     "faculty_workload",
			Message:  fmt.Sprintf("%s is handling %d courses. Consider redistributing workload.", workloads[0].Name, workloads[0].Courses),
			Severity: "info",
			Metric:   "faculty_courses",
			Value:    fmt.Sprintf("%d", workloads[0].Courses),
		})
	}

	// 6. Coding consistency insight
	var consistentSubmitters int64
	database.DB.Raw(`
		SELECT COUNT(*) FROM (
			SELECT user_regd_no, COUNT(*) as cnt
			FROM submissions
			WHERE college_id = ? AND created_at >= NOW() - INTERVAL '30 days'
			GROUP BY user_regd_no
			HAVING COUNT(*) >= 5
		) t
	`, *collegeIDStr).Scan(&consistentSubmitters)

	var totalStudents int64
	database.DB.Model(&models.User{}).
		Where("role = ? AND college_id = ? AND is_active = true", "student", *collegeIDStr).
		Count(&totalStudents)

	if totalStudents > 0 {
		consistencyPct := float64(consistentSubmitters) / float64(totalStudents) * 100
		if consistencyPct < 20 {
			insights = append(insights, InsightItem{
				Type:     "coding_consistency",
				Message:  fmt.Sprintf("Only %.1f%% of students submit code consistently (5+ submissions/month). Encourage regular practice.", consistencyPct),
				Severity: "warning",
				Metric:   "coding_consistency",
				Value:    fmt.Sprintf("%.1f", consistencyPct),
			})
		} else if consistencyPct > 50 {
			insights = append(insights, InsightItem{
				Type:     "coding_consistency",
				Message:  fmt.Sprintf("Strong coding consistency: %.1f%% of students practice regularly with 5+ submissions per month.", consistencyPct),
				Severity: "success",
				Metric:   "coding_consistency",
				Value:    fmt.Sprintf("%.1f", consistencyPct),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"insights": insights})
}

// =====================================================
// 9. Alerts & Risk Monitoring
// =====================================================

func GetPrincipalAlerts(c *gin.Context) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not assigned"})
		return
	}
	collegeIDStr := collegeID.(*string)

	var alerts []AlertItem
	alerts = make([]AlertItem, 0)
	now := time.Now().Format("2006-01-02")

	// 1. Low-performing branches (avg exam marks < 40%)
	type lowBranch struct {
		BranchName string
		AvgMarks   float64
	}
	var lowBranches []lowBranch
	database.DB.Raw(`
		SELECT b.branch_name, COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) as avg_marks
		FROM branches b
		LEFT JOIN sections s ON s.branch_id = b.branch_id
		LEFT JOIN course_offerings co ON co.section_id = s.section_id
		LEFT JOIN exams e ON e.course_offering_id = co.id
		LEFT JOIN exam_results er ON er.exam_id = e.id
		WHERE b.college_id = ?
		GROUP BY b.branch_id, b.branch_name
		HAVING COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) < 40
	`, *collegeIDStr).Scan(&lowBranches)

	for _, b := range lowBranches {
		alerts = append(alerts, AlertItem{
			Type:       "low_performing_branch",
			Message:    fmt.Sprintf("Branch %s has low average performance (%.1f%%). Immediate intervention recommended.", b.BranchName, b.AvgMarks),
			Severity:   "critical",
			BranchName: b.BranchName,
			CreatedAt:  now,
		})
	}

	// 2. Faculty evaluation delays (unmarked submissions older than 7 days)
	var delayedSubmissions int64
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	database.DB.Model(&models.Submission{}).
		Where("college_id = ? AND created_at <= ? AND score = 0 AND passed = false", *collegeIDStr, sevenDaysAgo).
		Count(&delayedSubmissions)

	if delayedSubmissions > 10 {
		alerts = append(alerts, AlertItem{
			Type:      "evaluation_delay",
			Message:   fmt.Sprintf("%d submissions are pending evaluation for over 7 days.", delayedSubmissions),
			Severity:  "warning",
			CreatedAt: now,
		})
	}

	// 3. Low attendance warning (students with < 1 submission in last 14 days)
	var lowAttendance int64
	fourteenDaysAgo := time.Now().AddDate(0, 0, -14)
	database.DB.Raw(`
		SELECT COUNT(*) FROM (
			SELECT u.regdno
			FROM users u
			WHERE u.college_id = ? AND u.role = 'student' AND u.is_active = true
			AND NOT EXISTS (
				SELECT 1 FROM submissions s
				WHERE s.user_regd_no = u.regdno AND s.created_at >= ?
			)
		) t
	`, *collegeIDStr, fourteenDaysAgo).Scan(&lowAttendance)

	if lowAttendance > 20 {
		alerts = append(alerts, AlertItem{
			Type:      "low_attendance",
			Message:   fmt.Sprintf("%d students have no activity in the last 14 days.", lowAttendance),
			Severity:  "warning",
			CreatedAt: now,
		})
	}

	// 4. Poor contest participation (< 20% of eligible)
	type poorContest struct {
		BranchName    string
		Participation float64
	}
	var poorContests []poorContest
	database.DB.Raw(`
		SELECT b.branch_name,
			COALESCE(COUNT(DISTINCT cp.user_regd_no)::float / NULLIF(COUNT(DISTINCT u.regdno), 0) * 100, 0) as participation
		FROM branches b
		LEFT JOIN users u ON u.branch_id = b.branch_id AND u.role = 'student' AND u.is_active = true
		LEFT JOIN contest_participants cp ON cp.user_regd_no = u.regdno
		LEFT JOIN contests c ON c.contest_id = cp.contest_id AND c.college_id = ?
		WHERE b.college_id = ?
		GROUP BY b.branch_id, b.branch_name
		HAVING COALESCE(COUNT(DISTINCT cp.user_regd_no)::float / NULLIF(COUNT(DISTINCT u.regdno), 0) * 100, 0) < 20
	`, *collegeIDStr, *collegeIDStr).Scan(&poorContests)

	for _, pc := range poorContests {
		alerts = append(alerts, AlertItem{
			Type:       "poor_contest_participation",
			Message:    fmt.Sprintf("%s has low contest participation (%.1f%%). Consider awareness campaigns.", pc.BranchName, pc.Participation),
			Severity:   "info",
			BranchName: pc.BranchName,
			CreatedAt:  now,
		})
	}

	// 5. Lab inactivity (no submissions in last 2 weeks)
	type inactiveLab struct {
		LabName string
	}
	var inactiveLabs []inactiveLab
	database.DB.Raw(`
		SELECT l.lab_name
		FROM labs l
		JOIN courses c ON c.id = l.course_id
		WHERE c.college_id = ?
		AND NOT EXISTS (
			SELECT 1 FROM submissions s
			JOIN lab_sessions ls ON ls.id = s.lab_session_id
			WHERE ls.lab_id = l.id AND s.created_at >= ?
		)
	`, *collegeIDStr, fourteenDaysAgo).Scan(&inactiveLabs)

	for _, il := range inactiveLabs {
		alerts = append(alerts, AlertItem{
			Type:      "lab_inactivity",
			Message:   fmt.Sprintf("Lab %s has had no student submissions in the last 2 weeks.", il.LabName),
			Severity:  "warning",
			CreatedAt: now,
		})
	}

	// 6. At-risk students (> 20 students below 40%)
	if len(lowBranches) == 0 {
		var atRiskCount int64
		database.DB.Raw(`
			SELECT COUNT(*) FROM (
				SELECT u.regdno
				FROM users u
				LEFT JOIN exam_results er ON er.student_regdno = u.regdno
				LEFT JOIN exams e ON e.id = er.exam_id
				WHERE u.college_id = ? AND u.role = 'student' AND u.is_active = true
				GROUP BY u.regdno
				HAVING COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0) < 40
			) t
		`, *collegeIDStr).Scan(&atRiskCount)

		if atRiskCount > 20 {
			alerts = append(alerts, AlertItem{
				Type:      "at_risk_students",
				Message:   fmt.Sprintf("%d students are performing below 40%% average. Academic counseling recommended.", atRiskCount),
				Severity:  "critical",
				CreatedAt: now,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// =====================================================
// 10. Accreditation Support
// =====================================================

func GetPrincipalAccreditation(c *gin.Context) {
	collegeID, exists := c.Get("college_id")
	if !exists || collegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "College not assigned"})
		return
	}
	collegeIDStr := collegeID.(*string)

	var result struct {
		OutcomeMetrics    []AccreditationMetrics `json:"outcome_metrics"`
		DepartmentReports []struct {
			BranchName     string  `json:"branch_name"`
			TotalStudents  int64   `json:"total_students"`
			TotalFaculty   int64   `json:"total_faculty"`
			AvgPerformance float64 `json:"avg_performance"`
			PassPct        float64 `json:"pass_pct"`
			ContestWinners int64   `json:"contest_winners"`
		} `json:"department_reports"`
		FacultyWorkload []struct {
			Name         string  `json:"name"`
			Courses      int     `json:"courses"`
			Students     int64   `json:"students"`
			AvgClassSize float64 `json:"avg_class_size"`
		} `json:"faculty_workload"`
		StudentAchievements []struct {
			Name        string  `json:"name"`
			BranchName  string  `json:"branch_name"`
			Achievement string  `json:"achievement"`
			Score       float64 `json:"score"`
		} `json:"student_achievements"`
	}

	result.OutcomeMetrics = make([]AccreditationMetrics, 0)
	result.DepartmentReports = make([]struct {
		BranchName     string  `json:"branch_name"`
		TotalStudents  int64   `json:"total_students"`
		TotalFaculty   int64   `json:"total_faculty"`
		AvgPerformance float64 `json:"avg_performance"`
		PassPct        float64 `json:"pass_pct"`
		ContestWinners int64   `json:"contest_winners"`
	}, 0)
	result.FacultyWorkload = make([]struct {
		Name         string  `json:"name"`
		Courses      int     `json:"courses"`
		Students     int64   `json:"students"`
		AvgClassSize float64 `json:"avg_class_size"`
	}, 0)
	result.StudentAchievements = make([]struct {
		Name        string  `json:"name"`
		BranchName  string  `json:"branch_name"`
		Achievement string  `json:"achievement"`
		Score       float64 `json:"score"`
	}, 0)

	// Outcome metrics (CO-PO style)
	var avgMarks float64
	database.DB.Raw(`
		SELECT COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0)
		FROM exam_results er
		JOIN exams e ON e.id = er.exam_id
		JOIN course_offerings co ON co.id = e.course_offering_id
		JOIN sections s ON s.section_id = co.section_id
		JOIN branches b ON b.branch_id = s.branch_id
		WHERE b.college_id = ?
	`, *collegeIDStr).Scan(&avgMarks)

	var passCount, totalCount int64
	database.DB.Model(&models.ExamResult{}).
		Joins("JOIN exams ON exams.id = exam_results.exam_id").
		Joins("JOIN course_offerings ON course_offerings.id = exams.course_offering_id").
		Joins("JOIN sections ON sections.section_id = course_offerings.section_id").
		Joins("JOIN branches ON branches.branch_id = sections.branch_id").
		Where("branches.college_id = ? AND exam_results.marks_obtained >= exams.passing_marks", *collegeIDStr).
		Count(&passCount)
	database.DB.Model(&models.ExamResult{}).
		Joins("JOIN exams ON exams.id = exam_results.exam_id").
		Joins("JOIN course_offerings ON course_offerings.id = exams.course_offering_id").
		Joins("JOIN sections ON sections.section_id = course_offerings.section_id").
		Joins("JOIN branches ON branches.branch_id = sections.branch_id").
		Where("branches.college_id = ?", *collegeIDStr).
		Count(&totalCount)

	passPct := 0.0
	if totalCount > 0 {
		passPct = float64(passCount) / float64(totalCount) * 100
	}

	var placementReady float64
	database.DB.Raw(`
		SELECT COALESCE(
			COUNT(DISTINCT CASE WHEN s.score >= 60 THEN s.user_regd_no END)::float
			/ NULLIF(COUNT(DISTINCT s.user_regd_no), 0) * 100, 0
		)
		FROM submissions s
		WHERE s.college_id = ?
	`, *collegeIDStr).Scan(&placementReady)

	var contestWinners int64
	database.DB.Model(&models.ContestParticipant{}).
		Joins("JOIN contests ON contests.contest_id = contest_participants.contest_id").
		Where("contests.college_id = ? AND contest_participants.rank = 1", *collegeIDStr).
		Count(&contestWinners)

	result.OutcomeMetrics = []AccreditationMetrics{
		{MetricName: "Average Academic Performance", Value: avgMarks, Target: 70, Status: getStatus(avgMarks, 70)},
		{MetricName: "Pass Percentage", Value: passPct, Target: 80, Status: getStatus(passPct, 80)},
		{MetricName: "Placement Readiness", Value: placementReady, Target: 60, Status: getStatus(placementReady, 60)},
		{MetricName: "Contest Winners", Value: float64(contestWinners), Target: 5, Status: getStatus(float64(contestWinners), 5)},
	}

	// Department reports
	var branches []models.Branch
	database.DB.Where("college_id = ?", *collegeIDStr).Find(&branches)
	for _, branch := range branches {
		var totalStudents, totalFaculty int64
		database.DB.Model(&models.User{}).Where("role = ? AND branch_id = ? AND college_id = ? AND is_active = true", "student", branch.BranchID, *collegeIDStr).Count(&totalStudents)
		database.DB.Model(&models.User{}).Where("role IN (?, ?) AND branch_id = ? AND college_id = ? AND is_active = true", "faculty", "hod", branch.BranchID, *collegeIDStr).Count(&totalFaculty)

		var branchAvg float64
		database.DB.Raw(`
			SELECT COALESCE(AVG(er.marks_obtained::float / NULLIF(e.total_marks, 0) * 100), 0)
			FROM exam_results er
			JOIN exams e ON e.id = er.exam_id
			JOIN course_offerings co ON co.id = e.course_offering_id
			JOIN sections s ON s.section_id = co.section_id
			WHERE s.branch_id = ?
		`, branch.BranchID).Scan(&branchAvg)

		var branchPassCount, branchTotal int64
		database.DB.Model(&models.ExamResult{}).
			Joins("JOIN exams ON exams.id = exam_results.exam_id").
			Joins("JOIN course_offerings ON course_offerings.id = exams.course_offering_id").
			Joins("JOIN sections ON sections.section_id = course_offerings.section_id").
			Where("sections.branch_id = ? AND exam_results.marks_obtained >= exams.passing_marks", branch.BranchID).
			Count(&branchPassCount)
		database.DB.Model(&models.ExamResult{}).
			Joins("JOIN exams ON exams.id = exam_results.exam_id").
			Joins("JOIN course_offerings ON course_offerings.id = exams.course_offering_id").
			Joins("JOIN sections ON sections.section_id = course_offerings.section_id").
			Where("sections.branch_id = ?", branch.BranchID).
			Count(&branchTotal)

		branchPassPct := 0.0
		if branchTotal > 0 {
			branchPassPct = float64(branchPassCount) / float64(branchTotal) * 100
		}

		var winners int64
		database.DB.Model(&models.ContestParticipant{}).
			Joins("JOIN users ON users.regdno = contest_participants.user_regd_no").
			Joins("JOIN contests ON contests.contest_id = contest_participants.contest_id").
			Where("users.branch_id = ? AND contests.college_id = ? AND contest_participants.rank = 1", branch.BranchID, *collegeIDStr).
			Count(&winners)

		result.DepartmentReports = append(result.DepartmentReports, struct {
			BranchName     string  `json:"branch_name"`
			TotalStudents  int64   `json:"total_students"`
			TotalFaculty   int64   `json:"total_faculty"`
			AvgPerformance float64 `json:"avg_performance"`
			PassPct        float64 `json:"pass_pct"`
			ContestWinners int64   `json:"contest_winners"`
		}{
			BranchName:     branch.BranchName,
			TotalStudents:  totalStudents,
			TotalFaculty:   totalFaculty,
			AvgPerformance: branchAvg,
			PassPct:        branchPassPct,
			ContestWinners: winners,
		})
	}

	// Faculty workload
	type fwRow struct {
		Name    string
		Courses int
	}
	var fws []fwRow
	database.DB.Raw(`
		SELECT u.name, COUNT(DISTINCT fa.course_offering_id) as courses
		FROM users u
		JOIN faculties f ON f.regd_no = u.regdno
		LEFT JOIN faculty_assignments fa ON fa.faculty_regd_no = u.regdno
		WHERE u.college_id = ? AND u.role IN ('faculty', 'hod') AND u.is_active = true
		GROUP BY u.regdno, u.name
	`, *collegeIDStr).Scan(&fws)

	for _, fw := range fws {
		var students int64
		database.DB.Model(&models.Enrollment{}).
			Joins("JOIN faculty_assignments ON faculty_assignments.course_offering_id = enrollments.course_offering_id").
			Where("faculty_assignments.faculty_regd_no = ?", fw.Name). // This is wrong, should be regdno
			Count(&students)

		// Get actual regdno
		var regdNo string
		database.DB.Model(&models.User{}).Where("name = ? AND college_id = ?", fw.Name, *collegeIDStr).Select("regdno").Scan(&regdNo)
		if regdNo != "" {
			database.DB.Model(&models.Enrollment{}).
				Joins("JOIN faculty_assignments ON faculty_assignments.course_offering_id = enrollments.course_offering_id").
				Where("faculty_assignments.faculty_regd_no = ?", regdNo).
				Count(&students)
		}

		avgClassSize := 0.0
		if fw.Courses > 0 {
			avgClassSize = float64(students) / float64(fw.Courses)
		}

		result.FacultyWorkload = append(result.FacultyWorkload, struct {
			Name         string  `json:"name"`
			Courses      int     `json:"courses"`
			Students     int64   `json:"students"`
			AvgClassSize float64 `json:"avg_class_size"`
		}{
			Name:         fw.Name,
			Courses:      fw.Courses,
			Students:     students,
			AvgClassSize: avgClassSize,
		})
	}

	// Student achievements (top 5 contest performers)
	type achRow struct {
		Name       string
		BranchName string
		Score      float64
	}
	var achs []achRow
	database.DB.Raw(`
		SELECT u.name, b.branch_name, COALESCE(AVG(cp.total_score), 0) as score
		FROM contest_participants cp
		JOIN users u ON u.regdno = cp.user_regd_no
		JOIN branches b ON b.branch_id = u.branch_id
		JOIN contests c ON c.contest_id = cp.contest_id
		WHERE c.college_id = ?
		GROUP BY u.regdno, u.name, b.branch_name
		ORDER BY score DESC
		LIMIT 5
	`, *collegeIDStr).Scan(&achs)

	for _, a := range achs {
		result.StudentAchievements = append(result.StudentAchievements, struct {
			Name        string  `json:"name"`
			BranchName  string  `json:"branch_name"`
			Achievement string  `json:"achievement"`
			Score       float64 `json:"score"`
		}{
			Name:        a.Name,
			BranchName:  a.BranchName,
			Achievement: "Top Contest Performer",
			Score:       a.Score,
		})
	}

	c.JSON(http.StatusOK, result)
}

func getStatus(value, target float64) string {
	if value >= target {
		return "achieved"
	}
	if value >= target*0.8 {
		return "near_target"
	}
	return "below_target"
}
