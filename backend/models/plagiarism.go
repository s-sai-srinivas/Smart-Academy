package models

import (
	"time"
)

// PlagiarismStatus represents the plagiarism classification
type PlagiarismStatus string

const (
	PlagiarismSafe        PlagiarismStatus = "SAFE"
	PlagiarismSuspicious  PlagiarismStatus = "SUSPICIOUS"
	PlagiarismPlagiarized PlagiarismStatus = "PLAGIARIZED"
)

// PlagiarismResult stores the comparison result between two submissions
type PlagiarismResult struct {
	CheckedAt         time.Time        `json:"checked_at"`
	Submission1       *Submission      `gorm:"foreignKey:SubmissionID1;references:ID" json:"submission_1,omitempty"`
	Submission2       *Submission      `gorm:"foreignKey:SubmissionID2;references:ID" json:"submission_2,omitempty"`
	Status            PlagiarismStatus `gorm:"size:20" json:"status"`
	ID                uint             `gorm:"primaryKey" json:"id"`
	SubmissionID1     uint             `gorm:"index" json:"submission_id_1"`
	SubmissionID2     uint             `gorm:"index" json:"submission_id_2"`
	SimilarityPercent float64          `json:"similarity_percent"`
}

// PlagiarismMatch stores detailed line-by-line match information
type PlagiarismMatch struct {
	PlagiarismResult   *PlagiarismResult `gorm:"foreignKey:PlagiarismResultID;references:ID" json:"-"`
	File1              string            `json:"file1"`
	File2              string            `json:"file2"`
	ID                 uint              `gorm:"primaryKey" json:"id"`
	PlagiarismResultID uint              `gorm:"index" json:"plagiarism_result_id"`
	StartLine1         int               `json:"start_line_1"`
	EndLine1           int               `json:"end_line_1"`
	StartLine2         int               `json:"start_line_2"`
	EndLine2           int               `json:"end_line_2"`
}

// ClassifySimilarity returns the plagiarism status based on similarity percentage
func ClassifySimilarity(similarity float64) PlagiarismStatus {
	switch {
	case similarity > 60:
		return PlagiarismPlagiarized
	case similarity >= 30:
		return PlagiarismSuspicious
	default:
		return PlagiarismSafe
	}
}
