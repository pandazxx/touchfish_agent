TEST_NAME="No matching agent branch returns empty"

run_case() {
  local repo="$TEST_TMP/repo_nobranch"
  local remote="$TEST_TMP/remote_nobranch.git"
  setup_repo "$repo" "$remote"

  # Create initial commit on main only
  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "initial" >/dev/null
  git -C "$repo" branch -M main
  git -C "$repo" push -u origin main >/dev/null

  # Create a branch that doesn't match the agent pattern
  git -C "$repo" checkout -b "feature/something" >/dev/null
  echo "feature" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "feature" >/dev/null
  git -C "$repo" push -u origin "feature/something" >/dev/null

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  cd "$repo"
  local found_branch
  found_branch=$(find_agent_branch "main")

  # Should return empty when no matching branch exists
  if [[ -n "$found_branch" ]]; then
    echo "Expected empty result but got '$found_branch'" >&2
    return 1
  fi
}
