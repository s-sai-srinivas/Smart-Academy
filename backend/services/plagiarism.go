package services

import (
	"coding-platform/config"
	"coding-platform/errors"
	"coding-platform/logger"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =====================================================
// Constants and Configuration
// =====================================================

// Plagiarism detection thresholds
const (
	// Minimum code length to trigger plagiarism check (prevents false positives from boilerplate)
	MinCodeLength = 100 // characters

	// Similarity thresholds
	ThresholdPlagiarized = 60.0 // >60% = PLAGIARIZED
	ThresholdSuspicious  = 30.0 // 30-60% = SUSPICIOUS
	// <30% = SAFE

	// Rate limiting
	MaxConcurrentChecks = 5 // Maximum concurrent plagiarism checks
	RateLimitWindow     = time.Hour
	MaxChecksPerHour    = 20 // Max checks per hour per problem

	// Disk space requirements
	MinDiskSpaceMB       = 500       // Minimum free disk space required
	MaxSubmissionsPerRun = 500       // Maximum submissions to process in one run
	MaxSubmissionSize    = 50 * 1024 // 50KB max per submission file

	// Directory validation
	AllowedBaseDirs = "/opt/jplag:/tmp/jplag:/var/jplag" // Colon-separated allowed base directories
)

// LanguageIDToJPlag maps Judge0 language IDs to JPlag language codes
var LanguageIDToJPlag = map[int]string{
	62: "java",       // Java (verified)
	71: "python3",    // Python 3
	70: "python",     // Python 2
	48: "c",          // C
	75: "c",          // C (Clang)
	52: "cpp",        // C++ (GCC 7.4.0)
	53: "cpp",        // C++ (GCC 8.3.0)
	54: "cpp",        // C++ (Clang 7.0.1)
	63: "javascript", // JavaScript (Node.js) - verified
	59: "javascript", // TypeScript (mapped to JS for JPlag)
	60: "go",         // Go
	50: "csharp",     // C# (Mono)
	81: "scala",      // Scala
	68: "php",        // PHP
}

// File extension mapping for each language
var languageExtensions = map[string]string{
	"java":       ".java",
	"python3":    ".py",
	"python":     ".py",
	"c":          ".c",
	"cpp":        ".cpp",
	"javascript": ".js",
	"go":         ".go",
	"csharp":     ".cs",
	"scala":      ".scala",
	"php":        ".php",
}

// =====================================================
// Rate Limiter
// =====================================================

// RateLimiter tracks plagiarism check requests per problem
type RateLimiter struct {
	requests map[uint][]time.Time
	mu       sync.Mutex
}

// Global rate limiter instance
var plagiarismRateLimiter = &RateLimiter{
	requests: make(map[uint][]time.Time),
}

// AllowCheck checks if a plagiarism check is allowed for the given problem
func (rl *RateLimiter) AllowCheck(problemID uint) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-RateLimitWindow)

	// Get existing timestamps for this problem
	timestamps := rl.requests[problemID]

	// Filter to only timestamps within the window
	var validTimestamps []time.Time
	for _, ts := range timestamps {
		if ts.After(windowStart) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check if under limit
	if len(validTimestamps) >= MaxChecksPerHour {
		rl.requests[problemID] = validTimestamps
		return false
	}

	// Add current timestamp and update map
	validTimestamps = append(validTimestamps, now)
	rl.requests[problemID] = validTimestamps
	return true
}

// GetRemainingChecks returns remaining checks allowed for this hour
func (rl *RateLimiter) GetRemainingChecks(problemID uint) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-RateLimitWindow)
	timestamps := rl.requests[problemID]

	count := 0
	for _, ts := range timestamps {
		if ts.After(windowStart) {
			count++
		}
	}

	return MaxChecksPerHour - count
}

// Cleanup old entries periodically
func (rl *RateLimiter) StartCleanup() {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	windowStart := time.Now().Add(-RateLimitWindow)
	for problemID, timestamps := range rl.requests {
		var valid []time.Time
		for _, ts := range timestamps {
			if ts.After(windowStart) {
				valid = append(valid, ts)
			}
		}
		if len(valid) == 0 {
			delete(rl.requests, problemID)
		} else {
			rl.requests[problemID] = valid
		}
	}
}

