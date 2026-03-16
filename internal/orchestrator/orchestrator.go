package orchestrator

import (
	"context"
	"log"
	"sync"

	"github.com/pandazxx/touchfish_agent/internal/ai"
	"github.com/pandazxx/touchfish_agent/internal/config"
	gitpkg "github.com/pandazxx/touchfish_agent/internal/git"
	ghpkg "github.com/pandazxx/touchfish_agent/internal/github"
	"github.com/pandazxx/touchfish_agent/internal/state"
	"github.com/pandazxx/touchfish_agent/internal/testrunner"
)

// Orchestrator manages per-team goroutines and handles graceful shutdown.
type Orchestrator struct {
	cfg    *config.Config
	git    gitpkg.GitClient
	gh     ghpkg.GitHubClient
	ai     ai.AIAgent
	runner testrunner.TestRunner
	store  state.StateStore
}

func New(cfg *config.Config, git gitpkg.GitClient, gh ghpkg.GitHubClient,
	aiAgent ai.AIAgent, runner testrunner.TestRunner, store state.StateStore) *Orchestrator {
	return &Orchestrator{
		cfg:    cfg,
		git:    git,
		gh:     gh,
		ai:     aiAgent,
		runner: runner,
		store:  store,
	}
}

// Run starts a goroutine for each configured team and blocks until ctx is cancelled.
func (o *Orchestrator) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for _, team := range o.cfg.Teams {
		wg.Add(1)
		tr := newTeamRunner(team.Name, o.cfg, o.git, o.gh, o.ai, o.runner, o.store)
		go func(name string) {
			defer wg.Done()
			log.Printf("[orchestrator] starting team %s", name)
			tr.run(ctx)
			log.Printf("[orchestrator] team %s stopped", name)
		}(team.Name)
	}

	wg.Wait()
	log.Println("[orchestrator] all teams stopped")
}
