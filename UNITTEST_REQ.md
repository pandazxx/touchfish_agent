# Unit Test Requirements

## 1. Purpose
Define comprehensive unit-test scenarios for the container-based AI workflow described in `DETAIL_REQUIREMENT.md`. The tests must validate expected behavior, edge cases, and graceful handling of failures across session lifecycle, GitHub integration, prompt generation, and configuration parsing.

## 2. Scope
Unit tests cover the agent logic and shell workflow responsible for:
- Session discovery and lifecycle.
- Issue-fix and requirement-change cycles.
- Prompt construction for `codex`.
- GitHub CLI (`gh`) interactions.
- Git interactions (mocked).
- Configuration handling and validation.
- Logging, error handling, and security safeguards.

## 3. Test Data and Mocking Requirements
- Mock all external integrations:
  - `gh` output (issues, PRs, branches, labels, comments).
  - `git` commands (branch listing, diff, status, commit, push).
  - `codex` invocation (validate exact prompt strings only; treat as black box).
- Use data-driven test case files with clear input/expected variables.
- Each test case must be isolated and should not depend on global state.
- Provide mock values for required environment variables (no real secrets).
- Tests must print every executed command for debugging, with a verbosity flag to control extra output.
- Provide a flag to disable containerized test execution (default on).

## 4. Configuration and Input Validation Scenarios
### 4.1 Required environment variables
- Missing `AGENT_NAME` → fail fast with clear error.
- Missing `GITHUB_TOKEN` → fail fast without echoing token.
- Missing `REPO_URL` → fail fast with clear error.
- Missing `WORKSPACE_DIR` → fail fast with clear error.

### 4.2 Optional environment variables
- `GH_HOST` unset → default to `github.com`.
- `LOG_LEVEL` invalid → default to `info` and log warning.
- `POLL_INTERVAL_SECONDS` missing or non-numeric → fallback to 60s and log warning.
- `MAIN_BRANCH_FALLBACKS` missing → default to `master,main`.
- `MAX_ISSUES_PER_CYCLE` missing → default to 1.
- `SESSION_IDLE_TIMEOUT` unset → idle timeout disabled.

### 4.3 Input edge cases
- Whitespace or empty-string values → treated as missing and handled gracefully.
- `MAIN_BRANCH_FALLBACKS` contains invalid branch names → skip invalid entries and log.
- `MAX_ISSUES_PER_CYCLE` set to 0 or negative → treat as 1 and log warning.

## 5. Session Start and Branch Discovery Scenarios
### 5.1 Branch discovery
- Branch list includes `agent/<AGENT_NAME>/*` → select first matching branch.
- No matching branches → loop waits and logs idle state.
- Matching branch already merged into base → skip and continue searching.
- Multiple matches → select first in deterministic order.

### 5.2 Repository reuse
- Existing clone present → reuse without reclone.
- Clone missing → perform clone into `<WORKSPACE_DIR>/repos/<repo-name>`.

### 5.3 Checkout
- Clean checkout succeeds → proceed to session.
- Checkout failure → clean working tree and retry once; if still fails, log and pause.

## 6. PR Management Scenarios
- No PR exists for session branch → create PR with title format `Agent Session: <branch-name>`.
- Single PR exists → select it.
- Multiple PRs exist → select most recently updated.
- PR already merged during cycle → skip remaining steps and trigger session exit.

## 7. Issue Fix Cycle Scenarios
### 7.1 Issue discovery
- Issues found with label `agent_to_fix` and PR mention → proceed.
- Issues without PR mention → ignore.
- Issues with wrong label casing or different label → ignore.
- More issues than `MAX_ISSUES_PER_CYCLE` → only process up to limit.

### 7.2 Label transitions
- Transition `agent_to_fix` → `agent_fixing` → `agent_pending_verify`.
- Label update failure → log error and re-enter loop after cooldown.

### 7.3 Prompt generation (Issue Fix)
- Validate exact prompt template with:
  - Full issue body verbatim (including multiline and markdown).
  - Multiple comments in correct order with numbering.
  - All placeholders replaced with actual values.
- Edge cases:
  - Issue with no comments → `## Issue Comments` section present but empty or marked accordingly.
  - Issue with special characters (quotes, backticks, emojis) preserved.

### 7.4 Issue comment posting
- Comment posted with prefix `<AGENT_NAME>:`.
- Failure to post comment → log error but continue to label update.

### 7.5 Commit behavior
- Commit message format: `Aaron: issue fix <issue-id> - <short-summary>`.
- Git push is mocked and never hits network.

## 8. Requirement Change Cycle Scenarios
### 8.1 Requirement change detection
- Diff present in `REQUIREMENTS.md` → trigger requirement-change cycle.
- Diff present in `CICD_REQUIREMENTS.md` → trigger requirement-change cycle.
- Diff present in referenced requirement files → trigger requirement-change cycle.
- No diff → return to loop without action.

### 8.2 Prompt generation (Requirement Change)
- Validate exact prompt template with:
  - Full snapshots of requirement files (or empty if missing).
  - Mocked git diff output for requirement files.
  - All placeholders replaced with actual values.
- Edge cases:
  - Missing `CICD_REQUIREMENTS.md` → include empty section.
  - Missing referenced files → log warning and continue with available files.

### 8.3 Commit behavior
- Commit message format: `Aaron: implement requirements <short-summary>`.
- Ensure diff-based summary is used for `<short-summary>`.

## 9. Main + Session Loop Behavior
- If issues exist → issue-fix cycle is selected over requirement changes.
- If no issues but requirement diff exists → requirement-change cycle runs.
- If neither applies → loop waits and stays on session branch.
- Loop respects `POLL_INTERVAL_SECONDS` and uses cooldown after failures.

## 10. Session Exit Scenarios
- PR merged detected → session exit logic triggered.
- `/compact` instruction sent to `codex` on exit.
- Session is considered closed after merge and compact.

## 11. Error Handling and Recovery
- GitHub API failure → exponential backoff, retry, and log.
- Rate limit exceeded → log and retry after backoff.
- Missing/invalid credentials → clear error; cycle halts without retries.
- Git checkout errors → clean working tree and retry once.
- Any cycle failure → log failure reason and re-enter loop after cooldown.

## 12. Logging and Security Scenarios
- Logs include timestamp, agent name, session branch, cycle type.
- Logs capture action outcomes and decision rationale.
- Tokens or secrets are never logged or echoed.
- Failure logs provide actionable detail without leaking secrets.

## 13. Unit Test Script Requirements (tests/unit_test.sh)
- Must mock `gh`, `git`, and `codex`.
- Must run in one-time containers by default; provide flag to disable container usage.
- Must support verbose flag (default off).
- Must print every executed command.
- Each test case stored in separate file with clear input/expected variables.
- Must treat `codex` as black box and verify exact prompt strings.
- Requirement-change tests must use mocked `REQUIREMENTS` content and mocked git diff output.
- Merged PR test must verify `/compact` prompt is sent to `codex`.
- Scenarios to include:
  - Issue fix with multiple comments.
  - Requirement changes.
  - PR merged.

## 14. Non-Functional Checks
- Tests must be deterministic and order-independent.
- No test should require network access or real credentials.
- Tests must be runnable in isolated containers and on host when container disabled.

## 15. Traceability
- Each test case should reference the section(s) of `DETAIL_REQUIREMENT.md` it validates.
- Ensure coverage mapping for:
  - Session lifecycle
  - Issue handling
  - Requirement handling
  - Prompt templates
  - Error handling
  - Logging/security