// =====================================================
// Concurrency Manager
// =====================================================

// ConcurrencyManager limits concurrent plagiarism checks
type ConcurrencyManager struct {
	sem chan struct{}
}

// Global concurrency manager
var plagiarismConcurrencyManager = &ConcurrencyManager{
	sem: make(chan struct{}, MaxConcurrentChecks),
}

// Acquire tries to acquire a slot for plagiarism check
func (cm *ConcurrencyManager) Acquire() bool {
	select {
	case cm.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release releases a slot
func (cm *ConcurrencyManager) Release() {
	select {
	case <-cm.sem:
	default:
	}
}

// =====================================================
// Data Structures
// =====================================================

// JPlagComparison represents a single comparison from JPlag results
type JPlagComparison struct {
	FirstSubmission  string  `json:"firstSubmission"`
	SecondSubmission string  `json:"secondSubmission"`
	Similarity       float64 `json:"similarity"`
}

// JPlagOverview represents the overview.json structure from JPlag 5.x
type JPlagOverview struct {
	Comparisons []JPlagComparison `json:"topComparisons"`
}

// SubmissionInfo contains info about a submission for plagiarism check
type SubmissionInfo struct {
	CollegeID  *string
	UserRegdNo string
	Username   string
	SourceCode string
	ID         uint
	LanguageID int
}

// PlagiarismCheckResult represents the result of a plagiarism check
type PlagiarismCheckResult struct {
	UserRegdNo1       string  `json:"user_regdno_1"`
	UserRegdNo2       string  `json:"user_regdno_2"`
	Username1         string  `json:"username_1"`
	Username2         string  `json:"username_2"`
	Status            string  `json:"status"`
	SubmissionID1     uint    `json:"submission_id_1"`
	SubmissionID2     uint    `json:"submission_id_2"`
	SimilarityPercent float64 `json:"similarity_percent"`
}

// PlagiarismStats contains statistics about plagiarism checks
type PlagiarismStats struct {
	LastCheckTime time.Time `json:"last_check_time"`
	TotalChecks   int       `json:"total_checks"`
	avgSimilarity float64
	FlaggedCount  int `json:"flagged_count"`
}

// =====================================================
// Security Validation Functions
// =====================================================

// validatePath ensures the path is within allowed directories
// This prevents path traversal and unauthorized directory access
func validatePath(path string) error {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check if path is within allowed base directories
	allowedDirs := strings.Split(AllowedBaseDirs, ":")
	for _, allowedDir := range allowedDirs {
		allowedClean := filepath.Clean(allowedDir)
		// Ensure the path starts with the allowed directory
		if strings.HasPrefix(cleanPath, allowedClean) {
			return nil
		}
	}

	return errors.Errorf(errors.ErrValidation, "path '%s' is not within allowed directories", path)
}

// validateDockerMount validates the Docker mount path
func validateDockerMount(baseDir string) error {
	// Ensure baseDir is an absolute path
	if !filepath.IsAbs(baseDir) {
		return errors.Errorf(errors.ErrValidation, "JPlag base directory must be an absolute path")
	}

	// Validate the path is allowed
	if err := validatePath(baseDir); err != nil {
		return err
	}

	// Check directory exists and is writable
	info, err := os.Stat(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Errorf(errors.ErrValidation, "JPlag base directory does not exist: %s", baseDir)
		}
		return errors.Wrap(err, errors.ErrDatabase, "failed to access JPlag base directory")
	}

	if !info.IsDir() {
		return errors.Errorf(errors.ErrValidation, "JPlag base directory is not a directory: %s", baseDir)
	}

	// Check write permission
	testFile := filepath.Join(baseDir, ".write_test")
	if err := os.WriteFile(testFile, []byte(""), 0600); err != nil {
		return errors.Errorf(errors.ErrValidation, "JPlag base directory is not writable: %s", baseDir)
	}
	os.Remove(testFile)

	return nil
}

// sanitizeRunID ensures the run ID is safe for use in paths
func sanitizeRunID(runID string) bool {
	// Only allow alphanumeric characters and hyphens
	matched, _ := regexp.MatchString("^[a-zA-Z0-9-]+$", runID)
	return matched
}

// =====================================================
// Disk Space Validation
// =====================================================

// checkDiskSpace verifies sufficient disk space is available
func checkDiskSpace(path string, requiredMB int) error {
	// Get the drive/root directory
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		dir = "."
	}

	// Use GetFreeDiskSpace which is implemented per-platform
	freeBytes, err := getFreeDiskSpace(dir)
	if err != nil {
		return errors.Wrap(err, errors.ErrInternal, "failed to check disk space")
	}

	availableMB := freeBytes / 1024 / 1024

	if availableMB < uint64(requiredMB) {
		return errors.Errorf(
			errors.ErrServiceUnavailable,
			"insufficient disk space: %d MB available, %d MB required",
			availableMB, requiredMB,
		)
	}

	return nil
}

