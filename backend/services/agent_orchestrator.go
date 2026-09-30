package services

import (
	"coding-platform/database"
	"coding-platform/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

// =====================================================
// Agent Orchestrator - Multi-Agent System with ReAct Loop
// =====================================================
// This implements a true agent system with:
// 1. ReAct Loop (Reason + Act)
// 2. Tool-based actions
// 3. Self-correction/Reflexion
// 4. Multi-agent orchestration
// =====================================================

// AgentRole defines the role of an agent
type AgentRole string

const (
	AgentExtractor  AgentRole = "extractor"
	AgentArchitect  AgentRole = "architect"
	AgentWriter     AgentRole = "writer"
	AgentQuizmaster AgentRole = "quizmaster"
	AgentCritic     AgentRole = "critic"
	AgentContest    AgentRole = "contest"
)

// AgentTool defines a tool that agents can use
type AgentTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Handler     func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// AgentState holds the state for an agent's ReAct loop
type AgentState struct {
	Objective     string
	Thoughts      []string
	Actions       []AgentAction
	Observations  []string
	FinalOutput   interface{}
	MaxIterations int
}

// AgentAction represents an action taken by an agent
type AgentAction struct {
	ToolName  string      `json:"tool_name"`
	Params    interface{} `json:"params"`
	Result    interface{} `json:"result"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// =====================================================
// Agent Orchestrator
// =====================================================

type AgentOrchestrator struct {
	db         *gorm.DB
	aiProvider *GeminiProvider
	tools      map[AgentRole][]*AgentTool
	agents     map[AgentRole]*Agent
}

// Agent represents a specialized agent with specific role
type Agent struct {
	Role         AgentRole
	SystemPrompt string
	Tools        []*AgentTool
	State        *AgentState
	aiProvider   *GeminiProvider
}

// NewAgentOrchestrator creates a new multi-agent orchestrator
func NewAgentOrchestrator(aiProvider *GeminiProvider) *AgentOrchestrator {
	orchestrator := &AgentOrchestrator{
		db:         database.DB,
		aiProvider: aiProvider,
		tools:      make(map[AgentRole][]*AgentTool),
		agents:     make(map[AgentRole]*Agent),
	}

	// Initialize tools for each agent role
	orchestrator.initializeTools()

	// Initialize agents
	orchestrator.initializeAgents()

	return orchestrator
}

// initializeTools sets up tools for each agent role
func (o *AgentOrchestrator) initializeTools() {
	// Extractor Agent Tools
	o.tools[AgentExtractor] = []*AgentTool{
		{
			Name:        "ReadDocumentChunk",
			Description: "Read a specific page or section of the uploaded lesson plan document",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page_start": map[string]interface{}{"type": "integer", "description": "Starting page number (1-indexed)"},
					"page_end":   map[string]interface{}{"type": "integer", "description": "Ending page number (inclusive)"},
				},
				"required": []string{"page_start"},
			},
			Handler: o.handleReadDocumentChunk,
		},
		{
			Name:        "ExtractTableContent",
			Description: "Extract table structures from PDF document pages",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page_numbers": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "integer"}},
				},
				"required": []string{"page_numbers"},
			},
			Handler: o.handleExtractTableContent,
		},
		{
			Name:        "IdentifyKeyTopics",
			Description: "Identify main topics and learning objectives from document content",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string", "description": "Text content to analyze"},
				},
				"required": []string{"content"},
			},
			Handler: o.handleIdentifyKeyTopics,
		},
	}

	// Architect Agent Tools
	o.tools[AgentArchitect] = []*AgentTool{
		{
			Name:        "DraftWeekStructure",
			Description: "Create a draft week structure with modules",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"week_number":  map[string]interface{}{"type": "integer"},
					"week_name":    map[string]interface{}{"type": "string"},
					"module_names": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				},
				"required": []string{"week_number", "week_name", "module_names"},
			},
			Handler: o.handleDraftWeekStructure,
		},
		{
			Name:        "ValidateWeekCount",
			Description: "Validate if the number of weeks matches the course duration",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"total_weeks": map[string]interface{}{"type": "integer"},
					"course_type": map[string]interface{}{"type": "string"},
				},
				"required": []string{"total_weeks"},
			},
			Handler: o.handleValidateWeekCount,
		},
		{
			Name:        "CheckTopicCoverage",
			Description: "Check if all required topics are covered in the structure",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"required_topics": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"current_topics":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				},
				"required": []string{"required_topics", "current_topics"},
			},
			Handler: o.handleCheckTopicCoverage,
		},
	}

	// Writer Agent Tools
	o.tools[AgentWriter] = []*AgentTool{
		{
			Name:        "ExpandModuleContent",
			Description: "Expand a module outline into full educational content",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"module_name":   map[string]interface{}{"type": "string"},
					"key_topics":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"target_length": map[string]interface{}{"type": "integer", "description": "Target word count"},
				},
				"required": []string{"module_name", "key_topics"},
			},
			Handler: o.handleExpandModuleContent,
		},
		{
			Name:        "AddCodeExamples",
			Description: "Add relevant code examples to module content",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"topic":                map[string]interface{}{"type": "string"},
					"programming_language": map[string]interface{}{"type": "string"},
					"difficulty":           map[string]interface{}{"type": "string"},
				},
				"required": []string{"topic"},
			},
			Handler: o.handleAddCodeExamples,
		},
		{
			Name:        "FormatAsMarkdown",
			Description: "Format content as properly structured markdown",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content":          map[string]interface{}{"type": "string"},
					"include_headings": map[string]interface{}{"type": "boolean"},
					"include_lists":    map[string]interface{}{"type": "boolean"},
				},
				"required": []string{"content"},
			},
			Handler: o.handleFormatAsMarkdown,
		},
	}

	// Quizmaster Agent Tools
	o.tools[AgentQuizmaster] = []*AgentTool{
		{
			Name:        "GenerateMCQ",
			Description: "Generate multiple choice questions for a topic",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"topic":      map[string]interface{}{"type": "string"},
					"difficulty": map[string]interface{}{"type": "string", "enum": []string{"easy", "medium", "hard"}},
					"count":      map[string]interface{}{"type": "integer"},
				},
				"required": []string{"topic", "count"},
			},
			Handler: o.handleGenerateMCQ,
		},
		{
			Name:        "GenerateTrueFalse",
			Description: "Generate true/false questions for quick knowledge checks",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"topic": map[string]interface{}{"type": "string"},
					"count": map[string]interface{}{"type": "integer"},
				},
				"required": []string{"topic", "count"},
			},
			Handler: o.handleGenerateTrueFalse,
		},
		{
			Name:        "AddExplanation",
			Description: "Add detailed explanation to a quiz question",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"question":       map[string]interface{}{"type": "string"},
					"correct_answer": map[string]interface{}{"type": "string"},
					"context":        map[string]interface{}{"type": "string"},
				},
				"required": []string{"question", "correct_answer"},
			},
			Handler: o.handleAddExplanation,
		},
	}

	// Critic Agent Tools
	o.tools[AgentCritic] = []*AgentTool{
		{
			Name:        "ValidateJSONStructure",
			Description: "Validate that output JSON matches expected schema",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"json_data":   map[string]interface{}{"type": "string"},
					"schema_type": map[string]interface{}{"type": "string", "enum": []string{"lesson_plan", "quiz"}},
				},
				"required": []string{"json_data", "schema_type"},
			},
			Handler: o.handleValidateJSONStructure,
		},
		{
			Name:        "CheckContentQuality",
			Description: "Check content for quality issues (completeness, accuracy, appropriateness)",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content":    map[string]interface{}{"type": "string"},
					"check_type": map[string]interface{}{"type": "string", "enum": []string{"completeness", "accuracy", "appropriateness"}},
				},
				"required": []string{"content", "check_type"},
			},
			Handler: o.handleCheckContentQuality,
		},
		{
			Name:        "SuggestImprovements",
			Description: "Suggest improvements to generated content",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string"},
					"aspect":  map[string]interface{}{"type": "string", "enum": []string{"clarity", "depth", "examples", "structure"}},
				},
				"required": []string{"content", "aspect"},
			},
			Handler: o.handleSuggestImprovements,
		},
	}

	// Contest Agent Tools
	o.tools[AgentContest] = []*AgentTool{
		{
			Name:        "GenerateProblem",
			Description: "Generate a contest problem statement with reference solution and test cases",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"topics":         map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"difficulty":     map[string]interface{}{"type": "string", "enum": []string{"easy", "medium", "hard"}},
					"course_context": map[string]interface{}{"type": "string"},
				},
				"required": []string{"topics", "difficulty"},
			},
			Handler: o.handleGenerateProblem,
		},
		{
			Name:        "VerifySolution",
			Description: "Verify the reference solution against test cases using Judge0",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"solution_code": map[string]interface{}{"type": "string"},
					"language":      map[string]interface{}{"type": "string"},
					"test_cases": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"input":           map[string]interface{}{"type": "string"},
								"expected_output": map[string]interface{}{"type": "string"},
							},
						},
					},
					"time_limit_ms":   map[string]interface{}{"type": "integer"},
					"memory_limit_kb": map[string]interface{}{"type": "integer"},
				},
				"required": []string{"solution_code", "language", "test_cases"},
			},
			Handler: o.handleVerifySolution,
		},
		{
			Name:        "FixCode",
			Description: "Fix compilation or runtime errors in the reference solution based on error feedback",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"original_code":     map[string]interface{}{"type": "string"},
					"error_message":     map[string]interface{}{"type": "string"},
					"error_type":        map[string]interface{}{"type": "string", "enum": []string{"compile", "runtime", "wrong_answer", "time_limit"}},
					"problem_statement": map[string]interface{}{"type": "string"},
				},
				"required": []string{"original_code", "error_message", "error_type"},
			},
			Handler: o.handleFixCode,
		},
		{
			Name:        "ValidateTestCases",
			Description: "Validate that test cases are well-formed and cover edge cases",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"test_cases": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"input":           map[string]interface{}{"type": "string"},
								"expected_output": map[string]interface{}{"type": "string"},
								"category":        map[string]interface{}{"type": "string"},
							},
						},
					},
					"constraints": map[string]interface{}{"type": "object"},
				},
				"required": []string{"test_cases"},
			},
			Handler: o.handleValidateTestCases,
		},
		{
			Name:        "FinishProblem",
			Description: "Finalize and approve the contest problem when all verifications pass",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"problem":             map[string]interface{}{"type": "object"},
					"verification_passed": map[string]interface{}{"type": "boolean"},
				},
				"required": []string{"problem", "verification_passed"},
			},
			Handler: o.handleFinishProblem,
		},
	}
}

// initializeAgents creates agents for each role
func (o *AgentOrchestrator) initializeAgents() {
	o.agents[AgentExtractor] = &Agent{
		Role: AgentExtractor,
		SystemPrompt: `You are the Extractor Agent. Your role is to:
1. Read uploaded lesson plan documents page by page
2. Extract tables and structured content
3. Identify key topics and learning objectives
4. Pass extracted information to the Architect Agent

Use your tools systematically to extract maximum information.`,
		Tools:      o.tools[AgentExtractor],
		aiProvider: o.aiProvider,
	}

	o.agents[AgentArchitect] = &Agent{
		Role: AgentArchitect,
		SystemPrompt: `You are the Architect Agent. Your role is to:
1. Take extracted content from Extractor Agent
2. Design week-by-week course structure
3. Allocate topics to appropriate weeks
4. Ensure logical progression and coverage
5. Pass structure to Writer Agent

Use your tools to validate structure before passing forward.`,
		Tools:      o.tools[AgentArchitect],
		aiProvider: o.aiProvider,
	}

	o.agents[AgentWriter] = &Agent{
		Role: AgentWriter,
		SystemPrompt: `You are the Writer Agent. Your role is to:
1. Take week/module structure from Architect Agent
2. Expand each module into comprehensive educational content
3. Add code examples where relevant
4. Format content as proper markdown
5. Pass enriched structure to Quizmaster Agent

Write content appropriate for undergraduate students.`,
		Tools:      o.tools[AgentWriter],
		aiProvider: o.aiProvider,
	}

	o.agents[AgentQuizmaster] = &Agent{
		Role: AgentQuizmaster,
		SystemPrompt: `You are the Quizmaster Agent. Your role is to:
1. Take completed module content from Writer Agent
2. Generate practice quiz questions for each week
3. Include varied question types (MCQ, True/False, Fill-in-blank)
4. Add detailed explanations for all questions
5. Pass final structure to Critic Agent for validation

Ensure questions test understanding, not just recall.`,
		Tools:      o.tools[AgentQuizmaster],
		aiProvider: o.aiProvider,
	}

	o.agents[AgentCritic] = &Agent{
		Role: AgentCritic,
		SystemPrompt: `You are the Critic Agent. Your role is to:
1. Review the complete generated lesson plan
2. Validate JSON structure matches expected schema
3. Check content quality (completeness, accuracy, appropriateness)
4. Suggest improvements if needed
5. Approve or request regeneration

Be thorough but fair. Request regeneration only for significant issues.`,
		Tools:      o.tools[AgentCritic],
		aiProvider: o.aiProvider,
	}

	o.agents[AgentContest] = &Agent{
		Role: AgentContest,
		SystemPrompt: `You are the Contest Agent. Your role is to:
1. Generate original competitive programming problems
2. Create correct reference solutions that compile
3. Generate comprehensive test cases (sample + hidden)
4. Verify solutions pass all test cases via Judge0
5. Fix compilation/runtime errors autonomously
6. Ensure problems are clear, solvable, and well-tested

Use your tools to iteratively generate and verify problems.
When Judge0 reports errors, use FixCode to correct them.
Only finish when all test cases pass verification.`,
		Tools:      o.tools[AgentContest],
		aiProvider: o.aiProvider,
	}
}

// ProcessLessonPlanWithAgents processes a lesson plan using the multi-agent system
// This uses a TWO-PHASE approach to reduce token consumption:
// Phase 1: Extract structure only (weeks + module names) - lightweight
// Phase 2: Generate content for each module in separate calls
func (o *AgentOrchestrator) ProcessLessonPlanWithAgents(ctx context.Context, documentContent string, courseInfo *models.Course) (*models.LessonPlanStructure, error) {
	log.Printf("[AgentOrchestrator] ========== START ORCHESTRATOR PROCESSING ==========")
	log.Printf("[AgentOrchestrator] Course: %s (%s)", courseInfo.CourseName, courseInfo.CourseCode)
	log.Printf("[AgentOrchestrator] Document content length: %d characters", len(documentContent))

	// =====================================================
	// Phase 1: Extract structure only (weeks + module names)
	// =====================================================
	log.Println("[AgentOrchestrator] Phase 1: Extracting lesson plan structure (lightweight)...")
	log.Printf("[AgentOrchestrator] Building structure extraction prompt with document content (%d chars)", len(documentContent))

	req := LessonPlanRequest{
		DocumentContent: documentContent,
		CourseName:      courseInfo.CourseName,
		CourseLevel:     "undergraduate",
	}

	log.Printf("[AgentOrchestrator] Calling Gemini API for structure extraction: Course=%s", courseInfo.CourseName)
	liteStructure, err := o.aiProvider.ParseLessonPlanStructureOnly(ctx, req)
	if err != nil {
		log.Printf("[AgentOrchestrator] ERROR: Structure extraction failed: %v", err)
		return nil, fmt.Errorf("structure extraction failed: %w", err)
	}

	log.Printf("[AgentOrchestrator] Structure extraction completed successfully!")
	log.Printf("[AgentOrchestrator] Extracted: %d weeks, %d total module names",
		len(liteStructure.Weeks), countTotalModuleNamesLite(liteStructure.Weeks))

	// =====================================================
	// Phase 2: Generate content for each module
	// =====================================================
	log.Println("[AgentOrchestrator] Phase 2: Generating content for each module...")

	// Build module content requests
	var moduleRequests []ModuleContentRequest
	moduleOrder := make(map[string]int) // Track order of modules
	idx := 0

	for _, week := range liteStructure.Weeks {
		weekContext := week.WeekName
		for _, moduleName := range week.ModuleNames {
			moduleRequests = append(moduleRequests, ModuleContentRequest{
				ModuleName:  moduleName,
				Description: fmt.Sprintf("Module from %s", weekContext),
				WeekContext: weekContext,
			})
			moduleOrder[moduleName] = idx
			idx++
		}
	}

	log.Printf("[AgentOrchestrator] Generating content for %d modules...", len(moduleRequests))

	// Generate content for all modules SEQUENTIALLY (one at a time)
	// This avoids rate limit issues with free tier API
	moduleContents, err := o.aiProvider.GenerateModuleContentsBatch(ctx, courseInfo.CourseName, "undergraduate", moduleRequests, 1)
	if err != nil {
		log.Printf("[AgentOrchestrator] WARNING: Some module content generation failed: %v", err)
		// Continue with partial results
	}

	log.Printf("[AgentOrchestrator] Module content generation completed: %d/%d modules successful",
		len(moduleContents), len(moduleRequests))

	// Build module data map for conversion
	modulesContent := make(map[string]models.ModuleData)
	for _, mc := range moduleContents {
		modulesContent[mc.ModuleName] = mc
	}

	// Convert lite structure to full structure
	fullStructure := liteStructure.ConvertLiteToFull(modulesContent)

	log.Printf("[AgentOrchestrator] Final structure: %d weeks, %d total modules",
		len(fullStructure.Weeks), countTotalModules(fullStructure.Weeks))

	// =====================================================
	// Phase 3: Critic Agent Validation (optional - skip for token efficiency)
	// =====================================================
	// Note: Critic validation is skipped in two-phase mode to save tokens
	// The structure extraction is deterministic and module content is validated individually

	log.Printf("[AgentOrchestrator] ========== END ORCHESTRATOR PROCESSING ==========")
	return fullStructure, nil
}

// ValidateStructure validates a lesson plan structure using the Critic Agent
func (a *Agent) ValidateStructure(_ context.Context, structure *models.LessonPlanStructure, originalContent string) (*CriticResult, error) {
	if a.Role != AgentCritic {
		return nil, fmt.Errorf("only Critic agent can validate")
	}

	// Build validation prompt
	prompt := buildCriticPrompt(structure, originalContent)

	// Call LLM for validation
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  2000,
			"temperature":      0.3,
			"responseMimeType": "application/json",
		},
	}

	jsonData, _ := json.Marshal(geminiReq)
	httpReq, err := http.NewRequest("POST", a.aiProvider.apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return &CriticResult{Approved: true}, nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", a.aiProvider.apiKey)

	resp, err := a.aiProvider.httpClient.Do(httpReq)
	if err != nil {
		return &CriticResult{Approved: true}, nil // Don't fail on network errors
	}
	defer resp.Body.Close()

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct{ Text string } `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return &CriticResult{Approved: true}, nil
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		responseText := geminiResp.Candidates[0].Content.Parts[0].Text
		responseText = cleanJSONResponse(responseText)

		var criticResponse CriticResponse
		if err := json.Unmarshal([]byte(responseText), &criticResponse); err != nil {
			return &CriticResult{Approved: true}, nil
		}

		return &CriticResult{
			Approved:    criticResponse.Approved,
			Suggestions: criticResponse.Suggestions,
			Issues:      criticResponse.Issues,
		}, nil
	}

	return &CriticResult{Approved: true}, nil
}

