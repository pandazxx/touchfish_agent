TEST_NAME="PR is updated after issue fix"

run_case() {
  local repo="$TEST_TMP/repo_prupdate"
  local remote="$TEST_TMP/remote_prupdate.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local gh_log="$TEST_TMP/gh_prupdate.log"
  local codex_log="$TEST_TMP/codex_prupdate.log"

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

  local gh_calls
  gh_calls=$(cat "$gh_log")

  # Verify PR comment was added
  if [[ "$gh_calls" != *"pr comment 101 --body Agent unit_agent updated: fixed issue #42."* ]]; then
    echo "Expected PR update comment not found" >&2
    echo "Got: $gh_calls" >&2
    return 1
  fi
}
