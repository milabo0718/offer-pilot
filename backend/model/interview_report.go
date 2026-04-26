package model

import "time"

type AbilityScores struct {
	Tech  int `json:"tech"`
	Eng   int `json:"eng"`
	PS    int `json:"ps"`
	Comm  int `json:"comm"`
	Learn int `json:"learn"`
	Fit   int `json:"fit"`
}

type InterviewReportData struct {
	SessionID      string        `json:"sessionId"`
	Summary        string        `json:"summary"`
	Recommendation string        `json:"recommendation"`
	Scores         AbilityScores `json:"scores"`
	Strengths      []string      `json:"strengths"`
	Risks          []string      `json:"risks"`
	ActionItems    []string      `json:"actionItems"`
	Detail         string        `json:"detail"`
	EvidenceCount  int           `json:"evidenceCount"`
	GeneratedAt    time.Time     `json:"generatedAt"`
	Fallback       bool          `json:"fallback"`
}

type InterviewReport struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID  string    `gorm:"uniqueIndex;not null;type:varchar(36)" json:"sessionId"`
	UserName   string    `gorm:"index;not null;type:varchar(20)" json:"username"`
	ModelType  string    `gorm:"type:varchar(20)" json:"modelType"`
	JDProfile  string    `gorm:"type:text" json:"jdProfile"`
	ReportJSON string    `gorm:"type:longtext" json:"reportJson"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