// getFreeDiskSpace returns free disk space in bytes for the given path
// Platform-specific implementation in plagiarism_windows.go or plagiarism_unix.go
func getFreeDiskSpace(path string) (uint64, error) {
	return getFreeDiskSpacePlatform(path)
}

// =====================================================
// Main Plagiarism Check Function
// =====================================================

// CheckPlagiarism runs JPlag on the given submissions for a problem
// This function includes all security checks, rate limiting, and validation
func CheckPlagiarism(problemID uint, submissions []SubmissionInfo) ([]PlagiarismCheckResult, error) {
	log := logger.GetGlobal().With("operation", "plagiarism_check").With("problem_id", problemID)

	// Validate submission count
	if len(submissions) < 2 {
		return nil, errors.NewValidation("need at least 2 submissions to check plagiarism")
	}

	// Check maximum submissions limit
	if len(submissions) > MaxSubmissionsPerRun {
		return nil, errors.Errorf(
			errors.ErrValidation,
			"too many submissions: %d (maximum: %d)",
			len(submissions), MaxSubmissionsPerRun,
		)
	}

	// Validate all submissions have the same language
	firstLang := submissions[0].LanguageID
	for _, sub := range submissions {
		if sub.LanguageID != firstLang {
			return nil, errors.NewValidation("all submissions must be in the same language")
		}
	}

	// Check language support
	jplagLang, ok := LanguageIDToJPlag[firstLang]
	if !ok {
		return nil, errors.Errorf(errors.ErrValidation, "unsupported language ID: %d", firstLang)
	}

	// Validate code lengths (prevent false positives from boilerplate)
	validSubmissions := make([]SubmissionInfo, 0, len(submissions))
	for _, sub := range submissions {
		if len(sub.SourceCode) >= MinCodeLength {
			validSubmissions = append(validSubmissions, sub)
		} else {
			log.Debug("Skipping submission - code too short",
				logger.KV("submission_id", sub.ID),
				logger.KV("code_length", len(sub.SourceCode)),
			)
		}
	}

	if len(validSubmissions) < 2 {
		return nil, errors.Errorf(
			errors.ErrValidation,
			"insufficient submissions with minimum code length (%d characters)",
			MinCodeLength,
		)
	}
	submissions = validSubmissions

	// Check rate limiting
	if !plagiarismRateLimiter.AllowCheck(problemID) {
		remaining := plagiarismRateLimiter.GetRemainingChecks(problemID)
		return nil, errors.Errorf(
			errors.ErrTooManyRequests,
			"rate limit exceeded. %d checks remaining this hour",
			remaining,
		)
	}

	// Acquire concurrency slot
	if !plagiarismConcurrencyManager.Acquire() {
		return nil, errors.NewServiceUnavailable("too many plagiarism checks running. please try again later")
	}
	defer plagiarismConcurrencyManager.Release()

	// Validate Docker mount path
	if err := validateDockerMount(config.AppConfig.JPlagBaseDir); err != nil {
		log.Error("Docker mount validation failed", err)
		return nil, errors.NewInternal("plagiarism service configuration error")
	}

	// Check disk space
	if err := checkDiskSpace(config.AppConfig.JPlagBaseDir, MinDiskSpaceMB); err != nil {
		log.Error("Disk space check failed", err)
		return nil, errors.NewServiceUnavailable("insufficient disk space for plagiarism check")
	}

	// Create unique run directory with validated run ID
	runID := uuid.New().String()
	if !sanitizeRunID(runID) {
		return nil, errors.NewInternal("failed to generate valid run ID")
	}

	runDir := filepath.Join(config.AppConfig.JPlagSubmissionsDir, fmt.Sprintf("run_%s", runID))
	resultsZipPath := filepath.Join(config.AppConfig.JPlagResultsDir, fmt.Sprintf("run_%s.zip", runID))
	resultsExtractDir := filepath.Join(config.AppConfig.JPlagResultsDir, fmt.Sprintf("run_%s", runID))

	// Validate all paths are within allowed directories
	for _, path := range []string{runDir, resultsZipPath, resultsExtractDir} {
		if err := validatePath(path); err != nil {
			return nil, errors.Wrap(err, errors.ErrInternal, "invalid path generated")
		}
	}

	// Cleanup on exit
	defer func() {
		os.RemoveAll(runDir)
		os.RemoveAll(resultsExtractDir)
		os.Remove(resultsZipPath)
	}()

	// Create submission directory
	if err := os.MkdirAll(runDir, 0755); err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "failed to create run directory")
	}

	// Create submission folder name to full info mapping
	submissionMap := make(map[string]SubmissionInfo)

	log.Info("Writing submissions for JPlag analysis",
		logger.KV("count", len(submissions)),
		logger.KV("language", jplagLang),
	)

	// Write submission files with size validation
	ext := languageExtensions[jplagLang]
	for _, sub := range submissions {
		// Validate submission size
		if len(sub.SourceCode) > MaxSubmissionSize {
			log.Warn("Submission too large, skipping",
				logger.KV("submission_id", sub.ID),
				logger.KV("size", len(sub.SourceCode)),
			)
			continue
		}

		folderName := fmt.Sprintf("s%d", sub.ID)
		submissionMap[folderName] = sub

		subDir := filepath.Join(runDir, folderName)
		if err := os.MkdirAll(subDir, 0755); err != nil {
			return nil, errors.Wrap(err, errors.ErrInternal, "failed to create submission directory")
		}

		filePath := filepath.Join(subDir, "solution"+ext)
		if err := os.WriteFile(filePath, []byte(sub.SourceCode), 0600); err != nil {
			return nil, errors.Wrap(err, errors.ErrInternal, "failed to write submission file")
		}
		log.Debug("Written submission for analysis",
			logger.KV("submission_id", sub.ID),
			logger.KV("path", filePath),
			logger.KV("size", len(sub.SourceCode)),
		)
	}

	// Run JPlag via Docker with validated command
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.AppConfig.JPlagTimeoutSeconds)*time.Second)
	defer cancel()

	// Build Docker command with validated paths
	// Security: Using validated paths only, no user input in mount paths
	dockerArgs := []string{"run", "--rm",
		"-v", fmt.Sprintf("%s:/data", config.AppConfig.JPlagBaseDir),
		config.AppConfig.JPlagDockerImage,
		"-l", jplagLang,
		fmt.Sprintf("/data/submissions/run_%s", runID),
		"-r", fmt.Sprintf("/data/results/run_%s", runID),
	}

	log.Info("Running JPlag analysis",
		logger.KV("command", strings.Join(dockerArgs, " ")),
		logger.KV("timeout_seconds", config.AppConfig.JPlagTimeoutSeconds),
	)

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	// Don't use CombinedOutput directly - capture separately for better control
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Error("JPlag execution timed out", err)
			return nil, errors.Errorf(errors.ErrTimeout, "JPlag execution timed out after %d seconds", config.AppConfig.JPlagTimeoutSeconds)
		}
		log.Warn("JPlag completed with warnings",
			logger.KV("error", err),
			logger.KV("stderr", stderr.String()),
		)
		// JPlag may still produce results even with non-zero exit code
	}

	log.Debug("JPlag completed",
		logger.KV("stdout", stdout.String()),
	)

	// Check if results ZIP exists (JPlag creates run_<uuid>.zip)
	if _, err := os.Stat(resultsZipPath); os.IsNotExist(err) {
		log.Error("No results ZIP found", nil)
		return nil, errors.NewInternal("JPlag did not produce results")
	}

	// Extract the ZIP file
	if err := os.MkdirAll(resultsExtractDir, 0755); err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "failed to create results directory")
	}

	unzipCmd := exec.Command("unzip", "-o", resultsZipPath, "-d", resultsExtractDir)
	unzipOutput, err := unzipCmd.CombinedOutput()
	if err != nil {
		log.Error("Unzip failed", err,
			logger.KV("output", string(unzipOutput)),
		)
		return nil, errors.Wrap(err, errors.ErrInternal, "failed to unzip results")
	}

	log.Info("Extracted JPlag results",
		logger.KV("path", resultsExtractDir),
	)

	// Parse results
	results, err := parseJPlagResults(resultsExtractDir, submissionMap)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "failed to parse JPlag results")
	}

	log.Info("Plagiarism check completed",
		logger.KV("comparisons", len(results)),
	)

	// Update stats
	updatePlagiarismStats(problemID, results)

	return results, nil
}

