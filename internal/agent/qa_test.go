package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pandazxx/touchfish_agent/internal/ai"
	"github.com/pandazxx/touchfish_agent/internal/contract"
	gitpkg "github.com/pandazxx/touchfish_agent/internal/git"
	ghpkg "github.com/pandazxx/touchfish_agent/internal/github"
	"github.com/pandazxx/touchfish_agent/internal/state"
)

func newTestQA(t *testing.T, git *mockGitClient, gh *mockGitHubClient, aiAgent *mockAIAgent) (*QAAgent, string) {
	t.Helper()
	dir := t.TempDir()
	stateDir := t.TempDir()
	store := state.NewFileStore(stateDir)

	qa := NewQAAgent(QAConfig{
		Git:          git,
		GitHub:       gh,
		AI:           aiAgent,
		Store:        store,
		TeamName:     "alpha",
		Branch:       "agent/alpha/feature",
		WorkDir:      dir,
		LoopInterval: time.Second,
		CommitAuthor: "QA Agent",
		CommitEmail:  "qa@test.dev",
	})

	ctx := context.Background()
	_ = store.Save(ctx, &state.TeamState{
		TeamName: "alpha",
		Session: &state.SessionState{
			Branch:   "agent/alpha/feature",
			PRNumber: 1,
		},
	})

	return qa, dir
}

func TestQATick_Idle(t *testing.T) {
	git := &mockGitClient{pullHead: "abc123", headHash: "abc123"}
	gh := &mockGitHubClient{}
	aiAgent := &mockAIAgent{}

	qa, _ := newTestQA(t, git, gh, aiAgent)
	err := qa.Tick(context.Background())
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if aiAgent.invokeCount != 0 {
		t.Errorf("expected 0 AI invocations for idle, got %d", aiAgent.invokeCount)
	}
}

func TestQATick_RequirementChanged(t *testing.T) {
	git := &mockGitClient{pullHead: "abc123", commitHash: "def456"}
	gh := &mockGitHubClient{}
	aiAgent := &mockAIAgent{}

	qa, dir := newTestQA(t, git, gh, aiAgent)
	os.WriteFile(filepath.Join(dir, "REQUIREMENT.md"), []byte("Build API"), 0644)
	os.WriteFile(filepath.Join(dir, "PROJECT_SETUP.md"), []byte("Go project"), 0644)

	err := qa.Tick(context.Background())
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if aiAgent.invokeCount != 1 {
		t.Errorf("expected 1 AI invocation, got %d", aiAgent.invokeCount)
	}
}

func TestQATick_ReviewNewCommits(t *testing.T) {
	git := &mockGitClient{
		pullHead: "newhead",
		headHash: "newhead",
		diffEntries: []gitpkg.DiffEntry{
			{Path: "main.go", Change: "Modified"},
		},
	}
	gh := &mockGitHubClient{}
	aiAgent := &mockAIAgent{
		result: &ai.InvokeResult{
			Stdout: `{"type": "missing_test", "requirement": "auth", "detail": "No login test"}`,
		},
	}

	qa, dir := newTestQA(t, git, gh, aiAgent)
	reqContent := []byte("Build API")
	os.WriteFile(filepath.Join(dir, "REQUIREMENT.md"), reqContent, 0644)
	os.WriteFile(filepath.Join(dir, "TEST_REQUIREMENTS.md"), []byte("Test all"), 0644)

	// Set lastReviewedCommit to a different hash so review triggers.
	store := qa.store
	ctx := context.Background()
	ts, _ := store.Load(ctx, "alpha")
	ts.Session.QA.LastReviewedCommit = "oldhead"
	// Set requirement hash to current content so it doesn't trigger requirement change.
	ts.Session.QA.LastRequirementHash = contract.ContentHash(reqContent)
	_ = store.Save(ctx, ts)

	err := qa.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if aiAgent.invokeCount != 1 {
		t.Errorf("expected 1 AI invocation for review, got %d", aiAgent.invokeCount)
	}
	if len(gh.createdIssues) != 1 {
		t.Errorf("expected 1 issue filed, got %d", len(gh.createdIssues))
	}
}

func TestQATick_ReviewDedup(t *testing.T) {
	git := &mockGitClient{
		pullHead: "newhead",
		headHash: "newhead",
		diffEntries: []gitpkg.DiffEntry{
			{Path: "main.go", Change: "Modified"},
		},
	}
	gh := &mockGitHubClient{
		// Already have the same issue filed.
		issues: []ghpkg.Issue{
			{Number: 100, Title: "[QA][agent/alpha/feature] missing_test: auth"},
		},
	}
	aiAgent := &mockAIAgent{
		result: &ai.InvokeResult{
			Stdout: `{"type": "missing_test", "requirement": "auth", "detail": "No login test"}`,
		},
	}

	qa, dir := newTestQA(t, git, gh, aiAgent)
	os.WriteFile(filepath.Join(dir, "TEST_REQUIREMENTS.md"), []byte("Test all"), 0644)

	store := qa.store
	ctx := context.Background()
	ts, _ := store.Load(ctx, "alpha")
	ts.Session.QA.LastReviewedCommit = "oldhead"
	// No REQUIREMENT.md file, so requirement change won't trigger.
	_ = store.Save(ctx, ts)

	err := qa.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	// Should NOT create a duplicate issue.
	if len(gh.createdIssues) != 0 {
		t.Errorf("expected 0 new issues (dedup), got %d", len(gh.createdIssues))
	}
}

func TestQATick_ReviewApproved(t *testing.T) {
	git := &mockGitClient{
		pullHead: "newhead",
		headHash: "newhead",
		diffEntries: []gitpkg.DiffEntry{
			{Path: "main.go", Change: "Modified"},
		},
	}
	gh := &mockGitHubClient{}
	aiAgent := &mockAIAgent{
		result: &ai.InvokeResult{
			Stdout: `{"type": "approved", "detail": "All tests align with requirements."}`,
		},
	}

	qa, dir := newTestQA(t, git, gh, aiAgent)
	os.WriteFile(filepath.Join(dir, "TEST_REQUIREMENTS.md"), []byte("Test all"), 0644)

	store := qa.store
	ctx := context.Background()
	ts, _ := store.Load(ctx, "alpha")
	ts.Session.QA.LastReviewedCommit = "oldhead"
	// No REQUIREMENT.md file, so requirement change won't trigger.
	_ = store.Save(ctx, ts)

	err := qa.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if len(gh.createdIssues) != 0 {
		t.Errorf("expected 0 issues for approved review, got %d", len(gh.createdIssues))
	}
}
