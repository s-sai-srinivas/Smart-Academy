package services

import (
	"coding-platform/models"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

// =====================================================
// Lesson Plan Processing Types
// =====================================================

// LessonPlanRequest contains parameters for lesson plan parsing
type LessonPlanRequest struct {
	DocumentContent string `json:"document_content"`
	CourseName      string `json:"course_name"`
	CourseLevel     string `json:"course_level"` // e.g., "undergraduate", "graduate"
}

// PracticeQuizRequest contains parameters for practice quiz generation
type PracticeQuizRequest struct {
	WeekData      models.WeekData `json:"week_data"`
	QuestionCount int             `json:"question_count"`
	DifficultyMix map[string]int  `json:"difficulty_mix"`
}

// =====================================================
// Extend AIProvider Interface
// =====================================================

// AIProvider extends the interface from ai_quiz.go
// Original interface already includes:
// - GenerateQuizQuestions(ctx context.Context, req QuizGenerationRequest) ([]GeneratedQuestion, error)
// - EstimateTokens(content string) int

// Add new methods for lesson plan processing:
type LessonPlanAIProvider interface {
	// ParseLessonPlan parses a lesson plan document and extracts structured content
	ParseLessonPlan(ctx context.Context, req LessonPlanRequest) (*models.LessonPlanStructure, error)
	// GeneratePracticeQuiz generates a practice quiz for a given week
	GeneratePracticeQuiz(ctx context.Context, req PracticeQuizRequest) ([]models.PracticeQuizQuestionData, error)
}

// =====================================================
// Gemini Provider - Lesson Plan Methods
// =====================================================

// ParseLessonPlan parses lesson plan content and extracts week/module structure
func (p *GeminiProvider) ParseLessonPlan(ctx context.Context, req LessonPlanRequest) (*models.LessonPlanStructure, error) {
	prompt := p.buildLessonPlanPrompt(req)

	// Lesson plan generation needs a much higher token limit than the global default
	// because the output JSON contains structured content for every module across all weeks.
	lessonPlanMaxTokens := 65536
	if p.maxTokens > lessonPlanMaxTokens {
		lessonPlanMaxTokens = p.maxTokens
	}

	// Build Gemini request body
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  lessonPlanMaxTokens,
			"temperature":      p.temperature,
			"responseMimeType": "application/json",
		},
		"safetySettings": []map[string]string{
			{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	log.Printf("[LessonPlanParser] Using maxOutputTokens=%d for lesson plan generation", lessonPlanMaxTokens)

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Gemini request: %w", err)
	}

	// Use header for API key (more secure than query string)
	body, _, err := p.doRequest(ctx, jsonData)
	if err != nil {
		return nil, err
	}

	structure, err := p.parseLessonPlanResponse(body)
	if err != nil {
		return nil, err
	}

	return structure, nil
}

// GeneratePracticeQuiz generates practice quiz questions for a week
func (p *GeminiProvider) GeneratePracticeQuiz(ctx context.Context, req PracticeQuizRequest) ([]models.PracticeQuizQuestionData, error) {
	prompt := p.buildPracticeQuizPrompt(req)

	// Build Gemini request body
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  4096, // Reduced for quiz generation
			"temperature":      0.7,
			"responseMimeType": "application/json",
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Gemini request: %w", err)
	}

	// Use header for API key (more secure than query string)
	body, _, err := p.doRequest(ctx, jsonData)
	if err != nil {
		return nil, err
	}

	questions, err := p.parsePracticeQuizResponse(body)
	if err != nil {
		return nil, err
	}

	return questions, nil
}

// ParseLessonPlanWithFeedback parses lesson plan with critic feedback for improvement
func (p *GeminiProvider) ParseLessonPlanWithFeedback(ctx context.Context, req LessonPlanRequest, suggestions []string) (*models.LessonPlanStructure, error) {
	prompt := p.buildLessonPlanPromptWithFeedback(req, suggestions)

	// Build Gemini request body
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  p.maxTokens,
			"temperature":      p.temperature,
			"responseMimeType": "application/json",
		},
		"safetySettings": []map[string]string{
			{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Gemini request: %w", err)
	}

	// Use header for API key (more secure than query string)
	body, _, err := p.doRequest(ctx, jsonData)
	if err != nil {
		return nil, err
	}

	structure, err := p.parseLessonPlanResponse(body)
	if err != nil {
		return nil, err
	}

	return structure, nil
}

// buildLessonPlanPromptWithFeedback creates prompt with critic feedback
func (p *GeminiProvider) buildLessonPlanPromptWithFeedback(req LessonPlanRequest, suggestions []string) string {
	basePrompt := p.buildLessonPlanPrompt(req)

	var sb strings.Builder
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n=== CRITIC FEEDBACK (IMPORTANT - MUST ADDRESS) ===\n")
	sb.WriteString("The previous generation had the following issues. Please create an improved version addressing these points:\n\n")
	for i, suggestion := range suggestions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, suggestion))
	}
	sb.WriteString("\nGenerate a complete, improved lesson plan that addresses ALL the above points.")

	return sb.String()
}

// =====================================================
// Prompt Building Methods
// =====================================================

// buildLessonPlanPrompt creates the prompt for lesson plan parsing
func (p *GeminiProvider) buildLessonPlanPrompt(req LessonPlanRequest) string {
	var sb strings.Builder

	sb.WriteString("## ROLE\n")
	sb.WriteString("You are an educational content analyzer specializing in parsing lesson plans ")
	sb.WriteString("and extracting structured week-by-week content for online course platforms.\n\n")

	sb.WriteString("## INPUT\n")
	sb.WriteString(fmt.Sprintf("Course: %s\n", req.CourseName))
	sb.WriteString(fmt.Sprintf("Level: %s\n\n", req.CourseLevel))
	sb.WriteString("Lesson Plan Content:\n")
	sb.WriteString("--- BEGIN DOCUMENT ---\n")
	sb.WriteString(req.DocumentContent)
	sb.WriteString("\n--- END DOCUMENT ---\n\n")

	sb.WriteString("## IMPORTANT: DATE TO WEEK MAPPING\n")
	sb.WriteString("The document contains DATES but NOT explicit week numbers.\n")
	sb.WriteString("You MUST derive week numbers from dates:\n")
	sb.WriteString("1. Find the earliest date - this is Week 1\n")
	sb.WriteString("2. Group dates into weekly intervals (7 days each)\n")
	sb.WriteString("3. Assign sequential week numbers (1, 2, 3, ...)\n")
	sb.WriteString("4. Use topics associated with each date range for module content\n\n")

	sb.WriteString("## TASK\n")
	sb.WriteString("Analyze the lesson plan and extract structured content:\n")
	sb.WriteString("1. Extract all dates and associated topics from the document\n")
	sb.WriteString("2. Group dates into weeks (each week = 7 days starting from first date)\n")
	sb.WriteString("3. For each week, identify 2-5 modules/topics based on the topics listed\n")
	sb.WriteString("4. For each module, create comprehensive educational content\n")
	sb.WriteString("5. Generate practice quiz questions for each week\n\n")

	sb.WriteString("## HANDLING TABLES\n")
	sb.WriteString("The input contains tables with DATE and TOPIC columns.\n")
	sb.WriteString("Example format you might see:\n")
	sb.WriteString("| Date       | Topic                    |\n")
	sb.WriteString("|------------|-------------------------|\n")
	sb.WriteString("| 2024-01-15 | Introduction to XYZ     |\n")
	sb.WriteString("| 2024-01-22 | Advanced Concepts       |\n")
	sb.WriteString("\nFor such tables:\n")
	sb.WriteString("- Group consecutive dates (within 7 days) into the same week\n")
	sb.WriteString("- Use the topic names as module names\n")
	sb.WriteString("- Expand each topic into full educational content\n")
	sb.WriteString("- Assign week_number sequentially starting from 1\n\n")

	sb.WriteString("## OUTPUT FORMAT\n")
	sb.WriteString("Return a JSON object with this EXACT structure:\n")
	sb.WriteString(`{
    "course_name": "Course Name from document",
    "total_weeks": 10,
    "weeks": [
        {
            "week_number": 1,
            "week_name": "Week 1: Introduction to Topic (Jan 15-21)",
            "modules": [
                {
                    "module_name": "Module Title from document",
                    "description": "2-3 sentence summary of what this module covers",
                    "content": "Concise markdown content (50-150 words) covering key points, core concepts, and brief examples"
                }
            ]
        }
    ]
}

CRITICAL RULES:
1. MUST have at least 1 week in the "weeks" array - empty weeks array means FAILURE
2. MUST derive week numbers from dates (Week 1 starts from earliest date)
3. Content MUST be in markdown format
4. Include code examples where relevant
5. Each module should have 50-150 words of concise educational content (key points and summary)
6. Maintain academic tone appropriate for ` + req.CourseLevel + ` students
7. Extract topic names from the document - do not make up generic names
8. If a date has multiple topics, create separate modules for each
9. Include week dates in week_name for clarity (e.g., "Week 1: Introduction (Jan 15-21)")

IMPORTANT: Return ONLY valid JSON. No markdown code fences. Response MUST start with { and end with }.`)

	return sb.String()
}

