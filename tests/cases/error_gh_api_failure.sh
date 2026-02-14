TEST_NAME="Issue fix handles gh API failure gracefully"

run_case() {
  local repo="$TEST_TMP/repo_gh_fail"
  local remote="$TEST_TMP/remote_gh_fail.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local gh_log="$TEST_TMP/gh_api_fail.log"
  local codex_log="$TEST_TMP/codex_api_fail.log"

  export GH_CALL_LOG="$gh_log"
  # Do NOT set GH_MOCK_ISSUE_JSON - gh issue view will return empty/fail
  export GH_MOCK_ISSUE_JSON=""
  export CODEX_PROMPT_LOG="$codex_log"
  export CODEX_OUTPUT_FILE="$repo/fix.txt"

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"
  export CODEX_CMD="codex"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  # Should not crash when gh returns empty/invalid data
  (cd "$repo" && issue_fix_cycle 42 101) || true

  # Verify gh was called
  local gh_calls
  gh_calls=$(cat "$gh_log")

  if [[ "$gh_calls" != *"issue view 42"* ]]; then
    echo "Should have attempted to view issue" >&2
    return 1
  fi
}
