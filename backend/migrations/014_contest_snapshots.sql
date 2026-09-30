-- Contest snapshot tables for post-contest visibility

CREATE TABLE IF NOT EXISTS contest_leaderboard_snapshots (
    id SERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL UNIQUE REFERENCES contests(contest_id) ON DELETE CASCADE,
    data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS contest_non_participants_snapshots (
    id SERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL UNIQUE REFERENCES contests(contest_id) ON DELETE CASCADE,
    data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
