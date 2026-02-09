# Low-Level Design

## 1. Configuration Schema

### Example config.yaml

```yaml
github:
  owner: "myorg"
  repo: "myproject"
  tokenEnv: "GITHUB_TOKEN"
  baseBranch: "master"

teams:
  - name: "alpha"
  - name: "beta"

ai:
  binaryPath: "/usr/local/bin/claude"
  defaultArgs: ["--no-interactive", "--print"]
  timeoutSec: 600

testRunner:
  image: "myproject-test:latest"
  testCmd: ["bash", "-c", "cd /code && go test ./..."]
  buildCmd: []
  timeoutSec: 300

polling:
  branchScanIntervalSec: 30
  agentLoopIntervalSec: 15
  issueScanIntervalSec: 20

workspace:
  baseDir: "/workspaces"

state:
  dir: "/data/state"

agent:
  maxTestRetries: 3
  commitAuthor: "SE Agent"
  commitEmail: "agent@touchfish.dev"
```

### Go Structs

```go
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
```

See section 1 defaults table in previous version. All sub-structs map 1:1 to YAML fields.

### Defaults and Validation

| Field | Default | Required |
|-------|---------|----------|
| github.owner | — | Yes |
| github.repo | — | Yes |
| github.token / tokenEnv | — | One required |
| github.baseBranch | "master" | No |
| teams | — | Yes (at least one) |
| ai.binaryPath | — | Yes |
| ai.timeoutSec | 600 | No |
| testRunner.image | — | Yes |
| testRunner.testCmd | — | Yes |
| testRunner.timeoutSec | 300 | No |
| polling.* | 30/15/20 | No |
| workspace.baseDir | "/workspaces" | No |
| state.dir | "/data/state" | No |
| agent.maxTestRetries | 3 | No |

---

## 2. Package Structure

```
cmd/touchfish/main.go           — entry point: load config, wire deps, start orchestrator
internal/
  config/                       — Config struct, Load(), Validate()
  git/                          — GitClient interface + go-git implementation
  github/                       — GitHubClient interface + REST implementation
  ai/                           — AIAgent interface + CLI implementation + prompt templates
  testrunner/                   — TestRunner interface + Docker CP implementation
  orchestrator/                 — state machine, session lifecycle
  agent/                        — SE and QA agent loops
  state/                        — StateStore interface + file-backed implementation
  contract/                     — requirement parsing and diff detection
```

### Dependency Graph

```
cmd/touchfish/main.go
  └── config
  └── orchestrator
        ├── agent
        │     ├── git
        │     ├── github
        │     ├── ai
        │     ├── testrunner
        │     └── contract
        ├── state
        └── github
```

Infrastructure packages (`git`, `github`, `ai`, `testrunner`, `contract`, `state`) never depend on each other or on `agent`/`orchestrator`.

---

## 3. Interfaces

### GitClient

```go
type GitClient interface {
    Clone(ctx, repoURL, localPath string, auth AuthConfig) error
    Pull(ctx, localPath string) (newHead string, err error)
    Push(ctx, localPath string) error
    ListRemoteBranches(ctx, repoURL, prefix string, auth AuthConfig) ([]Ref, error)
    HeadHash(ctx, localPath string) (string, error)
    DiffFiles(ctx, localPath, fromHash, toHash string) ([]DiffEntry, error)
    FileContent(ctx, localPath, filePath, hash string) ([]byte, error)
    StageAll(ctx, localPath string) error
    Commit(ctx, localPath, message, authorName, authorEmail string) (hash string, err error)
    Checkout(ctx, localPath, branch string) error
    CurrentBranch(ctx, localPath string) (string, error)
}
```

### GitHubClient

```go
type GitHubClient interface {
    // Pull Requests
    CreatePR(ctx, head, base, title, body string) (*PullRequest, error)
    GetPR(ctx, number int) (*PullRequest, error)
    UpdatePR(ctx, number int, title, body *string) error
    ListPRs(ctx, state PRState, head string) ([]PullRequest, error)
    AddPRComment(ctx, prNumber int, body string) error

    // Issues
    ListIssues(ctx, labels []string) ([]Issue, error)
    GetIssue(ctx, number int) (*Issue, error)
    CreateIssue(ctx, title, body string, labels []string) (*Issue, error)
    AddIssueLabel(ctx, issueNumber int, label string) error
    RemoveIssueLabel(ctx, issueNumber int, label string) error
    AddIssueComment(ctx, issueNumber int, body string) error
    ListIssueComments(ctx, issueNumber int) ([]Comment, error)

    // Commits
    ListCommits(ctx, branch string, limit int) ([]CommitInfo, error)
}
```