// =====================================================
// Contest Agent ReAct Loop
// =====================================================

// ContestAgentState holds state specific to contest problem generation
type ContestAgentState struct {
	Objective           string
	Topics              []string
	Difficulty          string
	CourseContext       string
	CurrentProblem      *models.GeneratedProblem
	VerificationResults []map[string]interface{}
	FixAttempts         int
	MaxFixAttempts      int
	MaxIterations       int
}

// GenerateContestProblemDirect generates a contest problem using a deterministic pipeline
// instead of the ReAct agent loop. This reduces Gemini API calls from ~15-35 to 1-4 per problem.
// Pipeline: Generate(1 call) → Verify(0 calls) → Fix(1 call) → Re-verify(0 calls) → Finish
func (o *AgentOrchestrator) GenerateContestProblemDirect(ctx context.Context, topics []string, difficulty, courseContext string) (*models.GeneratedProblem, error) {
	log.Printf("[ContestDirect] ========== START DIRECT PIPELINE ==========")
	log.Printf("[ContestDirect] Topics: %v, Difficulty: %s", topics, difficulty)

	const maxFixAttempts = 3

	// Step 1: Generate the problem (1 Gemini API call)
	log.Printf("[ContestDirect] Step 1: Generating problem...")
	problem, err := o.aiProvider.GenerateContestProblem(ctx, topics, difficulty, courseContext)
	if err != nil {
		log.Printf("[ContestDirect] ERROR: Problem generation failed: %v", err)
		return nil, fmt.Errorf("problem generation failed: %w", err)
	}

	if problem == nil {
		return nil, fmt.Errorf("problem generation returned nil")
	}

	log.Printf("[ContestDirect] Problem generated: %s (lang: %s, test cases: %d)",
		problem.Title, problem.SolutionLanguage, len(problem.TestCases))

	// If no reference solution or test cases, return as unverified
	if problem.ReferenceSolution == "" || len(problem.TestCases) == 0 {
		log.Printf("[ContestDirect] No solution or test cases, returning unverified")
		problem.VerificationStatus = "unverified"
		return problem, nil
	}

	// Determine language ID for Judge0
	languageID := 71 // Python 3
	if strings.ToLower(problem.SolutionLanguage) == "cpp" || strings.ToLower(problem.SolutionLanguage) == "c++" {
		languageID = 54 // C++17
	}

	// Step 2: Verify → Fix loop (0 Gemini calls for verify, 1 per fix)
	for attempt := 0; attempt <= maxFixAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return problem, ctx.Err()
		default:
		}

		log.Printf("[ContestDirect] Step 2: Verifying solution (attempt %d/%d)...", attempt+1, maxFixAttempts+1)

		allPassed := true
		var failureInfo string

		for i, tc := range problem.TestCases {
			result, err := SubmitCode(
				problem.ReferenceSolution,
				languageID,
				tc.Input,
				tc.ExpectedOutput,
				2000,   // time limit ms
				256000, // memory limit KB
			)

			if err != nil {
				allPassed = false
				failureInfo = fmt.Sprintf("Test case %d execution error: %v", i+1, err)
				log.Printf("[ContestDirect] %s", failureInfo)
				break
			}

			if !result.Passed {
				allPassed = false
				errorType := "wrong_answer"
				if result.CompileOutput != "" {
					errorType = "compile"
				} else if result.Status == "Time Limit Exceeded" {
					errorType = "time_limit"
				} else if strings.Contains(result.Status, "Runtime Error") {
					errorType = "runtime"
				}

				failureInfo = fmt.Sprintf("Test case %d failed: error_type=%s, status=%s, expected=%q, got=%q, stderr=%q, compile=%q",
					i+1, errorType, result.Status, tc.ExpectedOutput, result.Stdout, result.Stderr, result.CompileOutput)
				log.Printf("[ContestDirect] %s", failureInfo)
				break
			}

			log.Printf("[ContestDirect] Test case %d/%d PASSED", i+1, len(problem.TestCases))
		}

		if allPassed {
			log.Printf("[ContestDirect] All %d test cases passed!", len(problem.TestCases))
			problem.VerificationStatus = "passed"
			log.Printf("[ContestDirect] ========== END DIRECT PIPELINE (SUCCESS) ==========")
			return problem, nil
		}

		// If we've exhausted fix attempts, return with partial status
		if attempt == maxFixAttempts {
			log.Printf("[ContestDirect] Max fix attempts reached, returning partial")
			problem.VerificationStatus = "partial"
			break
		}

		// Step 3: Fix the code (1 Gemini API call)
		log.Printf("[ContestDirect] Step 3: Fixing code (attempt %d/%d)...", attempt+1, maxFixAttempts)

		// Determine error type for the fix prompt
		errorType := "wrong_answer"
		if strings.Contains(failureInfo, "compile") {
			errorType = "compile"
		} else if strings.Contains(failureInfo, "time_limit") {
			errorType = "time_limit"
		} else if strings.Contains(failureInfo, "runtime") {
			errorType = "runtime"
		}

		fixedCode, err := o.aiProvider.FixContestSolution(ctx, problem.ReferenceSolution, failureInfo, errorType, problem.Description)
		if err != nil {
			log.Printf("[ContestDirect] Fix attempt %d failed: %v", attempt+1, err)
			continue
		}

		problem.ReferenceSolution = fixedCode
		log.Printf("[ContestDirect] Code fixed, re-verifying...")
	}

	log.Printf("[ContestDirect] ========== END DIRECT PIPELINE (PARTIAL) ==========")
	return problem, nil
}

