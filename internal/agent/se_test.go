package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pandazxx/touchfish_agent/internal/ai"
	gitpkg "github.com/pandazxx/touchfish_agent/internal/git"
	ghpkg "github.com/pandazxx/touchfish_agent/internal/github"
	"github.com/pandazxx/touchfish_agent/internal/state"
	"github.com/pandazxx/touchfish_agent/internal/testrunner"
)

// --- Mock types ---

type mockGitClient struct {
	pullHead    string
	pullErr     error
	pushErr     error
	headHash    string
	stageErr    error
	commitHash  string
	commitErr   error
	diffEntries []gitpkg.DiffEntry
	fileContent map[string][]byte
}

func (m *mockGitClient) Clone(_ context.Context, _, _ string, _ gitpkg.AuthConfig) error { return nil }
func (m *mockGitClient) Pull(_ context.Context, _ string) (string, error) {
	return m.pullHead, m.pullErr
}
func (m *mockGitClient) Push(_ context.Context, _ string) error   { return m.pushErr }
func (m *mockGitClient) HeadHash(_ context.Context, _ string) (string, error) {
	return m.headHash, nil
}
func (m *mockGitClient) DiffFiles(_ context.Context, _, _, _ string) ([]gitpkg.DiffEntry, error) {
	return m.diffEntries, nil
}
func (m *mockGitClient) FileContent(_ context.Context, _, filePath, _ string) ([]byte, error) {
	return m.fileContent[filePath], nil
}
func (m *mockGitClient) StageAll(_ context.Context, _ string) error { return m.stageErr }
func (m *mockGitClient) Commit(_ context.Context, _, _, _, _ string) (string, error) {
	return m.commitHash, m.commitErr
}
func (m *mockGitClient) Checkout(_ context.Context, _, _ string) error   { return nil }
func (m *mockGitClient) CurrentBranch(_ context.Context, _ string) (string, error) {
	return "test-branch", nil
}
func (m *mockGitClient) ListRemoteBranches(_ context.Context, _, _ string, _ gitpkg.AuthConfig) ([]gitpkg.Ref, error) {
	return nil, nil
}

type mockGitHubClient struct {
	issues           []ghpkg.Issue
	createdIssues    []ghpkg.Issue
	labelAdded       []string
	labelRemoved     []string
	commentsAdded    []string
	issueComments    []ghpkg.Comment
}

func (m *mockGitHubClient) CreatePR(_ context.Context, _, _, _, _ string) (*ghpkg.PullRequest, error) {
	return &ghpkg.PullRequest{Number: 1}, nil
}
func (m *mockGitHubClient) GetPR(_ context.Context, _ int) (*ghpkg.PullRequest, error) {
	return &ghpkg.PullRequest{Number: 1, State: "open"}, nil
}
func (m *mockGitHubClient) UpdatePR(_ context.Context, _ int, _, _ *string) error { return nil }
func (m *mockGitHubClient) ListPRs(_ context.Context, _ ghpkg.PRState, _ string) ([]ghpkg.PullRequest, error) {
	return nil, nil
}
func (m *mockGitHubClient) AddPRComment(_ context.Context, _ int, _ string) error { return nil }
func (m *mockGitHubClient) ListIssues(_ context.Context, _ []string) ([]ghpkg.Issue, error) {
	return m.issues, nil
}
func (m *mockGitHubClient) GetIssue(_ context.Context, number int) (*ghpkg.Issue, error) {
	for _, i := range m.issues {
		if i.Number == number {
			return &i, nil
		}
	}
	return &ghpkg.Issue{Number: number}, nil
}
func (m *mockGitHubClient) CreateIssue(_ context.Context, title, body string, labels []string) (*ghpkg.Issue, error) {
	issue := ghpkg.Issue{Number: len(m.createdIssues) + 100, Title: title, Body: body, Labels: labels}
	m.createdIssues = append(m.createdIssues, issue)
	return &issue, nil
}
func (m *mockGitHubClient) AddIssueLabel(_ context.Context, _ int, label string) error {
	m.labelAdded = append(m.labelAdded, label)
	return nil
}
func (m *mockGitHubClient) RemoveIssueLabel(_ context.Context, _ int, label string) error {
	m.labelRemoved = append(m.labelRemoved, label)
	return nil
}
func (m *mockGitHubClient) AddIssueComment(_ context.Context, _ int, body string) error {
	m.commentsAdded = append(m.commentsAdded, body)
	return nil
}
func (m *mockGitHubClient) ListIssueComments(_ context.Context, _ int) ([]ghpkg.Comment, error) {
	return m.issueComments, nil
}
func (m *mockGitHubClient) ListCommits(_ context.Context, _ string, _ int) ([]ghpkg.CommitInfo, error) {
	return nil, nil
}

type mockAIAgent struct {
	invokeCount int
	result      *ai.InvokeResult
}

func (m *mockAIAgent) Invoke(_ context.Context, _, _ string) (*ai.InvokeResult, error) {
	m.invokeCount++
	if m.result != nil {
		return m.result, nil
	}
	return &ai.InvokeResult{ExitCode: 0, Stdout: "done"}, nil
}

type mockTestRunner struct {
	results []*testrunner.TestResult
	callIdx int
}

func (m *mockTestRunner) Run(_ context.Context, _ string) (*testrunner.TestResult, error) {
	if m.callIdx < len(m.results) {
		r := m.results[m.callIdx]
		m.callIdx++
		return r, nil
	}
	return &testrunner.TestResult{Passed: true}, nil
}