### AIAgent

```go
type AIAgent interface {
    Invoke(ctx, workDir string, prompt string) (*InvokeResult, error)
}

type InvokeResult struct {
    ExitCode int
    Stdout   string
    Stderr   string
    Duration time.Duration
}
```

### TestRunner

```go
type TestRunner interface {
    Run(ctx, sourceDir string) (*TestResult, error)
}

type TestResult struct {
    Passed   bool
    ExitCode int
    Stdout   string
    Stderr   string
    Duration time.Duration
}
```

### StateStore

```go
type StateStore interface {
    Load(ctx, teamName string) (*TeamState, error)
    Save(ctx, state *TeamState) error
    Delete(ctx, teamName string) error
}
```

---

## 4. State Management

### State Types

```go
type TeamState struct {
    TeamName    string        `json:"teamName"`
    Session     *SessionState `json:"session,omitempty"`  // nil = no active session
    LastUpdated time.Time     `json:"lastUpdated"`
}

type SessionState struct {
    Branch    string    `json:"branch"`
    PRNumber  int       `json:"prNumber"`
    StartedAt time.Time `json:"startedAt"`
    SE        SEState   `json:"se"`
    QA        QAState   `json:"qa"`
}

type SEState struct {
    LastProcessedCommit string `json:"lastProcessedCommit"`
    LastRequirementHash string `json:"lastRequirementHash"`
    LastTestReqHash     string `json:"lastTestReqHash"`
    CurrentIssueNumber  int    `json:"currentIssueNumber"`
    ConsecutiveFailures int    `json:"consecutiveFailures"`
    Phase               string `json:"phase"`
}

type QAState struct {
    LastReviewedCommit  string `json:"lastReviewedCommit"`
    LastRequirementHash string `json:"lastRequirementHash"`
    Phase               string `json:"phase"`
}
```

### Persistence

- One JSON file per team: `<state.dir>/<teamName>.json`
- Atomic writes: write to `.tmp`, then `os.Rename`
- Flush after every significant state transition

### Crash Recovery

**Principle: state file is a hint, git and GitHub are source of truth.**

1. Load state from disk
2. Verify session is still valid (branch exists, PR open) via GitHub API
3. If invalid → clear session, start fresh
4. If valid → pull latest, resume. Diffs detected naturally on next loop
5. Phase is advisory — worst case on crash is a redundant AI invocation (safe, all operations are idempotent)

### Thread Safety

SE and QA each only write to their own sub-state (`sess.SE` / `sess.QA`). Per-team `sync.Mutex` protects Save.

---

## 5. Orchestrator State Machine

### States

```
IDLE → SCANNING → SESSION_ACTIVE → SESSION_ENDING → IDLE
```

| State | Action | Exit |
|-------|--------|------|
| IDLE | Wait | Timer → SCANNING |
| SCANNING | `ListRemoteBranches("agent/<team>/")` | Found → SESSION_ACTIVE; None → IDLE |
| SESSION_ACTIVE | Clone repos, create PR, run SE+QA goroutines, monitor PR | PR merged/closed → SESSION_ENDING |
| SESSION_ENDING | Cancel SE+QA, wait, clear state | Done → IDLE |

### Structure

- One goroutine per configured team, each runs its own state machine
- Top-level orchestrator starts team goroutines and handles graceful shutdown via context cancellation

### Session Lifecycle

1. **Start:** Clone (or pull) repos into `<workspace>/<team>/se/` and `<workspace>/<team>/qa/`. Create PR if none exists. Record initial HEAD hash. Start SE and QA goroutines.
2. **Monitor:** Periodically check PR status. If merged or closed, cancel agent goroutines.
3. **End:** Wait for agents to finish. Clear session state. Return to scanning.

---

## 6. SE Agent Loop

### Tick Flow

