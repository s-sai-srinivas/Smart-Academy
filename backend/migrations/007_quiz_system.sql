-- =====================================================
-- Migration 007: Quiz System
-- =====================================================
-- Implements AI-powered quiz generation and management
-- Scoped to: College → Branch → Section (CohortYear) → CourseOffering
-- =====================================================

-- Drop existing tables if they exist (for fresh migrations)
DROP TABLE IF EXISTS quiz_answers CASCADE;
DROP TABLE IF EXISTS quiz_attempts CASCADE;
DROP TABLE IF EXISTS quiz_question_options CASCADE;
DROP TABLE IF EXISTS quiz_questions CASCADE;
DROP TABLE IF EXISTS quiz_theory_sources CASCADE;
DROP TABLE IF EXISTS quizzes CASCADE;

-- =====================================================
-- Table: quizzes - Quiz metadata, scoped to a CourseOffering
-- =====================================================
CREATE TABLE quizzes (
    id                  BIGSERIAL PRIMARY KEY,
    course_offering_id  BIGINT NOT NULL,
    title               VARCHAR(300) NOT NULL,
    description         TEXT,
    quiz_type           VARCHAR(30) NOT NULL DEFAULT 'ai_generated',
    status              VARCHAR(20) NOT NULL DEFAULT 'draft',
    time_limit_minutes  INTEGER NOT NULL DEFAULT 0,
    total_marks         INTEGER NOT NULL DEFAULT 0,
    passing_marks       INTEGER NOT NULL DEFAULT 0,
    max_attempts        INTEGER NOT NULL DEFAULT 1,
    shuffle_questions   BOOLEAN NOT NULL DEFAULT FALSE,
    shuffle_options     BOOLEAN NOT NULL DEFAULT FALSE,
    show_results_after  VARCHAR(20) NOT NULL DEFAULT 'immediately',
    scheduled_start     TIMESTAMP WITH TIME ZONE,
    scheduled_end       TIMESTAMP WITH TIME ZONE,
    created_by          VARCHAR(50) NOT NULL,
    ai_prompt           TEXT,
    ai_model_used       VARCHAR(50),
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT chk_quiz_type CHECK (quiz_type IN ('ai_generated', 'manual', 'hybrid')),
    CONSTRAINT chk_quiz_status CHECK (status IN ('draft', 'published', 'closed', 'archived')),
    CONSTRAINT chk_show_results_after CHECK (show_results_after IN ('immediately', 'after_close', 'never')),
    CONSTRAINT chk_time_limit CHECK (time_limit_minutes >= 0),
    CONSTRAINT chk_total_marks CHECK (total_marks >= 0),
    CONSTRAINT chk_passing_marks CHECK (passing_marks >= 0),
    CONSTRAINT chk_passing_marks_value CHECK (passing_marks <= total_marks OR total_marks = 0),
    CONSTRAINT chk_max_attempts CHECK (max_attempts >= 1),
    CONSTRAINT chk_scheduled_dates CHECK (scheduled_end IS NULL OR scheduled_start IS NULL OR scheduled_end > scheduled_start)
);

-- Indexes for quizzes
CREATE INDEX idx_quizzes_course_offering ON quizzes(course_offering_id);
CREATE INDEX idx_quizzes_created_by ON quizzes(created_by);
CREATE INDEX idx_quizzes_schedule ON quizzes(scheduled_start, scheduled_end);
CREATE INDEX idx_quizzes_status ON quizzes(status);
CREATE INDEX idx_quizzes_deleted_at ON quizzes(deleted_at);

-- =====================================================
-- Table: quiz_theory_sources - Tracks theory modules used as AI context
-- =====================================================
CREATE TABLE quiz_theory_sources (
    id                  BIGSERIAL PRIMARY KEY,
    quiz_id             BIGINT NOT NULL,
    theory_module_id    BIGINT NOT NULL,
    theory_week_id      BIGINT NOT NULL,

    -- Unique constraint: prevent duplicate module sources for same quiz
    CONSTRAINT uq_quiz_module UNIQUE (quiz_id, theory_module_id),

    -- Foreign keys
    CONSTRAINT fk_quiz_theory_sources_quiz
        FOREIGN KEY (quiz_id) REFERENCES quizzes(id) ON DELETE CASCADE,
    CONSTRAINT fk_quiz_theory_sources_module
        FOREIGN KEY (theory_module_id) REFERENCES theory_modules(id) ON DELETE CASCADE,
    CONSTRAINT fk_quiz_theory_sources_week
        FOREIGN KEY (theory_week_id) REFERENCES theory_weeks(id) ON DELETE CASCADE
);

