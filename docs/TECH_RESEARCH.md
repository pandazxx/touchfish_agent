# Tech Research

## Git Library Comparison

### Python Options

| | GitPython | Dulwich | pygit2 |
|---|---|---|---|
| Implementation | Shells out to `git` CLI | Pure Python | C bindings (libgit2) |
| In-memory repo | No | Yes | Partial |
| Requires git installed | Yes | No | No |
| Install ease | Easy (pip) | Easy (pip) | Hard (needs libgit2) |
| License | GPLv3 | Apache 2.0 | GPLv2 |
| Maintenance | Maintenance mode | Active (v1.0 released Jan 2026) | Active |
| Stars | ~5,000 | ~2,200 | ~1,700 |

### Go Options

| | go-git | git2go | os/exec |
|---|---|---|---|
| Implementation | Pure Go | C bindings (CGO) | Shells out |
| In-memory repo | Yes | Yes | No |
| Merge/rebase | No | Yes | Yes |
| Cross-compile | Easy | Hard (CGO) | Easy |
| Maintenance | Active (v6 released Jan 2026) | Unmaintained (last release Oct 2022) | N/A |
| Stars | ~7,200 | ~2,000 | N/A |

### Recommended: Dulwich (Python) or go-git (Go)

git2go is unmaintained — avoid. GitPython shells out to git CLI, defeating the purpose of using a library. pygit2 has painful installation and GPLv2 license.

---

## Missing Features

### go-git

| Missing Feature | Impact on Our Project |
|-----------------|----------------------|
| `merge` | None — agents don't merge |
| `rebase` | None — agents don't rebase |
| `sparse checkout` | None — agents work on full repo |
| `submodules` (partial) | Low — unlikely in agent workflow |
| `git stash` | Low — agents shouldn't need to stash |
| SSH auto-detection | Low — can configure explicitly |

### Dulwich

| Missing/Weak Feature | Impact on Our Project |
|----------------------|----------------------|
| `merge` | None — agents don't merge |
| `rebase` | None — agents don't rebase |
| `submodules` (historically buggy) | Low |
| `porcelain.pull()` (has overwritten local changes) | **Medium** — agents pull frequently |
| Shallow clone edge cases | Low — agents use full clones |
| Thinner porcelain layer (some ops need low-level API) | Medium — more code to write |

---

## Operations Required by Our Project

| Operation | go-git | Dulwich |
|-----------|--------|---------|
| `clone` | Yes | Yes |
| `pull` / `fetch` | Yes | Yes (but quirky) |
| `push` | Yes | Yes |
| `diff` | Yes | Yes |
| `commit` | Yes | Yes |
| `add` (stage files) | Yes | Yes |
| `status` | Yes | Yes |
| `log` | Yes | Yes |
| `ls-remote` / branch scan | Yes | Yes |
| In-memory repo (testing) | Yes | Yes |

---

## Language Comparison: Python vs Go

**Decision: Go** — but Python remains a viable fallback if Go proves too slow for development.

| Factor | Python | Go | Winner |
|--------|--------|-----|--------|
| Git library | Dulwich — pure, in-memory, but `pull()` quirky | go-git — pure, in-memory, fewer quirks | Go |
| GitHub REST API | `requests` + mock libs | `net/http` + `httptest` (all stdlib) | Tie |
| AI CLI subprocess | `subprocess` (stdlib) | `os/exec` (stdlib) | Tie |
| Testing | pytest — fast to write, easy mocking | built-in `testing` + `httptest` — no external deps | Tie |
| Minimal dependencies | Needs Python runtime + pip packages | Single static binary, zero runtime deps | **Go** |
| Container image | Python base image (~100MB+) | Scratch/alpine + binary (~10-20MB) | **Go** |
| Development speed | Fast to prototype, less boilerplate | More verbose, slower to write | **Python** |
| AI generating our code | Most fluent, largest training corpus | Good but Python stronger | **Python** |
| Type safety | Dynamic — bugs at runtime | Static — bugs at compile time | **Go** |
| Error handling | Exceptions — easy to miss | Explicit — verbose but forces handling | **Go** |
| Concurrency | asyncio/threading — workable but awkward | Goroutines — native, lightweight | **Go** |
| Long-running daemon | GIL concerns, resource leak risks | Built for long-running services | **Go** |

**Why Go:** This project is a long-running daemon managing state machines with future concurrent teams. Go's single binary deployment, goroutines, type safety, and go-git's maturity make it the stronger fit. The stdlib covers GitHub API and testing with zero external deps.

**Why Python could win:** Faster to develop, AI writes better Python, lower barrier for solo/SOHO developer. If Go development proves too slow, switching to Python + Dulwich is viable since the wrapper layer abstracts git operations.

---

## Tech Stack Decision

