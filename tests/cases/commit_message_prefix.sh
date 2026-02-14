TEST_NAME="Commit messages have Aaron prefix"

run_case() {
  local repo="$TEST_TMP/repo_prefix"
  local remote="$TEST_TMP/remote_prefix.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local git_log="$TEST_TMP/git_prefix.log"
  local gh_log="$TEST_TMP/gh_prefix.log"
  local codex_log="$TEST_TMP/codex_prefix.log"

  export GIT_CALL_LOG="$git_log"
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

  local git_calls
  git_calls=$(cat "$git_log")

  # Verify commit message starts with "Aaron:"
  if [[ "$git_calls" != *'commit -m Aaron: fix issue #42'* ]]; then
    echo "Expected commit message with 'Aaron:' prefix not found" >&2
    echo "Got: $git_calls" >&2
    return 1
  fi
}
