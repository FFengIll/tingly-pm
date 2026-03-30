# Eval Loop Specification

**Version:** 3.0
**Agent:** tingly-pm (AI project manager)
**Worker:** Claude Code (invoked per-round by the outer loop)
**Date:** 2026-03-30

---

## Overview

The eval loop is an **iterative, per-experiment improvement system** for an agent (prompt + tools + config). Each experiment is independently verified before the next one begins — good changes accumulate, bad changes are discarded immediately.

The outer loop provides a **seed** (exploration direction). The inner worker (you) uses that seed to guide fixture selection and hypothesis formation, then runs experiments one at a time with independent evaluation after each.

```
Seed → Baseline → Hypothesize → Experiment → Evaluate → Keep or Discard → Repeat
```

### Directory Convention

| Directory | Purpose | Managed by |
|-----------|---------|------------|
| `.eval/` | Per-round artifacts (baseline results, experiment reports, decisions) | Claude Code (each round writes here) |
| `.pm/` | Agent runtime data (tasks, sessions, config) | Agent itself |
| `.worktree/` | Experimental branches (gitignored, ephemeral) | `git worktree add` |

### Role

You are the **inner worker** of the eval loop. The outer loop (how many rounds, when to start, seeds, parallelism, etc.) is controlled externally via `eval-loop.sh` — you don't need to worry about it. Your job is to execute **one complete round** of the methodology below, using the round number, total, and seed passed to you in the prompt.

---

## The Loop

Each round is a **baseline + iterative experiments** cycle. The seed guides what to focus on; each experiment is independently evaluated before proceeding.

```
1. BASELINE
   ├─ Run smoke tests + seed-guided fixtures
   ├─ Record baseline pass rate
   └─ Identify failures → form hypotheses (prioritized by severity)

2. ITERATE (one experiment at a time)
   ├─ Pick top hypothesis
   ├─ Make the change
   ├─ EVALUATE (independent: new random+mutated fixtures, NOT the same ones from baseline)
   │   ├─ PASS (no regressions, targeted fix works) → KEEP the change
   │   └─ FAIL (regression or no improvement) → DISCARD the change (revert)
   ├─ Re-run smoke tests (regression gate)
   ├─ Record result
   └─ Repeat from "Pick top hypothesis" until no more viable hypotheses

3. COMMIT
   └─ All kept changes are already accumulated — commit once
```

Key principles:
- **One experiment at a time** — no parallel experiments. Focus and evaluate.
- **Independent evaluation** — each experiment is tested against a fresh fixture set, never the same ones used to form the hypothesis.
- **Keep or discard immediately** — don't batch. If it's bad, revert before moving on.
- **Smoke tests after every experiment** — cheap regression gate.
- **Seed-guided** — the exploration seed from the outer loop shapes which fixtures you sample and what hypotheses you form.

---

## Test Fixtures (Stream JSON)

Tests are defined as **JSONL fixture files** in `.eval/fixtures/`. Each line is one user message in the stream JSON protocol. The agent reads from stdin line by line, responds to stdout line by line — a file is just a replayable stream.

See `.eval/fixtures/INDEX.md` for the full fixture manifest.

### Stream JSON Protocol

```
stdin  → {"content": "user message 1"}  \n  →  stdout: {"role": "assistant", "content": "reply 1"}  \n
stdin  → {"content": "user message 2"}  \n  →  stdout: {"role": "assistant", "content": "reply 2"}  \n
...
```

Both sides are line-delimited JSON (JSONL/NDJSON). Session state persists across the stream.

### Execution

```bash
# Run any fixture (single or multi-turn)
cat .eval/fixtures/{name}.jsonl | timeout {N} ./tingly-pm -mode run -dir /tmp/test-{id} -config .pm 2>/dev/null

# With output capture
cat .eval/fixtures/{name}.jsonl | timeout {N} ./tingly-pm -mode run -dir /tmp/test-{id} -config .pm 2>/dev/null | tee /tmp/test-{id}.output.jsonl
```

Timeout by turn count: 1 turn=15s, 2-3 turns=30s, 4-5 turns=60s, 6+ turns=90s.

### Fixture Types

