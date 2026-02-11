package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pandazxx/touchfish_agent/internal/ai"
	"github.com/pandazxx/touchfish_agent/internal/contract"
	gitpkg "github.com/pandazxx/touchfish_agent/internal/git"
	ghpkg "github.com/pandazxx/touchfish_agent/internal/github"
	"github.com/pandazxx/touchfish_agent/internal/state"
	"github.com/pandazxx/touchfish_agent/internal/testrunner"
)

// SEAgent implements the software engineer agent loop.
type SEAgent struct {
	git        gitpkg.GitClient
	gh         ghpkg.GitHubClient
	ai         ai.AIAgent
	runner     testrunner.TestRunner
	store      state.StateStore
	teamName   string
	branch     string
	workDir    string
	loopInterval time.Duration
	maxRetries   int
	commitAuthor string
	commitEmail  string
}

// SEConfig holds the configuration for creating an SEAgent.
type SEConfig struct {
	Git          gitpkg.GitClient
	GitHub       ghpkg.GitHubClient
	AI           ai.AIAgent
	Runner       testrunner.TestRunner
	Store        state.StateStore
	TeamName     string
	Branch       string
	WorkDir      string
	LoopInterval time.Duration
	MaxRetries   int
	CommitAuthor string
	CommitEmail  string
}

func NewSEAgent(cfg SEConfig) *SEAgent {
	return &SEAgent{
		git:          cfg.Git,
		gh:           cfg.GitHub,
		ai:           cfg.AI,
		runner:       cfg.Runner,
		store:        cfg.Store,
		teamName:     cfg.TeamName,
		branch:       cfg.Branch,
		workDir:      cfg.WorkDir,
		loopInterval: cfg.LoopInterval,
		maxRetries:   cfg.MaxRetries,
		commitAuthor: cfg.CommitAuthor,
		commitEmail:  cfg.CommitEmail,
	}
}

// Run starts the SE agent loop, blocking until the context is cancelled.
func (se *SEAgent) Run(ctx context.Context) {
	ticker := time.NewTicker(se.loopInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := se.Tick(ctx); err != nil {
				log.Printf("[SE][%s] tick error: %v", se.teamName, err)
			}
		}
	}
}

// Tick executes a single SE iteration. Exported for testing.
func (se *SEAgent) Tick(ctx context.Context) error {
	// Pull latest.
	if _, err := se.git.Pull(ctx, se.workDir); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	teamState, err := se.store.Load(ctx, se.teamName)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	if teamState == nil || teamState.Session == nil {
		return nil // No active session.
	}

	// Priority 1: Issues labeled "Agent to fix".
	handled, err := se.handleIssues(ctx, teamState)
	if err != nil {
		return fmt.Errorf("handle issues: %w", err)
	}
	if handled {
		return nil
	}

	// Priority 2: Requirement diffs.
	handled, err = se.handleRequirementDiffs(ctx, teamState)
	if err != nil {
		return fmt.Errorf("handle requirement diffs: %w", err)
	}
	if handled {
		return nil
	}

	// Priority 3: Idle.
	return nil
}

func (se *SEAgent) handleIssues(ctx context.Context, ts *state.TeamState) (bool, error) {
	issues, err := se.gh.ListIssues(ctx, []string{"Agent to fix"})
	if err != nil {
		return false, fmt.Errorf("list issues: %w", err)
	}
	if len(issues) == 0 {
		return false, nil
	}

	issue := issues[0]
	ts.Session.SE.CurrentIssueNumber = issue.Number
	ts.Session.SE.Phase = "fixing_issue"
	_ = se.store.Save(ctx, ts)

	// Swap labels: "Agent to fix" → "Agent fixing".
	_ = se.gh.RemoveIssueLabel(ctx, issue.Number, "Agent to fix")
	_ = se.gh.AddIssueLabel(ctx, issue.Number, "Agent fixing")

	// Get issue details and comments.
	fullIssue, err := se.gh.GetIssue(ctx, issue.Number)
	if err != nil {
		return true, fmt.Errorf("get issue: %w", err)
	}
	comments, _ := se.gh.ListIssueComments(ctx, issue.Number)

	// Read project setup.
	projectSetup, _ := contract.ReadFileFromWorkspace(se.workDir, "PROJECT_SETUP.md")

	var commentParams []ai.CommentParam
	for _, c := range comments {
		commentParams = append(commentParams, ai.CommentParam{
			Author: c.Author,
			Date:   c.Date,
			Body:   c.Body,
		})
	}

	prompt, err := ai.RenderSEIssueFix(ai.SEIssueFixParams{
		ProjectSetup: string(projectSetup),
		IssueNumber:  fullIssue.Number,
		IssueTitle:   fullIssue.Title,
		IssueBody:    fullIssue.Body,
		Comments:     commentParams,
	})
	if err != nil {
		return true, fmt.Errorf("render prompt: %w", err)
	}

	// Invoke AI.
	_, err = se.ai.Invoke(ctx, se.workDir, prompt)
	if err != nil {
		return true, fmt.Errorf("AI invoke: %w", err)
	}

	// Test and commit.
	if err := se.testAndCommit(ctx, ts, fmt.Sprintf("Fix #%d: %s", issue.Number, issue.Title)); err != nil {
		return true, err
	}

	// Swap labels: "Agent fixing" → "Agent fixed to be verified".
	_ = se.gh.RemoveIssueLabel(ctx, issue.Number, "Agent fixing")
	_ = se.gh.AddIssueLabel(ctx, issue.Number, "Agent fixed to be verified")

	ts.Session.SE.CurrentIssueNumber = 0
	ts.Session.SE.Phase = "idle"
	_ = se.store.Save(ctx, ts)

	return true, nil
}

