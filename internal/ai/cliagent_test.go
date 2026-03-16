package ai

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeMockScript(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCLIAgent_InvokeSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}
	dir := t.TempDir()
	script := writeMockScript(t, dir, "mock-ai", "#!/bin/sh\necho \"AI output: $@\"\n")

	agent := NewCLIAgent(script, nil, 30)
	result, err := agent.Invoke(context.Background(), dir, "implement feature X")
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Stdout == "" {
		t.Error("expected non-empty stdout")
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestCLIAgent_InvokeWithArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}
	dir := t.TempDir()
	script := writeMockScript(t, dir, "mock-ai", "#!/bin/sh\necho \"args: $@\"\n")

	agent := NewCLIAgent(script, []string{"--no-interactive", "--print"}, 30)
	result, err := agent.Invoke(context.Background(), dir, "test prompt")
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	expected := "args: --no-interactive --print test prompt\n"
	if result.Stdout != expected {
		t.Errorf("expected stdout=%q, got %q", expected, result.Stdout)
	}
}

func TestCLIAgent_InvokeNonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}
	dir := t.TempDir()
	script := writeMockScript(t, dir, "mock-ai", "#!/bin/sh\necho \"error\" >&2\nexit 1\n")

	agent := NewCLIAgent(script, nil, 30)
	result, err := agent.Invoke(context.Background(), dir, "bad prompt")
	if err != nil {
		t.Fatalf("Invoke should not error on non-zero exit: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
	if result.Stderr == "" {
		t.Error("expected non-empty stderr")
	}
}

func TestCLIAgent_InvokeWorkDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell scripts not supported on Windows")
	}
	dir := t.TempDir()
	script := writeMockScript(t, dir, "mock-ai", "#!/bin/sh\npwd\n")

	workDir := t.TempDir()
	agent := NewCLIAgent(script, nil, 30)
	result, err := agent.Invoke(context.Background(), workDir, "prompt")
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	// The output should contain the work directory path.
	// On macOS, /tmp may resolve to /private/tmp.
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestCLIAgent_InvokeBinaryNotFound(t *testing.T) {
	agent := NewCLIAgent("/nonexistent/binary", nil, 30)
	_, err := agent.Invoke(context.Background(), t.TempDir(), "prompt")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}
