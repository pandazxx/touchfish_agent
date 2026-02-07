TEST_NAME="PR merged triggers compact"
DETAIL_SECTIONS="6.4,10,16"

run_case() {
  local repo="$TEST_TMP/repo_pr"
  local remote="$TEST_TMP/remote_pr.git"
  setup_repo "$repo" "$remote"

  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "base" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local gh_log="$TEST_TMP/gh_pr.log"
  export GH_CALL_LOG="$gh_log"
  export GH_PR_MERGED=true

  local codex_log="$TEST_TMP/codex_pr.log"
  export CODEX_PROMPT_LOG="$codex_log"
  export CODEX_OUTPUT_FILE="$TEST_TMP/pr_impl.txt"
  export CODEX_CMD="codex"

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"
  export REPO_DIR="$repo"
  export STATE_DIR="$TEST_TMP/state"
  export POLL_INTERVAL=1

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  session_loop 55 "agent/unit_agent/test" "main"

  if [[ ! -f "$codex_log" ]]; then
    echo "Expected codex prompt for compact not found" >&2
    return 1
  fi

  local prompt
  prompt=$(cat "$codex_log")
  assert_equal "$prompt" "/compact"
}
