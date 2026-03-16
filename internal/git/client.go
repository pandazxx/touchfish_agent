package git

import "context"

// AuthConfig holds credentials for git operations.
type AuthConfig struct {
	Token string
}

// Ref represents a git reference (branch or tag).
type Ref struct {
	Name string
	Hash string
}

// DiffEntry represents a file changed between two commits.
type DiffEntry struct {
	Path   string
	Change string // "Added", "Modified", "Deleted"
}

// GitClient abstracts git operations for cloning, branching, diffing, and committing.
type GitClient interface {
	Clone(ctx context.Context, repoURL, localPath string, auth AuthConfig) error
	Pull(ctx context.Context, localPath string) (newHead string, err error)
	Push(ctx context.Context, localPath string) error
	ListRemoteBranches(ctx context.Context, repoURL, prefix string, auth AuthConfig) ([]Ref, error)
	HeadHash(ctx context.Context, localPath string) (string, error)
	DiffFiles(ctx context.Context, localPath, fromHash, toHash string) ([]DiffEntry, error)
	FileContent(ctx context.Context, localPath, filePath, hash string) ([]byte, error)
	StageAll(ctx context.Context, localPath string) error
	Commit(ctx context.Context, localPath, message, authorName, authorEmail string) (hash string, err error)
	Checkout(ctx context.Context, localPath, branch string) error
	CurrentBranch(ctx context.Context, localPath string) (string, error)
}
