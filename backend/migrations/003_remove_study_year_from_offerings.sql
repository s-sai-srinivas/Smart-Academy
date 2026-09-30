-- Migration: Remove study_year_index from course_offerings table
-- study_year_index is no longer used; courses are tied to academic_year + section only

-- Drop the index first (if it exists)
DROP INDEX IF EXISTS idx_offerings_study_year;

-- Drop the column
ALTER TABLE course_offerings DROP COLUMN IF EXISTS study_year_index;

-- Drop the NOT NULL constraint check (PostgreSQL will handle this with DROP COLUMN)
-- No further action needed
