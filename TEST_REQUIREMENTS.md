# Test Requirements

This document defines the test-driven development requirements for the touchfish_agent project. Follow these guidelines when writing, modifying, or reviewing tests.

## Testing Philosophy

### Core Principles

1. **Blackbox Testing for AI Components**: Treat `codex` as a blackbox. Verify the exact prompt input, not internal behavior.
2. **Mocked External Dependencies**: All external systems (`gh`, `git` network operations) must be mocked.
3. **Data-Driven Tests**: Each test case is a separate file with clear input/expected variables.
4. **Containerized Execution**: Tests run in Docker containers by default for consistency.
5. **Library Mode**: Source `agent.sh` with `AGENT_LIBRARY_MODE=1` to test individual functions.

### What to Test vs What to Mock

| Component | Strategy | Rationale |
|-----------|----------|-----------|
| `codex` | Blackbox - verify full prompt | AI agent behavior is unpredictable; verify inputs only |
| `gh` | Mock with JSON responses | Avoid GitHub API calls; control test scenarios |
| `git` (network) | Mock push/pull operations | Avoid network; allow local repo operations |
| `git` (local) | Use real git | Need real repos for branch/commit testing |
| Shell functions | Test directly | Core logic under test |

## Test Infrastructure

### Directory Structure

```
tests/
├── unit_test.sh          # Test harness and runner
├── report.txt            # Generated test report
├── cases/                # Individual test case files
│   └── *.sh              # One file per test scenario
├── mocks/                # Mock implementations
│   ├── gh                # GitHub CLI mock
│   ├── git               # Git mock (partial)
│   └── codex             # Codex CLI mock
└── data/                 # Test fixtures
    └── *.json            # Mock API responses
```

### Test Harness Features

| Feature | Environment Variable | CLI Flag | Default |
|---------|---------------------|----------|---------|
| Verbose output | `UNIT_TEST_VERBOSE=1` | `--verbose` | Off |
| Container execution | `RUN_IN_CONTAINER=1` | `--no-container` | On |
| Report path | `TEST_REPORT=/path` | - | `tests/report.txt` |

### Available Assertions

```bash
assert_equal "$actual" "$expected"      # Exact match
assert_contains "$haystack" "$needle"   # Substring match
```

### Helper Functions

```bash
setup_repo "$repo_path" "$remote_path"  # Initialize git repo with remote
log_report "message"                    # Write to test report
```

## Test Case Structure

### Required Variables

Every test case file MUST define:

```bash
TEST_NAME="Human-readable test description"

run_case() {
  # Test implementation
  # Return 0 for pass, non-zero for fail
}
```

### Optional Variables

```bash
INPUT_*          # Input data for the test
EXPECTED_*       # Expected values for assertions
```

### Standard Test Case Template

```bash
TEST_NAME="Description of what this test verifies"

# Input data
INPUT_ISSUE_NUMBER=42
INPUT_PR_NUMBER=101

# Expected values
EXPECTED_PROMPT_CONTAINS="some expected text"

run_case() {
  # 1. Setup: Create repo and configure mocks
  local repo="$TEST_TMP/repo_name"
  local remote="$TEST_TMP/remote_name.git"
  setup_repo "$repo" "$remote"

  # 2. Create initial state
  echo "content" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "initial" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  # 3. Configure mock environment
  local gh_log="$TEST_TMP/gh.log"
  local codex_log="$TEST_TMP/codex.log"

  export GH_CALL_LOG="$gh_log"
  export GH_MOCK_ISSUE_JSON="$ROOT_DIR/tests/data/issue.json"
  export CODEX_PROMPT_LOG="$codex_log"
  export CODEX_OUTPUT_FILE="$repo/output.txt"

  # 4. Configure agent environment
  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"
  export CODEX_CMD="codex"

  # 5. Source agent and execute function under test
  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  (cd "$repo" && function_under_test "$INPUT_ISSUE_NUMBER" "$INPUT_PR_NUMBER")

  # 6. Verify results
  local actual
  actual=$(cat "$codex_log")
  assert_contains "$actual" "$EXPECTED_PROMPT_CONTAINS"
}
```

