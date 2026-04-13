package model

type InterviewReportDimension struct {
	Name    string `json:"name"`
	Score   int    `json:"score"`
	Comment string `json:"comment,omitempty"`
}

type InterviewReportScores struct {
	Tech  int `json:"tech"`
	Eng   int `json:"eng"`
	PS    int `json:"ps"`
	Comm  int `json:"comm"`
	Learn int `json:"learn"`
	Fit   int `json:"fit"`
}

type InterviewReport struct {
	SessionID     string                   `json:"sessionId"`
	OverallScore  int                      `json:"overallScore,omitempty"`
	Dimensions    []InterviewReportDimension `json:"dimensions"`
	Scores        InterviewReportScores    `json:"scores"`
	Summary       string                   `json:"summary,omitempty"`
	Strengths     []string                 `json:"strengths,omitempty"`
	Risks         []string                 `json:"risks,omitempty"`
	Suggestions   []string                 `json:"suggestions,omitempty"`
	Recommendation string                  `json:"recommendation,omitempty"`
	Detail        string                   `json:"detail,omitempty"`
}
