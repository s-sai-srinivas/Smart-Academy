package models

// =====================================================
// Transcript and Progress Helper Structs
// These are NOT database tables - they are used for query results
// =====================================================

// TranscriptRecord represents a flattened view for transcript queries
// Used for displaying student academic records
type TranscriptRecord struct {
	CourseCode     string `json:"course_code"`
	CourseName     string `json:"course_name"`
	AcademicYear   string `json:"academic_year"`
	Category       string `json:"category"`
	FinalGrade     string `json:"final_grade"`
	Status         string `json:"status"`
	EnrollmentType string `json:"enrollment_type"`
	Credits        int    `json:"credits"`
}

// StudentProgress represents a student's academic progress summary
type StudentProgress struct {
	RegdNo          string  `json:"regdno"`
	Name            string  `json:"name"`
	CohortYear      int     `json:"cohort_year"`
	CurrentYear     int     `json:"current_year"`   // Derived: current_academic_year - cohort_year + 1
	ProgressIndex   int     `json:"progress_index"` // Actual progress tracking
	CreditsEarned   int     `json:"credits_earned"`
	CreditsRequired int     `json:"credits_required"`
	CGPA            float64 `json:"cgpa"`
	ActiveBacklogs  int     `json:"active_backlogs"`
}

// CourseAvailability represents a course available to a student
type CourseAvailability struct {
	OfferingID  *uint  `json:"offering_id"`
	CourseCode  string `json:"course_code"`
	CourseName  string `json:"course_name"`
	CourseType  string `json:"course_type"`
	Category    string `json:"category"`
	SectionName string `json:"section_name"`
	FacultyName string `json:"faculty_name"`
	CourseID    uint   `json:"course_id"`
	Credits     int    `json:"credits"`
	IsMandatory bool   `json:"is_mandatory"`
	IsEnrolled  bool   `json:"is_enrolled"`
}

// EnrollmentSummary represents summary statistics for enrollments
type EnrollmentSummary struct {
	TotalCourses     int     `json:"total_courses"`
	CompletedCourses int     `json:"completed_courses"`
	PendingCourses   int     `json:"pending_courses"`
	FailedCourses    int     `json:"failed_courses"`
	TotalCredits     int     `json:"total_credits"`
	EarnedCredits    int     `json:"earned_credits"`
	CompletionRate   float64 `json:"completion_rate"` // percentage
}
