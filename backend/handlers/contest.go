package handlers

import (
	"coding-platform/config"
	"coding-platform/database"
	"coding-platform/models"
	"coding-platform/services"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// =====================================================
// Request/Response Structures
// =====================================================

type CreateContestRequest struct {
	Title           string `json:"title" binding:"required,min=3,max=200"`
	Description     string `json:"description"`
	StartTime       string `json:"start_time" binding:"required"` // ISO 8601 format
	EndTime         string `json:"end_time" binding:"required"`   // ISO 8601 format
	TargetCohorts   []int  `json:"target_cohorts"`                // Multiple batch targeting (new)
	TargetBranchIDs []uint `json:"target_branch_ids"`             // Multiple branch targeting (new)
	// Deprecated single fields (kept for backward compatibility)
	TargetCohort   *int   `json:"target_cohort,omitempty"`    // Deprecated: Use TargetCohorts
	TargetBranchID *uint  `json:"target_branch_id,omitempty"` // Deprecated: Use TargetBranchIDs
	ProblemIDs     []uint `json:"problem_ids"`                // Optional at creation; problems can be added later
	Points         []int  `json:"points"`                     // Must match problem_ids length if provided
	// Quiz options
	HasQuiz             bool `json:"has_quiz"`
	QuizDurationMinutes int  `json:"quiz_duration_minutes"`
}

type ContestResponse struct {
	StartTime              time.Time                `json:"start_time"`
	CreatedAt              time.Time                `json:"created_at"`
	UpdatedAt              time.Time                `json:"updated_at"`
	EndTime                time.Time                `json:"end_time"`
	TargetCohort           *int                     `json:"target_cohort,omitempty"`
	DisqualificationReason *string                  `json:"disqualification_reason,omitempty"`
	College                *models.College          `json:"college,omitempty"`
	TargetBranchName       *string                  `json:"target_branch_name,omitempty"`
	TargetBranchID         *uint                    `json:"target_branch_id,omitempty"`
	PracticeStartTime      *time.Time               `json:"practice_start_time,omitempty"`
	PracticeEndTime        *time.Time               `json:"practice_end_time,omitempty"`
	Description            string                   `json:"description"`
	CreatedBy              string                   `json:"created_by"`
	Title                  string                   `json:"title"`
	CollegeID              string                   `json:"college_id"`
	TargetBranchNames      []string                 `json:"target_branch_names,omitempty"`
	TargetCohortYears      []int                    `json:"target_cohort_years,omitempty"`
	Problems               []ContestProblemResponse `json:"problems,omitempty"`
	TargetBranchIDs        []uint                   `json:"target_branch_ids,omitempty"`
	Sections               []models.ContestSection  `json:"sections,omitempty"`
	ContestID              uint                     `json:"contest_id"`
	QuizDurationMinutes    int                      `json:"quiz_duration_minutes"`
	ProblemCount           int                      `json:"problem_count"`
	ParticipantCount       int                      `json:"participant_count"`
	IsActive               bool                     `json:"is_active"`
	IsFrozen               bool                     `json:"is_frozen"`
	PracticeEnabled        bool                     `json:"practice_enabled"`
	IsPracticeActive       bool                     `json:"is_practice_active"`
	HasJoined              bool                     `json:"has_joined"`
	HasFinished            bool                     `json:"has_finished"`
	HasDisqualified        bool                     `json:"has_disqualified"`
	HasQuiz                bool                     `json:"has_quiz"`
}

type ContestProblemResponse struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Difficulty   string `json:"difficulty"`
	ProblemID    uint   `json:"problem_id"`
	ContestID    uint   `json:"contest_id"`
	Points       int    `json:"points"`
	ProblemOrder int    `json:"problem_order"`
	IsSolved     bool   `json:"is_solved"`
	IsAttempted  bool   `json:"is_attempted"`
}

type ContestListResponse struct {
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	Title            string    `json:"title"`
	ContestID        uint      `json:"contest_id"`
	ParticipantCount int       `json:"participant_count"`
	ProblemCount     int       `json:"problem_count"`
	IsActive         bool      `json:"is_active"`
	IsFrozen         bool      `json:"is_frozen"`
}

type ContestLeaderboardEntry struct {
	UserRegdNo        string `json:"user_regdno"`
	Name              string `json:"name"`
	LastSubmitTime    string `json:"last_submit_time,omitempty"`
	Rank              int    `json:"rank"`
	TotalScore        int    `json:"total_score"`
	ProblemsSolved    int    `json:"problems_solved"`
	ProblemsAttempted int    `json:"problems_attempted"`
	BestTotalScore    int    `json:"best_total_score"`
}

// =====================================================
// Validation Helpers
// =====================================================

func validateContestTiming(startTime, endTime time.Time) error {
	now := time.Now()

	// Check if end time is in the past (contest already ended)
	if endTime.Before(now) {
		return fmt.Errorf("end time cannot be in the past")
	}

	// Check if end time is before start time
	if endTime.Before(startTime) {
		return fmt.Errorf("end time must be after start time")
	}

	// Check minimum duration (5 minutes)
	minDuration := 5 * time.Minute
	if endTime.Sub(startTime) < minDuration {
		return fmt.Errorf("contest duration must be at least %d minutes", int(minDuration.Minutes()))
	}

	return nil
}

func validateCollegeAccess(userCollegeID *string, contestCollegeID *string) error {
	if userCollegeID == nil || contestCollegeID == nil {
		return fmt.Errorf("college information required")
	}

	if *userCollegeID != *contestCollegeID {
		return fmt.Errorf("access denied: college mismatch")
	}

	return nil
}

// =====================================================
// Contest Handlers (Admin Only)
// =====================================================

