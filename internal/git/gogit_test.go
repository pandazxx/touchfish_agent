package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gogitcfg "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// initBareAndClone creates a bare repo and clones it to a work dir.
// Returns (barePath, clonePath).
func initBareAndClone(t *testing.T) (string, string) {
	t.Helper()
	bare := filepath.Join(t.TempDir(), "bare.git")
	_, err := gogit.PlainInit(bare, true)
	if err != nil {
		t.Fatalf("init bare: %v", err)
	}

	clone := filepath.Join(t.TempDir(), "work")
	repo, err := gogit.PlainClone(clone, false, &gogit.CloneOptions{
		URL: bare,
	})
	if err != nil {
		// Bare repo has no commits; clone will fail. Init + add remote instead.
		repo, err = gogit.PlainInit(clone, false)
		if err != nil {
			t.Fatalf("init clone: %v", err)
		}
		_, err = repo.CreateRemote(&gogitcfg.RemoteConfig{
			Name: "origin",
			URLs: []string{bare},
		})
		if err != nil {
			t.Fatalf("add remote: %v", err)
		}
	}

	// Create initial commit so HEAD exists.
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	readmePath := filepath.Join(clone, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		t.Fatalf("add: %v", err)
	}
	_, err = wt.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Push to bare so it has the commit.
	if err := repo.Push(&gogit.PushOptions{}); err != nil {
		t.Fatalf("push to bare: %v", err)
	}

	return bare, clone
}