// =====================================================
// Statistics Tracking
// =====================================================

var (
	plagiarismStatsMu sync.RWMutex
	plagiarismStats   = make(map[uint]*PlagiarismStats) // problemID -> stats
)

func updatePlagiarismStats(problemID uint, results []PlagiarismCheckResult) {
	plagiarismStatsMu.Lock()
	defer plagiarismStatsMu.Unlock()

	stats, exists := plagiarismStats[problemID]
	if !exists {
		stats = &PlagiarismStats{}
		plagiarismStats[problemID] = stats
	}

	stats.TotalChecks++
	stats.LastCheckTime = time.Now()

	// Calculate average similarity
	var totalSimilarity float64
	flaggedCount := 0
	for _, r := range results {
		totalSimilarity += r.SimilarityPercent
		if r.Status == "SUSPICIOUS" || r.Status == "PLAGIARIZED" {
			flaggedCount++
		}
	}

	if len(results) > 0 {
		stats.avgSimilarity = (stats.avgSimilarity*float64(stats.TotalChecks-1) + totalSimilarity) / float64(stats.TotalChecks)
	}
	stats.FlaggedCount += flaggedCount
}

// GetPlagiarismStats returns statistics for a problem
func GetPlagiarismStats(problemID uint) *PlagiarismStats {
	plagiarismStatsMu.RLock()
	defer plagiarismStatsMu.RUnlock()

	return plagiarismStats[problemID]
}

