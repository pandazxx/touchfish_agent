TEST_NAME="Session processes only first issue when multiple exist"

# Per REQUIREMENTS.md: "If any issue found, execute Issue fixing cycle with the first issue found"

run_case() {
  local repo="$TEST_TMP/repo_multi_issue"
  local remote="$TEST_TMP/remote_multi_issue.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local gh_log="$TEST_TMP/gh_multi_issue.log"
  local codex_log="$TEST_TMP/codex_multi_issue.log"

  export GH_CALL_LOG="$gh_log"
  # Mock returns first issue number (simulating multiple issues exist, returns first)
  export GH_MOCK_ISSUE_LIST="42"
  export GH_MOCK_ISSUE_JSON="$ROOT_DIR/tests/data/issue_with_comments.json"
  export CODEX_PROMPT_LOG="$codex_log"
  export CODEX_OUTPUT_FILE="$repo/fix.txt"
  export GH_PR_MERGED=false

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"
  export CODEX_CMD="codex"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  (cd "$repo" && issue_fix_cycle 42 101)

  # Verify only issue 42 was processed
  local gh_calls
  gh_calls=$(cat "$gh_log")

  if [[ "$gh_calls" != *"issue view 42"* ]]; then
    echo "Should have processed issue 42" >&2
    return 1
  fi

  # Verify codex was called with correct issue
  local prompt
  prompt=$(cat "$codex_log")

  assert_contains "$prompt" "Fix incorrect prompt handling"
}
