-- ============================================================================
-- Phase 1: Database Schema Refactoring
-- Migration: Lab & Theory Independent Structure
--
-- This script migrates from the old CourseType-based structure to the new
-- independent Lab and Theory entities linked to Course.
-- ============================================================================

-- ============================================================================
-- STEP 1: Create new tables for the new structure
-- ============================================================================

-- Create Labs table (1-to-1 with Course)
CREATE TABLE IF NOT EXISTS labs_v2 (
    lab_id SERIAL PRIMARY KEY,
    course_id BIGINT NOT NULL UNIQUE,
    lab_code VARCHAR(50) NOT NULL UNIQUE,
    lab_name VARCHAR(200) NOT NULL,
    description TEXT,
    created_by VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_labs_v2_course FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
);

-- Create Theories table (1-to-1 with Course)
CREATE TABLE IF NOT EXISTS theories_v2 (
    theory_id SERIAL PRIMARY KEY,
    course_id BIGINT NOT NULL UNIQUE,
    theory_code VARCHAR(50) NOT NULL UNIQUE,
    theory_name VARCHAR(200) NOT NULL,
    description TEXT,
    created_by VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_theories_v2_course FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
);

-- Create Lab Sessions table (belongs to Lab)
CREATE TABLE IF NOT EXISTS lab_sessions_v2 (
    session_id SERIAL PRIMARY KEY,
    lab_id BIGINT NOT NULL,
    section_id BIGINT,
    session_name VARCHAR(200) NOT NULL,
    session_order INT DEFAULT 0,
    topic_name VARCHAR(200) NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    max_attempts INT DEFAULT 0,
    created_by VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_lab_sessions_v2_lab FOREIGN KEY (lab_id) REFERENCES labs_v2(lab_id) ON DELETE CASCADE,
    CONSTRAINT fk_lab_sessions_v2_section FOREIGN KEY (section_id) REFERENCES sections(section_id)
);

-- Create Theory Weeks table (belongs to Theory)
CREATE TABLE IF NOT EXISTS theory_weeks (
    id SERIAL PRIMARY KEY,
    theory_id BIGINT NOT NULL,
    week_name VARCHAR(200) NOT NULL,
    week_order INT DEFAULT 0,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_theory_weeks_theory FOREIGN KEY (theory_id) REFERENCES theories(id) ON DELETE CASCADE
);

-- Create Theory Modules table (belongs to TheoryWeek)
CREATE TABLE IF NOT EXISTS theory_modules_v2 (
    module_id SERIAL PRIMARY KEY,
    theory_week_id BIGINT NOT NULL,
    module_name VARCHAR(200) NOT NULL,
    module_order INT DEFAULT 0,
    description TEXT,
    content TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_theory_modules_v2_week FOREIGN KEY (theory_week_id) REFERENCES theory_weeks(id) ON DELETE CASCADE
);

-- ============================================================================
-- STEP 2: Add new columns to existing tables
-- ============================================================================

-- Add LabSessionID to Problems table (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'problems' AND column_name = 'lab_session_id'
    ) THEN
        ALTER TABLE problems ADD COLUMN lab_session_id BIGINT;
        CREATE INDEX idx_problem_lab_session ON problems(lab_session_id);
        ALTER TABLE problems ADD CONSTRAINT fk_problems_lab_session
            FOREIGN KEY (lab_session_id) REFERENCES lab_sessions_v2(session_id) ON DELETE SET NULL;
    END IF;
END $$;

-- Add branches array column to courses (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'courses' AND column_name = 'branches'
    ) THEN
        ALTER TABLE courses ADD COLUMN branches TEXT[];
    END IF;
END $$;

-- ============================================================================
-- STEP 3: Migrate existing data
-- ============================================================================

-- Step 3.1: Create Labs from existing lab courses
INSERT INTO labs_v2 (course_id, lab_code, lab_name, description, created_by, is_active)
SELECT
    id,
    course_code,
    course_name,
    'Lab component for ' || course_name,
    created_by,
    is_active
FROM courses
WHERE course_type = 'lab'
ON CONFLICT (course_id) DO NOTHING;

-- Step 3.2: Create Theories from existing theory courses
INSERT INTO theories_v2 (course_id, theory_code, theory_name, description, created_by, is_active)
SELECT
    id,
    course_code,
    course_name,
    'Theory component for ' || course_name,
    created_by,
    is_active
FROM courses
WHERE course_type = 'theory'
ON CONFLICT (course_id) DO NOTHING;

