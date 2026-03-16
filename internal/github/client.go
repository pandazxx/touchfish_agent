package github

import "context"

// PRState filters pull request listings.
type PRState string

const (
	PRStateOpen   PRState = "open"
	PRStateClosed PRState = "closed"
	PRStateAll    PRState = "all"
)

type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Head   string `json:"head"`
	Base   string `json:"base"`
}

type Issue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	State  string   `json:"state"`
	Labels []string `json:"labels"`
}

type Comment struct {
	ID     int    `json:"id"`
	Author string `json:"author"`
	Body   string `json:"body"`
	Date   string `json:"date"`
}

type CommitInfo struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// GitHubClient abstracts GitHub API operations for PRs, issues, and commits.
type GitHubClient interface {
	// Pull Requests
	CreatePR(ctx context.Context, head, base, title, body string) (*PullRequest, error)
	GetPR(ctx context.Context, number int) (*PullRequest, error)
	UpdatePR(ctx context.Context, number int, title, body *string) error
	ListPRs(ctx context.Context, state PRState, head string) ([]PullRequest, error)
	AddPRComment(ctx context.Context, prNumber int, body string) error

	// Issues
	ListIssues(ctx context.Context, labels []string) ([]Issue, error)
	GetIssue(ctx context.Context, number int) (*Issue, error)
	CreateIssue(ctx context.Context, title, body string, labels []string) (*Issue, error)
	AddIssueLabel(ctx context.Context, issueNumber int, label string) error
	RemoveIssueLabel(ctx context.Context, issueNumber int, label string) error
	AddIssueComment(ctx context.Context, issueNumber int, body string) error
	ListIssueComments(ctx context.Context, issueNumber int) ([]Comment, error)

	// Commits
	ListCommits(ctx context.Context, branch string, limit int) ([]CommitInfo, error)
}
