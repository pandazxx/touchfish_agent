TEST_NAME="PR is created when none exists"

run_case() {
  local repo="$TEST_TMP/repo_prcreate"
  local remote="$TEST_TMP/remote_prcreate.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "initial" >/dev/null
  git -C "$repo" branch -M main
  git -C "$repo" push -u origin main >/dev/null

  local gh_log="$TEST_TMP/gh_prcreate.log"

  export GH_CALL_LOG="$gh_log"
  # Empty PR list means no existing PR
  export GH_MOCK_PR_LIST=""

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  cd "$repo"
  ensure_pr "agent/unit_agent/session1" "main" >/dev/null

  local gh_calls
  gh_calls=$(cat "$gh_log")

  # Verify PR create was called with correct parameters
  if [[ "$gh_calls" != *"pr create --head agent/unit_agent/session1 --base main"* ]]; then
    echo "Expected PR create command not found" >&2
    echo "Got: $gh_calls" >&2
    return 1
  fi

  # Verify PR title includes agent name and branch
  if [[ "$gh_calls" != *"--title Agent unit_agent session: agent/unit_agent/session1"* ]]; then
    echo "Expected PR title format not found" >&2
    echo "Got: $gh_calls" >&2
    return 1
  fi
}
