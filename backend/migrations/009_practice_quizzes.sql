-- Practice Quiz System (embedded in theory courses)
-- Separate from formal quiz system in 007_quiz_system.sql
-- Practice quizzes are for learning/self-assessment, not graded tests

-- =====================================================
-- Table: practice_quizzes
-- =====================================================
CREATE TABLE IF NOT EXISTS practice_quizzes (
    id BIGSERIAL PRIMARY KEY,
    theory_week_id BIGINT NOT NULL REFERENCES theory_weeks(id) ON DELETE CASCADE,
    title VARCHAR(300) NOT NULL,
    instructions TEXT,
    total_marks INT DEFAULT 0,
    passing_marks INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_practice_quizzes_week ON practice_quizzes(theory_week_id);

-- =====================================================
-- Table: practice_quiz_questions
-- =====================================================
CREATE TABLE IF NOT EXISTS practice_quiz_questions (
    id BIGSERIAL PRIMARY KEY,
    practice_quiz_id BIGINT NOT NULL REFERENCES practice_quizzes(id) ON DELETE CASCADE,
    question_type VARCHAR(20) NOT NULL,
    question_text TEXT NOT NULL,
    explanation TEXT,
    marks INT DEFAULT 1,
    difficulty VARCHAR(10) NOT NULL,
    order_index INT DEFAULT 0,
    source_module_id BIGINT REFERENCES theory_modules(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_practice_question_type CHECK (question_type IN ('mcq', 'multi_select', 'true_false', 'short_answer', 'fill_blank')),
    CONSTRAINT chk_practice_difficulty CHECK (difficulty IN ('easy', 'medium', 'hard'))
);

CREATE INDEX IF NOT EXISTS idx_practice_questions_quiz ON practice_quiz_questions(practice_quiz_id);
CREATE INDEX IF NOT EXISTS idx_practice_questions_source ON practice_quiz_questions(source_module_id);

-- =====================================================
-- Table: practice_quiz_options
-- =====================================================
CREATE TABLE IF NOT EXISTS practice_quiz_options (
    id BIGSERIAL PRIMARY KEY,
    question_id BIGINT NOT NULL REFERENCES practice_quiz_questions(id) ON DELETE CASCADE,
    option_text TEXT NOT NULL,
    is_correct BOOLEAN DEFAULT FALSE,
    order_index INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_practice_options_question ON practice_quiz_options(question_id);
CREATE INDEX IF NOT EXISTS idx_practice_options_correct ON practice_quiz_options(question_id, is_correct);

-- =====================================================
-- Table: generation_jobs
-- =====================================================
-- Tracks async AI generation jobs for lesson plan processing
CREATE TABLE IF NOT EXISTS generation_jobs (
    id BIGSERIAL PRIMARY KEY,
    admin_regdno VARCHAR(50) NOT NULL REFERENCES users(regdno) ON DELETE CASCADE,
    course_id BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    theory_id BIGINT REFERENCES theories(id) ON DELETE CASCADE,
    job_type VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    input_file_path VARCHAR(500),
    generated_data JSONB,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    CONSTRAINT chk_generation_status CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    CONSTRAINT chk_generation_job_type CHECK (job_type IN ('lesson_plan_parse', 'quiz_generate'))
);

CREATE INDEX IF NOT EXISTS idx_generation_jobs_admin ON generation_jobs(admin_regdno);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_course ON generation_jobs(course_id);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_theory ON generation_jobs(theory_id);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_status ON generation_jobs(status);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_job_type ON generation_jobs(job_type);

-- =====================================================
-- Comments for documentation
-- =====================================================
COMMENT ON TABLE practice_quizzes IS 'Practice quizzes embedded within theory weeks for student self-assessment';
COMMENT ON TABLE practice_quiz_questions IS 'Questions belonging to practice quizzes';
COMMENT ON TABLE practice_quiz_options IS 'MCQ/TF options for practice quiz questions';
COMMENT ON TABLE generation_jobs IS 'Async job tracking for AI-powered content generation';

COMMENT ON COLUMN practice_quizzes.theory_week_id IS 'Links quiz to a specific theory week';
COMMENT ON COLUMN practice_quiz_questions.source_module_id IS 'Theory module this question was generated from';
COMMENT ON COLUMN generation_jobs.job_type IS 'Type: lesson_plan_parse or quiz_generate';
COMMENT ON COLUMN generation_jobs.status IS 'Status: pending, processing, completed, failed';
COMMENT ON COLUMN generation_jobs.generated_data IS 'AI-generated content structure (JSONB)';