| Component | Choice | Notes |
|-----------|--------|-------|
| Language | **Go** | Single binary, stdlib covers most needs |
| Git library | **go-git** | Pure Go, in-memory repos for testing |
| GitHub | **REST API** (net/http) | Mock with httptest |
| AI Agent | **CLI** (os/exec) | Black box, mock subprocess |
| Testing | **Go built-in** (testing + httptest) | No external test deps |

---

## Integration Decisions

| Integration | Approach | Test Strategy |
|------------|----------|---------------|
| GitHub | REST API (direct HTTP) | Mock HTTP layer, return canned JSON responses |
| Git | Wrapper layer (switchable between git library and CLI) | Library: in-memory repos. CLI: local bare repos. |
| AI Agent | CLI (black box) | Mock subprocess, verify invocation args |

### GitHub: REST API
- AI can generate code and test mocks fluently for HTTP-based APIs
- Easy to mock (stub HTTP layer, return canned JSON)
- Full control over auth, pagination, rate limiting

### Git: Wrapper Layer
- Abstract git behind an interface so implementation can be swapped
- Options: git CLI, Dulwich (Python), go-git (Go)
- Decision deferred to implementation phase

### AI Agent: CLI as Black Box
- Invoke Claude Code / Codex via CLI subprocess
- Orchestrator constructs the prompt, CLI does the work
- Don't test AI output — test invocation args and post-invocation handling

---

## Container Architecture

### Decision: Single Go Binary + Ephemeral Test Containers

**Single Go binary** runs orchestrator, SE, and QA as goroutines. No inter-container communication. Test execution uses ephemeral containers via Docker CP.

### Components

| Component | Runs as | Needs |
|-----------|---------|-------|
| Orchestrator | Goroutine | GitHub REST API |
| SE agent | Goroutine | go-git, GitHub REST API, AI CLI, Docker API |
| QA agent | Goroutine | go-git, GitHub REST API, AI CLI |
| Test runner | Ephemeral container | Target project toolchain |

### Workspace Layout

SE and QA have **separate directories** on the same volume to avoid git lock conflicts. Sync through GitHub (push/pull).

```
/data/workspaces/
  └── <team>/
      ├── se/    ← SE's git clone
      └── qa/    ← QA's git clone
```

### Test Execution: Docker CP

SE copies code into a fresh ephemeral container for each test run. Clean room per run, SE workspace untouched.

```
SE goroutine
  → docker create <test-runner-image>
  → docker cp /workspaces/<team>/se/. container:/code
  → docker start container (runs standalone test command)
  → wait for exit code + capture stdout/stderr
  → docker rm container
  → pass: commit and push
  → fail: retry (max N) or file issue
```

**Why Docker CP over volume mount:**
- Clean room — no leftover artifacts from previous runs
- SE workspace stays pristine, no cleanup needed
- Complete isolation — test runner can't corrupt SE's workspace
- Copy overhead is negligible compared to install deps + compile + test
- Matches how CI/CD works — clean checkout, install, test

**Requirements:**
- Go binary's container needs Docker socket mounted (`/var/run/docker.sock`)
- Test runner image pre-built per project (by SRE during bootstrap)
- Standalone build/test command defined in `PROJECT_SETUP.md`

### Architecture Diagram

```
┌──────────────────────────────────────┐
│  Go Binary Container                  │
│  + AI CLI installed                   │
│  + Docker socket mounted              │
│                                       │
│  Orchestrator goroutine               │
│    ├── SE goroutine                   │
│    │     ├── /workspaces/alpha/se/    │
│    │     ├── AI CLI (os/exec)         │
│    │     └── docker cp + run ─────────────→ ┌─────────────────┐
│    └── QA goroutine                   │      │ Test Runner      │
│          └── /workspaces/alpha/qa/    │      │ (ephemeral)      │
│          └── AI CLI (os/exec)         │      │ project toolchain│
│                                       │      └─────────────────┘
└──────────────────────────────────────┘
```

### Go Binary is Project-Agnostic

The Go binary + AI CLI container has no knowledge of the target project's language or toolchain. Project-specific tools live only in the test runner image. SRE provides:
- Test runner image name (or Dockerfile)
- Standalone build command
- Standalone test command

### Container Options Considered

| Option | Description | Why not chosen |
|--------|-------------|----------------|
| One container per agent | SE and QA in separate containers | Unnecessary — single binary with goroutines is simpler |
| One container per team | SE + QA share container | Same as our approach but less explicit |
| Monolith | Everything in one container, no test isolation | No clean room for tests |
| Orchestrator spawns agent containers | Dynamic container creation per session | Over-engineered for our needs |
| Volume mount for tests | Share SE workspace with test runner | Risk of workspace corruption, needs host path config |
| Named volume for tests | Docker-managed shared volume | Same corruption risk as volume mount |
