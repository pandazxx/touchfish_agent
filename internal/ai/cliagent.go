package ai

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// CLIAgent implements AIAgent by invoking an external CLI binary via os/exec.
type CLIAgent struct {
	binaryPath  string
	defaultArgs []string
	timeout     time.Duration
}

func NewCLIAgent(binaryPath string, defaultArgs []string, timeoutSec int) *CLIAgent {
	return &CLIAgent{
		binaryPath:  binaryPath,
		defaultArgs: defaultArgs,
		timeout:     time.Duration(timeoutSec) * time.Second,
	}
}

func (a *CLIAgent) Invoke(ctx context.Context, workDir string, prompt string) (*InvokeResult, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	args := make([]string, len(a.defaultArgs))
	copy(args, a.defaultArgs)
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, a.binaryPath, args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := &InvokeResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return result, fmt.Errorf("running AI binary: %w", err)
		}
	}

	return result, nil
}