// =====================================================
// JPlag Results Parsing
// =====================================================

// parseJPlagResults reads and parses the JPlag output from extracted directory
func parseJPlagResults(resultsDir string, submissionMap map[string]SubmissionInfo) ([]PlagiarismCheckResult, error) {
	var results []PlagiarismCheckResult

	log := logger.GetGlobal().With("operation", "parse_jplag_results")

	// List all files in the results directory
	files, err := os.ReadDir(resultsDir)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "failed to read results directory")
	}

	// Look for comparison files (format: s1-s2.json)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()

		// Parse sX-sY.json files (comparison results)
		if strings.HasSuffix(name, ".json") && strings.Contains(name, "-") {
			// Extract submission IDs from filename like "s1-s2.json"
			baseName := strings.TrimSuffix(name, ".json")
			parts := strings.Split(baseName, "-")
			if len(parts) != 2 {
				continue
			}

			sub1, ok1 := submissionMap[parts[0]]
			sub2, ok2 := submissionMap[parts[1]]
			if !ok1 || !ok2 {
				log.Warn("Could not map comparison to submissions",
					logger.KV("parts", parts),
				)
				continue
			}

			// Read and parse the comparison JSON
			filePath := filepath.Join(resultsDir, name)
			data, err := os.ReadFile(filePath)
			if err != nil {
				log.Debug("Failed to read comparison file",
					logger.KV("file", name),
					logger.KV("error", err),
				)
				continue
			}

			// Parse comparison JSON to get similarity
			var comparison struct {
				Similarities map[string]float64 `json:"similarities"`
				ID1          string             `json:"id1"`
				ID2          string             `json:"id2"`
			}
			if err := json.Unmarshal(data, &comparison); err != nil {
				log.Debug("Failed to parse comparison file",
					logger.KV("file", name),
					logger.KV("error", err),
				)
				continue
			}

			// Get MAX similarity (most important metric)
			similarity := comparison.Similarities["MAX"]
			if similarity == 0 {
				// Try AVG as fallback
				similarity = comparison.Similarities["AVG"]
			}

			similarityPercent := similarity * 100
			status := classifyStatus(similarityPercent)

			log.Debug("Parsed comparison",
				logger.KV("file", name),
				logger.KV("similarity", similarityPercent),
				logger.KV("status", status),
			)

			results = append(results, PlagiarismCheckResult{
				SubmissionID1:     sub1.ID,
				SubmissionID2:     sub2.ID,
				UserRegdNo1:       sub1.UserRegdNo,
				UserRegdNo2:       sub2.UserRegdNo,
				Username1:         sub1.Username,
				Username2:         sub2.Username,
				SimilarityPercent: similarityPercent,
				Status:            status,
			})
		}
	}

	// If no comparison files found, try overview.json as fallback
	if len(results) == 0 {
		overviewPath := filepath.Join(resultsDir, "overview.json")
		if _, err := os.Stat(overviewPath); err == nil {
			log.Debug("Trying overview.json as fallback")
			return parseOverviewJSON(overviewPath, submissionMap)
		}
	}

	return results, nil
}

