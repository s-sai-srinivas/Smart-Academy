-- ============================================================================
-- Migration 013: Contest First AC Constraint
--
-- This migration adds a partial unique index to ensure only ONE accepted
-- submission per user/problem/contest combination. This is a defense-in-depth
-- measure to prevent race conditions in contest scoring.
--
-- Context: Without this constraint, concurrent submissions could both pass
-- the "previous success" check and both award points, resulting in duplicate
-- scores and compromised leaderboard integrity.
-- ============================================================================

-- ============================================================================
-- STEP 1: Create partial unique index for first accepted submission
-- ============================================================================

-- This index ensures that for any (contest_id, problem_id, user_regd_no) tuple,
-- there can be at most ONE row where passed = true.
-- The WHERE clause makes this a "partial" index that only applies to accepted submissions.
DO $$
BEGIN
    -- Check if index already exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'uq_contest_submission_first_ac'
    ) THEN
        -- Create the partial unique index
        -- Using CONCURRENTLY to avoid locking the table during creation
        -- Note: CONCURRENTLY cannot be used inside a transaction block
        CREATE UNIQUE INDEX CONCURRENTLY uq_contest_submission_first_ac
        ON contest_submissions (contest_id, problem_id, user_regd_no)
        WHERE passed = true;
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        -- If CONCURRENTLY fails (e.g., in transaction context), try without it
        IF NOT EXISTS (
            SELECT 1 FROM pg_indexes
            WHERE indexname = 'uq_contest_submission_first_ac'
        ) THEN
            CREATE UNIQUE INDEX uq_contest_submission_first_ac
            ON contest_submissions (contest_id, problem_id, user_regd_no)
            WHERE passed = true;
        END IF;
END $$;

-- ============================================================================
-- STEP 2: Clean up any existing duplicates (if any)
-- ============================================================================

-- This query identifies and removes duplicate accepted submissions, keeping
-- only the earliest one (by submitted_at or id)

DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    -- Check for duplicates
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT contest_id, problem_id, user_regd_no, COUNT(*) as cnt
        FROM contest_submissions
        WHERE passed = true
        GROUP BY contest_id, problem_id, user_regd_no
        HAVING COUNT(*) > 1
    ) duplicates;

    IF dup_count > 0 THEN
        RAISE NOTICE 'Found % duplicate accepted submissions. Cleaning up...', dup_count;

        -- Delete duplicates, keeping the earliest submission
        DELETE FROM contest_submissions
        WHERE id IN (
            SELECT cs.id
            FROM contest_submissions cs
            INNER JOIN (
                SELECT contest_id, problem_id, user_regd_no,
                       MIN(id) as keep_id
                FROM contest_submissions
                WHERE passed = true
                GROUP BY contest_id, problem_id, user_regd_no
                HAVING COUNT(*) > 1
            ) keep ON cs.contest_id = keep.contest_id
                  AND cs.problem_id = keep.problem_id
                  AND cs.user_regd_no = keep.user_regd_no
            WHERE cs.passed = true
              AND cs.id != keep.keep_id
        );

        RAISE NOTICE 'Duplicate cleanup completed.';
    ELSE
        RAISE NOTICE 'No duplicate accepted submissions found.';
    END IF;
END $$;

-- ============================================================================
-- VERIFICATION
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 013 completed successfully!';
    RAISE NOTICE 'Added partial unique index: uq_contest_submission_first_ac';
    RAISE NOTICE 'This ensures only one accepted submission per user/problem/contest';
END $$;

-- ============================================================================
-- ROLLBACK SCRIPT (use if migration needs to be reverted)
-- ============================================================================
--
-- DROP INDEX IF EXISTS uq_contest_submission_first_ac;
--
-- ============================================================================
-- END OF MIGRATION
-- ============================================================================