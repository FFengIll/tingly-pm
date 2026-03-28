# Agent Iterative Improvement Methodology

**Type:** Reusable framework
**Scope:** Any ReAct-based agent (Go, Claude Code as driver)
**Extracted from:** tingly-pm improvement loop (4 rounds → v2, 2026-03-28)

---

## Core Idea

Treat an agent (prompt + tools + config) as a trainable subject. An external driver (Claude Code) iteratively modifies the agent, executes it, observes output, evaluates behavior, and commits or reverts — analogous to model training at the agent level.

```
Modify → Build → Execute → Observe → Evaluate → Commit or Revert → Repeat
```

## v2 Key Changes

| Change | What | Why |
|--------|------|-----|
| Parallel fuzzing evaluation | Each round launches a batch of subagents, each testing **random feature subsets** | Fuzzing catches regressions in untested areas; sequential always tests the same path |
| Two-part loop | Part 1 = experiment + improve; Part 2 = verify improvements | Isolates "did it change" from "did it improve" — prevents confirmation bias |
| Verify-before-commit | Part 2 uses independent random tests; fail = revert all | Guarantees no regression leaks into main |

## Architecture

```
┌──────────────────────────────────────────────────────────────────────┐
│                       Claude Code (the driver)                        │
│                                                                       │
│  Per Round (two-part loop):                                           │
│                                                                       │
│  ┌─────────────────────────────────────────────────────────────┐     │
│  │ PART 1: Experiment & Improve                               │     │
│  │                                                              │     │
│  │ 1. Baseline: parallel fuzzing eval (N subagents, random)    │     │
│  │    → record baseline pass rate across all features          │     │
│  │ 2. Hypothesize: M improvements based on failures             │     │
│  │ 3. Launch M subagents in parallel                           │     │
│  │    Each: edit → build → test → execute → record             │     │
│  │ 4. Collect results, apply winners                           │     │
│  └─────────────────────────────────────────────────────────────┘     │
│                          │                                            │
│                          ▼                                            │
│  ┌─────────────────────────────────────────────────────────────┐     │
│  │ PART 2: Verify (independent evaluation)                     │     │
│  │                                                              │     │
│  │ 5. Launch N subagents in parallel (NEW random selections)   │     │
│  │    → test the SAME agent but with DIFFERENT feature subsets │     │
│  │ 6. Compare Part 2 results vs Part 1 baseline                │     │
│  │                                                              │     │
│  │ ┌─────────────────────────────────────────────┐             │     │
│  │ │ Part 2 pass rate ≥ baseline?               │             │     │
│  │ │   YES → commit all changes                 │             │     │
│  │ │   NO  → revert ALL changes, keep baseline  │             │     │
│  │ └─────────────────────────────────────────────┘             │     │
│  └─────────────────────────────────────────────────────────────┘     │
│                          │                                            │
│                          ▼                                            │
│  ┌─────────────────────────────────────────────────────────────┐     │
│  │ 7. Record learnings → feed into next round                   │     │
│  └─────────────────────────────────────────────────────────────┘     │
│           ▲                                                          │
│           └──────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Baseline (Parallel Fuzzing Evaluation)

Before any changes, establish ground truth via parallel subagents with **randomized feature selection**.

### Why Randomization?

Sequential testing always follows the same path. The developer picks tests that confirm their hypothesis. Fuzzing-style randomization:
- Discovers regressions in unexpected feature combinations
- Prevents overfitting to a fixed test order
- Simulates real-world usage where users don't follow a script

### Test Feature Pool

Define a pool of **known** independently testable features. This is a starting point, NOT an exhaustive list — subagents are encouraged to discover beyond it.

```
POOL = [
  "create_task: chinese input",
  "create_task: english input",
  "create_task: duplicate detection",
  "create_task: priority from keywords",
  "create_task: assign on create",
  "update_task: single field only",
  "update_task: status change",
  "update_task: nonexistent task",
  "list_tasks: empty board",
  "list_tasks: priority grouping",
  "list_tasks: age display",
  "list_tasks: blocker filter",
  "search_tasks: by title",
  "search_tasks: archived tasks",
  "dependency: add and list",
  "dependency: cycle detection",
  "archive: done vs dropped",
  "member: register and list",
  "error: empty input",
  "error: invalid task id",
  "context: resolve by name",
  "context: multi-step workflow",
  "language: input language matching",
  "output: conciseness",
  "report: daily generation",
  "timeline: event ordering",
]
```

The pool grows over rounds. New features discovered by subagents (see **Mutation**) are added after review.

### Random Selection + Mutation

Each subagent doesn't just blindly pick from the pool. It has **three selection modes** chosen by probability:

```
For each subagent:
  1. Read the agent's prompt, tools, and codebase (understand the subject)
  2. Select mode by probability:
     a. POOL_SAMPLE (60%) — pick a random subset from the pool
     b. MUTATE (30%) — based on understanding, generate a VARIANT of a pool item
        e.g., pool has "create_task: chinese input"
             → mutate to "create_task: mixed chinese+english title"
             → mutate to "create_task: extremely long title (>200 chars)"
             → mutate to "create_task: special characters in title"
     c. DISCOVER (10%) — propose an entirely NEW feature/edge case not in the pool
        e.g., "what happens if user assigns a task to a non-existent member?"
             → "create + immediately archive in same session"
             → "update priority multiple times in sequence"
