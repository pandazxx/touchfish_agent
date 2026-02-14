TEST_NAME="Branch discovery finds matching agent branch"

run_case() {
  local repo="$TEST_TMP/repo_branch"
  local remote="$TEST_TMP/remote_branch.git"
  setup_repo "$repo" "$remote"

  # Create initial commit on main
  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "initial" >/dev/null
  git -C "$repo" branch -M main
  git -C "$repo" push -u origin main >/dev/null

  # Create agent branch
  git -C "$repo" checkout -b "agent/unit_agent/session1" >/dev/null
  echo "change" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "agent change" >/dev/null
  git -C "$repo" push -u origin "agent/unit_agent/session1" >/dev/null

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  cd "$repo"
  local found_branch
  found_branch=$(find_agent_branch "main")

  if [[ "$found_branch" != "agent/unit_agent/session1" ]]; then
    echo "Expected branch 'agent/unit_agent/session1' but got '$found_branch'" >&2
    return 1
  fi
}
