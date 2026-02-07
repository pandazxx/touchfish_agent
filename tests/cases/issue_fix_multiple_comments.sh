TEST_NAME="Issue fix prompt includes multiple comments"
DETAIL_SECTIONS="6.3.1,11,17.1"

INPUT_ISSUE_NUMBER=101
INPUT_PR_NUMBER=55
INPUT_SESSION_BRANCH="agent/unit_agent/feature-123"
INPUT_BASE_BRANCH="main"
INPUT_PR_URL="https://example.com/pr/55"

run_case() {
  local repo="$TEST_TMP/repo_issue"
  local remote="$TEST_TMP/remote_issue.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local gh_log="$TEST_TMP/gh_issue.log"
  local codex_log="$TEST_TMP/codex_issue.log"

  export GH_CALL_LOG="$gh_log"
  export GH_MOCK_ISSUE_JSON="$ROOT_DIR/tests/data/issue_with_comments.json"
  export CODEX_PROMPT_LOG="$codex_log"
  export CODEX_OUTPUT_FILE="$repo/fix.txt"
  export CODEX_CMD="codex"

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"
  export REPO_DIR="$repo"
  export STATE_DIR="$TEST_TMP/state"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  (cd "$repo" && issue_fix_cycle "$INPUT_ISSUE_NUMBER" "$INPUT_PR_NUMBER")

  local prompt
  prompt=$(cat "$codex_log")

  local expected_prompt
  expected_prompt=$(
    cat <<EOF
TASK_TYPE: ISSUE_FIX
AGENT_NAME: unit_agent
SESSION_BRANCH: $INPUT_SESSION_BRANCH
PR_URL: $INPUT_PR_URL

## Issue Summary
TITLE: Fix incorrect prompt handling
URL: https://example.com/issues/101
LABELS: agent_to_fix, bug

## Issue Description
The agent fails to include context in its prompt.

## Issue Comments
COMMENT 1 BY alice: First comment with extra context.
COMMENT 2 BY bob: Second comment with more details.

## Repo Context
REPO_URL: file://$remote
BASE_BRANCH: $INPUT_BASE_BRANCH
CURRENT_BRANCH: $INPUT_SESSION_BRANCH

## Instructions
Fix the issue based on the description and comments.
Make minimal changes and preserve style.
Report what changed and why.
EOF
  )

  assert_equal "$prompt" "$expected_prompt"
}
