package database

import (
	"coding-platform/config"
	"coding-platform/models"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() error {
	var err error

	DB, err = gorm.Open(postgres.Open(config.AppConfig.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		// Disable automatic FK constraint creation during AutoMigrate
		// We'll create FK constraints manually via SQL for full control
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		return err
	}

	log.Println("Database connected successfully")
	return nil
}

// DropAllTables drops all tables in the database
// Use with caution - this will delete all data!
func DropAllTables() error {
	log.Println("Dropping all tables...")

	// Drop all tables in reverse dependency order
	// Table names are from a hardcoded array, not user input, so concatenation is safe
	tables := []string{
		"exam_results",
		"exams",
		"faculty_assignments",
		"enrollments",
		"course_offerings",
		"faculties",
		"students",
		"curriculum_courses",
		"curriculums",
		"regulations",
		"course_assignments",
		"user_streaks",
		"user_activities",
		"solve_sessions",
		"plagiarism_results",
		"plagiarism_matches",
		"submissions",
		"user_problem_completions",
		"contest_participants",
		"contest_problems",
		"contests",
		"test_cases",
		"problems",
		"theory_modules",
		"theory_weeks",
		"theories",
		"lab_sessions",
		"labs",
		"courses",
		"sections",
		"branches",
		"programs",
		"academic_years",
		"role_permissions",
		"permissions",
		"roles",
		"colleges",
		"users",
	}

	for _, table := range tables {
		// Use double quotes for identifier quoting (PostgreSQL standard)
		// Table names are from hardcoded array, not user input
		if err := DB.Exec(`DROP TABLE IF EXISTS "` + table + `" CASCADE`).Error; err != nil {
			log.Printf("Warning: failed to drop table %s: %v", table, err)
		}
	}

	log.Println("All tables dropped successfully")
	return nil
}

// FreshStart drops all tables and runs migration
func FreshStart() error {
	if err := DropAllTables(); err != nil {
		return err
	}
	return Migrate()
}

func Migrate() error {
	log.Println("Starting database migration...")

	// =====================================================
	// PRE-MIGRATION FIX: GORM v1.30 discovers related models during AutoMigrate
	// and tries DROP CONSTRAINT "uni_courses_course_code" (without IF EXISTS)
	// when Course.CourseCode no longer has a unique tag. This fails if the
	// constraint doesn't exist. Fix: drop any stale unique index/constraint
	// on courses.course_code BEFORE any AutoMigrate runs.
	// =====================================================
	var coursesTableExists int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = CURRENT_SCHEMA() AND table_name = 'courses'").Scan(&coursesTableExists)
	if coursesTableExists > 0 {
		log.Println("Pre-migration: cleaning up stale unique constraint on courses.course_code...")
		// The actual constraint may have different names depending on how it was created
		DB.Exec(`ALTER TABLE courses DROP CONSTRAINT IF EXISTS uni_courses_course_code`)
		DB.Exec(`ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_course_code_key`)
		DB.Exec(`DROP INDEX IF EXISTS uni_courses_course_code`)
		DB.Exec(`DROP INDEX IF EXISTS courses_course_code_key`)
	}

	// =====================================================
	// LEVEL 1: Independent tables (no foreign keys)
	// =====================================================
	if err := DB.AutoMigrate(
		&models.Role{},
		&models.Permission{},
		&models.College{},
		&models.Program{},
		&models.AcademicYear{},
		&models.Subject{}, // NEW - subjects for topic categorization
	); err != nil {
		return err
	}

	// Level 1.5: Topics (depend on Subjects)
	if err := DB.AutoMigrate(
		&models.Topic{}, // NEW - canonical topics for problems and lab sessions
	); err != nil {
		return err
	}

	// =====================================================
	// LEVEL 2: Academic structure (new batch-based)
	// =====================================================
	if err := DB.AutoMigrate(
		&models.RolePermission{},
		&models.User{},
		&models.Branch{},
		&models.Regulation{}, // NEW - curriculum versioning
		&models.Curriculum{}, // NEW - program-regulation curriculum
	); err != nil {
		return err
	}

	// =====================================================
	// LEVEL 3: Mappings and Sections (redesigned, no semester)
	// =====================================================
	if err := DB.AutoMigrate(
		&models.CurriculumCourse{}, // NEW - course to curriculum mapping
		&models.Section{},          // Redesigned - uses CohortYear, not Semester
	); err != nil {
		return err
	}

	// =====================================================
	// LEVEL 4: Course tables (unchanged structure, cleaned fields)
	// =====================================================
	// GORM's AutoMigrate generates DROP CONSTRAINT "uni_courses_course_code"
	// (without IF EXISTS) when CourseCode no longer has a unique tag.
	// The constraint doesn't exist, so we safely ignore that specific error.
	log.Println("Running AutoMigrate for Course tables...")

	if err := DB.AutoMigrate(
		&models.Course{},
		&models.Lab{},
		&models.Theory{},
	); err != nil {
		return err
	}

	// Level 4.5: Lab and Theory children
	if err := DB.AutoMigrate(
		&models.LabSession{},
		&models.TheoryWeek{},
		&models.TheoryModule{},
		&models.TheoryPDF{},
	); err != nil {
		return err
	}

	// =====================================================
	// LEVEL 5: User and Student/Faculty extensions
	// =====================================================
	// AutoMigrate without relationships creating FK constraints
	if err := DB.AutoMigrate(
		&models.Student{}, // NEW - student academic identity
		&models.Faculty{}, // NEW - faculty employment info
	); err != nil {
		return err
	}

	// IMPORTANT: Drop any backwards FK constraints that GORM created
	log.Println("Fixing foreign key constraints...")
	DB.Exec(`
		-- Drop ALL FK constraints related to students/faculties/users/branches that GORM may have incorrectly created
		ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_students_user CASCADE;
		ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_faculties_user CASCADE;
		ALTER TABLE IF EXISTS students DROP CONSTRAINT IF EXISTS fk_students_user CASCADE;
		ALTER TABLE IF EXISTS faculties DROP CONSTRAINT IF EXISTS fk_faculties_user CASCADE;
		ALTER TABLE IF EXISTS branches DROP CONSTRAINT IF EXISTS fk_faculties_branch CASCADE;
		ALTER TABLE IF EXISTS branches DROP CONSTRAINT IF EXISTS fk_students_branch CASCADE;
		ALTER TABLE IF EXISTS branches DROP CONSTRAINT IF EXISTS fk_students_curriculum CASCADE;
		ALTER TABLE IF EXISTS branches DROP CONSTRAINT IF EXISTS fk_students_section CASCADE;
	`)

	// Add correct FK with different names to avoid confusion: students.regd_no -> users.regdno
	log.Println("Adding correct FK constraint for students...")
	if err := DB.Exec(`
		ALTER TABLE students 
		ADD CONSTRAINT fk_students_regdno_users 
		FOREIGN KEY (regd_no) REFERENCES users(regdno) ON DELETE CASCADE
	`).Error; err != nil {
		log.Printf("Warning: Could not add FK constraint for students: %v", err)
	} else {
		log.Println("✓ Students FK constraint added correctly")
	}

	// Add correct FK with different names: faculties.regd_no -> users.regdno
	log.Println("Adding correct FK constraint for faculties...")
	if err := DB.Exec(`
		ALTER TABLE faculties 
		ADD CONSTRAINT fk_faculties_regdno_users 
		FOREIGN KEY (regd_no) REFERENCES users(regdno) ON DELETE CASCADE
	`).Error; err != nil {
		log.Printf("Warning: Could not add FK constraint for faculties: %v", err)
	} else {
		log.Println("✓ Faculties FK constraint added correctly")
	}

	// Add correct FK: students.branch_id -> branches.branch_id
	log.Println("Adding correct FK constraint for students → branches...")
	DB.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_students_branch_id'
			) THEN
				ALTER TABLE students
				ADD CONSTRAINT fk_students_branch_id
				FOREIGN KEY (branch_id) REFERENCES branches(branch_id) ON DELETE RESTRICT;
			END IF;
	END $$;
	`)
	// Add correct FK: faculties.branch_id -> branches.branch_id
	log.Println("Adding correct FK constraint for faculties → branches...")
	DB.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'fk_faculties_branch_id'
			) THEN
				ALTER TABLE faculties
				ADD CONSTRAINT fk_faculties_branch_id
				FOREIGN KEY (branch_id) REFERENCES branches(branch_id) ON DELETE RESTRICT;
			END IF;
	END $$;
	`)

	// =====================================================
	// LEVEL 6: CourseOffering (NEW)
	// =====================================================
	if err := DB.AutoMigrate(
		&models.CourseOffering{}, // NEW - academic course instance
	); err != nil {
		return err
	}

	// Drop study_year_index column from course_offerings (no longer used)
	DB.Exec(`ALTER TABLE course_offerings DROP COLUMN IF EXISTS study_year_index`)
	DB.Exec(`DROP INDEX IF EXISTS idx_offerings_study_year`)

	// Drop study_year_index column from curriculum_courses (no longer used)
	DB.Exec(`ALTER TABLE curriculum_courses DROP COLUMN IF EXISTS study_year_index`)
	DB.Exec(`DROP INDEX IF EXISTS idx_curriculum_courses_year`)
	DB.Exec(`DROP INDEX IF EXISTS curriculum_courses_curriculum_id_course_id_study_year_index_key`)

	// Make curriculum_id nullable (GORM AutoMigrate doesn't change NOT NULL constraints)
	DB.Exec(`ALTER TABLE course_offerings ALTER COLUMN curriculum_id DROP NOT NULL`)

	// Ensure users.regdno has a unique constraint (required for FK references)
	// GORM AutoMigrate won't add/change primary keys on existing tables
	log.Println("Ensuring unique constraint on users.regdno...")
	DB.Exec(`
		DO $$
		BEGIN
			-- Check if regdno is already the primary key or has a unique constraint
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conrelid = 'users'::regclass 
				AND contype IN ('p', 'u')
				AND EXISTS (
					SELECT 1 FROM unnest(conkey) AS k
					JOIN pg_attribute a ON a.attrelid = conrelid AND a.attnum = k
					WHERE a.attname = 'regdno'
				)
			) THEN
				ALTER TABLE users ADD CONSTRAINT users_regdno_unique UNIQUE (regdno);
			END IF;
		END $$;
	`)

	// =====================================================
	// LEVEL 7: Enrollment and Faculty Assignment (redesigned)
	// =====================================================
	if err := DB.AutoMigrate(
		&models.Enrollment{},        // Redesigned - replaces StudentEnrollment
		&models.FacultyAssignment{}, // Redesigned - replaces old assignment models
		&models.CourseAssignment{},  // Simplified faculty-to-course assignment for HOD management
	); err != nil {
		return err
	}

	// =====================================================
	// LEVEL 8: Exams and Results (NEW)
	// =====================================================
	if err := DB.AutoMigrate(
		&models.Exam{},       // NEW - exam linked to CourseOffering
		&models.ExamResult{}, // NEW - exam results
	); err != nil {
		return err
	}

	// =====================================================
	// LEVEL 9: Problems, Contests, Submissions (unchanged)
	// =====================================================
	if err := DB.AutoMigrate(
		&models.Problem{},
		&models.TestCase{},
	); err != nil {
		return err
	}

	// PRE-MIGRATION CLEANUP: Drop any incorrect FK constraints that may have been created
	// when FK constraints were disabled. These constraints are wrong because they create
	// reverse FKs from parent tables to child tables.
	log.Println("Cleaning up any incorrect FK constraints before contest migration...")
	DB.Exec(`ALTER TABLE IF EXISTS contests DROP CONSTRAINT IF EXISTS fk_contest_problems_contest CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS contests DROP CONSTRAINT IF EXISTS fk_contest_submissions_contest CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS contests DROP CONSTRAINT IF EXISTS fk_contest_participants_contest CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS contests DROP CONSTRAINT IF EXISTS fk_contest_problem_test_cases_contest CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS problems DROP CONSTRAINT IF EXISTS fk_contest_problems_problem CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS problems DROP CONSTRAINT IF EXISTS fk_contest_submissions_problem CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS problems DROP CONSTRAINT IF EXISTS fk_contest_problem_test_cases_problem CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS test_cases DROP CONSTRAINT IF EXISTS fk_contest_problem_test_cases_test_case CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_contest_submissions_user CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_contest_participants_user CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS colleges DROP CONSTRAINT IF EXISTS fk_contest_submissions_college CASCADE`)
	log.Println("✓ Cleanup completed")

	if err := DB.AutoMigrate(
		&models.Contest{},
		&models.ContestProblem{},
		&models.ContestParticipant{},
		&models.ContestSubmission{},
		&models.ContestProblemTestCase{},
		&models.ContestLeaderboardSnapshot{},
		&models.ContestNonParticipantsSnapshot{},
	); err != nil {
		return err
	}

	if err := DB.AutoMigrate(
		&models.Submission{},
		&models.UserProblemCompletion{},
		&models.SolveSession{}, // NEW - tracks solve sessions for accurate time tracking
	); err != nil {
		return err
	}

	// CLEANUP: Drop any incorrect reverse FK constraints on submissions and related tables
	log.Println("Cleaning up any incorrect FK constraints on submissions...")
	DB.Exec(`ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_submissions_user CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS problems DROP CONSTRAINT IF EXISTS fk_submissions_problem CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS contests DROP CONSTRAINT IF EXISTS fk_submissions_contest CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS courses DROP CONSTRAINT IF EXISTS fk_submissions_course CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS sections DROP CONSTRAINT IF EXISTS fk_submissions_section CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS lab_sessions DROP CONSTRAINT IF EXISTS fk_submissions_lab_session CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS colleges DROP CONSTRAINT IF EXISTS fk_submissions_college CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_user_problem_completions_user CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS problems DROP CONSTRAINT IF EXISTS fk_user_problem_completions_problem CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS submissions DROP CONSTRAINT IF EXISTS fk_user_problem_completions_submission CASCADE`)
	log.Println("✓ Submissions cleanup completed")

	// =====================================================
	// LEVEL 10: Plagiarism and Analytics (unchanged)
	// =====================================================
	// CLEANUP: Drop any incorrect reverse FK constraints on plagiarism tables
	log.Println("Cleaning up any incorrect FK constraints on plagiarism tables...")
	DB.Exec(`ALTER TABLE IF EXISTS submissions DROP CONSTRAINT IF EXISTS fk_plagiarism_results_submission1 CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS submissions DROP CONSTRAINT IF EXISTS fk_plagiarism_results_submission2 CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS plagiarism_results DROP CONSTRAINT IF EXISTS fk_plagiarism_matches_result CASCADE`)
	log.Println("✓ Plagiarism cleanup completed")

	if err := DB.AutoMigrate(
		&models.PlagiarismResult{},
		&models.PlagiarismMatch{},
	); err != nil {
		return err
	}

	// CLEANUP: Drop any incorrect reverse FK constraints on user streak/activity tables
	log.Println("Cleaning up any incorrect FK constraints on user streak tables...")
	DB.Exec(`ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_user_streaks_user CASCADE`)
	DB.Exec(`ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_user_activities_user CASCADE`)
	log.Println("✓ User streak cleanup completed")

	if err := DB.AutoMigrate(
		&models.UserStreak{},
		&models.UserActivity{},
	); err != nil {
		return err
	}

	// =====================================================
	// LEVEL 11: Quiz System (AI-powered quiz generation)
	// =====================================================
	log.Println("Migrating Level 11: Quiz System tables...")
	if err := DB.AutoMigrate(
		&models.Quiz{},
		&models.QuizTheorySource{},
		&models.QuizQuestion{},
		&models.QuizQuestionOption{},
		&models.QuizAttempt{},
		&models.QuizAnswer{},
	); err != nil {
		return err
	}
	log.Println("✓ Level 11: Quiz System tables migrated successfully")

	// =====================================================
	// LEVEL 12: Practice Quiz System (embedded in theory courses)
	// =====================================================
	log.Println("Migrating Level 12: Practice Quiz System tables...")
	if err := DB.AutoMigrate(
		&models.PracticeQuiz{},
		&models.PracticeQuizQuestion{},
		&models.PracticeQuizOption{},
		&models.GenerationJob{},
	); err != nil {
		return err
	}
	log.Println("✓ Level 12: Practice Quiz System tables migrated successfully")

	// =====================================================
	// LEVEL 13: Contest Generation System (AI-powered contest problems)
	// =====================================================
	log.Println("Migrating Level 13: Contest Generation System tables...")
	if err := DB.AutoMigrate(
		&models.ContestEditorial{},
	); err != nil {
		return err
	}
	log.Println("✓ Level 13: Contest Generation System tables migrated successfully")

	// =====================================================
	// FINAL CLEANUP: Remove ANY remaining backwards FK constraints
	// =====================================================
	log.Println("Final cleanup: Removing any remaining backwards FK constraints from users table...")
	DB.Exec(`
		ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_students_user CASCADE;
		ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_faculties_user CASCADE;
		ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_users_student CASCADE;
		ALTER TABLE IF EXISTS users DROP CONSTRAINT IF EXISTS fk_users_faculty CASCADE;
	`)
	log.Println("✓ Final cleanup completed - backwards FK constraints removed from users table")

	// =====================================================
	// FIX: enrollments table may have duplicate column (student_regd_no vs student_regdno)
	// GORM default naming created student_regd_no, migration SQL uses student_regdno
	// =====================================================
	DB.Exec(`
		DO $$
		BEGIN
			-- If both columns exist, drop the GORM-default one (student_regd_no)
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='enrollments' AND column_name='student_regd_no')
			   AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='enrollments' AND column_name='student_regdno') THEN
				ALTER TABLE enrollments DROP COLUMN student_regd_no;
				RAISE NOTICE 'Dropped duplicate column student_regd_no from enrollments';
			-- If only the GORM-default exists, rename it
			ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='enrollments' AND column_name='student_regd_no')
			      AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='enrollments' AND column_name='student_regdno') THEN
				ALTER TABLE enrollments RENAME COLUMN student_regd_no TO student_regdno;
				RAISE NOTICE 'Renamed student_regd_no to student_regdno in enrollments';
			END IF;
		END $$
	`)
	log.Println("✓ Enrollments column name fix applied")

	// Same fix for exam_results table
	DB.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='exam_results' AND column_name='student_regd_no')
			   AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='exam_results' AND column_name='student_regdno') THEN
				ALTER TABLE exam_results DROP COLUMN student_regd_no;
			ELSIF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='exam_results' AND column_name='student_regd_no')
			      AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='exam_results' AND column_name='student_regdno') THEN
				ALTER TABLE exam_results RENAME COLUMN student_regd_no TO student_regdno;
			END IF;
		END $$
	`)

	// =====================================================
	// ENSURE a current academic year exists
	// =====================================================
	var currentAY models.AcademicYear
	DB.Where("is_current = ?", true).First(&currentAY)
	if currentAY.AcademicYearID == 0 {
		// No year marked current – pick the most recent one and mark it
		var latest models.AcademicYear
		DB.Order("start_date DESC").First(&latest)
		if latest.AcademicYearID > 0 {
			DB.Model(&latest).Update("is_current", true)
			currentAY = latest
			log.Printf("✓ Marked academic year '%s' as current", latest.Name)
		}
	}

	// =====================================================
	// NOTE: CourseOfferings are created per-branch when admin creates a course
	// via CreateCourse handler. No blanket backfill here — that would assign
	// every course to every section regardless of branch.
	// =====================================================

	// =====================================================
	// BACKFILL: auto-enroll students in their section's offerings
	// Only enrolls students in offerings that match their section (which
	// already encodes the correct branch via the section's branch_id).
	// =====================================================
	var studentsWithSection int64
	var activeOfferings int64
	DB.Raw("SELECT COUNT(*) FROM students WHERE section_id IS NOT NULL AND status = 'active'").Scan(&studentsWithSection)
	DB.Raw("SELECT COUNT(*) FROM course_offerings WHERE is_active = true AND deleted_at IS NULL").Scan(&activeOfferings)
	log.Printf("Enrollment backfill: %d active students with sections, %d active offerings", studentsWithSection, activeOfferings)

	enrollResult := DB.Exec(`
		INSERT INTO enrollments (student_regdno, course_offering_id, college_id, type, status, attempt_no, enrolled_at)
		SELECT s.regd_no, co.id, u.college_id, 'Regular', 'enrolled', 1, NOW()
		FROM students s
		INNER JOIN users u ON u.regdno = s.regd_no
		INNER JOIN course_offerings co ON co.section_id = s.section_id
		WHERE s.status = 'active'
		  AND co.is_active = true
		  AND co.deleted_at IS NULL
		  AND u.college_id IS NOT NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM enrollments e
		    WHERE e.student_regdno = s.regd_no AND e.course_offering_id = co.id
		  )
	`)
	if enrollResult.Error != nil {
		log.Printf("ERROR auto-enrolling students: %v", enrollResult.Error)
	} else if enrollResult.RowsAffected > 0 {
		log.Printf("✓ Auto-enrolled students: %d new enrollment(s) created", enrollResult.RowsAffected)
	} else {
		log.Printf("No new enrollments created (students may lack section_id or already enrolled)")
	}

	// =====================================================
	// SECURITY: Add database constraints for college_id columns
	// =====================================================
	log.Println("Adding security constraints for college isolation...")

	// Create indexes on college_id columns for query performance
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_problems_college ON problems(college_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_users_college ON users(college_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_courses_college ON courses(college_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_branches_college ON branches(college_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_contests_college ON contests(college_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_user_problem_completions_college ON user_problem_completions(college_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_submissions_college ON submissions(college_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_enrollments_college ON enrollments(college_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_faculty_assignments_college ON faculty_assignments(college_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_contest_submissions_college ON contest_submissions(college_id)`)
	log.Println("✓ Created indexes on college_id columns")

	// =====================================================
	// PERFORMANCE: Add missing indexes for frequently queried columns
	// =====================================================
	log.Println("Adding performance indexes...")

	// Submissions table indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_submissions_user_regd_no ON submissions(user_regd_no)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_submissions_problem_id ON submissions(problem_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_submissions_passed ON submissions(passed)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_submissions_user_passed ON submissions(user_regd_no, passed)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_submissions_problem_stats ON submissions(problem_id, passed, execution_time)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_submissions_created_at ON submissions(created_at)`)

	// User activity and streaks indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_user_activities_user_date ON user_activities(user_regd_no, date)`)
	// Drop old non-unique index if it exists, then create unique index for ON CONFLICT to work
	DB.Exec(`DROP INDEX IF EXISTS idx_user_streaks_user_regd_no`)
	DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_user_streaks_user_regd_no_unique ON user_streaks(user_regd_no)`)

	// User problem completions indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_user_problem_completions_lookup ON user_problem_completions(user_regd_no, problem_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_user_problem_completions_problem ON user_problem_completions(problem_id)`)

	// Quiz indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_quiz_attempts_quiz_student ON quiz_attempts(quiz_id, student_regdno)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_quiz_attempts_student ON quiz_attempts(student_regdno)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_quiz_questions_quiz_id ON quiz_questions(quiz_id)`)

	// Faculty assignment indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_faculty_assignments_course ON faculty_assignments(course_offering_id, faculty_regd_no)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_faculty_assignments_faculty ON faculty_assignments(faculty_regd_no)`)

	// Course offering indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_course_offerings_course_section ON course_offerings(course_id, section_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_enrollments_student ON enrollments(student_regdno)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_enrollments_course ON enrollments(course_offering_id)`)

	// Problem indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_problems_created_by ON problems(created_by)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_problems_difficulty ON problems(difficulty)`)

	// Test case indexes
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_test_cases_problem ON test_cases(problem_id)`)

	log.Println("✓ Performance indexes added")

	// =====================================================
	// FOREIGN KEY CONSTRAINTS (Manual creation for correctness)
	// =====================================================
	log.Println("Creating foreign key constraints manually...")

	// Contest system FKs
	DB.Exec(`DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_contest_problems_contest') THEN
			ALTER TABLE contest_problems ADD CONSTRAINT fk_contest_problems_contest
			FOREIGN KEY (contest_id) REFERENCES contests(contest_id) ON DELETE CASCADE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_contest_problems_problem') THEN
			ALTER TABLE contest_problems ADD CONSTRAINT fk_contest_problems_problem
			FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_contest_participants_contest') THEN
			ALTER TABLE contest_participants ADD CONSTRAINT fk_contest_participants_contest
			FOREIGN KEY (contest_id) REFERENCES contests(contest_id) ON DELETE CASCADE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_contest_participants_user') THEN
			ALTER TABLE contest_participants ADD CONSTRAINT fk_contest_participants_user
			FOREIGN KEY (user_regd_no) REFERENCES users(regdno) ON DELETE CASCADE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_contest_submissions_contest') THEN
			ALTER TABLE contest_submissions ADD CONSTRAINT fk_contest_submissions_contest
			FOREIGN KEY (contest_id) REFERENCES contests(contest_id) ON DELETE CASCADE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_contest_submissions_problem') THEN
			ALTER TABLE contest_submissions ADD CONSTRAINT fk_contest_submissions_problem
			FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_contest_submissions_user') THEN
			ALTER TABLE contest_submissions ADD CONSTRAINT fk_contest_submissions_user
			FOREIGN KEY (user_regdno) REFERENCES users(regdno) ON DELETE CASCADE;
		END IF;
	END $$`)

	// Submission FKs
	DB.Exec(`DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_submissions_user') THEN
			ALTER TABLE submissions ADD CONSTRAINT fk_submissions_user
			FOREIGN KEY (user_regd_no) REFERENCES users(regdno) ON DELETE CASCADE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_submissions_problem') THEN
			ALTER TABLE submissions ADD CONSTRAINT fk_submissions_problem
			FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE;
		END IF;
	END $$`)

	// Enrollment FKs
	DB.Exec(`DO $$
	BEGIN
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_enrollments_student') THEN
			ALTER TABLE enrollments ADD CONSTRAINT fk_enrollments_student
			FOREIGN KEY (student_regdno) REFERENCES users(regdno) ON DELETE CASCADE;
		END IF;
		IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_enrollments_course_offering') THEN
			ALTER TABLE enrollments ADD CONSTRAINT fk_enrollments_course_offering
			FOREIGN KEY (course_offering_id) REFERENCES course_offerings(id) ON DELETE CASCADE;
		END IF;
	END $$`)

	log.Println("✓ Foreign key constraints created")

	// Add constraint to ensure college_id is set for non-global problems
	// Note: PostgreSQL doesn't support CHECK constraints with OR conditions on nullable fields elegantly
	// This is enforced at application level in handlers
	log.Println("✓ Security constraints applied (college validation enforced at handler level)")

	// =====================================================
	// STEP: Run SQL migration files (016_contest_generation_system.sql, etc.)
	// =====================================================
	log.Println("Running SQL migration files...")
	runSQLMigrations()

	log.Println("Database migration completed successfully!")
	return nil
}

// runSQLMigrations executes SQL migration files in order
func runSQLMigrations() {
	// Migration files are numbered sequentially for deterministic execution order
	migrationFiles := []string{
		"migrations/001_batch_based_architecture.sql",
		"migrations/002_separate_theory_lab_courses.sql",
		"migrations/003_remove_study_year_from_offerings.sql",
		"migrations/004_remove_study_year_from_curriculum_courses.sql",
		"migrations/005_college_id_to_string.sql",
		"migrations/006_contest_system_enhancements.sql",
		"migrations/007_quiz_system.sql",
		"migrations/008_phase1_lab_theory_refactor.sql",
		"migrations/009_practice_quizzes.sql",
		"migrations/010_contest_generation_system.sql",
		"migrations/011_database_constraints.sql",
		"migrations/012_enrollment_unique_constraint.sql",
		"migrations/013_contest_first_ac_constraint.sql",
		"migrations/019_theory_pdfs_table.sql",
	}

	for _, file := range migrationFiles {
		log.Printf("Running SQL migration: %s", file)

		sqlContent, err := readFile(file)
		if err != nil {
			log.Printf("Warning: Could not read migration file %s: %v", file, err)
			continue
		}

		// Execute each migration in a separate transaction
		// This ensures that if one migration fails, we can continue with the next
		err = DB.Transaction(func(tx *gorm.DB) error {
			return tx.Exec(string(sqlContent)).Error
		})

		if err != nil {
			log.Printf("Warning: Failed to execute migration %s: %v", file, err)
			// Continue to next migration - the transaction is already rolled back
			continue
		}

		log.Printf("✓ Migration %s completed", file)
	}

}

// readFile reads a file from the filesystem
func readFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return data, nil
}
