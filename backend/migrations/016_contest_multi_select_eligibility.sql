-- Migration: Contest Multi-Select Eligibility
-- Description: Support multiple batches and branches for contest eligibility
-- Date: 2026-04-12

-- =====================================================
-- STEP 1: Create junction tables for contest eligibility
-- =====================================================

-- Table for storing multiple target batches/cohorts
CREATE TABLE IF NOT EXISTS contest_target_cohorts (
    id BIGSERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL REFERENCES contests(contest_id) ON DELETE CASCADE,
    cohort_year INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (contest_id, cohort_year)
);

CREATE INDEX IF NOT EXISTS idx_contest_target_cohorts_contest ON contest_target_cohorts(contest_id);
CREATE INDEX IF NOT EXISTS idx_contest_target_cohorts_year ON contest_target_cohorts(cohort_year);

-- Table for storing multiple target branches
CREATE TABLE IF NOT EXISTS contest_target_branches (
    id BIGSERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL REFERENCES contests(contest_id) ON DELETE CASCADE,
    branch_id INTEGER NOT NULL REFERENCES branches(branch_id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (contest_id, branch_id)
);

CREATE INDEX IF NOT EXISTS idx_contest_target_branches_contest ON contest_target_branches(contest_id);
CREATE INDEX IF NOT EXISTS idx_contest_target_branches_branch ON contest_target_branches(branch_id);

-- =====================================================
-- STEP 2: Migrate existing single values to junction tables
-- =====================================================

-- Migrate existing target_cohort values to new junction table
INSERT INTO contest_target_cohorts (contest_id, cohort_year)
SELECT contest_id, target_cohort
FROM contests
WHERE target_cohort IS NOT NULL
ON CONFLICT (contest_id, cohort_year) DO NOTHING;

-- Migrate existing target_branch_id values to new junction table
INSERT INTO contest_target_branches (contest_id, branch_id)
SELECT contest_id, target_branch_id
FROM contests
WHERE target_branch_id IS NOT NULL
ON CONFLICT (contest_id, branch_id) DO NOTHING;

-- =====================================================
-- STEP 3: Add comments for documentation
-- =====================================================

COMMENT ON TABLE contest_target_cohorts IS 'Junction table storing multiple target batches/cohorts for contest eligibility';
COMMENT ON TABLE contest_target_branches IS 'Junction table storing multiple target branches for contest eligibility';

-- =====================================================
-- Migration Info
-- =====================================================
DO $$
BEGIN
    RAISE NOTICE 'Migration: Contest Multi-Select Eligibility created successfully';
    RAISE NOTICE '  - contest_target_cohorts: Junction table for multiple batches';
    RAISE NOTICE '  - contest_target_branches: Junction table for multiple branches';
    RAISE NOTICE '  - Existing single values migrated to junction tables';
END $$;