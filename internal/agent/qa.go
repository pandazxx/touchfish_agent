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
)

// QAAgent implements the QA test strategist agent loop.
type QAAgent struct {
	git          gitpkg.GitClient
	gh           ghpkg.GitHubClient
	ai           ai.AIAgent
	store        state.StateStore
	teamName     string
	branch       string
	workDir      string
	loopInterval time.Duration
	commitAuthor string
	commitEmail  string
}

// QAConfig holds the configuration for creating a QAAgent.
type QAConfig struct {
	Git          gitpkg.GitClient
	GitHub       ghpkg.GitHubClient
	AI           ai.AIAgent
	Store        state.StateStore
	TeamName     string
	Branch       string
	WorkDir      string
	LoopInterval time.Duration
	CommitAuthor string
	CommitEmail  string
}

func NewQAAgent(cfg QAConfig) *QAAgent {
	return &QAAgent{
		git:          cfg.Git,
		gh:           cfg.GitHub,
		ai:           cfg.AI,
		store:        cfg.Store,
		teamName:     cfg.TeamName,
		branch:       cfg.Branch,
		workDir:      cfg.WorkDir,
		loopInterval: cfg.LoopInterval,
		commitAuthor: cfg.CommitAuthor,
		commitEmail:  cfg.CommitEmail,
	}
}

// Run starts the QA agent loop, blocking until the context is cancelled.
func (qa *QAAgent) Run(ctx context.Context) {
	ticker := time.NewTicker(qa.loopInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := qa.Tick(ctx); err != nil {
				log.Printf("[QA][%s] tick error: %v", qa.teamName, err)
			}
		}
	}
}

// Tick executes a single QA iteration. Exported for testing.
func (qa *QAAgent) Tick(ctx context.Context) error {
	if _, err := qa.git.Pull(ctx, qa.workDir); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	teamState, err := qa.store.Load(ctx, qa.teamName)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	if teamState == nil || teamState.Session == nil {
		return nil
	}

	// Priority 1: REQUIREMENT.md changed.
	handled, err := qa.handleRequirementChange(ctx, teamState)
	if err != nil {
		return fmt.Errorf("handle requirement change: %w", err)
	}
	if handled {
		return nil
	}

	// Priority 2: Review new SE commits.
	handled, err = qa.handleReview(ctx, teamState)
	if err != nil {
		return fmt.Errorf("handle review: %w", err)
	}
	if handled {
		return nil
	}

	// Idle.
	return nil
}

func (qa *QAAgent) handleRequirementChange(ctx context.Context, ts *state.TeamState) (bool, error) {
	reqContent, _ := contract.ReadFileFromWorkspace(qa.workDir, "REQUIREMENT.md")
	if reqContent == nil {
		return false, nil
	}

	if !contract.HasChanged(reqContent, ts.Session.QA.LastRequirementHash) {
		return false, nil
	}

	ts.Session.QA.Phase = "generating_test_requirements"
	_ = qa.store.Save(ctx, ts)

	projectSetup, _ := contract.ReadFileFromWorkspace(qa.workDir, "PROJECT_SETUP.md")
	existingTestReq, _ := contract.ReadFileFromWorkspace(qa.workDir, "TEST_REQUIREMENTS.md")

	prompt, err := ai.RenderQATestRequirements(ai.QATestRequirementsParams{
		Requirement:     string(reqContent),
		ProjectSetup:    string(projectSetup),
		ReqDiff:         "", // Simplified: full content used instead of diff.
		ExistingContent: string(existingTestReq),
		HasExisting:     existingTestReq != nil,
	})
	if err != nil {
		return true, fmt.Errorf("render prompt: %w", err)
	}

	_, err = qa.ai.Invoke(ctx, qa.workDir, prompt)
	if err != nil {
		return true, fmt.Errorf("AI invoke: %w", err)
	}

	// Commit and push TEST_REQUIREMENTS.md.
	if err := qa.git.StageAll(ctx, qa.workDir); err != nil {
		return true, fmt.Errorf("stage: %w", err)
	}
	_, err = qa.git.Commit(ctx, qa.workDir, "Update TEST_REQUIREMENTS.md based on requirement changes",
		qa.commitAuthor, qa.commitEmail)
	if err != nil {
		return true, fmt.Errorf("commit: %w", err)
	}
	if err := qa.git.Push(ctx, qa.workDir); err != nil {
		// Push conflict: pull and retry once.
		if _, pullErr := qa.git.Pull(ctx, qa.workDir); pullErr != nil {
			return true, fmt.Errorf("pull after push fail: %w", pullErr)
		}
		if pushErr := qa.git.Push(ctx, qa.workDir); pushErr != nil {
			return true, fmt.Errorf("push retry: %w", pushErr)
		}
	}

	ts.Session.QA.LastRequirementHash = contract.ContentHash(reqContent)
	ts.Session.QA.Phase = "idle"
	_ = qa.store.Save(ctx, ts)

	return true, nil
}

