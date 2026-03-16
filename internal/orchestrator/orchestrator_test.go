package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/pandazxx/touchfish_agent/internal/config"
	"github.com/pandazxx/touchfish_agent/internal/state"
)

func TestOrchestrator_GracefulShutdown(t *testing.T) {
	cfg := testConfig()
	cfg.Teams = []config.TeamConfig{{Name: "alpha"}, {Name: "beta"}}
	store := state.NewFileStore(t.TempDir())

	orch := New(cfg,
		&mockGitClient{},
		&mockGitHubClient{},
		&mockAIAgent{},
		&mockTestRunner{},
		store,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		orch.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
		// OK: graceful shutdown completed.
	case <-time.After(2 * time.Second):
		t.Fatal("orchestrator did not shut down within timeout")
	}
}