// buildPracticeQuizPrompt creates the prompt for practice quiz generation
func (p *GeminiProvider) buildPracticeQuizPrompt(req PracticeQuizRequest) string {
	var sb strings.Builder

	sb.WriteString("## ROLE\n")
	sb.WriteString("You are an educational assessment designer creating practice quizzes ")
	sb.WriteString("for student self-assessment and learning reinforcement.\n\n")

	sb.WriteString("## WEEK CONTENT\n")
	sb.WriteString(fmt.Sprintf("Week %d: %s\n\n", req.WeekData.WeekNumber, req.WeekData.WeekName))

	for i, module := range req.WeekData.Modules {
		sb.WriteString(fmt.Sprintf("### Module %d: %s\n", i+1, module.ModuleName))
		sb.WriteString(fmt.Sprintf("Description: %s\n", module.Description))
		sb.WriteString(fmt.Sprintf("Content: %s\n\n", module.Content))
	}

	sb.WriteString("## TASK\n")
	sb.WriteString(fmt.Sprintf("Generate %d practice quiz questions covering all modules in this week.\n\n", req.QuestionCount))

	if len(req.DifficultyMix) > 0 {
		sb.WriteString("## DIFFICULTY DISTRIBUTION\n")
		for diff, count := range req.DifficultyMix {
			sb.WriteString(fmt.Sprintf("- %s: %d questions\n", diff, count))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## OUTPUT FORMAT\n")
	sb.WriteString("Return a JSON array with this exact structure:\n")
	sb.WriteString(`[
    {
        "question_text": "Clear, unambiguous question text",
        "question_type": "mcq",
        "options": [
            {"option_text": "Option A", "is_correct": false, "order_index": 0},
            {"option_text": "Option B", "is_correct": true, "order_index": 1},
            {"option_text": "Option C", "is_correct": false, "order_index": 2},
            {"option_text": "Option D", "is_correct": false, "order_index": 3}
        ],
        "explanation": "Detailed explanation of why the correct answer is right and why other options are wrong",
        "difficulty": "easy",
        "source_module_name": "Module Name this question tests"
    }
]

QUESTION TYPES:
- mcq: Multiple choice with 4 options, exactly one correct
- true_false: 2 options ("True" and "False")
- fill_blank: Sentence with a blank, options are possible words/phrases

RULES:
1. Questions should cover all modules proportionally
2. Include detailed explanations (2-4 sentences) for every question
3. Mix cognitive levels: recall, understanding, application
4. For MCQs, make distractors plausible but clearly incorrect
5. Avoid "all of the above" or "none of the above" options
6. Reference specific content from the modules in explanations
7. Questions should be self-contained and clear without module context

IMPORTANT: Return ONLY valid JSON array. No markdown code fences, no additional text.`)

	return sb.String()
}

// =====================================================
// Response Parsing Methods
// =====================================================

// parseLessonPlanResponse parses the Gemini API response for lesson plan parsing
func (p *GeminiProvider) parseLessonPlanResponse(body []byte) (*models.LessonPlanStructure, error) {
	// Log raw body for debugging
	log.Printf("[LessonPlanParser] Raw API response: %s...", string(body)[:min(300, len(body))])

	// Parse Gemini response structure
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("error parsing Gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s (code: %d)", geminiResp.Error.Message, geminiResp.Error.Code)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in Gemini response")
	}

	// Check if response was truncated due to token limit
	finishReason := geminiResp.Candidates[0].FinishReason
	log.Printf("[LessonPlanParser] Gemini finishReason: %s", finishReason)
	if finishReason == "MAX_TOKENS" {
		log.Printf("[LessonPlanParser] WARNING: Response was truncated due to maxOutputTokens limit! JSON will be incomplete.")
		return nil, fmt.Errorf("LLM response was truncated (hit token limit). The lesson plan document is too large for a single request. Try uploading a shorter document or increase AI_MAX_TOKENS")
	}

	responseContent := geminiResp.Candidates[0].Content.Parts[0].Text

	// Log raw response for debugging
	log.Printf("[LessonPlanParser] Raw response length: %d chars", len(responseContent))
	if len(responseContent) > 500 {
		log.Printf("[LessonPlanParser] Raw response preview: %s...", responseContent[:500])
	} else {
		log.Printf("[LessonPlanParser] Raw response: %s", responseContent)
	}

	// Clean the response
	responseContent = cleanJSONResponse(responseContent)

	// Try to find JSON object in response (in case there's extra text)
	jsonStart := strings.Index(responseContent, "{")
	jsonEnd := strings.LastIndex(responseContent, "}")
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		responseContent = responseContent[jsonStart : jsonEnd+1]
	}

	log.Printf("[LessonPlanParser] JSON to parse: %s...", responseContent[:min(300, len(responseContent))])

	// Parse the JSON structure
	var structure models.LessonPlanStructure
	if err := json.Unmarshal([]byte(responseContent), &structure); err != nil {
		// Log the JSON that failed to parse
		log.Printf("[LessonPlanParser] JSON parse error: %v", err)
		log.Printf("[LessonPlanParser] Invalid JSON: %s", responseContent[:min(500, len(responseContent))])
		return nil, fmt.Errorf("error parsing JSON from Gemini response: %w", err)
	}

	// Validate structure
	if len(structure.Weeks) == 0 {
		log.Printf("[LessonPlanParser] Parsed structure has no weeks. Full structure: %+v", structure)
		return nil, fmt.Errorf("no weeks found in parsed lesson plan")
	}

	totalModules := 0
	for i, week := range structure.Weeks {
		totalModules += len(week.Modules)
		if len(week.Modules) == 0 {
			log.Printf("Warning: Week %d (%s) has no modules", i+1, week.WeekName)
		}
	}

	log.Printf("[LessonPlanParser] Successfully parsed %d weeks with %d total modules", len(structure.Weeks), totalModules)
	return &structure, nil
}

