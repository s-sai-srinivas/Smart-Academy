package services

import (
	"coding-platform/database"
	"coding-platform/models"
	"time"
)

// UserService handles user-related business logic
type UserService struct{}

// NewUserService creates a new UserService instance
func NewUserService() *UserService {
	return &UserService{}
}

// UpdateActivity updates the user's daily activity record
// Uses UPSERT to prevent race conditions
func (s *UserService) UpdateActivity(userRegdNo string) error {
	today := time.Now().Truncate(24 * time.Hour)

	err := database.DB.Exec(`
		INSERT INTO user_activities (user_regd_no, date, count, created_at, updated_at)
		VALUES (?, ?, 1, NOW(), NOW())
		ON CONFLICT (user_regd_no, date) DO UPDATE SET
			count = user_activities.count + 1,
			updated_at = NOW()
	`, userRegdNo, today).Error

	return err
}

// UpdateStreak updates the user's submission streak
// Uses database-level UPSERT to prevent race conditions
func (s *UserService) UpdateStreak(userRegdNo string) error {
	today := time.Now().Truncate(24 * time.Hour)
	todayStr := today.Format("2006-01-02")

	err := database.DB.Exec(`
		INSERT INTO user_streaks (user_regd_no, current_streak, longest_streak, last_activity_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
		ON CONFLICT (user_regd_no) DO UPDATE SET
			current_streak = CASE
			 WHEN user_streaks.last_activity_date = ? THEN user_streaks.current_streak + 1
			 WHEN user_streaks.last_activity_date = ? - INTERVAL '1 day' THEN user_streaks.current_streak + 1
			 ELSE 1
			END,
			longest_streak = CASE
			 WHEN user_streaks.last_activity_date = ? THEN user_streaks.longest_streak
			 WHEN user_streaks.last_activity_date = ? - INTERVAL '1 day' THEN GREATEST(user_streaks.longest_streak, user_streaks.current_streak + 1)
			 ELSE GREATEST(user_streaks.longest_streak, 1)
			END,
			last_activity_date = ?,
			updated_at = NOW()
	`, userRegdNo, 1, 1, todayStr, todayStr, todayStr, todayStr, todayStr, todayStr, today).Error

	return err
}

// GetStreak retrieves the user's current and longest streak
func (s *UserService) GetStreak(userRegdNo string) (currentStreak, longestStreak int, err error) {
	var streak models.UserStreak
	err = database.DB.Where("user_regd_no = ?", userRegdNo).First(&streak).Error
	if err != nil {
		return 0, 0, err
	}
	return streak.CurrentStreak, streak.LongestStreak, nil
}

// CalculateRuntimePercentile calculates the percentile rank of a runtime
func (s *UserService) CalculateRuntimePercentile(problemID uint, runtime float64) (float64, error) {
	var countFaster, totalCount int64

	err := database.DB.Model(&models.Submission{}).
		Where("problem_id = ? AND passed = ? AND execution_time < ?", problemID, true, runtime).
		Count(&countFaster).Error
	if err != nil {
		return 0, err
	}

	err = database.DB.Model(&models.Submission{}).
		Where("problem_id = ? AND passed = ?", problemID, true).
		Count(&totalCount).Error
	if err != nil {
		return 0, err
	}

	if totalCount == 0 {
		return 0, nil
	}

	percentile := (float64(countFaster) / float64(totalCount)) * 100
	return percentile, nil
}

// CalculateMemoryPercentile calculates the percentile rank of memory usage
func (s *UserService) CalculateMemoryPercentile(problemID uint, memory int) (float64, error) {
	var countLower, totalCount int64

	err := database.DB.Model(&models.Submission{}).
		Where("problem_id = ? AND passed = ? AND memory < ?", problemID, true, memory).
		Count(&countLower).Error
	if err != nil {
		return 0, err
	}

	err = database.DB.Model(&models.Submission{}).
		Where("problem_id = ? AND passed = ?", problemID, true).
		Count(&totalCount).Error
	if err != nil {
		return 0, err
	}

	if totalCount == 0 {
		return 0, nil
	}

	percentile := (float64(countLower) / float64(totalCount)) * 100
	return percentile, nil
}

// GetRuntimeStats gets min, max, and average runtime for a problem
func (s *UserService) GetRuntimeStats(problemID uint) (min, max, avg float64, err error) {
	var stats struct {
		Min *float64
		Max *float64
		Avg *float64
	}

	err = database.DB.Model(&models.Submission{}).
		Select("MIN(execution_time) as min, MAX(execution_time) as max, AVG(execution_time) as avg").
		Where("problem_id = ? AND passed = ?", problemID, true).
		Scan(&stats).Error

	if err != nil {
		return 0, 0, 0, err
	}

	if stats.Min != nil {
		min = *stats.Min
	}
	if stats.Max != nil {
		max = *stats.Max
	}
	if stats.Avg != nil {
		avg = *stats.Avg
	}

	return min, max, avg, nil
}

// GetMemoryStats gets min, max, and average memory for a problem
func (s *UserService) GetMemoryStats(problemID uint) (min, max, avg float64, err error) {
	var stats struct {
		Min *int64
		Max *int64
		Avg *float64
	}

	err = database.DB.Model(&models.Submission{}).
		Select("MIN(memory) as min, MAX(memory) as max, AVG(memory) as avg").
		Where("problem_id = ? AND passed = ?", problemID, true).
		Scan(&stats).Error

	if err != nil {
		return 0, 0, 0, err
	}

	if stats.Min != nil {
		min = float64(*stats.Min)
	}
	if stats.Max != nil {
		max = float64(*stats.Max)
	}
	if stats.Avg != nil {
		avg = *stats.Avg
	}

	return min, max, avg, nil
}