// parseOverviewJSON parses the overview.json file directly
func parseOverviewJSON(path string, submissionMap map[string]SubmissionInfo) ([]PlagiarismCheckResult, error) {
	var results []PlagiarismCheckResult

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "failed to read overview.json")
	}

	var overview JPlagOverview
	if err := json.Unmarshal(data, &overview); err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "failed to parse overview.json")
	}

	for _, comp := range overview.Comparisons {
		sub1, ok1 := submissionMap[comp.FirstSubmission]
		sub2, ok2 := submissionMap[comp.SecondSubmission]
		if !ok1 || !ok2 {
			continue
		}

		similarityPercent := comp.Similarity * 100
		status := classifyStatus(similarityPercent)

		results = append(results, PlagiarismCheckResult{
			SubmissionID1:     sub1.ID,
			SubmissionID2:     sub2.ID,
			UserRegdNo1:       sub1.UserRegdNo,
			UserRegdNo2:       sub2.UserRegdNo,
			Username1:         sub1.Username,
			Username2:         sub2.Username,
			SimilarityPercent: similarityPercent,
			Status:            status,
		})
	}

	return results, nil
}

// classifyStatus returns the plagiarism status based on similarity percentage
func classifyStatus(similarity float64) string {
	switch {
	case similarity > ThresholdPlagiarized:
		return "PLAGIARIZED"
	case similarity >= ThresholdSuspicious:
		return "SUSPICIOUS"
	default:
		return "SAFE"
	}
}

// GetJPlagLanguage returns the JPlag language code for a Judge0 language ID
func GetJPlagLanguage(languageID int) (string, bool) {
	lang, ok := LanguageIDToJPlag[languageID]
	return lang, ok
}

// IsJPlagLanguageSupported checks if a language is supported by JPlag
func IsJPlagLanguageSupported(languageID int) bool {
	_, ok := LanguageIDToJPlag[languageID]
	return ok
}

// ParseSubmissionID extracts submission ID from a folder name like "s123"
func ParseSubmissionID(folderName string) (uint, error) {
	if !strings.HasPrefix(folderName, "s") {
		return 0, errors.NewValidation("invalid folder name format")
	}
	id, err := strconv.ParseUint(folderName[1:], 10, 32)
	if err != nil {
		return 0, errors.Wrap(err, errors.ErrValidation, "invalid submission ID in folder name")
	}
	return uint(id), nil
}

// Initialize starts background cleanup tasks
func Initialize() {
	plagiarismRateLimiter.StartCleanup()
	log.Printf("[Plagiarism] Rate limiter cleanup started")
}
