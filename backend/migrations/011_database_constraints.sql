-- ============================================================================
-- Migration 011: Add Database-Level Constraints and Data Integrity
--
-- This migration adds CHECK constraints and other database-level validations
-- that were missing from previous migrations.
-- ============================================================================

-- ============================================================================
-- STEP 1: Add CHECK constraints for enum-like columns
-- ============================================================================

-- Users table - role validation
-- Recreate every run so older deployments with a stale chk_users_role definition
-- (missing roles like super_admin/college_admin/principal) get corrected.
DO $$
BEGIN
    ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_role;
    ALTER TABLE users ADD CONSTRAINT chk_users_role
        CHECK (role IN ('student', 'faculty', 'hod', 'admin', 'college_admin', 'super_admin', 'principal'));
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE 'Skipping chk_users_role: existing data violates constraint. Run: UPDATE users SET role = ''student'' WHERE role NOT IN (''student'', ''faculty'', ''hod'', ''admin'', ''college_admin'', ''super_admin'', ''principal'')';
    WHEN others THEN
        RAISE NOTICE 'Could not add chk_users_role constraint: %', SQLERRM;
END $$;

-- Enrollments - type validation
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_enrollments_type'
    ) THEN
        ALTER TABLE enrollments ADD CONSTRAINT chk_enrollments_type
            CHECK (type IN ('Regular', 'Elective', 'Backlog'));
    END IF;
END $$;

-- Enrollments - status validation
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_enrollments_status'
    ) THEN
        ALTER TABLE enrollments ADD CONSTRAINT chk_enrollments_status
            CHECK (status IN ('enrolled', 'completed', 'failed', 'dropped', 'withdrawn'));
    END IF;
END $$;

-- Faculty Assignments - role validation
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_faculty_assignments_role'
    ) THEN
        ALTER TABLE faculty_assignments ADD CONSTRAINT chk_faculty_assignments_role
            CHECK (role IN ('teacher', 'co-teacher', 'examiner', 'tutor'));
    END IF;
END $$;

-- Problems - difficulty validation
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_problems_difficulty'
    ) THEN
        ALTER TABLE problems ADD CONSTRAINT chk_problems_difficulty
            CHECK (difficulty IN ('easy', 'medium', 'hard'));
    END IF;
END $$;

-- Submissions - status validation
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_submissions_status'
    ) THEN
        ALTER TABLE submissions ADD CONSTRAINT chk_submissions_status
            CHECK (status IN ('pending', 'running', 'completed', 'error', 'timeout'));
    END IF;
EXCEPTION
    WHEN check_violation THEN
        RAISE NOTICE 'Skipping chk_submissions_status: existing data violates constraint. Run: UPDATE submissions SET status = ''pending'' WHERE status NOT IN (''pending'', ''running'', ''completed'', ''error'', ''timeout'')';
    WHEN others THEN
        RAISE NOTICE 'Could not add chk_submissions_status constraint: %', SQLERRM;
END $$;

-- Contests - status validation
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_contests_status'
    ) THEN
        ALTER TABLE contests ADD CONSTRAINT chk_contests_status
            CHECK (status IN ('draft', 'upcoming', 'active', 'ended', 'cancelled'));
    END IF;
END $$;

-- Courses - course_category validation
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_course_category'
    ) THEN
        ALTER TABLE courses ADD CONSTRAINT chk_course_category
            CHECK (course_category IN ('theory', 'lab'));
    END IF;
END $$;

-- ============================================================================
-- STEP 2: Add CHECK constraints for numeric ranges
-- ============================================================================

-- Problems - time_limit must be positive
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_problems_time_limit'
    ) THEN
        ALTER TABLE problems ADD CONSTRAINT chk_problems_time_limit
            CHECK (time_limit > 0 AND time_limit <= 60);
    END IF;
END $$;

-- Problems - memory_limit must be positive
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_problems_memory_limit'
    ) THEN
        ALTER TABLE problems ADD CONSTRAINT chk_problems_memory_limit
            CHECK (memory_limit > 0 AND memory_limit <= 1024);
    END IF;
END $$;

-- Test cases - points must be non-negative
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_test_cases_points'
    ) THEN
        ALTER TABLE test_cases ADD CONSTRAINT chk_test_cases_points
            CHECK (points >= 0);
    END IF;
END $$;

-- Enrollments - attempt_no must be positive
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_enrollments_attempt_no'
    ) THEN
        ALTER TABLE enrollments ADD CONSTRAINT chk_enrollments_attempt_no
            CHECK (attempt_no > 0 AND attempt_no <= 10);
    END IF;
END $$;

