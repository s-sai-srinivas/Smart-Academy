package handlers

import (
	"coding-platform/database"
	"coding-platform/models"
	"coding-platform/services"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// =====================================================
// Contest Results CSV Export (Faculty/HOD)
// =====================================================

// GetContestResultsCSV returns an overview CSV of contest results
// Accessible by faculty and HOD for ended contests
func GetContestResultsCSV(c *gin.Context) {
	contestID := c.Param("id")
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check role - faculty or HOD can access
	if user.Role != "faculty" && user.Role != "hod" && user.Role != "admin" && user.Role != "college_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only faculty, HOD, or admin can access contest results"})
		return
	}

	// Get contest
	var contest models.Contest
	if err := database.DB.First(&contest, contestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Verify college access
	if err := validateCollegeAccess(user.CollegeID, contest.CollegeID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Check if contest has ended
	if !isContestEnded(contest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Contest has not ended yet. Results available only after contest ends."})
		return
	}

	// Get all eligible students for this contest (based on cohort/branch targeting)
	contestIDUint, _ := strconv.ParseUint(contestID, 10, 32)

	// Get contest problems for calculating totals
	var contestProblems []models.ContestProblem
	database.DB.Where("contest_id = ?", contestID).Order("problem_order").Find(&contestProblems)
	totalProblems := len(contestProblems)
	maxTotalScore := 0
	for _, cp := range contestProblems {
		maxTotalScore += cp.Points
	}

	// Get all participants with their details
	type ParticipantResult struct {
		RegisteredAt       time.Time
		UserRegdNo         string
		Name               string
		BranchName         string
		SectionName        string
		TotalScore         int
		ProblemsSolved     int
		ProblemsAttempted  int
		TotalAttempts      int
		SuccessfulAttempts int
		HasFinished        bool
	}

	var participants []ParticipantResult
	query := database.DB.Raw(`
		SELECT
			cp.user_regd_no,
			u.name,
			COALESCE(b.branch_name, '') as branch_name,
			COALESCE(s.section_name, '') as section_name,
			cp.registered_at,
			COALESCE(solved_stats.total_score, 0) as total_score,
			COALESCE(solved_stats.problems_solved, 0) as problems_solved,
			COALESCE(attempt_stats.problems_attempted, 0) as problems_attempted,
			COALESCE(attempt_stats.total_attempts, 0) as total_attempts,
			COALESCE(attempt_stats.successful_attempts, 0) as successful_attempts,
			cp.has_finished
		FROM contest_participants cp
		INNER JOIN users u ON u.regdno = cp.user_regd_no
		LEFT JOIN students st ON st.regd_no = cp.user_regd_no
		LEFT JOIN branches b ON b.branch_id = st.branch_id
		LEFT JOIN sections s ON s.section_id = st.section_id
		LEFT JOIN (
			SELECT
				solved_by_problem.contest_id,
				solved_by_problem.user_regd_no,
				SUM(solved_by_problem.problem_score) as total_score,
				COUNT(*) as problems_solved
			FROM (
				SELECT
					contest_id,
					user_regd_no,
					problem_id,
					MAX(score) as problem_score
				FROM contest_submissions
				WHERE contest_id = ? AND passed = true
				GROUP BY contest_id, user_regd_no, problem_id
			) solved_by_problem
			GROUP BY solved_by_problem.contest_id, solved_by_problem.user_regd_no
		) solved_stats ON solved_stats.contest_id = cp.contest_id
			AND solved_stats.user_regd_no = cp.user_regd_no
		LEFT JOIN (
			SELECT
				contest_id,
				user_regd_no,
				COUNT(DISTINCT problem_id) as problems_attempted,
				COUNT(*) as total_attempts,
				SUM(CASE WHEN passed = true THEN 1 ELSE 0 END) as successful_attempts
			FROM contest_submissions
			WHERE contest_id = ?
			GROUP BY contest_id, user_regd_no
		) attempt_stats ON attempt_stats.contest_id = cp.contest_id
			AND attempt_stats.user_regd_no = cp.user_regd_no
		WHERE cp.contest_id = ?
		ORDER BY total_score DESC, problems_solved DESC, cp.registered_at ASC
	`, contestIDUint, contestIDUint, contestIDUint)

	if err := query.Scan(&participants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participant results"})
		return
	}

	// Get eligible students who didn't join
	type EligibleStudent struct {
		UserRegdNo  string
		Name        string
		BranchName  string
		SectionName string
	}

	var eligibleStudents []EligibleStudent
	if contest.TargetCohort != nil {
		// Query students based on target cohort and branch
		branchFilter := ""
		args := []interface{}{*contest.CollegeID, *contest.TargetCohort, contestIDUint}

		if contest.TargetBranchID != nil {
			branchFilter = " AND st.branch_id = ?"
			args = append(args, *contest.TargetBranchID)
		}

		database.DB.Raw(`
			SELECT
				u.regdno as user_regd_no,
				u.name,
				COALESCE(b.branch_name, '') as branch_name,
				COALESCE(s.section_name, '') as section_name
			FROM students st
			INNER JOIN users u ON u.regdno = st.regd_no
			LEFT JOIN branches b ON b.branch_id = st.branch_id
			LEFT JOIN sections s ON s.section_id = st.section_id
			WHERE u.college_id = ? AND st.cohort_year = ?`+branchFilter+`
			AND u.regdno NOT IN (SELECT user_regd_no FROM contest_participants WHERE contest_id = ?)
		`, args...).Scan(&eligibleStudents)
	}

	// Build CSV
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=contest_%s_results_overview.csv", contestID))

	// CSV header
	_, _ = c.Writer.WriteString("Registration Number,Student Name,Branch,Section,Joined Contest,Join Time,Rank,Total Score,Max Score,Problems Solved,Problems Attempted,Problems Total,Accuracy Percentage,Submission Success Rate\n")

	// Write participant rows (with ranks)
	for i, p := range participants {
		// Calculate rank (ICPC style: same score = same rank)
		actualRank := i + 1
		if i > 0 && p.TotalScore == participants[i-1].TotalScore && p.ProblemsSolved == participants[i-1].ProblemsSolved {
			actualRank = i
		}

		// Calculate accuracy and success rate
		accuracyPercent := 0.0
		if p.TotalAttempts > 0 {
			accuracyPercent = float64(p.SuccessfulAttempts) / float64(p.TotalAttempts) * 100
		}

		successRate := 0.0
		if totalProblems > 0 {
			successRate = float64(p.ProblemsSolved) / float64(totalProblems) * 100
		}

		joinTime := p.RegisteredAt.Format("2006-01-02 15:04:05")

		row := fmt.Sprintf("%s,%s,%s,%s,Yes,%s,%d,%d,%d,%d,%d,%d,%.2f%%,%.2f%%\n",
			p.UserRegdNo,
			p.Name,
			p.BranchName,
			p.SectionName,
			joinTime,
			actualRank,
			p.TotalScore,
			maxTotalScore,
			p.ProblemsSolved,
			p.ProblemsAttempted,
			totalProblems,
			accuracyPercent,
			successRate,
		)
		_, _ = c.Writer.WriteString(row)
	}

	// Write eligible students who didn't join
	for _, s := range eligibleStudents {
		row := fmt.Sprintf("%s,%s,%s,%s,No,-,-,-,-,-,-,-,-\n",
			s.UserRegdNo,
			s.Name,
			s.BranchName,
			s.SectionName,
		)
		_, _ = c.Writer.WriteString(row)
	}
}

// GetContestDetailedCSV returns a detailed per-problem CSV of contest results
// Accessible by faculty and HOD for ended contests
func GetContestDetailedCSV(c *gin.Context) {
	contestID := c.Param("id")
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check role - faculty or HOD can access
	if user.Role != "faculty" && user.Role != "hod" && user.Role != "admin" && user.Role != "college_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only faculty, HOD, or admin can access contest results"})
		return
	}

	// Get contest
	var contest models.Contest
	if err := database.DB.First(&contest, contestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Verify college access
	if err := validateCollegeAccess(user.CollegeID, contest.CollegeID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Check if contest has ended
	if !isContestEnded(contest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Contest has not ended yet. Results available only after contest ends."})
		return
	}

	contestIDUint, _ := strconv.ParseUint(contestID, 10, 32)

	// Get contest problems
	var contestProblems []models.ContestProblem
	database.DB.Preload("Problem").Where("contest_id = ?", contestID).Order("problem_order").Find(&contestProblems)

	// Get all participants
	var participants []models.ContestParticipant
	database.DB.Where("contest_id = ?", contestID).Find(&participants)

	// Get user details
	regdNos := make([]string, len(participants))
	for i, p := range participants {
		regdNos[i] = p.UserRegdNo
	}

	var users []models.User
	database.DB.Where("regdno IN ?", regdNos).Find(&users)
	userMap := make(map[string]models.User)
	for _, u := range users {
		userMap[u.RegdNo] = u
	}

	// Get student details for branch/section
	var students []models.Student
	database.DB.Where("regd_no IN ?", regdNos).Find(&students)
	studentMap := make(map[string]models.Student)
	for _, s := range students {
		studentMap[s.RegdNo] = s
	}

	// Get branches
	var branches []models.Branch
	database.DB.Find(&branches)
	branchMap := make(map[uint]string)
	for _, b := range branches {
		branchMap[b.BranchID] = b.BranchName
	}

	// Get sections
	var sections []models.Section
	database.DB.Find(&sections)
	sectionMap := make(map[uint]string)
	for _, s := range sections {
		sectionMap[s.SectionID] = s.SectionName
	}

	// Get all submissions grouped by user and problem
	type ProblemSubmissionStats struct {
		FirstSubmitTime time.Time
		LastSubmitTime  time.Time
		UserRegdNo      string
		ProblemID       uint
		Attempts        int
		BestScore       int
		MaxScore        int
		IsSolved        bool
	}

	var submissionStats []ProblemSubmissionStats
	database.DB.Raw(`
		SELECT
			user_regd_no,
			problem_id,
			MIN(submitted_at) as first_submit_time,
			MAX(submitted_at) as last_submit_time,
			COUNT(*) as attempts,
			MAX(score) as best_score,
			MAX(max_score) as max_score,
			MAX(CASE WHEN passed = true THEN 1 ELSE 0 END) as is_solved
		FROM contest_submissions
		WHERE contest_id = ?
		GROUP BY user_regd_no, problem_id
		ORDER BY user_regd_no, problem_id
	`, contestIDUint).Scan(&submissionStats)

	// Create map for quick lookup
	statsByUserProblem := make(map[string]ProblemSubmissionStats) // key: "regdno_problemId"
	for _, stat := range submissionStats {
		key := fmt.Sprintf("%s_%d", stat.UserRegdNo, stat.ProblemID)
		statsByUserProblem[key] = stat
	}

	// Build CSV
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=contest_%s_results_detailed.csv", contestID))

	// CSV header
	_, _ = c.Writer.WriteString("Registration Number,Student Name,Branch,Section,Problem Title,Problem Order,Status,Score,Max Score,Attempts,First Submission Time,Last Submission Time\n")

	// Write rows for each participant and each problem
	for _, p := range participants {
		user := userMap[p.UserRegdNo]
		student := studentMap[p.UserRegdNo]

		branchName := ""
		sectionName := ""
		if student.BranchID > 0 {
			branchName = branchMap[student.BranchID]
		}
		if student.SectionID != nil {
			sectionName = sectionMap[*student.SectionID]
		}

		for _, cp := range contestProblems {
			key := fmt.Sprintf("%s_%d", p.UserRegdNo, cp.ProblemID)
			stats := statsByUserProblem[key]

			status := "Not Attempted"
			if stats.Attempts > 0 {
				if stats.IsSolved {
					status = "Solved"
				} else {
					status = "Attempted"
				}
			}

			firstSubmitTime := "-"
			lastSubmitTime := "-"
			if stats.Attempts > 0 {
				firstSubmitTime = stats.FirstSubmitTime.Format("2006-01-02 15:04:05")
				lastSubmitTime = stats.LastSubmitTime.Format("2006-01-02 15:04:05")
			}

			row := fmt.Sprintf("%s,%s,%s,%s,%s,%d,%s,%d,%d,%d,%s,%s\n",
				p.UserRegdNo,
				user.Name,
				branchName,
				sectionName,
				cp.Problem.Title,
				cp.ProblemOrder,
				status,
				stats.BestScore,
				cp.Points,
				stats.Attempts,
				firstSubmitTime,
				lastSubmitTime,
			)
			_, _ = c.Writer.WriteString(row)
		}
	}
}

// =====================================================
// HOD Contest Handlers
// =====================================================

// GetHODContests returns contests visible to HOD
// Always shows all contests for HOD's college (college-level, no branch filtering)
func GetHODContests(c *gin.Context) {
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.CollegeID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HOD must be assigned to a college"})
		return
	}

	// Always show all contests for HOD's college (college-level contests)
	query := database.DB.Model(&models.Contest{}).Where("college_id = ?", user.CollegeID)

	var contests []models.Contest
	if err := query.Order("created_at DESC").Find(&contests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load contests"})
		return
	}

	// Fetch counts
	countsMap := getContestCounts(contests)

	response := make([]ContestListResponse, 0, len(contests))
	for _, contest := range contests {
		counts := countsMap[contest.ContestID]
		response = append(response, ContestListResponse{
			ContestID:        contest.ContestID,
			Title:            contest.Title,
			StartTime:        contest.StartTime,
			EndTime:          contest.EndTime,
			ParticipantCount: counts.Participants,
			ProblemCount:     counts.Problems,
			IsActive:         isContestActive(contest),
			IsFrozen:         isLeaderboardFrozen(contest),
		})
	}

	c.JSON(http.StatusOK, response)
}

