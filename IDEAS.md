# Ideas & Brainstorm Summary

## TODOs

- [ ] Define contract formats for all agent interactions (see O1 for full list: file contracts, issue templates, label conventions, commit message conventions)

## Project Vision

An AI-powered virtual software team that uses GitHub as the control plane. Developers interact with AI agents through familiar GitHub primitives (branches, PRs, issues, commits) rather than a custom UI. Targets solo developers and tiny teams (2-3 members) running on home server / SOHO infrastructure.

---

## Conclusions

### 1. Polling over Event-driven (Decided)

The agent uses a **pull-based polling model**, not webhooks.

**Rationale:** Home server / SOHO environment means no static IP, behind NAT, no reliable public endpoint. Webhooks would require tunneling (ngrok, Cloudflare Tunnel, etc.) which adds dependencies and fragility — contradicting the "minimal dependencies" principle. Polling is self-healing and infrastructure-independent.

**Optimizations to consider:**
- Tiered polling intervals (frequent during active hours, slow at night)
- Lightweight checks (`git ls-remote`, GitHub API ETags) to minimize overhead
- Status visibility (heartbeat / last-checked indicator) so user knows the agent is alive

### 2. Agent Role Model (Decided)

Three distinct roles, modeled after a real software team:

| Role | Count | Responsibility |
|------|-------|----------------|
| **PM (Product Manager)** | 1 | Define features and requirements |
| **SE (Software Engineer)** | n | Implementation (feature code + test code) |
| **SRE (Site Reliability)** | 1 | Project setup, CI/CD, dev conventions |
| **QA (Test Engineer)** | 1 or n | Test strategy, test requirements, review test results, raise issues |

**Key distinction:** QA is a **test strategist**, not a test coder. QA defines *what* to test (scenarios, edge cases, acceptance criteria) via `TEST_REQUIREMENTS.md`. SE implements both the feature code and the test code. This means:
- Only SE writes code — no merge conflicts between agents
- QA focuses on the higher-value thinking: what should be tested and why
- Tests naturally align with implementation since the same agent writes both

**Known risk: "grading your own homework."** Since SE writes both the implementation and the tests, there's a risk that:
1. SE misinterprets QA's test requirements — tests don't cover what QA intended
2. SE unconsciously writes tests biased toward their implementation — tests pass but verify the wrong thing

**Mitigation: QA reviews SE's test code.** After SE implements test cases, QA reads the test code and compares it against original test requirements. If the tests don't match intent, QA files issues. This adds a review step to the cycle:
```
QA writes TEST_REQUIREMENTS.md
    → SE implements tests
        → QA reviews test code against original intent
            → QA files issues if tests don't match intent
```
QA doesn't need to *write* code, but must be able to *read and critique* it. This is consistent with the strategist role — a test strategist reviews whether their strategy was executed correctly.

### 3. PM and SRE Agents are Out of Scope (Decided)

Both PM and SRE roles are handled **outside this project** via conversational chat UI (e.g., Claude web, ChatGPT). These roles involve highly interactive, real-time discussions not suited for GitHub's async workflow.

**PM** brainstorms with the user and outputs `REQUIREMENT.md` — defines *what* to build.
**SRE** works with the user to set up the project and outputs `PROJECT_SETUP.md` — defines *how* to build and test. This includes:
- Project structure conventions (where source goes, where tests go)
- Test framework and how to run tests
- Naming conventions CI expects (e.g., `test_*.py`, `*.test.ts`)
- Quality gates (coverage thresholds, lint rules)
- Build and deploy commands
- Environment requirements

SRE is a **bootstrap role** — active during project setup, then idle. Reactivated on demand if CI/CD or project conventions need changes.

**Handoff contracts:**
```
[Out of scope]                    [Handoff contracts]           [This project]
User + PM   → brainstorm  →  REQUIREMENT.md          ──┐
                              (what to build)           ├──→  SE + QA workflow
User + SRE  → setup       →  PROJECT_SETUP.md        ──┘
                              (how to build & test)
```

SE references both: *what* from PM, *how* from SRE. QA also references `PROJECT_SETUP.md` to understand quality gates when writing test requirements.

### 4. User as Arbiter (Decided)

Agents do **not** resolve conflicts with each other. The user is the final judge. Agents are workers; the user is the manager.

- SE agent: implements requirements, fixes issues labeled `Agent to fix`
- QA agent: tests code, reports issues, but has no authority to block or revert
- All conflict resolution flows through the user via GitHub (issue comments, labels, close/reopen)
- No inter-agent communication protocol needed — agents interact with GitHub, user is the router

**Trade-off acknowledged:** The user becomes a bottleneck. If QA files issues and the user is unavailable, SE may have nothing to do. A fallback rule may be needed (e.g., SE continues with next requirements when no issues are pending).

### 5. SE and QA Work in Parallel (Decided)

SE and QA work in parallel with no sequencing gate between them.

- QA reads `REQUIREMENT.md` and outputs `TEST_REQUIREMENTS.md` (test scenarios, edge cases, acceptance criteria)
- SE starts implementing from `REQUIREMENT.md` immediately — does not wait for QA
- When `TEST_REQUIREMENTS.md` lands (or is updated), SE picks it up as just another requirement change and implements the test cases

**SE treats all requirement sources equally.** Whether a change comes from PM (`REQUIREMENT.md`) or QA (`TEST_REQUIREMENTS.md`), the SE agent's behavior is the same: detect the diff, implement it, commit.

