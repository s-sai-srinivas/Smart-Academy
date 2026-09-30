-- =====================================================
-- MIGRATION: Batch-Based Academic Architecture
-- Version: 001
-- Date: 2025-02-20
--
-- Changes:
-- - Removes: Semester dependency from Section
-- - Removes: Batch, Semester fields from Course
-- - Adds: Regulation table (curriculum versioning)
-- - Adds: Curriculum table (program-regulation combination)
-- - Adds: CurriculumCourse table (course to curriculum mapping)
-- - Adds: Student table (academic identity extension)
-- - Adds: Faculty table (employment info extension)
-- - Adds: CourseOffering table (academic course instance)
-- - Redesigns: Enrollment table (replaces StudentEnrollment)
-- - Redesigns: FacultyAssignment table (replaces FacultyCourseAssignment, CourseAssignment)
-- - Adds: Exam table (linked to CourseOffering)
-- - Adds: ExamResult table (student exam results)
-- =====================================================

BEGIN;

-- =====================================================
-- STEP 1: Create new tables
-- =====================================================

-- 1. Regulation table (curriculum versioning)
CREATE TABLE IF NOT EXISTS regulations (
    regulation_id SERIAL PRIMARY KEY,
    code VARCHAR(20) NOT NULL UNIQUE,
    effective_from_cohort INT NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE
);

COMMENT ON TABLE regulations IS 'Curriculum versions that apply to specific cohorts (e.g., R2021 for cohorts 2021-2024)';

-- 2. Curriculum table (program + regulation combination)
CREATE TABLE IF NOT EXISTS curriculums (
    curriculum_id SERIAL PRIMARY KEY,
    program_id INT NOT NULL REFERENCES programs(program_id) ON DELETE CASCADE,
    regulation_id INT NOT NULL REFERENCES regulations(regulation_id) ON DELETE CASCADE,
    duration_years INT NOT NULL DEFAULT 4,
    is_active BOOLEAN DEFAULT TRUE,
    UNIQUE(program_id, regulation_id)
);

COMMENT ON TABLE curriculums IS 'Complete academic structure for a program under a specific regulation';

-- 3. CurriculumCourse mapping table (courses in curriculum by study year)
CREATE TABLE IF NOT EXISTS curriculum_courses (
    id SERIAL PRIMARY KEY,
    curriculum_id INT NOT NULL REFERENCES curriculums(curriculum_id) ON DELETE CASCADE,
    course_id INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    study_year_index INT NOT NULL CHECK (study_year_index > 0),
    category VARCHAR(50), -- Core, Elective, Lab, OpenElective
    is_mandatory BOOLEAN DEFAULT TRUE,
    sequence_in_year INT DEFAULT 0,
    UNIQUE(curriculum_id, course_id, study_year_index)
);

COMMENT ON TABLE curriculum_courses IS 'Maps courses to specific study years within a curriculum';

