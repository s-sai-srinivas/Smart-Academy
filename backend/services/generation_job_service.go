package services

import (
	"coding-platform/database"
	"coding-platform/models"
	"coding-platform/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	pdfReader "github.com/ledongthuc/pdf"
	"gorm.io/gorm"
)

// GenerationJobService handles async generation jobs
type GenerationJobService struct {
	db           *gorm.DB
	aiProvider   *GeminiProvider
	orchestrator *AgentOrchestrator
}

// NewGenerationJobService creates a new generation job service
func NewGenerationJobService(aiProvider *GeminiProvider) *GenerationJobService {
	return &GenerationJobService{
		db:           database.DB,
		aiProvider:   aiProvider,
		orchestrator: NewAgentOrchestrator(aiProvider),
	}
}

// CreateJob creates a new generation job
func (s *GenerationJobService) CreateJob(adminRegdNo string, courseID *uint, theoryID *uint, jobType models.GenerationJobType, filePath string) (*models.GenerationJob, error) {
	job := &models.GenerationJob{
		AdminRegdNo:   adminRegdNo,
		CourseID:      courseID,
		TheoryID:      theoryID,
		JobType:       jobType,
		Status:        models.GenerationJobPending,
		InputFilePath: filePath,
	}

	if err := s.db.Create(job).Error; err != nil {
		return nil, fmt.Errorf("failed to create generation job: %w", err)
	}

	return job, nil
}

// GetJob retrieves a generation job by ID
func (s *GenerationJobService) GetJob(jobID uint) (*models.GenerationJob, error) {
	job := &models.GenerationJob{}
	if err := s.db.First(job, jobID).Error; err != nil {
		return nil, err
	}
	return job, nil
}

// GetJobWithRelations retrieves a generation job with related data
func (s *GenerationJobService) GetJobWithRelations(jobID uint) (*models.GenerationJob, error) {
	job := &models.GenerationJob{}
	if err := s.db.
		Preload("Course").
		Preload("Theory").
		First(job, jobID).Error; err != nil {
		return nil, err
	}
	return job, nil
}

// ProcessJob starts async processing of a generation job
func (s *GenerationJobService) ProcessJob(ctx context.Context, jobID uint) error {
	job := &models.GenerationJob{}
	if err := s.db.First(job, jobID).Error; err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	// Update status to processing
	job.Status = models.GenerationJobProcessing
	if err := s.db.Save(job).Error; err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Process based on job type
	var err error
	switch job.JobType {
	case models.GenerationJobLessonPlanParse:
		err = s.processLessonPlan(ctx, job)
	case models.GenerationJobQuizGenerate:
		err = s.processQuizGeneration(ctx, job)
	case models.GenerationJobContestGenerate:
		err = s.processContestGeneration(ctx, job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.JobType)
	}

	// Update job status based on result
	now := time.Now()
	if err != nil {
		job.Status = models.GenerationJobFailed
		job.ErrorMessage = err.Error()
		job.CompletedAt = &now
	} else {
		job.Status = models.GenerationJobCompleted
		job.CompletedAt = &now
	}

	if saveErr := s.db.Save(job).Error; saveErr != nil {
		return fmt.Errorf("failed to save job completion status: %w", saveErr)
	}

	return err
}