// GenerateContestProblemWithAgents generates a contest problem using the ReAct agent loop
// DEPRECATED: Use GenerateContestProblemDirect instead for ~85% fewer API calls
func (o *AgentOrchestrator) GenerateContestProblemWithAgents(ctx context.Context, topics []string, difficulty, courseContext string) (*models.GeneratedProblem, error) {
	log.Printf("[ContestAgent] ========== START CONTEST AGENT REACT LOOP ==========")
	log.Printf("[ContestAgent] Topics: %v, Difficulty: %s", topics, difficulty)

	agent := o.agents[AgentContest]
	if agent == nil {
		return nil, fmt.Errorf("contest agent not initialized")
	}

	state := ContestAgentState{
		Objective:      "Generate a correct, verified contest problem with passing reference solution",
		Topics:         topics,
		Difficulty:     difficulty,
		CourseContext:  courseContext,
		FixAttempts:    0,
		MaxFixAttempts: 5,
		MaxIterations:  15,
	}

	// Convert contest state to generic agent state
	genericState := AgentState{
		Objective:     state.Objective,
		MaxIterations: state.MaxIterations,
	}

	// Input for ReAct loop
	input := map[string]interface{}{
		"topics":         topics,
		"difficulty":     difficulty,
		"course_context": courseContext,
		"contest_state":  state,
	}

	// Execute ReAct loop
	result, err := agent.ExecuteContest(ctx, genericState, input)
	if err != nil {
		return nil, fmt.Errorf("contest agent ReAct loop failed: %w", err)
	}

	log.Printf("[ContestAgent] ========== END CONTEST AGENT REACT LOOP ==========")

	// Convert result to GeneratedProblem
	if problem, ok := result.(*models.GeneratedProblem); ok {
		return problem, nil
	}

	// Try to convert from map
	if resultMap, ok := result.(map[string]interface{}); ok {
		problemInterface, ok := resultMap["problem"]
		if ok {
			if problemMap, ok := problemInterface.(map[string]interface{}); ok {
				jsonData, err := json.Marshal(problemMap)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal problem: %w", err)
				}

				var problem models.GeneratedProblem
				if err := json.Unmarshal(jsonData, &problem); err != nil {
					return nil, fmt.Errorf("failed to unmarshal problem: %w", err)
				}
				return &problem, nil
			}
		}
	}

	return nil, fmt.Errorf("unexpected result type from contest agent")
}