// parsePracticeQuizResponse parses the Gemini API response for practice quiz generation
func (p *GeminiProvider) parsePracticeQuizResponse(body []byte) ([]models.PracticeQuizQuestionData, error) {
	// Parse Gemini response structure
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("error parsing Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in Gemini response")
	}

	responseContent := geminiResp.Candidates[0].Content.Parts[0].Text

	// Clean the response
	responseContent = cleanJSONResponse(responseContent)

	// Parse the JSON array
	var questions []models.PracticeQuizQuestionData
	if err := json.Unmarshal([]byte(responseContent), &questions); err != nil {
		return nil, fmt.Errorf("error parsing JSON from Gemini response: %w", err)
	}

	// Validate questions
	var validated []models.PracticeQuizQuestionData
	for i, q := range questions {
		if strings.TrimSpace(q.QuestionText) == "" {
			log.Printf("Skipping question %d: empty question text", i)
			continue
		}

		// Validate options for MCQ/TF
		if q.QuestionType == "mcq" || q.QuestionType == "true_false" {
			if len(q.Options) < 2 {
				log.Printf("Skipping question %d: insufficient options", i)
				continue
			}

			hasCorrect := false
			for _, opt := range q.Options {
				if opt.IsCorrect {
					hasCorrect = true
					break
				}
			}
			if !hasCorrect {
				log.Printf("Skipping question %d: no correct answer marked", i)
				continue
			}
		}

		// Set default difficulty if not specified
		if q.Difficulty == "" {
			q.Difficulty = "medium"
		}

		validated = append(validated, q)
	}

	if len(validated) == 0 {
		return nil, fmt.Errorf("no valid questions generated")
	}

	return validated, nil
}

// =====================================================
// Phase 1: Extract Structure Only (Lightweight)
// =====================================================

// ParseLessonPlanStructureOnly extracts only the week/module structure without content
// This is Phase 1 of the two-phase lesson plan parsing
// Output: ~500-2000 tokens instead of ~10000+ tokens
func (p *GeminiProvider) ParseLessonPlanStructureOnly(ctx context.Context, req LessonPlanRequest) (*models.LessonPlanStructureLite, error) {
	prompt := p.buildLessonPlanStructurePrompt(req)

	// Structure extraction needs enough tokens for the full structure
	// For large courses (16+ weeks with 5+ modules each), we need more tokens
	// Use the model's max tokens or a reasonable default (8192 is safe for most cases)
	structureMaxTokens := 8192
	if p.maxTokens > structureMaxTokens {
		structureMaxTokens = p.maxTokens
	}

	// Build Gemini request body
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  structureMaxTokens,
			"temperature":      0.3, // More deterministic for extraction
			"responseMimeType": "application/json",
		},
		"safetySettings": []map[string]string{
			{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	log.Printf("[LessonPlanStructure] Using maxOutputTokens=%d for structure extraction", structureMaxTokens)

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Gemini request: %w", err)
	}

	// Use header for API key (more secure than query string)
	body, _, err := p.doRequest(ctx, jsonData)
	if err != nil {
		return nil, err
	}

	structure, err := p.parseLessonPlanStructureLiteResponse(body)
	if err != nil {
		return nil, err
	}

	log.Printf("[LessonPlanStructure] Extracted %d weeks with %d total modules",
		len(structure.Weeks), countTotalModuleNames(structure.Weeks))
	return structure, nil
}

// parseLessonPlanStructureLiteResponse parses the lite structure response
func (p *GeminiProvider) parseLessonPlanStructureLiteResponse(body []byte) (*models.LessonPlanStructureLite, error) {
	log.Printf("[LessonPlanStructure] Raw API response: %s...", string(body)[:min(300, len(body))])

	// Parse Gemini response structure
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("error parsing Gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s (code: %d)", geminiResp.Error.Message, geminiResp.Error.Code)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in Gemini response")
	}

	// Check if response was truncated due to token limit
	finishReason := geminiResp.Candidates[0].FinishReason
	log.Printf("[LessonPlanStructure] Gemini finishReason: %s", finishReason)
	if finishReason == "MAX_TOKENS" {
		log.Printf("[LessonPlanStructure] WARNING: Response was truncated due to maxOutputTokens limit! JSON will be incomplete.")
		return nil, fmt.Errorf("LLM response was truncated (hit token limit). The lesson plan document is too large. Try uploading a shorter document or contact support")
	}

	responseContent := geminiResp.Candidates[0].Content.Parts[0].Text

	// Clean the response
	responseContent = cleanJSONResponse(responseContent)

	// Try to find JSON object in response
	jsonStart := strings.Index(responseContent, "{")
	jsonEnd := strings.LastIndex(responseContent, "}")
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		responseContent = responseContent[jsonStart : jsonEnd+1]
	}

	log.Printf("[LessonPlanStructure] JSON to parse: %s...", responseContent[:min(300, len(responseContent))])

	// Parse the JSON structure
	var structure models.LessonPlanStructureLite
	if err := json.Unmarshal([]byte(responseContent), &structure); err != nil {
		log.Printf("[LessonPlanStructure] JSON parse error: %v", err)
		log.Printf("[LessonPlanStructure] Invalid JSON (length=%d): %s", len(responseContent), responseContent[:min(500, len(responseContent))])

		// Try to repair truncated JSON
		repairedJSON := repairTruncatedJSON(responseContent)
		if repairedJSON != responseContent {
			log.Printf("[LessonPlanStructure] Attempting repair with: %s...", repairedJSON[:min(200, len(repairedJSON))])
			if repairErr := json.Unmarshal([]byte(repairedJSON), &structure); repairErr == nil {
				log.Printf("[LessonPlanStructure] JSON repair successful!")
				if len(structure.Weeks) > 0 {
					return &structure, nil
				}
			}
		}

		return nil, fmt.Errorf("error parsing JSON from Gemini response: %w", err)
	}

	// Validate structure
	if len(structure.Weeks) == 0 {
		log.Printf("[LessonPlanStructure] Parsed structure has no weeks. Full structure: %+v", structure)
		return nil, fmt.Errorf("no weeks found in parsed lesson plan")
	}

	totalModules := 0
	for i, week := range structure.Weeks {
		totalModules += len(week.ModuleNames)
		if len(week.ModuleNames) == 0 {
			log.Printf("Warning: Week %d (%s) has no modules", i+1, week.WeekName)
		}
	}

	log.Printf("[LessonPlanStructure] Successfully parsed %d weeks with %d total module names", len(structure.Weeks), totalModules)
	return &structure, nil
}

// =====================================================
// Phase 2: Generate Module Content (per module)
// =====================================================

// GenerateModuleContent generates content for a single module
// This is Phase 2 - called for each module extracted in Phase 1
// Output: ~500-1500 tokens per module
func (p *GeminiProvider) GenerateModuleContent(ctx context.Context, courseName, courseLevel, moduleName, moduleDescription string, weekContext string) (models.ModuleData, error) {
	prompt := p.buildModuleContentPrompt(courseName, courseLevel, moduleName, moduleDescription, weekContext)

	// Module content generation needs moderate tokens
	moduleMaxTokens := 4096

	// Build Gemini request body
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  moduleMaxTokens,
			"temperature":      0.5, // Some creativity for content
			"responseMimeType": "application/json",
		},
		"safetySettings": []map[string]string{
			{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return models.ModuleData{}, fmt.Errorf("error marshaling Gemini request: %w", err)
	}

	body, _, err := p.doRequest(ctx, jsonData)
	if err != nil {
		return models.ModuleData{}, err
	}

	moduleData, err := p.parseModuleContentResponse(body)
	if err != nil {
		return models.ModuleData{}, err
	}

	return moduleData, nil
}

