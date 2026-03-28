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

## Test Feature Pool

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
  "context: implicit references (this/that task)",
  "language: input language matching",
  "output: conciseness",
  "report: daily generation",
  "timeline: event ordering",
]
```

---

## Random Selection + Mutation

Each subagent has **three selection modes** chosen by probability:

```
For each subagent:
  1. Read the agent's prompt, tools, and codebase (understand the subject)
  2. Select mode by probability:
     a. POOL_SAMPLE (60%) — pick a random subset from the pool
     b. MUTATE (30%) — generate a VARIANT of a pool item
        e.g., pool has "create_task: chinese input"
             → mutate to "create_task: mixed chinese+english title"
             → mutate to "create_task: extremely long title (>200 chars)"
     c. DISCOVER (10%) — propose an entirely NEW feature/edge case
        e.g., "what happens if user assigns a task to a non-existent member?"
```

### Mutation Rules

1. **Read the agent's source** to understand boundaries
2. **Vary one dimension**: input format, sequence order, data volume, language, state
3. **Construct a valid test case**: input, execution command, expected behavior
4. **Label clearly**: "MUTATED FROM: {pool_item} → {new_test_description}"

### Discovery Protocol

1. **Analyze the agent's tool surface** — what combinations haven't been tested?
2. **Identify gaps** — what user scenario is not covered?
3. **Propose a test case** — input, execution command, expected behavior
4. **Label clearly**: "DISCOVERED: {new_feature} — rationale: {why}"
5. **Rate confidence**: "CONFIDENCE: high/medium/low"

### Main Context: Discovery Review

After all subagents report back, review DISCOVERED items:

```
For each DISCOVERED test:
  if confidence == "high" AND valid:
    → Add to POOL immediately
    → Run as additional verification
  if confidence == "medium":
    → Add to POOL as "candidate"
  if confidence == "low" OR invalid:
    → Discard, record idea for later
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

---

## tingly-pm Specifics

### Architecture

```
tingly-pm/
├── main.go              # Entry point, mode selection, agent creation
├── prompt/prompt.go     # System prompt (highest impact for behavior)
├── tools/tools.go       # 12 agent tools (medium impact)
├── board/               # Data layer (task CRUD, members, timeline, reports)
└── .pm/                 # Runtime data
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
| Stateless mode can't resolve task names | Use multi-message session |
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