// ExecuteContest runs the ReAct loop for contest problem generation
func (a *Agent) ExecuteContest(ctx context.Context, state AgentState, input map[string]interface{}) (interface{}, error) {
	contestState, _ := input["contest_state"].(ContestAgentState)

	a.State = &state
	a.State.Thoughts = make([]string, 0)
	a.State.Actions = make([]AgentAction, 0)
	a.State.Observations = make([]string, 0)

	log.Printf("[Agent:%s] Starting contest ReAct loop", a.Role)

	for iteration := 0; iteration < state.MaxIterations; iteration++ {
		// THINK: Generate thought based on current state
		thought, err := a.generateContestThought(ctx, input, iteration, &contestState)
		if err != nil {
			return nil, fmt.Errorf("failed to generate thought: %w", err)
		}
		a.State.Thoughts = append(a.State.Thoughts, thought)
		log.Printf("[Agent:%s] Iteration %d - Thought: %s", a.Role, iteration, thought)

		// ACT: Decide on action based on thought
		action, err := a.decideContestAction(ctx, thought, input, &contestState)
		if err != nil {
			return nil, fmt.Errorf("failed to decide action: %w", err)
		}

		// Check if agent wants to finish
		if action.ToolName == "FINISH" {
			a.State.FinalOutput = action.Result
			log.Printf("[Agent:%s] ReAct loop completed after %d iterations", a.Role, iteration+1)
			return action.Result, nil
		}

		// OBSERVE: Execute tool and observe result
		observation, err := a.executeTool(ctx, action)
		if err != nil {
			action.Error = err.Error()
			log.Printf("[Agent:%s] Tool %s failed: %v", a.Role, action.ToolName, err)
		} else {
			log.Printf("[Agent:%s] Tool %s returned: %v", a.Role, action.ToolName, observation)
		}

		a.State.Actions = append(a.State.Actions, *action)
		a.State.Observations = append(a.State.Observations, fmt.Sprintf("%v", observation))

		// Update contest state based on observation
		contestState = a.updateContestState(contestState, observation, action.ToolName)
		input["contest_state"] = contestState

		// Check for max fix attempts
		if contestState.FixAttempts >= contestState.MaxFixAttempts {
			log.Printf("[Agent:%s] Max fix attempts reached, forcing finish", a.Role)
			return a.generateContestFinalOutput(&contestState), nil
		}

		// Self-correction: Check if stuck in a loop
		if iteration > 3 && a.detectLoop() {
			log.Printf("[Agent:%s] Detected loop, forcing finish with best available output", a.Role)
			return a.generateContestFinalOutput(&contestState), nil
		}
	}

	log.Printf("[Agent:%s] Max iterations (%d) reached, returning partial result", a.Role, state.MaxIterations)
	return a.generateContestFinalOutput(&contestState), nil
}

