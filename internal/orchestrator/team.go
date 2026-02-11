package orchestrator

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/pandazxx/touchfish_agent/internal/agent"
	"github.com/pandazxx/touchfish_agent/internal/ai"
	"github.com/pandazxx/touchfish_agent/internal/config"
	gitpkg "github.com/pandazxx/touchfish_agent/internal/git"
	ghpkg "github.com/pandazxx/touchfish_agent/internal/github"
	"github.com/pandazxx/touchfish_agent/internal/state"
	"github.com/pandazxx/touchfish_agent/internal/testrunner"
)

// TeamPhase represents the current phase of a team's state machine.
type TeamPhase string

const (
	PhaseIdle          TeamPhase = "IDLE"
	PhaseScanning      TeamPhase = "SCANNING"
	PhaseSessionActive TeamPhase = "SESSION_ACTIVE"
	PhaseSessionEnding TeamPhase = "SESSION_ENDING"
)

// teamRunner manages the lifecycle of a single team.
type teamRunner struct {
	teamName string
	cfg      *config.Config
	git      gitpkg.GitClient
	gh       ghpkg.GitHubClient
	ai       ai.AIAgent
	runner   testrunner.TestRunner
	store    state.StateStore
	phase    TeamPhase
}

func newTeamRunner(teamName string, cfg *config.Config, git gitpkg.GitClient, gh ghpkg.GitHubClient,
	aiAgent ai.AIAgent, runner testrunner.TestRunner, store state.StateStore) *teamRunner {
	return &teamRunner{
		teamName: teamName,
		cfg:      cfg,
		git:      git,
		gh:       gh,
		ai:       aiAgent,
		runner:   runner,
		store:    store,
		phase:    PhaseIdle,
	}
}

// run is the main loop for a team goroutine.
func (tr *teamRunner) run(ctx context.Context) {
	scanInterval := time.Duration(tr.cfg.Polling.BranchScanIntervalSec) * time.Second
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	// Attempt crash recovery on startup.
	tr.recoverState(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tr.tick(ctx)
		}
	}
}

func (tr *teamRunner) tick(ctx context.Context) {
	switch tr.phase {
	case PhaseIdle:
		tr.phase = PhaseScanning
		tr.tickScanning(ctx)

	case PhaseScanning:
		tr.tickScanning(ctx)

	case PhaseSessionActive:
		tr.tickSessionActive(ctx)

	case PhaseSessionEnding:
		tr.tickSessionEnding(ctx)
	}
}

func (tr *teamRunner) tickScanning(ctx context.Context) {
	prefix := "agent/" + tr.teamName + "/"
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", tr.cfg.GitHub.Owner, tr.cfg.GitHub.Repo)

	refs, err := tr.git.ListRemoteBranches(ctx, repoURL, prefix, gitpkg.AuthConfig{Token: tr.cfg.GitHub.Token})
	if err != nil {
		log.Printf("[orchestrator][%s] scan error: %v", tr.teamName, err)
		tr.phase = PhaseIdle
		return
	}

	if len(refs) == 0 {
		tr.phase = PhaseIdle
		return
	}

	// Pick the first matching branch.
	branch := refs[0].Name
	log.Printf("[orchestrator][%s] found branch: %s", tr.teamName, branch)

	if err := tr.startSession(ctx, branch); err != nil {
		log.Printf("[orchestrator][%s] session start error: %v", tr.teamName, err)
		tr.phase = PhaseIdle
		return
	}

	tr.phase = PhaseSessionActive
}