-- 4. Student table (academic identity extension for User)
CREATE TABLE IF NOT EXISTS students (
    regdno VARCHAR(50) PRIMARY KEY REFERENCES users(regdno) ON DELETE CASCADE,
    curriculum_id INT NOT NULL REFERENCES curriculums(curriculum_id),
    cohort_year INT NOT NULL CHECK (cohort_year > 2000),
    admission_year INT NOT NULL CHECK (admission_year >= cohort_year),
    entry_level INT DEFAULT 1 CHECK (entry_level IN (1, 2, 3)),
    branch_id INT NOT NULL REFERENCES branches(branch_id),
    section_id INT REFERENCES sections(section_id) ON DELETE SET NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'dormant', 'graduated', 'withdrawn')),
    progress_index INT DEFAULT 1 CHECK (progress_index > 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE students IS 'Academic-specific information for students (batch-based identity)';

-- 5. Faculty table (employment info extension for User)
CREATE TABLE IF NOT EXISTS faculties (
    regdno VARCHAR(50) PRIMARY KEY REFERENCES users(regdno) ON DELETE CASCADE,
    branch_id INT NOT NULL REFERENCES branches(branch_id),
    designation VARCHAR(100),
    joining_date DATE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE faculties IS 'Employment-specific information for faculty members';

-- 6. CourseOffering table (specific instance of a course)
CREATE TABLE IF NOT EXISTS course_offerings (
    id SERIAL PRIMARY KEY,
    course_id INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    curriculum_id INT NOT NULL REFERENCES curriculums(curriculum_id),
    academic_year_id INT NOT NULL REFERENCES academic_years(academic_year_id) ON DELETE CASCADE,
    section_id INT NOT NULL REFERENCES sections(section_id) ON DELETE CASCADE,
    study_year_index INT NOT NULL CHECK (study_year_index > 0),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    UNIQUE(course_id, curriculum_id, academic_year_id, section_id)
);

COMMENT ON TABLE course_offerings IS 'Specific instance of a course offered to a section in an academic year';

-- 7. New Enrollment table (replaces StudentEnrollment)
CREATE TABLE IF NOT EXISTS enrollments (
    id SERIAL PRIMARY KEY,
    student_regdno VARCHAR(50) NOT NULL REFERENCES users(regdno) ON DELETE CASCADE,
    course_offering_id INT NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('Regular', 'Elective', 'Backlog')),
    status VARCHAR(20) DEFAULT 'enrolled' CHECK (status IN ('enrolled', 'completed', 'failed', 'dropped', 'withdrawn')),
    attempt_no INT DEFAULT 1 CHECK (attempt_no > 0),
    final_grade VARCHAR(10),
    enrolled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    UNIQUE(student_regdno, course_offering_id, attempt_no)
);

COMMENT ON TABLE enrollments IS 'Student enrollments in course offerings (replaces old StudentEnrollment)';

-- 8. New FacultyAssignment table (replaces FacultyCourseAssignment, CourseAssignment)
CREATE TABLE IF NOT EXISTS faculty_assignments (
    id SERIAL PRIMARY KEY,
    faculty_regdno VARCHAR(50) NOT NULL REFERENCES users(regdno) ON DELETE CASCADE,
    course_offering_id INT NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,
    role VARCHAR(20) DEFAULT 'teacher' CHECK (role IN ('teacher', 'co-teacher', 'examiner', 'tutor')),
    is_active BOOLEAN DEFAULT TRUE,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(faculty_regdno, course_offering_id)
);

COMMENT ON TABLE faculty_assignments IS 'Faculty assigned to course offerings (replaces old assignment models)';

-- 9. Exam table
CREATE TABLE IF NOT EXISTS exams (
    id SERIAL PRIMARY KEY,
    course_offering_id INT NOT NULL REFERENCES course_offerings(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('Midterm1', 'Midterm2', 'Final', 'Practical', 'Quiz', 'Assignment')),
    sequence INT DEFAULT 1,
    name VARCHAR(200),
    description TEXT,
    scheduled_date TIMESTAMP NOT NULL,
    duration_minutes INT NOT NULL CHECK (duration_minutes > 0),
    total_marks INT NOT NULL CHECK (total_marks > 0),
    passing_marks INT NOT NULL CHECK (passing_marks >= 0 AND passing_marks <= total_marks),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

COMMENT ON TABLE exams IS 'Examinations linked to course offerings';

-- 10. ExamResult table
CREATE TABLE IF NOT EXISTS exam_results (
    id SERIAL PRIMARY KEY,
    exam_id INT NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    student_regdno VARCHAR(50) NOT NULL REFERENCES users(regdno) ON DELETE CASCADE,
    marks_obtained INT NOT NULL CHECK (marks_obtained >= 0),
    grade VARCHAR(10),
    attempt_no INT DEFAULT 1 CHECK (attempt_no > 0),
    evaluated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    remarks TEXT,
    UNIQUE(exam_id, student_regdno, attempt_no)
);

COMMENT ON TABLE exam_results IS 'Student exam results (multiple attempts allowed)';

-- =====================================================
-- STEP 2: Migrate existing Section table structure
-- =====================================================

-- Add new columns to Section (CohortYear, SectionName)
-- Note: If columns already exist, these will be ignored (PostgreSQL IF NOT EXISTS)
DO $$
BEGIN
    -- Add cohort_year column if not exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'sections' AND column_name = 'cohort_year'
    ) THEN
        ALTER TABLE sections ADD COLUMN cohort_year INT;
    END IF;

    -- Add section_name column if not exists (may already exist as section_name)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'sections' AND column_name = 'section_name'
    ) THEN
        ALTER TABLE sections ADD COLUMN section_name VARCHAR(10);
    END IF;
END $$;

-- =====================================================
-- STEP 3: Migrate existing Course table structure
-- =====================================================

-- Add new course_type column if not exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'courses' AND column_name = 'course_type'
    ) THEN
        ALTER TABLE courses ADD COLUMN course_type VARCHAR(20);
    END IF;
END $$;

-- Migrate data from course_category to course_type if needed (only if course_category exists)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'courses' AND column_name = 'course_category'
    ) THEN
        UPDATE courses SET course_type = course_category WHERE course_type IS NULL AND course_category IS NOT NULL;
    END IF;
END $$;

-- =====================================================
-- STEP 4: Create indexes for performance
-- =====================================================