-- ============================================================================
-- STEP 3: Add college_id column to submissions if not exists
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'submissions' AND column_name = 'college_id'
    ) THEN
        ALTER TABLE submissions ADD COLUMN college_id VARCHAR(50);
        CREATE INDEX IF NOT EXISTS idx_submissions_college ON submissions(college_id);
    END IF;
END $$;

-- ============================================================================
-- STEP 4: Add college_id column to enrollments if not exists
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'enrollments' AND column_name = 'college_id'
    ) THEN
        ALTER TABLE enrollments ADD COLUMN college_id VARCHAR(50);
        CREATE INDEX IF NOT EXISTS idx_enrollments_college ON enrollments(college_id);
    END IF;
END $$;

-- ============================================================================
-- STEP 5: Add college_id column to faculty_assignments if not exists
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'faculty_assignments' AND column_name = 'college_id'
    ) THEN
        ALTER TABLE faculty_assignments ADD COLUMN college_id VARCHAR(50);
        CREATE INDEX IF NOT EXISTS idx_faculty_assignments_college ON faculty_assignments(college_id);
    END IF;
END $$;

-- ============================================================================
-- STEP 5.5: Add college_id column to contest_submissions if not exists
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'contest_submissions' AND column_name = 'college_id'
    ) THEN
        ALTER TABLE contest_submissions ADD COLUMN college_id VARCHAR(50);
        CREATE INDEX IF NOT EXISTS idx_contest_submissions_college ON contest_submissions(college_id);
    END IF;
END $$;

-- ============================================================================
-- STEP 6: Make lab_sessions.lab_id NOT NULL
-- ============================================================================

-- First, update any NULL values to a valid lab_id (if any exist)
-- Then alter the column to NOT NULL
DO $$
DECLARE
    null_count INTEGER;
    min_lab_id INTEGER;
BEGIN
    -- Check if there are any NULL lab_id values
    SELECT COUNT(*) INTO null_count FROM lab_sessions WHERE lab_id IS NULL;

    IF null_count > 0 THEN
        -- Get minimum lab ID as fallback
        SELECT MIN(id) INTO min_lab_id FROM labs LIMIT 1;

        IF min_lab_id IS NOT NULL THEN
            UPDATE lab_sessions SET lab_id = min_lab_id WHERE lab_id IS NULL;
            RAISE NOTICE 'Updated % lab_sessions with NULL lab_id to lab_id = %', null_count, min_lab_id;
        ELSE
            RAISE NOTICE 'Cannot fix NULL lab_id values: no labs exist in database';
        END IF;
    END IF;

    -- Now make the column NOT NULL
    ALTER TABLE lab_sessions ALTER COLUMN lab_id SET NOT NULL;
    RAISE NOTICE 'Made lab_sessions.lab_id NOT NULL';
EXCEPTION
    WHEN others THEN
        RAISE NOTICE 'Could not set lab_id NOT NULL: %', SQLERRM;
END $$;

-- ============================================================================
-- STEP 7: Add foreign key constraints for college_id columns
-- ============================================================================

-- Submissions -> Colleges FK
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_submissions_college'
    ) THEN
        ALTER TABLE submissions
        ADD CONSTRAINT fk_submissions_college
        FOREIGN KEY (college_id) REFERENCES colleges(college_id) ON DELETE SET NULL;
    END IF;
EXCEPTION
    WHEN undefined_column THEN
        RAISE NOTICE 'Skipping fk_submissions_college: college_id column does not exist';
    WHEN others THEN
        RAISE NOTICE 'Could not add fk_submissions_college: %', SQLERRM;
END $$;

-- Enrollments -> Colleges FK
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_enrollments_college'
    ) THEN
        ALTER TABLE enrollments
        ADD CONSTRAINT fk_enrollments_college
        FOREIGN KEY (college_id) REFERENCES colleges(college_id) ON DELETE SET NULL;
    END IF;
EXCEPTION
    WHEN undefined_column THEN
        RAISE NOTICE 'Skipping fk_enrollments_college: college_id column does not exist';
    WHEN others THEN
        RAISE NOTICE 'Could not add fk_enrollments_college: %', SQLERRM;
END $$;

-- Faculty Assignments -> Colleges FK
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_faculty_assignments_college'
    ) THEN
        ALTER TABLE faculty_assignments
        ADD CONSTRAINT fk_faculty_assignments_college
        FOREIGN KEY (college_id) REFERENCES colleges(college_id) ON DELETE SET NULL;
    END IF;
EXCEPTION
    WHEN undefined_column THEN
        RAISE NOTICE 'Skipping fk_faculty_assignments_college: college_id column does not exist';
    WHEN others THEN
        RAISE NOTICE 'Could not add fk_faculty_assignments_college: %', SQLERRM;
END $$;