```

### Mutation Rules

When a subagent picks MUTATE mode, it should:

1. **Read the agent's source** (prompt, tools, relevant board code) to understand boundaries
2. **Vary one dimension** of an existing test: input format, sequence order, data volume, language, state
3. **Construct a valid test case** — must include: input, execution command, expected behavior
4. **Label it clearly** — "MUTATED FROM: {pool_item} → {new_test_description}"

### Discovery Protocol

When a subagent picks DISCOVER mode, it should:

1. **Analyze the agent's tool surface** — what combinations haven't been tested?
2. **Identify gaps** — what user scenario is not covered by any pool item?
3. **Propose a test case** — must include: input, execution command, expected behavior
4. **Label it clearly** — "DISCOVERED: {new_feature} — rationale: {why this matters}"
5. **Rate confidence** — "CONFIDENCE: high/medium/low" (is this a real gap or speculative?)

### Main Context: Discovery Review

After all subagents report back, the main context reviews DISCOVERED items:

```
For each DISCOVERED test:
  if confidence == "high" AND main context agrees it's valid:
    → Add to POOL immediately (available for next round)
    → Run it now as an additional verification point
  if confidence == "medium":
    → Add to POOL as "candidate" (flagged for review)
    → Optionally run it to gather data
  if confidence == "low" OR main context disagrees:
    → Discard, but record the idea (may be useful later)
    → Provide brief feedback to inform future discovery
```

This creates a **growing, organic test pool** — each round the agent discovers more about itself.

### Execution

Launch N subagents in parallel. Each:
1. Gets its assigned test subset
2. Runs each test in a clean environment (`/tmp/test-agent-rN`)
3. Records pass/fail/observation for each test
4. Returns a structured report

### Baseline Pass Rate

Aggregate all subagent results:

```
Baseline: X/Y passed (Z%)
Breakdown by category:
  - Core operations: a/b
  - Edge cases: c/d
  - Contextual: e/f
  - Output quality: g/h
```

This is the number to beat in Part 2.

---

## Phase 2: Hypothesize

From baseline failures, form improvement hypotheses. Same as v1.

### Problem Severity Scale

| Severity | Meaning | Action |
|----------|---------|--------|
| Critical | Agent produces wrong data or crashes | Fix immediately |
| High | Agent behaves incorrectly but doesn't crash | Fix in this round |
| Medium | Agent could be better but works | Fix if easy |
| Low | Nice to have | Queue for future |

### Impact Ranking

1. **System prompt** — highest impact, zero code risk
2. **Tool behavior** — medium impact, fix bugs, improve output
3. **Tool set** — structural impact, add/remove/merge
4. **Agent configuration** — low impact

### Hypothesis Format

```
Exp [ID]: [Title]
Hypothesis: [What change will fix what problem]
Target: [prompt.go / tools.go / main.go]
Expected: [What the output should look like after]
```

---

## Phase 3: Experiment (Parallel)

Launch M subagents in parallel, one per hypothesis. Same as v1.

### Subagent Protocol

Each subagent receives:

```
1. What file(s) to edit
2. What change to make
3. How to verify: go build + go test
4. How to execute: test input via stdio
5. Report format: structured pass/fail/observation
```

### Subagent Isolation

- Each subagent works in the same directory (shared filesystem)
- Changes are NOT committed by subagents
- Main context collects results and decides
- For conflicting file changes, serialize or merge manually

### Subagent Report Format

```
EXPERIMENT: [ID] - [Title]
BUILD: pass/fail
TESTS: pass/fail
RESULT: pass/fail
OUTPUT: <actual JSON output from test>
OBSERVATIONS: <what you noticed>
```

### What to Watch For

- **False positives**: Test passes but agent used wrong tool or fabricated data
- **Invisible regressions**: New behavior breaks something not in the test suite
- **Over-fitting**: Prompt tuned for specific test inputs but fails on variations

---

## Phase 4: Evaluate & Apply Winners

Collect all reports. Compare. Apply winners. **Do NOT commit yet.**

### Decision Matrix

| Build | Tests | Behavior | Decision |
|-------|-------|----------|----------|
| fail | — | — | Reject immediately |
| pass | fail | — | Reject (broken existing tests) |
| pass | pass | wrong | Reject (behavior regression) |
| pass | pass | correct | Accept (stage for Part 2) |

### Integration Check

After applying all winning changes (but before commit):
1. `go build .` — must pass
2. `go test ./...` — must pass
3. Note: Full integration test deferred to Part 2

---

## Phase 5: Verify (Part 2 — Independent Evaluation)

This is the critical new phase. It uses **fresh random selections** to validate that improvements are real, not overfit to Part 1's test set.

### Why Separate Verification?

Part 1 experiments target specific failures. The agent may improve on those but regress elsewhere. A separate verification round with **different** random subsets catches this.

### Verification Protocol

1. **New random shuffle** — do NOT reuse Part 1's assignments
2. **Same pool size** — same number of subagents, same test budget
3. **Must include baseline failures** — ensure Part 1's fixes are re-tested
4. **Must include random extras** — ensure no regressions in unrelated areas

```
For Part 2 evaluation:
  1. Shuffle POOL again (different random seed)
  2. Assign each subagent a new slice
  3. REQUIRED: include all tests that Part 1 experiments targeted
  4. OPTIONAL: add 2-3 new test variations not in Part 1
