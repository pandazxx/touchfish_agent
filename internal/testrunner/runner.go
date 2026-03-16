package testrunner

import (
	"context"
	"time"
)

// TestResult holds the output from a test run.
type TestResult struct {
	Passed   bool
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// TestRunner abstracts test execution in an isolated environment.
type TestRunner interface {
	Run(ctx context.Context, sourceDir string) (*TestResult, error)
}
