# Detailed Requirements

## 1. Purpose and Scope

This project provides a container-based AI workflow that leverages the GitHub ecosystem for collaboration, requirement tracking, and implementation. The workflow is designed for solo developers or small teams (2–3 members) and emphasizes minimal dependencies while maintaining a clear, auditable history of changes through Git commits.

The system uses GitHub as the primary interaction interface for users and agents, with requirements and guidance stored in Markdown files, issues used for bug fixes and adjustments, and branches/PRs used to represent agent sessions.

## 2. Goals

- Use GitHub as the primary user-agent interface:
  - Markdown files for requirements.
  - GitHub issues for bug fixes/adjustments.
  - Branches/PRs as sessions.
- Record every transaction as Git commits.
- Keep dependencies minimal.
- Support containerized runtime for agents.

## 3. Resource Requirements

- A container host with physical storage for workspaces.
- Containers for agent runtime.
- Access to GitHub.
- An AI code agent (e.g., `codex`, `claude code`).

## 4. Programming Language

- Use shell scripts when possible.
- Use Python only if necessary.

## 5. Inputs

The system must accept and use these inputs:

- Agent name.
- GitHub token.
- Repository URL.

## 6. High-Level Workflow

### 6.1 Session Start

1. User clones the project into the host workspaces folder.
2. User starts agent containers in the project folder, with the agent name configured.
3. The agent periodically scans the repository for branches matching `agent/<agent_name>/*`.
4. The agent checks out the first matching branch and starts the session.

### 6.2 Session Process

1. When a session starts, the user updates requirement Markdown files and commits the changes.
2. The user creates GitHub issues for bug fixes or adjustments, labels them `Agent to fix`, and includes the session branch name in the issue description.

### 6.3 Main + Session Loop

1. While on the active session branch, repeat:
   - Look for issues that:
     - Mention the current PR in the title.
     - Have the label `agent_to_fix`.
   - If found, run the Issue Fixing Cycle.
   - If no issues are found, check for changes in requirement files and run the Implementation Cycle.
   - If neither applies, wait and repeat while staying on the same session branch.

#### 6.3.1 Issue Fixing Cycle

1. Agent scans for issues that:
   - Mention the current PR in the title.
   - Have the label `agent_to_fix`.
2. If found, select the first issue and change its label to `agent_fixing`.
3. Agent uses the code agent (e.g., `codex` CLI) to fix the issue based on description and comments using the prompt in **17.1 Issue Fix Prompt Template**.
4. Commit changes using the template: `Aaron: issue fix <issue-id> - <short-summary>` and push.
5. Add a comment to the issue starting with `<agent name>: ` to indicate agent-generated feedback.
6. Update label to `agent_pending_verify`.
7. Update the PR.

**Notes:**
- If verification fails, issues may be reverted to `agent_to_fix` and must be reprocessed with full issue context.
- Issues can be bugs or agent implementation faults.

#### 6.3.2 Implementation Cycle

1. If no issues are found, the agent watches commits on the current branch.
2. The agent compares changes to requirement files:
   - `REQUIREMENTS.md`
   - `CICD_REQUIREMENTS.md`
   - Any other files referenced by those requirement files.
3. The agent interprets diffs as requirement changes and implements them using the prompt in **17.2 Requirement Change Prompt Template**.
4. Commit changes using the template: `Aaron: implement requirements <short-summary>` and push.
5. Update the PR.

#### 6.3.3 Ending the Cycle

1. CI/CD is triggered by the agent or GitHub action.
2. The agent updates the PR.
3. The system returns to cycle scanning.

### 6.4 Session Exit

1. User approves the PR and deletes the session branch.
2. Agent monitors PR status and considers the session closed when merged.
3. Agent compacts session history and returns to branch scanning.

### 6.5 Exit Criteria

- Exit session loop when the PR is merged.
- Before exit, compact agent session.
- After exit, continue the next main loop cycle.

## 7. Documentation Requirements

- `BUILD.md`: instructions for building images.
- `USAGE.md`: usage instructions, including how to acquire and set tokens for `codex` and `gh`.

## 8. Unit Test Requirements

- Implement `tests/unit_test.sh` with the following requirements:
  - Mock `gh` to emulate GitHub output.
  - Mock `codex` to verify agent prompts.
  - Create one-time containers to run the test script and generate reports.
  - Use data-driven test cases with clear, readable inputs.
  - Use mock values for required environment variables; avoid entrypoints requiring real values.
  - Print every command executed during unit tests for debugging.
  - Provide a command-line flag to enable verbose output (default off).
  - Mock `git` to avoid network pushes.
  - Provide a command-line flag to disable running tests in containers (default on).
  - Store each test case in a separate file with clear input/expected variables.
  - Treat `gh` and `git` as mocked inputs; treat `codex` as a black box and verify exact prompts.
  - The requirement-change test must use mocked `REQUIREMENTS` content and mocked git diff output.
  - The merged PR test must verify that `/compact` prompt is sent to `codex`.
  - Cover scenarios:
    - Issue fix with multiple comments.
    - Requirement changes.
    - PR merged.

- Generate `tests/README.md` explaining how to run tests.

## 9. Commit and PR Expectations

- Every change made by the agent must be committed with proper descriptions.
- Session activities should update PR status as required.
- Git commit messages should reflect the action performed and be recorded for traceability.

## 10. Configuration and Environment Variables

The system must support configuration through environment variables or CLI flags, and must document them in `USAGE.md`. Minimum required configuration:

- `AGENT_NAME`: the agent identifier used for branch matching and issue comment prefixes.
- `GITHUB_TOKEN`: token with repo read/write, issues, and PR permissions.
- `REPO_URL`: the HTTPS/SSH URL for the target repository.
- `WORKSPACE_DIR`: host path for local checkouts and session data.
- `POLL_INTERVAL_SECONDS`: frequency to scan for branches/issues (default: 60s).
- `MAIN_BRANCH_FALLBACKS`: ordered list for default PR base (default: `master,main`).

Optional configuration:

- `GH_HOST`: GitHub hostname for enterprise usage (default: `github.com`).
- `LOG_LEVEL`: `debug|info|warn|error` (default: `info`).
- `MAX_ISSUES_PER_CYCLE`: cap number of issues processed per loop (default: 1).
- `SESSION_IDLE_TIMEOUT`: exit session loop if idle for too long (disabled by default).

## 11. GitHub Integration Details

The agent must use GitHub CLI (`gh`) for all GitHub operations. The following behaviors are required:

- Branch discovery:
  - Query remote branches for `agent/<AGENT_NAME>/*`.
  - Filter out branches already merged into base branch (`master`/`main`).
- PR management:
  - If a PR from the session branch to base does not exist, create one with a consistent title format:
    `Agent Session: <branch-name>`.
  - If multiple PRs exist, select the most recently updated one.
- Issue management:
  - Search issues where the PR identifier is mentioned in the title.
  - Identify issues labeled `agent_to_fix`.
  - Update labels in order: `agent_to_fix` → `agent_fixing` → `agent_pending_verify`.
  - Add a comment starting with `<AGENT_NAME>:` after fixes are committed.

## 12. Repository and Workspace Layout

The container must mount a host workspace with the following structure:

- `<WORKSPACE_DIR>/repos/<repo-name>`: cloned repositories for session use.
- `<WORKSPACE_DIR>/sessions/<agent-name>`: session metadata, logs, and temporary state.

The agent must be able to:

- Reuse an existing clone if present.
- Cleanly check out the session branch.
- Keep temporary state isolated per agent to avoid cross-session contamination.

## 13. Logging, Observability, and Auditability

The system must emit structured logs that include:

- Timestamp, agent name, session branch, and cycle type.
- GitHub API/CLI actions performed and their outcomes.
- Decision rationale (e.g., why an issue or requirement change was selected).

All logs should be stored in a session-specific log file, and the most recent operations should be printed to stdout for container logs.

## 14. Error Handling and Recovery

The agent must gracefully handle common failures:

- GitHub API failures or rate limits → exponential backoff and retry.
- Missing or invalid credentials → clear error and halt cycle.
- Git checkout errors → clean working tree and retry once.
- PR already merged during cycle → skip remaining steps, trigger session exit logic.

Any failed cycle must log the failure reason and re-enter the main loop after a cooldown.

## 15. Security and Access Control

- GitHub tokens must never be logged or echoed.
- Only required scopes should be used (repo, issues, PRs).
- Sensitive data in config files must be mounted via environment variables rather than committed files.
- Container images should avoid baking secrets.

## 16. Acceptance Criteria

The following outcomes must be demonstrable:

- A new session branch triggers PR creation and starts a session loop.
- An issue labeled `agent_to_fix` is processed end-to-end with label updates and a prefixed agent comment.
- Requirement changes in `REQUIREMENTS.md` trigger the implementation cycle.
- When a PR is merged, the session exits and the agent sends the `/compact` instruction to the code agent.

## 17. Prompt Template Requirements

The agent must use consistent, structured prompts when invoking the code agent (`codex` CLI). Prompts must be deterministic, include all required context, and be verifiable in unit tests. Use the templates below verbatim (including headings and labels), replacing placeholders with real values. Test cases must validate exact prompt strings against these templates.

### 17.1 Issue Fix Prompt Template

```
TASK_TYPE: ISSUE_FIX
AGENT_NAME: <agent-name>
SESSION_BRANCH: <branch-name>
PR_URL: <pr-url>

## Issue Summary
TITLE: <issue-title>
URL: <issue-url>
LABELS: <label-1>, <label-2>, <label-n>

## Issue Description
<full-issue-body-verbatim>

## Issue Comments
COMMENT 1 BY <author>: <comment-body-1>
COMMENT 2 BY <author>: <comment-body-2>
COMMENT n BY <author>: <comment-body-n>

## Repo Context
REPO_URL: <repo-url>
BASE_BRANCH: <master-or-main>
CURRENT_BRANCH: <current-branch>

## Instructions
Fix the issue based on the description and comments.
Make minimal changes and preserve style.
Report what changed and why.
```

### 17.2 Requirement Change Prompt Template

```
TASK_TYPE: REQUIREMENT_CHANGE
AGENT_NAME: <agent-name>
SESSION_BRANCH: <branch-name>
PR_URL: <pr-url>

## Requirement Files Snapshot
### REQUIREMENTS.md
<full-requirements-md-verbatim>

### CICD_REQUIREMENTS.md
<full-cicd-requirements-md-verbatim-or-empty-if-missing>

### OTHER_REFERENCED_REQUIREMENTS
<full-other-requirements-files-verbatim>

## Requirement Diff
<git-diff-output-for-requirement-files>

## Repo Context
REPO_URL: <repo-url>
BASE_BRANCH: <master-or-main>
CURRENT_BRANCH: <current-branch>

## Instructions
Implement the requirement changes described in the diff.
Ensure changes align with the full requirements snapshot.
Summarize updates for commit messaging.
```