**Single-turn** (1 message per file):
- `create-task-chinese.jsonl`, `create-task-english.jsonl`, `error-empty-input.jsonl`, etc.

**Multi-turn** (2+ messages per file, with `.expect.md` for per-turn grading):
- `context-resolve-by-name.jsonl` (3 turns), `workflow-create-dep-archive.jsonl` (5 turns), etc.

### Initial Fixture Set

56 fixtures total: 23 single-turn + 33 multi-turn. See `INDEX.md` for the full list. This is a starting set — subagents MUTATE and DISCOVER new fixtures each round.

### Expectation Files

All multi-turn fixtures have companion `.expect.md` files with per-turn grading criteria. Subagents read these to grade each turn independently.

### Automated Assertions

`eval-assert.sh` provides programmatic, deterministic PASS/FAIL by parsing tool call patterns in agent output:

```bash
./eval-assert.sh                    # all fixtures
./eval-assert.sh smoke              # smoke tests only
./eval-assert.sh -v create-task-english  # verbose
```

This serves as a reproducible regression gate independent of LLM-based grading. Use alongside `.expect.md` for comprehensive evaluation.

---

## Random Selection + Mutation (Fixture-Based)

Each subagent has **three selection modes** chosen by probability:

```
For each subagent:
  1. Read .eval/fixtures/INDEX.md for available fixtures
  2. Select mode by probability:
     a. POOL_SAMPLE (60%) — pick random fixtures from INDEX.md
     b. MUTATE (30%) — copy an existing .jsonl, modify lines, save as new fixture
        e.g., copy workflow-create-dep-archive.jsonl
             → insert an error turn between turns 2 and 3
             → save as mutated-dep-archive-with-error.jsonl
     c. DISCOVER (10%) — write a brand new .jsonl fixture from scratch
        e.g., "what happens with conflicting priority updates?"
        → write a new stream capturing that scenario
```

### Per-Subagent Budget

```
  - Single-turn fixtures: 2-3
  - Multi-turn fixtures: 1
  - Mutation/Discovery: 0-1 new fixtures created
  - Total turns cap: ~12 turns per subagent
```

### Mutation Rules

1. **Copy existing fixture** to a new `.jsonl` file
2. **Vary one dimension**: insert error turn, change language, reorder sequence, add redundant turn
3. **Write `.expect.md`** if the mutated fixture is multi-turn
4. **Update INDEX.md** with the new entry
5. **Label**: "MUTATED FROM: {original_fixture} → {new_description}"

### Discovery Protocol

1. **Analyze the agent's tool surface** — what stream combinations haven't been tested?
2. **Write a new `.jsonl`** capturing the scenario as a stream of messages
3. **Write `.expect.md`** with per-turn expectations
4. **Update INDEX.md** with the new entry
5. **Label**: "DISCOVERED: {description} — rationale: {why}"
6. **Rate confidence**: "CONFIDENCE: high/medium/low"

### Main Context: Discovery Review

After all subagents report back, review DISCOVERED fixtures:

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

### Fixture Maintenance

Every 3 rounds, subagents should:
1. **Identify semantically duplicate fixtures** (same tool combo + same edge case)
2. **Merge**: keep the more comprehensive one, remove the other
3. **Update INDEX.md** accordingly
4. **Report**: "Removed X duplicates, Y fixtures remain"

---

## Smoke Tests (Mandatory Suite)

A small set of fixtures that MUST be run in EVERY round — during baseline and after every experiment evaluation. These are never randomly selected — they are always included.

### Current Smoke Test Set

- `create-task-english.jsonl`      — core: task creation
- `create-task-chinese.jsonl`      — core: Chinese input
- `update-task-single-field.jsonl` — core: no hallucination on update
- `create-task-duplicate.jsonl`    — core: dedup
- `error-empty-input.jsonl`        — core: error handling

### Maintenance

- Subagents may PROMOTE a fixture to smoke test if it fails critically
- Smoke tests should stay at 5-8 fixtures (fixed cost budget)
- Remove from smoke set only if the feature is removed

---

## Phase 1: Baseline

Run the baseline to establish current performance and identify what to improve.

### Procedure

