-- ============================================================================
-- Migration 012: Add Enrollment Unique Constraint
--
-- This migration adds a unique constraint on (student_regdno, course_offering_id)
-- to prevent duplicate enrollments.
-- ============================================================================

-- ============================================================================
-- STEP 1: Add unique constraint for enrollments
-- ============================================================================

-- First, remove any duplicate enrollments that may exist (keep the latest one)
DO $$
DECLARE
    dup_record RECORD;
BEGIN
    FOR dup_record IN
        SELECT student_regdno, course_offering_id, COUNT(*) as cnt
        FROM enrollments
        GROUP BY student_regdno, course_offering_id
        HAVING COUNT(*) > 1
    LOOP
        -- Delete all but the most recent enrollment for each duplicate
        DELETE FROM enrollments
        WHERE student_regdno = dup_record.student_regdno
          AND course_offering_id = dup_record.course_offering_id
          AND id NOT IN (
              SELECT id FROM enrollments
              WHERE student_regdno = dup_record.student_regdno
                AND course_offering_id = dup_record.course_offering_id
              ORDER BY enrolled_at DESC
              LIMIT 1
          );
    END LOOP;
END $$;

-- Add the unique constraint
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_enrollments_student_offering'
    ) THEN
        ALTER TABLE enrollments
        ADD CONSTRAINT uq_enrollments_student_offering
        UNIQUE (student_regdno, course_offering_id);
    END IF;
END $$;

-- ============================================================================
-- VERIFICATION
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 012 completed successfully!';
    RAISE NOTICE 'Added unique constraint on enrollments(student_regdno, course_offering_id)';
END $$;

-- ============================================================================
-- ROLLBACK SCRIPT (use if migration needs to be reverted)
-- ============================================================================
--
-- ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS uq_enrollments_student_offering;
--
-- ============================================================================
-- END OF MIGRATION
-- ============================================================================