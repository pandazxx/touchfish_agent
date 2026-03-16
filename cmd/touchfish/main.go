package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pandazxx/touchfish_agent/internal/ai"
	"github.com/pandazxx/touchfish_agent/internal/config"
	gitpkg "github.com/pandazxx/touchfish_agent/internal/git"
	ghpkg "github.com/pandazxx/touchfish_agent/internal/github"
	"github.com/pandazxx/touchfish_agent/internal/orchestrator"
	"github.com/pandazxx/touchfish_agent/internal/state"
	"github.com/pandazxx/touchfish_agent/internal/testrunner"
)

func main() {
	configPath := flag.String("config", "/etc/touchfish/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := config.ResolveToken(cfg); err != nil {
		log.Fatalf("Failed to resolve GitHub token: %v", err)
	}

	// Wire dependencies.
	gitClient := gitpkg.NewGoGitClient(gitpkg.AuthConfig{Token: cfg.GitHub.Token})
	ghClient := ghpkg.NewRESTClient(cfg.GitHub.Owner, cfg.GitHub.Repo, cfg.GitHub.Token)
	aiAgent := ai.NewCLIAgent(cfg.AI.BinaryPath, cfg.AI.DefaultArgs, cfg.AI.TimeoutSec)
	runner := testrunner.NewDockerRunner(cfg.TestRunner.Image, cfg.TestRunner.TestCmd, cfg.TestRunner.TimeoutSec)
	store := state.NewFileStore(cfg.State.Dir)

	orch := orchestrator.New(cfg, gitClient, ghClient, aiAgent, runner, store)

	// Set up signal handling for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	log.Printf("Touchfish Agent starting with %d team(s)", len(cfg.Teams))
	orch.Run(ctx)
	log.Println("Touchfish Agent stopped")
}
