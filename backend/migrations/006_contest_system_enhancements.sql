-- Migration: Contest System with College-Based Access Control
-- Description: Add batch/branch targeting, simplified freeze logic, and proper submission tracking

-- =====================================================
-- CONTESTS TABLE - Add targeting and simplified freeze
-- =====================================================

-- Add new columns to contests table
ALTER TABLE contests ADD COLUMN IF NOT EXISTS target_cohort INTEGER;
ALTER TABLE contests ADD COLUMN IF NOT EXISTS target_branch_id INTEGER;
ALTER TABLE contests ADD COLUMN IF NOT EXISTS freeze_duration_minutes INTEGER DEFAULT 0;

-- Remove old complicated freeze columns if they exist
ALTER TABLE contests DROP COLUMN IF EXISTS freeze_before;
ALTER TABLE contests DROP COLUMN IF EXISTS leaderboard_frozen_at;

-- Add indexes for efficient college-based queries
CREATE INDEX IF NOT EXISTS idx_contests_college_start ON contests(college_id, start_time);
CREATE INDEX IF NOT EXISTS idx_contests_college_end ON contests(college_id, end_time);

-- =====================================================
-- CONTEST_PROBLEMS TABLE - Mandatory for contests
-- =====================================================

-- This table maps problems to contests with points and ordering
-- Without this, you cannot assign points per contest or control problem order
CREATE TABLE IF NOT EXISTS contest_problems (
    id BIGSERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL,
    problem_id INTEGER NOT NULL,
    points INTEGER NOT NULL DEFAULT 100,
    problem_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (contest_id) REFERENCES contests(contest_id) ON DELETE CASCADE,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    UNIQUE (contest_id, problem_id)
);

CREATE INDEX IF NOT EXISTS idx_contest_problems_contest ON contest_problems(contest_id);
CREATE INDEX IF NOT EXISTS idx_contest_problems_problem ON contest_problems(problem_id);

-- =====================================================
-- CONTEST_SUBMISSIONS TABLE - With proper attempt tracking
-- =====================================================

-- Drop existing table if it needs to be recreated
DROP TABLE IF EXISTS contest_submissions;

CREATE TABLE contest_submissions (
    id BIGSERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL,
    problem_id INTEGER NOT NULL,
    user_regd_no VARCHAR(50) NOT NULL,
    college_id VARCHAR(50),

    -- Submission details
    language_id INTEGER NOT NULL,
    source_code TEXT NOT NULL,
    submitted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    evaluated_at TIMESTAMP NULL,

    -- Results
    status VARCHAR(50) NOT NULL,                  -- 'completed', 'evaluated', 'pending', 'running'
    passed BOOLEAN DEFAULT FALSE,                 -- TRUE if all test cases passed
    score INTEGER DEFAULT 0,                      -- Points earned for this submission
    max_score INTEGER DEFAULT 0,                  -- Maximum possible points
    execution_time DOUBLE PRECISION,              -- In milliseconds
    memory_used INTEGER,                          -- In KB

    -- Attempt tracking for leaderboard (FIRST AC counts)
    attempt_number INTEGER NOT NULL DEFAULT 1,    -- Which attempt is this (1, 2, 3...)
    is_final BOOLEAN DEFAULT FALSE,               -- TRUE = this is the final accepted submission
    penalty_time INTEGER DEFAULT 0,               -- Penalty minutes (for future use)

    FOREIGN KEY (contest_id) REFERENCES contests(contest_id) ON DELETE CASCADE,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    FOREIGN KEY (user_regd_no) REFERENCES users(regdno) ON DELETE CASCADE,
    FOREIGN KEY (college_id) REFERENCES colleges(college_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_contest_submissions_contest_user ON contest_submissions(contest_id, user_regd_no);
CREATE INDEX IF NOT EXISTS idx_contest_submissions_contest_problem ON contest_submissions(contest_id, problem_id);
CREATE INDEX IF NOT EXISTS idx_contest_submissions_user_problem ON contest_submissions(user_regd_no, problem_id);
CREATE INDEX IF NOT EXISTS idx_contest_submissions_final ON contest_submissions(is_final);
CREATE INDEX IF NOT EXISTS idx_contest_submissions_college ON contest_submissions(college_id);

-- =====================================================
-- CONTEST_PROBLEM_TEST_CASES - For auto-scoring
-- =====================================================

CREATE TABLE IF NOT EXISTS contest_problem_test_cases (
    id BIGSERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL,
    problem_id INTEGER NOT NULL,
    test_case_id INTEGER NOT NULL,
    points_weight DOUBLE PRECISION DEFAULT 1.0,
    FOREIGN KEY (contest_id) REFERENCES contests(contest_id) ON DELETE CASCADE,
    FOREIGN KEY (problem_id) REFERENCES problems(id) ON DELETE CASCADE,
    FOREIGN KEY (test_case_id) REFERENCES test_cases(id) ON DELETE CASCADE,
    UNIQUE (contest_id, problem_id, test_case_id)
);

-- =====================================================
-- CONTEST_PARTICIPANTS - Add eligibility snapshot
-- =====================================================

-- Add eligibility snapshot to track user's eligibility at registration time
ALTER TABLE contest_participants ADD COLUMN IF NOT EXISTS eligibility_snapshot TEXT;

-- =====================================================
-- CONTESTS - Add updated_at for tracking changes
-- =====================================================

ALTER TABLE contests ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Create a trigger to update updated_at automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop the trigger if it exists and recreate
DROP TRIGGER IF EXISTS update_contests_updated_at ON contests;
CREATE TRIGGER update_contests_updated_at
    BEFORE UPDATE ON contests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();