package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"coding-platform/services"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DashboardResponse contains all dashboard data
type DashboardResponse struct {
	TopicProficiency []TopicProgress    `json:"topic_proficiency"`
	Activity         []ActivityDay      `json:"activity"`
	Leaderboard      []LeaderboardEntry `json:"leaderboard"`
	Courses          []CourseAssignment `json:"courses"`
	Stats            StatsData          `json:"stats"`
	CurrentUserRank  int                `json:"current_user_rank"`
}

type CourseAssignment struct {
	CourseCode   string `json:"course_code"`
	CourseName   string `json:"course_name"`
	CourseType   string `json:"course_type"`
	SectionName  string `json:"section_name,omitempty"`
	AcademicYear string `json:"academic_year,omitempty"`
	ID           uint   `json:"id"`
	Credits      int    `json:"credits"`
	CohortYear   int    `json:"cohort_year,omitempty"`
}

type StatsData struct {
	TotalSolved    int     `json:"total_solved"`
	TotalProblems  int     `json:"total_problems"`
	CurrentStreak  int     `json:"current_streak"`
	Accuracy       float64 `json:"accuracy"`
	AccuracyChange float64 `json:"accuracy_change"`
	CollegeRank    int     `json:"college_rank"`
	TotalPoints    int     `json:"total_points"`
}

type TopicProgress struct {
	Name     string `json:"name"`
	Progress int    `json:"progress"`
	Easy     int    `json:"easy"`
	Medium   int    `json:"medium"`
	Hard     int    `json:"hard"`
	Total    int    `json:"total"`
}

type ActivityDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type LeaderboardEntry struct {
	RegdNo string `json:"regdno"`
	Name   string `json:"name"`
	Rank   int    `json:"rank"`
	UserID uint   `json:"user_id,omitempty"`
	Solved int    `json:"solved"`
	Points int    `json:"points"`
}

// GetDashboard returns dashboard data for the authenticated user
func GetDashboard(c *gin.Context) {
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	role, _ := c.Get("role")
	userRole := ""
	if role != nil {
		userRole = role.(string)
	}

	// Get stats
	stats := getStats(userRegdNo.(string))

	// Get topic proficiency
	topics := getTopicProficiency(userRegdNo.(string))

	// Get activity heatmap (last 365 days)
	activity := getActivity(userRegdNo.(string))

	// Get leaderboard
	leaderboard, userRank := getLeaderboard(userRegdNo.(string))

	// Get courses based on role
	courses := getCoursesByRole(userRegdNo.(string), userRole)

	response := DashboardResponse{
		Stats:            stats,
		TopicProficiency: topics,
		Activity:         activity,
		Leaderboard:      leaderboard,
		CurrentUserRank:  userRank,
		Courses:          courses,
	}

	c.JSON(http.StatusOK, response)
}