1. **Run smoke tests** — always included, never randomized
2. **Run seed-guided fixtures** — sample from `INDEX.md` guided by the exploration seed, plus mutate/discover new fixtures
3. **Record pass/fail for each fixture**
4. **Identify failures** — categorize by severity, form hypotheses

### Seed Guidance

The exploration seed (provided in the prompt) shapes baseline fixture selection:
- **Focus area** — prioritize fixtures related to that area
- **Edge case direction** — mutate fixtures toward that edge case
- **Discovery direction** — discover new fixtures exploring that domain

### Baseline Report Format

```markdown
## Baseline Results

**Seed:** [exploration seed from prompt]
**Fixtures tested:** [count]
- Smoke tests: X/Y PASS
- Seed-guided: A/B PASS

**Baseline Pass Rate:** X/Y (Z%)

**Identified Issues (by severity):**
1. [Critical] [Issue description] → Hypothesis: [what to try]
2. [High] [Issue description] → Hypothesis: [what to try]
3. [Medium] [Issue description] → Hypothesis: [what to try]
```

---

## Phase 2: Iterative Experiments

From baseline failures, form hypotheses and test them **one at a time**. Each experiment gets an independent evaluation before the next one begins.

### Problem Severity Scale

| Severity | Meaning | Action |
|----------|---------|--------|
| Critical | Agent produces wrong data or crashes | Fix immediately |
| High | Agent behaves incorrectly but doesn't crash | Fix in this round |
| Medium | Agent could be better but works | Fix if easy |
| Low | Nice to have | Queue for future |

### Hypothesis Format

```
Exp [ID]: [Title]
Hypothesis: [What change will fix what problem]
Target: [prompt.go / tools.go / main.go]
Expected: [What the output should look like after]
```

### Subagent Autonomy

The experiment subagent should:
1. Understand the hypothesis and target file(s)
2. Decide the specific change to make (the main context does not prescribe exact edits)
3. Make the change, build (`go build`), and test (`go test`)
4. Report what was changed and why

### Experiment Report Format

```markdown
## Experiment [ID] - [Title]

**Hypothesis:** [What change will fix what problem]
**Target:** [file]

**Build:** PASS/FAIL
**Tests:** PASS/FAIL
**Result:** PASS/FAIL

**Output:** [actual JSON output]
**Observations:** [what you noticed]
```

---

## Phase 3: Evaluate (Per-Experiment)

After each experiment, run an **independent evaluation** using a fresh fixture set. This is the gate that decides keep or discard.

### Evaluation Protocol

1. **Fresh fixtures** — do NOT reuse the fixtures that motivated this experiment. Sample a new set from `INDEX.md` (seed-guided).
2. **Include smoke tests** — always re-run the full smoke suite as a regression gate.
3. **Include the targeted failure** — the specific fixture(s) that failed in baseline and this experiment aimed to fix.
4. **Include random extras** — check for regressions in unrelated areas.

### Keep / Discard Decision

```markdown
## Evaluation: Experiment [ID] - [Title]

**Fixtures tested:** [count]
**Targeted fix:** PASS/FAIL (the specific failure this experiment addressed)
**Smoke tests:** ALL PASS / FAILURES: [which ones]
**Regressions:** none / [which fixtures regressed]

**Decision:**
- If targeted fix PASS + smoke tests ALL PASS + no regressions
  → ✅ KEEP: commit this experiment's changes
- Otherwise
  → ❌ DISCARD: revert this experiment's changes (git checkout -- . on changed files)
```

### Stacking

Experiments are sequential. Each experiment builds on top of the last kept state:
```
baseline → exp1 (KEEP) → exp2 (DISCARD, revert to exp1 state) → exp3 (KEEP) → ...
```
Only the cumulative kept changes are committed at the end of the round.

---

## Output Artifacts

Each round generates these files in `.eval/round-{N}/`:

| File | Content |
|------|---------|
| `baseline.md` | Phase 1 baseline pass rate, identified issues, hypotheses |
| `experiments.md` | All experiment reports with individual keep/discard decisions |
| `final-report.md` | Summary of kept changes, discarded experiments, learnings, next round recommendations, cost & timing |

New fixtures (MUTATE/DISCOVER) are written to `.eval/fixtures/` and updated in `INDEX.md`.

---

## tingly-pm Specifics

