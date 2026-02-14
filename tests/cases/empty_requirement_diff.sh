TEST_NAME="Implementing cycle handles empty diff gracefully"

run_case() {
  local repo="$TEST_TMP/repo_empty_diff"
  local remote="$TEST_TMP/remote_empty_diff.git"
  setup_repo "$repo" "$remote"

  echo "# Requirements" > "$repo/REQUIREMENTS.md"
  git -C "$repo" add REQUIREMENTS.md
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local last_sha
  last_sha=$(git -C "$repo" rev-parse HEAD)

  local codex_log="$TEST_TMP/codex_empty_diff.log"

  export CODEX_PROMPT_LOG="$codex_log"
  export CODEX_OUTPUT_FILE="$repo/impl.txt"
  export CODEX_CMD="codex"
  # Empty diff - no changes between commits
  export GIT_MOCK_DIFF=""

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  (cd "$repo" && implementing_cycle "$last_sha" "$repo/REQUIREMENTS.md")

  local prompt
  prompt=$(cat "$codex_log")

  # Should still include the requirements file content
  assert_contains "$prompt" "# Requirements"
  assert_contains "$prompt" "----- $repo/REQUIREMENTS.md -----"

  # Diff section should be empty but present
  assert_contains "$prompt" "Changed requirements diff:"
}