func getStats(userRegdNo string) StatsData {
	var stats StatsData

	// Get user's college
	var user models.User
	database.DB.Where("regdno = ?", userRegdNo).First(&user)

	// Check cache first
	cache := services.GetCache()
	cacheKey := fmt.Sprintf("%s%s", services.DashboardCachePrefix, userRegdNo)
	var cachedStats StatsData
	if cache.Get(cacheKey+":stats", &cachedStats) {
		return cachedStats
	}

	// Total problems solved
	var totalSolved int64
	database.DB.Model(&models.UserProblemCompletion{}).
		Where("user_regd_no = ?", userRegdNo).
		Count(&totalSolved)
	stats.TotalSolved = int(totalSolved)

	// Total problems available (filtered by college)
	var totalProblems int64
	if user.CollegeID != nil {
		database.DB.Model(&models.Problem{}).
			Where("college_id = ?", *user.CollegeID).
			Count(&totalProblems)
	}
	stats.TotalProblems = int(totalProblems)

	// Get streak data
	var streak models.UserStreak
	if err := database.DB.Where("user_regd_no = ?", userRegdNo).First(&streak).Error; err == nil {
		// Stale-streak guard: if the user's last activity was before yesterday,
		// their streak has lapsed. Show 0 immediately without waiting for the
		// next submission to trigger the SQL reset.
		yesterday := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -1)
		lastActivity := streak.LastActivityDate.Truncate(24 * time.Hour)
		if !lastActivity.IsZero() && lastActivity.Before(yesterday) {
			stats.CurrentStreak = 0
		} else {
			stats.CurrentStreak = streak.CurrentStreak
		}
		stats.TotalPoints = streak.TotalPoints
	}

	// Calculate accuracy based on unique problems solved correctly vs attempted
	// Accuracy = (problems solved correctly) / (unique problems attempted) * 100
	// This prevents accuracy from decreasing when re-submitting the same problem
	var uniqueProblemsAttempted, uniqueProblemsSolved int64

	// Count unique problems the user has attempted (submitted at least once)
	database.DB.Raw(`
		SELECT COUNT(DISTINCT problem_id)
		FROM submissions
		WHERE user_regd_no = ?
	`, userRegdNo).Scan(&uniqueProblemsAttempted)

	// Count unique problems the user has solved correctly (at least one passing submission)
	database.DB.Raw(`
		SELECT COUNT(DISTINCT problem_id)
		FROM submissions
		WHERE user_regd_no = ? AND passed = ?
	`, userRegdNo, true).Scan(&uniqueProblemsSolved)

	if uniqueProblemsAttempted > 0 {
		stats.Accuracy = float64(uniqueProblemsSolved) / float64(uniqueProblemsAttempted) * 100
	}

	// Calculate accuracy change (compare with last week's accuracy)
	var lastWeekSolved, lastWeekAttempted int64
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	database.DB.Raw(`
		SELECT COUNT(DISTINCT problem_id)
		FROM submissions
		WHERE user_regd_no = ? AND submitted_at < ?
	`, userRegdNo, oneWeekAgo).Scan(&lastWeekAttempted)
	database.DB.Raw(`
		SELECT COUNT(DISTINCT problem_id)
		FROM submissions
		WHERE user_regd_no = ? AND passed = ? AND submitted_at < ?
	`, userRegdNo, true, oneWeekAgo).Scan(&lastWeekSolved)

	var lastWeekAccuracy float64
	if lastWeekAttempted > 0 {
		lastWeekAccuracy = float64(lastWeekSolved) / float64(lastWeekAttempted) * 100
	}
	stats.AccuracyChange = stats.Accuracy - lastWeekAccuracy

	// Calculate college rank using optimized single-pass query
	// Uses window function approach instead of 3 correlated subqueries
	if user.CollegeID != nil {
		var rank int64
		database.DB.Raw(`
			WITH user_scores AS (
				SELECT
					u.regdno,
					COALESCE(SUM(p.points), 0) as total_points,
					COUNT(upc.id) as solved_count
				FROM users u
				LEFT JOIN user_problem_completions upc ON upc.user_regd_no = u.regdno
				LEFT JOIN problems p ON p.id = upc.problem_id
				WHERE u.college_id = ? AND LOWER(u.role) = 'student'
				GROUP BY u.regdno
			),
			current_user AS (
				SELECT total_points, solved_count FROM user_scores WHERE regdno = ?
			),
			ranked AS (
				SELECT
					regdno,
					total_points,
					solved_count,
					RANK() OVER (ORDER BY total_points DESC, solved_count DESC) as rnk
				FROM user_scores
			)
			SELECT rnk FROM ranked WHERE regdno = ?
		`, *user.CollegeID, userRegdNo, userRegdNo).Scan(&rank)
		stats.CollegeRank = int(rank)
		if stats.CollegeRank == 0 {
			stats.CollegeRank = 1
		}
	} else {
		stats.CollegeRank = 1
	}

	// Cache the result
	cache.Set(cacheKey+":stats", stats, services.DashboardCacheTTL)

	return stats
}