```

### Go / No-Go Decision

```
Part 2 Results:
  Pass rate: X/Y (Z%)
  Baseline was: A/B (C%)

  if Z >= C:
    → COMMIT: improvements are real, no regressions
    → Record learnings, advance to next round

  if Z < C:
    → REVERT ALL: git checkout -- . (or git stash + drop)
    → Analyze what regressed
    → Feed failure analysis into next round's hypotheses
```

### Regression Analysis (on failure)

If Part 2 fails:

```
1. Compare Part 1 winners vs Part 2 failures
   - Which experiment caused the regression?
   - Was it a prompt change or code change?

2. Categorize:
   - "Overfit" — agent passes targeted tests but fails variations
   - "Conflict" — two winning experiments conflict when combined
   - "Fragile" — change works in isolation but breaks under integration

3. Record:
   - What regressed
   - Why it regressed
   - What to try differently next round
```

---

## Complete Loop Summary

```
PART 1: Experiment
├── 1a. Parallel fuzzing baseline (N subagents: sample + mutate + discover)
├── 1b. Review discoveries → grow pool
├── 1c. Analyze failures → M hypotheses
├── 1d. Parallel experiments (M subagents, one per hypothesis)
├── 1e. Evaluate experiments → apply winners
└── (NO commit yet)

PART 2: Verify
├── 2a. Parallel fuzzing evaluation (N subagents, NEW random+mutated subsets)
├── 2b. Review new discoveries → grow pool
├── 2c. Compare vs baseline
├── 2d. PASS → commit | FAIL → revert all
└── Record learnings

→ Repeat (pool grows each round)
```

---

## Subagent Prompt Template

For each evaluation subagent:

```
You are evaluating the tingly-pm agent. Your job has two parts:
PART A: Run tests and report results
PART B: Optionally propose new tests (mutation/discovery)

=== SETUP ===
1. Read the agent's source: prompt/prompt.go, tools/tools.go, board/*.go
2. Understand the agent's capabilities, boundaries, and known constraints

=== PART A: RUN TESTS ===

YOUR ASSIGNED TESTS:
{random_tests_or_mutations}

FOR EACH TEST:
1. Create clean dir: mkdir -p /tmp/test-eval-{round}-{agent_id}
2. Execute: echo '...' | ./tingly-pm -mode run -dir /tmp/test-eval-{round}-{agent_id} -config .pm 2>/dev/null
3. Check output against expected behavior
4. Report: PASS or FAIL with observation

=== PART B: PROPOSE NEW TESTS (optional) ===

After running your assigned tests, think about what's NOT being tested:
- What edge case did you notice while reading the code?
- What combination of features hasn't been exercised?
- What user scenario is missing from the test pool?

You may propose up to 2 new test cases. For each:
- Label: "DISCOVERED:" or "MUTATED FROM:"
- Input, execution command, expected behavior
- Confidence: high/medium/low
- Rationale: why this matters

=== REPORT FORMAT ===

## Assigned Tests
### Test: {name}
STATUS: PASS/FAIL
EXPECTED: {what should happen}
GOT: {actual output summary}
NOTE: {any observations}

## Proposed New Tests
### DISCOVERED/MUTATED: {description}
CONFIDENCE: high/medium/low
RATIONALE: {why}
INPUT: {test command}
EXPECTED: {what should happen}