func (se *SEAgent) handleRequirementDiffs(ctx context.Context, ts *state.TeamState) (bool, error) {
	reqContent, _ := contract.ReadFileFromWorkspace(se.workDir, "REQUIREMENT.md")
	testReqContent, _ := contract.ReadFileFromWorkspace(se.workDir, "TEST_REQUIREMENTS.md")

	reqHash := ""
	if reqContent != nil {
		reqHash = contract.ContentHash(reqContent)
	}
	testReqHash := ""
	if testReqContent != nil {
		testReqHash = contract.ContentHash(testReqContent)
	}

	reqChanged := reqContent != nil && contract.HasChanged(reqContent, ts.Session.SE.LastRequirementHash)
	testReqChanged := testReqContent != nil && contract.HasChanged(testReqContent, ts.Session.SE.LastTestReqHash)

	if !reqChanged && !testReqChanged {
		return false, nil
	}

	ts.Session.SE.Phase = "implementing"
	_ = se.store.Save(ctx, ts)

	projectSetup, _ := contract.ReadFileFromWorkspace(se.workDir, "PROJECT_SETUP.md")

	prompt, err := ai.RenderSEImplementation(ai.SEImplementationParams{
		ProjectSetup:     string(projectSetup),
		Requirement:      string(reqContent),
		TestRequirements: string(testReqContent),
		ReqDiff:          "", // Full diff would require git operations; simplified for now.
		TestReqDiff:      "",
		ReqChanged:       reqChanged,
		TestReqChanged:   testReqChanged,
	})
	if err != nil {
		return true, fmt.Errorf("render prompt: %w", err)
	}

	_, err = se.ai.Invoke(ctx, se.workDir, prompt)
	if err != nil {
		return true, fmt.Errorf("AI invoke: %w", err)
	}

	if err := se.testAndCommit(ctx, ts, "Implement requirement changes"); err != nil {
		return true, err
	}

	ts.Session.SE.LastRequirementHash = reqHash
	ts.Session.SE.LastTestReqHash = testReqHash
	ts.Session.SE.Phase = "idle"
	_ = se.store.Save(ctx, ts)

	return true, nil
}

func (se *SEAgent) testAndCommit(ctx context.Context, ts *state.TeamState, message string) error {
	for attempt := 0; attempt <= se.maxRetries; attempt++ {
		result, err := se.runner.Run(ctx, se.workDir)
		if err != nil {
			return fmt.Errorf("test run: %w", err)
		}

		if result.Passed {
			if err := se.git.StageAll(ctx, se.workDir); err != nil {
				return fmt.Errorf("stage: %w", err)
			}
			hash, err := se.git.Commit(ctx, se.workDir, message, se.commitAuthor, se.commitEmail)
			if err != nil {
				return fmt.Errorf("commit: %w", err)
			}
			if err := se.git.Push(ctx, se.workDir); err != nil {
				return fmt.Errorf("push: %w", err)
			}
			ts.Session.SE.LastProcessedCommit = hash
			ts.Session.SE.ConsecutiveFailures = 0
			return nil
		}

		// Tests failed: invoke AI to fix.
		if attempt < se.maxRetries {
			prompt, _ := ai.RenderSETestFix(ai.SETestFixParams{
				TestStdout: result.Stdout,
				TestStderr: result.Stderr,
			})
			_, _ = se.ai.Invoke(ctx, se.workDir, prompt)
		}
	}

	// Exhausted retries: commit WIP and file escape issue.
	ts.Session.SE.ConsecutiveFailures++
	_ = se.git.StageAll(ctx, se.workDir)
	_, _ = se.git.Commit(ctx, se.workDir, "WIP: "+message+" (tests failing)", se.commitAuthor, se.commitEmail)
	_ = se.git.Push(ctx, se.workDir)

	_, _ = se.gh.CreateIssue(ctx,
		fmt.Sprintf("[SE][%s] Tests failing after %d retries", se.branch, se.maxRetries),
		fmt.Sprintf("SE agent exhausted %d test fix retries for: %s\nBranch: %s", se.maxRetries, message, se.branch),
		[]string{"Agent to fix"},
	)

	return nil
}