func CreateContest(c *gin.Context) {
	var req CreateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user info
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time format. Use ISO 8601 format"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_time format. Use ISO 8601 format"})
		return
	}

	// Validate timing
	if err := validateContestTiming(startTime, endTime); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate problem IDs / points length (only if problems provided)
	if len(req.ProblemIDs) > 0 && len(req.ProblemIDs) != len(req.Points) {
		// If points not provided, assign default 100 per problem
		if len(req.Points) == 0 {
			req.Points = make([]int, len(req.ProblemIDs))
			for i := range req.Points {
				req.Points[i] = 100
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Points array must match problem_ids length"})
			return
		}
	}

	// Get user's college
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Handle batch targeting - use new array format if provided, fallback to deprecated single field
	targetCohorts := req.TargetCohorts
	if len(targetCohorts) == 0 && req.TargetCohort != nil {
		targetCohorts = []int{*req.TargetCohort}
	}

	// Validate batch targeting - at least one batch is required
	if len(targetCohorts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target batch is required. Open-to-all contests are not allowed."})
		return
	}

	// Handle branch targeting - use new array format if provided, fallback to deprecated single field
	targetBranchIDs := req.TargetBranchIDs
	if len(targetBranchIDs) == 0 && req.TargetBranchID != nil {
		targetBranchIDs = []uint{*req.TargetBranchID}
	}

	// Validate all branches exist and belong to user's college (if branches specified)
	if len(targetBranchIDs) > 0 && user.CollegeID != nil {
		for _, branchID := range targetBranchIDs {
			var branch models.Branch
			if err := database.DB.Where("branch_id = ?", branchID).First(&branch).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid branch ID %d specified", branchID)})
				return
			}
			// Verify branch belongs to user's college
			if branch.CollegeID != *user.CollegeID {
				c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Branch %d does not belong to your college", branchID)})
				return
			}
		}
	}

	// Validate all problems belong to user's college (only if problems provided)
	if len(req.ProblemIDs) > 0 && user.CollegeID != nil {
		var problemCount int64
		database.DB.Model(&models.Problem{}).
			Where("college_id = ?", *user.CollegeID).
			Where("id IN ?", req.ProblemIDs).
			Count(&problemCount)

		if int(problemCount) != len(req.ProblemIDs) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Some problems don't exist or don't belong to your college"})
			return
		}
	}

	// Create contest within a transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// For backward compatibility, set single TargetCohort if only one batch is specified
	var legacyTargetCohort *int
	var legacyTargetBranchID *uint
	if len(targetCohorts) == 1 {
		legacyTargetCohort = &targetCohorts[0]
	}
	if len(targetBranchIDs) == 1 {
		legacyTargetBranchID = &targetBranchIDs[0]
	}

	contest := models.Contest{
		CollegeID:           user.CollegeID,
		Title:               req.Title,
		Description:         req.Description,
		StartTime:           startTime,
		EndTime:             endTime,
		TargetCohort:        legacyTargetCohort,   // Backward compatibility
		TargetBranchID:      legacyTargetBranchID, // Backward compatibility
		HasQuiz:             req.HasQuiz,
		QuizDurationMinutes: req.QuizDurationMinutes,
		CreatedBy:           userRegdNo.(string),
	}

	if err := tx.Create(&contest).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contest"})
		return
	}

	// Create target cohort junction table records
	for _, cohortYear := range targetCohorts {
		targetCohortRecord := models.ContestTargetCohort{
			ContestID:  contest.ContestID,
			CohortYear: cohortYear,
		}
		if err := tx.Create(&targetCohortRecord).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set target batches"})
			return
		}
	}

	// Create target branch junction table records
	for _, branchID := range targetBranchIDs {
		targetBranchRecord := models.ContestTargetBranch{
			ContestID: contest.ContestID,
			BranchID:  branchID,
		}
		if err := tx.Create(&targetBranchRecord).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set target branches"})
			return
		}
	}

	// Add problems to contest (if any provided)
	for i, problemID := range req.ProblemIDs {
		contestProblem := models.ContestProblem{
			ContestID:    contest.ContestID,
			ProblemID:    problemID,
			Points:       req.Points[i],
			ProblemOrder: i,
		}
		if err := tx.Create(&contestProblem).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add problems to contest"})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize contest creation"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"contest_id":        contest.ContestID,
		"message":           "Contest created successfully",
		"target_cohorts":    targetCohorts,
		"target_branch_ids": targetBranchIDs,
	})
}

func GetContests(c *gin.Context) {
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

	// Build query based on user role
	query := database.DB.Model(&models.Contest{})

	switch user.Role {
	case "college_admin", "admin", "faculty":
		// Admins and faculty see all contests for their college
		query = query.Where("college_id = ?", user.CollegeID)

	case "student":
		// Students can only see contests they're eligible for
		if user.CollegeID == nil {
			c.JSON(http.StatusOK, []ContestListResponse{})
			return
		}

		// Get student details
		var student models.Student
		if err := database.DB.Where("regd_no = ?", userRegdNo).First(&student).Error; err != nil {
			c.JSON(http.StatusOK, []ContestListResponse{})
			return
		}

		// Eligible contests:
		// Show ALL contests the student is eligible for, including ended ones
		// Eligibility based on: college match, cohort match (if specified), branch match (if specified)
		// This allows students to see and join contests even if admin extends time after they originally ended
		var contests []models.Contest
		if err := database.DB.Raw(`
			SELECT DISTINCT c.* FROM contests c
			WHERE c.college_id = ?
			  AND (
			    -- Check junction table for cohorts
			    EXISTS (SELECT 1 FROM contest_target_cohorts ct WHERE ct.contest_id = c.contest_id AND ct.cohort_year = ?)
			    OR
			    -- Fallback to legacy single cohort field (for backward compatibility)
			    (NOT EXISTS (SELECT 1 FROM contest_target_cohorts ct WHERE ct.contest_id = c.contest_id) AND (c.target_cohort IS NULL OR c.target_cohort = ?))
			  )
			  AND (
			    -- No branch restriction (junction table empty)
			    NOT EXISTS (SELECT 1 FROM contest_target_branches cb WHERE cb.contest_id = c.contest_id)
			    OR
			    -- Check junction table for branches
			    EXISTS (SELECT 1 FROM contest_target_branches cb WHERE cb.contest_id = c.contest_id AND cb.branch_id = ?)
			  )
			ORDER BY c.start_time DESC
		`, *user.CollegeID, student.CohortYear, student.CohortYear, student.BranchID).Scan(&contests).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load contests"})
			return
		}

		// Fetch all counts in a single query (fix N+1)
		countsMap := getContestCounts(contests)

		// Map to response
		response := make([]ContestListResponse, 0, len(contests))
		for _, c := range contests {
			counts := countsMap[c.ContestID]
			response = append(response, ContestListResponse{
				ContestID:        c.ContestID,
				Title:            c.Title,
				StartTime:        c.StartTime,
				EndTime:          c.EndTime,
				ParticipantCount: counts.Participants,
				ProblemCount:     counts.Problems,
				IsActive:         isContestActive(c),
				IsFrozen:         isLeaderboardFrozen(c),
			})
		}

		c.JSON(http.StatusOK, response)
		return

	default:
		c.JSON(http.StatusOK, []ContestListResponse{})
		return
	}

	// For admins - get all contests for their college
	var contests []models.Contest
	if err := query.Order("created_at DESC").Find(&contests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load contests"})
		return
	}

	// Fetch all counts in a single query (fix N+1)
	countsMap := getContestCounts(contests)

	response := make([]ContestListResponse, 0, len(contests))
	for _, c := range contests {
		counts := countsMap[c.ContestID]
		response = append(response, ContestListResponse{
			ContestID:        c.ContestID,
			Title:            c.Title,
			StartTime:        c.StartTime,
			EndTime:          c.EndTime,
			ParticipantCount: counts.Participants,
			ProblemCount:     counts.Problems,
			IsActive:         isContestActive(c),
			IsFrozen:         isLeaderboardFrozen(c),
		})
	}

	c.JSON(http.StatusOK, response)
}

// contestCounts holds participant and problem counts for a contest
type contestCounts struct {
	Participants int
	Problems     int
}

// getContestCounts fetches participant and problem counts for all contests in a single pass
// This fixes the N+1 query problem where each contest required 2 additional queries
func getContestCounts(contests []models.Contest) map[uint]contestCounts {
	result := make(map[uint]contestCounts)

	if len(contests) == 0 {
		return result
	}

	// Extract contest IDs
	contestIDs := make([]uint, len(contests))
	for i, c := range contests {
		contestIDs[i] = c.ContestID
	}

	// Initialize all contests with 0 counts
	for _, id := range contestIDs {
		result[id] = contestCounts{}
	}

	// Single query for participant counts
	type participantCountRow struct {
		ContestID uint `gorm:"column:contest_id"`
		Count     int  `gorm:"column:count"`
	}
	var participantCounts []participantCountRow
	database.DB.Model(&models.ContestParticipant{}).
		Select("contest_id, COUNT(*) as count").
		Where("contest_id IN ?", contestIDs).
		Group("contest_id").
		Scan(&participantCounts)

	for _, row := range participantCounts {
		if counts, ok := result[row.ContestID]; ok {
			counts.Participants = row.Count
			result[row.ContestID] = counts
		}
	}

	// Single query for problem counts
	type problemCountRow struct {
		ContestID uint `gorm:"column:contest_id"`
		Count     int  `gorm:"column:count"`
	}
	var problemCounts []problemCountRow
	database.DB.Model(&models.ContestProblem{}).
		Select("contest_id, COUNT(*) as count").
		Where("contest_id IN ?", contestIDs).
		Group("contest_id").
		Scan(&problemCounts)

	for _, row := range problemCounts {
		if counts, ok := result[row.ContestID]; ok {
			counts.Problems = row.Count
			result[row.ContestID] = counts
		}
	}

	return result
}

