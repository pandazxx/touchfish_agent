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
