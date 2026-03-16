package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// RESTClient implements GitHubClient using the GitHub REST API.
type RESTClient struct {
	baseURL string
	owner   string
	repo    string
	token   string
	client  *http.Client
}

// NewRESTClient creates a client for the public GitHub API.
func NewRESTClient(owner, repo, token string) *RESTClient {
	return &RESTClient{
		baseURL: "https://api.github.com",
		owner:   owner,
		repo:    repo,
		token:   token,
		client:  &http.Client{},
	}
}

// NewRESTClientWithBaseURL creates a client with a custom base URL (for testing).
func NewRESTClientWithBaseURL(baseURL, owner, repo, token string) *RESTClient {
	return &RESTClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		owner:   owner,
		repo:    repo,
		token:   token,
		client:  &http.Client{},
	}
}

func (c *RESTClient) url(path string, args ...any) string {
	formatted := fmt.Sprintf(path, args...)
	return fmt.Sprintf("%s/repos/%s/%s%s", c.baseURL, c.owner, c.repo, formatted)
}

func (c *RESTClient) do(ctx context.Context, method, url string, reqBody, respBody any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("github API error %d: %s", resp.StatusCode, string(respData))
	}

	if respBody != nil && len(respData) > 0 {
		if err := json.Unmarshal(respData, respBody); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

// --- Pull Requests ---

type createPRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

type ghPullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Head   struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

func prFromGH(gh ghPullRequest) PullRequest {
	return PullRequest{
		Number: gh.Number,
		Title:  gh.Title,
		Body:   gh.Body,
		State:  gh.State,
		Head:   gh.Head.Ref,
		Base:   gh.Base.Ref,
	}
}

func (c *RESTClient) CreatePR(ctx context.Context, head, base, title, body string) (*PullRequest, error) {
	var gh ghPullRequest
	err := c.do(ctx, http.MethodPost, c.url("/pulls"), createPRRequest{
		Title: title, Body: body, Head: head, Base: base,
	}, &gh)
	if err != nil {
		return nil, err
	}
	pr := prFromGH(gh)
	return &pr, nil
}

func (c *RESTClient) GetPR(ctx context.Context, number int) (*PullRequest, error) {
	var gh ghPullRequest
	if err := c.do(ctx, http.MethodGet, c.url("/pulls/%d", number), nil, &gh); err != nil {
		return nil, err
	}
	pr := prFromGH(gh)
	return &pr, nil
}

func (c *RESTClient) UpdatePR(ctx context.Context, number int, title, body *string) error {
	reqBody := map[string]string{}
	if title != nil {
		reqBody["title"] = *title
	}
	if body != nil {
		reqBody["body"] = *body
	}
	return c.do(ctx, http.MethodPatch, c.url("/pulls/%d", number), reqBody, nil)
}

func (c *RESTClient) ListPRs(ctx context.Context, state PRState, head string) ([]PullRequest, error) {
	u := c.url("/pulls?state=%s", string(state))
	if head != "" {
		u += "&head=" + c.owner + ":" + head
	}
	var ghPRs []ghPullRequest
	if err := c.do(ctx, http.MethodGet, u, nil, &ghPRs); err != nil {
		return nil, err
	}
	prs := make([]PullRequest, len(ghPRs))
	for i, gh := range ghPRs {
		prs[i] = prFromGH(gh)
	}
	return prs, nil
}

func (c *RESTClient) AddPRComment(ctx context.Context, prNumber int, body string) error {
	return c.do(ctx, http.MethodPost, c.url("/issues/%d/comments", prNumber),
		map[string]string{"body": body}, nil)
}

// --- Issues ---

type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func issueFromGH(gh ghIssue) Issue {
	labels := make([]string, len(gh.Labels))
	for i, l := range gh.Labels {
		labels[i] = l.Name
	}
	return Issue{
		Number: gh.Number,
		Title:  gh.Title,
		Body:   gh.Body,
		State:  gh.State,
		Labels: labels,
	}
}

func (c *RESTClient) ListIssues(ctx context.Context, labels []string) ([]Issue, error) {
	u := c.url("/issues")
	if len(labels) > 0 {
		u += "?labels=" + url.QueryEscape(strings.Join(labels, ","))
	}
	var ghIssues []ghIssue
	if err := c.do(ctx, http.MethodGet, u, nil, &ghIssues); err != nil {
		return nil, err
	}
	issues := make([]Issue, len(ghIssues))
	for i, gh := range ghIssues {
		issues[i] = issueFromGH(gh)
	}
	return issues, nil
}

func (c *RESTClient) GetIssue(ctx context.Context, number int) (*Issue, error) {
	var gh ghIssue
	if err := c.do(ctx, http.MethodGet, c.url("/issues/%d", number), nil, &gh); err != nil {
		return nil, err
	}
	issue := issueFromGH(gh)
	return &issue, nil
}

func (c *RESTClient) CreateIssue(ctx context.Context, title, body string, labels []string) (*Issue, error) {
	reqBody := map[string]any{
		"title":  title,
		"body":   body,
		"labels": labels,
	}
	var gh ghIssue
	if err := c.do(ctx, http.MethodPost, c.url("/issues"), reqBody, &gh); err != nil {
		return nil, err
	}
	issue := issueFromGH(gh)
	return &issue, nil
}

func (c *RESTClient) AddIssueLabel(ctx context.Context, issueNumber int, label string) error {
	return c.do(ctx, http.MethodPost, c.url("/issues/%d/labels", issueNumber),
		map[string][]string{"labels": {label}}, nil)
}

func (c *RESTClient) RemoveIssueLabel(ctx context.Context, issueNumber int, label string) error {
	return c.do(ctx, http.MethodDelete, c.url("/issues/%d/labels/%s", issueNumber, label), nil, nil)
}

func (c *RESTClient) AddIssueComment(ctx context.Context, issueNumber int, body string) error {
	return c.do(ctx, http.MethodPost, c.url("/issues/%d/comments", issueNumber),
		map[string]string{"body": body}, nil)
}

type ghComment struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
}

func (c *RESTClient) ListIssueComments(ctx context.Context, issueNumber int) ([]Comment, error) {
	var ghComments []ghComment
	if err := c.do(ctx, http.MethodGet, c.url("/issues/%d/comments", issueNumber), nil, &ghComments); err != nil {
		return nil, err
	}
	comments := make([]Comment, len(ghComments))
	for i, gh := range ghComments {
		comments[i] = Comment{
			ID:     gh.ID,
			Author: gh.User.Login,
			Body:   gh.Body,
			Date:   gh.CreatedAt,
		}
	}
	return comments, nil
}

// --- Commits ---

type ghCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
			Date string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

func (c *RESTClient) ListCommits(ctx context.Context, branch string, limit int) ([]CommitInfo, error) {
	u := c.url("/commits?sha=%s&per_page=%d", branch, limit)
	var ghCommits []ghCommit
	if err := c.do(ctx, http.MethodGet, u, nil, &ghCommits); err != nil {
		return nil, err
	}
	commits := make([]CommitInfo, len(ghCommits))
	for i, gh := range ghCommits {
		commits[i] = CommitInfo{
			SHA:     gh.SHA,
			Message: gh.Commit.Message,
			Author:  gh.Commit.Author.Name,
			Date:    gh.Commit.Author.Date,
		}
	}
	return commits, nil
}