-- Indexes for quiz_theory_sources
CREATE INDEX idx_quiz_theory_sources_quiz_id ON quiz_theory_sources(quiz_id);
CREATE INDEX idx_quiz_theory_sources_module_id ON quiz_theory_sources(theory_module_id);
CREATE INDEX idx_quiz_theory_sources_week_id ON quiz_theory_sources(theory_week_id);

-- =====================================================
-- Table: quiz_questions - Individual questions within a quiz
-- =====================================================
CREATE TABLE quiz_questions (
    id                  BIGSERIAL PRIMARY KEY,
    quiz_id             BIGINT NOT NULL,
    question_type       VARCHAR(20) NOT NULL,
    question_text       TEXT NOT NULL,
    explanation         TEXT,
    marks               INTEGER NOT NULL DEFAULT 1,
    difficulty          VARCHAR(10) NOT NULL,
    order_index         INTEGER NOT NULL DEFAULT 0,
    source_module_id    BIGINT,
    bloom_level         VARCHAR(20),
    is_ai_generated     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_question_type CHECK (question_type IN ('mcq', 'multi_select', 'true_false', 'short_answer', 'fill_blank')),
    CONSTRAINT chk_difficulty CHECK (difficulty IN ('easy', 'medium', 'hard')),
    CONSTRAINT chk_bloom_level CHECK (bloom_level IN ('remember', 'understand', 'apply', 'analyze', 'evaluate', 'create') OR bloom_level IS NULL),
    CONSTRAINT chk_marks CHECK (marks > 0)
);

-- Indexes for quiz_questions
CREATE INDEX idx_quiz_questions_quiz_id ON quiz_questions(quiz_id);
CREATE INDEX idx_quiz_questions_source_module ON quiz_questions(source_module_id);
CREATE INDEX idx_quiz_questions_difficulty ON quiz_questions(difficulty);
CREATE INDEX idx_quiz_questions_order ON quiz_questions(quiz_id, order_index);

