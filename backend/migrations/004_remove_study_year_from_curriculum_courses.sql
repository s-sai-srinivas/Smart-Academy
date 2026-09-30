-- Migration: Remove study_year_index from curriculum_courses table
-- study_year_index is no longer used; courses are tied to academic_year + section only

BEGIN;

-- Drop the unique constraint that includes study_year_index
DROP INDEX IF EXISTS curriculum_courses_curriculum_id_course_id_study_year_index_key;

-- Drop the index on study_year_index
DROP INDEX IF EXISTS idx_curriculum_courses_year;

-- Remove the study_year_index column
ALTER TABLE curriculum_courses DROP COLUMN IF EXISTS study_year_index;

-- Recreate unique constraint without study_year_index
CREATE UNIQUE INDEX IF NOT EXISTS curriculum_courses_curriculum_id_course_id_key
ON curriculum_courses(curriculum_id, course_id);

COMMIT;

COMMENT ON TABLE curriculum_courses IS 'Maps courses to a curriculum without year dependency';