```
pull latest
├── Priority 1: issues labeled "Agent to fix" for this session?
│   → pick first issue
│   → update label: "Agent to fix" → "Agent fixing"
│   → invoke AI (issue fix prompt)
│   → test and commit
│   → update label: "Agent fixing" → "Agent fixed to be verified"
│
├── Priority 2: requirement diffs? (REQUIREMENT.md or TEST_REQUIREMENTS.md changed)
│   → detect via content hash comparison
│   → invoke AI (implementation prompt)
│   → test and commit
│
└── Priority 3: idle
```

### Test-and-Commit with Retry

```
invoke AI to implement/fix
loop (max N retries):
    run tests (Docker CP → ephemeral container)
    if pass:
        commit and push → done
    if fail:
        invoke AI with test output (test fix prompt)
if exhausted:
    commit WIP + file escape issue
```

### Push Conflict Handling

If push fails (non-fast-forward): pull, then re-run tests. If pull fails (merge conflict): file issue for user.

---

## 7. QA Agent Loop

### Tick Flow

```
pull latest
├── Priority 1: REQUIREMENT.md changed?
│   → invoke AI (test requirements prompt)
│   → commit and push TEST_REQUIREMENTS.md
│
├── Priority 2: new SE commits since last review?
│   → get changed files
│   → invoke AI (review prompt)
│   → parse structured JSON findings
│   → file issues (with dedup by title)
│
└── Idle
```

### Push Conflict

QA only writes TEST_REQUIREMENTS.md. If push fails: pull and retry once. Not time-critical.

---

## 8. Prompt Templates

### 8.1 SE: Issue Fix

**Context passed to AI:**
- PROJECT_SETUP.md (full content)
- Issue title, body, and all comments (full conversation thread)
- Current branch name for context

```
You are a software engineer working on this project.

## Project conventions
{PROJECT_SETUP.md content}

## Issue to fix
#{issue.number}: {issue.title}

{issue.body}

### Comments
{for each comment:}
**{comment.author}** ({comment.date}):
{comment.body}
{end for}

## Instructions
1. Read the relevant source code to understand the issue
2. Implement a fix based on the issue description and the discussion in comments
3. Update or add tests to cover this fix
4. Make minimal, focused changes

Do NOT commit. Just modify the files.
```

### 8.2 SE: Requirement Implementation

**Context passed to AI:**
- PROJECT_SETUP.md (full content)
- REQUIREMENT.md (full content)
- TEST_REQUIREMENTS.md (full content, if exists)
- What changed: which file(s) changed, and the diff showing what was added/modified/removed

```
You are a software engineer working on this project.

## Project conventions
{PROJECT_SETUP.md content}

## Full feature requirements
{REQUIREMENT.md content}

## Full test requirements
{TEST_REQUIREMENTS.md content, or "No test requirements yet."}

## What changed
{if REQUIREMENT.md changed:}
### REQUIREMENT.md changes
```diff
{diff of REQUIREMENT.md from last processed version to current}
```
{end if}

{if TEST_REQUIREMENTS.md changed:}
### TEST_REQUIREMENTS.md changes
```diff
{diff of TEST_REQUIREMENTS.md from last processed version to current}
```
{end if}

## Instructions
1. Read the existing code to understand current state
2. Focus on implementing the CHANGES shown above — do not re-implement what already exists
3. Write tests as specified in the test requirements
4. Follow project conventions for file naming, test framework, etc.

Do NOT commit. Just modify the files.
```

### 8.3 SE: Test Fix

**Context passed to AI:**
- Test output (stdout + stderr from test runner)

```
You are a software engineer. Your tests are failing.

## Test output
{test stdout}
{test stderr}

## Instructions
1. Analyze the test failures
2. Fix the code (implementation or tests) to make all tests pass
3. Make minimal changes — only fix what's broken

Do NOT commit. Just modify the files.
```

### 8.4 QA: Test Requirements Generation

**Context passed to AI:**
- REQUIREMENT.md (full content)
- PROJECT_SETUP.md (full content)
- Existing TEST_REQUIREMENTS.md (if exists)
- Diff of REQUIREMENT.md showing what changed

