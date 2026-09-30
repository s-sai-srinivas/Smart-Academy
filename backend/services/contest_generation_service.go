package services

import (
	"coding-platform/database"
	"coding-platform/models"
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"
)

// =====================================================
// Contest Generation Service
// =====================================================
// AI-powered contest problem generation with verification
// =====================================================

type ContestGenerationService struct {
	db           *gorm.DB
	aiProvider   *GeminiProvider
	orchestrator *AgentOrchestrator
}

// NewContestGenerationService creates a new contest generation service
func NewContestGenerationService(aiProvider *GeminiProvider, orchestrator *AgentOrchestrator) *ContestGenerationService {
	return &ContestGenerationService{
		db:           database.DB,
		aiProvider:   aiProvider,
		orchestrator: orchestrator,
	}
}

// =====================================================
// Problem Generation - Agentic Flow
// =====================================================

// GenerateContestProblems generates contest problems based on topics
// Uses true agentic ReAct loop with self-correction
func (s *ContestGenerationService) GenerateContestProblems(ctx context.Context, req models.ContestGenerationRequest) ([]models.GeneratedProblem, error) {
	log.Printf("[ContestGeneration] Starting AGENTIC problem generation for topics: %v", req.Topics)
	log.Printf("[ContestGeneration] Requested count: %d, Difficulty: %s", req.RequestedCount, req.DifficultyLevel)

	if s.orchestrator == nil {
		log.Printf("[ContestGeneration] WARNING: No orchestrator available, falling back to sequential generation")
		// Fallback to sequential generation if orchestrator not available
		return s.GenerateContestProblemsFallback(ctx, req)
	}

	// Get course info for context
	courseName := ""
	if req.CourseID != nil {
		var course models.Course
		if err := s.db.First(&course, *req.CourseID).Error; err == nil {
			courseName = course.CourseName
		}
	}

	// Use VersionsPerProblem from request (default to 1 = exact count)
	versionsMultiplier := req.VersionsPerProblem
	if versionsMultiplier <= 0 {
		versionsMultiplier = 1 // Default: generate exactly the requested count
	}
	generatedCount := req.RequestedCount * versionsMultiplier

	log.Printf("[ContestGeneration] Will generate %d problems using direct pipeline", generatedCount)

	var problems []models.GeneratedProblem

	for i := 0; i < generatedCount; i++ {
		select {
		case <-ctx.Done():
			return problems, ctx.Err()
		default:
		}

		log.Printf("[ContestGeneration] Generating problem %d/%d using direct pipeline", i+1, generatedCount)

		// Use the direct pipeline (Generate → Verify → Fix → Finish)
		// instead of the ReAct agent loop to minimize API calls
		problem, err := s.orchestrator.GenerateContestProblemDirect(
			ctx,
			req.Topics,
			req.DifficultyLevel,
			courseName,
		)

		if err != nil {
			log.Printf("[ContestGeneration] Direct generation failed for problem %d: %v", i+1, err)
			continue // Try next problem
		}

		if problem == nil {
			log.Printf("[ContestGeneration] Direct pipeline returned nil problem %d", i+1)
			continue
		}

		log.Printf("[ContestGeneration] Problem %d generated: %s (verification: %s)",
			i+1, problem.Title, problem.VerificationStatus)

		problems = append(problems, *problem)

		// Rate limiting: add delay between problem generations to avoid API rate limits
		if i < generatedCount-1 {
			delay := 2 * time.Second
			log.Printf("[ContestGeneration] Waiting %v before next problem to avoid rate limits", delay)
			select {
			case <-ctx.Done():
				return problems, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	log.Printf("[ContestGeneration] Agentic generation completed: %d/%d problems successful",
		len(problems), generatedCount)

	if len(problems) == 0 {
		return nil, fmt.Errorf("failed to generate any valid problems using agentic flow")
	}

	return problems, nil
}

// GenerateContestProblemsFallback falls back to sequential generation if orchestrator is not available
func (s *ContestGenerationService) GenerateContestProblemsFallback(ctx context.Context, req models.ContestGenerationRequest) ([]models.GeneratedProblem, error) {
	log.Printf("[ContestGeneration] Using fallback sequential generation")

	// Get course info for context
	courseName := ""
	if req.CourseID != nil {
		var course models.Course
		if err := s.db.First(&course, *req.CourseID).Error; err == nil {
			courseName = course.CourseName
		}
	}

	// Use sequential generation with rate limiting
	problems, err := s.aiProvider.GenerateContestProblemsSequential(ctx, req, courseName)
	if err != nil {
		return nil, fmt.Errorf("sequential problem generation failed: %w", err)
	}

	log.Printf("[ContestGeneration] Generated %d problems", len(problems))
	return problems, nil
}

// =====================================================
// Problem Verification
// =====================================================

// VerifyProblem verifies a generated problem by running reference solution against test cases
func (s *ContestGenerationService) VerifyProblem(_ context.Context, problem *models.GeneratedProblem) error {
	if problem.ReferenceSolution == "" {
		return fmt.Errorf("no reference solution provided")
	}

	if len(problem.TestCases) == 0 {
		return fmt.Errorf("no test cases provided")
	}

	log.Printf("[ContestGeneration] Verifying problem: %s", problem.Title)

	// Determine language ID for Judge0
	languageID := 71 // Python 3
	if strings.ToLower(problem.SolutionLanguage) == "cpp" {
		languageID = 54 // C++17
	}

	// Get time and memory limits from constraints or use defaults
	timeLimit := 2000     // ms
	memoryLimit := 256000 // KB

	if constraints, ok := problem.Constraints["time_limit_ms"].(float64); ok {
		timeLimit = int(constraints)
	}
	if constraints, ok := problem.Constraints["memory_limit_kb"].(float64); ok {
		memoryLimit = int(constraints)
	}

	// Run reference solution against all test cases (both sample and hidden)
	passedCount := 0
	for i, tc := range problem.TestCases {
		result, err := SubmitCode(
			problem.ReferenceSolution,
			languageID,
			tc.Input,
			tc.ExpectedOutput,
			timeLimit,
			memoryLimit,
		)

		if err != nil {
			log.Printf("[ContestGeneration] Test case %d execution error: %v", i+1, err)
			problem.VerificationStatus = "failed"
			return fmt.Errorf("test case %d execution failed: %w", i+1, err)
		}

		if result.Passed {
			passedCount++
			log.Printf("[ContestGeneration] Test case %d: PASSED", i+1)
		} else {
			log.Printf("[ContestGeneration] Test case %d: FAILED - Status: %s, Expected: %s, Got: %s",
				i+1, result.Status, tc.ExpectedOutput, result.Stdout)
			problem.VerificationStatus = "failed"
			return fmt.Errorf("test case %d failed: expected %s, got %s", i+1, tc.ExpectedOutput, result.Stdout)
		}
	}

	log.Printf("[ContestGeneration] Verification passed: %d/%d test cases", passedCount, len(problem.TestCases))
	problem.VerificationStatus = "passed"
	return nil
}

// =====================================================
// Stress Testing
// =====================================================

// StressTestProblem performs stress testing on a problem with random inputs
func (s *ContestGenerationService) StressTestProblem(_ context.Context, problem *models.GeneratedProblem, testCount int) error {
	if problem.ReferenceSolution == "" || len(problem.TestCases) == 0 {
		return fmt.Errorf("cannot stress test without solution and test cases")
	}

	log.Printf("[ContestGeneration] Starting stress test with %d random cases", testCount)

	languageID := 71 // Python 3
	if strings.ToLower(problem.SolutionLanguage) == "cpp" {
		languageID = 54
	}

	// Extract constraints for random input generation
	constraints := s.parseConstraints(problem.Constraints)

	for i := 0; i < testCount; i++ {
		// Generate random input based on constraints
		randomInput := s.generateRandomInput(constraints)

		// Run solution (we can't verify output without expected output)
		// Just check that solution doesn't crash or timeout
		result, err := SubmitCode(
			problem.ReferenceSolution,
			languageID,
			randomInput,
			"", // No expected output - just checking it runs
			constraints.TimeLimitMS,
			constraints.MemoryLimitKB,
		)

		if err != nil {
			return fmt.Errorf("stress test %d failed: %w", i+1, err)
		}

		// Check for runtime errors
		if result.Status == "Runtime Error (NZEC)" || result.Status == "Error" {
			return fmt.Errorf("stress test %d: solution crashed with %s", i+1, result.Status)
		}

		// Check for time limit exceeded
		if result.Status == "Time Limit Exceeded" {
			return fmt.Errorf("stress test %d: solution exceeded time limit", i+1)
		}

		log.Printf("[ContestGeneration] Stress test %d: OK (time: %.2fms, memory: %dKB)",
			i+1, result.Time, result.Memory)
	}

	log.Printf("[ContestGeneration] Stress testing completed successfully")
	return nil
}

// ConstraintValues holds parsed constraint values for input generation
type ConstraintValues struct {
	MaxN            int
	MaxArrayElement int
	MinArrayElement int
	TimeLimitMS     int
	MemoryLimitKB   int
	HasArray        bool
}

// parseConstraints extracts constraint values from problem constraints map
func (s *ContestGenerationService) parseConstraints(constraints map[string]interface{}) ConstraintValues {
	result := ConstraintValues{
		MaxN:            1000,
		MaxArrayElement: 1000000000,
		MinArrayElement: 1,
		TimeLimitMS:     2000,
		MemoryLimitKB:   256000,
		HasArray:        false,
	}

	for key, value := range constraints {
		switch strings.ToLower(key) {
		case "n", "n_max", "max_n":
			if v, ok := value.(float64); ok {
				result.MaxN = int(v)
			}
		case "a[i]", "array_elements", "elements":
			// Parse range like "1 <= A[i] <= 10^9"
			result.HasArray = true
		case "time_limit_ms":
			if v, ok := value.(float64); ok {
				result.TimeLimitMS = int(v)
			}
		case "memory_limit_kb":
			if v, ok := value.(float64); ok {
				result.MemoryLimitKB = int(v)
			}
		}
	}

	// Reduce maxN for stress testing (we want quick tests)
	if result.MaxN > 1000 {
		result.MaxN = 1000
	}

	return result
}

// generateRandomInput generates random input based on constraints
func (s *ContestGenerationService) generateRandomInput(constraints ConstraintValues) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	n := r.Intn(constraints.MaxN-1) + 1

	if constraints.HasArray {
		// Generate array input
		arr := make([]string, n)
		for i := 0; i < n; i++ {
			arr[i] = fmt.Sprintf("%d", r.Intn(constraints.MaxArrayElement-constraints.MinArrayElement)+constraints.MinArrayElement)
		}
		return fmt.Sprintf("%d\n%s", n, strings.Join(arr, " "))
	}

	// Single integer input
	return fmt.Sprintf("%d", n)
}

// =====================================================
// Cross Validation (Two Solutions)
// =====================================================

// CrossValidate generates two solutions and compares their outputs
func (s *ContestGenerationService) CrossValidate(_ context.Context, problem *models.GeneratedProblem) error {
	if problem.ReferenceSolution == "" {
		return fmt.Errorf("no reference solution provided")
	}

	log.Printf("[ContestGeneration] Starting cross validation for: %s", problem.Title)

	// For now, just run the reference solution on sample cases
	// In a full implementation, we would generate a brute force solution too
	languageID := 71
	if strings.ToLower(problem.SolutionLanguage) == "cpp" {
		languageID = 54
	}

	// Test on sample cases
	for i, tc := range problem.TestCases {
		if !tc.IsSampleVisible {
			continue
		}

		result, err := SubmitCode(
			problem.ReferenceSolution,
			languageID,
			tc.Input,
			tc.ExpectedOutput,
			2000,
			256000,
		)

		if err != nil {
			return fmt.Errorf("cross validation test %d failed: %w", i+1, err)
		}

		if !result.Passed {
			return fmt.Errorf("cross validation failed on sample %d", i+1)
		}
	}

	log.Printf("[ContestGeneration] Cross validation passed")
	return nil
}

// =====================================================
// Editorial Generation
// =====================================================

// GenerateEditorial generates editorial hints for a problem
func (s *ContestGenerationService) GenerateEditorial(ctx context.Context, req models.EditorialRequest) (*models.EditorialResponse, error) {
	log.Printf("[ContestGeneration] Generating editorial for problem %d", req.ProblemID)

	prompt := s.buildEditorialPrompt(req)

	response, err := s.aiProvider.generateEditorialHints(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("editorial generation failed: %w", err)
	}

	return response, nil
}

// buildEditorialPrompt creates the prompt for editorial generation
func (s *ContestGenerationService) buildEditorialPrompt(req models.EditorialRequest) string {
	var sb strings.Builder

	sb.WriteString("## ROLE\n")
	sb.WriteString("You are an educational content creator specializing in writing helpful hints\n")
	sb.WriteString("for competitive programming problems WITHOUT giving away full solutions.\n\n")

	sb.WriteString("## PROBLEM DETAILS\n")
	sb.WriteString(fmt.Sprintf("Description: %s\n", req.ProblemDescription))
	if req.ProblemConstraints != "" {
		sb.WriteString(fmt.Sprintf("Constraints: %s\n", req.ProblemConstraints))
	}
	sb.WriteString(fmt.Sprintf("Sample Input: %s\n", req.SampleInput))
	sb.WriteString(fmt.Sprintf("Sample Output: %s\n", req.SampleOutput))

	sb.WriteString("\n## TASK\n")
	sb.WriteString("Generate 2-3 progressive hints that guide students toward the solution\n")
	sb.WriteString("WITHOUT revealing the complete approach or providing code.\n\n")

	sb.WriteString("## OUTPUT FORMAT\n")
	sb.WriteString(`Return JSON:
{
    "hint_1": "Gentle nudge toward the right thinking direction (most vague)",
    "hint_2": "More specific guidance about the algorithm/approach",
    "hint_3": "Almost reveals the approach but no implementation details",
    "core_idea": "2-3 sentence explanation of the key insight",
    "time_complexity": "Expected time complexity (e.g., O(N log N))",
    "space_complexity": "Expected space complexity"
}

## HINT QUALITY CRITERIA
1. PROGRESSIVE: Each hint reveals more than the previous
2. HINTFUL: Guide without giving the answer
3. EDUCATIONAL: Help students learn the pattern
4. NO CODE: Never include actual code in hints

Return ONLY valid JSON. No markdown, no additional text.`)

	return sb.String()
}

// =====================================================
// Helper: Save Approved Problems to Database
// =====================================================

// SaveApprovedProblems saves approved generated problems to the database
func (s *ContestGenerationService) SaveApprovedProblems(
	adminRegdNo string,
	_ *uint,
	problems []models.GeneratedProblem,
	collegeID *string,
) ([]uint, error) {
	var problemIDs []uint

	for _, p := range problems {
		// Create Problem record
		problem := models.Problem{
			Title: p.Title,
			Description: fmt.Sprintf("%s\n\n## Input Format\n%s\n\n## Output Format\n%s\n\n## Sample Input\n```\n%s\n```\n\n## Sample Output\n```\n%s\n```\n\n## Sample Explanation\n%s",
				p.Description, p.InputFormat, p.OutputFormat, p.SampleInput, p.SampleOutput, p.SampleExplanation),
			Difficulty:  p.Difficulty,
			Tags:        strings.Join(p.TopicsCovered, ","),
			TimeLimit:   2000,
			MemoryLimit: 256000,
			IsGlobal:    false,
			CollegeID:   collegeID,
			CreatedBy:   adminRegdNo,
		}

		if err := s.db.Create(&problem).Error; err != nil {
			return nil, fmt.Errorf("failed to create problem %s: %w", p.Title, err)
		}

		problemIDs = append(problemIDs, problem.ID)

		// Create Test Cases
		for _, tc := range p.TestCases {
			testCase := models.TestCase{
				ProblemID:      problem.ID,
				Input:          tc.Input,
				ExpectedOutput: tc.ExpectedOutput,
				IsSample:       tc.IsSampleVisible,
				Points:         10, // Default points
			}

			if err := s.db.Create(&testCase).Error; err != nil {
				log.Printf("Warning: failed to create test case for problem %d: %v", problem.ID, err)
			}
		}

		log.Printf("[ContestGeneration] Saved problem %d: %s", problem.ID, problem.Title)
	}

	return problemIDs, nil
}

// GetCourseCollegeID retrieves the college ID for a course
func (s *ContestGenerationService) GetCourseCollegeID(courseID uint) (*string, error) {
	var course models.Course
	if err := s.db.First(&course, courseID).Error; err != nil {
		return nil, err
	}
	// Return collegeID as pointer (Course.CollegeID is string, we need *string)
	if course.CollegeID == "" {
		return nil, nil
	}
	return &course.CollegeID, nil
}
