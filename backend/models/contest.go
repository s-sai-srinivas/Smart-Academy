package models

import "time"

// ContestTargetCohort represents multiple target batches for a contest
type ContestTargetCohort struct {
	CreatedAt  time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	Contest    *Contest  `gorm:"foreignKey:ContestID;references:ContestID;constraint:OnDelete:CASCADE" json:"contest,omitempty"`
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID  uint      `gorm:"index;not null" json:"contest_id"`
	CohortYear int       `gorm:"not null" json:"cohort_year"`
}

// ContestTargetBranch represents multiple target branches for a contest
type ContestTargetBranch struct {
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	Contest   *Contest  `gorm:"foreignKey:ContestID;references:ContestID;constraint:OnDelete:CASCADE" json:"contest,omitempty"`
	Branch    *Branch   `gorm:"foreignKey:BranchID;references:BranchID;constraint:OnDelete:CASCADE" json:"branch,omitempty"`
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID uint      `gorm:"index;not null" json:"contest_id"`
	BranchID  uint      `gorm:"not null" json:"branch_id"`
}

// Contest represents coding competitions
type Contest struct {
	StartTime           time.Time             `gorm:"not null;index" json:"start_time"`
	UpdatedAt           time.Time             `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt           time.Time             `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	EndTime             time.Time             `gorm:"not null;index" json:"end_time"`
	TargetBranchID      *uint                 `json:"target_branch_id,omitempty"`
	CollegeID           *string               `gorm:"index" json:"college_id"`
	TargetCohort        *int                  `json:"target_cohort,omitempty"`
	ContestQuiz         *ContestQuiz          `gorm:"foreignKey:ContestID" json:"contest_quiz,omitempty"`
	PracticeStartTime   *time.Time            `json:"practice_start_time,omitempty"`
	PracticeEndTime     *time.Time            `json:"practice_end_time,omitempty"`
	TargetBranch        *Branch               `gorm:"foreignKey:TargetBranchID" json:"target_branch,omitempty"`
	College             *College              `gorm:"foreignKey:CollegeID" json:"college,omitempty"`
	Description         string                `gorm:"type:text" json:"description"`
	CreatedBy           string                `json:"created_by"`
	Title               string                `gorm:"size:200;not null" json:"title"`
	Sections            []ContestSection      `gorm:"foreignKey:ContestID" json:"sections,omitempty"`
	ContestSubmissions  []ContestSubmission   `gorm:"foreignKey:ContestID" json:"submissions,omitempty"`
	TargetBranchNames   []string              `gorm:"-" json:"target_branch_names,omitempty"`
	TargetCohorts       []ContestTargetCohort `gorm:"foreignKey:ContestID;constraint:OnDelete:CASCADE" json:"target_cohorts,omitempty"`
	TargetBranches      []ContestTargetBranch `gorm:"foreignKey:ContestID;constraint:OnDelete:CASCADE" json:"target_branches,omitempty"`
	ContestProblems     []ContestProblem      `gorm:"foreignKey:ContestID" json:"contest_problems,omitempty"`
	ContestParticipants []ContestParticipant  `gorm:"foreignKey:ContestID" json:"participants,omitempty"`
	TargetBranchIDs     []uint                `gorm:"-" json:"target_branch_ids,omitempty"`
	TargetCohortYears   []int                 `gorm:"-" json:"target_cohort_years,omitempty"`
	ContestID           uint                  `gorm:"primaryKey;autoIncrement" json:"contest_id"`
	QuizDurationMinutes int                   `gorm:"default:0" json:"quiz_duration_minutes"`
	IsActive            bool                  `gorm:"-" json:"is_active"`
	IsFrozen            bool                  `gorm:"-" json:"is_frozen"`
	PracticeEnabled     bool                  `gorm:"default:false" json:"practice_enabled"`
	HasQuiz             bool                  `gorm:"default:false" json:"has_quiz"`
}