// processLessonPlan processes a lesson plan parsing job using the multi-agent system
func (s *GenerationJobService) processLessonPlan(ctx context.Context, job *models.GenerationJob) error {
	log.Printf("[GenerationJobService] ========== START LESSON PLAN PROCESSING ==========")
	log.Printf("[GenerationJobService] Job ID: %d, Course ID: %v", job.ID, job.CourseID)

	// Step 1: Read the uploaded file
	log.Printf("[GenerationJobService] Step 1: Reading uploaded file: %s", job.InputFilePath)
	fileContent, err := s.readFile(job.InputFilePath)
	if err != nil {
		log.Printf("[GenerationJobService] ERROR: Failed to read file: %v", err)
		return fmt.Errorf("failed to read file: %w", err)
	}
	log.Printf("[GenerationJobService] File read successfully. Content length: %d characters", len(fileContent))
	log.Printf("[GenerationJobService] === DOCUMENT CONTENT PREVIEW (first 1000 chars) ===")
	log.Printf("%s", fileContent[:min(1000, len(fileContent))])
	if len(fileContent) > 1000 {
		log.Printf("... [content truncated, %d more characters]", len(fileContent)-1000)
	}
	log.Printf("[GenerationJobService] === END DOCUMENT PREVIEW ===")

	// Step 2: Get course info for context
	var course models.Course
	var hasCourse bool
	if job.CourseID != nil {
		log.Printf("[GenerationJobService] Step 2: Fetching course info for ID: %d", *job.CourseID)
		if err := s.db.First(&course, *job.CourseID).Error; err != nil {
			log.Printf("[GenerationJobService] ERROR: Course not found: %v", err)
			return fmt.Errorf("course not found: %w", err)
		}
		hasCourse = true
		log.Printf("[GenerationJobService] Course found: %s (%s)", course.CourseName, course.CourseCode)
	} else {
		log.Printf("[GenerationJobService] Step 2: No course ID provided, using default course context")
	}

	// Step 3: Use multi-agent system to process lesson plan
	log.Printf("[GenerationJobService] Step 3: Starting multi-agent processing...")
	if hasCourse {
		log.Printf("[GenerationJobService] Course Name: %s, Level: undergraduate", course.CourseName)
	}

	structure, err := s.orchestrator.ProcessLessonPlanWithAgents(ctx, fileContent, &course)
	if err != nil {
		log.Printf("[GenerationJobService] ERROR: Multi-agent processing failed: %v", err)
		return fmt.Errorf("multi-agent processing failed: %w", err)
	}

	log.Printf("[GenerationJobService] Multi-agent processing completed successfully!")
	log.Printf("[GenerationJobService] Generated structure: %d weeks, %d total modules", len(structure.Weeks), countTotalModules(structure.Weeks))

	// Log the generated structure
	for i, week := range structure.Weeks {
		log.Printf("[GenerationJobService]   Week %d: %s (week_number=%d, modules=%d)",
			i+1, week.WeekName, week.WeekNumber, len(week.Modules))
		for j, module := range week.Modules {
			log.Printf("[GenerationJobService]     Module %d: %s - %s",
				j+1, module.ModuleName, truncateString(module.Description, 100))
		}
	}

	// Step 4: Store result in GeneratedData
	log.Printf("[GenerationJobService] Step 4: Marshaling generated data...")
	generatedData, err := json.Marshal(structure)
	if err != nil {
		log.Printf("[GenerationJobService] ERROR: Failed to marshal generated data: %v", err)
		return fmt.Errorf("failed to marshal generated data: %w", err)
	}

	job.GeneratedData = generatedData
	log.Printf("[GenerationJobService] Generated data stored. Size: %d bytes", len(generatedData))
	log.Printf("[GenerationJobService] ========== END LESSON PLAN PROCESSING ==========")
	return nil
}

// processQuizGeneration processes a quiz generation job
func (s *GenerationJobService) processQuizGeneration(ctx context.Context, job *models.GenerationJob) error {
	// Parse the request from GeneratedData
	var req PracticeQuizRequest
	if err := json.Unmarshal(job.GeneratedData, &req); err != nil {
		return fmt.Errorf("invalid job data: %w", err)
	}

	// Generate practice quiz
	questions, err := s.aiProvider.GeneratePracticeQuiz(ctx, req)
	if err != nil {
		return fmt.Errorf("AI quiz generation failed: %w", err)
	}

	// Store result
	generatedData, err := json.Marshal(questions)
	if err != nil {
		return fmt.Errorf("failed to marshal generated data: %w", err)
	}

	job.GeneratedData = generatedData
	return nil
}

