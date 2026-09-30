-- Remove unused contest fields that have been removed from frontend
-- These fields are no longer needed:
-- - is_public: All contests are college-specific, not public
-- - show_leaderboard: Leaderboard is always shown, no toggle needed
-- - freeze_duration_minutes: Not used, leaderboard freeze not implemented

ALTER TABLE contests
    DROP COLUMN IF EXISTS is_public,
    DROP COLUMN IF EXISTS show_leaderboard,
    DROP COLUMN IF EXISTS freeze_duration_minutes;