func GetContestDetails(c *gin.Context) {
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

	// Get problems for this contest
	var contestProblems []models.ContestProblem
	if err := database.DB.Preload("Problem").Where("contest_id = ?", contestID).Order("problem_order").Find(&contestProblems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load contest problems"})
		return
	}

	// Get sections with their problems and quizzes
	var sections []models.ContestSection
	database.DB.Where("contest_id = ?", contestID).Order("order_index ASC").
		Preload("ContestProblems.Problem").
		Preload("ContestQuiz.Questions.Options").
		Find(&sections)

	// Check if current user has joined the contest
	var participantCount int64
	database.DB.Model(&models.ContestParticipant{}).
		Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).
		Count(&participantCount)

	// Get participant details (including has_finished)
	var participant models.ContestParticipant
	hasFinished := false
	hasDisqualified := false
	var disqualificationReason *string
	if participantCount > 0 {
		database.DB.Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).First(&participant)
		hasFinished = participant.HasFinished
		hasDisqualified = participant.Disqualified
		disqualificationReason = participant.DisqualificationReason
	}

	// Build response
	response := ContestResponse{
		ContestID:      contest.ContestID,
		CollegeID:      *contest.CollegeID,
		Title:          contest.Title,
		Description:    contest.Description,
		StartTime:      contest.StartTime,
		EndTime:        contest.EndTime,
		TargetCohort:   contest.TargetCohort,
		TargetBranchID: contest.TargetBranchID,
		// Practice mode fields
		PracticeEnabled:   contest.PracticeEnabled,
		PracticeStartTime: contest.PracticeStartTime,
		PracticeEndTime:   contest.PracticeEndTime,
		IsPracticeActive:  isPracticeActive(&contest),
		// Quiz fields
		HasQuiz:                contest.HasQuiz,
		QuizDurationMinutes:    contest.QuizDurationMinutes,
		CreatedBy:              contest.CreatedBy,
		CreatedAt:              contest.CreatedAt,
		UpdatedAt:              contest.UpdatedAt,
		IsActive:               isContestActive(contest),
		IsFrozen:               isLeaderboardFrozen(contest),
		ProblemCount:           len(contestProblems),
		ParticipantCount:       getParticipantCount(contest.ContestID),
		HasJoined:              participantCount > 0,
		HasFinished:            hasFinished,
		HasDisqualified:        hasDisqualified,
		DisqualificationReason: disqualificationReason,
		Sections:               sections,
	}

	// Load target cohorts from junction table
	var targetCohorts []models.ContestTargetCohort
	database.DB.Where("contest_id = ?", contest.ContestID).Find(&targetCohorts)
	response.TargetCohortYears = make([]int, 0, len(targetCohorts))
	for _, tc := range targetCohorts {
		response.TargetCohortYears = append(response.TargetCohortYears, tc.CohortYear)
	}

	// Load target branches from junction table
	var targetBranches []models.ContestTargetBranch
	database.DB.Where("contest_id = ?", contest.ContestID).Find(&targetBranches)
	response.TargetBranchIDs = make([]uint, 0, len(targetBranches))
	response.TargetBranchNames = make([]string, 0, len(targetBranches))
	for _, tb := range targetBranches {
		response.TargetBranchIDs = append(response.TargetBranchIDs, tb.BranchID)
		// Load branch name
		var branch models.Branch
		if err := database.DB.First(&branch, tb.BranchID).Error; err == nil {
			response.TargetBranchNames = append(response.TargetBranchNames, branch.BranchName)
		}
	}

	// Load branch name if applicable (legacy field)
	if contest.TargetBranchID != nil {
		var branch models.Branch
		if err := database.DB.First(&branch, *contest.TargetBranchID).Error; err == nil {
			response.TargetBranchName = &branch.BranchName
		}
	}

	// Get solved problem IDs for this user in this contest
	solvedProblemIDs := make(map[uint]bool)
	attemptedProblemIDs := make(map[uint]bool)
	if participantCount > 0 {
		var solvedIDs []uint
		database.DB.Model(&models.ContestSubmission{}).
			Where("contest_id = ? AND user_regd_no = ? AND passed = ?", contestID, userRegdNo, true).
			Distinct("problem_id").
			Pluck("problem_id", &solvedIDs)
		for _, pid := range solvedIDs {
			solvedProblemIDs[pid] = true
		}

		// Get attempted (but not solved) problem IDs
		var attemptedIDs []uint
		database.DB.Model(&models.ContestSubmission{}).
			Where("contest_id = ? AND user_regd_no = ? AND passed = ?", contestID, userRegdNo, false).
			Distinct("problem_id").
			Pluck("problem_id", &attemptedIDs)
		for _, pid := range attemptedIDs {
			if !solvedProblemIDs[pid] {
				attemptedProblemIDs[pid] = true
			}
		}
	}

	// Load problems
	for _, cp := range contestProblems {
		response.Problems = append(response.Problems, ContestProblemResponse{
			ProblemID:    cp.ProblemID,
			ContestID:    cp.ContestID,
			Title:        cp.Problem.Title,
			Description:  cp.Problem.Description,
			Difficulty:   cp.Problem.Difficulty,
			Points:       cp.Points,
			ProblemOrder: cp.ProblemOrder,
			IsSolved:     solvedProblemIDs[cp.ProblemID],
			IsAttempted:  attemptedProblemIDs[cp.ProblemID],
		})
	}

	c.JSON(http.StatusOK, response)
}

// =====================================================
// Contest Participation (Students)
// =====================================================

func JoinContest(c *gin.Context) {
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

	// Check eligibility for students
	if user.Role == "student" {
		if !isStudentEligibleForContest(userRegdNo.(string), contest) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not eligible for this contest"})
			return
		}
	}

	// Check if already joined
	var existingParticipation models.ContestParticipant
	if err := database.DB.Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).First(&existingParticipation).Error; err == nil {
		if existingParticipation.Disqualified {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are disqualified from this contest"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Already registered"})
		return
	}

	// Check if contest is still accepting participants
	if isContestEnded(contest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot join ended contest"})
		return
	}

	// Create participation record
	participation := models.ContestParticipant{
		ContestID:  contest.ContestID,
		UserRegdNo: userRegdNo.(string),
	}

	// Store eligibility snapshot for students
	if user.Role == "student" {
		var student models.Student
		if err := database.DB.Where("regd_no = ?", userRegdNo).First(&student).Error; err == nil {
			snapshot := fmt.Sprintf(`{"cohort":%d,"branch_id":%d}`, student.CohortYear, student.BranchID)
			participation.EligibilitySnapshot = &snapshot
		}
	}

	if err := database.DB.Create(&participation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully joined contest"})
}

