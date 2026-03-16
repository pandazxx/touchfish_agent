package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	GitHub     GitHubConfig     `yaml:"github"`
	Teams      []TeamConfig     `yaml:"teams"`
	AI         AIConfig         `yaml:"ai"`
	TestRunner TestRunnerConfig `yaml:"testRunner"`
	Polling    PollingConfig    `yaml:"polling"`
	Workspace  WorkspaceConfig  `yaml:"workspace"`
	State      StateConfig      `yaml:"state"`
	Agent      AgentConfig      `yaml:"agent"`
}

type GitHubConfig struct {
	Owner      string `yaml:"owner"`
	Repo       string `yaml:"repo"`
	Token      string `yaml:"token"`
	TokenEnv   string `yaml:"tokenEnv"`
	BaseBranch string `yaml:"baseBranch"`
}

type TeamConfig struct {
	Name string `yaml:"name"`
}

type AIConfig struct {
	BinaryPath  string   `yaml:"binaryPath"`
	DefaultArgs []string `yaml:"defaultArgs"`
	TimeoutSec  int      `yaml:"timeoutSec"`
}

type TestRunnerConfig struct {
	Image      string   `yaml:"image"`
	TestCmd    []string `yaml:"testCmd"`
	BuildCmd   []string `yaml:"buildCmd"`
	TimeoutSec int      `yaml:"timeoutSec"`
}

type PollingConfig struct {
	BranchScanIntervalSec int `yaml:"branchScanIntervalSec"`
	AgentLoopIntervalSec  int `yaml:"agentLoopIntervalSec"`
	IssueScanIntervalSec  int `yaml:"issueScanIntervalSec"`
}

type WorkspaceConfig struct {
	BaseDir string `yaml:"baseDir"`
}

type StateConfig struct {
	Dir string `yaml:"dir"`
}

type AgentConfig struct {
	MaxTestRetries int    `yaml:"maxTestRetries"`
	CommitAuthor   string `yaml:"commitAuthor"`
	CommitEmail    string `yaml:"commitEmail"`
}

// Load reads a YAML config file and returns a validated Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	applyDefaults(&cfg)

	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// ResolveToken sets cfg.GitHub.Token from the environment variable specified
// by cfg.GitHub.TokenEnv, if Token is not already set.
func ResolveToken(cfg *Config) error {
	if cfg.GitHub.Token != "" {
		return nil
	}
	if cfg.GitHub.TokenEnv == "" {
		return fmt.Errorf("github: either token or tokenEnv must be set")
	}
	token := os.Getenv(cfg.GitHub.TokenEnv)
	if token == "" {
		return fmt.Errorf("github: environment variable %q is empty", cfg.GitHub.TokenEnv)
	}
	cfg.GitHub.Token = token
	return nil
}

// Validate checks that all required fields are set.
func Validate(cfg *Config) error {
	if cfg.GitHub.Owner == "" {
		return fmt.Errorf("github.owner is required")
	}
	if cfg.GitHub.Repo == "" {
		return fmt.Errorf("github.repo is required")
	}
	if cfg.GitHub.Token == "" && cfg.GitHub.TokenEnv == "" {
		return fmt.Errorf("github: either token or tokenEnv must be set")
	}
	if len(cfg.Teams) == 0 {
		return fmt.Errorf("at least one team is required")
	}
	for i, t := range cfg.Teams {
		if t.Name == "" {
			return fmt.Errorf("teams[%d].name is required", i)
		}
	}
	if cfg.AI.BinaryPath == "" {
		return fmt.Errorf("ai.binaryPath is required")
	}
	if cfg.TestRunner.Image == "" {
		return fmt.Errorf("testRunner.image is required")
	}
	if len(cfg.TestRunner.TestCmd) == 0 {
		return fmt.Errorf("testRunner.testCmd is required")
	}
	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.GitHub.BaseBranch == "" {
		cfg.GitHub.BaseBranch = "master"
	}
	if cfg.AI.TimeoutSec == 0 {
		cfg.AI.TimeoutSec = 600
	}
	if cfg.TestRunner.TimeoutSec == 0 {
		cfg.TestRunner.TimeoutSec = 300
	}
	if cfg.Polling.BranchScanIntervalSec == 0 {
		cfg.Polling.BranchScanIntervalSec = 30
	}
	if cfg.Polling.AgentLoopIntervalSec == 0 {
		cfg.Polling.AgentLoopIntervalSec = 15
	}
	if cfg.Polling.IssueScanIntervalSec == 0 {
		cfg.Polling.IssueScanIntervalSec = 20
	}
	if cfg.Workspace.BaseDir == "" {
		cfg.Workspace.BaseDir = "/workspaces"
	}
	if cfg.State.Dir == "" {
		cfg.State.Dir = "/data/state"
	}
	if cfg.Agent.MaxTestRetries == 0 {
		cfg.Agent.MaxTestRetries = 3
	}
}