// processContestGeneration processes a contest problem generation job
func (s *GenerationJobService) processContestGeneration(ctx context.Context, job *models.GenerationJob) error {
	log.Printf("[ContestGeneration] Starting contest problem generation for job %d", job.ID)

	// Parse the request data stored in GeneratedData
	var reqData struct {
		DifficultyLevel string   `json:"difficulty_level"`
		Topics          []string `json:"topics"`
		RequestedCount  int      `json:"requested_count"`
	}
	if err := json.Unmarshal(job.GeneratedData, &reqData); err != nil {
		return fmt.Errorf("invalid job data: %w", err)
	}

	// Build the contest generation request
	req := models.ContestGenerationRequest{
		CourseID:        job.CourseID,
		Topics:          reqData.Topics,
		RequestedCount:  reqData.RequestedCount,
		DifficultyLevel: reqData.DifficultyLevel,
	}

	// Use the ContestGenerationService with agentic flow
	contestService := NewContestGenerationService(s.aiProvider, s.orchestrator)
	problems, err := contestService.GenerateContestProblems(ctx, req)
	if err != nil {
		return fmt.Errorf("contest problem generation failed: %w", err)
	}

	log.Printf("[ContestGeneration] Generated %d problems for job %d", len(problems), job.ID)

	// Store generated problems back in job
	generatedData, err := json.Marshal(problems)
	if err != nil {
		return fmt.Errorf("failed to marshal generated problems: %w", err)
	}

	job.GeneratedData = generatedData
	return nil
}