func getTopicProficiency(userRegdNo string) []TopicProgress {
	var topics []TopicProgress

	// Get user's college
	var user models.User
	database.DB.Where("regdno = ?", userRegdNo).First(&user)

	// Check cache first
	cache := services.GetCache()
	cacheKey := fmt.Sprintf("%s%s", services.DashboardCachePrefix, userRegdNo)
	var cachedTopics []TopicProgress
	if cache.Get(cacheKey+":topics", &cachedTopics) {
		return cachedTopics
	}

	// Single optimized query to get all topic proficiency data
	// This replaces 7 queries per topic with a single GROUP BY query
	type TopicStat struct {
		Topic      string `gorm:"column:topic"`
		EasySolved int64  `gorm:"column:easy_solved"`
		MedSolved  int64  `gorm:"column:med_solved"`
		HardSolved int64  `gorm:"column:hard_solved"`
		EasyTotal  int64  `gorm:"column:easy_total"`
		MedTotal   int64  `gorm:"column:med_total"`
		HardTotal  int64  `gorm:"column:hard_total"`
	}

	var topicStats []TopicStat
	// Build parameterized query to avoid SQL injection
	collegeClause, collegeArgs := getCollegeFilterClause(&user)
	queryArgs := []interface{}{userRegdNo, userRegdNo}
	queryArgs = append(queryArgs, collegeArgs...)

	collectQuery := database.DB.Raw(`
		WITH topic_problems AS (
			SELECT
				TRIM(p.tags) as topic,
				p.difficulty,
				p.id as problem_id,
				CASE WHEN upc.user_regd_no = ? THEN 1 ELSE 0 END as is_solved
			FROM problems p
			LEFT JOIN user_problem_completions upc ON upc.problem_id = p.id AND upc.user_regd_no = ?
			WHERE p.tags IS NOT NULL AND p.tags != '' AND p.tags != ' '
			`+collegeClause+`
		)
		SELECT
			topic,
			COUNT(CASE WHEN is_solved = 1 AND difficulty = 'easy' THEN 1 END) as easy_solved,
			COUNT(CASE WHEN is_solved = 1 AND difficulty = 'medium' THEN 1 END) as med_solved,
			COUNT(CASE WHEN is_solved = 1 AND difficulty = 'hard' THEN 1 END) as hard_solved,
			COUNT(CASE WHEN difficulty = 'easy' THEN 1 END) as easy_total,
			COUNT(CASE WHEN difficulty = 'medium' THEN 1 END) as med_total,
			COUNT(CASE WHEN difficulty = 'hard' THEN 1 END) as hard_total
		FROM topic_problems
		GROUP BY topic
		HAVING COUNT(*) > 0
		ORDER BY topic
	`, queryArgs...).Scan(&topicStats)

	if collectQuery.Error != nil {
		fmt.Printf("Error fetching topic proficiency: %v\n", collectQuery.Error)
		return []TopicProgress{}
	}

	// Convert to response format
	for _, ts := range topicStats {
		total := ts.EasyTotal + ts.MedTotal + ts.HardTotal
		progress := 0
		if total > 0 {
			progress = int((ts.EasySolved + ts.MedSolved + ts.HardSolved) * 100 / total)
		}
		topics = append(topics, TopicProgress{
			Name:     ts.Topic,
			Progress: progress,
			Easy:     int(ts.EasySolved),
			Medium:   int(ts.MedSolved),
			Hard:     int(ts.HardSolved),
			Total:    int(total),
		})
	}

	// Cache the result
	cache.Set(cacheKey+":topics", topics, services.TopicProficiencyTTL)

	return topics
}

// getCollegeFilterClause returns a SQL clause and parameter value to filter by college
// This avoids SQL injection by using parameterized queries
func getCollegeFilterClause(user *models.User) (clause string, args []interface{}) {
	if user.CollegeID != nil {
		return "AND p.college_id = ?", []interface{}{*user.CollegeID}
	}
	return "", nil
}

func getActivity(userRegdNo string) []ActivityDay {
	var activity []ActivityDay

	// Get last 365 days of activity from UserActivity table
	var activities []models.UserActivity
	startDate := time.Now().AddDate(-1, 0, 0)

	database.DB.Where("user_regd_no = ? AND date >= ?", userRegdNo, startDate).
		Order("date ASC").
		Find(&activities)

	// Create a map for quick lookup
	activityMap := make(map[string]int)
	for _, a := range activities {
		dateStr := a.Date.Format("2006-01-02")
		activityMap[dateStr] = a.Count
	}

	// Generate all days in range
	for d := startDate; !d.After(time.Now()); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		count := activityMap[dateStr]
		activity = append(activity, ActivityDay{
			Date:  dateStr,
			Count: count,
		})
	}

	return activity
}