## Mock Configuration

### GitHub CLI Mock (`tests/mocks/gh`)

| Environment Variable | Purpose | Example |
|---------------------|---------|---------|
| `GH_CALL_LOG` | Log file for gh calls (required) | `$TEST_TMP/gh.log` |
| `GH_MOCK_ISSUE_JSON` | JSON file for `gh issue view` | `tests/data/issue.json` |
| `GH_MOCK_ISSUE_LIST` | Output for `gh issue list` | `42` |
| `GH_MOCK_PR_LIST` | Output for `gh pr list` | `101` |
| `GH_PR_MERGED` | Return value for `gh pr view --json merged` | `true` or `false` |

### Git Mock (`tests/mocks/git`)

| Environment Variable | Purpose | Example |
|---------------------|---------|---------|
| `GIT_CALL_LOG` | Log file for git calls | `$TEST_TMP/git.log` |
| `GIT_MOCK_DIFF` | Mocked output for `git diff` | Diff string |
| `GIT_REAL_BIN` | Path to real git (auto-set) | `/usr/bin/git` |

**Note**: Git mock passes through to real git for most commands. Only `diff`, `push`, `commit`, `config`, and `remote get-url` are intercepted.

### Codex Mock (`tests/mocks/codex`)

| Environment Variable | Purpose | Example |
|---------------------|---------|---------|
| `CODEX_PROMPT_LOG` | File to capture stdin prompt (required) | `$TEST_TMP/codex.log` |
| `CODEX_OUTPUT_FILE` | File to create (simulates codex output) | `$repo/impl.txt` |

## Required Test Coverage

### Main Loop Tests

| Scenario | Function | Verification |
|----------|----------|--------------|
| Branch discovery | `find_agent_branch` | Returns branch matching `agent/<name>/*` |
| Skip merged branches | `find_agent_branch` | Ignores branches merged into base |
| No matching branch | `find_agent_branch` | Returns empty string |
| PR creation | `ensure_pr` | Calls `gh pr create` with correct args |
| PR exists | `ensure_pr` | Returns existing PR number |

### Session Loop Tests

| Scenario | Function | Verification |
|----------|----------|--------------|
| Issue found | `session_loop` | Triggers `issue_fix_cycle` |
| Requirement changed | `session_loop` | Triggers `implementing_cycle` |
| PR merged | `session_loop` | Calls `compact_session`, exits loop |
| Idle state | `session_loop` | No codex call, continues polling |

### Issue Fixing Cycle Tests

| Scenario | Function | Verification |
|----------|----------|--------------|
| Prompt generation | `issue_fix_cycle` | Prompt includes title, body, all comments |
| Label: to_fix → fixing | `issue_fix_cycle` | `gh issue edit` called correctly |
| Label: fixing → pending_verify | `issue_fix_cycle` | After successful fix |
| Comment prefix | `issue_fix_cycle` | Comment starts with `<agent_name>:` |
| Commit prefix | `issue_fix_cycle` | Commit message starts with `Aaron:` |
| PR update | `issue_fix_cycle` | `gh pr comment` called |
| No changes | `issue_fix_cycle` | No commit, no label change to pending |
| Re-verification | `issue_fix_cycle` | Handles issues cycled back to `to_fix` |

### Implementing Cycle Tests

| Scenario | Function | Verification |
|----------|----------|--------------|
| Single file | `implementing_cycle` | Prompt includes file content and diff |
| Multiple files | `implementing_cycle` | All files included in prompt |
| Commit prefix | `implementing_cycle` | Commit message starts with `Aaron:` |
| No changes | `implementing_cycle` | No commit created |

### Session Exit Tests

| Scenario | Function | Verification |
|----------|----------|--------------|
| PR merged | `compact_session` | `/compact` prompt sent to codex |

### Error Handling Tests

