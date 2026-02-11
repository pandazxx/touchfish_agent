package orchestrator

import (
	"context"
	"testing"

	"github.com/pandazxx/touchfish_agent/internal/ai"
	"github.com/pandazxx/touchfish_agent/internal/config"
	gitpkg "github.com/pandazxx/touchfish_agent/internal/git"
	ghpkg "github.com/pandazxx/touchfish_agent/internal/github"
	"github.com/pandazxx/touchfish_agent/internal/state"
	"github.com/pandazxx/touchfish_agent/internal/testrunner"
)

// --- Mocks ---

type mockGitClient struct {
	cloneErr    error
	pullHead    string
	pushErr     error
	headHash    string
	commitHash  string
	remoteBranches []gitpkg.Ref
	checkoutBranch string
}

func (m *mockGitClient) Clone(_ context.Context, _, _ string, _ gitpkg.AuthConfig) error {
	return m.cloneErr
}
func (m *mockGitClient) Pull(_ context.Context, _ string) (string, error) {
	return m.pullHead, nil
}
func (m *mockGitClient) Push(_ context.Context, _ string) error   { return m.pushErr }
func (m *mockGitClient) HeadHash(_ context.Context, _ string) (string, error) {
	return m.headHash, nil
}
func (m *mockGitClient) DiffFiles(_ context.Context, _, _, _ string) ([]gitpkg.DiffEntry, error) {
	return nil, nil
}
func (m *mockGitClient) FileContent(_ context.Context, _, _, _ string) ([]byte, error) {
	return nil, nil
}
func (m *mockGitClient) StageAll(_ context.Context, _ string) error { return nil }
func (m *mockGitClient) Commit(_ context.Context, _, _, _, _ string) (string, error) {
	return m.commitHash, nil
}
func (m *mockGitClient) Checkout(_ context.Context, _, branch string) error {
	m.checkoutBranch = branch
	return nil
}
func (m *mockGitClient) CurrentBranch(_ context.Context, _ string) (string, error) {
	return "master", nil
}
func (m *mockGitClient) ListRemoteBranches(_ context.Context, _, _ string, _ gitpkg.AuthConfig) ([]gitpkg.Ref, error) {
	return m.remoteBranches, nil
}

type mockGitHubClient struct {
	prs           []ghpkg.PullRequest
	createdPR     *ghpkg.PullRequest
	prState       string
	issues        []ghpkg.Issue
	createdIssues []ghpkg.Issue
}

func (m *mockGitHubClient) CreatePR(_ context.Context, _, _, _, _ string) (*ghpkg.PullRequest, error) {
	if m.createdPR != nil {
		return m.createdPR, nil
	}
	return &ghpkg.PullRequest{Number: 1}, nil
}
func (m *mockGitHubClient) GetPR(_ context.Context, _ int) (*ghpkg.PullRequest, error) {
	s := "open"
	if m.prState != "" {
		s = m.prState
	}
	return &ghpkg.PullRequest{Number: 1, State: s}, nil
}
func (m *mockGitHubClient) UpdatePR(_ context.Context, _ int, _, _ *string) error { return nil }
func (m *mockGitHubClient) ListPRs(_ context.Context, _ ghpkg.PRState, _ string) ([]ghpkg.PullRequest, error) {
	return m.prs, nil
}
func (m *mockGitHubClient) AddPRComment(_ context.Context, _ int, _ string) error { return nil }
func (m *mockGitHubClient) ListIssues(_ context.Context, _ []string) ([]ghpkg.Issue, error) {
	return m.issues, nil
}
func (m *mockGitHubClient) GetIssue(_ context.Context, n int) (*ghpkg.Issue, error) {
	return &ghpkg.Issue{Number: n}, nil
}
func (m *mockGitHubClient) CreateIssue(_ context.Context, title, body string, labels []string) (*ghpkg.Issue, error) {
	issue := ghpkg.Issue{Number: len(m.createdIssues) + 100, Title: title}
	m.createdIssues = append(m.createdIssues, issue)
	return &issue, nil
}
func (m *mockGitHubClient) AddIssueLabel(_ context.Context, _ int, _ string) error    { return nil }
func (m *mockGitHubClient) RemoveIssueLabel(_ context.Context, _ int, _ string) error { return nil }
func (m *mockGitHubClient) AddIssueComment(_ context.Context, _ int, _ string) error  { return nil }
func (m *mockGitHubClient) ListIssueComments(_ context.Context, _ int) ([]ghpkg.Comment, error) {
	return nil, nil
}
func (m *mockGitHubClient) ListCommits(_ context.Context, _ string, _ int) ([]ghpkg.CommitInfo, error) {
	return nil, nil
}

type mockAIAgent struct{}

func (m *mockAIAgent) Invoke(_ context.Context, _, _ string) (*ai.InvokeResult, error) {
	return &ai.InvokeResult{ExitCode: 0}, nil
}

type mockTestRunner struct{}

func (m *mockTestRunner) Run(_ context.Context, _ string) (*testrunner.TestResult, error) {
	return &testrunner.TestResult{Passed: true}, nil
}