func GetContestLeaderboard(c *gin.Context) {
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

	// For ended contests, return the stored snapshot if available.
	if isContestEnded(contest) {
		var snapshot models.ContestLeaderboardSnapshot
		if err := database.DB.Where("contest_id = ?", contest.ContestID).First(&snapshot).Error; err == nil {
			var stored []ContestLeaderboardEntry
			if err := json.Unmarshal(snapshot.Data, &stored); err == nil {
				c.JSON(http.StatusOK, stored)
				return
			}
		}
	}

	// If frozen, return frozen leaderboard snapshot
	if isLeaderboardFrozen(contest) {
		contestIDUint, _ := strconv.ParseUint(contestID, 10, 32)
		returnFrozenLeaderboard(c, uint(contestIDUint))
		return
	}

	// Get leaderboard - Show ALL participants with any submissions
	// Score calculation:
	// - total_score: sum of max score per problem (only fully passing submissions)
	// - problems_solved: count of problems with passing submissions
	// - problems_attempted: count of problems with any submission
	// - best_partial_score: sum of best partial scores for non-passing attempts
	query := database.DB.Raw(`
		SELECT
			cp.user_regd_no,
			u.name,
			COALESCE(SUM(passing.max_score), 0) as total_score,
			COUNT(DISTINCT passing.problem_id) as problems_solved,
			COUNT(DISTINCT all_subs.problem_id) as problems_attempted,
			COALESCE(SUM(all_subs.best_score), 0) as best_total_score,
			MAX(all_subs.last_submit_time) as last_submit_time
		FROM contest_participants cp
		INNER JOIN users u ON u.regdno = cp.user_regd_no
		-- Get max score per user per problem for PASSING submissions only
		LEFT JOIN (
			SELECT contest_id, user_regd_no, problem_id, MAX(score) as max_score
			FROM contest_submissions
			WHERE passed = true AND contest_id = ? AND is_practice = false
			GROUP BY contest_id, user_regd_no, problem_id
		) passing ON passing.contest_id = cp.contest_id AND passing.user_regd_no = cp.user_regd_no
		-- Get best score per user per problem for ALL submissions (including partial)
		LEFT JOIN (
			SELECT
				contest_id,
				user_regd_no,
				problem_id,
				MAX(score) as best_score,
				MAX(submitted_at) as last_submit_time
			FROM contest_submissions
			WHERE contest_id = ? AND is_practice = false
			GROUP BY contest_id, user_regd_no, problem_id
		) all_subs ON all_subs.contest_id = cp.contest_id AND all_subs.user_regd_no = cp.user_regd_no
		WHERE cp.contest_id = ?
			AND u.college_id = (SELECT college_id FROM contests WHERE contest_id = ?)
		GROUP BY cp.user_regd_no, u.name
		HAVING COUNT(DISTINCT all_subs.problem_id) > 0
		ORDER BY total_score DESC, problems_solved DESC, best_total_score DESC, last_submit_time ASC
		LIMIT 100
	`, contestID, contestID, contestID, contestID)

	type LeaderboardRow struct {
		LastSubmitTime    time.Time `json:"last_submit_time"`
		UserRegdNo        string    `json:"user_regdno"`
		Name              string    `json:"name"`
		TotalScore        int       `json:"total_score"`
		ProblemsSolved    int       `json:"problems_solved"`
		ProblemsAttempted int       `json:"problems_attempted"`
		BestTotalScore    int       `json:"best_total_score"`
	}

	var rows []LeaderboardRow
	if err := query.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}

	// Build leaderboard with ranks
	var leaderboard []ContestLeaderboardEntry
	for i, row := range rows {
		entry := ContestLeaderboardEntry{
			Rank:              i + 1,
			UserRegdNo:        row.UserRegdNo,
			Name:              row.Name,
			TotalScore:        row.TotalScore,
			ProblemsSolved:    row.ProblemsSolved,
			ProblemsAttempted: row.ProblemsAttempted,
			BestTotalScore:    row.BestTotalScore,
		}
		if !row.LastSubmitTime.IsZero() {
			entry.LastSubmitTime = row.LastSubmitTime.Format("2006-01-02T15:04:05Z")
		}
		leaderboard = append(leaderboard, entry)
	}

	// Persist a snapshot for ended contests to keep results stable after completion.
	if isContestEnded(contest) {
		var existing models.ContestLeaderboardSnapshot
		err := database.DB.Where("contest_id = ?", contest.ContestID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if data, marshalErr := json.Marshal(leaderboard); marshalErr == nil {
				snapshot := models.ContestLeaderboardSnapshot{
					ContestID: contest.ContestID,
					Data:      data,
				}
				_ = database.DB.Create(&snapshot).Error
			}
		}
	}

	c.JSON(http.StatusOK, leaderboard)
}

func returnFrozenLeaderboard(c *gin.Context, contestID uint) {
	// Get contest
	var contest models.Contest
	if err := database.DB.First(&contest, contestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}
	// Since freeze feature was removed, use contest end time as cutoff
	freezeTime := contest.EndTime

	// Query submissions as of freeze time - use max score per problem
	query := database.DB.Raw(`
		SELECT
			cp.user_regd_no,
			u.name,
			COALESCE(SUM(passing.max_score), 0) as total_score,
			COUNT(DISTINCT passing.problem_id) as problems_solved,
			COUNT(DISTINCT all_subs.problem_id) as problems_attempted,
			COALESCE(SUM(all_subs.best_score), 0) as best_total_score
		FROM contest_participants cp
		INNER JOIN users u ON u.regdno = cp.user_regd_no
		-- Get max score per user per problem for PASSING submissions before freeze
		LEFT JOIN (
			SELECT contest_id, user_regd_no, problem_id, MAX(score) as max_score
			FROM contest_submissions
			WHERE passed = true AND submitted_at <= ? AND is_practice = false
			GROUP BY contest_id, user_regd_no, problem_id
		) passing ON passing.contest_id = cp.contest_id AND passing.user_regd_no = cp.user_regd_no
		-- Get best score per user per problem for ALL submissions before freeze
		LEFT JOIN (
			SELECT contest_id, user_regd_no, problem_id, MAX(score) as best_score
			FROM contest_submissions
			WHERE submitted_at <= ? AND is_practice = false
			GROUP BY contest_id, user_regd_no, problem_id
		) all_subs ON all_subs.contest_id = cp.contest_id AND all_subs.user_regd_no = cp.user_regd_no
		WHERE cp.contest_id = ?
			AND u.college_id = (SELECT college_id FROM contests WHERE contest_id = ?)
		GROUP BY cp.user_regd_no, u.name
		HAVING COUNT(DISTINCT all_subs.problem_id) > 0
		ORDER BY total_score DESC, problems_solved DESC, best_total_score DESC
		LIMIT 100
	`, freezeTime, freezeTime, contestID, contestID)

	type FrozenLeaderboardRow struct {
		UserRegdNo        string `json:"user_regdno"`
		Name              string `json:"name"`
		TotalScore        int    `json:"total_score"`
		ProblemsSolved    int    `json:"problems_solved"`
		ProblemsAttempted int    `json:"problems_attempted"`
		BestTotalScore    int    `json:"best_total_score"`
	}

	var rows []FrozenLeaderboardRow
	if err := query.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}

	// Build leaderboard
	var leaderboard []ContestLeaderboardEntry
	for i, row := range rows {
		entry := ContestLeaderboardEntry{
			Rank:              i + 1,
			UserRegdNo:        row.UserRegdNo,
			Name:              row.Name,
			TotalScore:        row.TotalScore,
			ProblemsSolved:    row.ProblemsSolved,
			ProblemsAttempted: row.ProblemsAttempted,
			BestTotalScore:    row.BestTotalScore,
		}
		leaderboard = append(leaderboard, entry)
	}

	c.JSON(http.StatusOK, leaderboard)
}

// =====================================================
// Contest Submission
// =====================================================