func getLeaderboard(userRegdNo string) ([]LeaderboardEntry, int) {
	var entries []LeaderboardEntry
	var userRank int

	// Get user's college
	var user models.User
	database.DB.Where("regdno = ?", userRegdNo).First(&user)

	if user.CollegeID == nil {
		// No college assigned - return empty
		return []LeaderboardEntry{}, 0
	}

	// Check cache first
	cache := services.GetCache()
	cacheKey := fmt.Sprintf("%scollege:%s", services.LeaderboardCachePrefix, *user.CollegeID)
	type CachedLeaderboard struct {
		Entries  []LeaderboardEntry
		UserRank int
	}
	var cached CachedLeaderboard
	if cache.Get(cacheKey, &cached) {
		// Find user's rank in cached data
		for i, e := range cached.Entries {
			if e.RegdNo == userRegdNo {
				return cached.Entries, i + 1
			}
		}
		// User not in top entries, return cached entries but recalculate rank
		entries = cached.Entries
		// Fall through to rank calculation below
	}

	if len(entries) == 0 {
		// Get top users by problems solved (filtered by college)
		var results []struct {
			UserRegdNo string
			Username   string
			Solved     int64
			Points     int
		}

		err := database.DB.Raw(`
			SELECT
				u.regdno as user_regd_no,
				u.name as username,
				COUNT(upc.id) as solved,
				COALESCE(SUM(s.score), 0) as points
			FROM users u
			LEFT JOIN user_problem_completions upc ON upc.user_regd_no = u.regdno
			LEFT JOIN submissions s ON s.id = upc.first_submission_id
			WHERE LOWER(u.role) = 'student' AND u.college_id = ?
			GROUP BY u.regdno, u.name
			ORDER BY points DESC, solved DESC
			LIMIT 20
		`, *user.CollegeID).Scan(&results).Error
		if err != nil {
			fmt.Printf("Error fetching leaderboard: %v\n", err)
			return []LeaderboardEntry{}, 0
		}

		// Build leaderboard
		for i, r := range results {
			entries = append(entries, LeaderboardEntry{
				Rank:   i + 1,
				RegdNo: r.UserRegdNo,
				Name:   r.Username,
				Solved: int(r.Solved),
				Points: r.Points,
			})
		}

		// Cache the leaderboard
		cache.Set(cacheKey, CachedLeaderboard{Entries: entries, UserRank: 0}, services.LeaderboardCacheTTL)
	}

	// Find user rank in cached results
	for i, e := range entries {
		if e.RegdNo == userRegdNo {
			return entries, i + 1
		}
	}

	// If user not in top 20, find their actual rank (filtered by college)
	var rankResult struct {
		Rank int
	}
	database.DB.Raw(`
		WITH user_scores AS (
			SELECT
				u.regdno,
				COALESCE(SUM(s.score), 0) as total_points
			FROM users u
			LEFT JOIN user_problem_completions upc ON upc.user_regd_no = u.regdno
			LEFT JOIN submissions s ON s.id = upc.first_submission_id
			WHERE u.college_id = ? AND LOWER(u.role) = 'student'
			GROUP BY u.regdno
		),
		current_score AS (
			SELECT total_points FROM user_scores WHERE regdno = ?
		)
		SELECT COUNT(*) + 1 as rank
		FROM user_scores
		WHERE total_points > (SELECT total_points FROM current_score)
	`, *user.CollegeID, userRegdNo).Scan(&rankResult)
	userRank = rankResult.Rank
	if userRank == 0 {
		userRank = 1
	}

	return entries, userRank
}

