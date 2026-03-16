package git

import (
	"context"
	"fmt"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// GoGitClient implements GitClient using go-git.
type GoGitClient struct {
	auth AuthConfig
}

func NewGoGitClient(auth AuthConfig) *GoGitClient {
	return &GoGitClient{auth: auth}
}

func (g *GoGitClient) httpAuth() *http.BasicAuth {
	if g.auth.Token == "" {
		return nil
	}
	return &http.BasicAuth{
		Username: "x-access-token",
		Password: g.auth.Token,
	}
}

func (g *GoGitClient) Clone(_ context.Context, repoURL, localPath string, auth AuthConfig) error {
	httpAuth := &http.BasicAuth{
		Username: "x-access-token",
		Password: auth.Token,
	}
	if auth.Token == "" {
		httpAuth = nil
	}

	_, err := gogit.PlainClone(localPath, false, &gogit.CloneOptions{
		URL:  repoURL,
		Auth: httpAuth,
	})
	if err != nil {
		return fmt.Errorf("clone: %w", err)
	}
	return nil
}

func (g *GoGitClient) Pull(_ context.Context, localPath string) (string, error) {
	repo, err := gogit.PlainOpen(localPath)
	if err != nil {
		return "", fmt.Errorf("open repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("worktree: %w", err)
	}

	err = wt.Pull(&gogit.PullOptions{
		Auth: g.httpAuth(),
	})
	if err != nil && err != gogit.NoErrAlreadyUpToDate {
		return "", fmt.Errorf("pull: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("head: %w", err)
	}
	return head.Hash().String(), nil
}

func (g *GoGitClient) Push(_ context.Context, localPath string) error {
	repo, err := gogit.PlainOpen(localPath)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	err = repo.Push(&gogit.PushOptions{
		Auth: g.httpAuth(),
	})
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	return nil
}

func (g *GoGitClient) ListRemoteBranches(_ context.Context, repoURL, prefix string, auth AuthConfig) ([]Ref, error) {
	httpAuth := &http.BasicAuth{
		Username: "x-access-token",
		Password: auth.Token,
	}
	if auth.Token == "" {
		httpAuth = nil
	}

	rem := gogit.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{repoURL},
	})

	refs, err := rem.List(&gogit.ListOptions{
		Auth: httpAuth,
	})
	if err != nil {
		return nil, fmt.Errorf("list remote: %w", err)
	}

	var result []Ref
	for _, r := range refs {
		name := r.Name().Short()
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		result = append(result, Ref{
			Name: name,
			Hash: r.Hash().String(),
		})
	}
	return result, nil
}

func (g *GoGitClient) HeadHash(_ context.Context, localPath string) (string, error) {
	repo, err := gogit.PlainOpen(localPath)
	if err != nil {
		return "", fmt.Errorf("open repo: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("head: %w", err)
	}
	return head.Hash().String(), nil
}

func (g *GoGitClient) DiffFiles(_ context.Context, localPath, fromHash, toHash string) ([]DiffEntry, error) {
	repo, err := gogit.PlainOpen(localPath)
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	fromCommit, err := repo.CommitObject(plumbing.NewHash(fromHash))
	if err != nil {
		return nil, fmt.Errorf("from commit: %w", err)
	}

	toCommit, err := repo.CommitObject(plumbing.NewHash(toHash))
	if err != nil {
		return nil, fmt.Errorf("to commit: %w", err)
	}

	fromTree, err := fromCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("from tree: %w", err)
	}

	toTree, err := toCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("to tree: %w", err)
	}

	changes, err := fromTree.Diff(toTree)
	if err != nil {
		return nil, fmt.Errorf("diff: %w", err)
	}

	var entries []DiffEntry
	for _, c := range changes {
		entry := DiffEntry{}
		from := c.From
		to := c.To

		switch {
		case from.Name == "" && to.Name != "":
			entry.Path = to.Name
			entry.Change = "Added"
		case from.Name != "" && to.Name == "":
			entry.Path = from.Name
			entry.Change = "Deleted"
		default:
			entry.Path = to.Name
			entry.Change = "Modified"
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (g *GoGitClient) FileContent(_ context.Context, localPath, filePath, hash string) ([]byte, error) {
	repo, err := gogit.PlainOpen(localPath)
	if err != nil {
		return nil, fmt.Errorf("open repo: %w", err)
	}

	commit, err := repo.CommitObject(plumbing.NewHash(hash))
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("tree: %w", err)
	}

	file, err := tree.File(filePath)
	if err != nil {
		return nil, fmt.Errorf("file %q: %w", filePath, err)
	}

	content, err := file.Contents()
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return []byte(content), nil
}

func (g *GoGitClient) StageAll(_ context.Context, localPath string) error {
	repo, err := gogit.PlainOpen(localPath)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	_, err = wt.Add(".")
	if err != nil {
		return fmt.Errorf("stage all: %w", err)
	}
	return nil
}

func (g *GoGitClient) Commit(_ context.Context, localPath, message, authorName, authorEmail string) (string, error) {
	repo, err := gogit.PlainOpen(localPath)
	if err != nil {
		return "", fmt.Errorf("open repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("worktree: %w", err)
	}

	hash, err := wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}
	return hash.String(), nil
}

func (g *GoGitClient) Checkout(_ context.Context, localPath, branch string) error {
	repo, err := gogit.PlainOpen(localPath)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("worktree: %w", err)
	}

	// Try to check out an existing branch first.
	err = wt.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil {
		// If it doesn't exist, create it.
		err = wt.Checkout(&gogit.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(branch),
			Create: true,
		})
		if err != nil {
			return fmt.Errorf("checkout %q: %w", branch, err)
		}
	}
	return nil
}

func (g *GoGitClient) CurrentBranch(_ context.Context, localPath string) (string, error) {
	repo, err := gogit.PlainOpen(localPath)
	if err != nil {
		return "", fmt.Errorf("open repo: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("head: %w", err)
	}

	if !head.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not a branch")
	}
	return head.Name().Short(), nil
}