**SE agent core loop:**
```
while session is active:
    pull latest changes
    diff = compare current state vs last processed state
    if diff in REQUIREMENT.md or TEST_REQUIREMENTS.md:
        implement the changes
        commit
    if issues labeled "Agent to fix":
        fix issue
        commit
    sleep(poll_interval)
```

**Rationale:**
- No idle time — SE starts immediately, no waiting for QA
- Architecturally simple — no special sequencing or "wait for QA" state
- Rework is cheap for AI agents — if QA's test specs require rethinking, it's handled in the normal cycle
- Matches real-world dynamics where requirements arrive incrementally

---

## Open Points for Next Session

### O1. Contract Formats (Detail Design)

All agent interaction flows through contracts — markdown files and GitHub issues. The format and structure of each contract needs to be defined in detail design.

**File-based contracts:**

| Contract | Producer | Consumer(s) | Questions to resolve |
|----------|----------|-------------|---------------------|
| `REQUIREMENT.md` | PM (out of scope) | SE, QA | Freeform vs. structured template? Incremental updates — how does SE know what's new vs. already implemented? |
| `TEST_REQUIREMENTS.md` | QA | SE | How granular — one scenario per line, or grouped by feature? How to link test requirements back to REQUIREMENT.md items? |
| `PROJECT_SETUP.md` | SRE (out of scope) | SE, QA | What sections are mandatory? How prescriptive vs. flexible? |

**Issue-based contracts:**

| Contract | Producer | Consumer(s) | Questions to resolve |
|----------|----------|-------------|---------------------|
| Bug / defect issue | QA or User | SE | Issue template: what fields are required (repro steps, expected vs. actual, severity)? |
| Test mismatch issue | QA | SE | How to reference the specific test requirement that was misimplemented? |
| Implementation issue | SE | User | When SE is blocked or needs clarification, what's the format? |

**Label conventions:**

| Label | Meaning | Set by | Questions to resolve |
|-------|---------|--------|---------------------|
| `Agent to fix` | Issue ready for SE to pick up | User or QA | Priority levels needed? |
| `Agent fixing` | SE is working on it | SE | Timeout if SE stalls? |
| `Agent fixed to be verified` | SE done, awaiting verification | SE | Who verifies — QA, user, or both? |

**Commit message conventions:**
- Should commits reference issue numbers?
- Should commits indicate which requirement item they address?
- Format for SE commits vs. QA commits?

All of the above to be defined during detail design phase.

### ~~O2. Validation Phase Details~~ (Resolved)

Validation has two tracks:

**Track 1: Test execution (SE responsibility, CI as safety net)**
- SE runs tests locally before committing. Fix until pass, then commit clean code.
- CI runs on every commit as a double-check (environment differences, integration issues).
- **Escape hatch:** If SE fails to pass tests after N retries, SE commits what it has and files an issue describing the failure. User triages. This prevents silent infinite loops where SE is stuck and user has no visibility.

**Alternative approaches considered (may revisit):**
- *(Option C)* Always commit regardless, let CI catch failures, failures become issues. Simpler but noisy git history.
- *(Option D)* SE commits failing code to a sub-branch (e.g., `agent/se-1/feature-x/wip`). User inspects without polluting main feature branch. Cleaner but adds branching complexity.

**Track 2: Test code verification (QA responsibility)**
- QA watches for **any** SE commit — not just test code changes.
- On every change, QA reviews test code alignment against `TEST_REQUIREMENTS.md`:
  - Do existing tests still match QA's intent?
  - Are there new feature changes lacking corresponding tests?
- QA files issues for misalignment.

**Updated agent loops:**
```
SE loop:                            QA loop:
  pull changes                        pull changes
  diff requirements?                  diff from SE commits?
    → implement code + tests            → review test code alignment
    → run tests locally                 → review CI results
    → fix until pass (max N retries)    → file issues if needed
    → commit (or file issue if stuck)   sleep
  issues labeled "Agent to fix"?
    → fix, run tests, commit
  sleep
```

### ~~O4. User Bottleneck Mitigation~~ (Resolved)

No special "user away" logic needed. SE follows a fixed priority order:

1. **Issues labeled `Agent to fix`** (highest priority)
2. **New requirement diffs** (`REQUIREMENT.md` or `TEST_REQUIREMENTS.md`)
3. **Idle** — poll and wait (lowest)

SE works down the list. The user bottleneck only exists when all requirements are implemented and no issues are assigned — which is the correct stopping point. No auto-escalation or timeout needed.

### ~~O5. Agent Identity and Branch Convention~~ (Resolved)

**Team concept:** Agents are organized into teams, not individual identities. Each team has exactly 1 SE + 1 QA. The team is the unit of work.

**Branch convention:** `agent/<team_name>/<feature>`
- Example: `agent/alpha/feature-auth`
- Both SE and QA in team "alpha" watch for `agent/alpha/*` branches
- No role encoded in the branch name — the branch represents the session/feature, not the agent

**One branch at a time per team.** A team works on a single active branch. When the branch merges, the team scans for the next one.

**User creates the branch** with `REQUIREMENT.md` committed. Agents detect it and start working.

**Scaling:** Add more teams for parallel features. Each team is fully independent.
```
Team alpha → agent/alpha/feature-auth    → SE-alpha + QA-alpha
Team beta  → agent/beta/feature-payments → SE-beta  + QA-beta
```
No cross-team coordination. User is the only one who sees across teams.

### ~~O6. CI/CD Ownership~~ (Resolved)

CI/CD is owned by the **SRE role** (out of scope). SRE sets up CI/CD as a bootstrap step before feature work begins. CI/CD is treated as shared infrastructure, not an ongoing agent responsibility. Changes to CI/CD go back through the user + SRE chat, same as PM requirement changes.
