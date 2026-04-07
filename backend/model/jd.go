package model

type JDParseResult struct {
	JobTitle   string   `json:"jobTitle"`
	Skills     []string `json:"skills"`
	Experience string   `json:"experience"`
	Keywords   []string `json:"keywords"`
	Summary    string   `json:"summary"`
}
