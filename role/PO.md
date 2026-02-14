# Product Owner Review: TEST_REQUIREMENTS.md

**Reviewer**: PO
**Date**: 2026-02-03
**Status**: Review Complete

---

## Executive Summary

The TEST_REQUIREMENTS.md provides a comprehensive testing framework for the touchfish_agent project. The document is well-structured and aligns with the functional requirements in REQUIREMENTS.md and the workflow described in README.md. However, there are gaps and improvements needed before the test suite can be considered production-ready.

---

## Alignment Analysis

### Requirements Coverage Matrix

| Functional Requirement | Test Coverage | Status |
|------------------------|---------------|--------|
| Branch discovery (`agent/<name>/*`) | `branch_discovery.sh` | Covered |
| Skip merged branches | `branch_skip_merged.sh` | Covered |
| No matching branch | `no_agent_branch.sh` | Covered |
| PR creation | `pr_creation.sh` | Covered |
| Issue fix with comments | `issue_fix_multiple_comments.sh` | Covered |
| Label transitions (to_fix -> fixing -> pending_verify) | `issue_label_transitions.sh` | Covered |
| Comment prefix (`<agent_name>:`) | `issue_comment_prefix.sh` | Covered |
| Commit prefix (`Aaron:`) | `commit_message_prefix.sh` | Covered |
| PR update after fix | `pr_update_on_fix.sh` | Covered |
| Re-verification cycle | `issue_reverification.sh` | Covered |
| Requirement changes | `requirement_changes.sh` | Covered |
| Multiple requirement files | `multiple_requirement_files.sh` | Covered |
| PR merged -> /compact | `pr_merged.sh` | Covered |
| Idle state (no action) | `session_loop_idle.sh` | Covered |
| No changes detected | `no_changes_detected.sh` | Covered |

**Overall Coverage**: 15/15 core scenarios covered (100%)

---

## Findings

### Strengths

1. **Clear Blackbox Testing Philosophy**: The approach of treating `codex` as a blackbox and verifying exact prompts is sound for AI-driven systems where output is non-deterministic.

2. **Data-Driven Test Structure**: Separating test cases into individual files with clear INPUT/EXPECTED variables promotes maintainability and readability.

3. **Comprehensive Mock Strategy**: The mock configuration table clearly defines what to mock (gh, git network) vs. what to test directly (shell functions, local git).

4. **Good Test Template**: The standard test case template (lines 89-137) provides a clear pattern for contributors.

5. **Existing Test Quality**: The reviewed test cases (`issue_fix_multiple_comments.sh`, `pr_merged.sh`, `requirement_changes.sh`) follow the prescribed patterns correctly.

---

### Gaps & Concerns

#### HIGH Priority

1. **Missing Error Handling Tests**
   - No tests for network failure scenarios
   - No tests for invalid GitHub API responses
   - No tests for codex execution failures
   - **Recommendation**: Add test cases for error conditions in `tests/cases/error_*.sh`

2. **Missing Concurrent Scenario Tests**
   - No tests for multiple issues with `agent_to_fix` label
   - No tests for simultaneous requirement changes and issue fixes
   - **Recommendation**: Add `multiple_issues_priority.sh` test case

3. **Incomplete Session Loop Exit Verification**
   - `pr_merged.sh` only verifies `/compact` is sent but not the full compact session flow
   - No verification that agent returns to main loop scanning
   - **Recommendation**: Expand `pr_merged.sh` to verify complete exit behavior

#### MEDIUM Priority

4. **No Negative Test for Commit Prefix**
   - Tests verify `Aaron:` prefix is present but no test verifies rejection of commits without prefix
   - **Recommendation**: Add validation test

5. **Missing `gh issue edit` Call Verification**
   - Label transition tests should verify exact `gh issue edit` command arguments
   - TEST_REQUIREMENTS.md line 295-299 shows the pattern but existing tests may not follow it

6. **No Test for Issue Description Mention Parsing**
   - REQUIREMENTS.md specifies issues must "mention current PR in the title"
   - No test verifies this filtering logic
   - **Recommendation**: Add `issue_pr_mention_filtering.sh`

7. **Requirements File Discovery**
   - `scripts/requirements_files.py` exists but no integration test verifies it works with agent.sh
   - **Recommendation**: Add integration test for recursive requirement discovery

#### LOW Priority

8. **Test Documentation Gap**
   - `tests/README.md` exists but TEST_REQUIREMENTS.md doesn't reference it
   - Should ensure documentation consistency

9. **No Performance/Timeout Tests**
   - No tests for POLL_INTERVAL behavior
   - No tests for long-running codex sessions

---

## Acceptance Criteria Validation

### From REQUIREMENTS.md "Unit Test requirements"

| Requirement | Implemented | Notes |
|-------------|-------------|-------|
| Mock `gh` command | Yes | `tests/mocks/gh` |
| Mock `codex` command | Yes | `tests/mocks/codex` |
| One-time containers for tests | Yes | `unit_test.sh` creates disposable containers |
| Data-driven test cases | Yes | Separate files with variables |
| Mock env variables | Yes | Explicit in test cases |
| Print commands during tests | Yes | Via `--verbose` flag |
| Verbose flag (default off) | Yes | `--verbose` CLI option |
| Mock `git` network operations | Yes | `tests/mocks/git` |
| Container flag (default on) | Yes | `--no-container` option |
| Separate test case files | Yes | `tests/cases/*.sh` |
| Blackbox codex verification | Yes | Exact prompt comparison |
| Mocked REQUIREMENTS content | Yes | `requirement_changes.sh` |
| Mocked git diff | Yes | `GIT_MOCK_DIFF` env var |
| `/compact` verification | Yes | `pr_merged.sh` |
| Issue fix with comments | Yes | `issue_fix_multiple_comments.sh` |
| Requirement changes scenario | Yes | `requirement_changes.sh` |
| PR merged scenario | Yes | `pr_merged.sh` |

**Compliance**: 17/17 explicit requirements met (100%)

---

## Recommendations

### Immediate Actions (Before Release)

1. **Add error handling test cases**:
   - `tests/cases/error_gh_api_failure.sh`
   - `tests/cases/error_codex_timeout.sh`
   - `tests/cases/error_git_push_rejected.sh`

2. **Add edge case tests**:
   - `tests/cases/multiple_issues_same_pr.sh`
   - `tests/cases/issue_no_comments.sh`
   - `tests/cases/empty_requirement_diff.sh`

3. **Verify `gh issue edit` calls** in existing label transition tests

### Future Improvements

1. Consider adding integration tests that exercise the full `agent.sh` main loop
2. Add test coverage reporting
3. Consider property-based testing for prompt generation logic
4. Add CI badge for test status in README.md

---

## Decision Required

**Question for Development Team**: Should we require error handling tests before the initial release, or accept them as follow-up work?

- **Option A**: Block release until error handling tests are complete
- **Option B**: Release with current coverage, track error tests as technical debt
- **Option C**: Add basic error tests for critical paths only (gh failure, codex failure)

**PO Recommendation**: Option C - Add critical error tests only to balance coverage with delivery speed.

---

## Sign-off

- [ ] Development team has reviewed feedback
- [ ] Gap items added to backlog
- [ ] Priority agreed upon
- [ ] Test plan updated if needed

---

*Document generated during PO review session*
