package state

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFileStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	ctx := context.Background()

	original := &TeamState{
		TeamName: "alpha",
		Session: &SessionState{
			Branch:    "agent/alpha/feature-1",
			PRNumber:  42,
			StartedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			SE: SEState{
				LastProcessedCommit: "abc123",
				Phase:               "implementing",
			},
			QA: QAState{
				LastReviewedCommit: "def456",
				Phase:              "reviewing",
			},
		},
	}

	if err := store.Save(ctx, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, "alpha")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("Load returned nil")
	}
	if loaded.TeamName != "alpha" {
		t.Errorf("expected teamName=alpha, got %q", loaded.TeamName)
	}
	if loaded.Session == nil {
		t.Fatal("loaded session is nil")
	}
	if loaded.Session.Branch != "agent/alpha/feature-1" {
		t.Errorf("expected branch=agent/alpha/feature-1, got %q", loaded.Session.Branch)
	}
	if loaded.Session.PRNumber != 42 {
		t.Errorf("expected prNumber=42, got %d", loaded.Session.PRNumber)
	}
	if loaded.Session.SE.LastProcessedCommit != "abc123" {
		t.Errorf("expected SE commit=abc123, got %q", loaded.Session.SE.LastProcessedCommit)
	}
	if loaded.Session.QA.LastReviewedCommit != "def456" {
		t.Errorf("expected QA commit=def456, got %q", loaded.Session.QA.LastReviewedCommit)
	}
	if loaded.LastUpdated.IsZero() {
		t.Error("expected LastUpdated to be set")
	}
}

func TestFileStore_LoadNonExistent(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	ctx := context.Background()

	loaded, err := store.Load(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil for non-existent team, got %+v", loaded)
	}
}

func TestFileStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	ctx := context.Background()

	state := &TeamState{TeamName: "todelete"}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete(ctx, "todelete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	loaded, err := store.Load(ctx, "todelete")
	if err != nil {
		t.Fatalf("Load after delete: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestFileStore_DeleteNonExistent(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	ctx := context.Background()

	if err := store.Delete(ctx, "nope"); err != nil {
		t.Fatalf("Delete non-existent: %v", err)
	}
}

func TestFileStore_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	ctx := context.Background()

	state := &TeamState{TeamName: "atomic"}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify no .tmp file remains
	matches, _ := filepath.Glob(filepath.Join(dir, "*.tmp"))
	if len(matches) > 0 {
		t.Errorf("temp file should not remain: %v", matches)
	}

	// Verify the final file exists
	if _, err := os.Stat(filepath.Join(dir, "atomic.json")); err != nil {
		t.Errorf("expected state file to exist: %v", err)
	}
}

func TestFileStore_NilSession(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	ctx := context.Background()

	state := &TeamState{TeamName: "nosession"}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, "nosession")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Session != nil {
		t.Error("expected nil session")
	}
}

func TestFileStore_ConcurrentSaves(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			state := &TeamState{
				TeamName: "concurrent",
				Session: &SessionState{
					Branch: "agent/concurrent/test",
				},
			}
			if err := store.Save(ctx, state); err != nil {
				t.Errorf("concurrent Save: %v", err)
			}
		}()
	}
	wg.Wait()

	loaded, err := store.Load(ctx, "concurrent")
	if err != nil {
		t.Fatalf("Load after concurrent saves: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil state after concurrent saves")
	}
	if loaded.Session.Branch != "agent/concurrent/test" {
		t.Errorf("unexpected branch: %q", loaded.Session.Branch)
	}
}

func TestFileStore_SaveCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "state")
	store := NewFileStore(dir)
	ctx := context.Background()

	state := &TeamState{TeamName: "mkdirtest"}
	if err := store.Save(ctx, state); err != nil {
		t.Fatalf("Save should create directory: %v", err)
	}

	loaded, err := store.Load(ctx, "mkdirtest")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil state")
	}
}
