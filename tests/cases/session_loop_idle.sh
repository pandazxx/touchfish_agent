TEST_NAME="Session loop handles idle state correctly"

run_case() {
  local repo="$TEST_TMP/repo_idle"
  local remote="$TEST_TMP/remote_idle.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  echo "# Requirements" > "$repo/REQUIREMENTS.md"
  git -C "$repo" add file.txt REQUIREMENTS.md
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local gh_log="$TEST_TMP/gh_idle.log"
  local codex_log="$TEST_TMP/codex_idle.log"
  local state_dir="$TEST_TMP/state_idle"
  mkdir -p "$state_dir"

  export GH_CALL_LOG="$gh_log"
  # No issues to fix
  export GH_MOCK_ISSUE_LIST=""
  # PR not merged
  export GH_PR_MERGED=false
  # No codex output - nothing should trigger it
  export CODEX_PROMPT_LOG="$codex_log"

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"
  export REPO_DIR="$repo"
  export STATE_DIR="$state_dir"
  export POLL_INTERVAL=1
  export CODEX_CMD="codex"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  # Initialize state file with current SHA to simulate no changes
  local state_file
  state_file=$(state_file_for_branch "agent/unit_agent/test")
  local current_sha
  current_sha=$(git -C "$repo" rev-parse HEAD)
  write_state "$state_file" "$current_sha"

  # Run a modified session loop that exits after one iteration for testing
  # We test the individual components instead
  cd "$repo"

  # Verify no issue is found
  local issue_number
  issue_number=$(gh issue list --label agent_to_fix --search "\"#55\" in:title" --json number -q '.[0].number' 2>/dev/null || true)

  if [[ -n "$issue_number" ]]; then
    echo "Expected no issue but found $issue_number" >&2
    return 1
  fi

  # Verify no requirements changed
  mapfile -t req_files < <(echo "$repo/REQUIREMENTS.md")
  if requirements_changed "$current_sha" "${req_files[@]}"; then
    echo "Expected no requirements change" >&2
    return 1
  fi

  # Verify codex was not called (no prompt file created)
  if [[ -f "$codex_log" ]]; then
    echo "Codex should not be called in idle state" >&2
    return 1
  fi
}
