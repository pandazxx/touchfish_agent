TEST_NAME="Issue can be reprocessed after verification fails"

# This test verifies that when an issue cycles back from agent_pending_verify
# to agent_to_fix, the agent can process it again with all comments including
# the previous fix attempt.

INPUT_ISSUE_NUMBER=42
INPUT_PR_NUMBER=101

run_case() {
  local repo="$TEST_TMP/repo_reverify"
  local remote="$TEST_TMP/remote_reverify.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local gh_log="$TEST_TMP/gh_reverify.log"
  local codex_log="$TEST_TMP/codex_reverify.log"

  # Create mock issue with previous agent comment (simulating re-verification scenario)
  local mock_issue="$TEST_TMP/reverify_issue.json"
  cat > "$mock_issue" <<'EOF'
{
  "title": "Fix authentication bug #101",
  "body": "Login fails with invalid token.",
  "comments": [
    {"author": {"login": "user1"}, "body": "This happens on the signup page too."},
    {"author": {"login": "test_agent"}, "body": "test_agent: implemented fix in commit abc123."},
    {"author": {"login": "user1"}, "body": "Still broken, the fix didn't work for edge cases."}
  ]
}
EOF

  export GH_CALL_LOG="$gh_log"
  export GH_MOCK_ISSUE_JSON="$mock_issue"
  export CODEX_PROMPT_LOG="$codex_log"
  export CODEX_OUTPUT_FILE="$repo/fix.txt"

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="test_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"
  export CODEX_CMD="codex"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  (cd "$repo" && issue_fix_cycle "$INPUT_ISSUE_NUMBER" "$INPUT_PR_NUMBER")

  local prompt
  prompt=$(cat "$codex_log")

  # Verify prompt includes all comments including previous agent fix attempt
  assert_contains "$prompt" "This happens on the signup page too."
  assert_contains "$prompt" "test_agent: implemented fix in commit abc123."
  assert_contains "$prompt" "Still broken, the fix didn't work for edge cases."
}
