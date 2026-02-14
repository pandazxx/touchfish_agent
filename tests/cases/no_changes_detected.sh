TEST_NAME="No commit when codex produces no changes"

run_case() {
  local repo="$TEST_TMP/repo_nochange"
  local remote="$TEST_TMP/remote_nochange.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local initial_sha
  initial_sha=$(git -C "$repo" rev-parse HEAD)

  local gh_log="$TEST_TMP/gh_nochange.log"
  local codex_log="$TEST_TMP/codex_nochange.log"

  export GH_CALL_LOG="$gh_log"
  export GH_MOCK_ISSUE_JSON="$ROOT_DIR/tests/data/issue_with_comments.json"
  export CODEX_PROMPT_LOG="$codex_log"
  # Do NOT set CODEX_OUTPUT_FILE - codex won't create any files

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"
  export CODEX_CMD="codex"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  (cd "$repo" && issue_fix_cycle 42 101)

  local final_sha
  final_sha=$(git -C "$repo" rev-parse HEAD)

  # No new commit should be created
  if [[ "$initial_sha" != "$final_sha" ]]; then
    echo "Expected no commit, but SHA changed from $initial_sha to $final_sha" >&2
    return 1
  fi

  # Should not have added comment or updated label to pending_verify
  local gh_calls
  gh_calls=$(cat "$gh_log")

  if [[ "$gh_calls" == *"agent_pending_verify"* ]]; then
    echo "Should not transition to agent_pending_verify when no changes made" >&2
    return 1
  fi
}