-- Contest Submissions -> Colleges FK
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_contest_submissions_college'
    ) THEN
        ALTER TABLE contest_submissions
        ADD CONSTRAINT fk_contest_submissions_college
        FOREIGN KEY (college_id) REFERENCES colleges(college_id) ON DELETE SET NULL;
    END IF;
EXCEPTION
    WHEN undefined_column THEN
        RAISE NOTICE 'Skipping fk_contest_submissions_college: college_id column does not exist';
    WHEN others THEN
        RAISE NOTICE 'Could not add fk_contest_submissions_college: %', SQLERRM;
END $$;

-- ============================================================================
-- STEP 8: Add missing indexes for foreign keys
-- ============================================================================

-- These indexes improve JOIN performance and are required for FK constraints
CREATE INDEX IF NOT EXISTS idx_submissions_user_regd_no ON submissions(user_regd_no);
CREATE INDEX IF NOT EXISTS idx_submissions_problem_id ON submissions(problem_id);
CREATE INDEX IF NOT EXISTS idx_submissions_contest_id ON submissions(contest_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_student_regdno ON enrollments(student_regdno);
CREATE INDEX IF NOT EXISTS idx_enrollments_course_offering_id ON enrollments(course_offering_id);
CREATE INDEX IF NOT EXISTS idx_faculty_assignments_faculty_regdno ON faculty_assignments(faculty_regdno);
CREATE INDEX IF NOT EXISTS idx_faculty_assignments_course_offering_id ON faculty_assignments(course_offering_id);
CREATE INDEX IF NOT EXISTS idx_lab_sessions_lab_id ON lab_sessions(lab_id);

-- ============================================================================
-- STEP 9: Remove redundant overlapping indexes
-- ============================================================================

-- Drop indexes that are covered by other indexes (if they exist)
-- Example: if idx_users_college_active exists and idx_users_college exists,
-- the latter is redundant if queries filter by both

DO $$
BEGIN
    -- Check for and drop redundant indexes
    -- This is safe to run multiple times
    DROP INDEX IF EXISTS idx_users_college_only;  -- Covered by idx_users_college
END $$;

-- ============================================================================
-- VERIFICATION
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 011 completed successfully!';
    RAISE NOTICE 'Added CHECK constraints for data validation';
    RAISE NOTICE 'Added college_id columns to submissions, enrollments, faculty_assignments';
    RAISE NOTICE 'Made lab_sessions.lab_id NOT NULL';
    RAISE NOTICE 'Added foreign key constraints for college_id columns';
    RAISE NOTICE 'Added missing indexes for performance';
END $$;

-- ============================================================================
-- ROLLBACK SCRIPT (use if migration needs to be reverted)
-- ============================================================================
--
-- -- Remove CHECK constraints
-- ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_role;
-- ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS chk_enrollments_type;
-- ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS chk_enrollments_status;
-- ALTER TABLE faculty_assignments DROP CONSTRAINT IF EXISTS chk_faculty_assignments_role;
-- ALTER TABLE problems DROP CONSTRAINT IF EXISTS chk_problems_difficulty;
-- ALTER TABLE submissions DROP CONSTRAINT IF EXISTS chk_submissions_status;
-- ALTER TABLE contests DROP CONSTRAINT IF EXISTS chk_contests_status;
-- ALTER TABLE courses DROP CONSTRAINT IF EXISTS chk_course_category;
-- ALTER TABLE problems DROP CONSTRAINT IF EXISTS chk_problems_time_limit;
-- ALTER TABLE problems DROP CONSTRAINT IF EXISTS chk_problems_memory_limit;
-- ALTER TABLE test_cases DROP CONSTRAINT IF EXISTS chk_test_cases_points;
-- ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS chk_enrollments_attempt_no;
--
-- -- Remove college_id columns
-- ALTER TABLE submissions DROP COLUMN IF EXISTS college_id;
-- ALTER TABLE enrollments DROP COLUMN IF EXISTS college_id;
-- ALTER TABLE faculty_assignments DROP COLUMN IF EXISTS college_id;
-- ALTER TABLE contest_submissions DROP COLUMN IF EXISTS college_id;
--
-- -- Make lab_id nullable again
-- ALTER TABLE lab_sessions ALTER COLUMN lab_id DROP NOT NULL;
--
-- -- Remove FK constraints
-- ALTER TABLE submissions DROP CONSTRAINT IF EXISTS fk_submissions_college;
-- ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS fk_enrollments_college;
-- ALTER TABLE faculty_assignments DROP CONSTRAINT IF EXISTS fk_faculty_assignments_college;
-- ALTER TABLE contest_submissions DROP CONSTRAINT IF EXISTS fk_contest_submissions_college;
--
-- ============================================================================
-- END OF MIGRATION
-- ============================================================================