func testConfig() *config.Config {
	return &config.Config{
		GitHub: config.GitHubConfig{
			Owner:      "testorg",
			Repo:       "testrepo",
			Token:      "test-token",
			BaseBranch: "master",
		},
		Teams:   []config.TeamConfig{{Name: "alpha"}},
		AI:      config.AIConfig{BinaryPath: "/usr/bin/echo", TimeoutSec: 30},
		TestRunner: config.TestRunnerConfig{Image: "test:latest", TestCmd: []string{"test"}, TimeoutSec: 30},
		Polling: config.PollingConfig{
			BranchScanIntervalSec: 1,
			AgentLoopIntervalSec:  1,
			IssueScanIntervalSec:  1,
		},
		Workspace: config.WorkspaceConfig{BaseDir: "/tmp/ws"},
		State:     config.StateConfig{Dir: "/tmp/state"},
		Agent:     config.AgentConfig{MaxTestRetries: 2, CommitAuthor: "Agent", CommitEmail: "agent@test.dev"},
	}
}

func TestTeamRunner_IdleToScanning_NoBranches(t *testing.T) {
	git := &mockGitClient{}
	gh := &mockGitHubClient{}
	store := state.NewFileStore(t.TempDir())
	cfg := testConfig()

	tr := newTeamRunner("alpha", cfg, git, gh, &mockAIAgent{}, &mockTestRunner{}, store)
	if tr.phase != PhaseIdle {
		t.Errorf("expected IDLE, got %s", tr.phase)
	}

	tr.tick(context.Background())
	// After scan with no branches, should return to IDLE.
	if tr.phase != PhaseIdle {
		t.Errorf("expected IDLE after no branches found, got %s", tr.phase)
	}
}

func TestTeamRunner_ScanningFindsSession(t *testing.T) {
	git := &mockGitClient{
		remoteBranches: []gitpkg.Ref{
			{Name: "agent/alpha/feature-1", Hash: "abc123"},
		},
	}
	gh := &mockGitHubClient{}
	cfg := testConfig()
	cfg.Workspace.BaseDir = t.TempDir()
	store := state.NewFileStore(t.TempDir())

	tr := newTeamRunner("alpha", cfg, git, gh, &mockAIAgent{}, &mockTestRunner{}, store)
	tr.tick(context.Background())

	if tr.phase != PhaseSessionActive {
		t.Errorf("expected SESSION_ACTIVE, got %s", tr.phase)
	}
}

func TestTeamRunner_SessionEndingOnClosedPR(t *testing.T) {
	git := &mockGitClient{}
	gh := &mockGitHubClient{prState: "closed"}
	cfg := testConfig()
	store := state.NewFileStore(t.TempDir())

	// Pre-create session state.
	ctx := context.Background()
	_ = store.Save(ctx, &state.TeamState{
		TeamName: "alpha",
		Session: &state.SessionState{
			Branch:   "agent/alpha/feature",
			PRNumber: 1,
		},
	})

	tr := newTeamRunner("alpha", cfg, git, gh, &mockAIAgent{}, &mockTestRunner{}, store)
	tr.phase = PhaseSessionActive

	tr.tick(ctx)
	if tr.phase != PhaseSessionEnding {
		t.Errorf("expected SESSION_ENDING, got %s", tr.phase)
	}

	// Another tick should move to IDLE.
	tr.tick(ctx)
	if tr.phase != PhaseIdle {
		t.Errorf("expected IDLE after ending, got %s", tr.phase)
	}
}

func TestTeamRunner_RecoverState_ValidSession(t *testing.T) {
	git := &mockGitClient{}
	gh := &mockGitHubClient{prState: "open"}
	cfg := testConfig()
	store := state.NewFileStore(t.TempDir())

	ctx := context.Background()
	_ = store.Save(ctx, &state.TeamState{
		TeamName: "alpha",
		Session: &state.SessionState{
			Branch:   "agent/alpha/feature",
			PRNumber: 1,
		},
	})

	tr := newTeamRunner("alpha", cfg, git, gh, &mockAIAgent{}, &mockTestRunner{}, store)
	tr.recoverState(ctx)

	if tr.phase != PhaseSessionActive {
		t.Errorf("expected SESSION_ACTIVE after recovery, got %s", tr.phase)
	}
}

func TestTeamRunner_RecoverState_ClosedPR(t *testing.T) {
	git := &mockGitClient{}
	gh := &mockGitHubClient{prState: "closed"}
	cfg := testConfig()
	store := state.NewFileStore(t.TempDir())

	ctx := context.Background()
	_ = store.Save(ctx, &state.TeamState{
		TeamName: "alpha",
		Session: &state.SessionState{
			Branch:   "agent/alpha/feature",
			PRNumber: 1,
		},
	})

	tr := newTeamRunner("alpha", cfg, git, gh, &mockAIAgent{}, &mockTestRunner{}, store)
	tr.recoverState(ctx)

	if tr.phase != PhaseIdle {
		t.Errorf("expected IDLE after recovery with closed PR, got %s", tr.phase)
	}

	// State should have session cleared.
	ts, _ := store.Load(ctx, "alpha")
	if ts.Session != nil {
		t.Error("expected nil session after recovery with closed PR")
	}
}

func TestTeamRunner_RecoverState_NoState(t *testing.T) {
	git := &mockGitClient{}
	gh := &mockGitHubClient{}
	cfg := testConfig()
	store := state.NewFileStore(t.TempDir())

	tr := newTeamRunner("alpha", cfg, git, gh, &mockAIAgent{}, &mockTestRunner{}, store)
	tr.recoverState(context.Background())

	if tr.phase != PhaseIdle {
		t.Errorf("expected IDLE when no state, got %s", tr.phase)
	}
}
