package state

import "time"

type TeamState struct {
	TeamName    string        `json:"teamName"`
	Session     *SessionState `json:"session,omitempty"`
	LastUpdated time.Time     `json:"lastUpdated"`
}

type SessionState struct {
	Branch    string    `json:"branch"`
	PRNumber  int       `json:"prNumber"`
	StartedAt time.Time `json:"startedAt"`
	SE        SEState   `json:"se"`
	QA        QAState   `json:"qa"`
}

type SEState struct {
	LastProcessedCommit string `json:"lastProcessedCommit"`
	LastRequirementHash string `json:"lastRequirementHash"`
	LastTestReqHash     string `json:"lastTestReqHash"`
	CurrentIssueNumber  int    `json:"currentIssueNumber"`
	ConsecutiveFailures int    `json:"consecutiveFailures"`
	Phase               string `json:"phase"`
}

type QAState struct {
	LastReviewedCommit  string `json:"lastReviewedCommit"`
	LastRequirementHash string `json:"lastRequirementHash"`
	Phase               string `json:"phase"`
}