func TestGoGitClient_HeadHash(t *testing.T) {
	_, clone := initBareAndClone(t)
	client := NewGoGitClient(AuthConfig{})
	ctx := context.Background()

	hash, err := client.HeadHash(ctx, clone)
	if err != nil {
		t.Fatalf("HeadHash: %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("expected 40-char hash, got %q", hash)
	}
}

func TestGoGitClient_CurrentBranch(t *testing.T) {
	_, clone := initBareAndClone(t)
	client := NewGoGitClient(AuthConfig{})
	ctx := context.Background()

	branch, err := client.CurrentBranch(ctx, clone)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "master" {
		t.Errorf("expected master, got %q", branch)
	}
}

func TestGoGitClient_Checkout(t *testing.T) {
	_, clone := initBareAndClone(t)
	client := NewGoGitClient(AuthConfig{})
	ctx := context.Background()

	if err := client.Checkout(ctx, clone, "feature-1"); err != nil {
		t.Fatalf("Checkout: %v", err)
	}

	branch, err := client.CurrentBranch(ctx, clone)
	if err != nil {
		t.Fatalf("CurrentBranch: %v", err)
	}
	if branch != "feature-1" {
		t.Errorf("expected feature-1, got %q", branch)
	}

	// Checkout back to master (existing branch).
	if err := client.Checkout(ctx, clone, "master"); err != nil {
		t.Fatalf("Checkout back: %v", err)
	}
	branch, _ = client.CurrentBranch(ctx, clone)
	if branch != "master" {
		t.Errorf("expected master, got %q", branch)
	}
}

func TestGoGitClient_StageAllAndCommit(t *testing.T) {
	_, clone := initBareAndClone(t)
	client := NewGoGitClient(AuthConfig{})
	ctx := context.Background()

	// Write a new file.
	if err := os.WriteFile(filepath.Join(clone, "new.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := client.StageAll(ctx, clone); err != nil {
		t.Fatalf("StageAll: %v", err)
	}

	hash, err := client.Commit(ctx, clone, "add new file", "Test", "test@test.com")
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("expected 40-char hash, got %q", hash)
	}

	// Verify HEAD changed.
	headHash, _ := client.HeadHash(ctx, clone)
	if headHash != hash {
		t.Errorf("expected HEAD=%s, got %s", hash, headHash)
	}
}

func TestGoGitClient_DiffFiles(t *testing.T) {
	_, clone := initBareAndClone(t)
	client := NewGoGitClient(AuthConfig{})
	ctx := context.Background()

	fromHash, _ := client.HeadHash(ctx, clone)

	// Create and commit a new file.
	if err := os.WriteFile(filepath.Join(clone, "added.txt"), []byte("content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := client.StageAll(ctx, clone); err != nil {
		t.Fatal(err)
	}
	toHash, err := client.Commit(ctx, clone, "add file", "Test", "test@test.com")
	if err != nil {
		t.Fatal(err)
	}

	entries, err := client.DiffFiles(ctx, clone, fromHash, toHash)
	if err != nil {
		t.Fatalf("DiffFiles: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d: %+v", len(entries), entries)
	}
	if entries[0].Path != "added.txt" {
		t.Errorf("expected path=added.txt, got %q", entries[0].Path)
	}
	if entries[0].Change != "Added" {
		t.Errorf("expected change=Added, got %q", entries[0].Change)
	}
}

func TestGoGitClient_FileContent(t *testing.T) {
	_, clone := initBareAndClone(t)
	client := NewGoGitClient(AuthConfig{})
	ctx := context.Background()

	hash, _ := client.HeadHash(ctx, clone)
	content, err := client.FileContent(ctx, clone, "README.md", hash)
	if err != nil {
		t.Fatalf("FileContent: %v", err)
	}
	if string(content) != "# Test\n" {
		t.Errorf("expected '# Test\\n', got %q", string(content))
	}
}

func TestGoGitClient_CloneFromBare(t *testing.T) {
	bare, _ := initBareAndClone(t)
	client := NewGoGitClient(AuthConfig{})
	ctx := context.Background()

	cloneDst := filepath.Join(t.TempDir(), "clone2")
	if err := client.Clone(ctx, bare, cloneDst, AuthConfig{}); err != nil {
		t.Fatalf("Clone: %v", err)
	}

	hash, err := client.HeadHash(ctx, cloneDst)
	if err != nil {
		t.Fatalf("HeadHash on clone: %v", err)
	}
	if len(hash) != 40 {
		t.Errorf("expected 40-char hash, got %q", hash)
	}
}

func TestGoGitClient_PushAndPull(t *testing.T) {
	bare, clone := initBareAndClone(t)
	client := NewGoGitClient(AuthConfig{})
	ctx := context.Background()

	// Add a commit and push.
	if err := os.WriteFile(filepath.Join(clone, "pushed.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = client.StageAll(ctx, clone)
	_, _ = client.Commit(ctx, clone, "push test", "Test", "test@test.com")
	if err := client.Push(ctx, clone); err != nil {
		t.Fatalf("Push: %v", err)
	}

	// Clone again and verify the pushed file is there.
	clone2 := filepath.Join(t.TempDir(), "clone2")
	if err := client.Clone(ctx, bare, clone2, AuthConfig{}); err != nil {
		t.Fatalf("Clone: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(clone2, "pushed.txt"))
	if err != nil {
		t.Fatalf("read pushed file: %v", err)
	}
	if string(data) != "data" {
		t.Errorf("expected 'data', got %q", string(data))
	}

	// Now make another commit in clone, push, then pull from clone2.
	if err := os.WriteFile(filepath.Join(clone, "another.txt"), []byte("more"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = client.StageAll(ctx, clone)
	_, _ = client.Commit(ctx, clone, "another commit", "Test", "test@test.com")
	_ = client.Push(ctx, clone)

	newHead, err := client.Pull(ctx, clone2)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if len(newHead) != 40 {
		t.Errorf("expected 40-char hash, got %q", newHead)
	}

	data, err = os.ReadFile(filepath.Join(clone2, "another.txt"))
	if err != nil {
		t.Fatalf("read pulled file: %v", err)
	}
	if string(data) != "more" {
		t.Errorf("expected 'more', got %q", string(data))
	}
}

func TestGoGitClient_ListRemoteBranches(t *testing.T) {
	bare, clone := initBareAndClone(t)
	client := NewGoGitClient(AuthConfig{})
	ctx := context.Background()

	// Create a branch and push it.
	_ = client.Checkout(ctx, clone, "agent/alpha/feature-1")
	if err := os.WriteFile(filepath.Join(clone, "feature.txt"), []byte("f"), 0644); err != nil {
		t.Fatal(err)
	}
	_ = client.StageAll(ctx, clone)
	_, _ = client.Commit(ctx, clone, "feature commit", "Test", "test@test.com")

	repo, _ := gogit.PlainOpen(clone)
	_ = repo.Push(&gogit.PushOptions{})

	// List with prefix filter.
	refs, err := client.ListRemoteBranches(ctx, bare, "agent/alpha/", AuthConfig{})
	if err != nil {
		t.Fatalf("ListRemoteBranches: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %+v", len(refs), refs)
	}
	if refs[0].Name != "agent/alpha/feature-1" {
		t.Errorf("expected name=agent/alpha/feature-1, got %q", refs[0].Name)
	}
}