// GetHODContestDetails returns contest details for HOD view
func GetHODContestDetails(c *gin.Context) {
	// HOD can use the same GetContestDetails handler
	GetContestDetails(c)
}

// =====================================================
// Contest Plagiarism Handler
// =====================================================

// ContestPlagiarismStudent represents plagiarism data for a student in a contest
type ContestPlagiarismStudent struct {
	UserRegdNo             string  `json:"user_regd_no"`
	Name                   string  `json:"name"`
	Status                 string  `json:"status"`
	MaxPlagiarismPercent   float64 `json:"max_plagiarism_percent"`
	ProblemsWithPlagiarism int     `json:"problems_with_plagiarism"`
}

// GetContestPlagiarism returns plagiarism analysis for all contest submissions
// Accessible by faculty and HOD
func GetContestPlagiarism(c *gin.Context) {
	contestID := c.Param("id")
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check role - faculty or HOD can access
	if user.Role != "faculty" && user.Role != "hod" && user.Role != "admin" && user.Role != "college_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only faculty, HOD, or admin can access contest plagiarism results"})
		return
	}

	// Get contest
	var contest models.Contest
	if err := database.DB.First(&contest, contestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Verify college access
	if err := validateCollegeAccess(user.CollegeID, contest.CollegeID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	contestIDUint, _ := strconv.ParseUint(contestID, 10, 32)

	// Get all contest submissions
	type SubmissionInfo struct {
		UserRegdNo string
		SourceCode string
		ID         uint
		ProblemID  uint
		LanguageID int
	}

	var submissions []SubmissionInfo
	database.DB.Raw(`
		SELECT id, user_regd_no, problem_id, language_id, source_code
		FROM contest_submissions
		WHERE contest_id = ? AND passed = true
		ORDER BY user_regd_no, problem_id
	`, contestIDUint).Scan(&submissions)

	if len(submissions) < 2 {
		c.JSON(http.StatusOK, gin.H{
			"contest_id": contestIDUint,
			"message":    "Not enough submissions for plagiarism detection",
			"students":   []ContestPlagiarismStudent{},
		})
		return
	}

	// Group submissions by problem for plagiarism check
	submissionsByProblem := make(map[uint][]SubmissionInfo)
	for _, sub := range submissions {
		submissionsByProblem[sub.ProblemID] = append(submissionsByProblem[sub.ProblemID], sub)
	}

	// Run plagiarism check for each problem
	studentPlagiarismMap := make(map[string]*ContestPlagiarismStudent)

	for problemID, problemSubmissions := range submissionsByProblem {
		if len(problemSubmissions) < 2 {
			continue
		}

		// Group by language
		byLanguage := make(map[int][]SubmissionInfo)
		for _, sub := range problemSubmissions {
			byLanguage[sub.LanguageID] = append(byLanguage[sub.LanguageID], sub)
		}

		for _, langSubmissions := range byLanguage {
			if len(langSubmissions) < 2 {
				continue
			}

			// Prepare submissions for plagiarism check
			var plagSubmissions []services.SubmissionInfo
			for _, sub := range langSubmissions {
				plagSubmissions = append(plagSubmissions, services.SubmissionInfo{
					ID:         sub.ID,
					UserRegdNo: sub.UserRegdNo,
					SourceCode: sub.SourceCode,
					LanguageID: sub.LanguageID,
				})
			}

			// Run plagiarism check
			results, err := services.CheckPlagiarism(problemID, plagSubmissions)
			if err != nil {
				continue // Skip this problem on error
			}

			// Update student plagiarism data
			for _, result := range results {
				// Get users involved
				var user1RegdNo, user2RegdNo string
				for _, sub := range langSubmissions {
					if sub.ID == result.SubmissionID1 {
						user1RegdNo = sub.UserRegdNo
					}
					if sub.ID == result.SubmissionID2 {
						user2RegdNo = sub.UserRegdNo
					}
				}

				// Skip if same user
				if user1RegdNo == user2RegdNo {
					continue
				}

				// Update or create student entries
				for _, regdNo := range []string{user1RegdNo, user2RegdNo} {
					if _, exists := studentPlagiarismMap[regdNo]; !exists {
						studentPlagiarismMap[regdNo] = &ContestPlagiarismStudent{
							UserRegdNo:             regdNo,
							MaxPlagiarismPercent:   0,
							ProblemsWithPlagiarism: 0,
							Status:                 "SAFE",
						}
					}

					if result.SimilarityPercent > studentPlagiarismMap[regdNo].MaxPlagiarismPercent {
						studentPlagiarismMap[regdNo].MaxPlagiarismPercent = result.SimilarityPercent
					}

					if result.SimilarityPercent >= 30 {
						studentPlagiarismMap[regdNo].ProblemsWithPlagiarism++
					}

					if result.SimilarityPercent > 60 {
						studentPlagiarismMap[regdNo].Status = "PLAGIARIZED"
					} else if result.SimilarityPercent >= 30 && studentPlagiarismMap[regdNo].Status != "PLAGIARIZED" {
						studentPlagiarismMap[regdNo].Status = "SUSPICIOUS"
					}
				}
			}
		}
	}

	// Get user names
	var regdNos []string
	for regdNo := range studentPlagiarismMap {
		regdNos = append(regdNos, regdNo)
	}

	var users []models.User
	database.DB.Where("regdno IN ?", regdNos).Find(&users)
	userNameMap := make(map[string]string)
	for _, u := range users {
		userNameMap[u.RegdNo] = u.Name
	}

	// Add all participants who didn't have plagiarism checks (no submissions or unique solutions)
	var participants []models.ContestParticipant
	database.DB.Where("contest_id = ?", contestID).Find(&participants)

	for _, p := range participants {
		if _, exists := studentPlagiarismMap[p.UserRegdNo]; !exists {
			studentPlagiarismMap[p.UserRegdNo] = &ContestPlagiarismStudent{
				UserRegdNo:             p.UserRegdNo,
				Name:                   userNameMap[p.UserRegdNo],
				MaxPlagiarismPercent:   0,
				ProblemsWithPlagiarism: 0,
				Status:                 "SAFE",
			}
		}
		studentPlagiarismMap[p.UserRegdNo].Name = userNameMap[p.UserRegdNo]
	}

	// Convert map to slice and sort by max plagiarism percent
	var students []ContestPlagiarismStudent
	for _, student := range studentPlagiarismMap {
		students = append(students, *student)
	}

	// Sort by max plagiarism percent descending
	for i := 0; i < len(students); i++ {
		for j := i + 1; j < len(students); j++ {
			if students[i].MaxPlagiarismPercent < students[j].MaxPlagiarismPercent {
				students[i], students[j] = students[j], students[i]
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"contest_id":        contestIDUint,
		"total_students":    len(students),
		"plagiarized_count": countByStatus(students, "PLAGIARIZED"),
		"suspicious_count":  countByStatus(students, "SUSPICIOUS"),
		"safe_count":        countByStatus(students, "SAFE"),
		"students":          students,
	})
}

// RunContestPlagiarismCheck forces a fresh plagiarism analysis
// Uses POST to prevent caching and indicate action
func RunContestPlagiarismCheck(c *gin.Context) {
	// Delegate to GetContestPlagiarism - it already runs fresh analysis each time
	GetContestPlagiarism(c)
}

func countByStatus(students []ContestPlagiarismStudent, status string) int {
	count := 0
	for _, s := range students {
		if s.Status == status {
			count++
		}
	}
	return count
}

// =====================================================
// Non-Joined Eligible Students Handler
// =====================================================

// NonJoinedStudent represents a student who is eligible but hasn't joined the contest
type NonJoinedStudent struct {
	UserRegdNo  string `json:"user_regdno"`
	Name        string `json:"name"`
	BranchName  string `json:"branch_name"`
	SectionName string `json:"section_name"`
	CohortYear  int    `json:"cohort_year"`
}

type ContestNonParticipantsSnapshotResponse struct {
	TargetCohort      *int               `json:"target_cohort,omitempty"`
	TargetBranchID    *uint              `json:"target_branch_id,omitempty"`
	TargetCohortYears []int              `json:"target_cohort_years,omitempty"`
	TargetBranchIDs   []uint             `json:"target_branch_ids,omitempty"`
	TargetBranchNames []string           `json:"target_branch_names,omitempty"`
	NonJoinedStudents []NonJoinedStudent `json:"non_joined_students"`
	ContestID         uint               `json:"contest_id"`
	TotalEligible     int64              `json:"total_eligible"`
	TotalJoined       int64              `json:"total_joined"`
	TotalNotJoined    int                `json:"total_not_joined"`
}

// GetContestNonParticipants returns eligible students who haven't joined the contest
// Accessible by faculty, HOD, admin, and college_admin
// Supports both new junction tables (multi-select) and deprecated single fields
func GetContestNonParticipants(c *gin.Context) {
	contestID := c.Param("id")
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check role - faculty, HOD, admin, or college_admin can access
	if user.Role != "faculty" && user.Role != "hod" && user.Role != "admin" && user.Role != "college_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only faculty, HOD, or admin can access this data"})
		return
	}

	// Get contest
	var contest models.Contest
	if err := database.DB.First(&contest, contestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Verify college access (strict isolation)
	if err := validateCollegeAccess(user.CollegeID, contest.CollegeID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	contestIDUint64, err := strconv.ParseUint(contestID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contest ID"})
		return
	}
	contestIDUint := uint(contestIDUint64)

	// For ended contests, return the stored snapshot if available.
	if isContestEnded(contest) {
		var snapshot models.ContestNonParticipantsSnapshot
		if err := database.DB.Where("contest_id = ?", contestIDUint).First(&snapshot).Error; err == nil {
			var stored ContestNonParticipantsSnapshotResponse
			if err := json.Unmarshal(snapshot.Data, &stored); err == nil {
				c.JSON(http.StatusOK, stored)
				return
			}
		}
	}

	// Load target cohorts from junction table
	var targetCohorts []models.ContestTargetCohort
	database.DB.Where("contest_id = ?", contestIDUint).Find(&targetCohorts)

	// Load target branches from junction table
	var targetBranches []models.ContestTargetBranch
	database.DB.Where("contest_id = ?", contestIDUint).Find(&targetBranches)

	// Extract cohort years and branch IDs
	var targetCohortYears []int
	var targetBranchIDs []uint
	var targetBranchNames []string

	for _, tc := range targetCohorts {
		targetCohortYears = append(targetCohortYears, tc.CohortYear)
	}
	for _, tb := range targetBranches {
		targetBranchIDs = append(targetBranchIDs, tb.BranchID)
		// Get branch name
		var branch models.Branch
		if err := database.DB.First(&branch, tb.BranchID).Error; err == nil {
			targetBranchNames = append(targetBranchNames, branch.BranchName)
		}
	}

	// Fallback to deprecated single fields if junction tables are empty (backward compatibility)
	if len(targetCohortYears) == 0 && contest.TargetCohort != nil {
		targetCohortYears = []int{*contest.TargetCohort}
	}
	if len(targetBranchIDs) == 0 && contest.TargetBranchID != nil {
		targetBranchIDs = []uint{*contest.TargetBranchID}
		// Get branch name for deprecated field
		var branch models.Branch
		if err := database.DB.First(&branch, *contest.TargetBranchID).Error; err == nil {
			targetBranchNames = []string{branch.BranchName}
		}
	}

	// If no targeting specified, return empty (open contest - can't determine eligibility)
	if len(targetCohortYears) == 0 {
		payload := ContestNonParticipantsSnapshotResponse{
			ContestID:         contestIDUint,
			TargetCohortYears: targetCohortYears,
			TargetBranchIDs:   targetBranchIDs,
			TargetBranchNames: targetBranchNames,
			TargetCohort:      contest.TargetCohort,
			TargetBranchID:    contest.TargetBranchID,
			TotalEligible:     0,
			TotalJoined:       0,
			TotalNotJoined:    0,
			NonJoinedStudents: []NonJoinedStudent{},
		}
		saveNonParticipantsSnapshot(contestIDUint, payload, contest)
		c.JSON(http.StatusOK, payload)
		return
	}

	// Build eligibility query using junction tables
	// Students must match ANY of the target cohorts AND ANY of the target branches (if branches are specified)
	var eligibleStudents []NonJoinedStudent
	var totalEligible int64

	// Build cohort filter - match ANY of the target cohorts
	cohortFilter := " AND st.cohort_year IN (?)"

	// Build branch filter - match ANY of the target branches (if specified)
	branchFilter := ""
	if len(targetBranchIDs) > 0 {
		branchFilter = " AND st.branch_id IN (?)"
	}

	// Query for non-joined eligible students
	// Note: Students must match cohort AND (if branches specified) branch
	nonJoinedQuery := `
		SELECT
			u.regdno as user_regd_no,
			u.name,
			COALESCE(b.branch_name, '') as branch_name,
			COALESCE(s.section_name, '') as section_name,
			st.cohort_year
		FROM students st
		INNER JOIN users u ON u.regdno = st.regd_no
		LEFT JOIN branches b ON b.branch_id = st.branch_id
		LEFT JOIN sections s ON s.section_id = st.section_id
		WHERE u.college_id = ?
		` + cohortFilter + branchFilter + `
		AND u.regdno NOT IN (SELECT user_regd_no FROM contest_participants WHERE contest_id = ?)
		ORDER BY u.regdno
	`

	// Build args for query
	args := []interface{}{*contest.CollegeID, targetCohortYears}
	if len(targetBranchIDs) > 0 {
		args = append(args, targetBranchIDs)
	}
	args = append(args, contestIDUint)

	database.DB.Raw(nonJoinedQuery, args...).Scan(&eligibleStudents)

	// Get total eligible count (all students matching the criteria)
	countQuery := `
		SELECT COUNT(DISTINCT u.regdno)
		FROM students st
		INNER JOIN users u ON u.regdno = st.regd_no
		WHERE u.college_id = ?
		AND st.cohort_year IN (?)
	`
	countArgs := []interface{}{*contest.CollegeID, targetCohortYears}

	if len(targetBranchIDs) > 0 {
		countQuery += " AND st.branch_id IN (?)"
		countArgs = append(countArgs, targetBranchIDs)
	}

	database.DB.Raw(countQuery, countArgs...).Count(&totalEligible)

	// Get joined count
	var joinedCount int64
	database.DB.Model(&models.ContestParticipant{}).
		Where("contest_id = ?", contestIDUint).
		Count(&joinedCount)

	payload := ContestNonParticipantsSnapshotResponse{
		ContestID:         contestIDUint,
		TargetCohortYears: targetCohortYears,
		TargetBranchIDs:   targetBranchIDs,
		TargetBranchNames: targetBranchNames,
		TargetCohort:      contest.TargetCohort,
		TargetBranchID:    contest.TargetBranchID,
		TotalEligible:     totalEligible,
		TotalJoined:       joinedCount,
		TotalNotJoined:    len(eligibleStudents),
		NonJoinedStudents: eligibleStudents,
	}

	saveNonParticipantsSnapshot(contestIDUint, payload, contest)

	c.JSON(http.StatusOK, payload)
}

// Helper to save snapshot for ended contests
func saveNonParticipantsSnapshot(contestIDUint uint, payload ContestNonParticipantsSnapshotResponse, contest models.Contest) {
	if !isContestEnded(contest) {
		return
	}
	var existing models.ContestNonParticipantsSnapshot
	snapErr := database.DB.Where("contest_id = ?", contestIDUint).First(&existing).Error
	if errors.Is(snapErr, gorm.ErrRecordNotFound) {
		if data, marshalErr := json.Marshal(payload); marshalErr == nil {
			snapshot := models.ContestNonParticipantsSnapshot{
				ContestID: contestIDUint,
				Data:      data,
			}
			_ = database.DB.Create(&snapshot).Error
		}
	}
}
