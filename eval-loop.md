# Eval Loop Specification

**Version:** 2.0
**Agent:** tingly-pm (AI project manager)
**Driver:** Claude Code (via `eval-loop.sh`)
**Date:** 2026-03-28

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
# Run 4 rounds of improvement
./eval-loop.sh

# Run with custom rounds or model
./eval-loop.sh -n 5 -m opus

# Dry run (preview commands)
./eval-loop.sh --dry-run
```

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

Timeout by turn count: 1 turn=30s, 2-3 turns=60s, 4-5 turns=120s, 6+ turns=180s.

### Fixture Types

**Single-turn** (1 message per file):
- `create-task-chinese.jsonl`, `create-task-english.jsonl`, `error-empty-input.jsonl`, etc.

**Multi-turn** (2+ messages per file, with `.expect.md` for per-turn grading):
- `context-resolve-by-name.jsonl` (3 turns), `workflow-create-dep-archive.jsonl` (5 turns), etc.

### Initial Fixture Set

25 fixtures total: 16 single-turn + 9 multi-turn. See `INDEX.md` for the full list. This is a starting point — subagents MUTATE and DISCOVER new fixtures each round.

### Expectation Files

Multi-turn fixtures have companion `.expect.md` files with per-turn grading criteria (prose, not machine-parseable). Subagents read these to grade each turn independently.

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
  - Single-turn fixtures: 3-5
  - Multi-turn fixtures: 1-2
  - Mutation/Discovery: 0-2 new fixtures created
  - Total turns cap: ~20 turns per subagent
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

**Decision:**
- If Z >= C → ✅ COMMIT: improvements verified, no regressions
- If Z < C → ❌ REVERT ALL: git checkout -- . (analyze what regressed)
```

---

## Output Artifacts

Each round generates these files in `.eval/round-{N}/`:

| File | Content |
|------|---------|
| `baseline-results.md` | Part 1 baseline pass rate, identified issues |
| `experiments-part1.md` | Hypotheses, experiment reports, winner selection |
| `verification-part2.md` | Part 2 verification results, go/no-go decision |
| `final-report.md` | Summary of changes, learnings, next round recommendations |

New fixtures (MUTATE/DISCOVER) are written to `.eval/fixtures/` and updated in `INDEX.md`.

---

## tingly-pm Specifics

### Architecture

```
tingly-pm/
├── main.go              # Entry point, mode selection, agent creation
├── prompt/prompt.go     # System prompt (highest impact for behavior)
├── tools/tools.go       # 12 agent tools (medium impact)
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

### Current Tool Inventory (12 tools)

| Tool | Purpose |
|------|---------|
| `create_task` | Create task with dedup |
| `update_task` | Update fields (no hallucination) |
| `get_task` | Read task detail |
| `list_tasks` | List/filter with grouping + age |
| `archive_task` | Move to archive |
| `search_tasks` | Full-text search (active + archive) |
| `add_comment` | Append comment to task body |
| `register_member` | Add team member |
| `list_members` | List team members |
| `add_dependency` | Add blocked_by relation |
| `remove_dependency` | Remove relation |
| `list_timeline` | Read recent timeline events |
| `generate_report` | Generate + save report |
| `summary` | Quick status stats |

### Removed Tools (Don't Re-Add)

| Tool | Removed in | Why | Replacement |
|------|-----------|-----|-------------|
| `assign_task` | Round 2 | Subset of `update_task(assignee=...)` | Prompt says "only set specified fields" |
| `show_blockers` | Round 4 | Merged into `list_tasks(show_blockers=true)` | Filter field |

### Known Constraints

| Constraint | Workaround |
|------------|-----------|
| Stateless mode can't resolve task names | Use JSONL fixture files (stream JSON) |
| Session persistence requires `.pm/sessions/` | Accept: cross-session context is lost |
| Console formatter leaks to stdout in chat mode | Accept: chat mode has ANSI output |

---

## Prompt Engineering Patterns

From 5 rounds of improvement:

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
- **Don't add more than ~80 lines** — beyond that, attention degrades

### Current Prompt Sections

```
1. Task Creation     — dedup, fuzzy matching, slug generation
2. Task Updates      — CRITICAL: no hallucination
3. Task References   — resolve informal names via search
4. Task ID Format    — TASK-YYYYMMDD-HHmmss
5. Status Lifecycle  — active vs terminal states
6. Priority Guide    — explicit Chinese/English mappings
7. Response Style    — language matching, conciseness
8. Context Resolution — implicit references (this/that task)
```

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

The `eval-loop.sh` script handles the outer loop:

```bash
#!/usr/bin/env bash
# Usage: ./eval-loop.sh [options]

# Options:
#   -n, --rounds <N>      Number of rounds (default: 4)
#   -m, --model <model>   Claude model (omit for default)
#   -p, --prompt <file>   Custom prompt file
#       --dry-run         Print commands without executing
#   -h, --help            Show help
```

The script reads the prompt and passes it to `claude -p`, which:
1. Reads this spec document
2. Executes ONE complete round
3. Writes per-round artifacts to `.eval/`
4. Commits or reverts based on Part 2 verification
5. Appends results to improvement log

---

## Improvement Log

The improvement log tracks all rounds:

| Round | Focus | Experiments | Pass Rate | Approach |
|-------|-------|-------------|-----------|----------|
| 1 | Fix fundamentals | 8 tests | 8/8 (100%) | Sequential |
| 2 | Priority, tools, search | 6 tests | 6/6 (100%) | Sequential |
| 3 | Contextual reasoning | 4 tests | 4/4 (100%) | Sequential |
| 4 | Output, timeline, consolidation | 3 parallel + integration | 3/3 + 1/1 | Parallel |
| 5 | Context resolution, fuzzy dupes | 2 parallel + verify | 2/2 + verified | v2 Parallel |

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
tools: 12 (removed assign_task, show_blockers; added list_timeline)
prompt: ~73 lines, 8 structured sections (added Context Resolution)
stdio mode: clean JSON
```

---

## Next Round Recommendations

Based on discovered gaps:

1. **High Priority:** Cross-language duplicate detection
2. **Medium Priority:** Aggressive duplicate threshold tuning
3. **Low Priority:** Fix "list members" to show both humans and agents
4. **Future:** Stateful context in `run` mode (architectural change)
