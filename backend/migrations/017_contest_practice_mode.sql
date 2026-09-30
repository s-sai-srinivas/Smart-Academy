-- Add practice mode fields to contests table
-- This allows admins to enable contest problems for practice after the contest ends

ALTER TABLE contests
    ADD COLUMN IF NOT EXISTS practice_enabled boolean DEFAULT false,
    ADD COLUMN IF NOT EXISTS practice_start_time timestamp NULL,
    ADD COLUMN IF NOT EXISTS practice_end_time timestamp NULL;

-- Add is_practice flag to contest_submissions to distinguish practice submissions
-- Practice submissions don't count toward contest leaderboard/scoring
ALTER TABLE contest_submissions
    ADD COLUMN IF NOT EXISTS is_practice boolean DEFAULT false;

-- Add index for practice mode queries
CREATE INDEX IF NOT EXISTS idx_contests_practice_enabled ON contests(practice_enabled) WHERE practice_enabled = true;
CREATE INDEX IF NOT EXISTS idx_contest_submissions_practice ON contest_submissions(contest_id, is_practice);

-- Comment on columns for documentation
COMMENT ON COLUMN contests.practice_enabled IS 'Whether practice mode is enabled for this ended contest';
COMMENT ON COLUMN contests.practice_start_time IS 'When practice mode starts (usually set when admin enables it)';
COMMENT ON COLUMN contests.practice_end_time IS 'When practice mode ends (admin configurable)';
COMMENT ON COLUMN contest_submissions.is_practice IS 'Whether this submission was made during practice mode (not counted for scoring)';