func newTestSE(t *testing.T, git *mockGitClient, gh *mockGitHubClient, aiAgent *mockAIAgent, runner *mockTestRunner) (*SEAgent, string) {
	t.Helper()
	dir := t.TempDir()
	stateDir := t.TempDir()
	store := state.NewFileStore(stateDir)

	se := NewSEAgent(SEConfig{
		Git:          git,
		GitHub:       gh,
		AI:           aiAgent,
		Runner:       runner,
		Store:        store,
		TeamName:     "alpha",
		Branch:       "agent/alpha/feature",
		WorkDir:      dir,
		LoopInterval: time.Second,
		MaxRetries:   2,
		CommitAuthor: "SE Agent",
		CommitEmail:  "se@test.dev",
	})

	// Create initial state with an active session.
	ctx := context.Background()
	_ = store.Save(ctx, &state.TeamState{
		TeamName: "alpha",
		Session: &state.SessionState{
			Branch:   "agent/alpha/feature",
			PRNumber: 1,
		},
	})

	return se, dir
}

func TestSETick_Idle(t *testing.T) {
	git := &mockGitClient{pullHead: "abc123"}
	gh := &mockGitHubClient{issues: nil}
	aiAgent := &mockAIAgent{}
	runner := &mockTestRunner{}

	se, _ := newTestSE(t, git, gh, aiAgent, runner)
	err := se.Tick(context.Background())
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if aiAgent.invokeCount != 0 {
		t.Errorf("expected 0 AI invocations for idle, got %d", aiAgent.invokeCount)
	}
}

func TestSETick_IssuePriority(t *testing.T) {
	git := &mockGitClient{pullHead: "abc123", commitHash: "def456"}
	gh := &mockGitHubClient{
		issues: []ghpkg.Issue{
			{Number: 10, Title: "Fix bug", Body: "Bug details", Labels: []string{"Agent to fix"}},
		},
	}
	aiAgent := &mockAIAgent{}
	runner := &mockTestRunner{
		results: []*testrunner.TestResult{{Passed: true}},
	}

	se, dir := newTestSE(t, git, gh, aiAgent, runner)
	// Create PROJECT_SETUP.md.
	os.WriteFile(filepath.Join(dir, "PROJECT_SETUP.md"), []byte("Go project"), 0644)

	err := se.Tick(context.Background())
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if aiAgent.invokeCount != 1 {
		t.Errorf("expected 1 AI invocation, got %d", aiAgent.invokeCount)
	}
	if len(gh.labelRemoved) < 1 || gh.labelRemoved[0] != "Agent to fix" {
		t.Errorf("expected 'Agent to fix' to be removed, got %v", gh.labelRemoved)
	}
	if len(gh.labelAdded) < 2 || gh.labelAdded[1] != "Agent fixed to be verified" {
		t.Errorf("expected 'Agent fixed to be verified' to be added, got %v", gh.labelAdded)
	}
}

func TestSETick_RequirementDiff(t *testing.T) {
	git := &mockGitClient{pullHead: "abc123", commitHash: "def456"}
	gh := &mockGitHubClient{issues: nil} // No issues → falls to priority 2.
	aiAgent := &mockAIAgent{}
	runner := &mockTestRunner{
		results: []*testrunner.TestResult{{Passed: true}},
	}

	se, dir := newTestSE(t, git, gh, aiAgent, runner)
	// Create REQUIREMENT.md so hash differs from stored empty hash.
	os.WriteFile(filepath.Join(dir, "REQUIREMENT.md"), []byte("Build REST API"), 0644)

	err := se.Tick(context.Background())
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if aiAgent.invokeCount != 1 {
		t.Errorf("expected 1 AI invocation, got %d", aiAgent.invokeCount)
	}
}

func TestSETick_TestRetry(t *testing.T) {
	git := &mockGitClient{pullHead: "abc123", commitHash: "def456"}
	gh := &mockGitHubClient{issues: nil}
	aiAgent := &mockAIAgent{}
	runner := &mockTestRunner{
		results: []*testrunner.TestResult{
			{Passed: false, ExitCode: 1, Stdout: "FAIL", Stderr: "error"},
			{Passed: true}, // Pass on retry.
		},
	}

	se, dir := newTestSE(t, git, gh, aiAgent, runner)
	os.WriteFile(filepath.Join(dir, "REQUIREMENT.md"), []byte("Build API"), 0644)

	err := se.Tick(context.Background())
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	// 1 implementation invoke + 1 test fix invoke.
	if aiAgent.invokeCount != 2 {
		t.Errorf("expected 2 AI invocations (implement + fix), got %d", aiAgent.invokeCount)
	}
}

func TestSETick_ExhaustedRetries(t *testing.T) {
	git := &mockGitClient{pullHead: "abc123", commitHash: "wip123"}
	gh := &mockGitHubClient{issues: nil}
	aiAgent := &mockAIAgent{}
	runner := &mockTestRunner{
		results: []*testrunner.TestResult{
			{Passed: false, Stdout: "FAIL 1"},
			{Passed: false, Stdout: "FAIL 2"},
			{Passed: false, Stdout: "FAIL 3"},
		},
	}

	se, dir := newTestSE(t, git, gh, aiAgent, runner)
	os.WriteFile(filepath.Join(dir, "REQUIREMENT.md"), []byte("API"), 0644)

	err := se.Tick(context.Background())
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	// Should have created an escape issue.
	if len(gh.createdIssues) != 1 {
		t.Errorf("expected 1 escape issue, got %d", len(gh.createdIssues))
	}
}