func SubmitContestSolution(c *gin.Context) {
	contestID := c.Param("id")
	problemID := c.Param("problemId")

	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		SourceCode string `json:"source_code" binding:"required"`
		LanguageID int    `json:"language_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate code size to prevent memory issues
	const maxCodeSize = 100 * 1024 // 100KB
	if len(req.SourceCode) > maxCodeSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Code size exceeds limit. Maximum allowed: %d KB", maxCodeSize/1024),
		})
		return
	}

	// Parse contest and problem IDs
	contestIDUint, err1 := strconv.ParseUint(contestID, 10, 32)
	problemIDUint, err2 := strconv.ParseUint(problemID, 10, 32)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contest or problem ID"})
		return
	}

	// Note: Rate limiting is handled by middleware.ContestRateLimitMiddleware()
	// which limits to 5 submissions per minute per user

	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
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

	// Check if contest is active or in practice mode
	isActive := isContestActive(contest)
	isPractice := isPracticeActive(&contest)
	if !isActive && !isPractice {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Contest is not active and practice mode is not enabled"})
		return
	}

	// Check eligibility for students
	if user.Role == "student" {
		if !isStudentEligibleForContest(userRegdNo.(string), contest) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not eligible for this contest"})
			return
		}
	}

	// Check if user is registered. In practice mode, auto-register eligible students.
	var participation models.ContestParticipant
	if err := database.DB.Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).First(&participation).Error; err != nil {
		if isPractice && user.Role == "student" {
			participation = models.ContestParticipant{
				ContestID:  uint(contestIDUint),
				UserRegdNo: userRegdNo.(string),
			}

			var student models.Student
			if err := database.DB.Where("regd_no = ?", userRegdNo).First(&student).Error; err == nil {
				snapshot := fmt.Sprintf(`{"cohort":%d,"branch_id":%d}`, student.CohortYear, student.BranchID)
				participation.EligibilitySnapshot = &snapshot
			}

			if err := database.DB.Create(&participation).Error; err != nil {
				// Handle race where another request created the participant first.
				if err := database.DB.Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).First(&participation).Error; err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register for practice mode"})
					return
				}
			}
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "You must join the contest first"})
			return
		}
	}

	// Check if user is disqualified
	if participation.Disqualified {
		c.JSON(http.StatusForbidden, gin.H{
			"error":       "You have been disqualified from this contest.",
			"redirect_to": "contests",
		})
		return
	}

	// Finished users can still submit in practice mode.
	if participation.HasFinished && !isPractice {
		c.JSON(http.StatusForbidden, gin.H{"error": "You have already finished this contest. No further submissions allowed."})
		return
	}

	// Get problem details
	var problem models.Problem
	if err := database.DB.Where("id = ?", problemID).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	// Verify problem belongs to this contest
	var contestProblem models.ContestProblem
	if err := database.DB.Where("contest_id = ? AND problem_id = ?", contestID, problemID).First(&contestProblem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found in this contest"})
		return
	}

	// Prevent re-submission if the problem is already fully solved (except during practice mode)
	// Practice mode allows students to re-solve problems for learning
	if !isPractice {
		var solvedCount int64
		database.DB.Model(&models.ContestSubmission{}).
			Where("contest_id = ? AND problem_id = ? AND user_regd_no = ? AND passed = ?", contestID, problemID, userRegdNo, true).
			Count(&solvedCount)
		if solvedCount > 0 {
			c.JSON(http.StatusForbidden, gin.H{
				"error":       "Problem already solved. Further submissions are not allowed.",
				"redirect_to": "contest_problems",
			})
			return
		}
	}

	// Get all test cases for this problem
	var testCases []models.TestCase
	if err := database.DB.Where("problem_id = ?", problemID).Find(&testCases).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch test cases"})
		return
	}

	if len(testCases) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No test cases found for this problem"})
		return
	}

	// Check if job queue is enabled - use queue for burst handling
	// Contest submissions get HIGH priority for faster processing
	if config.AppConfig.JobQueueEnabled && config.AppConfig.RedisEnabled {
		// Convert test cases to queue format
		queueTestCases := make([]services.TestCaseData, len(testCases))
		for i, tc := range testCases {
			queueTestCases[i] = services.TestCaseData{
				Input:          tc.Input,
				ExpectedOutput: tc.ExpectedOutput,
				TimeLimitMs:    problem.TimeLimit,
				MemoryLimitKB:  problem.MemoryLimit,
				Points:         tc.Points,
				IsSample:       tc.IsSample,
			}
		}

		// Create job with HIGH priority (contest submission)
		contID := uint(contestIDUint)
		job := services.NewJob(

			services.JobTypeSubmit,

			services.PriorityHigh, // Highest priority for contests
			req.SourceCode,

			req.LanguageID,

			queueTestCases,

			services.ConvertModelsProblem(problem),

			userRegdNo.(string),

			*user.CollegeID,

			&contID, // Contest ID for tracking
		)

		jobID := services.EnqueueJob(job)

		c.JSON(http.StatusAccepted, gin.H{

			"job_id": jobID,

			"status": "pending",

			"message": "Contest submission queued for processing",

			"poll_url": fmt.Sprintf("/api/jobs/%s", jobID),
		})

		return
	}

	// Fallback: Direct execution when job queue is disabled
	// Execute code against test cases using shared executor
	execResult := ExecuteCodeAgainstTestCases(req.SourceCode, req.LanguageID, problem, testCases)
	if execResult.ErrorMessage != "" {
		// Log detailed error internally
		log.Printf("[ContestSubmit] Code execution failed: %v", execResult.ErrorMessage)
		// Return safe error to user
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to execute code. Please try again later.",
		})
		return
	}

	// Calculate attempt number for this user/problem/contest
	var attemptCount int64
	database.DB.Model(&models.ContestSubmission{}).
		Where("contest_id = ? AND problem_id = ? AND user_regd_no = ?", contestID, problemID, userRegdNo).
		Count(&attemptCount)

	// Get problem points from ContestProblem table (authoritative score for this problem)
	problemPoints := contestProblem.Points

	// Calculate penalty only during the official contest window.
	penaltyMinutes := 0
	if !isPractice {
		penaltyMinutes = max(0, int(time.Since(contest.StartTime).Minutes()))
	}

	// Use a transaction for atomic update to prevent race conditions
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Lock participant row to serialize score updates for this user in this contest.
	// Without this lock, concurrent accepted submissions can both pass previous-success checks.
	var lockedParticipant models.ContestParticipant
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).
		First(&lockedParticipant).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock participant state"})
		return
	}

	// Check if there's a previous successful submission with row lock (FOR UPDATE)
	// This prevents race conditions when concurrent submissions both try to update scores
	var previousSuccess models.ContestSubmission
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("contest_id = ? AND problem_id = ? AND user_regd_no = ? AND passed = ?",
			contestID, problemID, userRegdNo, true).
		Order("attempt_number DESC").
		First(&previousSuccess).Error

	// Handle unexpected errors (not "record not found" which is expected)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check previous submissions"})
		return
	}

	if previousSuccess.ID != 0 && !isPractice {
		tx.Rollback()
		c.JSON(http.StatusForbidden, gin.H{
			"error":       "Problem already solved. Further submissions are not allowed.",
			"redirect_to": "contest_problems",
		})
		return
	}

	// Create contest submission record
	isFirstAC := execResult.AllPassed && previousSuccess.ID == 0

	// Calculate score: practice submissions are tracked but never score leaderboard points.
	submissionScore := 0
	if execResult.AllPassed && !isPractice {
		submissionScore = problemPoints
	}

	submission := models.ContestSubmission{
		ContestID:     uint(contestIDUint),
		ProblemID:     uint(problemIDUint),
		UserRegdNo:    userRegdNo.(string),
		CollegeID:     contest.CollegeID, // Direct college reference for efficient filtering
		LanguageID:    req.LanguageID,
		SourceCode:    req.SourceCode,
		Status:        "completed",
		Passed:        execResult.AllPassed,
		Score:         submissionScore, // 0 for failed, problemPoints for passed
		MaxScore:      problemPoints,   // Always show max possible points
		ExecutionTime: execResult.TotalTime,
		MemoryUsed:    execResult.MaxMemory,
		SubmittedAt:   time.Now(),
		AttemptNumber: int(attemptCount) + 1,
		IsFinal:       isFirstAC,      // First AC is final
		PenaltyTime:   penaltyMinutes, // ICPC-style penalty time
		IsPractice:    isPractice,     // Practice mode submission (doesn't count for scoring)
	}

	// Save submission
	if err := tx.Create(&submission).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save submission"})
		return
	}

	// Recompute participant totals from canonical submission history.
	// This self-heals any historical drift in contest_participants aggregates.
	var scoreSummary struct {
		TotalScore     int
		ProblemsSolved int
	}
	if err := tx.Raw(`
		SELECT
			COALESCE(SUM(solved.problem_score), 0) as total_score,
			COUNT(*) as problems_solved
		FROM (
			SELECT
				problem_id,
				MAX(score) as problem_score
			FROM contest_submissions
			WHERE contest_id = ? AND user_regd_no = ? AND passed = true AND is_practice = false
			GROUP BY problem_id
		) solved
	`, contestID, userRegdNo).Scan(&scoreSummary).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to recompute participant score"})
		return
	}

	if err := tx.Model(&models.ContestParticipant{}).
		Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).
		Updates(map[string]interface{}{
			"total_score":        scoreSummary.TotalScore,
			"problems_solved":    scoreSummary.ProblemsSolved,
			"last_submission_at": submission.SubmittedAt,
		}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update participant score"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Return response in same format as lab submissions for consistency
	response := gin.H{
		"submission_id": submission.ID,
		"all_passed":    execResult.AllPassed,
		"passed":        execResult.AllPassed,
		"total_tests":   execResult.TotalTests,
		"passed_tests":  execResult.PassedCount,
		"score":         submissionScore, // Awarded score for this submission
		"max_score":     problemPoints,
		"test_results":  execResult.Results,
		"attempt":       submission.AttemptNumber,
		"is_first_ac":   isFirstAC,
		"penalty_time":  penaltyMinutes,
		"is_practice":   isPractice,
	}
	if execResult.AllPassed {
		response["redirect_to"] = "contest_problems"
		response["problem_locked"] = true
	}

	c.JSON(http.StatusOK, response)
}

// =====================================================
// Contest Submission History (for Submissions tab)
// =====================================================

// GetContestProblemSubmissions returns the current user's submissions for a specific contest problem
func GetContestProblemSubmissions(c *gin.Context) {
	contestID := c.Param("id")
	problemID := c.Param("problemId")

	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse pagination
	page := 1
	limit := 20
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	offset := (page - 1) * limit

	// Check disqualification status
	var participation models.ContestParticipant
	if err := database.DB.Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).First(&participation).Error; err == nil {
		if participation.Disqualified {
			c.JSON(http.StatusForbidden, gin.H{
				"error":       "You have been disqualified from this contest.",
				"redirect_to": "contests",
			})
			return
		}
	}

	// Count total
	var total int64
	database.DB.Model(&models.ContestSubmission{}).
		Where("contest_id = ? AND problem_id = ? AND user_regd_no = ?", contestID, problemID, userRegdNo).
		Count(&total)

	// Fetch submissions (without source_code to keep payload small)
	var submissions []models.ContestSubmission
	database.DB.
		Select("id, contest_id, problem_id, user_regd_no, language_id, submitted_at, status, passed, score, max_score, execution_time, memory_used, attempt_number").
		Where("contest_id = ? AND problem_id = ? AND user_regd_no = ?", contestID, problemID, userRegdNo).
		Order("submitted_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&submissions)

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	})
}

// GetContestSubmissionCode returns the source code for a specific contest submission
func GetContestSubmissionCode(c *gin.Context) {
	submissionID := c.Param("submissionId")
	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var submission models.ContestSubmission
	if err := database.DB.
		Select("id, source_code, language_id, user_regd_no").
		Where("id = ?", submissionID).
		First(&submission).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Submission not found"})
		return
	}

	// Only allow own submissions (or admin)
	role, _ := c.Get("role")
	if role != "admin" && submission.UserRegdNo != userRegdNo.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot view other users' code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          submission.ID,
		"source_code": submission.SourceCode,
		"language_id": submission.LanguageID,
	})
}

// =====================================================
// Helper Functions
// =====================================================

func isContestActive(contest models.Contest) bool {
	now := time.Now()
	return now.After(contest.StartTime) && now.Before(contest.EndTime)
}

func isContestEnded(contest models.Contest) bool {
	return time.Now().After(contest.EndTime)
}

// isPracticeActive checks if practice mode is currently active for a contest
// Practice mode allows students to access and submit to ended contests for learning
func isPracticeActive(contest *models.Contest) bool {
	if !contest.PracticeEnabled {
		return false
	}
	now := time.Now()
	// If practice start time is set, check if we're past it
	if contest.PracticeStartTime != nil && now.Before(*contest.PracticeStartTime) {
		return false
	}
	// If practice end time is set, check if we're before it
	if contest.PracticeEndTime != nil && now.After(*contest.PracticeEndTime) {
		return false
	}
	return true
}

// isLeaderboardFrozen checks if the leaderboard is currently frozen
// Since freeze feature was removed, always returns false
func isLeaderboardFrozen(_ models.Contest) bool {
	return false
}

func isStudentEligibleForContest(userRegdNo string, contest models.Contest) bool {
	// Get student details
	var student models.Student
	if err := database.DB.Where("regd_no = ?", userRegdNo).First(&student).Error; err != nil {
		return false
	}

	// Check cohort requirement via junction table
	// First check if there are any target cohorts in junction table
	var targetCohorts []models.ContestTargetCohort
	database.DB.Where("contest_id = ?", contest.ContestID).Find(&targetCohorts)

	if len(targetCohorts) > 0 {
		// Use junction table for eligibility check
		cohortEligible := false
		for _, tc := range targetCohorts {
			if student.CohortYear == tc.CohortYear {
				cohortEligible = true
				break
			}
		}
		if !cohortEligible {
			return false
		}
	} else if contest.TargetCohort != nil {
		// Fallback to legacy single cohort field
		if student.CohortYear != *contest.TargetCohort {
			return false
		}
	}

	// Check branch requirement via junction table
	var targetBranches []models.ContestTargetBranch
	database.DB.Where("contest_id = ?", contest.ContestID).Find(&targetBranches)

	if len(targetBranches) > 0 {
		// Use junction table for eligibility check
		branchEligible := false
		for _, tb := range targetBranches {
			if student.BranchID == tb.BranchID {
				branchEligible = true
				break
			}
		}
		if !branchEligible {
			return false
		}
	} else if contest.TargetBranchID != nil {
		// Fallback to legacy single branch field
		if student.BranchID != *contest.TargetBranchID {
			return false
		}
	}

	return true
}

func getParticipantCount(contestID uint) int {
	var count int64
	database.DB.Model(&models.ContestParticipant{}).
		Where("contest_id = ?", contestID).
		Count(&count)
	return int(count)
}

// UpdateContest updates an existing contest (admin only)
func UpdateContest(c *gin.Context) {
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

	// Check if user is admin or college_admin
	if user.Role != "admin" && user.Role != "college_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can update contests"})
		return
	}

	var req CreateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_time format"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_time format"})
		return
	}

	// Handle batch targeting - use new array format if provided, fallback to deprecated single field
	targetCohorts := req.TargetCohorts
	if len(targetCohorts) == 0 && req.TargetCohort != nil {
		targetCohorts = []int{*req.TargetCohort}
	}

	// Validate batch targeting - at least one batch is required
	if len(targetCohorts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target batch is required. Open-to-all contests are not allowed."})
		return
	}

	// Handle branch targeting - use new array format if provided, fallback to deprecated single field
	targetBranchIDs := req.TargetBranchIDs
	if len(targetBranchIDs) == 0 && req.TargetBranchID != nil {
		targetBranchIDs = []uint{*req.TargetBranchID}
	}

	// Find existing contest
	var contest models.Contest
	if err := database.DB.Where("contest_id = ?", contestID).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Verify college access
	if err := validateCollegeAccess(user.CollegeID, contest.CollegeID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Use transaction for updating contest and junction tables
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// For backward compatibility, set single TargetCohort if only one batch is specified
	var legacyTargetCohort *int
	var legacyTargetBranchID *uint
	if len(targetCohorts) == 1 {
		legacyTargetCohort = &targetCohorts[0]
	}
	if len(targetBranchIDs) == 1 {
		legacyTargetBranchID = &targetBranchIDs[0]
	}

	// Update contest
	contest.Title = req.Title
	contest.Description = req.Description
	contest.StartTime = startTime
	contest.EndTime = endTime
	contest.TargetCohort = legacyTargetCohort     // Backward compatibility
	contest.TargetBranchID = legacyTargetBranchID // Backward compatibility
	contest.HasQuiz = req.HasQuiz
	contest.QuizDurationMinutes = req.QuizDurationMinutes

	if err := tx.Save(&contest).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contest"})
		return
	}

	// Delete existing target cohorts and create new ones
	if err := tx.Where("contest_id = ?", contest.ContestID).Delete(&models.ContestTargetCohort{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear target batches"})
		return
	}
	for _, cohortYear := range targetCohorts {
		targetCohortRecord := models.ContestTargetCohort{
			ContestID:  contest.ContestID,
			CohortYear: cohortYear,
		}
		if err := tx.Create(&targetCohortRecord).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set target batches"})
			return
		}
	}

	// Delete existing target branches and create new ones
	if err := tx.Where("contest_id = ?", contest.ContestID).Delete(&models.ContestTargetBranch{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear target branches"})
		return
	}
	for _, branchID := range targetBranchIDs {
		targetBranchRecord := models.ContestTargetBranch{
			ContestID: contest.ContestID,
			BranchID:  branchID,
		}
		if err := tx.Create(&targetBranchRecord).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set target branches"})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize contest update"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Contest updated successfully",
		"target_cohorts":    targetCohorts,
		"target_branch_ids": targetBranchIDs,
	})
}

// DeleteContest deletes a contest (admin only)
func DeleteContest(c *gin.Context) {
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

	// Check if user is admin or college_admin
	if user.Role != "admin" && user.Role != "college_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can delete contests"})
		return
	}

	// Find contest
	var contest models.Contest
	if err := database.DB.Where("contest_id = ?", contestID).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Verify college access
	if err := validateCollegeAccess(user.CollegeID, contest.CollegeID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Delete contest (cascade will handle related records)
	if err := database.DB.Delete(&contest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contest deleted successfully"})
}

// EnablePracticeModeRequest contains the request body for enabling practice mode
type EnablePracticeModeRequest struct {
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Enabled   bool       `json:"enabled"`
}

// EnablePracticeMode allows admin to enable/disable practice mode for ended contests
// Practice mode allows students to access contest problems and submit solutions after contest ends
func EnablePracticeMode(c *gin.Context) {
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

	// Check if user is admin or college_admin
	if user.Role != "admin" && user.Role != "college_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can enable practice mode"})
		return
	}

	// Find contest
	var contest models.Contest
	if err := database.DB.Where("contest_id = ?", contestID).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Verify college access
	if err := validateCollegeAccess(user.CollegeID, contest.CollegeID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Parse request
	var req EnablePracticeModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If enabling practice mode, verify contest has ended
	if req.Enabled {
		if !isContestEnded(contest) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Practice mode can only be enabled for ended contests"})
			return
		}
		// Set default start time to now if not provided
		if req.StartTime == nil {
			now := time.Now()
			req.StartTime = &now
		}
	}

	// Update contest
	contest.PracticeEnabled = req.Enabled
	contest.PracticeStartTime = req.StartTime
	contest.PracticeEndTime = req.EndTime

	if err := database.DB.Save(&contest).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "Practice mode updated successfully",
		"practice_enabled":    contest.PracticeEnabled,
		"practice_start_time": contest.PracticeStartTime,
		"practice_end_time":   contest.PracticeEndTime,
		"is_practice_active":  isPracticeActive(&contest),
	})
}

// AddProblemToContest adds a problem to a contest (admin only)
func AddProblemToContest(c *gin.Context) {
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

	// Check if user is admin or college_admin
	if user.Role != "admin" && user.Role != "college_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can add problems to contests"})
		return
	}

	var req struct {
		SectionID *uint `json:"section_id,omitempty"`
		ProblemID uint  `json:"problem_id" binding:"required"`
		Points    int   `json:"points"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find contest
	var contest models.Contest
	if err := database.DB.Where("contest_id = ?", contestID).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Verify college access
	if err := validateCollegeAccess(user.CollegeID, contest.CollegeID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Validate section if provided
	if req.SectionID != nil && *req.SectionID > 0 {
		var section models.ContestSection
		if err := database.DB.Where("id = ? AND contest_id = ?", *req.SectionID, contestID).First(&section).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid section for this contest"})
			return
		}
		if section.SectionType != "coding" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Problems can only be added to coding sections"})
			return
		}
	}

	// Check if problem exists and belongs to college
	var problem models.Problem
	if err := database.DB.Where("id = ?", req.ProblemID).First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	if user.CollegeID != nil && problem.CollegeID != nil && *problem.CollegeID != *user.CollegeID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Problem does not belong to your college"})
		return
	}

	// Check if problem already in contest (same section or unsectioned)
	var existing int64
	scope := database.DB.Model(&models.ContestProblem{}).Where("contest_id = ? AND problem_id = ?", contestID, req.ProblemID)
	if req.SectionID != nil && *req.SectionID > 0 {
		scope = scope.Where("section_id = ?", *req.SectionID)
	} else {
		scope = scope.Where("section_id IS NULL")
	}
	scope.Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Problem is already in this section"})
		return
	}

	// Get current max order within section
	var maxOrder int
	orderScope := database.DB.Model(&models.ContestProblem{}).Where("contest_id = ?", contestID)
	if req.SectionID != nil && *req.SectionID > 0 {
		orderScope = orderScope.Where("section_id = ?", *req.SectionID)
	} else {
		orderScope = orderScope.Where("section_id IS NULL")
	}
	orderScope.Select("COALESCE(MAX(problem_order), -1)").Scan(&maxOrder)

	// Add problem to contest
	contestProblem := models.ContestProblem{
		ContestID:    contest.ContestID,
		ProblemID:    req.ProblemID,
		Points:       req.Points,
		ProblemOrder: maxOrder + 1,
		SectionID:    req.SectionID,
	}

	if err := database.DB.Create(&contestProblem).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add problem to contest"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Problem added to contest successfully", "problem_order": contestProblem.ProblemOrder})
}

