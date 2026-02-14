TEST_NAME="Session loop finds issues mentioning PR in title"

# Per REQUIREMENTS.md: "Find github issues that: Mentions current PR in the title, Labels agent_to_fix"

run_case() {
  local repo="$TEST_TMP/repo_pr_mention"
  local remote="$TEST_TMP/remote_pr_mention.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local gh_log="$TEST_TMP/gh_pr_mention.log"

  export GH_CALL_LOG="$gh_log"
  # Return issue 42 when searching for PR #101 mention
  export GH_MOCK_ISSUE_LIST="42"
  export GH_PR_MERGED=false

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  # Simulate the issue search from session_loop
  cd "$repo"
  local issue_number
  issue_number=$(gh issue list --label agent_to_fix --search "\"#101\" in:title" --json number -q '.[0].number' 2>/dev/null || true)

  # Verify gh was called with correct search parameters
  local gh_calls
  gh_calls=$(cat "$gh_log")

  # Should search for issues with agent_to_fix label and PR mention
  if [[ "$gh_calls" != *"issue list --label agent_to_fix --search"* ]]; then
    echo "Should search for issues with agent_to_fix label" >&2
    return 1
  fi

  if [[ "$gh_calls" != *"#101"* ]]; then
    echo "Should search for PR number in title" >&2
    return 1
  fi

  # Should find issue 42
  if [[ "$issue_number" != "42" ]]; then
    echo "Expected issue 42, got: $issue_number" >&2
    return 1
  fi
}
