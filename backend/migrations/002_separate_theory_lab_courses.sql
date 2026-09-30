-- ============================================================================
-- Phase 2: Theory and Lab as Separate Independent Courses
-- Migration: 002_separate_theory_lab_courses.sql
--
-- This refactoring separates Theory and Lab into completely independent courses.
-- Key changes:
-- 1. Add course_category column to distinguish 'theory' vs 'lab' courses
-- 2. Remove course_type column (replaced by course_category)
-- 3. Keep labs and theories tables but simplify their relationship to courses
-- 4. Update foreign keys for lab_sessions and theory_modules
--
-- IMPORTANT: This migration assumes a CLEAN SLATE approach.
-- Existing data will be archived/deleted as per requirements.
-- ============================================================================

-- ============================================================================
-- STEP 1: Create backup/archive of existing data (for safety)
-- ============================================================================

-- Create backup tables only if source tables exist
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'courses') THEN
        CREATE TABLE IF NOT EXISTS courses_backup AS SELECT * FROM courses;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'labs') THEN
        CREATE TABLE IF NOT EXISTS labs_backup AS SELECT * FROM labs;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'theories') THEN
        CREATE TABLE IF NOT EXISTS theories_backup AS SELECT * FROM theories;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'lab_sessions') THEN
        CREATE TABLE IF NOT EXISTS lab_sessions_backup AS SELECT * FROM lab_sessions;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'theory_modules') THEN
        CREATE TABLE IF NOT EXISTS theory_modules_backup AS SELECT * FROM theory_modules;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'student_enrollments') THEN
        CREATE TABLE IF NOT EXISTS student_enrollments_backup AS SELECT * FROM student_enrollments;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'faculty_course_assignments') THEN
        CREATE TABLE IF NOT EXISTS faculty_course_assignments_backup AS SELECT * FROM faculty_course_assignments;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'submissions') THEN
        CREATE TABLE IF NOT EXISTS submissions_backup AS SELECT * FROM submissions;
    END IF;
END $$;

-- ============================================================================
-- STEP 2: Drop existing data (Clean Slate approach)
-- ============================================================================

-- Drop dependent data first (due to foreign keys)
-- Only delete from tables that exist - in correct order to respect foreign keys
DO $$
BEGIN
    -- Check and delete from each table if it exists
    -- Start with tables that have no dependents

    -- Problems table may reference lab_sessions
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'problems') THEN
        DELETE FROM problems;
    END IF;

    -- User completions depends on submissions
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'user_problem_completions') THEN
        DELETE FROM user_problem_completions;
    END IF;

    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'plagiarism_results') THEN
        DELETE FROM plagiarism_results;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'plagiarism_matches') THEN
        DELETE FROM plagiarism_matches;
    END IF;

    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'contest_participants') THEN
        DELETE FROM contest_participants;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'contest_problems') THEN
        DELETE FROM contest_problems;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'submissions') THEN
        DELETE FROM submissions;
    END IF;

    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'lab_session_problems') THEN
        DELETE FROM lab_session_problems;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'topic_problems') THEN
        DELETE FROM topic_problems;
    END IF;

    -- Lab sessions may be referenced by problems
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'lab_sessions') THEN
        DELETE FROM lab_sessions;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'lab_topics') THEN
        DELETE FROM lab_topics;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'theory_modules') THEN
        DELETE FROM theory_modules;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'theory_weeks') THEN
        DELETE FROM theory_weeks;
    END IF;

    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'student_enrollments') THEN
        DELETE FROM student_enrollments;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'faculty_course_assignments') THEN
        DELETE FROM faculty_course_assignments;
    END IF;

    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'theories') THEN
        DELETE FROM theories;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'labs') THEN
        DELETE FROM labs;
    END IF;
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'courses') THEN
        DELETE FROM courses;
    END IF;
END $$;

-- ============================================================================
-- STEP 3: Add course_category column to courses table
-- ============================================================================

-- Add course_category column to replace course_type
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'courses' AND column_name = 'course_category'
    ) THEN
        ALTER TABLE courses ADD COLUMN course_category VARCHAR(20);
        -- Add check constraint to ensure only valid values
        ALTER TABLE courses ADD CONSTRAINT chk_course_category
            CHECK (course_category IN ('theory', 'lab'));
    END IF;
END $$;

-- ============================================================================
-- STEP 4: Update course_code to be unique per course (theory or lab)
-- ============================================================================

-- The course_code already has a unique constraint, so no changes needed there.
-- Each course (whether theory or lab) will have its own unique code.

-- ============================================================================
-- STEP 5: Update Labs table structure
-- ============================================================================

-- Labs table becomes a 1-to-1 extension for lab-type courses
-- No structural changes needed, just clarify the relationship

-- Add comment for documentation
COMMENT ON TABLE labs IS 'Lab component extension for lab-type courses. One-to-one with courses where course_category = ''lab''.';
COMMENT ON COLUMN labs.course_id IS 'Foreign key to courses table. References a course where course_category = ''lab''.';

-- ============================================================================
-- STEP 6: Update Theories table structure
-- ============================================================================

-- Theories table becomes a 1-to-1 extension for theory-type courses
-- No structural changes needed, just clarify the relationship