// RemoveContestProblem removes a problem from a contest (admin only)
func RemoveContestProblem(c *gin.Context) {
	contestID := c.Param("id")
	problemID := c.Param("problemId")

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

	if user.Role != "admin" && user.Role != "college_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only admins can remove problems from contests"})
		return
	}

	// Verify contest exists and belongs to this college
	var contest models.Contest
	if err := database.DB.Where("contest_id = ?", contestID).First(&contest).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}
	if err := validateCollegeAccess(user.CollegeID, contest.CollegeID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	result := database.DB.Where("contest_id = ? AND problem_id = ?", contestID, problemID).Delete(&models.ContestProblem{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove problem from contest"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found in this contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Problem removed from contest successfully"})
}

// =====================================================
// GetContestProblem — GET /contests/:id/problems/:problemId
// Returns full problem details for a contest participant.
// =====================================================

type ContestProblemDetailResponse struct {
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Difficulty      string            `json:"difficulty"`
	Tags            string            `json:"tags"`
	SampleTestCases []models.TestCase `json:"sample_test_cases"`
	ID              uint              `json:"id"`
	TimeLimit       int               `json:"time_limit"`
	MemoryLimit     int               `json:"memory_limit"`
	PointsInContest int               `json:"points_in_contest"`
	ProblemOrder    int               `json:"problem_order"`
	IsSolved        bool              `json:"is_solved"`
}

