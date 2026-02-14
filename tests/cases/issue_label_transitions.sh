TEST_NAME="Issue fix updates labels correctly"

run_case() {
  local repo="$TEST_TMP/repo_labels"
  local remote="$TEST_TMP/remote_labels.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local gh_log="$TEST_TMP/gh_labels.log"
  local codex_log="$TEST_TMP/codex_labels.log"

  export GH_CALL_LOG="$gh_log"
  export GH_MOCK_ISSUE_JSON="$ROOT_DIR/tests/data/issue_with_comments.json"
  export CODEX_PROMPT_LOG="$codex_log"
  export CODEX_OUTPUT_FILE="$repo/fix.txt"

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"
  export CODEX_CMD="codex"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  (cd "$repo" && issue_fix_cycle 42 101)

  # Verify label transitions were called
  local gh_calls
  gh_calls=$(cat "$gh_log")

  # Check agent_to_fix -> agent_fixing transition
  if [[ "$gh_calls" != *"issue edit 42 --add-label agent_fixing --remove-label agent_to_fix"* ]]; then
    echo "Expected label transition to agent_fixing not found in gh calls" >&2
    echo "Got: $gh_calls" >&2
    return 1
  fi

  # Check agent_fixing -> agent_pending_verify transition
  if [[ "$gh_calls" != *"issue edit 42 --add-label agent_pending_verify --remove-label agent_fixing"* ]]; then
    echo "Expected label transition to agent_pending_verify not found in gh calls" >&2
    echo "Got: $gh_calls" >&2
    return 1
  fi
}
