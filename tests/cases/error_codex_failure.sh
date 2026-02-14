TEST_NAME="Issue fix handles codex execution failure gracefully"

run_case() {
  local repo="$TEST_TMP/repo_codex_fail"
  local remote="$TEST_TMP/remote_codex_fail.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local initial_sha
  initial_sha=$(git -C "$repo" rev-parse HEAD)

  local gh_log="$TEST_TMP/gh_codex_fail.log"

  export GH_CALL_LOG="$gh_log"
  export GH_MOCK_ISSUE_JSON="$ROOT_DIR/tests/data/issue_with_comments.json"
  # Set CODEX_CMD to a non-existent command to simulate failure
  export CODEX_CMD="nonexistent_codex_command_12345"

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  # Should not crash, should handle gracefully
  (cd "$repo" && issue_fix_cycle 42 101) || true

  local final_sha
  final_sha=$(git -C "$repo" rev-parse HEAD)

  # No commit should be created when codex fails
  if [[ "$initial_sha" != "$final_sha" ]]; then
    echo "Should not create commit when codex fails" >&2
    return 1
  fi

  local gh_calls
  gh_calls=$(cat "$gh_log")

  # Should NOT transition to pending_verify when codex fails
  if [[ "$gh_calls" == *"agent_pending_verify"* ]]; then
    echo "Should not transition to pending_verify when codex fails" >&2
    return 1
  fi

  # Label should have been set to fixing at start
  if [[ "$gh_calls" != *"--add-label agent_fixing"* ]]; then
    echo "Should have attempted to set fixing label" >&2
    return 1
  fi
}