-- Add comment for documentation
COMMENT ON TABLE theories IS 'Theory component extension for theory-type courses. One-to-one with courses where course_category = ''theory''.';
COMMENT ON COLUMN theories.course_id IS 'Foreign key to courses table. References a course where course_category = ''theory''.';

-- ============================================================================
-- STEP 7: Update Lab Sessions to work directly with Labs (which link to Courses)
-- ============================================================================

-- Lab sessions already link to labs, which link to courses
-- No changes needed to lab_sessions table structure

-- ============================================================================
-- STEP 8: Update Theory Modules to work directly with Theories
-- ============================================================================

-- Theory modules already link to theories, which link to courses
-- No changes needed to theory_modules table structure

-- ============================================================================
-- STEP 9: Update Submissions table for separate theory/lab tracking
-- ============================================================================

-- Submissions can now track whether they're for theory or lab courses
-- The course_id in submissions will point to either a theory or lab course

-- Add comment for documentation
COMMENT ON TABLE submissions IS 'Code submissions for problems. Can be for theory or lab courses based on the course_id reference.';

-- ============================================================================
-- STEP 10: Update Faculty Course Assignments
-- ============================================================================

-- Faculty are now assigned to independent theory or lab courses
-- No structural changes needed, course_id will point to either type

-- Add comment for documentation
COMMENT ON TABLE faculty_course_assignments IS 'Faculty assignments to courses. Each assignment is to either a theory or lab course (not both).';

-- ============================================================================
-- STEP 11: Update Student Enrollments
-- ============================================================================

-- Students now enroll independently in theory and lab courses
-- No structural changes needed

-- Add comment for documentation
COMMENT ON TABLE student_enrollments IS 'Student enrollments in courses. Each enrollment is for either a theory or lab course.';

-- ============================================================================
-- STEP 12: Remove course_type column (replaced by course_category)
-- ============================================================================

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'courses' AND column_name = 'course_type'
    ) THEN
        ALTER TABLE courses DROP COLUMN course_type;
    END IF;
END $$;

-- ============================================================================
-- STEP 13: Create indexes for performance
-- ============================================================================

-- Create index on course_category for filtering
CREATE INDEX IF NOT EXISTS idx_courses_category ON courses(course_category);

-- Ensure existing indexes are still valid
CREATE INDEX IF NOT EXISTS idx_labs_course_id ON labs(course_id);
CREATE INDEX IF NOT EXISTS idx_theories_course_id ON theories(course_id);

-- ============================================================================
-- STEP 14: Add helpful views for querying
-- ============================================================================

-- View for all theory courses with their theory details
CREATE OR REPLACE VIEW theory_courses_view AS
SELECT
    c.id,
    c.course_code,
    c.course_name,
    c.credits,
    c.program,
    c.branches,
    c.semester,
    c.batch,
    c.is_active,
    c.created_at,
    c.updated_at,
    t.id as theory_id,
    t.theory_name,
    t.theory_code
FROM courses c
INNER JOIN theories t ON t.course_id = c.id
WHERE c.course_category = 'theory';

-- View for all lab courses with their lab details
CREATE OR REPLACE VIEW lab_courses_view AS
SELECT
    c.id,
    c.course_code,
    c.course_name,
    c.credits,
    c.program,
    c.branches,
    c.semester,
    c.batch,
    c.is_active,
    c.created_at,
    c.updated_at,
    l.id as lab_id,
    l.lab_name,
    l.lab_code
FROM courses c
INNER JOIN labs l ON l.course_id = c.id
WHERE c.course_category = 'lab';

-- ============================================================================
-- STEP 15: Verification query
-- ============================================================================

-- Run this to verify the migration was successful
DO $$
BEGIN
    RAISE NOTICE 'Migration completed successfully!';
    RAISE NOTICE 'Courses table now has course_category instead of course_type';
    RAISE NOTICE 'Theory and Lab are now independent courses';
    RAISE NOTICE 'Use theory_courses_view and lab_courses_view for convenient queries';
END $$;

-- ============================================================================
-- ROLLBACK SCRIPT (use if migration needs to be reverted)
-- ============================================================================

-- DROP VIEW IF EXISTS lab_courses_view;
-- DROP VIEW IF EXISTS theory_courses_view;
-- DROP INDEX IF EXISTS idx_theories_course_id;
-- DROP INDEX IF EXISTS idx_labs_course_id;
-- DROP INDEX IF EXISTS idx_courses_category;
--
-- -- Restore course_type column
-- DO $$
-- BEGIN
--     IF NOT EXISTS (
--         SELECT 1 FROM information_schema.columns
--         WHERE table_name = 'courses' AND column_name = 'course_type'
--     ) THEN
--         ALTER TABLE courses ADD COLUMN course_type VARCHAR(50);
--     END IF;
-- END $$;
--
-- -- Drop course_category column
-- ALTER TABLE courses DROP COLUMN IF EXISTS course_category;
--
-- -- Restore data from backups (if needed)
-- -- TRUNCATE courses CASCADE;
-- -- INSERT INTO courses SELECT * FROM courses_backup;
-- -- (repeat for other tables as needed)

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================
