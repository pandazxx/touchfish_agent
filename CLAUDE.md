# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AI workflow automation agent that uses GitHub as the primary interface. The agent monitors branches, handles issues, and implements requirements automatically via an AI code agent (codex/claude).

## Commands

### Build & Run
```bash
docker compose build                    # Build the agent image
docker compose build --no-cache         # Clean build
docker compose up --build               # Start the agent
docker compose down                     # Stop the agent
```

### Testing
```bash
./tests/unit_test.sh                    # Run tests in Docker (default)
./tests/unit_test.sh --no-container     # Run tests locally
./tests/unit_test.sh --verbose          # Enable verbose output
```

Test report output: `./tests/report.txt`

## Architecture

### Core Components

- **`scripts/agent.sh`** - Main agent logic (~400 lines). Contains the main loop, session loop, issue fixing cycle, and implementing cycle.
- **`scripts/entrypoint.sh`** - Container entry point
- **`scripts/requirements_files.py`** - Discovers requirement files recursively via regex

### Workflow Loops

1. **Main Loop**: Scans for branches matching `agent/<agent_name>/*`, checks out first unmerged match, creates PR if missing
2. **Session Loop**: Polls for issues labeled `agent_to_fix` or requirement file changes at `POLL_INTERVAL` seconds
3. **Issue Fixing Cycle**: Updates labels (`agent_to_fix` → `agent_fixing` → `agent_pending_verify`), invokes codex with issue content, commits changes
4. **Implementing Cycle**: Detects requirement diffs, invokes codex with full requirements + diff, commits changes
5. **Session Exit**: Detects merged PR, runs `/compact` on codex, returns to main loop

### Testing Structure

- **`tests/cases/`** - Individual test case files with input/expected variables
- **`tests/mocks/`** - Mock implementations for `gh`, `git`, and `codex`
- **`tests/data/`** - Test fixtures
- Source `agent.sh` in library mode with `AGENT_LIBRARY_MODE=1`
- Tests verify full codex prompt input (blackbox testing)

## Environment Variables

Required: `AGENT_NAME`, `GITHUB_TOKEN`, `REPO_URL`, `WORKSPACE_DIR`, `CODEX_SESSION_FILE`

Optional: `POLL_INTERVAL` (default: 60), `CODEX_CMD` (default: codex), `CODEX_API_KEY`, `OPENAI_API_KEY`, `AGENT_GIT_NAME`, `AGENT_GIT_EMAIL`

## Agent Conventions

- Address user as "Moses", greet with "What's up Moses"
- Store conversations in `CONVERSATION_HISTORY.md`
- Prefix all commit messages with "Aaron: "
- Requirements are defined in: `REQUIREMENTS.md`, `AGENT_REQUIREMENTS.md`, `CICD_REQUIREMENTS.md`

## GitHub Label Flow

`agent_to_fix` → `agent_fixing` → `agent_pending_verify`

Issues may cycle back to `agent_to_fix` if verification fails.
