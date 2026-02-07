#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
REPORT_PATH=${TEST_REPORT:-"$ROOT_DIR/tests/report.txt"}
VERBOSE=${UNIT_TEST_VERBOSE:-0}
RUN_IN_CONTAINER_FLAG=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --verbose)
      VERBOSE=1
      shift
      ;;
    --no-container)
      RUN_IN_CONTAINER_FLAG=0
      shift
      ;;
    *)
      break
      ;;
  esac
done

run_in_container() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker is required to run the tests" >&2
    exit 1
  fi

  local container_id
  local local_report
  local_report="$REPORT_PATH"

  docker build -t touchfish_agent_test "$ROOT_DIR" >/dev/null
  container_id=$(docker create \
    --entrypoint /bin/bash \
    -e RUN_IN_CONTAINER=1 \
    -e TEST_REPORT="/work/tests/report.txt" \
    -e UNIT_TEST_VERBOSE="$VERBOSE" \
    -e AGENT_NAME="unit_agent" \
    -e GITHUB_TOKEN="test_token" \
    -e REPO_URL="https://example.com/repo.git" \
    -e WORKSPACE_DIR="/work" \
    touchfish_agent_test \
    /work/tests/unit_test.sh)

  docker cp "$ROOT_DIR" "${container_id}:/work"
  docker start -a "$container_id"
  mkdir -p "$(dirname "$local_report")"
  docker cp "${container_id}:/work/tests/report.txt" "$local_report" >/dev/null 2>&1 || true
  docker rm "$container_id" >/dev/null
}

if [[ "${RUN_IN_CONTAINER:-}" != "1" ]]; then
  if [[ "$RUN_IN_CONTAINER_FLAG" == "1" ]]; then
    run_in_container
    exit $?
  fi
fi

GIT_REAL_BIN=$(command -v git)
export GIT_REAL_BIN
export GIT_CALL_LOG=${GIT_CALL_LOG:-"$ROOT_DIR/tests/git_calls.log"}
PATH="$ROOT_DIR/tests/mocks:$PATH"

TEST_TMP=$(mktemp -d)
mkdir -p "$(dirname "$REPORT_PATH")"
: > "$REPORT_PATH"
exec > >(tee -a "$REPORT_PATH") 2>&1

if [[ "$VERBOSE" == "1" ]]; then
  exec 3>&2
else
  exec 3>>"$REPORT_PATH"
fi
export BASH_XTRACEFD=3
export PS4='+ ${BASH_SOURCE##*/}:${LINENO}: '
set -x

cleanup() {
  rm -rf "$TEST_TMP"
}
trap cleanup EXIT

log_report() {
  echo "$*" | tee -a "$REPORT_PATH"
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  if [[ "$haystack" != *"$needle"* ]]; then
    echo "Expected to find: $needle" >&2
    return 1
  fi
}

assert_equal() {
  local actual="$1"
  local expected="$2"
  if [[ "$actual" != "$expected" ]]; then
    echo "Expected full prompt mismatch" >&2
    return 1
  fi
}

setup_repo() {
  local repo="$1"
  local remote="$2"

  mkdir -p "$repo" "$remote"
  git init -q "$repo"
  git init -q --bare "$remote"
  git -C "$repo" config user.email "test@example.com"
  git -C "$repo" config user.name "Test"
  git -C "$repo" remote add origin "$remote"
}

reset_test_env() {
  unset GH_CALL_LOG GH_MOCK_ISSUE_JSON GH_MOCK_ISSUE_LIST GH_PR_MERGED GH_MOCK_PR_LIST
  unset CODEX_PROMPT_LOG CODEX_OUTPUT_FILE CODEX_CMD
  unset GIT_MOCK_DIFF
}

run_test_case() {
  local case_file="$1"

  reset_test_env

  # shellcheck source=/dev/null
  source "$case_file"

  if [[ -z "${TEST_NAME:-}" || -z "${DETAIL_SECTIONS:-}" || -z "${run_case:-}" ]]; then
    log_report "FAIL: invalid test case $case_file"
    return 1
  fi

  log_report "CASE: $TEST_NAME (DETAIL_REQUIREMENT.md: $DETAIL_SECTIONS)"

  if run_case; then
    log_report "PASS: $TEST_NAME"
  else
    log_report "FAIL: $TEST_NAME"
    return 1
  fi
}

failures=0

for case_file in "$ROOT_DIR/tests/cases/"*.sh; do
  run_test_case "$case_file" || failures=$((failures+1))
done

if [[ $failures -ne 0 ]]; then
  log_report "FAILED: $failures test(s) failed"
  exit 1
fi

log_report "SUCCESS: all tests passed"
