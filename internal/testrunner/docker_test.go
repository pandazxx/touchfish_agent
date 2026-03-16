package testrunner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestDockerRunner_WithMockScript tests the DockerRunner using a mock docker script.
// The mock script simulates the Docker CLI commands.
func TestDockerRunner_WithMockScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}

	// Create a mock docker script.
	mockDir := t.TempDir()
	mockDocker := filepath.Join(mockDir, "docker")
	script := `#!/bin/sh
case "$1" in
  create)
    echo "container-created"
    ;;
  cp)
    # no-op
    ;;
  start)
    # no-op
    ;;
  wait)
    echo "0"
    ;;
  logs)
    echo "PASS: all tests"
    echo "warnings here" >&2
    ;;
  rm)
    # no-op
    ;;
  stop)
    # no-op
    ;;
esac
`
	if err := os.WriteFile(mockDocker, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	// Prepend mock dir to PATH.
	t.Setenv("PATH", mockDir+":"+os.Getenv("PATH"))

	runner := NewDockerRunner("test-image:latest", []string{"go", "test", "./..."}, 60)
	result, err := runner.Run(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.Passed {
		t.Error("expected Passed=true")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "PASS: all tests") {
		t.Errorf("expected stdout to contain test output, got %q", result.Stdout)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestDockerRunner_FailingTests(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}

	mockDir := t.TempDir()
	mockDocker := filepath.Join(mockDir, "docker")
	script := `#!/bin/sh
case "$1" in
  create) ;;
  cp) ;;
  start) ;;
  wait)
    echo "1"
    ;;
  logs)
    echo "FAIL: TestFoo" >&2
    ;;
  rm) ;;
  stop) ;;
esac
`
	if err := os.WriteFile(mockDocker, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", mockDir+":"+os.Getenv("PATH"))

	runner := NewDockerRunner("test-image:latest", []string{"make", "test"}, 60)
	result, err := runner.Run(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Passed {
		t.Error("expected Passed=false")
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestDockerRunner_CreateFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}

	mockDir := t.TempDir()
	mockDocker := filepath.Join(mockDir, "docker")
	script := `#!/bin/sh
case "$1" in
  create)
    echo "Error: image not found" >&2
    exit 1
    ;;
  rm) ;;
esac
`
	if err := os.WriteFile(mockDocker, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", mockDir+":"+os.Getenv("PATH"))

	runner := NewDockerRunner("bad-image:latest", []string{"test"}, 60)
	_, err := runner.Run(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("expected error when docker create fails")
	}
}

func TestRandomID(t *testing.T) {
	id1 := randomID()
	id2 := randomID()
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
	if len(id1) != 16 {
		t.Errorf("expected 16-char hex ID, got %q", id1)
	}
}