### Architecture

```
tingly-pm/
├── main.go              # Entry point, mode selection, agent creation
├── prompt/prompt.go     # System prompt (highest impact for behavior)
├── tools/tools.go       # 18 agent tools (medium impact)
├── board/               # Data layer (task CRUD, members, timeline, reports)
├── .pm/                 # Runtime data
└── .eval/
    └── fixtures/        # Stream JSON test fixtures (*.jsonl + *.expect.md + INDEX.md)
```

### Impact Ranking

| File | Impact | Risk | Typical Change |
|------|--------|------|---------------|
| `prompt/prompt.go` | Highest | Zero | Add/modify behavioral instructions |
| `tools/tools.go` | Medium | Low | Fix bugs, improve output, add/remove tools |
| `main.go` | Low | Medium | Agent config (iterations, memory, modes) |
| `board/*.go` | Low | Low | Data layer (rarely needs changes) |

### Current Tool Inventory (10 tools)

| Tool | Purpose |
|------|---------|
| `create_task` | Create task with dedup |
| `update_task` | Update fields, archive (status=done/dropped), append comments (body_append) |
| `get_task` | Read task detail |
| `query_tasks` | List/filter with grouping + age, OR full-text search |
| `upsert_member` | Add or update team member |
| `query_members` | List/filter/search members by name AND labels |
| `remove_member` | Remove team member |
| `manage_dependency` | Add or remove blocked_by relation (action="add"/"remove") |
| `generate_report` | Generate report: daily, weekly, summary, or timeline |
| `save_session` | Save current session state |

### Removed Tools (Don't Re-Add)

| Tool | Removed in | Why | Replacement |
|------|-----------|-----|-------------|
| `add_comment` | Round 11 | Merged into update_task(body_append) | Comment is just body append |
| `archive_task` | Round 11 | Merged into update_task(status="done"/"dropped") | Auto-archive on terminal status |
| `summary` | Round 11 | Merged into generate_report(type="summary") | Unified report generation |
| `list_timeline` | Round 11 | Merged into generate_report(type="timeline") | Unified report generation |
| `assign_task` | Round 2 | Subset of `update_task(assignee=...)` | Prompt says "only set specified fields" |
| `show_blockers` | Round 4 | Merged into `list_tasks(show_blockers=true)` | Filter field |
| `register_member` | Round 8 | Merged into `upsert_member` | Upsert handles create + update |
| `update_member` | Round 8 | Merged into `upsert_member` | Upsert handles create + update |
| `list_tasks` | Round 9 | Merged into `query_tasks` | `SearchTasks` for search, `ListTasks` for listing |
| `search_tasks` | Round 9 | Merged into `query_tasks` | `SearchTasks` for search, `ListTasks` for listing |
| `list_members` | Round 9 | Merged into `query_members` | `SearchMembers` for search, `ListMembers` for listing |
| `search_members` | Round 9 | Merged into `query_members` | `SearchMembers` for search, `ListMembers` for listing |
| `add_dependency` | Round 9 | Merged into `manage_dependency(action="add")` | `AddDependency` tool |
| `remove_dependency` | Round 9 | Merged into `manage_dependency(action="remove")` | `RemoveDependency` tool |

### Known Constraints

| Constraint | Workaround |
|------------|-----------|
| Stateless mode can't resolve task names | Use JSONL fixture files (stream JSON) |
| Session persistence requires `.pm/sessions/` | Accept: cross-session context is lost |
| Console formatter leaks to stdout in chat mode | Accept: chat mode has ANSI output |

---

## Prompt Engineering Patterns

From 8 rounds of improvement:

### Do's

- **Keep sections organized** with `##` headers
- **Use `CRITICAL:` prefix** for rules the model repeatedly violates
- **Provide Chinese + English examples** for keyword mappings
- **Use positive instructions** ("search first") not negative ("don't ask for ID")
- **Document context dependencies** ("this requires session context")

### Don'ts

- **Don't flatten the structure** — sections are attention anchors
- **Don't add instructions that require context the model might not have**
- **Don't duplicate tool descriptions** — they're auto-generated
- **Don't add overly specific examples** — they cause overfitting
- **Don't add more than ~80 lines** — beyond that, attention degrades. See **Prompt Size Management** for tiered guidance.