// ContestSection represents a section within a contest (quiz or coding)
type ContestSection struct {
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Contest         *Contest         `gorm:"foreignKey:ContestID;references:ContestID;constraint:OnDelete:CASCADE" json:"contest,omitempty"`
	ContestQuiz     *ContestQuiz     `gorm:"foreignKey:SectionID" json:"contest_quiz,omitempty"`
	SectionName     string           `gorm:"size:200;not null" json:"section_name"`
	SectionType     string           `gorm:"size:20;not null;default:coding" json:"section_type"`
	Description     string           `gorm:"type:text" json:"description"`
	ContestProblems []ContestProblem `gorm:"foreignKey:SectionID" json:"contest_problems,omitempty"`
	ID              uint             `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID       uint             `gorm:"index;not null" json:"contest_id"`
	DurationMinutes int              `gorm:"default:0" json:"duration_minutes"`
	OrderIndex      int              `gorm:"default:0" json:"order_index"`
}

// ContestProblem maps problems to contests with points and ordering
type ContestProblem struct {
	CreatedAt    time.Time       `gorm:"default:CURRENT_TIMESTAMP" json:"created_at"`
	SectionID    *uint           `gorm:"index" json:"section_id,omitempty"`
	Contest      *Contest        `gorm:"foreignKey:ContestID;references:ContestID;constraint:OnDelete:CASCADE" json:"contest,omitempty"`
	Problem      *Problem        `gorm:"foreignKey:ProblemID;references:ID;constraint:OnDelete:CASCADE" json:"problem,omitempty"`
	Section      *ContestSection `gorm:"foreignKey:SectionID;constraint:OnDelete:SET NULL" json:"section,omitempty"`
	ID           uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID    uint            `gorm:"index;not null" json:"contest_id"`
	ProblemID    uint            `gorm:"index;not null" json:"problem_id"`
	Points       int             `gorm:"default:100" json:"points"`
	ProblemOrder int             `gorm:"default:0" json:"problem_order"`
}

// ContestSubmission tracks submissions to contest problems with attempt tracking
type ContestSubmission struct {
	SubmittedAt   time.Time  `gorm:"not null;index" json:"submitted_at"`
	Contest       *Contest   `gorm:"foreignKey:ContestID;references:ContestID;constraint:OnDelete:CASCADE" json:"contest,omitempty"`
	College       *College   `gorm:"foreignKey:CollegeID;references:CollegeID" json:"college,omitempty"`
	User          *User      `gorm:"foreignKey:UserRegdNo;references:RegdNo;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	CollegeID     *string    `gorm:"index;size:50" json:"college_id"`
	Problem       *Problem   `gorm:"foreignKey:ProblemID;references:ID;constraint:OnDelete:CASCADE" json:"problem,omitempty"`
	EvaluatedAt   *time.Time `json:"evaluated_at,omitempty"`
	UserRegdNo    string     `gorm:"index;size:50;not null" json:"user_regdno"`
	SourceCode    string     `gorm:"type:text" json:"source_code"`
	Status        string     `gorm:"size:50" json:"status"`
	AttemptNumber int        `gorm:"default:1" json:"attempt_number"`
	Score         int        `gorm:"default:0" json:"score"`
	MaxScore      int        `gorm:"default:0" json:"max_score"`
	ExecutionTime float64    `json:"execution_time"`
	MemoryUsed    int        `json:"memory_used"`
	ID            uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	PenaltyTime   int        `gorm:"default:0" json:"penalty_time"`
	LanguageID    int        `json:"language_id"`
	ProblemID     uint       `gorm:"index;not null" json:"problem_id"`
	ContestID     uint       `gorm:"index;not null" json:"contest_id"`
	IsFinal       bool       `gorm:"default:false" json:"is_final"`
	IsPractice    bool       `gorm:"default:false" json:"is_practice"`
	Passed        bool       `gorm:"default:false" json:"passed"`
}

// ContestParticipant tracks participants and their scores
type ContestParticipant struct {
	RegisteredAt           time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"registered_at"`
	User                   *User      `gorm:"foreignKey:UserRegdNo;references:RegdNo;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Contest                *Contest   `gorm:"foreignKey:ContestID;references:ContestID;constraint:OnDelete:CASCADE" json:"contest,omitempty"`
	EligibilitySnapshot    *string    `gorm:"type:text" json:"eligibility_snapshot,omitempty"`
	LastSubmissionAt       *time.Time `json:"last_submission_at"`
	DisqualificationReason *string    `gorm:"type:text" json:"disqualification_reason,omitempty"`
	FinishedAt             *time.Time `json:"finished_at,omitempty"`
	DisqualifiedAt         *time.Time `json:"disqualified_at,omitempty"`
	UserRegdNo             string     `gorm:"primaryKey;size:50" json:"user_regdno"`
	TotalScore             int        `gorm:"default:0" json:"total_score"`
	EscViolations          int        `gorm:"default:0" json:"esc_violations"`
	ContestID              uint       `gorm:"primaryKey" json:"contest_id"`
	ProblemsSolved         int        `gorm:"default:0" json:"problems_solved"`
	HasFinished            bool       `gorm:"default:false" json:"has_finished"`
	Disqualified           bool       `gorm:"default:false" json:"disqualified"`
}

// ContestProblemTestCase maps test cases to contest problems for auto-scoring
type ContestProblemTestCase struct {
	Contest      *Contest  `gorm:"foreignKey:ContestID;references:ContestID;constraint:OnDelete:CASCADE" json:"contest,omitempty"`
	Problem      *Problem  `gorm:"foreignKey:ProblemID;references:ID;constraint:OnDelete:CASCADE" json:"problem,omitempty"`
	TestCase     *TestCase `gorm:"foreignKey:TestCaseID;references:ID;constraint:OnDelete:CASCADE" json:"test_case,omitempty"`
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ContestID    uint      `gorm:"index;not null" json:"contest_id"`
	ProblemID    uint      `gorm:"index;not null" json:"problem_id"`
	TestCaseID   uint      `gorm:"not null" json:"test_case_id"`
	PointsWeight float64   `gorm:"default:1.0" json:"points_weight"`
}