// readFile reads content from the uploaded file.
// For PDF files, it extracts the text content using a PDF parser.
// For other files (txt, md, etc.), it reads the raw content directly.
func (s *GenerationJobService) readFile(filePath string) (string, error) {
	if strings.HasSuffix(strings.ToLower(filePath), ".pdf") {
		return s.readPDFText(filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// readPDFText extracts plain text from a PDF file using ledongthuc/pdf.
func (s *GenerationJobService) readPDFText(filePath string) (string, error) {
	f, r, err := pdfReader.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var textBuilder strings.Builder
	totalPages := r.NumPage()
	log.Printf("[GenerationJobService] PDF has %d pages", totalPages)

	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			log.Printf("[GenerationJobService] Warning: failed to extract text from page %d: %v", pageIndex, err)
			continue
		}
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	extracted := textBuilder.String()
	if strings.TrimSpace(extracted) == "" {
		return "", fmt.Errorf("no text could be extracted from PDF (it may be scanned/image-based)")
	}

	log.Printf("[GenerationJobService] Extracted %d characters from PDF", len(extracted))
	return extracted, nil
}

// SaveContentOptions holds options for saving generated content
type SaveContentOptions struct {
	GenerateQuizzes bool // Whether to generate practice quizzes (default: false to save API calls)
}

// DefaultSaveContentOptions returns default options (quizzes disabled by default)
func DefaultSaveContentOptions() SaveContentOptions {
	return SaveContentOptions{
		GenerateQuizzes: false, // Disabled by default to reduce API calls
	}
}

// SaveGeneratedContent saves AI-generated content to the database
// This is called after admin approves the generated content
// NOTE: This skips quiz generation by default to save API calls
func (s *GenerationJobService) SaveGeneratedContent(job *models.GenerationJob, modifications *models.LessonPlanStructure) error {
	return s.SaveGeneratedContentWithOptions(job, modifications, DefaultSaveContentOptions())
}

// SaveGeneratedContentWithOptions saves AI-generated content to the database with configurable options
// Use this when you want to control quiz generation behavior
func (s *GenerationJobService) SaveGeneratedContentWithOptions(job *models.GenerationJob, modifications *models.LessonPlanStructure, options SaveContentOptions) error {
	if job.Status != models.GenerationJobCompleted {
		return fmt.Errorf("job not completed yet")
	}

	// Parse generated data
	var structure models.LessonPlanStructure
	if modifications != nil {
		// Use modifications if provided
		structure = *modifications
	} else {
		// Use original generated data
		if err := json.Unmarshal(job.GeneratedData, &structure); err != nil {
			return fmt.Errorf("failed to parse generated data: %w", err)
		}
	}

	// Get or create Theory
	var theory models.Theory
	if job.TheoryID != nil {
		if err := s.db.First(&theory, *job.TheoryID).Error; err != nil {
			return fmt.Errorf("theory not found: %w", err)
		}
	} else {
		// Check if theory exists for course
		if job.CourseID == nil {
			return fmt.Errorf("cannot save content without course ID")
		}
		if err := s.db.Where("course_id = ?", *job.CourseID).First(&theory).Error; err != nil {
			// Create new theory
			course := &models.Course{}
			if err := s.db.First(course, *job.CourseID).Error; err != nil {
				return fmt.Errorf("course not found: %w", err)
			}

			theory = models.Theory{
				CourseID:   course.ID,
				TheoryName: course.CourseName,
				TheoryCode: course.CourseCode,
			}
			if err := s.db.Create(&theory).Error; err != nil {
				return fmt.Errorf("failed to create theory: %w", err)
			}
		}
	}

	// Create weeks and modules
	for _, weekData := range structure.Weeks {
		week := models.TheoryWeek{
			TheoryID:  theory.ID,
			WeekName:  weekData.WeekName,
			WeekOrder: weekData.WeekNumber,
		}
		if err := s.db.Create(&week).Error; err != nil {
			return fmt.Errorf("failed to create week %d: %w", weekData.WeekNumber, err)
		}

		// Create modules for this week
		moduleIDMap := make(map[string]uint)
		for _, moduleData := range weekData.Modules {
			module := models.TheoryModule{
				TheoryWeekID: week.ID,
				ModuleName:   moduleData.ModuleName,
				Description:  moduleData.Description,
				Content:      moduleData.Content,
			}
			if err := s.db.Create(&module).Error; err != nil {
				return fmt.Errorf("failed to create module %s: %w", moduleData.ModuleName, err)
			}
			moduleIDMap[moduleData.ModuleName] = module.ID
		}

		// Generate and save practice quiz for this week (only if enabled)
		if options.GenerateQuizzes {
			if err := s.generateAndSavePracticeQuiz(week.ID, weekData, moduleIDMap); err != nil {
				// Log error but continue - practice quiz is not critical
				log.Printf("Warning: failed to generate practice quiz for week %d: %v\n", weekData.WeekNumber, err)
			}
		} else {
			log.Printf("[GenerationJobService] Skipping quiz generation for week %d (disabled to save API calls)", weekData.WeekNumber)
		}
	}

	return nil
}

// generateAndSavePracticeQuiz generates and saves a practice quiz for a week
func (s *GenerationJobService) generateAndSavePracticeQuiz(weekID uint, weekData models.WeekData, moduleIDMap map[string]uint) error {
	// Generate practice quiz using AI
	req := PracticeQuizRequest{
		WeekData:      weekData,
		QuestionCount: 5, // Default, could be configurable
		DifficultyMix: map[string]int{"easy": 2, "medium": 2, "hard": 1},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	questions, err := s.aiProvider.GeneratePracticeQuiz(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to generate practice quiz: %w", err)
	}

	// Create practice quiz record
	practiceQuiz := models.PracticeQuiz{
		TheoryWeekID: weekID,
		Title:        fmt.Sprintf("Week %d Practice Quiz", weekData.WeekNumber),
		Instructions: "This practice quiz is for self-assessment. Select the best answer for each question.",
	}
	if err := s.db.Create(&practiceQuiz).Error; err != nil {
		return fmt.Errorf("failed to create practice quiz: %w", err)
	}

	// Convert and save questions
	for i, q := range questions {
		question := models.PracticeQuizQuestion{
			PracticeQuizID: practiceQuiz.ID,
			QuestionType:   models.QuestionType(q.QuestionType),
			QuestionText:   q.QuestionText,
			Explanation:    q.Explanation,
			Difficulty:     models.QuestionDifficulty(q.Difficulty),
			OrderIndex:     i,
			Marks:          1,
		}

		// Map source module
		if moduleID, ok := moduleIDMap[q.SourceModuleName]; ok {
			question.SourceModuleID = &moduleID
		}

		// Convert options
		for _, opt := range q.Options {
			question.Options = append(question.Options, models.PracticeQuizOption{
				OptionText: opt.OptionText,
				IsCorrect:  opt.IsCorrect,
				OrderIndex: opt.OrderIndex,
			})
		}

		if err := s.db.Create(&question).Error; err != nil {
			return fmt.Errorf("failed to create question %d: %w", i, err)
		}
	}

	// Update total marks
	totalMarks := len(questions)
	practiceQuiz.TotalMarks = totalMarks
	s.db.Save(&practiceQuiz)

	return nil
}

// GeneratePracticeQuizForWeek generates a practice quiz for a specific week on demand
// This is called when faculty explicitly requests quiz generation (lazy/on-demand approach)
// Returns the created practice quiz ID
func (s *GenerationJobService) GeneratePracticeQuizForWeek(ctx context.Context, theoryWeekID uint, questionCount int) (uint, error) {
	log.Printf("[GenerationJobService] Generating practice quiz for week %d (on-demand)", theoryWeekID)

	// Get the theory week with modules
	var week models.TheoryWeek
	if err := s.db.Preload("Modules").First(&week, theoryWeekID).Error; err != nil {
		return 0, fmt.Errorf("theory week not found: %w", err)
	}

	// Check if a quiz already exists for this week
	var existingQuiz models.PracticeQuiz
	if err := s.db.Where("theory_week_id = ?", theoryWeekID).First(&existingQuiz).Error; err == nil {
		log.Printf("[GenerationJobService] Quiz already exists for week %d, returning existing quiz ID %d", theoryWeekID, existingQuiz.ID)
		return existingQuiz.ID, nil
	}

	// Build week data from modules
	weekData := models.WeekData{
		WeekNumber: week.WeekOrder,
		WeekName:   week.WeekName,
	}

	moduleIDMap := make(map[string]uint)
	for _, module := range week.Modules {
		weekData.Modules = append(weekData.Modules, models.ModuleData{
			ModuleName:  module.ModuleName,
			Description: module.Description,
			Content:     module.Content,
		})
		moduleIDMap[module.ModuleName] = module.ID
	}

	// Set default question count
	if questionCount <= 0 {
		questionCount = 5
	}

	// Generate practice quiz using AI
	req := PracticeQuizRequest{
		WeekData:      weekData,
		QuestionCount: questionCount,
		DifficultyMix: map[string]int{"easy": 2, "medium": 2, "hard": 1},
	}

	questions, err := s.aiProvider.GeneratePracticeQuiz(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("failed to generate practice quiz: %w", err)
	}

	// Create practice quiz record
	practiceQuiz := models.PracticeQuiz{
		TheoryWeekID: theoryWeekID,
		Title:        fmt.Sprintf("Week %d Practice Quiz", week.WeekOrder),
		Instructions: "This practice quiz is for self-assessment. Select the best answer for each question.",
	}
	if err := s.db.Create(&practiceQuiz).Error; err != nil {
		return 0, fmt.Errorf("failed to create practice quiz: %w", err)
	}

	// Convert and save questions
	for i, q := range questions {
		question := models.PracticeQuizQuestion{
			PracticeQuizID: practiceQuiz.ID,
			QuestionType:   models.QuestionType(q.QuestionType),
			QuestionText:   q.QuestionText,
			Explanation:    q.Explanation,
			Difficulty:     models.QuestionDifficulty(q.Difficulty),
			OrderIndex:     i,
			Marks:          1,
		}

		// Map source module
		if moduleID, ok := moduleIDMap[q.SourceModuleName]; ok {
			question.SourceModuleID = &moduleID
		}

		// Convert options
		for _, opt := range q.Options {
			question.Options = append(question.Options, models.PracticeQuizOption{
				OptionText: opt.OptionText,
				IsCorrect:  opt.IsCorrect,
				OrderIndex: opt.OrderIndex,
			})
		}

		if err := s.db.Create(&question).Error; err != nil {
			log.Printf("[GenerationJobService] Warning: failed to create question %d: %v", i, err)
		}
	}

	// Update total marks
	practiceQuiz.TotalMarks = len(questions)
	s.db.Save(&practiceQuiz)

	log.Printf("[GenerationJobService] Practice quiz created successfully for week %d, quiz ID: %d", theoryWeekID, practiceQuiz.ID)
	return practiceQuiz.ID, nil
}

// GenerateAllPracticeQuizzes generates practice quizzes for all weeks in a theory
// This is a batch operation that generates quizzes on demand
func (s *GenerationJobService) GenerateAllPracticeQuizzes(ctx context.Context, theoryID uint, questionCountPerWeek int) ([]uint, error) {
	log.Printf("[GenerationJobService] Generating practice quizzes for all weeks in theory %d", theoryID)

	// Get all weeks for the theory
	var weeks []models.TheoryWeek
	if err := s.db.Where("theory_id = ?", theoryID).Find(&weeks).Error; err != nil {
		return nil, fmt.Errorf("failed to get weeks: %w", err)
	}

	var quizIDs []uint
	for _, week := range weeks {
		quizID, err := s.GeneratePracticeQuizForWeek(ctx, week.ID, questionCountPerWeek)
		if err != nil {
			log.Printf("[GenerationJobService] Warning: failed to generate quiz for week %d: %v", week.ID, err)
			continue
		}
		quizIDs = append(quizIDs, quizID)

		// Add delay between quiz generations to avoid rate limits
		time.Sleep(2 * time.Second)
	}

	log.Printf("[GenerationJobService] Generated %d practice quizzes for theory %d", len(quizIDs), theoryID)
	return quizIDs, nil
}

// DeleteUploadedFile removes the uploaded file after processing
func (s *GenerationJobService) DeleteUploadedFile(filePath string) error {
	if filePath == "" {
		return nil
	}
	return os.Remove(filePath)
}

// CleanupOldJobs removes old completed jobs and their files
func (s *GenerationJobService) CleanupOldJobs(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	var jobs []models.GenerationJob
	if err := s.db.Where("status IN ? AND completed_at < ?",
		[]models.GenerationJobStatus{models.GenerationJobCompleted, models.GenerationJobFailed},
		cutoff).Find(&jobs).Error; err != nil {
		return err
	}

	for _, job := range jobs {
		// Delete file if exists
		if job.InputFilePath != "" {
			os.Remove(job.InputFilePath)
		}
		// Delete job record
		s.db.Delete(&job)
	}

	return nil
}

// ValidateFileForUpload validates an uploaded file
func ValidateFileForUpload(file *multipart.FileHeader) error {
	allowedTypes := []string{"pdf", "docx", "txt"}
	maxSizeMB := 50

	return utils.ValidateFile(file, allowedTypes, maxSizeMB)
}

// Helper function for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// SaveUploadedFile saves an uploaded file to the uploads directory
func SaveUploadedFile(file *multipart.FileHeader, subdir string) (string, error) {
	uploadPath := filepath.Join("./uploads", subdir)
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), utils.SanitizeFilename(file.Filename))
	filePath := filepath.Join(uploadPath, filename)

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		os.Remove(filePath)
		return "", err
	}

	return filePath, nil
}
