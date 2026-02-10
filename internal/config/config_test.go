package config

import (
	"os"
	"path/filepath"
	"testing"
)

const validYAML = `
github:
  owner: "myorg"
  repo: "myproject"
  tokenEnv: "GITHUB_TOKEN"
teams:
  - name: "alpha"
ai:
  binaryPath: "/usr/local/bin/claude"
testRunner:
  image: "myproject-test:latest"
  testCmd: ["go", "test", "./..."]
`

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_Valid(t *testing.T) {
	path := writeConfig(t, validYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GitHub.Owner != "myorg" {
		t.Errorf("expected owner=myorg, got %q", cfg.GitHub.Owner)
	}
	if cfg.GitHub.BaseBranch != "master" {
		t.Errorf("expected default baseBranch=master, got %q", cfg.GitHub.BaseBranch)
	}
	if len(cfg.Teams) != 1 || cfg.Teams[0].Name != "alpha" {
		t.Errorf("unexpected teams: %+v", cfg.Teams)
	}
}

func TestLoad_Defaults(t *testing.T) {
	path := writeConfig(t, validYAML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AI.TimeoutSec != 600 {
		t.Errorf("expected ai.timeoutSec=600, got %d", cfg.AI.TimeoutSec)
	}
	if cfg.TestRunner.TimeoutSec != 300 {
		t.Errorf("expected testRunner.timeoutSec=300, got %d", cfg.TestRunner.TimeoutSec)
	}
	if cfg.Polling.BranchScanIntervalSec != 30 {
		t.Errorf("expected polling.branchScanIntervalSec=30, got %d", cfg.Polling.BranchScanIntervalSec)
	}
	if cfg.Polling.AgentLoopIntervalSec != 15 {
		t.Errorf("expected polling.agentLoopIntervalSec=15, got %d", cfg.Polling.AgentLoopIntervalSec)
	}
	if cfg.Polling.IssueScanIntervalSec != 20 {
		t.Errorf("expected polling.issueScanIntervalSec=20, got %d", cfg.Polling.IssueScanIntervalSec)
	}
	if cfg.Workspace.BaseDir != "/workspaces" {
		t.Errorf("expected workspace.baseDir=/workspaces, got %q", cfg.Workspace.BaseDir)
	}
	if cfg.State.Dir != "/data/state" {
		t.Errorf("expected state.dir=/data/state, got %q", cfg.State.Dir)
	}
	if cfg.Agent.MaxTestRetries != 3 {
		t.Errorf("expected agent.maxTestRetries=3, got %d", cfg.Agent.MaxTestRetries)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	yaml := `
github:
  owner: "myorg"
  repo: "myproject"
  tokenEnv: "GITHUB_TOKEN"
  baseBranch: "main"
teams:
  - name: "alpha"
ai:
  binaryPath: "/usr/local/bin/claude"
  timeoutSec: 1200
  defaultArgs: ["--no-interactive", "--print"]
testRunner:
  image: "test:latest"
  testCmd: ["make", "test"]
  timeoutSec: 600
polling:
  branchScanIntervalSec: 60
workspace:
  baseDir: "/tmp/ws"
state:
  dir: "/tmp/state"
agent:
  maxTestRetries: 5
  commitAuthor: "Bot"
  commitEmail: "bot@test.dev"
`
	path := writeConfig(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GitHub.BaseBranch != "main" {
		t.Errorf("expected baseBranch=main, got %q", cfg.GitHub.BaseBranch)
	}
	if cfg.AI.TimeoutSec != 1200 {
		t.Errorf("expected ai.timeoutSec=1200, got %d", cfg.AI.TimeoutSec)
	}
	if cfg.Polling.BranchScanIntervalSec != 60 {
		t.Errorf("expected branchScanIntervalSec=60, got %d", cfg.Polling.BranchScanIntervalSec)
	}
	if cfg.Workspace.BaseDir != "/tmp/ws" {
		t.Errorf("expected baseDir=/tmp/ws, got %q", cfg.Workspace.BaseDir)
	}
	if cfg.Agent.MaxTestRetries != 5 {
		t.Errorf("expected maxTestRetries=5, got %d", cfg.Agent.MaxTestRetries)
	}
	if cfg.Agent.CommitAuthor != "Bot" {
		t.Errorf("expected commitAuthor=Bot, got %q", cfg.Agent.CommitAuthor)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeConfig(t, "not: [valid: yaml")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestValidate_MissingOwner(t *testing.T) {
	path := writeConfig(t, `
github:
  repo: "myproject"
  tokenEnv: "GITHUB_TOKEN"
teams:
  - name: "alpha"
ai:
  binaryPath: "/usr/local/bin/claude"
testRunner:
  image: "test:latest"
  testCmd: ["make", "test"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing owner")
	}
}

func TestValidate_MissingToken(t *testing.T) {
	path := writeConfig(t, `
github:
  owner: "myorg"
  repo: "myproject"
teams:
  - name: "alpha"
ai:
  binaryPath: "/usr/local/bin/claude"
testRunner:
  image: "test:latest"
  testCmd: ["make", "test"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing token/tokenEnv")
	}
}

func TestValidate_NoTeams(t *testing.T) {
	path := writeConfig(t, `
github:
  owner: "myorg"
  repo: "myproject"
  tokenEnv: "GITHUB_TOKEN"
teams: []
ai:
  binaryPath: "/usr/local/bin/claude"
testRunner:
  image: "test:latest"
  testCmd: ["make", "test"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for no teams")
	}
}

func TestValidate_MissingAIBinary(t *testing.T) {
	path := writeConfig(t, `
github:
  owner: "myorg"
  repo: "myproject"
  tokenEnv: "GITHUB_TOKEN"
teams:
  - name: "alpha"
ai:
  binaryPath: ""
testRunner:
  image: "test:latest"
  testCmd: ["make", "test"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing ai.binaryPath")
	}
}

func TestValidate_MissingTestRunnerImage(t *testing.T) {
	path := writeConfig(t, `
github:
  owner: "myorg"
  repo: "myproject"
  tokenEnv: "GITHUB_TOKEN"
teams:
  - name: "alpha"
ai:
  binaryPath: "/usr/local/bin/claude"
testRunner:
  image: ""
  testCmd: ["make", "test"]
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing testRunner.image")
	}
}

func TestResolveToken_DirectToken(t *testing.T) {
	cfg := &Config{GitHub: GitHubConfig{Token: "direct-token"}}
	if err := ResolveToken(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GitHub.Token != "direct-token" {
		t.Errorf("expected token=direct-token, got %q", cfg.GitHub.Token)
	}
}

func TestResolveToken_FromEnv(t *testing.T) {
	t.Setenv("TEST_GH_TOKEN", "env-token-value")
	cfg := &Config{GitHub: GitHubConfig{TokenEnv: "TEST_GH_TOKEN"}}
	if err := ResolveToken(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GitHub.Token != "env-token-value" {
		t.Errorf("expected token=env-token-value, got %q", cfg.GitHub.Token)
	}
}

func TestResolveToken_MissingEnvVar(t *testing.T) {
	cfg := &Config{GitHub: GitHubConfig{TokenEnv: "NONEXISTENT_TOKEN_VAR"}}
	err := ResolveToken(cfg)
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
}

func TestResolveToken_NeitherSet(t *testing.T) {
	cfg := &Config{GitHub: GitHubConfig{}}
	err := ResolveToken(cfg)
	if err == nil {
		t.Fatal("expected error when neither token nor tokenEnv set")
	}
}