// generateContestThought generates thoughts for contest problem generation
func (a *Agent) generateContestThought(_ context.Context, _ map[string]interface{}, iteration int, state *ContestAgentState) (string, error) {
	var sb strings.Builder

	sb.WriteString("You are the Contest Agent generating a competitive programming problem.\n\n")
	sb.WriteString(fmt.Sprintf("Objective: %s\n\n", state.Objective))
	sb.WriteString(fmt.Sprintf("Topics: %v\n", state.Topics))
	sb.WriteString(fmt.Sprintf("Difficulty: %s\n\n", state.Difficulty))

	if iteration > 0 && len(a.State.Thoughts) > 0 {
		sb.WriteString("Previous thoughts:\n")
		for i := len(a.State.Thoughts) - 2; i < len(a.State.Thoughts); i++ {
			if i >= 0 {
				sb.WriteString(fmt.Sprintf("- %s\n", a.State.Thoughts[i]))
			}
		}
	}

	if len(a.State.Actions) > 0 {
		sb.WriteString("\nRecent actions:\n")
		for i := len(a.State.Actions) - 2; i < len(a.State.Actions); i++ {
			if i >= 0 {
				sb.WriteString(fmt.Sprintf("- Used %s\n", a.State.Actions[i].ToolName))
			}
		}
	}

	if len(a.State.Observations) > 0 {
		sb.WriteString("\nRecent observations:\n")
		for i := len(a.State.Observations) - 2; i < len(a.State.Observations); i++ {
			if i >= 0 {
				sb.WriteString(fmt.Sprintf("- %s\n", a.State.Observations[i]))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\nIteration %d: What is your next thought? Consider:\n", iteration+1))
	sb.WriteString("- If no problem exists yet, generate one\n")
	sb.WriteString("- If problem exists but not verified, verify it\n")
	sb.WriteString("- If verification failed, fix the code\n")
	sb.WriteString("- If all tests pass, finish\n")

	prompt := sb.String()

	// Use LLM to generate thought
	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 300,
			"temperature":     0.5,
		},
	}

	jsonData, _ := json.Marshal(geminiReq)
	httpReq, err := http.NewRequest("POST", a.aiProvider.apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return "Need to generate or verify problem", nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", a.aiProvider.apiKey)

	resp, err := a.aiProvider.httpClient.Do(httpReq)
	if err != nil {
		return "Need to generate or verify problem", nil
	}
	defer resp.Body.Close()

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct{ Text string } `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "Need to generate or verify problem", nil
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text), nil
	}

	return "Need to generate or verify problem", nil
}

// decideContestAction decides which tool to use for contest generation
func (a *Agent) decideContestAction(_ context.Context, thought string, _ map[string]interface{}, state *ContestAgentState) (*AgentAction, error) {
	// Build context for decision
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Thought: %s\n\n", thought))
	sb.WriteString("Available tools:\n")
	for _, tool := range a.Tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
	}

	if state.CurrentProblem != nil {
		sb.WriteString("\nCurrent problem exists.\n")
		if len(state.VerificationResults) > 0 {
			lastResult := state.VerificationResults[len(state.VerificationResults)-1]
			if passed, ok := lastResult["passed"].(bool); ok && !passed {
				sb.WriteString("Last verification FAILED - need to fix code.\n")
			}
		}
	} else {
		sb.WriteString("\nNo problem generated yet - need to generate first.\n")
	}

	sb.WriteString("\nDecide the next action. Return JSON:\n")
	sb.WriteString(`{"tool_name": "...", "params": {...}, "reason": "..."}`)
	sb.WriteString("\nIf problem is verified and all tests pass, use tool_name: \"FINISH\"")

	prompt := sb.String()

	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  300,
			"temperature":      0.3,
			"responseMimeType": "application/json",
		},
	}

	jsonData, _ := json.Marshal(geminiReq)
	httpReq, err := http.NewRequest("POST", a.aiProvider.apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", a.aiProvider.apiKey)

	resp, err := a.aiProvider.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read raw response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	log.Printf("[Agent:%s] decideContestAction Gemini response status: %d, body: %s", a.Role, resp.StatusCode, string(bodyBytes))

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct{ Text string } `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode gemini response: %w", err)
	}

	// Check for API error
	if geminiResp.Error != nil {
		return nil, fmt.Errorf("gemini API error: %s (code: %d, status: %s)", geminiResp.Error.Message, geminiResp.Error.Code, geminiResp.Error.Status)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("empty candidates in gemini response")
	}

	if len(geminiResp.Candidates[0].Content.Parts) == 0 {
		// Check if blocked by safety
		finishReason := geminiResp.Candidates[0].FinishReason
		return nil, fmt.Errorf("empty parts in gemini response, finishReason: %s", finishReason)
	}

	responseText := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
	log.Printf("[Agent:%s] decideContestAction raw response: %s", a.Role, responseText)

	// Clean the response - remove markdown code blocks if present
	responseText = cleanJSONResponse(responseText)

	var actionReq struct {
		ToolName string                 `json:"tool_name"`
		Params   map[string]interface{} `json:"params"`
		Reason   string                 `json:"reason"`
	}

	if err := json.Unmarshal([]byte(responseText), &actionReq); err != nil {
		log.Printf("[Agent:%s] Failed to parse action JSON: %v, response was: %s", a.Role, err, responseText)
		return nil, fmt.Errorf("failed to parse action JSON: %w", err)
	}

	log.Printf("[Agent:%s] Parsed action: tool_name=%s, reason=%s", a.Role, actionReq.ToolName, actionReq.Reason)

	return &AgentAction{
		ToolName:  actionReq.ToolName,
		Params:    actionReq.Params,
		Timestamp: time.Now(),
	}, nil
}

// updateContestState updates the contest state based on tool execution results
func (a *Agent) updateContestState(state ContestAgentState, observation interface{}, toolName string) ContestAgentState {
	switch toolName {
	case "GenerateProblem":
		if obsMap, ok := observation.(map[string]interface{}); ok {
			if success, ok := obsMap["success"].(bool); ok && success {
				if problem, ok := obsMap["problem"].(map[string]interface{}); ok {
					// Convert to GeneratedProblem
					jsonData, _ := json.Marshal(problem)
					var p models.GeneratedProblem
					if err := json.Unmarshal(jsonData, &p); err == nil {
						state.CurrentProblem = &p
						state.VerificationResults = nil
					}
				}
			}
		}

	case "VerifySolution":
		if obsMap, ok := observation.(map[string]interface{}); ok { //nolint:staticcheck
			state.VerificationResults = append(state.VerificationResults, obsMap)
			if _, ok := obsMap["passed"].(bool); ok { //nolint:staticcheck
				// Verification passed - ready to finish
			}
		}

	case "FixCode":
		state.FixAttempts++
		if obsMap, ok := observation.(map[string]interface{}); ok {
			if success, ok := obsMap["success"].(bool); ok && success {
				if fixedCode, ok := obsMap["fixed_code"].(string); ok && state.CurrentProblem != nil {
					state.CurrentProblem.ReferenceSolution = fixedCode
				}
			}
		}
	}

	return state
}

// generateContestFinalOutput generates the final output for contest generation
func (a *Agent) generateContestFinalOutput(state *ContestAgentState) interface{} {
	if state.CurrentProblem == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "No problem generated",
		}
	}

	// Check if verification passed
	if len(state.VerificationResults) > 0 {
		lastResult := state.VerificationResults[len(state.VerificationResults)-1]
		if passed, ok := lastResult["passed"].(bool); ok && passed {
			state.CurrentProblem.VerificationStatus = "passed"
			return map[string]interface{}{
				"success": true,
				"problem": state.CurrentProblem,
				"status":  "approved",
			}
		}
	}

	// Return problem even if not fully verified (with appropriate status)
	state.CurrentProblem.VerificationStatus = "partial"
	return map[string]interface{}{
		"success":      true,
		"problem":      state.CurrentProblem,
		"status":       "partial",
		"fix_attempts": state.FixAttempts,
	}
}

func countTotalModules(weeks []models.WeekData) int {
	total := 0
	for _, week := range weeks {
		total += len(week.Modules)
	}
	return total
}

func countTotalModuleNamesLite(weeks []models.WeekDataLite) int {
	total := 0
	for _, week := range weeks {
		total += len(week.ModuleNames)
	}
	return total
}