// getCoursesByRole returns courses based on user role (batch-based)
// Faculty: courses they are assigned to teach via FacultyAssignment
// Student: courses they are enrolled in via Enrollment/CourseOffering
func getCoursesByRole(userRegdNo string, role string) []CourseAssignment {
	var courses []CourseAssignment

	if role == "faculty" {
		// Use FacultyAssignment model (new batch-based)
		type AssignedCourse struct {
			CourseCode   string
			CourseName   string
			CourseType   string
			SectionName  string
			AcademicYear string
			CourseID     uint
			Credits      int
			CohortYear   int
		}

		var assignedCourses []AssignedCourse
		err := database.DB.Raw(`
			SELECT
				c.id as course_id,
				c.course_code,
				c.course_name,
				c.course_type,
				c.credits,
				s.section_name,
				s.cohort_year,
				ay.name as academic_year
			FROM faculty_assignments fa
			INNER JOIN course_offerings co ON co.id = fa.course_offering_id
			INNER JOIN courses c ON c.id = co.course_id
			INNER JOIN sections s ON s.section_id = co.section_id
			INNER JOIN academic_years ay ON ay.academic_year_id = co.academic_year_id
			WHERE fa.faculty_regd_no = ? AND fa.is_active = true
			ORDER BY c.course_code
		`, userRegdNo).Scan(&assignedCourses)

		if err != nil {
			fmt.Printf("Error fetching faculty courses: %v\n", err)
		}

		for _, ac := range assignedCourses {
			courses = append(courses, CourseAssignment{
				ID:           ac.CourseID,
				CourseCode:   ac.CourseCode,
				CourseName:   ac.CourseName,
				CourseType:   ac.CourseType,
				Credits:      ac.Credits,
				SectionName:  ac.SectionName,
				CohortYear:   ac.CohortYear,
				AcademicYear: ac.AcademicYear,
			})
		}

	} else if role == "student" {
		// Get student's branch, cohort, and program info for validation
		type StudentInfo struct {
			BranchID   uint
			CohortYear int
			ProgramID  uint
		}
		var studentInfo StudentInfo
		database.DB.Raw(`
			SELECT st.branch_id, st.cohort_year, b.program_id
			FROM students st
			INNER JOIN branches b ON b.branch_id = st.branch_id
			WHERE st.regd_no = ?
		`, userRegdNo).First(&studentInfo)

		// Get student's courses via Enrollment and CourseOffering (batch-based)
		// Filter strictly by exact cohort_year (joining year) + branch + program match
		type EnrolledCourse struct {
			CourseCode     string
			CourseName     string
			CourseType     string
			SectionName    string
			AcademicYear   string
			EnrollmentType string
			CourseID       uint
			Credits        int
			CohortYear     int
		}

		var enrolledCourses []EnrolledCourse
		err := database.DB.Raw(`
			SELECT
				c.id as course_id,
				c.course_code,
				c.course_name,
				c.course_type,
				c.credits,
				s.section_name,
				s.cohort_year,
				ay.name as academic_year,
				e.type as enrollment_type
			FROM enrollments e
			INNER JOIN course_offerings co ON co.id = e.course_offering_id
			INNER JOIN courses c ON c.id = co.course_id
			INNER JOIN sections s ON s.section_id = co.section_id
			INNER JOIN branches b ON b.branch_id = s.branch_id
			INNER JOIN academic_years ay ON ay.academic_year_id = co.academic_year_id
			WHERE e.student_regdno = ?
			  AND e.status IN ('enrolled', 'completed')
			  AND co.is_active = true
			  AND co.deleted_at IS NULL
			  AND s.branch_id = ?
			  AND s.cohort_year = ?
			  AND b.program_id = ?
			ORDER BY ay.start_date DESC, c.course_code
		`, userRegdNo, studentInfo.BranchID, studentInfo.CohortYear, studentInfo.ProgramID).Scan(&enrolledCourses)

		if err != nil {
			fmt.Printf("Error fetching student courses: %v\n", err)
		}

		for _, ec := range enrolledCourses {
			courses = append(courses, CourseAssignment{
				ID:           ec.CourseID,
				CourseCode:   ec.CourseCode,
				CourseName:   ec.CourseName,
				CourseType:   ec.CourseType,
				Credits:      ec.Credits,
				SectionName:  ec.SectionName,
				CohortYear:   ec.CohortYear,
				AcademicYear: ec.AcademicYear,
			})
		}
	}

	return courses
}
