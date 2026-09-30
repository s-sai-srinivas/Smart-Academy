-- Migration: Contest Generation System
-- Description: Add support for AI-powered contest problem generation
-- Date: 2026-03-12

-- =====================================================
-- STEP 1: Extend generation_jobs table
-- =====================================================

-- Add contest_id column to generation_jobs for linking to contests
ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS contest_id INTEGER REFERENCES contests(contest_id) ON DELETE SET NULL;

-- Create index for efficient contest-based queries
CREATE INDEX IF NOT EXISTS idx_generation_jobs_contest ON generation_jobs(contest_id);

-- Update job_type constraint to include 'contest_generate'
-- First drop the existing constraint
ALTER TABLE generation_jobs DROP CONSTRAINT IF EXISTS chk_generation_job_type;

-- Recreate constraint with new job type
ALTER TABLE generation_jobs ADD CONSTRAINT chk_generation_job_type
    CHECK (job_type IN ('lesson_plan_parse', 'quiz_generate', 'contest_generate'));

-- =====================================================
-- STEP 2: Contest Editorials Table (Optional - Phase 2)
-- =====================================================
-- Stores editorial hints generated after contest completion
-- Helps students learn from problems without revealing full solutions

CREATE TABLE IF NOT EXISTS contest_editorials (
    id BIGSERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL REFERENCES contests(contest_id) ON DELETE CASCADE,
    problem_id INTEGER NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
    hint_1 TEXT NOT NULL,
    hint_2 TEXT,
    hint_3 TEXT,
    core_idea TEXT NOT NULL,
    time_complexity VARCHAR(100),
    space_complexity VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- Unique constraint: one editorial per contest problem
CREATE UNIQUE INDEX IF NOT EXISTS uk_contest_problem_editorial
    ON contest_editorials(contest_id, problem_id);

-- Index for efficient lookup by contest
CREATE INDEX IF NOT EXISTS idx_contest_editorials_contest ON contest_editorials(contest_id);

-- Index for efficient lookup by problem
CREATE INDEX IF NOT EXISTS idx_contest_editorials_problem ON contest_editorials(problem_id);

-- =====================================================
-- STEP 3: Comments for documentation
-- =====================================================

COMMENT ON TABLE contest_editorials IS 'Editorial hints for contest problems - generated post-contest for student learning';
COMMENT ON COLUMN contest_editorials.hint_1 IS 'First gentle hint - most vague guidance';
COMMENT ON COLUMN contest_editorials.hint_2 IS 'Second hint - more specific guidance';
COMMENT ON COLUMN contest_editorials.hint_3 IS 'Third hint - almost reveals approach but no code';
COMMENT ON COLUMN contest_editorials.core_idea IS '2-3 sentence explanation of the key insight';
COMMENT ON COLUMN contest_editorials.time_complexity IS 'Expected time complexity (e.g., O(N log N))';
COMMENT ON COLUMN contest_editorials.space_complexity IS 'Expected space complexity (e.g., O(N))';

-- =====================================================
-- Migration Info
-- =====================================================
DO $$
BEGIN
    RAISE NOTICE 'Migration: Contest Generation System tables created successfully';
    RAISE NOTICE '  - generation_jobs: Extended with contest_id and new job_type';
    RAISE NOTICE '  - contest_editorials: Editorial hints for post-contest learning';
    RAISE NOTICE '';
    RAISE NOTICE 'AI Contest Generation Workflow:';
    RAISE NOTICE '  1. Faculty submits topics via ContestGenerationRequest';
    RAISE NOTICE '  2. AI generates 3x problems asynchronously (generation_jobs)';
    RAISE NOTICE '  3. Faculty reviews and approves best problems';
    RAISE NOTICE '  4. Approved problems saved to problems table';
    RAISE NOTICE '  5. Contest runs normally with approved problems';
    RAISE NOTICE '  6. Post-contest: AI generates editorial hints';
END $$;
