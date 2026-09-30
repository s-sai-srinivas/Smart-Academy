-- Migration: Create theory_pdfs table for PDF uploads to theory courses
-- This table stores uploaded PDF files linked to theory courses and optionally modules

CREATE TABLE IF NOT EXISTS "theory_pdfs" (
    "id" BIGSERIAL PRIMARY KEY,
    "created_at" TIMESTAMPTZ,
    "updated_at" TIMESTAMPTZ,
    "deleted_at" TIMESTAMPTZ,
    "file_name" VARCHAR(500) NOT NULL,
    "display_name" VARCHAR(500),
    "file_path" VARCHAR(1000) NOT NULL,
    "file_size" BIGINT DEFAULT 0,
    "theory_id" BIGINT NOT NULL,
    "theory_module_id" BIGINT
);

-- Add indexes for common query patterns
CREATE INDEX IF NOT EXISTS "idx_theory_pdfs_theory_id" ON "theory_pdfs" ("theory_id");
CREATE INDEX IF NOT EXISTS "idx_theory_pdfs_theory_module_id" ON "theory_pdfs" ("theory_module_id");
CREATE INDEX IF NOT EXISTS "idx_theory_pdfs_deleted_at" ON "theory_pdfs" ("deleted_at");
