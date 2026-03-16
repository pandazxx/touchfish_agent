package testrunner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"crypto/rand"
	"encoding/hex"
)

// DockerRunner implements TestRunner using Docker CLI (create → cp → start → wait → logs → rm).
type DockerRunner struct {
	image      string
	testCmd    []string
	timeoutSec int
}

func NewDockerRunner(image string, testCmd []string, timeoutSec int) *DockerRunner {
	return &DockerRunner{
		image:      image,
		testCmd:    testCmd,
		timeoutSec: timeoutSec,
	}
}

func (d *DockerRunner) Run(ctx context.Context, sourceDir string) (*TestResult, error) {
	containerID := "touchfish-test-" + randomID()
	start := time.Now()

	// docker create --name <id> --network none <image> <testCmd...>
	createArgs := []string{"create", "--name", containerID, "--network", "none", d.image}
	createArgs = append(createArgs, d.testCmd...)
	if err := d.runDocker(ctx, createArgs...); err != nil {
		return nil, fmt.Errorf("docker create: %w", err)
	}

	// Ensure cleanup.
	defer d.runDocker(context.Background(), "rm", "-f", containerID)

	// docker cp <sourceDir>/. <id>:/code
	if err := d.runDocker(ctx, "cp", sourceDir+"/.", containerID+":/code"); err != nil {
		return nil, fmt.Errorf("docker cp: %w", err)
	}

	// docker start <id>
	if err := d.runDocker(ctx, "start", containerID); err != nil {
		return nil, fmt.Errorf("docker start: %w", err)
	}

	// docker wait <id> (with timeout)
	waitCtx, waitCancel := context.WithTimeout(ctx, time.Duration(d.timeoutSec)*time.Second)
	defer waitCancel()

	exitCode, err := d.dockerWait(waitCtx, containerID)
	if err != nil {
		// Timeout: stop the container.
		_ = d.runDocker(context.Background(), "stop", "-t", "5", containerID)
		return nil, fmt.Errorf("docker wait: %w", err)
	}

	// docker logs <id>
	stdout, stderr, err := d.dockerLogs(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("docker logs: %w", err)
	}

	return &TestResult{
		Passed:   exitCode == 0,
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		Duration: time.Since(start),
	}, nil
}

func (d *DockerRunner) runDocker(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %s", err, stderr.String())
	}
	return nil
}

func (d *DockerRunner) dockerWait(ctx context.Context, containerID string) (int, error) {
	cmd := exec.CommandContext(ctx, "docker", "wait", containerID)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return -1, err
	}
	code, err := strconv.Atoi(strings.TrimSpace(stdout.String()))
	if err != nil {
		return -1, fmt.Errorf("parse exit code: %w", err)
	}
	return code, nil
}

func (d *DockerRunner) dockerLogs(ctx context.Context, containerID string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "docker", "logs", containerID)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", "", err
	}
	return stdout.String(), stderr.String(), nil
}

func randomID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