```
You are a QA test strategist. Your job is to define WHAT to test, not HOW to code it.

## Feature requirements
{REQUIREMENT.md content}

## What changed in requirements
```diff
{diff of REQUIREMENT.md from last processed version to current}
```

## Project conventions
{PROJECT_SETUP.md content}

{if TEST_REQUIREMENTS.md exists:}
## Existing TEST_REQUIREMENTS.md
{existing content}

Update or extend based on the requirement changes. Preserve items that are still relevant. Remove items for deleted requirements.
{else:}
No existing TEST_REQUIREMENTS.md. Create from scratch.
{end if}

## Instructions
Write TEST_REQUIREMENTS.md with:
1. Test scenarios grouped by feature area
2. For each scenario: description, input conditions, expected outcome, edge cases
3. Priority (critical / high / medium / low)
4. Do NOT write test code — only describe what should be tested

Write directly to the file TEST_REQUIREMENTS.md.
```

### 8.5 QA: Code Review

**Context passed to AI:**
- TEST_REQUIREMENTS.md (full content)
- REQUIREMENT.md (full content)
- PROJECT_SETUP.md (full content)
- List of changed files with change type

```
You are a QA test strategist reviewing code changes made by an SE agent.

## Your test requirements
{TEST_REQUIREMENTS.md content}

## Feature requirements
{REQUIREMENT.md content}

## Project conventions
{PROJECT_SETUP.md content}

## Changed files since last review ({fromHash}..{toHash})
{for each changed file:}
- {file.path} ({Added|Modified|Deleted})
{end for}

## Instructions
Review whether the SE's implementation and tests correctly fulfill the test requirements.

For each finding, output a JSON object on its own line:
{"type": "missing_test", "requirement": "...", "detail": "..."}
{"type": "wrong_test", "requirement": "...", "detail": "..."}
{"type": "missing_coverage", "feature": "...", "detail": "..."}

If everything looks good:
{"type": "approved", "detail": "All tests align with requirements."}

Only output JSON lines. No other text.
```

---

## 9. Container Architecture

### Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /touchfish ./cmd/touchfish/

FROM alpine:3.19
RUN apk add --no-cache git docker-cli
COPY --from=builder /touchfish /usr/local/bin/touchfish
# AI CLI installation depends on provider (Claude Code, Codex, etc.)
ENTRYPOINT ["touchfish"]
CMD ["--config", "/etc/touchfish/config.yaml"]
```

### Volume Layout

```
Host /data/touchfish/
  ├── config.yaml          → /etc/touchfish/config.yaml (read-only)
  ├── state/               → /data/state
  └── workspaces/          → /workspaces
      ├── alpha/
      │   ├── se/
      │   └── qa/
      └── beta/
          ├── se/
          └── qa/
```

### Docker Compose

```yaml
services:
  touchfish:
    build: .
    volumes:
      - ./config.yaml:/etc/touchfish/config.yaml:ro
      - /data/touchfish/state:/data/state
      - /data/touchfish/workspaces:/workspaces
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - GITHUB_TOKEN=${GITHUB_TOKEN}
    restart: unless-stopped
```

### Test Runner Lifecycle

```
docker create --name <id> --network none <image>
docker cp <sourceDir>/. <id>:/code
docker start <id>
docker wait <id>                → exit code
docker logs <id>                → stdout/stderr
docker rm <id>                  → cleanup
```

Timeout enforced via `docker stop` + `docker rm` if `docker wait` exceeds `testRunner.timeoutSec`.

### Security

| Concern | Mitigation |
|---------|------------|
| Docker socket | Only deploy on trusted home server |
| GitHub token | Env var, never in config file |
| AI API keys | Managed by AI CLI, out of scope |
| Test runner | `--network none` by default |

---

## 10. Implementation Sequencing

| Step | Package | Test approach |
|------|---------|---------------|
| 1 | `config` | Valid/invalid YAML, defaults, validation |
| 2 | `state` | Load/save/delete with temp dirs, atomic write |
| 3 | `git` | In-memory repos (go-git memory storage) |
| 4 | `github` | httptest server with canned JSON |
| 5 | `ai` | Mock binary (script that echoes), prompt tests |
| 6 | `testrunner` | Mock Docker CLI, optional real Docker integration |
| 7 | `contract` | Hash diff detection, requirement parsing |
| 8 | `agent` | Mocked interfaces: verify SE priority, retry, QA flow |
| 9 | `orchestrator` | Mocked interfaces: state machine transitions |
| 10 | `cmd/touchfish` | Wire config, mock externals, verify startup/shutdown |
