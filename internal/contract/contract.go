package contract

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ContentHash returns the SHA-256 hex digest of the given content.
func ContentHash(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}

// HasChanged returns true if the current content hash differs from the stored hash.
func HasChanged(content []byte, storedHash string) bool {
	return ContentHash(content) != storedHash
}

// ReadFileFromWorkspace reads a file from the workspace directory.
// Returns the content and nil error, or nil content and nil error if the file doesn't exist.
func ReadFileFromWorkspace(workspaceDir, fileName string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(workspaceDir, fileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", fileName, err)
	}
	return data, nil
}

// ReviewFinding represents a single QA review finding parsed from JSON output.
type ReviewFinding struct {
	Type        string `json:"type"`
	Requirement string `json:"requirement,omitempty"`
	Feature     string `json:"feature,omitempty"`
	Detail      string `json:"detail"`
}

// ParseReviewFindings parses the QA agent's JSON-lines output into structured findings.
func ParseReviewFindings(output string) ([]ReviewFinding, error) {
	var findings []ReviewFinding
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var f ReviewFinding
		if err := json.Unmarshal([]byte(line), &f); err != nil {
			continue // Skip non-JSON lines.
		}
		if f.Type == "" {
			continue // Skip lines that don't have a type.
		}
		findings = append(findings, f)
	}
	return findings, nil
}

// IssueForFinding generates a GitHub issue title for a review finding.
func IssueForFinding(f ReviewFinding, branch string) string {
	subject := f.Requirement
	if subject == "" {
		subject = f.Feature
	}
	if subject == "" {
		subject = f.Detail
	}
	// Truncate long subjects.
	if len(subject) > 80 {
		subject = subject[:77] + "..."
	}
	return fmt.Sprintf("[QA][%s] %s: %s", branch, f.Type, subject)
}

// IssueMentionsBranch checks whether an issue title contains the branch name,
// used for deduplication of QA-filed issues.
func IssueMentionsBranch(issueTitle, branch string) bool {
	return strings.Contains(issueTitle, "["+branch+"]")
}
