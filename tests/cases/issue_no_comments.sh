TEST_NAME="Issue fix works with no comments"

run_case() {
  local repo="$TEST_TMP/repo_no_comments"
  local remote="$TEST_TMP/remote_no_comments.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  # Create mock issue with no comments
  local mock_issue="$TEST_TMP/issue_no_comments.json"
  cat > "$mock_issue" <<'EOF'
{
  "title": "Simple bug fix needed",
  "body": "The button does not work.",
  "comments": []
}
EOF

  local gh_log="$TEST_TMP/gh_no_comments.log"
  local codex_log="$TEST_TMP/codex_no_comments.log"

  export GH_CALL_LOG="$gh_log"
  export GH_MOCK_ISSUE_JSON="$mock_issue"
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

  local prompt
  prompt=$(cat "$codex_log")

  # Verify prompt structure is correct even without comments
  local expected_prompt
  expected_prompt=$(cat <<'EOF'
You are an automated coding agent. Fix the issue below in this repository.

Issue: Simple bug fix needed

The button does not work.

Comments:
EOF
  )

  assert_equal "$prompt" "$expected_prompt"
}