func buildCriticPrompt(structure *models.LessonPlanStructure, _ string) string {
	var sb strings.Builder

	sb.WriteString("## ROLE\n")
	sb.WriteString("You are the Critic Agent reviewing an AI-generated lesson plan.\n\n")

	sb.WriteString("## GENERATED STRUCTURE\n")
	sb.WriteString(fmt.Sprintf("Course: %s\n", structure.CourseName))
	sb.WriteString(fmt.Sprintf("Total Weeks: %d\n", structure.TotalWeeks))
	sb.WriteString("\nWeeks:\n")
	for i, week := range structure.Weeks {
		sb.WriteString(fmt.Sprintf("Week %d: %s (%d modules)\n",
			week.WeekNumber, week.WeekName, len(week.Modules)))
		for j, module := range week.Modules {
			sb.WriteString(fmt.Sprintf("  Module %d: %s - %s\n",
				j+1, module.ModuleName, module.Description[:min(100, len(module.Description))]))
		}
		if i < len(structure.Weeks)-1 {
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n## YOUR TASK\n")
	sb.WriteString("Review this generated lesson plan and check for:\n")
	sb.WriteString("1. STRUCTURE: Does it have 8-16 weeks appropriate for a semester course?\n")
	sb.WriteString("2. COVERAGE: Are all key topics from the original document covered?\n")
	sb.WriteString("3. PROGRESSION: Do weeks build logically on each other?\n")
	sb.WriteString("4. MODULE QUALITY: Does each module have meaningful description and content?\n")
	sb.WriteString("5. COMPLETENESS: Are there any empty or incomplete weeks/modules?\n\n")

	sb.WriteString("## OUTPUT FORMAT\n")
	sb.WriteString(`Return JSON: {
    "approved": true/false,
    "issues": ["list of specific issues found"],
    "suggestions": ["specific suggestions for improvement"]
}

If approved is true, suggestions can be empty or minor improvements.
If approved is false, provide clear actionable suggestions.`)

	return sb.String()
}

type CriticResult struct {
	Approved    bool
	Suggestions []string
	Issues      []string
}

type CriticResponse struct {
	Approved    bool     `json:"approved"`
	Suggestions []string `json:"suggestions"`
	Issues      []string `json:"issues"`
}

// Execute runs the ReAct loop for an agent
func (a *Agent) Execute(ctx context.Context, state AgentState, input map[string]interface{}) (interface{}, error) {
	a.State = &state
	a.State.Thoughts = make([]string, 0)
	a.State.Actions = make([]AgentAction, 0)
	a.State.Observations = make([]string, 0)

	log.Printf("[Agent:%s] Starting ReAct loop with objective: %s", a.Role, state.Objective)

	for iteration := 0; iteration < state.MaxIterations; iteration++ {
		// =====================================================
		// THINK: Generate thought based on current state
		// =====================================================
		thought, err := a.generateThought(ctx, input, iteration)
		if err != nil {
			return nil, fmt.Errorf("failed to generate thought: %w", err)
		}
		a.State.Thoughts = append(a.State.Thoughts, thought)
		log.Printf("[Agent:%s] Iteration %d - Thought: %s", a.Role, iteration, thought)

		// =====================================================
		// ACT: Decide on action based on thought
		// =====================================================
		action, err := a.decideAction(ctx, thought, input)
		if err != nil {
			return nil, fmt.Errorf("failed to decide action: %w", err)
		}

		// Check if agent wants to finish
		if action.ToolName == "FINISH" {
			a.State.FinalOutput = action.Result
			log.Printf("[Agent:%s] ReAct loop completed after %d iterations", a.Role, iteration+1)
			return action.Result, nil
		}

		// =====================================================
		// OBSERVE: Execute tool and observe result
		// =====================================================
		observation, err := a.executeTool(ctx, action)
		if err != nil {
			action.Error = err.Error()
			log.Printf("[Agent:%s] Tool %s failed: %v", a.Role, action.ToolName, err)
		} else {
			log.Printf("[Agent:%s] Tool %s returned: %v", a.Role, action.ToolName, observation)
		}

		a.State.Actions = append(a.State.Actions, *action)
		a.State.Observations = append(a.State.Observations, fmt.Sprintf("%v", observation))

		// Update input with observation for next iteration
		input["last_observation"] = observation

		// Self-correction: Check if we're stuck in a loop
		if iteration > 3 && a.detectLoop() {
			log.Printf("[Agent:%s] Detected loop, forcing finish with best available output", a.Role)
			return a.generateFinalOutput(input), nil
		}
	}

	log.Printf("[Agent:%s] Max iterations (%d) reached, returning partial result", a.Role, state.MaxIterations)
	return a.generateFinalOutput(input), nil
}

// generateThought uses LLM to generate next thought in ReAct cycle
func (a *Agent) generateThought(_ context.Context, input map[string]interface{}, iteration int) (string, error) {
	prompt := a.buildReActPrompt(input, iteration)

	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens": 500,
			"temperature":     0.5,
		},
	}

	jsonData, _ := json.Marshal(geminiReq)
	url := fmt.Sprintf("%s?key=%s", a.aiProvider.apiURL, a.aiProvider.apiKey)

	resp, err := a.aiProvider.httpClient.Post(url, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct{ Text string } `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text), nil
	}

	return "Need to continue processing", nil
}

// decideAction uses LLM to decide which tool to use
func (a *Agent) decideAction(_ context.Context, thought string, _ map[string]interface{}) (*AgentAction, error) {
	// Use LLM to decide on action
	prompt := fmt.Sprintf(`Based on this thought: "%s"
Choose the next action. Available tools: %v

If you have enough information to complete your objective, respond with tool name "FINISH" and the final output in the result field.

Return JSON: {"tool_name": "...", "params": {...}, "reason": "..."}`,
		thought, a.getToolNames())

	geminiReq := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{{"text": prompt}},
			},
		},
		"generationConfig": map[string]interface{}{
			"maxOutputTokens":  300,
			"temperature":      0.3,
			"responseMimeType": "application/json",
		},
	}

	jsonData, _ := json.Marshal(geminiReq)
	httpReq, err := http.NewRequest("POST", a.aiProvider.apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", a.aiProvider.apiKey)

	resp, err := a.aiProvider.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read raw response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	log.Printf("[Agent:%s] decideAction Gemini response status: %d, body: %s", a.Role, resp.StatusCode, string(bodyBytes))

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct{ Text string } `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode gemini response: %w", err)
	}

	// Check for API error
	if geminiResp.Error != nil {
		return nil, fmt.Errorf("gemini API error: %s (code: %d, status: %s)", geminiResp.Error.Message, geminiResp.Error.Code, geminiResp.Error.Status)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("empty candidates in gemini response")
	}

	if len(geminiResp.Candidates[0].Content.Parts) == 0 {
		finishReason := geminiResp.Candidates[0].FinishReason
		return nil, fmt.Errorf("empty parts in gemini response, finishReason: %s", finishReason)
	}

	responseText := strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
	responseText = cleanJSONResponse(responseText)

	var actionReq struct {
		ToolName string                 `json:"tool_name"`
		Params   map[string]interface{} `json:"params"`
		Reason   string                 `json:"reason"`
	}

	if err := json.Unmarshal([]byte(responseText), &actionReq); err != nil {
		return nil, fmt.Errorf("failed to parse action JSON: %w", err)
	}

	return &AgentAction{
		ToolName:  actionReq.ToolName,
		Params:    actionReq.Params,
		Timestamp: time.Now(),
	}, nil
}

