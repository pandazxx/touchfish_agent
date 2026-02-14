TEST_NAME="Implementing cycle with multiple requirement files"

INPUT_REQ_CONTENT=$'# Requirements\n\nFeature requirements.\n'
INPUT_CICD_CONTENT=$'# CI/CD Requirements\n\nPipeline requirements.\n'
INPUT_MOCK_DIFF=$'diff --git a/REQUIREMENTS.md b/REQUIREMENTS.md\nindex 1111111..2222222 100644\n--- a/REQUIREMENTS.md\n+++ b/REQUIREMENTS.md\n@@ -1,3 +1,3 @@\n # Requirements\n \n-Old feature.\n+Feature requirements.\ndiff --git a/CICD_REQUIREMENTS.md b/CICD_REQUIREMENTS.md\nindex 3333333..4444444 100644\n--- a/CICD_REQUIREMENTS.md\n+++ b/CICD_REQUIREMENTS.md\n@@ -1,3 +1,3 @@\n # CI/CD Requirements\n \n-Old pipeline.\n+Pipeline requirements.\n'

run_case() {
  local repo="$TEST_TMP/repo_multi"
  local remote="$TEST_TMP/remote_multi.git"
  setup_repo "$repo" "$remote"

  printf '%s' "$INPUT_REQ_CONTENT" > "$repo/REQUIREMENTS.md"
  printf '%s' "$INPUT_CICD_CONTENT" > "$repo/CICD_REQUIREMENTS.md"

  git -C "$repo" add REQUIREMENTS.md CICD_REQUIREMENTS.md
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local last_sha
  last_sha=$(git -C "$repo" rev-parse HEAD)

  local codex_log="$TEST_TMP/codex_multi.log"
  export CODEX_PROMPT_LOG="$codex_log"
  export CODEX_OUTPUT_FILE="$repo/impl.txt"
  export CODEX_CMD="codex"
  export GIT_MOCK_DIFF="$INPUT_MOCK_DIFF"

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  (cd "$repo" && implementing_cycle "$last_sha" "$repo/REQUIREMENTS.md" "$repo/CICD_REQUIREMENTS.md")

  local prompt
  prompt=$(cat "$codex_log")

  # Verify both files are included in prompt
  assert_contains "$prompt" "----- $repo/REQUIREMENTS.md -----"
  assert_contains "$prompt" "Feature requirements."
  assert_contains "$prompt" "----- $repo/CICD_REQUIREMENTS.md -----"
  assert_contains "$prompt" "Pipeline requirements."

  # Verify diff contains both files
  assert_contains "$prompt" "diff --git a/REQUIREMENTS.md"
  assert_contains "$prompt" "diff --git a/CICD_REQUIREMENTS.md"
}
