# Ideas & Brainstorm Summary

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
| **QA (Test Engineer)** | 1 or n | Test strategy, test requirements, CI/CD, review test results, raise issues |

**Key distinction:** QA is a **test strategist**, not a test coder. QA defines *what* to test (scenarios, edge cases, acceptance criteria) via `TEST_REQUIREMENTS.md`. SE implements both the feature code and the test code. This means:
- Only SE writes code — no merge conflicts between agents
- QA focuses on the higher-value thinking: what should be tested and why
- Tests naturally align with implementation since the same agent writes both

### 3. PM Agent is Out of Scope (Decided)

The PM role is handled **outside this project** via a conversational chat UI (e.g., Claude web, ChatGPT). Brainstorming and requirements gathering are highly interactive and real-time — GitHub's async, commit-based workflow is not suited for this phase.

**Handoff contract:** The PM phase outputs a structured file (e.g., `REQUIREMENT.md`) committed to the branch. This project picks up from that point and handles `Requirements -> SE -> QA -> loop`.

```
[Chat UI + User]  -->  REQUIREMENT.md  -->  [This Project]  -->  Code, Tests, PRs
                       (the contract)
```

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

### O1. REQUIREMENT.md Format

What does the handoff document look like?
- Freeform markdown vs. structured template (Goals, User Stories, Acceptance Criteria)?
- Incremental updates — does the user rewrite the whole file or append? How does SE know what's new vs. already implemented?

### O2. Validation Phase Details

After SE implements and QA reviews test results:
- How does QA trigger test execution — via CI, or does QA run tests directly?
- What signals "implementation complete" so QA knows to start reviewing results?
- How does QA report test failures — issue per failure, or a summary issue?

### O4. User Bottleneck Mitigation

What should agents do when the user is not available to triage?
- SE fallback behavior when no issues are assigned
- Priority rules: issues vs. new requirements
- Should there be an auto-escalation or timeout mechanism?

### O5. Agent Identity and Branch Convention

The naming convention `agent/<agent_name>/*` needs refinement for multi-role agents:
- Does the branch name encode the role (e.g., `agent/se-1/feature-auth`, `agent/qa/feature-auth`)?
- How are multiple SE agents assigned to different tasks?
- Who creates the branches — user or PM output?

### O6. CI/CD Ownership

The README mentions "Agent or GitHub Action triggers CI/CD." With a dedicated QA role:
- Is CI/CD setup and maintenance the QA agent's responsibility?
- Does QA own the pipeline definition, or just consume its results?