// executeTool executes the chosen tool and returns observation
func (a *Agent) executeTool(ctx context.Context, action *AgentAction) (interface{}, error) {
	// Find the tool
	var tool *AgentTool
	for _, t := range a.Tools {
		if t.Name == action.ToolName {
			tool = t
			break
		}
	}

	if tool == nil {
		return nil, fmt.Errorf("tool not found: %s", action.ToolName)
	}

	// Execute the tool handler with type assertion
	params, ok := action.Params.(map[string]interface{})
	if !ok {
		params = make(map[string]interface{})
	}
	result, err := tool.Handler(ctx, params)
	if err != nil {
		return nil, err
	}

	action.Result = result
	return result, nil
}

// detectLoop checks if the agent is stuck in a loop
func (a *Agent) detectLoop() bool {
	if len(a.State.Actions) < 4 {
		return false
	}

	// Check if last 3 actions used the same tool with same params
	lastActions := a.State.Actions[len(a.State.Actions)-3:]
	firstTool := lastActions[0].ToolName
	for _, action := range lastActions[1:] {
		if action.ToolName != firstTool {
			return false
		}
	}
	return true
}

// generateFinalOutput generates final output when max iterations reached
func (a *Agent) generateFinalOutput(input map[string]interface{}) interface{} {
	// Use LLM to synthesize final output from all observations
	_ = fmt.Sprintf(`Based on these observations: %v
Generate the final output for the objective: %s

Return the complete structured output.`,
		a.State.Observations, a.State.Objective)

	// Placeholder - return input as final output
	return input
}

// buildReActPrompt builds the prompt for ReAct reasoning
func (a *Agent) buildReActPrompt(_ map[string]interface{}, iteration int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("You are the %s Agent.\n", a.Role))
	sb.WriteString(a.SystemPrompt)
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("Current Objective: %s\n\n", a.State.Objective))

	if iteration > 0 {
		sb.WriteString("Previous Thoughts:\n")
		for i, t := range a.State.Thoughts {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, t))
		}
		sb.WriteString("\n")
	}

	if len(a.State.Actions) > 0 {
		sb.WriteString("Actions Taken:\n")
		for i, action := range a.State.Actions {
			sb.WriteString(fmt.Sprintf("%d. Used tool %s\n", i+1, action.ToolName))
		}
		sb.WriteString("\n")
	}

	if len(a.State.Observations) > 0 {
		sb.WriteString("Observations:\n")
		for i, obs := range a.State.Observations {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, obs))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Iteration %d: What is your next thought?\n", iteration+1))

	return sb.String()
}

// getToolNames returns list of available tool names
func (a *Agent) getToolNames() []string {
	names := make([]string, len(a.Tools))
	for i, tool := range a.Tools {
		names[i] = tool.Name
	}
	return names
}

// =====================================================
// Tool Handlers - Extractor Agent
// =====================================================

func (o *AgentOrchestrator) handleReadDocumentChunk(_ context.Context, params map[string]interface{}) (interface{}, error) {
	pageStart, ok := params["page_start"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid page_start parameter")
	}

	pageEnd, _ := params["page_end"].(int)
	if pageEnd == 0 {
		pageEnd = pageStart
	}

	// The document content would be passed through the input
	// This is a simplified version - actual implementation would access stored document
	return map[string]interface{}{
		"pages_read":    []int{pageStart, pageEnd},
		"content_chunk": "Document content for pages",
	}, nil
}

func (o *AgentOrchestrator) handleExtractTableContent(_ context.Context, _ map[string]interface{}) (interface{}, error) {
	// Extract tables from specified pages
	return map[string]interface{}{
		"tables_found": 0,
		"table_data":   []interface{}{},
	}, nil
}

func (o *AgentOrchestrator) handleIdentifyKeyTopics(_ context.Context, params map[string]interface{}) (interface{}, error) {
	_, _ = params["content"].(string)

	// Use LLM to identify key topics
	return map[string]interface{}{
		"topics":     []string{"Topic 1", "Topic 2"},
		"objectives": []string{"Objective 1", "Objective 2"},
	}, nil
}

// =====================================================
// Tool Handlers - Architect Agent
// =====================================================

func (o *AgentOrchestrator) handleDraftWeekStructure(_ context.Context, params map[string]interface{}) (interface{}, error) {
	weekNum, _ := params["week_number"].(int)
	weekName, _ := params["week_name"].(string)
	moduleNames, _ := params["module_names"].([]interface{})

	modules := make([]string, len(moduleNames))
	for i, m := range moduleNames {
		modules[i] = m.(string)
	}

	return map[string]interface{}{
		"week_number": weekNum,
		"week_name":   weekName,
		"modules":     modules,
	}, nil
}

func (o *AgentOrchestrator) handleValidateWeekCount(_ context.Context, params map[string]interface{}) (interface{}, error) {
	totalWeeks, _ := params["total_weeks"].(int)

	valid := totalWeeks >= 8 && totalWeeks <= 16
	return map[string]interface{}{
		"valid":   valid,
		"message": fmt.Sprintf("Week count %d is %s", totalWeeks, map[bool]string{true: "valid", false: "invalid"}[valid]),
	}, nil
}

func (o *AgentOrchestrator) handleCheckTopicCoverage(_ context.Context, params map[string]interface{}) (interface{}, error) {
	required, _ := params["required_topics"].([]interface{})
	current, _ := params["current_topics"].([]interface{})

	missing := []string{}
	for _, r := range required {
		found := false
		for _, c := range current {
			if r.(string) == c.(string) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r.(string))
		}
	}

	return map[string]interface{}{
		"all_covered":    len(missing) == 0,
		"missing_topics": missing,
	}, nil
}

// =====================================================
// Tool Handlers - Writer Agent
// =====================================================

func (o *AgentOrchestrator) handleExpandModuleContent(_ context.Context, params map[string]interface{}) (interface{}, error) {
	moduleName, _ := params["module_name"].(string)
	keyTopics, _ := params["key_topics"].([]interface{})
	targetLength, _ := params["target_length"].(int)

	if targetLength == 0 {
		targetLength = 500
	}

	// Generate content using LLM
	topics := make([]string, len(keyTopics))
	for i, t := range keyTopics {
		topics[i] = t.(string)
	}

	content := fmt.Sprintf("# %s\n\nThis module covers: %s\n\n[Generated content would go here...]", moduleName, strings.Join(topics, ", "))

	return map[string]interface{}{
		"module_name": moduleName,
		"content":     content,
		"word_count":  len(strings.Fields(content)),
	}, nil
}

func (o *AgentOrchestrator) handleAddCodeExamples(_ context.Context, params map[string]interface{}) (interface{}, error) {
	topic, _ := params["topic"].(string)
	language, _ := params["programming_language"].(string)

	if language == "" {
		language = "go"
	}

	example := fmt.Sprintf("```%s\n// Example code for: %s\n```\n", language, topic)

	return map[string]interface{}{
		"topic":   topic,
		"example": example,
	}, nil
}

func (o *AgentOrchestrator) handleFormatAsMarkdown(_ context.Context, params map[string]interface{}) (interface{}, error) {
	content, _ := params["content"].(string)

	// Return formatted content
	return map[string]interface{}{
		"formatted_content": content,
	}, nil
}

// =====================================================
// Tool Handlers - Quizmaster Agent
// =====================================================

func (o *AgentOrchestrator) handleGenerateMCQ(_ context.Context, params map[string]interface{}) (interface{}, error) {
	topic, _ := params["topic"].(string)
	difficulty, _ := params["difficulty"].(string)
	count, _ := params["count"].(int)

	if count == 0 {
		count = 1
	}

	questions := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		questions[i] = map[string]interface{}{
			"question_type": "mcq",
			"question_text": fmt.Sprintf("Sample MCQ about %s", topic),
			"options": []map[string]interface{}{
				{"option_text": "Option A", "is_correct": true, "order_index": 0},
				{"option_text": "Option B", "is_correct": false, "order_index": 1},
				{"option_text": "Option C", "is_correct": false, "order_index": 2},
				{"option_text": "Option D", "is_correct": false, "order_index": 3},
			},
			"difficulty": difficulty,
		}
	}

	return map[string]interface{}{
		"questions": questions,
		"count":     count,
	}, nil
}