func (qa *QAAgent) handleReview(ctx context.Context, ts *state.TeamState) (bool, error) {
	currentHead, err := qa.git.HeadHash(ctx, qa.workDir)
	if err != nil {
		return false, fmt.Errorf("head hash: %w", err)
	}

	if ts.Session.QA.LastReviewedCommit == currentHead {
		return false, nil // No new commits.
	}

	// If this is the first review, skip (nothing to diff from).
	if ts.Session.QA.LastReviewedCommit == "" {
		ts.Session.QA.LastReviewedCommit = currentHead
		_ = qa.store.Save(ctx, ts)
		return false, nil
	}

	ts.Session.QA.Phase = "reviewing"
	_ = qa.store.Save(ctx, ts)

	fromHash := ts.Session.QA.LastReviewedCommit
	toHash := currentHead

	// Get changed files.
	diffs, err := qa.git.DiffFiles(ctx, qa.workDir, fromHash, toHash)
	if err != nil {
		return true, fmt.Errorf("diff files: %w", err)
	}
	if len(diffs) == 0 {
		ts.Session.QA.LastReviewedCommit = currentHead
		ts.Session.QA.Phase = "idle"
		_ = qa.store.Save(ctx, ts)
		return false, nil
	}

	// Read context files.
	testReqContent, _ := contract.ReadFileFromWorkspace(qa.workDir, "TEST_REQUIREMENTS.md")
	reqContent, _ := contract.ReadFileFromWorkspace(qa.workDir, "REQUIREMENT.md")
	projectSetup, _ := contract.ReadFileFromWorkspace(qa.workDir, "PROJECT_SETUP.md")

	var changedFiles []ai.ChangedFileParam
	for _, d := range diffs {
		changedFiles = append(changedFiles, ai.ChangedFileParam{
			Path:   d.Path,
			Change: d.Change,
		})
	}

	prompt, err := ai.RenderQAReview(ai.QAReviewParams{
		TestRequirements: string(testReqContent),
		Requirement:      string(reqContent),
		ProjectSetup:     string(projectSetup),
		FromHash:         fromHash,
		ToHash:           toHash,
		ChangedFiles:     changedFiles,
	})
	if err != nil {
		return true, fmt.Errorf("render prompt: %w", err)
	}

	result, err := qa.ai.Invoke(ctx, qa.workDir, prompt)
	if err != nil {
		return true, fmt.Errorf("AI invoke: %w", err)
	}

	// Parse findings and file issues.
	findings, _ := contract.ParseReviewFindings(result.Stdout)
	for _, f := range findings {
		if f.Type == "approved" {
			continue
		}

		title := contract.IssueForFinding(f, qa.branch)

		// Dedup: check existing issues.
		existingIssues, _ := qa.gh.ListIssues(ctx, []string{"Agent to fix"})
		alreadyFiled := false
		for _, existing := range existingIssues {
			if existing.Title == title {
				alreadyFiled = true
				break
			}
		}
		if alreadyFiled {
			continue
		}

		_, _ = qa.gh.CreateIssue(ctx, title, f.Detail, []string{"Agent to fix"})
	}

	ts.Session.QA.LastReviewedCommit = currentHead
	ts.Session.QA.Phase = "idle"
	_ = qa.store.Save(ctx, ts)

	return true, nil
}
