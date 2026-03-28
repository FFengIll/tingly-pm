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

## Directory Convention

| Directory | Purpose | Managed by |
|-----------|---------|------------|
| `.eval/` | Per-round artifacts (baseline results, experiment reports, decisions) | Claude Code (each round writes here) |
| `.pm/` | Agent runtime data (tasks, sessions, config) | Agent itself |

`.eval/round-{N}/` contains everything from round N: subagent reports, pass rates, commit or revert decision.

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

### Test Fixtures (Stream JSON)

Tests are defined as **JSONL fixture files** in `.eval/fixtures/`. Each line is one user message in the stream JSON protocol. The agent reads stdin line by line, responds stdout line by line — a file is just a replayable stream.

```
stdin  → {"content": "msg 1"}  \n  →  stdout: {"role": "assistant", "content": "reply 1"}  \n
stdin  → {"content": "msg 2"}  \n  →  stdout: {"role": "assistant", "content": "reply 2"}  \n
```

See `.eval/fixtures/INDEX.md` for the fixture manifest. Single-turn fixtures have 1 line; multi-turn fixtures have 2+ lines and companion `.expect.md` files for per-turn grading.

Execution:
```bash
cat .eval/fixtures/{name}.jsonl | timeout {N} ./tingly-pm -mode run -dir /tmp/test-{id} -config .pm 2>/dev/null
```

The pool is represented by the fixture index. It grows over rounds as subagents MUTATE and DISCOVER new fixtures.

### Random Selection + Mutation (Fixture-Based)

Each subagent has **three selection modes** chosen by probability:

```
For each subagent:
  1. Read .eval/fixtures/INDEX.md for available fixtures
  2. Select mode by probability:
     a. POOL_SAMPLE (60%) — pick random fixtures from INDEX.md
     b. MUTATE (30%) — copy existing .jsonl, modify stream lines, save as new fixture
     c. DISCOVER (10%) — write brand new .jsonl fixture from scratch

Per-subagent budget:
  - Single-turn fixtures: 3-5
  - Multi-turn fixtures: 1-2
  - Mutation/Discovery: 0-2 new fixtures
  - Total turns cap: ~20 per subagent
```

### Mutation Rules (Fixture-Based)

When a subagent picks MUTATE mode, it should:

1. **Copy an existing .jsonl** to a new file in `.eval/fixtures/`
2. **Vary one dimension** of the stream: insert error turn, change language, reorder sequence, add redundant turn
3. **Write `.expect.md`** if the mutated fixture is multi-turn
4. **Update INDEX.md** with the new entry
5. **Label**: "MUTATED FROM: {original_fixture} → {new_description}"

### Discovery Protocol (Fixture-Based)

When a subagent picks DISCOVER mode, it should:

1. **Analyze the agent's tool surface** — what stream combinations haven't been tested?
2. **Write a new `.jsonl** capturing the scenario as a message stream
3. **Write `.expect.md`** with per-turn expectations
4. **Update INDEX.md** with the new entry
5. **Label**: "DISCOVERED: {description} — rationale: {why}"
6. **Rate confidence**: "CONFIDENCE: high/medium/low"

### Main Context: Discovery Review

After all subagents report back, the main context reviews DISCOVERED fixtures:

```
For each DISCOVERED fixture:
  if confidence == "high" AND valid:
    → Keep in .eval/fixtures/, update INDEX.md
    → Run as additional verification
  if confidence == "medium":
    → Keep as candidate in INDEX.md (flagged)
  if confidence == "low" OR invalid:
    → Remove file, record idea for later
```

This creates a **growing, organic fixture set** — each round the agent discovers more about itself.

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
PART A: Run test fixtures and report results
PART B: Optionally mutate/discover new fixtures

=== SETUP ===
1. Read the agent's source: prompt/prompt.go, tools/tools.go, board/*.go
2. Understand the agent's capabilities, boundaries, and known constraints
3. Read .eval/fixtures/INDEX.md for available test fixtures

=== PART A: RUN TESTS ===

YOUR ASSIGNED FIXTURES:
{random_fixtures_from_index}

FOR EACH FIXTURE:
1. Create clean dir: mkdir -p /tmp/test-eval-{round}-{agent_id}
2. Read the .jsonl fixture file
3. Read .expect.md if it exists (per-turn grading criteria)
4. Execute:
   cat .eval/fixtures/{name}.jsonl | timeout {N} ./tingly-pm -mode run -dir /tmp/test-eval-{round}-{agent_id} -config .pm 2>/dev/null | tee /tmp/test-eval-{round}-{agent_id}.output.jsonl
   Timeout: 1 turn=30s, 2-3 turns=60s, 4-5 turns=120s, 6+ turns=180s
5. Check output against expected behavior
6. For multi-turn: grade EACH turn independently, overall PASS = all turns pass
7. Report: PASS or FAIL with observation

=== PART B: MUTATE/DISCOVER NEW FIXTURES (optional) ===

After running your assigned fixtures, think about what's NOT being tested:
- What edge case did you notice while reading the code?
- What stream combination hasn't been exercised?
- What multi-turn scenario is missing from the fixtures?

You may create up to 2 new fixtures. For each:
- Label: "DISCOVERED:" or "MUTATED FROM:"
- Write a .jsonl file in .eval/fixtures/
- Write .expect.md if multi-turn
- Update INDEX.md
- Confidence: high/medium/low
- Rationale: why this matters

=== REPORT FORMAT ===

## Assigned Fixtures
### Fixture: {name}.jsonl (N turns)
STATUS: PASS/FAIL (or PARTIAL X/N turns)
EXPECTED: {from .expect.md or INDEX.md description}
GOT: {actual output summary per turn}
NOTE: {any observations}

## Proposed New Fixtures
### DISCOVERED/MUTATED: {description}
CONFIDENCE: high/medium/low
RATIONALE: {why}
FILE: {path to new .jsonl}
EXPECTED: {per-turn behavior}

## SUMMARY
Assigned: X/Y fixtures passed
Proposed: N new fixtures (H high, M medium, L low confidence)
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

## Execution Modes (Stream JSON)

All tests use the **stream JSON protocol** via JSONL fixture files:

| Mode | Command | What it Tests |
|------|---------|---------------|
| Fixture (single-turn) | `cat fixture.jsonl \| agent -mode run` | Raw tool correctness |
| Fixture (multi-turn) | `cat fixture.jsonl \| agent -mode run` | Contextual reasoning, cross-turn state |

**Stream JSON protocol**: both stdin and stdout are line-delimited JSON (JSONL/NDJSON). One `{"content": "..."}` per line in, one `{"role": "assistant", "content": "..."}` per line out. Session state persists across the stream.

**Key distinction**: Single-turn fixtures isolate tool behavior. Multi-turn fixtures test cross-turn reasoning (reference resolution, multi-step workflows, error recovery).

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
### Don't: Trust Single-Turn Fixtures for Contextual Features
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