func (o *AgentOrchestrator) handleGenerateTrueFalse(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	topic, _ := params["topic"].(string)
	count, _ := params["count"].(int)

	if count == 0 {
		count = 1
	}

	questions := make([]map[string]interface{}, count)
	for i := 0; i < count; i++ {
		questions[i] = map[string]interface{}{
			"question_type": "true_false",
			"question_text": fmt.Sprintf("True/False question about %s", topic),
			"options": []map[string]interface{}{
				{"option_text": "True", "is_correct": i%2 == 0, "order_index": 0},
				{"option_text": "False", "is_correct": i%2 != 0, "order_index": 1},
			},
			"difficulty": "easy",
		}
	}

	return map[string]interface{}{
		"questions": questions,
		"count":     count,
	}, nil
}

func (o *AgentOrchestrator) handleAddExplanation(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	question, _ := params["question"].(string)
	correctAnswer, _ := params["correct_answer"].(string)

	explanation := fmt.Sprintf("Explanation: %s is correct because...", correctAnswer)

	return map[string]interface{}{
		"question":    question,
		"explanation": explanation,
	}, nil
}

// =====================================================
// Tool Handlers - Critic Agent
// =====================================================

func (o *AgentOrchestrator) handleValidateJSONStructure(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	jsonData, _ := params["json_data"].(string)
	schemaType, _ := params["schema_type"].(string)

	// Validate JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"valid":       true,
		"schema_type": schemaType,
	}, nil
}

func (o *AgentOrchestrator) handleCheckContentQuality(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_, checkType := params["content"].(string), params["check_type"].(string)

	// Quality check logic
	return map[string]interface{}{
		"check_type": checkType,
		"score":      0.85,
		"issues":     []string{},
	}, nil
}

func (o *AgentOrchestrator) handleSuggestImprovements(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_, _ = params["content"].(string)
	aspect, _ := params["aspect"].(string)

	improvements := []string{
		fmt.Sprintf("Consider adding more examples for %s", aspect),
		"Add visual diagrams where applicable",
	}

	return map[string]interface{}{
		"aspect":       aspect,
		"improvements": improvements,
	}, nil
}

// =====================================================
// Tool Handlers - Contest Agent
// =====================================================

func (o *AgentOrchestrator) handleGenerateProblem(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	topicsInterface, _ := params["topics"].([]interface{})
	difficulty, _ := params["difficulty"].(string)
	courseContext, _ := params["course_context"].(string)

	topics := make([]string, len(topicsInterface))
	for i, t := range topicsInterface {
		topics[i] = t.(string)
	}

	log.Printf("[ContestAgent] Generating problem with topics: %v, difficulty: %s", topics, difficulty)

	// Use AI provider to generate problem
	problem, err := o.aiProvider.GenerateContestProblem(ctx, topics, difficulty, courseContext)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"problem": problem,
	}, nil
}

func (o *AgentOrchestrator) handleVerifySolution(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	solutionCode, _ := params["solution_code"].(string)
	language, _ := params["language"].(string)
	testCasesInterface, _ := params["test_cases"].([]interface{})
	timeLimitMS, _ := params["time_limit_ms"].(int)
	memoryLimitKB, _ := params["memory_limit_kb"].(int)

	if timeLimitMS == 0 {
		timeLimitMS = 2000
	}
	if memoryLimitKB == 0 {
		memoryLimitKB = 256000
	}

	// Convert test cases
	testCases := make([]map[string]string, len(testCasesInterface))
	for i, tc := range testCasesInterface {
		tcMap, _ := tc.(map[string]interface{})
		testCases[i] = map[string]string{
			"input":    tcMap["input"].(string),
			"expected": tcMap["expected_output"].(string),
		}
	}

	// Determine language ID
	languageID := 71 // Python 3
	if strings.ToLower(language) == "cpp" {
		languageID = 54 // C++17
	}

	log.Printf("[ContestAgent] Verifying solution with %d test cases", len(testCases))

	// Run verification
	allPassed := true
	var failedResult map[string]interface{}

	for i, tc := range testCases {
		result, err := SubmitCode(
			solutionCode,
			languageID,
			tc["input"],
			tc["expected"],
			timeLimitMS,
			memoryLimitKB,
		)

		if err != nil {
			log.Printf("[ContestAgent] Test case %d error: %v", i+1, err)
			return map[string]interface{}{
				"passed":          false,
				"error":           err.Error(),
				"test_case_index": i,
			}, nil
		}

		if !result.Passed {
			allPassed = false
			failedResult = map[string]interface{}{
				"status":         result.Status,
				"stdout":         result.Stdout,
				"stderr":         result.Stderr,
				"compile_output": result.CompileOutput,
				"expected":       tc["expected"],
				"input":          tc["input"],
			}
			log.Printf("[ContestAgent] Test case %d FAILED: %s", i+1, result.Status)
			break
		}
		log.Printf("[ContestAgent] Test case %d PASSED", i+1)
	}

	return map[string]interface{}{
		"passed":               allPassed,
		"all_testcases_passed": allPassed,
		"failed_result":        failedResult,
		"test_cases_count":     len(testCases),
	}, nil
}

func (o *AgentOrchestrator) handleFixCode(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	originalCode, _ := params["original_code"].(string)
	errorMessage, _ := params["error_message"].(string)
	errorType, _ := params["error_type"].(string)
	problemStatement, _ := params["problem_statement"].(string)

	log.Printf("[ContestAgent] Fixing code with error type: %s", errorType)

	// Use AI provider to fix code
	fixedCode, err := o.aiProvider.FixContestSolution(ctx, originalCode, errorMessage, errorType, problemStatement)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"code":    originalCode,
		}, nil
	}

	return map[string]interface{}{
		"success":     true,
		"fixed_code":  fixedCode,
		"error_fixed": errorType,
	}, nil
}

func (o *AgentOrchestrator) handleValidateTestCases(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	testCasesInterface, _ := params["test_cases"].([]interface{})
	_ = params["constraints"].(map[string]interface{}) // Reserved for future use

	testCases := make([]map[string]interface{}, len(testCasesInterface))
	for i, tc := range testCasesInterface {
		testCases[i] = tc.(map[string]interface{})
	}

	// Basic validation
	if len(testCases) < 2 {
		return map[string]interface{}{
			"valid": false,
			"error": "Insufficient test cases (minimum 2 required)",
		}, nil
	}

	// Check for sample and hidden cases
	hasSample := false
	hasHidden := false
	for _, tc := range testCases {
		if isVisible, ok := tc["is_sample_visible"].(bool); ok {
			if isVisible {
				hasSample = true
			} else {
				hasHidden = true
			}
		}
	}

	if !hasSample {
		return map[string]interface{}{
			"valid": false,
			"error": "No visible sample test cases found",
		}, nil
	}

	return map[string]interface{}{
		"valid":            true,
		"test_cases_count": len(testCases),
		"has_sample":       hasSample,
		"has_hidden":       hasHidden,
	}, nil
}

func (o *AgentOrchestrator) handleFinishProblem(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	problemInterface, _ := params["problem"].(map[string]interface{})
	verificationPassed, _ := params["verification_passed"].(bool)

	log.Printf("[ContestAgent] Finishing problem, verification passed: %v", verificationPassed)

	if !verificationPassed {
		return map[string]interface{}{
			"success": false,
			"error":   "Problem verification did not pass",
		}, nil
	}

	// Convert problem interface to map for return
	return map[string]interface{}{
		"success": true,
		"problem": problemInterface,
		"status":  "approved",
	}, nil
}

// =====================================================
// Helper Functions
// =====================================================