| Scenario | Test File | Verification |
|----------|-----------|--------------|
| Codex execution failure | `error_codex_failure.sh` | No commit, no label to pending_verify |
| GitHub API failure | `error_gh_api_failure.sh` | Graceful handling, no crash |
| Git push rejected | `error_git_push_rejected.sh` | Push attempted, flow continues |

### Edge Case Tests

| Scenario | Test File | Verification |
|----------|-----------|--------------|
| Multiple issues same PR | `multiple_issues_same_pr.sh` | Only first issue processed |
| Issue with no comments | `issue_no_comments.sh` | Prompt structure correct |
| Empty requirement diff | `empty_requirement_diff.sh` | File content still included |
| PR mention filtering | `issue_pr_mention_filtering.sh` | Correct search parameters |

## Writing New Test Cases

### Step-by-Step Guide

1. **Create test file**: `tests/cases/<descriptive_name>.sh`

2. **Define TEST_NAME**: Use present tense, describe the behavior
   ```bash
   TEST_NAME="Issue fix updates labels correctly"
   ```

3. **Define inputs** (if needed):
   ```bash
   INPUT_ISSUE_NUMBER=42
   INPUT_MOCK_DIFF="..."
   ```

4. **Implement run_case()**:
   - Setup repo with `setup_repo`
   - Configure mocks via environment variables
   - Source agent with `AGENT_LIBRARY_MODE=1`
   - Call function under test
   - Assert results

5. **Run test**:
   ```bash
   ./tests/unit_test.sh --no-container --verbose
   ```

### Test Data Files

For complex mock data (e.g., GitHub API responses), create JSON files in `tests/data/`:

```json
{
  "title": "Issue title",
  "body": "Issue description",
  "comments": [
    {"author": {"login": "user1"}, "body": "Comment text"}
  ]
}
```

Reference in test: `export GH_MOCK_ISSUE_JSON="$ROOT_DIR/tests/data/filename.json"`

## Verification Patterns

### Verify Codex Prompt (Exact Match)

```bash
local prompt
prompt=$(cat "$codex_log")

local expected_prompt
expected_prompt=$(cat <<EOF
You are an automated coding agent...
EOF
)

assert_equal "$prompt" "$expected_prompt"
```

### Verify Codex Prompt (Contains)

```bash
local prompt
prompt=$(cat "$codex_log")

assert_contains "$prompt" "Expected substring"
```

### Verify GH CLI Calls

```bash
local gh_calls
gh_calls=$(cat "$gh_log")

if [[ "$gh_calls" != *"issue edit 42 --add-label agent_fixing"* ]]; then
  echo "Expected gh call not found" >&2
  return 1
fi
```

### Verify No Commit

```bash
local initial_sha final_sha
initial_sha=$(git -C "$repo" rev-parse HEAD)

# ... run test ...

final_sha=$(git -C "$repo" rev-parse HEAD)

if [[ "$initial_sha" != "$final_sha" ]]; then
  echo "Unexpected commit created" >&2
  return 1
fi
```

### Verify Branch Discovery

```bash
local found_branch
found_branch=$(find_agent_branch "main")

if [[ "$found_branch" != "expected/branch/name" ]]; then
  echo "Wrong branch: $found_branch" >&2
  return 1
fi
```

## Test Execution

### Local Development

```bash
# Run all tests in container (default)
./tests/unit_test.sh

# Run without container (faster iteration)
./tests/unit_test.sh --no-container

# Verbose output for debugging
./tests/unit_test.sh --no-container --verbose
```

### CI/CD Integration

```bash
# Run in container, check exit code
./tests/unit_test.sh
exit_code=$?

# Report available at
cat ./tests/report.txt
```

### Interpreting Results

```
PASS: Issue fix with multiple comments
PASS: Requirement changes
FAIL: PR merged
FAILED: 1 test(s) failed
```

Exit code: 0 = all pass, 1 = failures

## Checklist for Test Reviews