-- Student indexes
CREATE INDEX IF NOT EXISTS idx_students_curriculum ON students(curriculum_id);
CREATE INDEX IF NOT EXISTS idx_students_cohort ON students(cohort_year);
CREATE INDEX IF NOT EXISTS idx_students_branch ON students(branch_id);
CREATE INDEX IF NOT EXISTS idx_students_section ON students(section_id);
CREATE INDEX IF NOT EXISTS idx_students_status ON students(status);

-- Faculty indexes
CREATE INDEX IF NOT EXISTS idx_faculties_branch ON faculties(branch_id);
CREATE INDEX IF NOT EXISTS idx_faculties_active ON faculties(is_active);

-- Curriculum indexes
CREATE INDEX IF NOT EXISTS idx_curriculums_program ON curriculums(program_id);
CREATE INDEX IF NOT EXISTS idx_curriculums_regulation ON curriculums(regulation_id);
CREATE INDEX IF NOT EXISTS idx_curriculums_active ON curriculums(is_active);

-- CurriculumCourse indexes
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_curriculum ON curriculum_courses(curriculum_id);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_course ON curriculum_courses(course_id);
CREATE INDEX IF NOT EXISTS idx_curriculum_courses_year ON curriculum_courses(study_year_index);

-- CourseOffering indexes
CREATE INDEX IF NOT EXISTS idx_offerings_course ON course_offerings(course_id);
CREATE INDEX IF NOT EXISTS idx_offerings_curriculum ON course_offerings(curriculum_id);
CREATE INDEX IF NOT EXISTS idx_offerings_academic_year ON course_offerings(academic_year_id);
CREATE INDEX IF NOT EXISTS idx_offerings_section ON course_offerings(section_id);
CREATE INDEX IF NOT EXISTS idx_offerings_study_year ON course_offerings(study_year_index);
CREATE INDEX IF NOT EXISTS idx_offerings_active ON course_offerings(is_active);

-- Enrollment indexes
CREATE INDEX IF NOT EXISTS idx_enrollments_student ON enrollments(student_regdno);
CREATE INDEX IF NOT EXISTS idx_enrollments_offering ON enrollments(course_offering_id);
CREATE INDEX IF NOT EXISTS idx_enrollments_status ON enrollments(status);
CREATE INDEX IF NOT EXISTS idx_enrollments_type ON enrollments(type);

-- FacultyAssignment indexes
CREATE INDEX IF NOT EXISTS idx_faculty_assignments_faculty ON faculty_assignments(faculty_regdno);
CREATE INDEX IF NOT EXISTS idx_faculty_assignments_offering ON faculty_assignments(course_offering_id);
CREATE INDEX IF NOT EXISTS idx_faculty_assignments_active ON faculty_assignments(is_active);

-- Exam indexes
CREATE INDEX IF NOT EXISTS idx_exams_offering ON exams(course_offering_id);
CREATE INDEX IF NOT EXISTS idx_exams_scheduled ON exams(scheduled_date);
CREATE INDEX IF NOT EXISTS idx_exams_active ON exams(is_active);

-- ExamResult indexes
CREATE INDEX IF NOT EXISTS idx_exam_results_exam ON exam_results(exam_id);
CREATE INDEX IF NOT EXISTS idx_exam_results_student ON exam_results(student_regdno);

COMMIT;

-- =====================================================
-- STEP 5: Data Migration Helper Functions
-- =====================================================

-- Function to calculate StudyYearIndex from CohortYear and AcademicYear
CREATE OR REPLACE FUNCTION calculate_study_year_index(p_cohort_year INT, p_academic_year_start INT)
RETURNS INT AS $$
BEGIN
    RETURN p_academic_year_start - p_cohort_year + 1;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION calculate_study_year_index IS 'Calculates study year index (1, 2, 3, 4) from cohort year and academic year';

-- =====================================================
-- STEP 6: Cleanup (Run separately after data migration)
-- =====================================================
-- Uncomment and run these ONLY after verifying data has been migrated
-- and old tables are no longer needed

-- BEGIN;
--
-- -- Drop old semester dependency from Section
-- ALTER TABLE sections DROP CONSTRAINT IF EXISTS sections_semester_id_fkey;
-- ALTER TABLE sections DROP COLUMN IF EXISTS semester_id;
--
-- -- Drop old columns from Course
-- ALTER TABLE courses DROP COLUMN IF EXISTS semester;
-- ALTER TABLE courses DROP COLUMN IF EXISTS batch;
-- ALTER TABLE courses DROP COLUMN IF EXISTS course_category;
--
-- -- Drop old tables (if no longer needed)
-- DROP TABLE IF EXISTS student_enrollments;
-- DROP TABLE IF EXISTS faculty_course_assignments;
-- DROP TABLE IF EXISTS course_assignments;
-- DROP TABLE IF EXISTS semesters;
--
-- COMMIT;
