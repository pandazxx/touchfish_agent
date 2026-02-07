TEST_NAME="Requirement change prompt includes snapshots and diff"
DETAIL_SECTIONS="6.3.2,8,17.2"

INPUT_SESSION_BRANCH="agent/unit_agent/feature-456"
INPUT_BASE_BRANCH="main"
INPUT_PR_URL="https://example.com/pr/88"
INPUT_REQUIREMENTS_CONTENT=$'# Requirements\n\nInitial requirements.\nUpdated requirements.\n'
INPUT_CICD_CONTENT=$'# CI/CD Requirements\n\nPipeline requirements.\n'
INPUT_OTHER_CONTENT=$'# Additional Requirements\n\nExtra requirements for integrations.\n'
INPUT_MOCK_DIFF=$'diff --git a/REQUIREMENTS.md b/REQUIREMENTS.md\nindex 1111111..2222222 100644\n--- a/REQUIREMENTS.md\n+++ b/REQUIREMENTS.md\n@@ -1,3 +1,4 @@\n # Requirements\n \n Initial requirements.\n+Updated requirements.\n'

run_case() {
  local repo="$TEST_TMP/repo_req"
  local remote="$TEST_TMP/remote_req.git"
  local other_file="$repo/EXTRA_REQUIREMENTS.md"
  setup_repo "$repo" "$remote"

  printf '%s' "$INPUT_REQUIREMENTS_CONTENT" > "$repo/REQUIREMENTS.md"
  printf '%s' "$INPUT_CICD_CONTENT" > "$repo/CICD_REQUIREMENTS.md"
  printf '%s' "$INPUT_OTHER_CONTENT" > "$other_file"

  git -C "$repo" add REQUIREMENTS.md CICD_REQUIREMENTS.md EXTRA_REQUIREMENTS.md
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local last_sha
  last_sha=$(git -C "$repo" rev-parse HEAD)

  printf '%s' "$INPUT_REQUIREMENTS_CONTENT" > "$repo/REQUIREMENTS.md"

  local codex_log="$TEST_TMP/codex_req.log"
  export CODEX_PROMPT_LOG="$codex_log"
  export CODEX_OUTPUT_FILE="$repo/impl.txt"
  export CODEX_CMD="codex"
  export GIT_MOCK_DIFF="$INPUT_MOCK_DIFF"

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"
  export REPO_DIR="$repo"
  export STATE_DIR="$TEST_TMP/state"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  (cd "$repo" && implementing_cycle "$last_sha" "$repo/REQUIREMENTS.md" "$repo/CICD_REQUIREMENTS.md" "$other_file")

  local prompt
  prompt=$(cat "$codex_log")

  local expected_prompt
  expected_prompt=$(
    cat <<EOF
TASK_TYPE: REQUIREMENT_CHANGE
AGENT_NAME: unit_agent
SESSION_BRANCH: $INPUT_SESSION_BRANCH
PR_URL: $INPUT_PR_URL

## Requirement Files Snapshot
### REQUIREMENTS.md
$INPUT_REQUIREMENTS_CONTENT

### CICD_REQUIREMENTS.md
$INPUT_CICD_CONTENT

### OTHER_REFERENCED_REQUIREMENTS
$INPUT_OTHER_CONTENT

## Requirement Diff
$INPUT_MOCK_DIFF

## Repo Context
REPO_URL: file://$remote
BASE_BRANCH: $INPUT_BASE_BRANCH
CURRENT_BRANCH: $INPUT_SESSION_BRANCH

## Instructions
Implement the requirement changes described in the diff.
Ensure changes align with the full requirements snapshot.
Summarize updates for commit messaging.
EOF
  )

  assert_equal "$prompt" "$expected_prompt"
}