- [ ] Test has descriptive `TEST_NAME`
- [ ] Test uses `setup_repo` for git initialization
- [ ] All external calls are mocked (`gh`, network `git`)
- [ ] `AGENT_LIBRARY_MODE=1` is set before sourcing `agent.sh`
- [ ] Assertions verify the correct behavior
- [ ] Test cleans up (handled by harness via `$TEST_TMP`)
- [ ] Test is deterministic (no race conditions, no external dependencies)
- [ ] Input/expected variables are clearly named
- [ ] Codex prompts are verified exactly as generated (blackbox principle)
- [ ] Error scenarios return gracefully (no crashes)

## Test Case Inventory

### Core Scenarios (from REQUIREMENTS.md)

| Test File | Category | Description |
|-----------|----------|-------------|
| `issue_fix_multiple_comments.sh` | Issue Fixing | Prompt includes all comments |
| `requirement_changes.sh` | Implementing | Diff and content in prompt |
| `pr_merged.sh` | Session Exit | `/compact` sent to codex |

### Main Loop Tests

| Test File | Description |
|-----------|-------------|
| `branch_discovery.sh` | Finds matching `agent/<name>/*` branch |
| `branch_skip_merged.sh` | Ignores merged branches |
| `no_agent_branch.sh` | Returns empty when no match |
| `pr_creation.sh` | Creates PR with correct title/body |

### Issue Fixing Tests

| Test File | Description |
|-----------|-------------|
| `issue_label_transitions.sh` | Labels: to_fix → fixing → pending_verify |
| `issue_comment_prefix.sh` | Comment starts with `<agent_name>:` |
| `issue_reverification.sh` | Handles issues cycled back |
| `commit_message_prefix.sh` | Commit starts with `Aaron:` |
| `pr_update_on_fix.sh` | PR comment added after fix |
| `no_changes_detected.sh` | No commit when codex produces nothing |
| `issue_no_comments.sh` | Works with empty comments array |

### Implementing Cycle Tests

| Test File | Description |
|-----------|-------------|
| `multiple_requirement_files.sh` | All files in prompt |
| `empty_requirement_diff.sh` | Handles empty diff gracefully |
| `requirements_changed_detection.sh` | Change detection logic |

### Session Loop Tests

| Test File | Description |
|-----------|-------------|
| `session_loop_idle.sh` | No action when idle |
| `issue_pr_mention_filtering.sh` | Correct issue search params |
| `multiple_issues_same_pr.sh` | Only first issue processed |

### Error Handling Tests

| Test File | Description |
|-----------|-------------|
| `error_codex_failure.sh` | Codex command not found/fails |
| `error_gh_api_failure.sh` | GitHub API returns empty/error |
| `error_git_push_rejected.sh` | Git push operation handling |

**Total Test Cases**: 23

## PO Review Findings (2026-02-03)

Based on Product Owner review in `role/PO.md`:

### Addressed Items

| Finding | Priority | Resolution |
|---------|----------|------------|
| Missing error handling tests | HIGH | Added `error_*.sh` test cases |
| Missing concurrent scenario tests | HIGH | Added `multiple_issues_same_pr.sh` |
| Missing edge case tests | HIGH | Added `issue_no_comments.sh`, `empty_requirement_diff.sh` |
| Issue PR mention filtering | MEDIUM | Added `issue_pr_mention_filtering.sh` |
| `gh issue edit` call verification | MEDIUM | Already in `issue_label_transitions.sh` |

### Deferred Items (Future Improvements)

| Finding | Priority | Notes |
|---------|----------|-------|
| Full integration tests | LOW | Track as technical debt |
| Test coverage reporting | LOW | Consider adding coverage tools |
| Performance/timeout tests | LOW | Not critical for initial release |
| Property-based testing | LOW | Future enhancement |

### Compliance Status

- REQUIREMENTS.md unit test requirements: **17/17 (100%)**
- Core functional scenarios: **15/15 (100%)**
- Error handling scenarios: **3/3 (100%)**
- Edge case scenarios: **4/4 (100%)**

## Related Documentation

- `tests/README.md` - How to run tests
- `role/PO.md` - Product Owner review feedback
- `REQUIREMENTS.md` - Functional requirements (source of test scenarios)
