package services

import (
	"bytes"
	"coding-platform/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// =====================================================
// AI Provider Interface
// =====================================================

// AIProvider defines the interface for LLM providers
type AIProvider interface {
	// GenerateQuizQuestions generates quiz questions from theory content
	GenerateQuizQuestions(ctx context.Context, req QuizGenerationRequest) ([]GeneratedQuestion, error)
	// EstimateTokens estimates the number of tokens in a text
	EstimateTokens(content string) int
}

// =====================================================
// Request/Response Types
// =====================================================

// ModuleContent represents a theory module's content for AI processing
type ModuleContent struct {
	WeekName       string `json:"week_name"`
	ModuleName     string `json:"module_name"`
	Description    string `json:"description"`
	Content        string `json:"content"`
	TheoryWeekID   uint   `json:"theory_week_id"`
	TheoryModuleID uint   `json:"theory_module_id"`
}

// QuizGenerationRequest contains all parameters for AI quiz generation
type QuizGenerationRequest struct {
	TheoryContent []ModuleContent `json:"theory_content"`
	TeacherPrompt string          `json:"teacher_prompt"`
	QuestionCount int             `json:"question_count"`
	DifficultyMix map[string]int  `json:"difficulty_mix"` // {"easy": 5, "medium": 3, "hard": 2}
	QuestionTypes []string        `json:"question_types"` // ["mcq", "true_false"]
	TotalMarks    int             `json:"total_marks"`
}

// GeneratedOption represents a single option for MCQ/multi-select questions
type GeneratedOption struct {
	OptionText string `json:"option_text"`
	IsCorrect  bool   `json:"is_correct"`
	OrderIndex int    `json:"order_index"`
}

// GeneratedQuestion represents a single AI-generated question
type GeneratedQuestion struct {
	QuestionText   string            `json:"question_text"`
	QuestionType   string            `json:"question_type"`
	Options        []GeneratedOption `json:"options"`
	Explanation    string            `json:"explanation"`
	Difficulty     string            `json:"difficulty"`
	BloomLevel     string            `json:"bloom_level"`
	Marks          int               `json:"marks"`
	SourceWeekID   uint              `json:"source_week_id"`
	SourceModuleID uint              `json:"source_module_id"`
}

// =====================================================
// Gemini Provider Implementation
// =====================================================

// GeminiProvider implements AIProvider using Google's Gemini API
type GeminiProvider struct {
	apiKey       string
	model        string
	maxTokens    int
	temperature  float64
	apiURL       string
	httpClient   *http.Client
	rateLimitRPS float64         // Requests per second
	rateLimiter  *APIRateLimiter // Token-aware rate limiter
}

// NewGeminiProvider creates a new Gemini provider with rate limiting
func NewGeminiProvider(apiKey, model string, maxTokens int, temperature float64, rateLimitRPM int) *GeminiProvider {
	// Convert RPM to RPS (requests per second)
	rateLimitRPS := float64(rateLimitRPM) / 60.0

	// Create rate limiter config based on RPM
	rlConfig := APIRateLimiterConfig{
		RequestsPerMinute: rateLimitRPM,
		TokensPerMinute:   1000000, // Gemini free tier default
		SafetyMargin:      0.9,     // Use 90% of limits
	}

	return &GeminiProvider{
		apiKey:       apiKey,
		model:        model,
		maxTokens:    maxTokens,
		temperature:  temperature,
		apiURL:       fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model),
		rateLimitRPS: rateLimitRPS,
		rateLimiter:  NewAPIRateLimiter(rlConfig),
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

// EstimateTokens estimates tokens using 4 chars ≈ 1 token heuristic
func (p *GeminiProvider) EstimateTokens(content string) int {
	return len(content) / 4
}

// isRetryableGeminiError determines if a Gemini API error is worth retrying
func isRetryableGeminiError(statusCode int, err error) bool {
	if err != nil {
		// Network errors are generally retryable
		return true
	}
	// Retry on server errors and rate limiting
	return statusCode == 429 || statusCode >= 500
}

// GenerateQuizQuestions generates questions using Gemini API with retry and rate limiting
func (p *GeminiProvider) GenerateQuizQuestions(ctx context.Context, req QuizGenerationRequest) ([]GeneratedQuestion, error) {
	prompt := p.buildPrompt(req)

	// Calculate input tokens and adjust max output tokens
	inputTokens := p.EstimateTokens(prompt)
	availableTokens := p.maxTokens - inputTokens - 500 // Reserve buffer for response overhead

	if availableTokens < 1000 {
		return nil, fmt.Errorf("input too large (%d tokens), need at least 1000 tokens for output. Consider reducing content size", inputTokens)
	}

	// Build Gemini request body with calculated output tokens
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
			"maxOutputTokens":  availableTokens,
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

	// Retry logic with exponential backoff
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Apply rate limiting before each request
		if p.rateLimitRPS > 0 && attempt > 0 {
			waitTime := time.Duration(1.0/p.rateLimitRPS) * time.Second
			time.Sleep(waitTime)
		}

		// Add exponential backoff for retries
		if attempt > 0 {
			backoffDuration := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			log.Printf("Retry attempt %d after %v backoff", attempt, backoffDuration)
			time.Sleep(backoffDuration)
		}

		body, statusCode, err := p.doRequest(ctx, jsonData)
		if err == nil {
			// Parse the response
			questions, parseErr := p.parseGeminiResponse(body, req.TheoryContent)
			if parseErr != nil {
				return nil, fmt.Errorf("error parsing response: %w", parseErr)
			}
			return questions, nil
		}

		lastErr = err
		if !isRetryableGeminiError(statusCode, err) {
			break
		}

		log.Printf("Gemini API error (status %d): %v, will retry", statusCode, err)
	}

	return nil, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}

// doRequest performs the actual HTTP request to Gemini API
// Returns the response body, status code, and error
func (p *GeminiProvider) doRequest(ctx context.Context, jsonData []byte) ([]byte, int, error) {
	// Build URL without API key in query string (use header instead)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, fmt.Errorf("error creating HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// Use header for API key (more secure than query string)
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("Gemini API error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, resp.StatusCode, nil
}

// =====================================================
// Gemini Provider Methods
// =====================================================

// buildPrompt creates the full prompt for Gemini requests
func (p *GeminiProvider) buildPrompt(req QuizGenerationRequest) string {
	var sb strings.Builder

	sb.WriteString("## THEORY CONTENT CONTEXT\n\n")
	for i, module := range req.TheoryContent {
		sb.WriteString(fmt.Sprintf("### Week %d: %s\n", i+1, module.WeekName))
		sb.WriteString(fmt.Sprintf("#### Module: %s\n", module.ModuleName))
		sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", module.Description))
		sb.WriteString(fmt.Sprintf("**Content:**\n%s\n\n", module.Content))
		sb.WriteString("---\n\n")
	}

	sb.WriteString("## INSTRUCTIONS\n\n")
	if req.TeacherPrompt != "" {
		sb.WriteString(fmt.Sprintf("Teacher's specific instructions:\n%s\n\n", req.TeacherPrompt))
	}

	sb.WriteString("## QUESTION REQUIREMENTS\n\n")
	sb.WriteString(fmt.Sprintf("- Total questions: %d\n", req.QuestionCount))
	if len(req.DifficultyMix) > 0 {
		sb.WriteString("- Difficulty distribution:\n")
		for diff, count := range req.DifficultyMix {
			sb.WriteString(fmt.Sprintf("  * %s: %d questions\n", diff, count))
		}
	}
	if len(req.QuestionTypes) > 0 {
		sb.WriteString(fmt.Sprintf("- Question types: %s\n", strings.Join(req.QuestionTypes, ", ")))
	}
	sb.WriteString("\n")

	sb.WriteString("## OUTPUT JSON SCHEMA\n\n")
	sb.WriteString(`Output a JSON object with "questions" array. Each question:
{
  "questions": [
    {
      "question_text": "Question text",
      "question_type": "mcq|multi_select|true_false|short_answer|fill_blank",
      "options": [{"option_text": "Option", "is_correct": false, "order_index": 0}],
      "explanation": "Explanation here",
      "difficulty": "easy|medium|hard",
      "bloom_level": "remember|understand|apply|analyze|evaluate|create",
      "marks": 1,
      "source_week_id": 123,
      "source_module_id": 456
    }
  ]
}

RULES:
1. MCQ/multi_select: 4 options, exactly one is_correct: true
2. true_false: 2 options ("True", "False")
3. short_answer/fill_blank: empty options array
4. Questions MUST be from provided content only
5. Include explanations referencing source material
6. Map source_week_id and source_module_id from input`)

	return sb.String()
}

// parseGeminiResponse parses Gemini API response
func (p *GeminiProvider) parseGeminiResponse(body []byte, theoryContent []ModuleContent) ([]GeneratedQuestion, error) {
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

	// Clean the response: strip markdown fences and fix trailing commas
	responseContent = cleanJSONResponse(responseContent)

	// Parse the JSON from the response using flexible types for numeric fields
	// Gemini sometimes returns numbers as strings
	type flexibleOption struct {
		OptionText string      `json:"option_text"`
		IsCorrect  interface{} `json:"is_correct"`
		OrderIndex interface{} `json:"order_index"`
	}
	type flexibleQuestion struct {
		QuestionText   string           `json:"question_text"`
		QuestionType   string           `json:"question_type"`
		Options        []flexibleOption `json:"options"`
		Explanation    string           `json:"explanation"`
		Difficulty     string           `json:"difficulty"`
		BloomLevel     string           `json:"bloom_level"`
		Marks          interface{}      `json:"marks"`
		SourceWeekID   interface{}      `json:"source_week_id"`
		SourceModuleID interface{}      `json:"source_module_id"`
	}
	var response struct {
		Questions []flexibleQuestion `json:"questions"`
	}

	if err := json.Unmarshal([]byte(responseContent), &response); err != nil {
		return nil, fmt.Errorf("error parsing JSON from Gemini response: %w", err)
	}

	// Convert flexible types to concrete GeneratedQuestion structs
	questions := make([]GeneratedQuestion, 0, len(response.Questions))
	for _, fq := range response.Questions {
		q := GeneratedQuestion{
			QuestionText:   fq.QuestionText,
			QuestionType:   fq.QuestionType,
			Explanation:    fq.Explanation,
			Difficulty:     fq.Difficulty,
			BloomLevel:     fq.BloomLevel,
			Marks:          toInt(fq.Marks),
			SourceWeekID:   toUint(fq.SourceWeekID),
			SourceModuleID: toUint(fq.SourceModuleID),
		}
		for _, fo := range fq.Options {
			q.Options = append(q.Options, GeneratedOption{
				OptionText: fo.OptionText,
				IsCorrect:  toBool(fo.IsCorrect),
				OrderIndex: toInt(fo.OrderIndex),
			})
		}
		questions = append(questions, q)
	}

	// Validate and enrich questions
	validated, err := validateQuestions(questions, theoryContent)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	return validated, nil
}

// =====================================================
// Validation
// =====================================================

// validateQuestions validates generated questions meet requirements
func validateQuestions(questions []GeneratedQuestion, theoryContent []ModuleContent) ([]GeneratedQuestion, error) {
	validated := make([]GeneratedQuestion, 0, len(questions))

	// Build module ID lookup
	moduleIDs := make(map[uint]bool)
	weekIDs := make(map[uint]bool)
	for _, mc := range theoryContent {
		moduleIDs[mc.TheoryModuleID] = true
		weekIDs[mc.TheoryWeekID] = true
	}

	for i, q := range questions {
		// Validate question text
		if strings.TrimSpace(q.QuestionText) == "" {
			log.Printf("Skipping question %d: empty question text", i)
			continue
		}

		// Validate question type
		validTypes := map[string]bool{"mcq": true, "multi_select": true, "true_false": true, "short_answer": true, "fill_blank": true}
		if !validTypes[q.QuestionType] {
			log.Printf("Skipping question %d: invalid type '%s'", i, q.QuestionType)
			continue
		}

		// Validate difficulty
		validDifficulties := map[string]bool{"easy": true, "medium": true, "hard": true}
		if !validDifficulties[q.Difficulty] {
			q.Difficulty = "medium" // Default to medium
		}

		// Validate Bloom level
		validBloomLevels := map[string]bool{"remember": true, "understand": true, "apply": true, "analyze": true, "evaluate": true, "create": true}
		if !validBloomLevels[q.BloomLevel] {
			q.BloomLevel = "understand" // Default to understand
		}

		// Validate options for MCQ/multi_select/true_false
		if q.QuestionType == "mcq" || q.QuestionType == "multi_select" || q.QuestionType == "true_false" {
			if len(q.Options) < 2 {
				log.Printf("Skipping question %d: insufficient options (%d)", i, len(q.Options))
				continue
			}

			// Ensure at least one correct answer
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

		// Validate source module/week IDs
		if q.SourceModuleID > 0 && !moduleIDs[q.SourceModuleID] {
			// Map to first available module if source not found
			if len(theoryContent) > 0 {
				q.SourceModuleID = theoryContent[0].TheoryModuleID
				q.SourceWeekID = theoryContent[0].TheoryWeekID
			}
		}

		// Set default marks if not specified
		if q.Marks <= 0 {
			q.Marks = 1
		}

		validated = append(validated, q)
	}

	if len(validated) == 0 {
		return nil, fmt.Errorf("no valid questions generated")
	}

	return validated, nil
}

// =====================================================
// Helper Functions
// =====================================================

// BuildTheoryContext builds structured context from theory modules
func BuildTheoryContext(modules []models.TheoryWeek) []ModuleContent {
	var content []ModuleContent

	for _, week := range modules {
		for _, module := range week.Modules {
			content = append(content, ModuleContent{
				WeekName:       week.WeekName,
				ModuleName:     module.ModuleName,
				Description:    module.Description,
				Content:        module.Content,
				TheoryWeekID:   week.ID,
				TheoryModuleID: module.ID,
			})
		}
	}

	return content
}

// EstimateTotalTokens estimates total tokens for request
func EstimateTotalTokens(content []ModuleContent, teacherPrompt string) int {
	var totalChars int
	for _, mc := range content {
		totalChars += len(mc.WeekName) + len(mc.ModuleName) + len(mc.Description) + len(mc.Content)
	}
	totalChars += len(teacherPrompt)

	// Add overhead for prompt template (~2000 chars)
	totalChars += 2000

	return totalChars / 4 // 4 chars ≈ 1 token
}

// ChunkContentByTokenBudget splits content if it exceeds token budget
func ChunkContentByTokenBudget(content []ModuleContent, maxTokens int) ([][]ModuleContent, error) {
	if len(content) == 0 {
		return nil, fmt.Errorf("empty content")
	}

	var chunks [][]ModuleContent
	var currentChunk []ModuleContent
	currentTokens := 0

	// Reserve tokens for prompt template
	reservedTokens := 2000
	availableTokens := maxTokens - reservedTokens

	for _, mc := range content {
		moduleTokens := EstimateTotalTokens([]ModuleContent{mc}, "")

		if currentTokens+moduleTokens > availableTokens && len(currentChunk) > 0 {
			// Start new chunk
			chunks = append(chunks, currentChunk)
			currentChunk = nil
			currentTokens = 0
		}

		currentChunk = append(currentChunk, mc)
		currentTokens += moduleTokens
	}

	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks, nil
}

// cleanJSONResponse strips markdown code fences and removes trailing commas
// that Gemini sometimes includes in generated JSON.
func cleanJSONResponse(s string) string {
	// Strip markdown code fences (```json ... ``` or ``` ... ```)
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Remove opening fence (with optional language tag)
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
		// Remove closing fence
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}

	// Remove trailing commas before } or ] (invalid in strict JSON)
	// Matches: comma, optional whitespace/newlines, then } or ]
	re := regexp.MustCompile(`,\s*([}\]])`)
	s = re.ReplaceAllString(s, "$1")

	return s
}

// toUint converts an interface{} (string or number) to uint
func toUint(v interface{}) uint {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return uint(val)
	case json.Number:
		n, _ := val.Int64()
		if n < 0 {
			return 0
		}
		return uint(n)
	case string:
		n, _ := strconv.ParseUint(val, 10, 64)
		return uint(n)
	default:
		return 0
	}
}

// toInt converts an interface{} (string or number) to int
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case json.Number:
		n, _ := val.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(val)
		return n
	default:
		return 0
	}
}

// toBool converts an interface{} (string or bool or number) to bool
func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case string:
		b, _ := strconv.ParseBool(val)
		return b
	default:
		return false
	}
}

// GetAIProvider returns the appropriate AI provider based on configuration
func GetAIProvider(providerType, apiKey, model string, maxTokens int, temperature float64, rateLimitRPM int) (AIProvider, error) {
	switch strings.ToLower(providerType) {
	case "gemini", "google":
		return NewGeminiProvider(apiKey, model, maxTokens, temperature, rateLimitRPM), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s (supported: gemini)", providerType)
	}
}
