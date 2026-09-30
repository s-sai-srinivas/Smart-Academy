package main

import (
	"coding-platform/config"
	"coding-platform/database"
	"coding-platform/handlers"
	"coding-platform/middleware"
	"coding-platform/services"
	"log"

	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load configuration
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed super admin from environment variables (idempotent)
	if err := handlers.SeedSuperAdmin(); err != nil {
		log.Printf("Warning: Failed to seed super admin: %v", err)
	}

	// Initialize cache service (Redis or in-memory fallback)
	if err := services.InitCache(); err != nil {
		log.Printf("Warning: Cache initialization error: %v. Using in-memory fallback", err)
	}

	// Initialize job queue for code execution (requires Redis)
	if config.AppConfig.JobQueueEnabled && config.AppConfig.RedisEnabled {
		if err := services.InitJobQueue(); err != nil {
			log.Printf("Warning: Job queue initialization error: %v. Falling back to direct execution", err)
		} else {
			services.StartWorkers(config.AppConfig.JobQueueWorkers)
			log.Printf("Job queue started with %d workers", config.AppConfig.JobQueueWorkers)
		}
	}

	// Set Gin mode
	gin.SetMode(config.AppConfig.GinMode)

	// Register custom validators for input validation
	middleware.RegisterCustomValidators()

	// Initialize Prometheus metrics
	middleware.InitMetrics()

	// Create router
	router := gin.New()

	// Add recovery middleware (handles panics)
	router.Use(middleware.RecoveryMiddleware())

	// Add request logger middleware
	router.Use(middleware.RequestLoggerMiddleware())

	// Add Prometheus metrics middleware
	router.Use(middleware.PrometheusMiddleware())

	// Add request body size limit middleware (prevents DoS via large payloads)
	router.Use(middleware.DefaultRequestBodyLimitMiddleware())

	// CORS middleware
	// Build a map for O(1) origin lookup
	allowedOrigins := make(map[string]bool)
	for _, origin := range config.AppConfig.CORSAllowedOrigins {
		allowedOrigins[origin] = true
	}

	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// Check if origin is in allowed list
			if allowedOrigins[origin] {
				return true
			}
			// In development, allow any localhost origin
			if config.AppConfig.GinMode == "debug" && strings.HasPrefix(origin, "http://localhost") {
				return true
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API routes
	api := router.Group("/api")
	// Apply global rate limiting middleware (IP-based enforcement currently bypassed)
	api.Use(middleware.GlobalRateLimitMiddleware())
	{
		// Public routes
		auth := api.Group("/auth")
		{
			// Login uses rate limiting middleware (IP-based enforcement currently bypassed)
			auth.POST("/login", middleware.LoginRateLimitMiddleware(), handlers.Login)
			// Note: /seed-faculty moved to protected admin routes below (Fix 9)
		}

		// Public college list for login dropdown
		api.GET("/colleges", handlers.GetColleges)

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// Reference data routes (require auth for college-scoped filtering)
			protected.GET("/semesters", handlers.GetSemesters)
			protected.GET("/branches", handlers.GetBranches)
			protected.GET("/sections", handlers.GetSections)
			protected.GET("/batches", handlers.GetBatches)
			protected.GET("/programs", handlers.GetPrograms)
			protected.GET("/regulations", handlers.GetRegulations)
			protected.GET("/topics", handlers.GetTopics)
			protected.GET("/subjects", handlers.GetSubjects)

			// User info and auth management
			protected.GET("/me", handlers.GetMe)
			protected.POST("/auth/logout", handlers.Logout)                                                                 // Fix 4: token revocation
			protected.POST("/auth/change-password", middleware.PasswordResetRateLimitMiddleware(), handlers.ChangePassword) // Fix 5: self-service password change

			// WebSocket connection for real-time events (force logout, notifications)
			protected.GET("/ws", handlers.WebSocketHandler)

			// Problem routes (require authentication)
			protected.GET("/problems", handlers.GetProblems)
			protected.GET("/problems/:id", handlers.GetProblem)

			// Submission routes (require authentication)
			protected.GET("/submissions", handlers.GetSubmissions)
			protected.GET("/submissions/:id", handlers.GetSubmission)
			protected.GET("/submissions/stats", handlers.GetSubmissionStats)

			// Job queue routes (for polling execution status)
			protected.GET("/jobs/:id", handlers.GetJobStatus)

			// Course listing (requires auth for semester-based filtering)
			protected.GET("/courses", handlers.GetCourses)
			protected.GET("/courses/:id", handlers.GetCourse)
			protected.GET("/courses/:id/topics", handlers.GetLabTopics)
			protected.GET("/topics/:id/problems", handlers.GetTopicProblems)
			protected.GET("/topics-overview", handlers.GetTopicsOverview)

			// User's own submissions and completed problems
			protected.GET("/my/submissions", handlers.GetUserSubmissions)
			protected.GET("/my/completed-problems", handlers.GetUserCompletedProblems)
			protected.GET("/my/problems/:id/completed", handlers.CheckProblemCompletion)
			protected.GET("/submissions/:id/code", handlers.GetSubmissionCode) // Lazy load submission code

			// Solve session tracking for accurate time tracking
			protected.POST("/problems/:id/start", handlers.StartSolveSession)
			protected.GET("/problems/:id/session", handlers.GetActiveSolveSession)
			protected.POST("/sessions/:sessionId/end", handlers.EndSolveSession)

			protected.GET("/dashboard", handlers.GetDashboard)
			protected.GET("/profile", handlers.GetProfile)
			protected.POST("/submit", middleware.SubmitRateLimitMiddleware(), handlers.SubmitCode)
			protected.POST("/run", middleware.RunRateLimitMiddleware(), handlers.RunCode) // Fixed: now requires authentication
			protected.GET("/problems/:id/submissions", handlers.GetProblemSubmissions)

			// Student quiz routes
			protected.GET("/quizzes", handlers.ListAvailableQuizzes)
			protected.GET("/quizzes/:id", handlers.GetQuizForStudent)
			protected.POST("/quizzes/:id/start", handlers.StartAttempt)
			protected.POST("/quizzes/:id/submit", handlers.SubmitAttempt)
			protected.GET("/quizzes/:id/result", handlers.GetQuizResult)

			// Theory practice quiz - read-only, accessible to all authenticated users (incl. students)
			protected.GET("/theory/weeks/:id/practice-quiz", handlers.GetPracticeQuizForWeek)

			// Theory PDF download
			protected.GET("/theory-pdfs/:id", handlers.ServeTheoryPDF)
			// Theory PDF list — accessible to all authenticated users (students need to view PDFs)
			protected.GET("/theories/:id/pdfs", handlers.GetTheoryPDFs)

			// Contest routes
			protected.GET("/contests", handlers.GetContests)
			protected.GET("/contests/:id", handlers.GetContestDetails)
			protected.POST("/contests/:id/join", handlers.JoinContest)
			protected.POST("/contests/:id/finish", handlers.FinishContest)
			protected.POST("/contests/:id/violations/esc", handlers.ReportEscViolation)
			protected.GET("/contests/:id/leaderboard", handlers.GetContestLeaderboard)
			protected.GET("/contests/:id/problems/:problemId", handlers.GetContestProblem)
			protected.POST("/contests/:id/problems/:problemId/submit", middleware.ContestRateLimitMiddleware(), handlers.SubmitContestSolution)
			protected.GET("/contests/:id/problems/:problemId/submissions", handlers.GetContestProblemSubmissions)
			protected.GET("/contest-submissions/:submissionId/code", handlers.GetContestSubmissionCode)

			// Contest Quiz (student-facing)
			protected.GET("/contests/:id/quiz", handlers.GetContestQuizForStudent)
			protected.POST("/contests/:id/quiz/start", handlers.StartContestQuiz)
			protected.POST("/contests/:id/quiz/submit", handlers.SubmitContestQuiz)
			protected.GET("/contests/:id/quiz/result", handlers.GetContestQuizResult)

			// Admin-only routes
			admin := protected.Group("/admin")
			admin.Use(middleware.AdminOnly())
			{
				// Dashboard stats
				admin.GET("/stats", handlers.GetAdminStats)

				// Job queue stats (monitoring)
				admin.GET("/job-stats", handlers.GetJobQueueStats)

				// Curriculum management
				admin.POST("/curriculums", handlers.CreateCurriculum)
				admin.GET("/curriculums", handlers.GetCurriculums)
				admin.POST("/curriculums/:id/courses", handlers.AddCourseToCurriculum)
				admin.GET("/curriculums/:id/courses", handlers.GetCurriculumCourses)
				admin.DELETE("/curriculums/:id/courses/:courseId", handlers.RemoveCourseFromCurriculum)

				// Course management
				admin.POST("/courses", handlers.CreateCourse)
				admin.GET("/courses", handlers.GetAdminCourses)
				admin.PUT("/courses/:id", handlers.UpdateCourse)
				admin.DELETE("/courses/:id", handlers.DeleteCourse)
				admin.POST("/courses/:id/branches", handlers.AssignCourseBranches)
				admin.GET("/courses/:id/branches", handlers.GetCourseBranches)

				// Lab management
				admin.POST("/labs", handlers.CreateLab)
				admin.GET("/courses/:id/lab", handlers.GetCourseLab)
				admin.GET("/labs/:id/sessions", handlers.GetCourseLabSessions)

				// Theory management
				admin.POST("/theories", handlers.CreateTheory)
				admin.GET("/courses/:id/theory", handlers.GetCourseTheory)

				// Structure Management (Sessions, Weeks, Modules)
				admin.POST("/labs/sessions", handlers.CreateLabSession)
				admin.POST("/sessions/problems", handlers.AddProblemToSession)
				admin.POST("/theories/weeks", handlers.CreateTheoryWeek)
				admin.POST("/weeks/modules", handlers.CreateTheoryModule)
				admin.PUT("/weeks/modules/:id", handlers.UpdateTheoryModule)
				admin.DELETE("/theories/weeks/:id", handlers.DeleteTheoryWeek)
				admin.DELETE("/weeks/modules/:id", handlers.DeleteTheoryModule)

				// Theory PDF Management
				admin.POST("/theories/pdf/upload", handlers.UploadTheoryPDF)
				admin.GET("/theories/:id/pdfs", handlers.GetTheoryPDFs)
				admin.DELETE("/theories/pdfs/:id", handlers.DeleteTheoryPDF)

				// Lesson Plan Processing (AI-powered course content generation)
				admin.POST("/lesson-plans/upload", handlers.UploadLessonPlan)
				admin.GET("/lesson-plans/jobs/:id", handlers.GetGenerationJobStatus)
				admin.GET("/lesson-plans/jobs/:id/preview", handlers.GetGenerationPreview)
				admin.POST("/lesson-plans/jobs/:id/approve", handlers.ApproveAndSaveGeneration)
				admin.GET("/theory/weeks/:id/practice-quiz", handlers.GetPracticeQuizForWeek)
				admin.PUT("/theory/weeks/:id/practice-quiz", handlers.UpdatePracticeQuiz)
				admin.POST("/theory/weeks/:id/practice-quiz/regenerate", handlers.RegeneratePracticeQuiz)

				// Contest Generation (AI-powered contest problem generation)
				admin.POST("/contests/generate", handlers.GenerateContestProblems)
				admin.GET("/contests/generation/:id", handlers.GetContestGenerationJobStatus)
				admin.GET("/contests/generation/:id/problems", handlers.GetGeneratedProblems)
				admin.GET("/contests/generation/:id/problems/:index", handlers.GetGeneratedProblemDetail)
				admin.POST("/contests/generation/:id/approve", handlers.ApproveGeneratedProblems)
				admin.POST("/contests/generation/:id/regenerate", handlers.RegenerateProblem)
				admin.POST("/contests/editorial/generate", handlers.GenerateEditorial)
				admin.GET("/contests/:id/problems/:problem_id/editorial", handlers.GetEditorial)

				// Problem management
				admin.POST("/problems", handlers.CreateAdminProblem)
				admin.GET("/problems", handlers.GetAdminProblems)
				admin.POST("/problems/:id/testcases", handlers.CreateTestCase)
				admin.GET("/problems/:id/testcases", handlers.GetTestCases)
				admin.DELETE("/problems/:id/testcases/:testcase_id", handlers.DeleteTestCase)

				// Plagiarism detection routes
				admin.GET("/plagiarism/submissions/:id", handlers.CheckSubmissionPlagiarism)
				admin.GET("/plagiarism/problems/:id", handlers.CheckProblemPlagiarism)
				admin.GET("/plagiarism/results/:problem_id", handlers.GetPlagiarismResults)

				// Student onboarding routes
				admin.GET("/students/template", handlers.DownloadStudentTemplate)
				admin.POST("/students/bulk-upload", handlers.BulkUploadStudents)

				// Faculty onboarding routes
				admin.GET("/faculty/template", handlers.DownloadFacultyTemplate)
				admin.POST("/faculty/bulk-upload", handlers.BulkUploadFaculty)

				// Principal management
				admin.POST("/principal/create", handlers.CreatePrincipal)

				// Faculty seeding (Fix 9: moved from public route - requires admin auth)
				admin.POST("/seed-faculty", handlers.SeedFacultyUser)

				// Password reset for users who forgot their password (Fix 5)
				admin.POST("/users/:regdno/reset-password", middleware.PasswordResetRateLimitMiddleware(), handlers.AdminResetPassword)

				// Contest management (admin/college_admin)
				admin.POST("/contests", handlers.CreateContest)
				admin.PUT("/contests/:id", handlers.UpdateContest)
				admin.DELETE("/contests/:id", handlers.DeleteContest)
				admin.POST("/contests/:id/problems", handlers.AddProblemToContest)
				admin.DELETE("/contests/:id/problems/:problemId", handlers.RemoveContestProblem)
				admin.POST("/contests/:id/practice", handlers.EnablePracticeMode) // Practice mode control

				// Contest Quiz management (admin)
				admin.POST("/contests/:id/quiz", handlers.UpsertContestQuiz)
				admin.GET("/contests/:id/quiz", handlers.GetContestQuiz)
				admin.DELETE("/contests/:id/quiz", handlers.DeleteContestQuiz)

				// Contest Section management (admin)
				admin.PUT("/contests/:id/sections", handlers.UpsertContestSections)
				admin.GET("/contests/:id/sections", handlers.GetContestSections)
			}

			// Super admin-only routes (college lifecycle management)
			superAdmin := protected.Group("/super-admin")
			superAdmin.Use(middleware.SuperAdminOnly())
			{
				superAdmin.POST("/colleges", handlers.CreateCollege)
				superAdmin.GET("/colleges", handlers.ListAllColleges)
				superAdmin.GET("/colleges/:id", handlers.GetCollegeByID)
				superAdmin.PUT("/colleges/:id/status", handlers.UpdateCollegeStatus)

				// Global practice problems management
				superAdmin.POST("/problems", handlers.CreateGlobalProblem)
				superAdmin.GET("/problems", handlers.GetGlobalProblems)
				superAdmin.PUT("/problems/:id", handlers.UpdateGlobalProblem)
				superAdmin.DELETE("/problems/:id", handlers.DeleteGlobalProblem)
				superAdmin.POST("/problems/:id/testcases", handlers.CreateGlobalProblemTestCase)

				// Subject & Topic management
				superAdmin.POST("/subjects", handlers.CreateSubject)
				superAdmin.GET("/subjects", handlers.GetSubjects)
				superAdmin.PUT("/subjects/:id", handlers.UpdateSubject)
				superAdmin.DELETE("/subjects/:id", handlers.DeleteSubject)
				superAdmin.POST("/topics", handlers.CreateTopic)
				superAdmin.PUT("/topics/:id", handlers.UpdateTopic)
				superAdmin.DELETE("/topics/:id", handlers.DeleteTopic)
			}

			// Practice routes - accessible to all authenticated users (students)
			practice := protected.Group("/practice")
			{
				practice.GET("/problems", handlers.GetPracticeProblems)
				practice.GET("/problems/:id", handlers.GetPracticeProblem)
				practice.POST("/submit", middleware.SubmitRateLimitMiddleware(), handlers.SubmitPracticeSolution)
				practice.POST("/run", middleware.RunRateLimitMiddleware(), handlers.RunPracticeCode)
				practice.GET("/problems/:id/submissions", handlers.GetPracticeProblemSubmissions)
			}

			// Faculty-only routes
			faculty := protected.Group("")
			faculty.Use(middleware.FacultyOnly())
			{
				faculty.GET("/faculty/courses/:id/analytics", handlers.GetCourseAnalytics)
				faculty.GET("/faculty/courses/:id/sections/:sectionId/analytics", handlers.GetSectionAnalytics)
				faculty.GET("/faculty/students/:id/analytics", handlers.GetStudentAnalytics)

				// Quiz routes for faculty
				faculty.GET("/faculty/my-offerings", handlers.GetMyOfferings)
				faculty.GET("/faculty/offerings/:id/theory-modules", handlers.GetTheoryModulesForOffering)
				faculty.POST("/faculty/quizzes/generate", handlers.GenerateAIQuestions)
				faculty.POST("/faculty/quizzes", handlers.CreateQuiz)
				faculty.GET("/faculty/quizzes", handlers.ListMyQuizzes)
				faculty.GET("/faculty/quizzes/:id", handlers.GetQuiz)
				faculty.PUT("/faculty/quizzes/:id", handlers.UpdateQuiz)
				faculty.DELETE("/faculty/quizzes/:id", handlers.DeleteQuiz)
				faculty.POST("/faculty/quizzes/:id/publish", handlers.PublishQuiz)
				faculty.POST("/faculty/quizzes/:id/close", handlers.CloseQuiz)
				faculty.POST("/faculty/quizzes/:id/questions", handlers.SaveQuestions)
				faculty.PUT("/faculty/quizzes/:id/questions/:qid", handlers.EditQuestion)
				faculty.DELETE("/faculty/quizzes/:id/questions/:qid", handlers.DeleteQuestion)
				faculty.POST("/faculty/quizzes/:id/regenerate", handlers.RegenerateQuestions)
				faculty.GET("/faculty/quizzes/:id/analytics", handlers.GetQuizAnalytics)
				faculty.GET("/faculty/quizzes/:id/attempts", handlers.ListStudentAttempts)

				// Faculty contest routes (read-only with CSV export and plagiarism)
				faculty.GET("/faculty/contests/:id/results/csv", handlers.GetContestResultsCSV)
				faculty.GET("/faculty/contests/:id/results/detailed-csv", handlers.GetContestDetailedCSV)
				faculty.GET("/faculty/contests/:id/plagiarism", handlers.GetContestPlagiarism)
				faculty.POST("/faculty/contests/:id/plagiarism/check", handlers.RunContestPlagiarismCheck)
				faculty.GET("/faculty/contests/:id/non-participants", handlers.GetContestNonParticipants)
			}

			// HOD-only routes
			hod := protected.Group("/hod")
			hod.Use(middleware.HODOnly())
			{
				// Dashboard and Analytics
				hod.GET("/stats", handlers.GetHODDashboardStats)
				hod.GET("/analytics", handlers.GetBranchAnalytics)

				// Course Analytics (HOD can view all sections)
				hod.GET("/analytics/course/:id", handlers.GetHODCourseAnalytics)
				hod.GET("/analytics/course/:id/section/:sectionId", handlers.GetHODSectionAnalytics)
				// Use faculty's student analytics (already available to authenticated users)
				hod.GET("/analytics/student/:id", handlers.GetStudentAnalytics)

				// Faculty Management
				hod.GET("/faculty", handlers.GetHODFaculty)
				hod.PUT("/faculty/role", handlers.UpdateFacultyRole)
				hod.POST("/faculty/assign", handlers.AssignFacultyToCourse)
				hod.POST("/faculty/assign-all", handlers.AssignFacultyToAllSections)
				hod.GET("/faculty/assignments", handlers.GetCourseAssignments)

				// Courses (read-only for HOD)
				hod.GET("/courses", handlers.GetHODCourses)
				hod.GET("/courses/:courseId/sections", handlers.GetSectionsForCourse)
				hod.GET("/course-offerings", handlers.GetHODCourseOfferings)

				// Contest routes for HOD (read-only with CSV export and plagiarism)
				hod.GET("/contests", handlers.GetHODContests)
				hod.GET("/contests/:id", handlers.GetHODContestDetails)
				hod.GET("/contests/:id/results/csv", handlers.GetContestResultsCSV)
				hod.GET("/contests/:id/results/detailed-csv", handlers.GetContestDetailedCSV)
				hod.GET("/contests/:id/plagiarism", handlers.GetContestPlagiarism)
				hod.POST("/contests/:id/plagiarism/check", handlers.RunContestPlagiarismCheck)
				hod.GET("/contests/:id/non-participants", handlers.GetContestNonParticipants)
			}

			// Principal-only routes
			principal := protected.Group("/principal")
			principal.Use(middleware.PrincipalOnly())
			{
				principal.GET("/stats", handlers.GetPrincipalStats)
				principal.GET("/departments", handlers.GetDepartmentsWithHOD)
				principal.POST("/assign-hod", handlers.AssignHOD)
				principal.POST("/remove-hod", handlers.RemoveHOD)

				// Principal Analytics
				principal.GET("/analytics/overview", handlers.GetPrincipalAnalyticsOverview)
				principal.GET("/analytics/courses", handlers.GetPrincipalCourseAnalytics)
				principal.GET("/analytics/labs", handlers.GetPrincipalLabAnalytics)
				principal.GET("/analytics/contests", handlers.GetPrincipalContestAnalytics)
				principal.GET("/analytics/branches", handlers.GetPrincipalBranchAnalytics)
				principal.GET("/analytics/students", handlers.GetPrincipalStudentAnalytics)
				principal.GET("/analytics/years", handlers.GetPrincipalYearAnalytics)
				principal.GET("/analytics/insights", handlers.GetPrincipalInsights)
				principal.GET("/analytics/alerts", handlers.GetPrincipalAlerts)
				principal.GET("/analytics/accreditation", handlers.GetPrincipalAccreditation)
			}
		}
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Start server
	log.Printf("Server starting on port %s", config.AppConfig.Port)
	if err := router.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