// GenerateModuleContentsBatch generates content for multiple modules using BATCHING
// This reduces API calls from N modules to ~N/batchSize calls
// Default batch size is 4 modules per API call
func (p *GeminiProvider) GenerateModuleContentsBatch(ctx context.Context, courseName, courseLevel string, modules []ModuleContentRequest, _ int) ([]models.ModuleData, error) {
	if len(modules) == 0 {
		return []models.ModuleData{}, nil
	}

	// Configuration for batching
	const batchSize = 4 // Generate 4 modules per API call

	log.Printf("[ModuleContentGen] Starting BATCH generation for %d modules (batch size: %d)", len(modules), batchSize)

	var allResults []models.ModuleData

	// Process modules in batches
	for batchStart := 0; batchStart < len(modules); batchStart += batchSize {
		select {
		case <-ctx.Done():
			return allResults, ctx.Err()
		default:
		}

		batchEnd := batchStart + batchSize
		if batchEnd > len(modules) {
			batchEnd = len(modules)
		}

		batch := modules[batchStart:batchEnd]
		batchNum := (batchStart / batchSize) + 1
		totalBatches := (len(modules) + batchSize - 1) / batchSize

		log.Printf("[ModuleContentGen] Processing batch %d/%d (%d modules)", batchNum, totalBatches, len(batch))

		// Use rate limiter before API call
		estimatedTokens := 8000 // Estimate for batch generation
		if p.rateLimiter != nil {
			if wait, err := p.rateLimiter.WaitForSlot(ctx, estimatedTokens); err != nil {
				return allResults, fmt.Errorf("rate limiter error: %w", err)
			} else if wait > 0 {
				log.Printf("[ModuleContentGen] Rate limiter caused wait of %v", wait)
			}
		}

		// Generate content for the batch
		batchResults, err := p.GenerateBatchModuleContent(ctx, courseName, courseLevel, batch)
		if err != nil {
			// Check if it's a rate limit error - use retry
			if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit") {
				log.Printf("[ModuleContentGen] Rate limit hit, retrying batch %d after delay", batchNum)
				time.Sleep(60 * time.Second) // Wait before retry

				// Retry once
				batchResults, err = p.GenerateBatchModuleContent(ctx, courseName, courseLevel, batch)
				if err != nil {
					log.Printf("[ModuleContentGen] Batch %d failed after retry: %v", batchNum, err)
					// Continue with other batches instead of failing completely
					continue
				}
			} else {
				log.Printf("[ModuleContentGen] Batch %d failed: %v", batchNum, err)
				continue
			}
		}

		allResults = append(allResults, batchResults...)

		// Add delay between batches to avoid rate limits (only if rate limiter didn't already wait)
		if batchEnd < len(modules) && p.rateLimiter == nil {
			delay := 3 * time.Second
			log.Printf("[ModuleContentGen] Waiting %v before next batch", delay)
			select {
			case <-ctx.Done():
				return allResults, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	log.Printf("[ModuleContentGen] Completed: %d/%d modules generated successfully in %d API calls",
		len(allResults), len(modules), (len(modules)+batchSize-1)/batchSize)
	return allResults, nil
}

// GenerateBatchModuleContent generates content for multiple modules in a SINGLE API call
// This is the key optimization - reduces N API calls to 1 call per batch
func (p *GeminiProvider) GenerateBatchModuleContent(ctx context.Context, courseName, courseLevel string, modules []ModuleContentRequest) ([]models.ModuleData, error) {
	if len(modules) == 0 {
		return []models.ModuleData{}, nil
	}

	prompt := p.buildBatchModuleContentPrompt(courseName, courseLevel, modules)

	// Batch generation needs more tokens since we're generating multiple modules
	batchMaxTokens := 8192 // Enough for 4 modules with content

	// Build Gemini request body
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  batchMaxTokens,
			"temperature":      0.5, // Some creativity for content
			"responseMimeType": "application/json",
		},
		"safetySettings": []map[string]string{
			{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Gemini request: %w", err)
	}

	// Use header for API key (more secure than query string)
	body, _, err := p.doRequest(ctx, jsonData)
	if err != nil {
		return nil, err
	}

	moduleData, err := p.parseBatchModuleContentResponse(body)
	if err != nil {
		return nil, err
	}

	return moduleData, nil
}

// buildBatchModuleContentPrompt creates the prompt for batch module content generation
func (p *GeminiProvider) buildBatchModuleContentPrompt(courseName, courseLevel string, modules []ModuleContentRequest) string {
	var sb strings.Builder

	sb.WriteString("## ROLE\n")
	sb.WriteString("You are an educational content creator specializing in writing\n")
	sb.WriteString("concise, clear educational content for online courses.\n\n")

	sb.WriteString("## CONTEXT\n")
	sb.WriteString(fmt.Sprintf("Course: %s (%s)\n\n", courseName, courseLevel))

	sb.WriteString("## MODULES TO GENERATE\n")
	sb.WriteString(fmt.Sprintf("Generate content for ALL %d modules below. You MUST generate content for EACH module.\n\n", len(modules)))

	for i, module := range modules {
		sb.WriteString(fmt.Sprintf("### Module %d: %s\n", i+1, module.ModuleName))
		if module.WeekContext != "" {
			sb.WriteString(fmt.Sprintf("Week: %s\n", module.WeekContext))
		}
		if module.Description != "" {
			sb.WriteString(fmt.Sprintf("Context: %s\n", module.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## TASK\n")
	sb.WriteString("Generate educational content for EACH of the modules above.\n")
	sb.WriteString("The content for each module should be:\n")
	sb.WriteString("- Concise: 50-150 words (key points and summary)\n")
	sb.WriteString("- In markdown format with headings, lists, and code examples where relevant\n")
	sb.WriteString("- Appropriate for undergraduate students\n")
	sb.WriteString("- Self-contained and clear\n\n")

	sb.WriteString("## OUTPUT FORMAT\n")
	sb.WriteString(fmt.Sprintf("Return a JSON object with a 'modules' array containing EXACTLY %d modules:\n", len(modules)))
	sb.WriteString(`{
    "modules": [
        {
            "module_name": "Module name (same as input)",
            "description": "2-3 sentence summary of what this module covers",
            "content": "Markdown content with key points, concepts, and brief examples"
        }
    ]
}

CRITICAL RULES:
1. You MUST generate content for ALL modules listed above
2. Each module_name must match the input module name exactly
3. The 'modules' array must have EXACTLY the same number of modules as the input
4. Do not skip any modules
5. Content must be in markdown format

IMPORTANT: Return ONLY valid JSON. No markdown code fences.`)

	return sb.String()
}

// parseBatchModuleContentResponse parses the response for batch module content generation
func (p *GeminiProvider) parseBatchModuleContentResponse(body []byte) ([]models.ModuleData, error) {
	// Parse Gemini response structure
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("error parsing Gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s (code: %d)", geminiResp.Error.Message, geminiResp.Error.Code)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in Gemini response")
	}

	responseContent := geminiResp.Candidates[0].Content.Parts[0].Text

	// Clean the response
	responseContent = cleanJSONResponse(responseContent)

	// Try to find JSON object in response
	jsonStart := strings.Index(responseContent, "{")
	jsonEnd := strings.LastIndex(responseContent, "}")
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		responseContent = responseContent[jsonStart : jsonEnd+1]
	}

	// Parse the batch response structure
	var batchResponse struct {
		Modules []models.ModuleData `json:"modules"`
	}

	if err := json.Unmarshal([]byte(responseContent), &batchResponse); err != nil {
		log.Printf("[ModuleContentGen] JSON parse error: %v", err)
		log.Printf("[ModuleContentGen] Invalid JSON: %s", responseContent[:min(500, len(responseContent))])

		// Try to repair truncated JSON
		repairedJSON := repairTruncatedJSON(responseContent)
		if repairedJSON != responseContent {
			log.Printf("[ModuleContentGen] Attempting JSON repair...")
			if repairErr := json.Unmarshal([]byte(repairedJSON), &batchResponse); repairErr == nil && len(batchResponse.Modules) > 0 {
				log.Printf("[ModuleContentGen] JSON repair successful, got %d modules", len(batchResponse.Modules))
			}
		}

		if len(batchResponse.Modules) == 0 {
			return nil, fmt.Errorf("error parsing JSON from Gemini response: %w", err)
		}
	}

	// Validate and filter modules
	var validModules []models.ModuleData
	for _, m := range batchResponse.Modules {
		if strings.TrimSpace(m.ModuleName) != "" && strings.TrimSpace(m.Content) != "" {
			validModules = append(validModules, m)
		}
	}

	if len(validModules) == 0 {
		return nil, fmt.Errorf("no valid modules in response")
	}

	log.Printf("[ModuleContentGen] Parsed %d valid modules from batch response", len(validModules))
	return validModules, nil
}

// ModuleContentRequest contains data for generating module content
type ModuleContentRequest struct {
	ModuleName  string
	Description string
	WeekContext string // e.g., "Week 3: Data Structures"
}

// parseModuleContentResponse parses the module content response
func (p *GeminiProvider) parseModuleContentResponse(body []byte) (models.ModuleData, error) {
	// Parse Gemini response structure
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return models.ModuleData{}, fmt.Errorf("error parsing Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return models.ModuleData{}, fmt.Errorf("no content in Gemini response")
	}

	responseContent := geminiResp.Candidates[0].Content.Parts[0].Text

	// Clean the response
	responseContent = cleanJSONResponse(responseContent)

	// Parse the JSON object
	var moduleData models.ModuleData
	if err := json.Unmarshal([]byte(responseContent), &moduleData); err != nil {
		log.Printf("[ModuleContentGen] JSON parse error: %v", err)
		log.Printf("[ModuleContentGen] Invalid JSON: %s", responseContent[:min(500, len(responseContent))])
		return models.ModuleData{}, fmt.Errorf("error parsing JSON from Gemini response: %w", err)
	}

	// Validate module data
	if strings.TrimSpace(moduleData.ModuleName) == "" {
		return models.ModuleData{}, fmt.Errorf("module name is empty")
	}

	log.Printf("[ModuleContentGen] Generated content for module: %s (%d chars)",
		moduleData.ModuleName, len(moduleData.Content))
	return moduleData, nil
}

// =====================================================
// Prompt Building for Phase 1 & Phase 2
// =====================================================

// buildLessonPlanStructurePrompt creates the prompt for Phase 1 structure extraction
func (p *GeminiProvider) buildLessonPlanStructurePrompt(req LessonPlanRequest) string {
	var sb strings.Builder

	sb.WriteString("## ROLE\n")
	sb.WriteString("You are an educational content analyzer specializing in extracting\n")
	sb.WriteString("course structure from lesson plans. Your task is to extract ONLY\n")
	sb.WriteString("the structure (week names and module titles) WITHOUT generating content.\n\n")

	sb.WriteString("## INPUT\n")
	sb.WriteString(fmt.Sprintf("Course: %s\n", req.CourseName))
	sb.WriteString(fmt.Sprintf("Level: %s\n\n", req.CourseLevel))
	sb.WriteString("Lesson Plan Content:\n")
	sb.WriteString("--- BEGIN DOCUMENT ---\n")
	sb.WriteString(req.DocumentContent)
	sb.WriteString("\n--- END DOCUMENT ---\n\n")

	sb.WriteString("## IMPORTANT: DATE TO WEEK MAPPING\n")
	sb.WriteString("The document contains DATES but NOT explicit week numbers.\n")
	sb.WriteString("You MUST derive week numbers from dates:\n")
	sb.WriteString("1. Find the earliest date - this is Week 1\n")
	sb.WriteString("2. Group dates into weekly intervals (7 days each)\n")
	sb.WriteString("3. Assign sequential week numbers (1, 2, 3, ...)\n")
	sb.WriteString("4. Extract module names associated with each date range\n\n")

	sb.WriteString("## TASK\n")
	sb.WriteString("Extract the structure from the lesson plan:\n")
	sb.WriteString("1. Extract all dates and group them into weeks\n")
	sb.WriteString("2. For each week, extract the module/topic NAMES ONLY\n")
	sb.WriteString("3. DO NOT generate any content, descriptions, or explanations\n")
	sb.WriteString("4. Return ONLY the structure with week names and module names\n\n")

	sb.WriteString("## OUTPUT FORMAT\n")
	sb.WriteString("Return a JSON object with this EXACT structure:\n")
	sb.WriteString(`{
    "course_name": "Course Name from document",
    "total_weeks": 10,
    "weeks": [
        {
            "week_number": 1,
            "week_name": "Week 1: Introduction to Topic (Jan 15-21)",
            "date_range": "Jan 15-21, 2024",
            "module_names": ["Module Title 1", "Module Title 2", "Module Title 3"]
        }
    ]
}

CRITICAL RULES:
1. MUST have at least 1 week in the "weeks" array
2. MUST derive week numbers from dates (Week 1 starts from earliest date)
3. Extract ONLY module names/titles from the document - do not generate content
4. Module names should be the actual topic names from the document
5. Keep week_name descriptive with dates included
6. date_range is optional but helpful (format: "Jan 15-21, 2024")

IMPORTANT: Return ONLY valid JSON. No markdown code fences.`)

	return sb.String()
}

// buildModuleContentPrompt creates the prompt for Phase 2 module content generation
func (p *GeminiProvider) buildModuleContentPrompt(courseName, courseLevel, moduleName, moduleDescription, weekContext string) string {
	var sb strings.Builder

	sb.WriteString("## ROLE\n")
	sb.WriteString("You are an educational content creator specializing in writing\n")
	sb.WriteString("concise, clear educational content for online courses.\n\n")

	sb.WriteString("## CONTEXT\n")
	sb.WriteString(fmt.Sprintf("Course: %s (%s)\n", courseName, courseLevel))
	if weekContext != "" {
		sb.WriteString(fmt.Sprintf("Week: %s\n", weekContext))
	}
	sb.WriteString(fmt.Sprintf("Module: %s\n", moduleName))
	if moduleDescription != "" {
		sb.WriteString(fmt.Sprintf("Description: %s\n", moduleDescription))
	}
	sb.WriteString("\n")

	sb.WriteString("## TASK\n")
	sb.WriteString("Generate educational content for the above module.\n")
	sb.WriteString("The content should be:\n")
	sb.WriteString("- Concise: 50-150 words (key points and summary)\n")
	sb.WriteString("- In markdown format with headings, lists, and code examples where relevant\n")
	sb.WriteString("- Appropriate for undergraduate students\n")
	sb.WriteString("- Self-contained and clear\n\n")

	sb.WriteString("## OUTPUT FORMAT\n")
	sb.WriteString(`Return a JSON object with this EXACT structure:
{
    "module_name": "Module name (same as input)",
    "description": "2-3 sentence summary of what this module covers",
    "content": "Markdown content with key points, concepts, and brief examples"
}

CONTENT GUIDELINES:
1. Start with a brief introduction to the topic
2. Cover key concepts and definitions
3. Include a small code example if applicable
4. End with a summary or key takeaway

IMPORTANT: Return ONLY valid JSON. No markdown code fences.`)

	return sb.String()
}

// countTotalModuleNames counts total module names in lite weeks
func countTotalModuleNames(weeks []models.WeekDataLite) int {
	total := 0
	for _, week := range weeks {
		total += len(week.ModuleNames)
	}
	return total
}

// =====================================================
// Helper Functions
// =====================================================

// EstimateTokensForLessonPlan estimates tokens for lesson plan content
func EstimateTokensForLessonPlan(content string) int {
	// Rough estimate: 4 characters ≈ 1 token
	// Add overhead for prompt template (~500 tokens)
	return len(content)/4 + 500
}

// ChunkLessonPlanByTokenBudget splits content if it exceeds token budget
func ChunkLessonPlanByTokenBudget(content string, maxTokens int) []string {
	if len(content) <= maxTokens*4 {
		return []string{content}
	}

	// Split by paragraphs (double newlines)
	paragraphs := strings.Split(content, "\n\n")
	var chunks []string
	var currentChunk strings.Builder
	currentTokens := 0

	for _, para := range paragraphs {
		paraTokens := len(para) / 4
		if currentTokens+paraTokens > maxTokens && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentTokens = 0
		}
		currentChunk.WriteString(para)
		currentChunk.WriteString("\n\n")
		currentTokens += paraTokens
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// ConvertPracticeQuizQuestions converts AI-generated questions to model format
func ConvertPracticeQuizQuestions(_ uint, questions []models.PracticeQuizQuestionData, moduleMap map[string]uint) []models.PracticeQuizQuestion {
	result := make([]models.PracticeQuizQuestion, 0, len(questions))

	for i, q := range questions {
		pq := models.PracticeQuizQuestion{
			QuestionType: models.QuestionType(q.QuestionType),
			QuestionText: q.QuestionText,
			Explanation:  q.Explanation,
			Difficulty:   models.QuestionDifficulty(q.Difficulty),
			OrderIndex:   i,
			Marks:        1,
		}

		// Map source module name to ID
		if moduleID, ok := moduleMap[q.SourceModuleName]; ok {
			pq.SourceModuleID = &moduleID
		}

		// Convert options
		for _, opt := range q.Options {
			pq.Options = append(pq.Options, models.PracticeQuizOption{
				OptionText: opt.OptionText,
				IsCorrect:  opt.IsCorrect,
				OrderIndex: opt.OrderIndex,
			})
		}

		result = append(result, pq)
	}

	return result
}

// =====================================================
// Contest Generation Methods
// =====================================================
// New sequential generation methods are implemented below with rate limiting
// Old batch generation methods (generateContestProblems, parseContestProblemsResponse) removed

// generateEditorialHints generates editorial hints for a problem
func (p *GeminiProvider) generateEditorialHints(ctx context.Context, prompt string) (*models.EditorialResponse, error) {
	// Build Gemini request body
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  2048,
			"temperature":      0.3, // More deterministic for hints
			"responseMimeType": "application/json",
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Gemini request: %w", err)
	}

	// Use header for API key (more secure than query string)
	body, _, err := p.doRequest(ctx, jsonData)
	if err != nil {
		return nil, err
	}

	editorial, err := p.parseEditorialResponse(body)
	if err != nil {
		return nil, err
	}

	return editorial, nil
}

// parseEditorialResponse parses the Gemini API response for editorial hints
func (p *GeminiProvider) parseEditorialResponse(body []byte) (*models.EditorialResponse, error) {
	// Parse Gemini response structure
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("error parsing Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in Gemini response")
	}

	responseContent := geminiResp.Candidates[0].Content.Parts[0].Text

	// Clean the response
	responseContent = cleanJSONResponse(responseContent)

	// Parse the JSON object
	var editorial models.EditorialResponse
	if err := json.Unmarshal([]byte(responseContent), &editorial); err != nil {
		return nil, fmt.Errorf("error parsing JSON from Gemini response: %w", err)
	}

	return &editorial, nil
}

// =====================================================
// Contest Problem Generation - Sequential with Rate Limiting
// =====================================================

// ContestProblemGenerationRequest contains parameters for generating a single contest problem
type ContestProblemGenerationRequest struct {
	Topics          []string `json:"topics"`
	DifficultyLevel string   `json:"difficulty_level"`
	CourseName      string   `json:"course_name"`
	ProblemNumber   int      `json:"problem_number"` // 1-indexed problem number
	TotalProblems   int      `json:"total_problems"`
}

// GenerateContestProblemSequential generates a single contest problem with rate limiting
// This method is designed to avoid Gemini API rate limits by generating problems one at a time
func (p *GeminiProvider) GenerateContestProblemSequential(ctx context.Context, req ContestProblemGenerationRequest) (*models.GeneratedProblem, error) {
	prompt := p.buildSingleContestProblemPrompt(req)

	// Build Gemini request body
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  8192, // Token limit for single problem
			"temperature":      0.5,
			"responseMimeType": "application/json",
		},
		"safetySettings": []map[string]string{
			{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Gemini request: %w", err)
	}

	// Use header for API key (more secure than query string)
	body, _, err := p.doRequest(ctx, jsonData)
	if err != nil {
		return nil, err
	}

	problem, err := p.parseSingleContestProblemResponse(body)
	if err != nil {
		return nil, err
	}

	return problem, nil
}

// GenerateContestProblemsSequential generates multiple contest problems one at a time with rate limiting
// This avoids hitting Gemini API rate limits when generating many problems
func (p *GeminiProvider) GenerateContestProblemsSequential(ctx context.Context, req models.ContestGenerationRequest, courseName string) ([]models.GeneratedProblem, error) {
	var problems []models.GeneratedProblem

	// Use VersionsPerProblem from request (default to 1 = exact count)
	versionsMultiplier := req.VersionsPerProblem
	if versionsMultiplier <= 0 {
		versionsMultiplier = 1
	}
	generatedCount := req.RequestedCount * versionsMultiplier

	log.Printf("[ContestGeneration] Starting sequential generation of %d problems", generatedCount)

	// Rate limiting configuration
	const (
		baseDelay  = 1000 * time.Millisecond // Base delay between API calls
		maxDelay   = 10 * time.Second        // Maximum delay after rate limit
		maxRetries = 3                       // Maximum retries per problem
	)

	for i := 0; i < generatedCount; i++ {
		select {
		case <-ctx.Done():
			return problems, ctx.Err()
		default:
		}

		// Build request for single problem
		singleReq := ContestProblemGenerationRequest{
			Topics:          req.Topics,
			DifficultyLevel: req.DifficultyLevel,
			CourseName:      courseName,
			ProblemNumber:   i + 1,
			TotalProblems:   generatedCount,
		}

		var problem *models.GeneratedProblem
		var err error
		retryDelay := baseDelay

		// Retry loop with exponential backoff
		for attempt := 0; attempt < maxRetries; attempt++ {
			// Wait before retry (not on first attempt)
			if attempt > 0 {
				log.Printf("[ContestGeneration] Retry attempt %d/%d for problem %d, waiting %v",
					attempt, maxRetries, i+1, retryDelay)
				select {
				case <-ctx.Done():
					return problems, ctx.Err()
				case <-time.After(retryDelay):
				}
				retryDelay *= 2 // Exponential backoff
				if retryDelay > maxDelay {
					retryDelay = maxDelay
				}
			}

			log.Printf("[ContestGeneration] Generating problem %d/%d (attempt %d/%d)",
				i+1, generatedCount, attempt+1, maxRetries)

			problem, err = p.GenerateContestProblemSequential(ctx, singleReq)
			if err == nil {
				// Success
				problems = append(problems, *problem)
				log.Printf("[ContestGeneration] Successfully generated problem %d: %s", i+1, problem.Title)
				break
			}

			// Check if it's a rate limit error (429)
			if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit") {
				log.Printf("[ContestGeneration] Rate limit hit for problem %d, will retry after %v", i+1, retryDelay)
				continue
			}

			// For non-rate-limit errors, log and continue to next problem
			log.Printf("[ContestGeneration] Failed to generate problem %d: %v", i+1, err)
			break
		}

		if err != nil && problem == nil {
			log.Printf("[ContestGeneration] Skipping problem %d after %d failed attempts", i+1, maxRetries)
			continue
		}

		// Add delay between successful generations to avoid rate limits
		if i < generatedCount-1 {
			delay := baseDelay + time.Duration(rand.Intn(500))*time.Millisecond
			log.Printf("[ContestGeneration] Waiting %v before generating next problem", delay)
			select {
			case <-ctx.Done():
				return problems, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	log.Printf("[ContestGeneration] Sequential generation completed: %d/%d problems successful",
		len(problems), generatedCount)

	if len(problems) == 0 {
		return nil, fmt.Errorf("failed to generate any valid problems")
	}

	return problems, nil
}

// buildSingleContestProblemPrompt creates the prompt for generating a single contest problem
func (p *GeminiProvider) buildSingleContestProblemPrompt(req ContestProblemGenerationRequest) string {
	var sb strings.Builder

	sb.WriteString("## ROLE\n")
	sb.WriteString("You are an expert competitive programming problem setter.\n\n")

	sb.WriteString("## CONTEST CONTEXT\n")
	if req.CourseName != "" {
		sb.WriteString(fmt.Sprintf("- Course: %s\n", req.CourseName))
	}
	sb.WriteString(fmt.Sprintf("- Topics: %s\n", strings.Join(req.Topics, ", ")))
	if req.DifficultyLevel != "" {
		sb.WriteString(fmt.Sprintf("- Difficulty: %s\n", req.DifficultyLevel))
	}
	sb.WriteString(fmt.Sprintf("- Problem %d of %d in this set\n\n", req.ProblemNumber, req.TotalProblems))

	sb.WriteString("## TASK\n")
	sb.WriteString("Generate ONE original coding problem covering the specified topics.\n")
	sb.WriteString("This is problem ")
	sb.WriteString(fmt.Sprintf("%d out of %d total problems being generated.\n", req.ProblemNumber, req.TotalProblems))
	sb.WriteString("Make sure each problem is unique and covers different aspects of the topics.\n\n")

	sb.WriteString("## REQUIRED OUTPUT FORMAT\n")
	sb.WriteString("Return a JSON object with this EXACT structure:\n")
	sb.WriteString(`{
    "title": "Creative, descriptive title",
    "description": "Detailed problem statement with story/context (200-500 words)",
    "input_format": "Precise input specification",
    "output_format": "Exact output requirement",
    "constraints": {
        "n": "1 <= N <= 10^5",
        "array_elements": "1 <= A[i] <= 10^9"
    },
    "sample_input": "3\n1 2 3",
    "sample_output": "6",
    "sample_explanation": "Explanation of why this output is correct",
    "difficulty": "easy",
    "topics_covered": ["topic1", "topic2"],
    "solution_approach": "Brief description of intended solution",
    "reference_solution": "Python or C++ code",
    "solution_language": "python",
    "test_cases": [
        {
            "category": "sample",
            "input": "3\n1 2 3",
            "expected_output": "6",
            "description": "Basic example",
            "points": 10,
            "is_sample_visible": true
        }
    ]
}

## PROBLEM QUALITY CRITERIA
1. CLARITY: Unambiguous problem statement
2. SOLVABILITY: Guaranteed to have at least one correct solution
3. CONSTRAINTS: Appropriate for difficulty level
   - Easy: N <= 1000, O(N^2) acceptable
   - Medium: N <= 10^5, O(N log N) required
   - Hard: N <= 10^5-10^6, optimal solution required
4. EDGE CASES: Problem should have identifiable edge cases
5. ORIGINALITY: Not a direct copy of well-known problems

## TEST CASE REQUIREMENTS
This problem MUST include EXACTLY 5 test cases:
1. 2 sample cases (is_sample_visible: true) — simple, illustrative examples
2. 3 hidden cases (is_sample_visible: false) — edge case, mid-range, stress case

## IMPORTANT RULES
- Problem must be self-contained and solvable independently
- Input/output formats must be precisely specified
- Constraints must be realistic for competitive programming
- Reference solution must be correct and efficient
- Generate exactly 5 test cases: 2 visible sample + 3 hidden
- Make this problem unique compared to other problems in the set

Return ONLY valid JSON object. No markdown code fences, no additional text.`)

	return sb.String()
}

// parseSingleContestProblemResponse parses the response for a single contest problem
func (p *GeminiProvider) parseSingleContestProblemResponse(body []byte) (*models.GeneratedProblem, error) {
	// Parse Gemini response structure
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("error parsing Gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s (code: %d)", geminiResp.Error.Message, geminiResp.Error.Code)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in Gemini response")
	}

	responseContent := geminiResp.Candidates[0].Content.Parts[0].Text

	// Clean the response
	responseContent = cleanJSONResponse(responseContent)

	// Parse the JSON object
	var problem models.GeneratedProblem
	if err := json.Unmarshal([]byte(responseContent), &problem); err != nil {
		log.Printf("[ContestGeneration] JSON parse error: %v", err)
		log.Printf("[ContestGeneration] Invalid JSON: %s", responseContent[:min(500, len(responseContent))])
		return nil, fmt.Errorf("error parsing JSON from Gemini response: %w", err)
	}

	// Validate problem
	if strings.TrimSpace(problem.Title) == "" {
		return nil, fmt.Errorf("problem title is empty")
	}

	if strings.TrimSpace(problem.Description) == "" {
		return nil, fmt.Errorf("problem description is empty")
	}

	if len(problem.TestCases) == 0 {
		return nil, fmt.Errorf("no test cases provided")
	}

	// Set default difficulty if not specified
	if problem.Difficulty == "" {
		problem.Difficulty = "medium"
	}

	log.Printf("[ContestGeneration] Successfully generated problem: %s", problem.Title)
	return &problem, nil
}

// =====================================================
// Contest Generation - Agentic Flow Methods
// =====================================================

// GenerateContestProblem generates a single contest problem for the agent (without verification)
// This is called by the Contest Agent's GenerateProblem tool
func (p *GeminiProvider) GenerateContestProblem(ctx context.Context, topics []string, difficulty, courseContext string) (*models.GeneratedProblem, error) {
	prompt := p.buildAgenticContestProblemPrompt(topics, difficulty, courseContext)

	// Build Gemini request body
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  8192,
			"temperature":      0.5,
			"responseMimeType": "application/json",
		},
		"safetySettings": []map[string]string{
			{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
			{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_MEDIUM_AND_ABOVE"},
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Gemini request: %w", err)
	}

	// Use header for API key (more secure than query string)
	body, _, err := p.doRequest(ctx, jsonData)
	if err != nil {
		return nil, err
	}

	problem, err := p.parseAgenticContestProblemResponse(body)
	if err != nil {
		return nil, err
	}

	return problem, nil
}

// FixContestSolution fixes a reference solution based on error feedback
// This is called by the Contest Agent's FixCode tool
func (p *GeminiProvider) FixContestSolution(ctx context.Context, originalCode, errorMessage, errorType, problemStatement string) (string, error) {
	prompt := p.buildFixSolutionPrompt(originalCode, errorMessage, errorType, problemStatement)

	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{
						"text": prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  4096,
			"temperature":      0.3,
			"responseMimeType": "text/plain",
		},
	}

	jsonData, err := json.Marshal(geminiReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling Gemini request: %w", err)
	}

	body, _, err := p.doRequest(ctx, jsonData)
	if err != nil {
		return "", err
	}

	// Parse response
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("error parsing Gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in Gemini response")
	}

	fixedCode := geminiResp.Candidates[0].Content.Parts[0].Text
	fixedCode = strings.TrimSpace(fixedCode)

	// Strip markdown code fences if present
	fixedCode = strings.TrimPrefix(fixedCode, "```go\n")
	fixedCode = strings.TrimPrefix(fixedCode, "```cpp\n")
	fixedCode = strings.TrimPrefix(fixedCode, "```python\n")
	fixedCode = strings.TrimPrefix(fixedCode, "```")
	if idx := strings.LastIndex(fixedCode, "```"); idx != -1 {
		fixedCode = fixedCode[:idx]
	}
	fixedCode = strings.TrimSpace(fixedCode)

	return fixedCode, nil
}

// buildAgenticContestProblemPrompt builds the prompt for agentic problem generation
func (p *GeminiProvider) buildAgenticContestProblemPrompt(topics []string, difficulty, courseContext string) string {
	var sb strings.Builder

	sb.WriteString("## ROLE\n")
	sb.WriteString("You are an expert competitive programming problem setter.\n\n")

	sb.WriteString("## TASK\n")
	sb.WriteString("Generate ONE original coding problem covering these topics: ")
	sb.WriteString(strings.Join(topics, ", "))
	sb.WriteString("\n\n")

	if courseContext != "" {
		sb.WriteString(fmt.Sprintf("Course Context: %s\n\n", courseContext))
	}

	sb.WriteString(fmt.Sprintf("Difficulty Level: %s\n\n", difficulty))

	sb.WriteString("## REQUIRED OUTPUT FORMAT\n")
	sb.WriteString("Return a JSON object with this EXACT structure:\n")
	sb.WriteString(`{
    "title": "Creative, descriptive title",
    "description": "Detailed problem statement with story/context (200-500 words)",
    "input_format": "Precise input specification",
    "output_format": "Exact output requirement",
    "constraints": {
        "n": "1 <= N <= 10^5",
        "array_elements": "1 <= A[i] <= 10^9",
        "time_limit_ms": 2000,
        "memory_limit_kb": 256000
    },
    "sample_input": "3\n1 2 3",
    "sample_output": "6",
    "sample_explanation": "Explanation of why this output is correct",
    "difficulty": "easy",
    "topics_covered": ["topic1", "topic2"],
    "solution_approach": "Brief description of intended solution",
    "reference_solution": "Python or C++ code that is CORRECT and COMPILABLE",
    "solution_language": "python",
    "test_cases": [
        {
            "category": "sample",
            "input": "3\n1 2 3",
            "expected_output": "6",
            "description": "Basic example",
            "points": 10,
            "is_sample_visible": true
        }
    ]
}

## CRITICAL REQUIREMENTS
1. The reference_solution MUST be correct and compile without errors
2. Generate EXACTLY 5 test cases: 2 visible sample + 3 hidden
3. Problem must be solvable with the given constraints
4. Include edge cases in hidden test cases

Return ONLY valid JSON object. No markdown code fences, no additional text.`)

	return sb.String()
}

// buildFixSolutionPrompt builds the prompt for fixing a solution
func (p *GeminiProvider) buildFixSolutionPrompt(originalCode, errorMessage, errorType, problemStatement string) string {
	var sb strings.Builder

	sb.WriteString("## ROLE\n")
	sb.WriteString("You are an expert programmer debugging competitive programming solutions.\n\n")

	sb.WriteString("## PROBLEM STATEMENT\n")
	sb.WriteString(problemStatement)
	sb.WriteString("\n\n")

	sb.WriteString("## ORIGINAL CODE\n")
	sb.WriteString("```")
	sb.WriteString(originalCode)
	sb.WriteString("\n```\n\n")

	sb.WriteString(fmt.Sprintf("## ERROR TYPE: %s\n\n", errorType))
	sb.WriteString("## ERROR MESSAGE\n")
	sb.WriteString("```\n")
	sb.WriteString(errorMessage)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## TASK\n")
	sb.WriteString("Fix the code above to resolve the error.\n")
	sb.WriteString("- For compilation errors: Fix syntax issues, missing imports, type errors\n")
	sb.WriteString("- For runtime errors: Fix logic bugs, array bounds, null references\n")
	sb.WriteString("- For wrong answer: Fix the algorithm logic\n")
	sb.WriteString("- For time limit: Optimize the algorithm\n\n")

	sb.WriteString("## OUTPUT\n")
	sb.WriteString("Return ONLY the fixed code. No explanations, no markdown fences.\n")
	sb.WriteString("The code must be complete and ready to compile/run.\n")

	return sb.String()
}

// parseAgenticContestProblemResponse parses the response for agentic problem generation
func (p *GeminiProvider) parseAgenticContestProblemResponse(body []byte) (*models.GeneratedProblem, error) {
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("error parsing Gemini response: %w", err)
	}

	if geminiResp.Error != nil {
		return nil, fmt.Errorf("Gemini API error: %s (code: %d)", geminiResp.Error.Message, geminiResp.Error.Code)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in Gemini response")
	}

	responseContent := geminiResp.Candidates[0].Content.Parts[0].Text
	responseContent = cleanJSONResponse(responseContent)

	var problem models.GeneratedProblem
	if err := json.Unmarshal([]byte(responseContent), &problem); err != nil {
		log.Printf("[ContestAgent] JSON parse error: %v", err)
		log.Printf("[ContestAgent] Invalid JSON (length=%d): %s", len(responseContent), responseContent[:min(500, len(responseContent))])

		// Try to repair truncated JSON
		repairedJSON := repairTruncatedJSON(responseContent)
		if repairedJSON != responseContent {
			log.Printf("[ContestAgent] Attempting repair with: %s...", repairedJSON[:min(200, len(repairedJSON))])
			if repairErr := json.Unmarshal([]byte(repairedJSON), &problem); repairErr == nil && strings.TrimSpace(problem.Title) != "" {
				log.Printf("[ContestAgent] JSON repair successful!")
				goto validate
			}
		}

		return nil, fmt.Errorf("error parsing JSON from Gemini response: %w", err)
	}

validate:
	// Validate problem
	if strings.TrimSpace(problem.Title) == "" {
		return nil, fmt.Errorf("problem title is empty")
	}

	if strings.TrimSpace(problem.Description) == "" {
		return nil, fmt.Errorf("problem description is empty")
	}

	if len(problem.TestCases) == 0 {
		return nil, fmt.Errorf("no test cases provided")
	}

	if problem.Difficulty == "" {
		problem.Difficulty = "medium"
	}

	log.Printf("[ContestAgent] Generated problem: %s", problem.Title)
	return &problem, nil
}

// repairTruncatedJSON attempts to repair truncated JSON by closing unclosed brackets
func repairTruncatedJSON(jsonStr string) string {
	// Count unclosed brackets
	parenCount := 0
	bracketCount := 0
	braceCount := 0
	inString := false
	escape := false

	for _, ch := range jsonStr {
		if escape {
			escape = false
			continue
		}
		if ch == '\\' && inString {
			escape = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '(':
			parenCount++
		case ')':
			parenCount--
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		case '{':
			braceCount++
		case '}':
			braceCount--
		}
	}

	// If we're in a string, close it
	if inString {
		jsonStr += "\""
	}

	// Close unclosed structures (in reverse order of typical nesting)
	// For lesson plan structure, we typically have:
	// {"weeks": [{"week_number": ..., "week_name": "...", "module_names": [...]}]}

	// Close any unclosed brackets
	for bracketCount > 0 {
		jsonStr += "]"
		bracketCount--
	}

	// Close any unclosed braces
	for braceCount > 0 {
		jsonStr += "}"
		braceCount--
	}

	return jsonStr
}
