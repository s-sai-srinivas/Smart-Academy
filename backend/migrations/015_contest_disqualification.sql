-- Add contest disqualification tracking
ALTER TABLE contest_participants
    ADD COLUMN IF NOT EXISTS esc_violations integer DEFAULT 0,
    ADD COLUMN IF NOT EXISTS disqualified boolean DEFAULT false,
    ADD COLUMN IF NOT EXISTS disqualified_at timestamp NULL,
    ADD COLUMN IF NOT EXISTS disqualification_reason text NULL;

-- Optional: ensure esc_violations is never negative
ALTER TABLE contest_participants
    ADD CONSTRAINT contest_participants_esc_violations_non_negative
    CHECK (esc_violations >= 0);