### Current Prompt Sections

```
1. Task Creation     — dedup, fuzzy matching, slug generation
2. Task Updates      — CRITICAL: no hallucination
3. Member Assignment — validation, label assignment
4. Task References   — resolve informal names via search
5. Context Resolution — implicit references (this/that task)
6. Task ID Format    — TASK-YYYYMMDD-HHmmss
7. Status Lifecycle  — active vs terminal states
8. Priority Guide    — explicit Chinese/English mappings
9. Response Style    — language matching, conciseness
10. Tool Call Efficiency — batch when possible, avoid redundant calls
11. Task Reassignment — move assignee, update references
12. Member Labels    — role-based tags (e.g., frontend, backend)
```

### Prompt Size Management

| Tier | Lines | Guidance |
|------|-------|----------|
| Healthy | < 120 | Normal operation |
| Warning | 120–150 | Review before adding; consider merging low-value sections |
| Critical | > 150 | MUST compress before adding anything new |

Compression strategies (in order of preference):
1. **Merge related sections** (e.g., "Task Updates" + "Task Reassignment")
2. **Remove examples** that the model has consistently followed for 3+ rounds
3. **Move rarely-needed rules** to a "reference" block (lower attention priority)
4. **Shorten verbose Chinese/English keyword mappings** to a table

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
4. **Branch per round** — use `.worktree/exp-round-{N}` for isolation, merge to main only when confident
5. **Evaluate before keeping** — each experiment must pass independent evaluation + smoke tests before its changes are kept
6. **Revert immediately** — if an experiment fails evaluation, revert that experiment's changes before trying the next
7. **Commit with intent** — message explains what behavior improved
8. **Record everything** — learnings feed into next round's hypotheses

---

## Improvement Log

The improvement log tracks all rounds:

| Round | Focus | Experiments | Pass Rate | Subagents | Approach |
|-------|-------|-------------|-----------|----------|
| 1 | Fix fundamentals | 8 tests | 8/8 (100%) | — | Sequential |
| 2 | Priority, tools, search | 6 tests | 6/6 (100%) | — | Sequential |
| 3 | Contextual reasoning | 4 tests | 4/4 (100%) | — | Sequential |
| 4 | Output, timeline, consolidation | 3 parallel + integration | 3/3 + 1/1 | 3 | Parallel |
| 5 | Context resolution, fuzzy dupes | 2 parallel + verify | 2/2 + verified | 2 | v2 Parallel |
| 6 | Personnel focus, member validation | 7 new fixtures | 91→98% | 2 | v2 Parallel |
| 7 | Personnel CRUD (search/update/remove) | 9 new fixtures | 100→95% | 2 | v2 Parallel |
| 8 | Tool consolidation, colon format | 6 new fixtures | 97.2% | 2 | v2 Parallel |
| 2/2 (2nd loop) | Tool dedup, code cleanup | 5 new fixtures | 97.2% | 2 | v2 Parallel |
| 11 | CRUD consolidation | 4 merges | 100% | 1 | Manual |

### Starting State

```
tools: 14 (create_task, update_task, get_task, list_tasks, archive_task,
       search_tasks, add_comment, register_member, list_members,
       assign_task, add_dependency, remove_dependency, show_blockers,
       generate_report, summary, save_session)
prompt: 27 lines, flat unstructured
stdio mode: broken (ANSI escape codes in JSON output)
```

### Ending State

```
tools: 10 (create_task, update_task, get_task, list_tasks,
       archive_task, search_tasks, add_comment,
       upsert_member, remove_member, list_members, search_members,
       add_dependency, remove_dependency, generate_report, summary, save_session, list_timeline)
prompt: ~148 lines, 12 structured sections
stdio mode: clean JSON
fixtures: 56 (23 single + 33 multi)
assertions: eval-assert.sh (programmatic regression gate)
```

---

## Next Round Recommendations

Based on discovered gaps:

1. **High Priority:** Cross-language duplicate detection
2. **Medium Priority:** Aggressive duplicate threshold tuning
3. **Low Priority:** Fix "list members" to show both humans and agents
4. **Future:** Stateful context in `run` mode (architectural change)
