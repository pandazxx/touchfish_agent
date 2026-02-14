TEST_NAME="Requirements change detection works correctly"

run_case() {
  local repo="$TEST_TMP/repo_detect"
  local remote="$TEST_TMP/remote_detect.git"
  setup_repo "$repo" "$remote"

  echo "# Initial" > "$repo/REQUIREMENTS.md"
  git -C "$repo" add REQUIREMENTS.md
  git -C "$repo" commit -m "initial" >/dev/null
  git -C "$repo" push -u origin HEAD >/dev/null

  local base_sha
  base_sha=$(git -C "$repo" rev-parse HEAD)

  # Make a change to requirements
  echo "# Updated" > "$repo/REQUIREMENTS.md"
  git -C "$repo" add REQUIREMENTS.md
  git -C "$repo" commit -m "update" >/dev/null

  export AGENT_LIBRARY_MODE=1
  export AGENT_NAME="unit_agent"
  export GITHUB_TOKEN="dummy"
  export REPO_URL="file://$remote"

  # shellcheck source=/dev/null
  source "$ROOT_DIR/scripts/agent.sh"

  cd "$repo"

  # Should detect change
  if ! requirements_changed "$base_sha" "$repo/REQUIREMENTS.md"; then
    echo "Expected to detect requirements change" >&2
    return 1
  fi

  # Update base_sha to current - should NOT detect change
  local current_sha
  current_sha=$(git rev-parse HEAD)

  if requirements_changed "$current_sha" "$repo/REQUIREMENTS.md"; then
    echo "Should not detect change when at same commit" >&2
    return 1
  fi
}