-- Step 3.3: Migrate Lab Sessions to new structure
-- Note: This maps existing lab_sessions to the new labs_v2 table
INSERT INTO lab_sessions_v2 (
    lab_id, section_id, session_name, session_order, topic_name,
    start_time, end_time, max_attempts, created_by
)
SELECT
    l2.lab_id,
    ls.section_id,
    ls.title,
    0,
    COALESCE(ls.topic, 'General Lab'),
    ls.start_time,
    ls.end_time,
    ls.max_attempts,
    ls.created_by
FROM lab_sessions ls
INNER JOIN labs l ON ls.lab_id = l.lab_id
INNER JOIN labs_v2 l2 ON l2.course_id = ls.course_id
ON CONFLICT DO NOTHING;

-- Step 3.4: Migrate Theory Modules to new structure
-- First, group by week_number and create TheoryWeek entries
INSERT INTO theory_weeks (theory_id, week_name, week_order, description)
SELECT DISTINCT
    t2.theory_id,
    'Week ' || tm.week_number::text,
    tm.week_number,
    'Week ' || tm.week_number::text || ' content'
FROM theory_modules tm
INNER JOIN theories t ON tm.course_id = t.theory_id
INNER JOIN theories_v2 t2 ON t2.course_id = t.theory_id
WHERE NOT EXISTS (
    SELECT 1 FROM theory_weeks tw
    WHERE tw.theory_id = t2.theory_id AND tw.week_order = tm.week_number
)
ON CONFLICT DO NOTHING;

-- Then migrate the actual modules
INSERT INTO theory_modules_v2 (
    theory_week_id, module_name, module_order, description, content
)
SELECT
    tw.id,
    tm.title,
    tm.order_index,
    tm.description,
    tm.content
FROM theory_modules tm
INNER JOIN theories t ON tm.course_id = t.theory_id
INNER JOIN theories_v2 t2 ON t2.course_id = t.theory_id
INNER JOIN theory_weeks tw ON tw.theory_id = t2.theory_id AND tw.week_order = tm.week_number
ON CONFLICT DO NOTHING;

-- Step 3.5: Link Problems to Lab Sessions (if they were linked to LabTopic)
-- This is a placeholder - actual logic depends on how topics map to sessions
-- UPDATE problems
-- SET lab_session_id = (
--     SELECT session_id FROM lab_sessions_v2
--     WHERE topic_name IN (SELECT topic_name FROM lab_topics WHERE ...)
-- )
-- WHERE id IN (...);

-- ============================================================================
-- STEP 4: Create indexes for performance
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_labs_v2_course_id ON labs_v2(course_id);
CREATE INDEX IF NOT EXISTS idx_theories_v2_course_id ON theories_v2(course_id);
CREATE INDEX IF NOT EXISTS idx_lab_sessions_v2_lab_id ON lab_sessions_v2(lab_id);
CREATE INDEX IF NOT EXISTS idx_theory_weeks_theory_id ON theory_weeks(theory_id);
CREATE INDEX IF NOT EXISTS idx_theory_modules_v2_week_id ON theory_modules_v2(theory_week_id);

-- ============================================================================
-- STEP 5: Add comments for documentation
-- ============================================================================

COMMENT ON TABLE labs_v2 IS 'Lab subjects linked 1-to-1 with Course';
COMMENT ON TABLE theories_v2 IS 'Theory subjects linked 1-to-1 with Course';
COMMENT ON TABLE lab_sessions_v2 IS 'Timed lab sessions belonging to a Lab';
COMMENT ON TABLE theory_weeks IS 'Weekly structure for Theory content';
COMMENT ON TABLE theory_modules_v2 IS 'Modular content within Theory weeks';

COMMENT ON COLUMN lab_sessions_v2.topic_name IS 'Topic name as a field (not a separate table)';
COMMENT ON COLUMN courses.branches IS 'Array of branch codes for multi-branch support';

-- ============================================================================
-- ROLLBACK SCRIPT (use if migration needs to be reverted)
-- ============================================================================

-- DROP TABLE IF EXISTS theory_modules_v2 CASCADE;
-- DROP TABLE IF EXISTS theory_weeks CASCADE;
-- DROP TABLE IF EXISTS lab_sessions_v2 CASCADE;
-- DROP TABLE IF EXISTS theories_v2 CASCADE;
-- DROP TABLE IF EXISTS labs_v2 CASCADE;

-- ALTER TABLE problems DROP COLUMN IF EXISTS lab_session_id CASCADE;
-- ALTER TABLE courses DROP COLUMN IF EXISTS branches CASCADE;
