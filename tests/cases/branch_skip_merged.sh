TEST_NAME="Branch discovery skips merged branches"

run_case() {
  local repo="$TEST_TMP/repo_merged"
  local remote="$TEST_TMP/remote_merged.git"
  setup_repo "$repo" "$remote"

  # Create initial commit on main
  echo "base" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "initial" >/dev/null
  git -C "$repo" branch -M main
  git -C "$repo" push -u origin main >/dev/null

  # Create and merge first agent branch
  git -C "$repo" checkout -b "agent/unit_agent/merged_session" >/dev/null
  echo "merged change" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "merged change" >/dev/null
  git -C "$repo" push -u origin "agent/unit_agent/merged_session" >/dev/null

  # Merge into main
  git -C "$repo" checkout main >/dev/null
  git -C "$repo" merge "agent/unit_agent/merged_session" -m "merge" >/dev/null
  git -C "$repo" push origin main >/dev/null

  # Create unmerged agent branch
  git -C "$repo" checkout -b "agent/unit_agent/active_session" >/dev/null
  echo "active change" > "$repo/file.txt"
  git -C "$repo" add file.txt
  git -C "$repo" commit -m "active change" >/dev/null
  git -C "$repo" push -u origin "agent/unit_agent/active_session" >/dev/null

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  cd "$repo"
  local found_branch
  found_branch=$(find_agent_branch "main")

  # Should find the unmerged branch, not the merged one
  if [[ "$found_branch" != "agent/unit_agent/active_session" ]]; then
    echo "Expected 'agent/unit_agent/active_session' but got '$found_branch'" >&2
    return 1
  fi
}