func (tr *teamRunner) startSession(ctx context.Context, branch string) error {
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", tr.cfg.GitHub.Owner, tr.cfg.GitHub.Repo)
	auth := gitpkg.AuthConfig{Token: tr.cfg.GitHub.Token}

	seDir := filepath.Join(tr.cfg.Workspace.BaseDir, tr.teamName, "se")
	qaDir := filepath.Join(tr.cfg.Workspace.BaseDir, tr.teamName, "qa")

	// Clone for SE and QA.
	if err := tr.git.Clone(ctx, repoURL, seDir, auth); err != nil {
		return fmt.Errorf("clone SE: %w", err)
	}
	if err := tr.git.Checkout(ctx, seDir, branch); err != nil {
		return fmt.Errorf("checkout SE: %w", err)
	}

	if err := tr.git.Clone(ctx, repoURL, qaDir, auth); err != nil {
		return fmt.Errorf("clone QA: %w", err)
	}
	if err := tr.git.Checkout(ctx, qaDir, branch); err != nil {
		return fmt.Errorf("checkout QA: %w", err)
	}

	// Create PR if none exists.
	prs, err := tr.gh.ListPRs(ctx, ghpkg.PRStateOpen, branch)
	if err != nil {
		return fmt.Errorf("list PRs: %w", err)
	}

	var prNumber int
	if len(prs) > 0 {
		prNumber = prs[0].Number
	} else {
		pr, err := tr.gh.CreatePR(ctx, branch, tr.cfg.GitHub.BaseBranch,
			fmt.Sprintf("[%s] Agent work: %s", tr.teamName, branch), "Automated PR by Touchfish Agent")
		if err != nil {
			return fmt.Errorf("create PR: %w", err)
		}
		prNumber = pr.Number
	}

	// Save state.
	teamState := &state.TeamState{
		TeamName: tr.teamName,
		Session: &state.SessionState{
			Branch:    branch,
			PRNumber:  prNumber,
			StartedAt: time.Now().UTC(),
		},
	}
	if err := tr.store.Save(ctx, teamState); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	// Start SE and QA agents.
	loopInterval := time.Duration(tr.cfg.Polling.AgentLoopIntervalSec) * time.Second

	seAgent := agent.NewSEAgent(agent.SEConfig{
		Git:          tr.git,
		GitHub:       tr.gh,
		AI:           tr.ai,
		Runner:       tr.runner,
		Store:        tr.store,
		TeamName:     tr.teamName,
		Branch:       branch,
		WorkDir:      seDir,
		LoopInterval: loopInterval,
		MaxRetries:   tr.cfg.Agent.MaxTestRetries,
		CommitAuthor: tr.cfg.Agent.CommitAuthor,
		CommitEmail:  tr.cfg.Agent.CommitEmail,
	})

	qaAgent := agent.NewQAAgent(agent.QAConfig{
		Git:          tr.git,
		GitHub:       tr.gh,
		AI:           tr.ai,
		Store:        tr.store,
		TeamName:     tr.teamName,
		Branch:       branch,
		WorkDir:      qaDir,
		LoopInterval: loopInterval,
		CommitAuthor: tr.cfg.Agent.CommitAuthor,
		CommitEmail:  tr.cfg.Agent.CommitEmail,
	})

	go seAgent.Run(ctx)
	go qaAgent.Run(ctx)

	return nil
}

func (tr *teamRunner) tickSessionActive(ctx context.Context) {
	// Check PR status.
	ts, err := tr.store.Load(ctx, tr.teamName)
	if err != nil || ts == nil || ts.Session == nil {
		tr.phase = PhaseSessionEnding
		return
	}

	pr, err := tr.gh.GetPR(ctx, ts.Session.PRNumber)
	if err != nil {
		log.Printf("[orchestrator][%s] PR check error: %v", tr.teamName, err)
		return
	}

	if pr.State != "open" {
		log.Printf("[orchestrator][%s] PR #%d is %s, ending session", tr.teamName, ts.Session.PRNumber, pr.State)
		tr.phase = PhaseSessionEnding
	}
}

func (tr *teamRunner) tickSessionEnding(ctx context.Context) {
	// Clear session state.
	ts, _ := tr.store.Load(ctx, tr.teamName)
	if ts != nil {
		ts.Session = nil
		_ = tr.store.Save(ctx, ts)
	}

	tr.phase = PhaseIdle
	log.Printf("[orchestrator][%s] session ended, returning to idle", tr.teamName)
}

func (tr *teamRunner) recoverState(ctx context.Context) {
	ts, err := tr.store.Load(ctx, tr.teamName)
	if err != nil || ts == nil || ts.Session == nil {
		return
	}

	log.Printf("[orchestrator][%s] recovering session for branch %s", tr.teamName, ts.Session.Branch)

	// Verify PR is still open.
	pr, err := tr.gh.GetPR(ctx, ts.Session.PRNumber)
	if err != nil || pr.State != "open" {
		log.Printf("[orchestrator][%s] PR no longer open, clearing state", tr.teamName)
		ts.Session = nil
		_ = tr.store.Save(ctx, ts)
		return
	}

	// Resume session.
	tr.phase = PhaseSessionActive
}