-- Foreign keys
ALTER TABLE quiz_questions
    ADD CONSTRAINT fk_quiz_questions_quiz
        FOREIGN KEY (quiz_id) REFERENCES quizzes(id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_quiz_questions_module
        FOREIGN KEY (source_module_id) REFERENCES theory_modules(id) ON DELETE SET NULL;

-- =====================================================
-- Table: quiz_question_options - MCQ / multi-select options
-- =====================================================
CREATE TABLE quiz_question_options (
    id                  BIGSERIAL PRIMARY KEY,
    question_id         BIGINT NOT NULL,
    option_text         TEXT NOT NULL,
    is_correct          BOOLEAN NOT NULL DEFAULT FALSE,
    order_index         INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign key
    CONSTRAINT fk_quiz_question_options_question
        FOREIGN KEY (question_id) REFERENCES quiz_questions(id) ON DELETE CASCADE
);

-- Indexes for quiz_question_options
CREATE INDEX idx_quiz_question_options_question_id ON quiz_question_options(question_id);
CREATE INDEX idx_quiz_question_options_is_correct ON quiz_question_options(is_correct);
CREATE INDEX idx_quiz_question_options_order ON quiz_question_options(question_id, order_index);

-- =====================================================
-- Table: quiz_attempts - Student attempt tracking
-- =====================================================
CREATE TABLE quiz_attempts (
    id                  BIGSERIAL PRIMARY KEY,
    quiz_id             BIGINT NOT NULL,
    student_regdno      VARCHAR(50) NOT NULL,
    attempt_number      INTEGER NOT NULL,
    started_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    submitted_at        TIMESTAMP WITH TIME ZONE,
    score               INTEGER NOT NULL DEFAULT 0,
    max_score           INTEGER NOT NULL DEFAULT 0,
    percentage          DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    status              VARCHAR(20) NOT NULL DEFAULT 'in_progress',

    -- Unique constraint: prevent duplicate attempt numbers for same student/quiz
    CONSTRAINT uq_quiz_student_attempt UNIQUE (quiz_id, student_regdno, attempt_number),

    -- Constraints
    CONSTRAINT chk_attempt_status CHECK (status IN ('in_progress', 'submitted', 'graded', 'timed_out')),
    CONSTRAINT chk_score CHECK (score >= 0),
    CONSTRAINT chk_max_score CHECK (max_score >= 0),
    CONSTRAINT chk_percentage CHECK (percentage >= 0 AND percentage <= 100),
    CONSTRAINT chk_attempt_number CHECK (attempt_number >= 1),
    CONSTRAINT chk_submitted_after_started CHECK (submitted_at IS NULL OR submitted_at >= started_at),

    -- Foreign keys
    CONSTRAINT fk_quiz_attempts_quiz
        FOREIGN KEY (quiz_id) REFERENCES quizzes(id) ON DELETE CASCADE,
    CONSTRAINT fk_quiz_attempts_student
        FOREIGN KEY (student_regdno) REFERENCES users(regdno) ON DELETE CASCADE
);

-- Indexes for quiz_attempts
CREATE INDEX idx_quiz_attempts_quiz_id ON quiz_attempts(quiz_id);
CREATE INDEX idx_quiz_attempts_student_regdno ON quiz_attempts(student_regdno);
CREATE INDEX idx_quiz_attempts_submitted_at ON quiz_attempts(submitted_at);
CREATE INDEX idx_quiz_attempts_status ON quiz_attempts(status);

-- =====================================================
-- Table: quiz_answers - Per-question responses
-- =====================================================
CREATE TABLE quiz_answers (
    id                  BIGSERIAL PRIMARY KEY,
    attempt_id          BIGINT NOT NULL,
    question_id         BIGINT NOT NULL,
    selected_option_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    text_answer         TEXT,
    is_correct          BOOLEAN NOT NULL DEFAULT FALSE,
    marks_awarded       INTEGER NOT NULL DEFAULT 0,

    -- Constraints
    CONSTRAINT chk_marks_awarded CHECK (marks_awarded >= 0),

    -- Foreign keys
    CONSTRAINT fk_quiz_answers_attempt
        FOREIGN KEY (attempt_id) REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    CONSTRAINT fk_quiz_answers_question
        FOREIGN KEY (question_id) REFERENCES quiz_questions(id) ON DELETE CASCADE
);

-- Indexes for quiz_answers
CREATE INDEX idx_quiz_answers_attempt_id ON quiz_answers(attempt_id);
CREATE INDEX idx_quiz_answers_question_id ON quiz_answers(question_id);
CREATE INDEX idx_quiz_answers_is_correct ON quiz_answers(is_correct);

-- =====================================================
-- Comments for documentation
-- =====================================================
COMMENT ON TABLE quizzes IS 'Quiz metadata, scoped to a CourseOffering (which encodes college/branch/batch/section)';
COMMENT ON TABLE quiz_theory_sources IS 'Tracks which theory modules were used as context for AI quiz generation';
COMMENT ON TABLE quiz_questions IS 'Individual questions within a quiz';
COMMENT ON TABLE quiz_question_options IS 'MCQ and multi-select question options';
COMMENT ON TABLE quiz_attempts IS 'Student quiz attempt tracking';
COMMENT ON TABLE quiz_answers IS 'Per-question responses within a quiz attempt';

COMMENT ON COLUMN quizzes.course_offering_id IS 'Foreign key to course_offerings - primary scope anchor';
COMMENT ON COLUMN quizzes.quiz_type IS 'ai_generated, manual, or hybrid';
COMMENT ON COLUMN quizzes.status IS 'draft, published, closed, or archived';
COMMENT ON COLUMN quizzes.show_results_after IS 'When students can see results: immediately, after_close, or never';
COMMENT ON COLUMN quiz_questions.question_type IS 'mcq, multi_select, true_false, short_answer, or fill_blank';
COMMENT ON COLUMN quiz_questions.bloom_level IS 'Bloom taxonomy level: remember, understand, apply, analyze, evaluate, create';
COMMENT ON COLUMN quiz_answers.selected_option_ids IS 'JSONB array of selected option IDs for MCQ/multi_select';

-- =====================================================
-- Migration Info
-- =====================================================
DO $$
BEGIN
    RAISE NOTICE 'Migration 007: Quiz System tables created successfully';
    RAISE NOTICE '  - quizzes: Quiz metadata and configuration';
    RAISE NOTICE '  - quiz_theory_sources: AI context tracking';
    RAISE NOTICE '  - quiz_questions: Question bank';
    RAISE NOTICE '  - quiz_question_options: MCQ options';
    RAISE NOTICE '  - quiz_attempts: Student attempts';
    RAISE NOTICE '  - quiz_answers: Per-question responses';
END $$;