func GetContestProblem(c *gin.Context) {
	contestID := c.Param("id")
	problemID := c.Param("problemId")

	userRegdNo, exists := c.Get("regdno")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Load user
	var user models.User
	if err := database.DB.Where("regdno = ?", userRegdNo).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Load contest and verify college
	var contest models.Contest
	if err := database.DB.First(&contest, contestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}
	if err := validateCollegeAccess(user.CollegeID, contest.CollegeID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	isPractice := isPracticeActive(&contest)

	// Students must have joined active contests.
	// For ended contests with practice mode active, auto-register eligible students.
	if user.Role == "student" {
		if !isStudentEligibleForContest(userRegdNo.(string), contest) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not eligible for this contest"})
			return
		}

		var participation models.ContestParticipant
		if err := database.DB.Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).First(&participation).Error; err != nil {
			if isPractice {
				participation = models.ContestParticipant{
					ContestID:  contest.ContestID,
					UserRegdNo: userRegdNo.(string),
				}

				var student models.Student
				if err := database.DB.Where("regd_no = ?", userRegdNo).First(&student).Error; err == nil {
					snapshot := fmt.Sprintf(`{"cohort":%d,"branch_id":%d}`, student.CohortYear, student.BranchID)
					participation.EligibilitySnapshot = &snapshot
				}

				if err := database.DB.Create(&participation).Error; err != nil {
					if err := database.DB.Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).First(&participation).Error; err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register for practice mode"})
						return
					}
				}
			} else {
				c.JSON(http.StatusForbidden, gin.H{"error": "You must join this contest to view problems"})
				return
			}
		}
		if participation.Disqualified {
			c.JSON(http.StatusForbidden, gin.H{
				"error":       "You have been disqualified from this contest.",
				"redirect_to": "contests",
			})
			return
		}
		if participation.HasFinished && !isPractice {
			c.JSON(http.StatusForbidden, gin.H{"error": "You have already finished this contest."})
			return
		}
		if !isContestActive(contest) && !isPractice {
			if contest.StartTime.After(time.Now()) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Contest has not started yet"})
			} else {
				c.JSON(http.StatusForbidden, gin.H{"error": "Contest has ended"})
			}
			return
		}
	}

	// Load contest-problem link (verifies problem belongs to this contest)
	var cp models.ContestProblem
	if err := database.DB.Where("contest_id = ? AND problem_id = ?", contestID, problemID).First(&cp).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found in this contest"})
		return
	}

	// Load full problem with sample test cases
	var problem models.Problem
	if err := database.DB.Where("id = ?", problemID).
		Preload("TestCases", "is_sample = ?", true).
		First(&problem).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Problem not found"})
		return
	}

	sampleCases := problem.TestCases
	if sampleCases == nil {
		sampleCases = []models.TestCase{}
	}

	// Check if solved
	var solvedCount int64
	database.DB.Model(&models.ContestSubmission{}).
		Where("contest_id = ? AND problem_id = ? AND user_regd_no = ? AND passed = ?", contestID, problemID, userRegdNo, true).
		Count(&solvedCount)
	if user.Role == "student" && solvedCount > 0 && !isPractice {
		c.JSON(http.StatusForbidden, gin.H{
			"error":       "Problem already solved. Access is locked.",
			"redirect_to": "contest_problems",
		})
		return
	}

	c.JSON(http.StatusOK, ContestProblemDetailResponse{
		ID:              problem.ID,
		Title:           problem.Title,
		Description:     problem.Description,
		Difficulty:      problem.Difficulty,
		Tags:            problem.Tags,
		TimeLimit:       problem.TimeLimit,
		MemoryLimit:     problem.MemoryLimit,
		PointsInContest: cp.Points,
		ProblemOrder:    cp.ProblemOrder,
		IsSolved:        solvedCount > 0,
		SampleTestCases: sampleCases,
	})
}

