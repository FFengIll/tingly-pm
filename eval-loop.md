# Eval Loop Specification

**Version:** 2.1
**Agent:** tingly-pm (AI project manager)
**Driver:** Claude Code (via `eval-loop.sh`)
**Date:** 2026-03-29

---

## Overview

The eval loop is a **two-part parallel fuzzing evaluation system** for iterative agent improvement. It treats an agent (prompt + tools + config) as a trainable subject, analogous to model training at the agent level.

```
Modify → Build → Execute → Observe → Evaluate → Commit or Revert → Repeat
```

### Directory Convention

| Directory | Purpose | Managed by |
|-----------|---------|------------|
| `.eval/` | Per-round artifacts (baseline results, experiment reports, decisions) | Claude Code (each round writes here) |
| `.pm/` | Agent runtime data (tasks, sessions, config) | Agent itself |

### Usage

```bash
./eval-loop.sh              # default: 4 rounds, auto-detect next round, parallel subagents
./eval-loop.sh -n 5 -m opus # custom rounds and model
./eval-loop.sh --dry-run    # preview without executing
```

See README.md for full CLI reference (`-s`, `-j`, `-d`, etc).

---

## The Two-Part Loop

Each improvement round follows the v2 methodology:

```
PART 1: Experiment & Improve
├── 1a. Parallel fuzzing baseline (N subagents: sample + mutate + discover)
├── 1b. Review discoveries → grow pool
├── 1c. Analyze failures → M hypotheses
├── 1d. Parallel experiments (M subagents, one per hypothesis)
├── 1e. Evaluate experiments → apply winners
└── (NO commit yet)

PART 2: Verify (Independent Evaluation)
├── 2a. Parallel fuzzing evaluation (N subagents, NEW random+mutated subsets)
├── 2b. Review new discoveries → grow pool
├── 2c. Compare vs baseline
├── 2d. PASS → commit | FAIL → revert all
└── Record learnings → feed into next round

→ Repeat (pool grows each round)
```

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

A small set of fixtures that MUST be run in EVERY round's Part 1 AND Part 2. These are never randomly selected — they are always included.

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

## Part 1: Baseline (Parallel Fuzzing)

Launch N subagents in parallel. Each:
1. Gets its assigned test subset
2. Runs each test in a clean environment (`/tmp/test-agent-rN`)
3. Records pass/fail/observation for each test
4. Returns a structured report

### Baseline Report Format

```markdown
## Baseline Results

**Setup:** N subagents with randomized feature selection
- Subagent 1: [category list] (X/Y PASS)
- Subagent 2: [category list] (X/Y PASS)
- ...

**Baseline Pass Rate:** X/Y (Z%)

**Identified Issues:**
1. [Issue description]
2. [Issue description]
```

---

## Part 1: Experiments

From baseline failures, form improvement hypotheses.

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

### Subagent Protocol

Each experiment subagent receives:
1. What file(s) to edit
2. What change to make
3. How to verify: `go build` + `go test`
4. How to execute: test input via stdio
5. Report format: structured pass/fail/observation

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

## Part 2: Verify (Independent Evaluation)

This is the critical phase — fresh random selections to validate improvements are real, not overfit to Part 1's test set.

### Verification Protocol

1. **New random shuffle** — do NOT reuse Part 1's assignments
2. **Same pool size** — same number of subagents, same test budget
3. **Must include baseline failures** — ensure Part 1's fixes are re-tested
4. **Must include random extras** — ensure no regressions in unrelated areas

### Go / No-Go Decision

```markdown
## Part 2 Verification Results

| Subagent | Pass Rate | New Features | Regressions |
|----------|-----------|--------------|-------------|
| 1 | X/Y (Z%) | [status] | [yes/no] |
| 2 | X/Y (Z%) | [status] | [yes/no] |
| ... | ... | ... | ... |

**Aggregate:** X/Y (Z%)
**Baseline was:** A/B (C%)

**Decision criteria (ALL must pass):**
1. **Aggregate pass rate:** Part2 >= Baseline
2. **Smoke test coverage:** ALL smoke tests must PASS
3. **No critical regression:** any fixture that passed in baseline and now fails = automatic FAIL (even if aggregate is higher)

**Result:**
- If all 3 pass → ✅ COMMIT: improvements verified, no regressions
- If any fails → ❌ REVERT ALL: git checkout -- . (analyze what regressed)
```

---

## Output Artifacts

Each round generates these files in `.eval/round-{N}/`:

| File | Content |
|------|---------|
| `baseline-results.md` | Part 1 baseline pass rate, identified issues |
| `experiments-part1.md` | Hypotheses, experiment reports, winner selection |
| `verification-part2.md` | Part 2 verification results, go/no-go decision |
| `final-report.md` | Summary of changes, learnings, next round recommendations, cost & timing (total subagent calls, approximate tokens, wall-clock time) |

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
4. **Branch per round** — merge to main only when confident
5. **Verify before commit** — Part 2 must pass before any commit
6. **Revert immediately** — if Part 2 fails, revert ALL
7. **Commit with intent** — message explains what behavior improved
8. **Record everything** — learnings feed into next round's hypotheses

---

## Running the Loop

The `eval-loop.sh` script handles the outer loop. It passes the prompt to `claude -p`, which reads this spec and executes one complete round.

Key behaviors you should be aware of:
- **Round numbering is auto-detected** — the script scans `.eval/` for existing round artifacts and starts at the next number. Your round number in the prompt reflects this.
- **Serial mode** — when the prompt contains `EXECUTION MODE: SERIAL`, you MUST run all subagents sequentially (one at a time), not in parallel. This applies to both Part 1 and Part 2 phases.
- **Exploration seed** — when the prompt contains an `EXPLORATION SEED`, prioritize that focus area in your hypotheses and test selection.

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
