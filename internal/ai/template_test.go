package ai

import (
	"strings"
	"testing"
)

func TestRenderSEIssueFix(t *testing.T) {
	result, err := RenderSEIssueFix(SEIssueFixParams{
		ProjectSetup: "Use Go 1.22",
		IssueNumber:  42,
		IssueTitle:   "Fix login bug",
		IssueBody:    "Login fails when password has special chars",
		Comments: []CommentParam{
			{Author: "user1", Date: "2024-01-01", Body: "Confirmed on Chrome"},
		},
	})
	if err != nil {
		t.Fatalf("RenderSEIssueFix: %v", err)
	}
	if !strings.Contains(result, "Use Go 1.22") {
		t.Error("expected project setup content")
	}
	if !strings.Contains(result, "#42: Fix login bug") {
		t.Error("expected issue title")
	}
	if !strings.Contains(result, "special chars") {
		t.Error("expected issue body")
	}
	if !strings.Contains(result, "**user1** (2024-01-01)") {
		t.Error("expected comment")
	}
	if !strings.Contains(result, "Do NOT commit") {
		t.Error("expected instructions")
	}
}

func TestRenderSEImplementation(t *testing.T) {
	result, err := RenderSEImplementation(SEImplementationParams{
		ProjectSetup:     "Go project",
		Requirement:      "Build a REST API",
		TestRequirements: "Test all endpoints",
		ReqDiff:          "+Added new endpoint",
		ReqChanged:       true,
		TestReqChanged:   false,
	})
	if err != nil {
		t.Fatalf("RenderSEImplementation: %v", err)
	}
	if !strings.Contains(result, "Go project") {
		t.Error("expected project setup")
	}
	if !strings.Contains(result, "Build a REST API") {
		t.Error("expected requirement")
	}
	if !strings.Contains(result, "+Added new endpoint") {
		t.Error("expected req diff")
	}
	if !strings.Contains(result, "Do NOT commit") {
		t.Error("expected instructions")
	}
}

func TestRenderSEImplementation_NoTestReq(t *testing.T) {
	result, err := RenderSEImplementation(SEImplementationParams{
		ProjectSetup: "Go project",
		Requirement:  "Build API",
		ReqChanged:   true,
	})
	if err != nil {
		t.Fatalf("RenderSEImplementation: %v", err)
	}
	if !strings.Contains(result, "No test requirements yet.") {
		t.Error("expected default test requirements text")
	}
}

func TestRenderSETestFix(t *testing.T) {
	result, err := RenderSETestFix(SETestFixParams{
		TestStdout: "FAIL: TestLogin",
		TestStderr: "exit status 1",
	})
	if err != nil {
		t.Fatalf("RenderSETestFix: %v", err)
	}
	if !strings.Contains(result, "FAIL: TestLogin") {
		t.Error("expected test stdout")
	}
	if !strings.Contains(result, "exit status 1") {
		t.Error("expected test stderr")
	}
}

func TestRenderQATestRequirements_New(t *testing.T) {
	result, err := RenderQATestRequirements(QATestRequirementsParams{
		Requirement:  "Build REST API",
		ProjectSetup: "Go project",
		ReqDiff:      "+Added auth",
		HasExisting:  false,
	})
	if err != nil {
		t.Fatalf("RenderQATestRequirements: %v", err)
	}
	if !strings.Contains(result, "Build REST API") {
		t.Error("expected requirement")
	}
	if !strings.Contains(result, "Create from scratch") {
		t.Error("expected 'Create from scratch' for new")
	}
}

func TestRenderQATestRequirements_Existing(t *testing.T) {
	result, err := RenderQATestRequirements(QATestRequirementsParams{
		Requirement:     "Build REST API",
		ProjectSetup:    "Go project",
		ReqDiff:         "+Added auth",
		ExistingContent: "Existing test reqs here",
		HasExisting:     true,
	})
	if err != nil {
		t.Fatalf("RenderQATestRequirements: %v", err)
	}
	if !strings.Contains(result, "Existing test reqs here") {
		t.Error("expected existing content")
	}
	if strings.Contains(result, "Create from scratch") {
		t.Error("should not contain 'Create from scratch' when existing")
	}
}

func TestRenderQAReview(t *testing.T) {
	result, err := RenderQAReview(QAReviewParams{
		TestRequirements: "Test all endpoints",
		Requirement:      "Build API",
		ProjectSetup:     "Go project",
		FromHash:         "abc123",
		ToHash:           "def456",
		ChangedFiles: []ChangedFileParam{
			{Path: "main.go", Change: "Modified"},
			{Path: "handler.go", Change: "Added"},
		},
	})
	if err != nil {
		t.Fatalf("RenderQAReview: %v", err)
	}
	if !strings.Contains(result, "abc123") || !strings.Contains(result, "def456") {
		t.Error("expected hash references")
	}
	if !strings.Contains(result, "- main.go (Modified)") {
		t.Error("expected changed file listing")
	}
	if !strings.Contains(result, "- handler.go (Added)") {
		t.Error("expected added file listing")
	}
	if !strings.Contains(result, "Only output JSON lines") {
		t.Error("expected JSON output instructions")
	}
}
