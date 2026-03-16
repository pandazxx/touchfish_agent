package ai

import (
	"context"
	"time"
)

// InvokeResult holds the output from an AI agent invocation.
type InvokeResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// AIAgent abstracts the invocation of an AI CLI tool.
type AIAgent interface {
	Invoke(ctx context.Context, workDir string, prompt string) (*InvokeResult, error)
}