## SUMMARY
Assigned: X/Y passed
Proposed: N new tests (H high, M medium, L low confidence)
```

---

## Fuzzing Strategies

### Strategy 1: Uniform Random + Mutation

Default strategy. Each subagent:
- 60% picks random items from the pool
- 30% generates mutated variants of pool items
- 10% proposes entirely new features

Simple, organic, and the pool grows over time.

### Strategy 2: Failure-Weighted

After baseline, weight randomization toward categories with lower pass rates. Ensures problem areas get more scrutiny.

```
weights = {
  "core":     baseline.core_pass / baseline.core_total,
  "edge":     baseline.edge_pass / baseline.edge_total,
  "context":  baseline.ctx_pass / baseline.ctx_total,
  "output":   baseline.out_pass / baseline.out_total,
}
// Invert weights — lower pass rate = higher selection probability
// Categories that failed more get tested more in Part 2
```

Mutation probability also shifts: failed categories get higher mutation rate (40% mutate, 20% discover, 40% sample).

### Strategy 3: Adversarial

For Part 2, deliberately construct tests that combine features that changed with features that didn't. Catches cross-feature regressions.

```
Part 2 test generation:
  changed_features = features modified by Part 1 experiments
  unchanged_features = POOL - changed_features
  tests = cross_product(changed_features, unchanged_features) |> sample
```

Discovery mode is forced here: each subagent must propose at least one adversarial test combining a changed + unchanged feature.

**Recommendation**: Start with Strategy 1 (uniform random). Switch to Strategy 2 or 3 if regression rate is high.

---

## Execution Modes

| Mode | Command | What it Tests |
|------|---------|---------------|
| Single request | `echo '...' \| agent -mode run` | Raw tool correctness (stateless) |
| Multi-message | `printf '...\n...\n...' \| agent -mode run` | Contextual reasoning (stateful session) |

**Key distinction**: Single-request mode isolates tool behavior. Multi-message mode tests whether the agent can reason across turns (reference resolution, multi-step workflows).

### Isolation

Every test needs a clean state directory:

```bash
mkdir -p /tmp/test-agent-rN-{subagent_id}
```

---

## Prompt Engineering Patterns

From 4 rounds of prompt improvement:

### Pattern 1: Structured Sections > Flat List

Organize prompt into titled sections with `##` headers. The LLM uses section headers as attention anchors.

### Pattern 2: CRITICAL Prefix for Hard Rules

When the model consistently violates a rule, prefix with `CRITICAL:` and give explicit examples.

### Pattern 3: Explicit Language Mappings

If the model operates in multiple languages, provide explicit keyword mappings.

### Pattern 4: Behavioral Instructions Over Constraints

Instead of "don't do X", prefer "do Y instead".

### Pattern 5: Context-Dependent Behavior

Acknowledge limitations explicitly. Document what requires session context.

---

## Tool Design Patterns

From 4 rounds of tool improvement:

### Pattern 1: Fewer Tools > More Convenience Wrappers
### Pattern 2: Merge Query + Filter
### Pattern 3: Read Before Write
### Pattern 4: Output Formatting Lives in the Tool
### Pattern 5: Human-Readable Derivatives

(See v1 for full descriptions — unchanged)

---

## Anti-Patterns

### Don't: Tune Prompt for Single Test Inputs
### Don't: Change Multiple Things at Once (per experiment)
### Don't: Trust Single-Request Results for Contextual Features
### Don't: Keep Redundant Tools "Just in Case"
### Don't: Skip the Integration Test
### Don't: Reuse Part 1 Test Assignments in Part 2

**New**: If Part 2 uses the same test subset as Part 1, you're testing overfitting, not improvement. Always reshuffle.

### Don't: Treat Discovered Tests as Immediately Valid

Subagent-discovered features are proposals, not facts. The main context must review and decide. Blindly adding every discovery pollutes the pool with noise.

### Don't: Let Discovery Override Known Pool Items

Discovered tests supplement the pool, not replace it. Always ensure core pool items get tested — discovery is the bonus, not the baseline.

---

## Evaluation Criteria

Universal criteria for any agent improvement:

| Criterion | Check |
|-----------|-------|
| Tool correctness | Right tool, valid args, no unnecessary calls |
| No hallucination | No fabricated field values, IDs, or names |
| Output quality | Concise, accurate, scannable |
| Language consistency | Matches user's language |
| Error handling | Graceful degradation with helpful message |
| Contextual reasoning | Resolves references across turns |
| State consistency | Data written matches data readable |
| Regression-free | Existing behavior still works |

---

## Safety Protocol

1. **Build before run** — compilation must succeed
2. **Test before commit** — existing tests must pass
3. **Isolated environment** — never use real user data for testing
4. **Branch per round** — merge to main only when confident
5. **Verify before commit** — Part 2 must pass before any commit
6. **Revert immediately** — if Part 2 fails, revert ALL, don't patch forward
7. **Commit with intent** — message explains what behavior improved
8. **Record everything** — learnings feed into next round's hypotheses
