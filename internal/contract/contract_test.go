package contract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContentHash(t *testing.T) {
	hash1 := ContentHash([]byte("hello"))
	hash2 := ContentHash([]byte("hello"))
	hash3 := ContentHash([]byte("world"))

	if hash1 != hash2 {
		t.Error("same content should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("different content should produce different hash")
	}
	if len(hash1) != 64 {
		t.Errorf("expected 64-char hex hash, got %d chars", len(hash1))
	}
}

func TestHasChanged(t *testing.T) {
	content := []byte("test content")
	hash := ContentHash(content)

	if HasChanged(content, hash) {
		t.Error("same content should not be flagged as changed")
	}
	if !HasChanged([]byte("modified"), hash) {
		t.Error("different content should be flagged as changed")
	}
}

func TestReadFileFromWorkspace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "REQUIREMENT.md"), []byte("# Reqs"), 0644); err != nil {
		t.Fatal(err)
	}

	data, err := ReadFileFromWorkspace(dir, "REQUIREMENT.md")
	if err != nil {
		t.Fatalf("ReadFileFromWorkspace: %v", err)
	}
	if string(data) != "# Reqs" {
		t.Errorf("expected '# Reqs', got %q", string(data))
	}
}

func TestReadFileFromWorkspace_NotExists(t *testing.T) {
	dir := t.TempDir()
	data, err := ReadFileFromWorkspace(dir, "NONEXISTENT.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Errorf("expected nil for non-existent file, got %q", string(data))
	}
}

func TestParseReviewFindings_Valid(t *testing.T) {
	output := `{"type": "missing_test", "requirement": "auth", "detail": "No login test"}
{"type": "wrong_test", "requirement": "api", "detail": "Wrong endpoint tested"}
{"type": "approved", "detail": "All good"}
`
	findings, err := ParseReviewFindings(output)
	if err != nil {
		t.Fatalf("ParseReviewFindings: %v", err)
	}
	if len(findings) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(findings))
	}
	if findings[0].Type != "missing_test" {
		t.Errorf("expected type=missing_test, got %q", findings[0].Type)
	}
	if findings[0].Requirement != "auth" {
		t.Errorf("expected requirement=auth, got %q", findings[0].Requirement)
	}
	if findings[2].Type != "approved" {
		t.Errorf("expected type=approved, got %q", findings[2].Type)
	}
}

func TestParseReviewFindings_MixedContent(t *testing.T) {
	output := `Some non-JSON preamble
{"type": "missing_coverage", "feature": "auth", "detail": "Missing edge case"}
More text here
`
	findings, err := ParseReviewFindings(output)
	if err != nil {
		t.Fatalf("ParseReviewFindings: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Feature != "auth" {
		t.Errorf("expected feature=auth, got %q", findings[0].Feature)
	}
}

func TestParseReviewFindings_Empty(t *testing.T) {
	findings, err := ParseReviewFindings("")
	if err != nil {
		t.Fatalf("ParseReviewFindings: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestIssueForFinding(t *testing.T) {
	f := ReviewFinding{
		Type:        "missing_test",
		Requirement: "user authentication",
		Detail:      "No test for password reset",
	}
	title := IssueForFinding(f, "agent/alpha/feature-1")
	if title != "[QA][agent/alpha/feature-1] missing_test: user authentication" {
		t.Errorf("unexpected title: %q", title)
	}
}

func TestIssueForFinding_FallbackToFeature(t *testing.T) {
	f := ReviewFinding{
		Type:    "missing_coverage",
		Feature: "auth module",
		Detail:  "Missing edge case",
	}
	title := IssueForFinding(f, "branch")
	if title != "[QA][branch] missing_coverage: auth module" {
		t.Errorf("unexpected title: %q", title)
	}
}

func TestIssueForFinding_FallbackToDetail(t *testing.T) {
	f := ReviewFinding{
		Type:   "approved",
		Detail: "All tests align",
	}
	title := IssueForFinding(f, "b")
	if title != "[QA][b] approved: All tests align" {
		t.Errorf("unexpected title: %q", title)
	}
}

func TestIssueMentionsBranch(t *testing.T) {
	if !IssueMentionsBranch("[QA][agent/alpha/f1] missing_test: auth", "agent/alpha/f1") {
		t.Error("expected match")
	}
	if IssueMentionsBranch("[QA][agent/beta/f1] missing_test: auth", "agent/alpha/f1") {
		t.Error("expected no match for different branch")
	}
	if IssueMentionsBranch("Regular issue title", "agent/alpha/f1") {
		t.Error("expected no match for non-QA issue")
	}
}
