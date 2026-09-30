-- Migration: Change college_id from integer (serial) to varchar across all tables
-- This allows super_admin to specify a custom college ID string when creating colleges.
--
-- Run inside a transaction:
--   psql -d your_db -f 005_college_id_to_string.sql

BEGIN;

-- 1. Drop all foreign key constraints referencing colleges.college_id
ALTER TABLE branches DROP CONSTRAINT IF EXISTS fk_branches_college;
ALTER TABLE branches DROP CONSTRAINT IF EXISTS branches_college_id_fkey;
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_college;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_college_id_fkey;
ALTER TABLE contests DROP CONSTRAINT IF EXISTS fk_contests_college;
ALTER TABLE contests DROP CONSTRAINT IF EXISTS contests_college_id_fkey;
ALTER TABLE problems DROP CONSTRAINT IF EXISTS fk_problems_college;
ALTER TABLE problems DROP CONSTRAINT IF EXISTS problems_college_id_fkey;

-- Also drop any GORM-generated constraints (naming varies)
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (
        SELECT tc.constraint_name, tc.table_name
        FROM information_schema.table_constraints tc
        JOIN information_schema.constraint_column_usage ccu
          ON tc.constraint_name = ccu.constraint_name
        WHERE ccu.table_name = 'colleges'
          AND ccu.column_name = 'college_id'
          AND tc.constraint_type = 'FOREIGN KEY'
    ) LOOP
        EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I', r.table_name, r.constraint_name);
    END LOOP;
END $$;

-- 2. Drop the primary key on colleges so we can alter the column type
ALTER TABLE colleges DROP CONSTRAINT IF EXISTS colleges_pkey;

-- 3. Drop the default (auto-increment sequence) on college_id
ALTER TABLE colleges ALTER COLUMN college_id DROP DEFAULT;

-- 4. Convert college_id columns to varchar(50)
ALTER TABLE colleges ALTER COLUMN college_id TYPE varchar(50) USING college_id::varchar(50);
ALTER TABLE branches ALTER COLUMN college_id TYPE varchar(50) USING college_id::varchar(50);
ALTER TABLE users ALTER COLUMN college_id TYPE varchar(50) USING college_id::varchar(50);
ALTER TABLE contests ALTER COLUMN college_id TYPE varchar(50) USING college_id::varchar(50);
ALTER TABLE problems ALTER COLUMN college_id TYPE varchar(50) USING college_id::varchar(50);

-- 5. Re-add primary key on colleges
ALTER TABLE colleges ADD PRIMARY KEY (college_id);

-- 6. Re-add foreign key constraints
ALTER TABLE branches
    ADD CONSTRAINT fk_branches_college
    FOREIGN KEY (college_id) REFERENCES colleges(college_id) ON DELETE CASCADE;

ALTER TABLE users
    ADD CONSTRAINT fk_users_college
    FOREIGN KEY (college_id) REFERENCES colleges(college_id) ON DELETE SET NULL;

ALTER TABLE contests
    ADD CONSTRAINT fk_contests_college
    FOREIGN KEY (college_id) REFERENCES colleges(college_id) ON DELETE SET NULL;

ALTER TABLE problems
    ADD CONSTRAINT fk_problems_college
    FOREIGN KEY (college_id) REFERENCES colleges(college_id) ON DELETE SET NULL;

-- 7. Drop the old sequence if it exists (was used for autoIncrement)
DROP SEQUENCE IF EXISTS colleges_college_id_seq;

COMMIT;