// =====================================================
// FinishContest — POST /contests/:id/finish
// Marks the contest as finished for the current user.
// After finishing, no further submissions are allowed.
// =====================================================
func FinishContest(c *gin.Context) {
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

	// Check if contest is active
	if !isContestActive(contest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot finish a contest that is not active"})
		return
	}

	// Check if user is registered
	var participation models.ContestParticipant
	if err := database.DB.Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).First(&participation).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must join the contest first"})
		return
	}

	// Check if user has finished the contest
	if participation.HasFinished {
		c.JSON(http.StatusForbidden, gin.H{"error": "You have already finished this contest. No further submissions allowed."})
		return
	}

	// Check if already finished
	if participation.HasFinished {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Contest already finished"})
		return
	}

	// Mark as finished
	now := time.Now()
	participation.HasFinished = true
	participation.FinishedAt = &now

	if err := database.DB.Save(&participation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finish contest"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Contest finished successfully",
		"total_score":  participation.TotalScore,
		"solved_count": participation.ProblemsSolved,
		"finished_at":  now,
	})
}

// =====================================================
// Contest Violation Tracking (ESC Key)
// =====================================================

func ReportEscViolation(c *gin.Context) {
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

	// Contest must be active
	if !isContestActive(contest) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Contest is not active"})
		return
	}

	// Check if user is registered
	var participation models.ContestParticipant
	if err := database.DB.Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).First(&participation).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "You must join the contest first"})
		return
	}

	if participation.Disqualified {
		c.JSON(http.StatusForbidden, gin.H{
			"error":          "You have been disqualified from this contest.",
			"disqualified":   true,
			"esc_violations": participation.EscViolations,
		})
		return
	}

	// Lock participant row to safely increment violations
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var locked models.ContestParticipant
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("contest_id = ? AND user_regd_no = ?", contestID, userRegdNo).
		First(&locked).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock participant"})
		return
	}

	locked.EscViolations += 1

	// Track violations for plagiarism/integrity records, but do NOT disqualify
	// Violations are recorded but user can continue the contest
	// This allows instructors to review fullscreen exit patterns later for potential plagiarism

	if err := tx.Save(&locked).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update violations"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit violation update"})
		return
	}

	// Return violation count without disqualification
	response := gin.H{
		"esc_violations": locked.EscViolations,
		"disqualified":   false,
		"message":        fmt.Sprintf("Fullscreen exit recorded (violation #%d). Exiting fullscreen mode counts as a plagiarism indicator. Please stay in fullscreen mode.", locked.EscViolations),
	}

	c.JSON(http.StatusOK, response)
}

// ==================== SECTION MANAGEMENT ====================

// UpsertContestSections replaces all sections for a contest
func UpsertContestSections(c *gin.Context) {
	contestID := c.Param("id")

	var req struct {
		Sections []struct {
			ID              *uint  `json:"id,omitempty"`
			SectionName     string `json:"section_name" binding:"required"`
			SectionType     string `json:"section_type" binding:"required,oneof=coding quiz"`
			Description     string `json:"description"`
			DurationMinutes int    `json:"duration_minutes"`
		} `json:"sections" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var contest models.Contest
	if err := database.DB.First(&contest, contestID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contest not found"})
		return
	}

	// Delete existing sections
	database.DB.Where("contest_id = ?", contestID).Delete(&models.ContestSection{})
	database.DB.Model(&models.ContestProblem{}).Where("contest_id = ?", contestID).Update("section_id", nil)

	cID := parseContestID(contestID)
	for i, s := range req.Sections {
		section := models.ContestSection{
			ContestID:       uint(cID),
			SectionName:     s.SectionName,
			SectionType:     s.SectionType,
			Description:     s.Description,
			DurationMinutes: s.DurationMinutes,
			OrderIndex:      i,
		}
		database.DB.Create(&section)
	}

	var sections []models.ContestSection
	database.DB.Where("contest_id = ?", contestID).Order("order_index ASC").Find(&sections)

	c.JSON(http.StatusOK, gin.H{"message": "Sections updated", "sections": sections})
}

// GetContestSections returns all sections for a contest
func GetContestSections(c *gin.Context) {
	contestID := c.Param("id")

	var sections []models.ContestSection
	database.DB.Where("contest_id = ?", contestID).
		Order("order_index ASC").
		Preload("ContestProblems.Problem").
		Preload("ContestQuiz.Questions.Options").
		Find(&sections)

	c.JSON(http.StatusOK, gin.H{"sections": sections})
}

func parseContestID(id string) uint64 {
	v, _ := strconv.ParseUint(id, 10, 64)
	return v
}
