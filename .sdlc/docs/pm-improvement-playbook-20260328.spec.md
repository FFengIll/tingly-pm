# tingly-pm Improvement Playbook

**Scope**: tingly-pm specific improvement guidance
**Based on**: 4 rounds of iterative improvement (2026-03-28)
**Method**: [Agent Iterative Improvement Methodology v2](agent-improvement-methodology-20260328.spec.md) — parallel fuzzing eval + two-part verify-before-commit loop

---

## How to Use This Document

Before starting a new improvement round on tingly-pm:

1. Read the **Baseline Test Suite** — run it first to find current problems
2. Check **Known Constraints** — don't waste time on architectural limitations
3. Check **Prompt Do's and Don'ts** — don't regress on solved problems
4. Check **Tool Design Rules** — don't re-introduce removed patterns
5. Run your experiments following the **two-part loop** (see below)
6. **Update this document** with new learnings

### Two-Part Loop (v2)

Each improvement round has two parts:

**Part 1 — Experiment & Improve:**
1. Launch parallel fuzzing subagents with random feature subsets → baseline pass rate
2. Analyze failures → form hypotheses
3. Launch parallel experiment subagents (one per hypothesis)
4. Apply winners (but **don't commit yet**)

**Part 2 — Verify:**
1. Launch NEW parallel fuzzing subagents (reshuffled, different subsets)
2. Compare pass rate vs baseline
3. **Pass rate ≥ baseline → commit**
4. **Pass rate < baseline → revert ALL changes**

This prevents confirmation bias and catches cross-feature regressions.

---

## Architecture Overview

```
tingly-pm/
├── main.go              # Entry point, mode selection, agent creation
├── prompt/prompt.go     # System prompt (highest impact for behavior)
├── tools/tools.go       # 12 agent tools (medium impact)
├── board/               # Data layer (task CRUD, members, timeline, reports)
│   ├── task.go          # Task struct, CRUD, status lifecycle
│   ├── task_file.go     # Markdown frontmatter parse/format
│   ├── member.go        # Member CRUD
│   ├── timeline.go      # Append + read events (JSONL)
│   ├── report.go        # Summary + daily/weekly report generation
│   └── board.go         # EnsureInit (directory + git setup)
└── .pm/                 # Runtime data (tasks/, archive/, members.json, timeline.jsonl)
├── .eval/               # Eval loop artifacts (round-N/ with reports, decisions)
└── eval-loop.sh         # Outer loop driver (triggers claude -p per round)
```

### Impact Ranking

| File | Impact | Risk | Typical Change |
|------|--------|------|---------------|
| `prompt/prompt.go` | Highest | Zero | Add/modify behavioral instructions |
| `tools/tools.go` | Medium | Low | Fix bugs, improve output, add/remove tools |
| `main.go` | Low | Medium | Agent config (iterations, memory, modes) |
| `board/*.go` | Low | Low | Data layer (rarely needs changes for behavior) |

---

## Baseline Test Suite

Run this before every improvement round. Uses JSONL fixture files for both single and multi-turn tests.

```bash
# Setup
mkdir -p /tmp/test-pm-baseline
CONFIG="-config .pm"

# Multi-turn: full workflow (create, dedup, dependency, assign, archive, list)
cat .eval/fixtures/workflow-create-dep-archive.jsonl \
  | timeout 120 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null

# Multi-turn: context resolution by name
cat .eval/fixtures/context-resolve-by-name.jsonl \
  | timeout 60 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null

# Multi-turn: implicit references (this/that task)
cat .eval/fixtures/context-implicit-reference.jsonl \
  | timeout 120 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null

# Multi-turn: cross-language context
cat .eval/fixtures/context-cross-language.jsonl \
  | timeout 60 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null
```

### Expected Results

See individual `.expect.md` files for per-turn grading criteria.

| Fixture | Key Checks |
|---------|------------|
| workflow-create-dep-archive | Create p0/p1, dep by name, archive by name, clean list |
| context-resolve-by-name | Create, update by name (no ID asked), list confirms |
| context-implicit-reference | "第一个任务" resolves correctly, archives |
| context-cross-language | Chinese create, English "first task" update works |

### Single-Turn Quick Tests

```bash
# Error handling
cat .eval/fixtures/update-task-nonexistent.jsonl \
  | timeout 30 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null
# Expected: graceful error with suggestion

# Empty input
cat .eval/fixtures/error-empty-input.jsonl \
  | timeout 30 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null
# Expected: helpful intro message

# English input
cat .eval/fixtures/create-task-english.jsonl \
  | timeout 30 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null
# Expected: English response, correct priority

# Summary
cat .eval/fixtures/summary-stats.jsonl \
  | timeout 30 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null
# Expected: project stats
```

---

## Known Constraints

These are architectural limitations. Don't try to fix them via prompt.

| Constraint | Why | Workaround |
|------------|-----|-----------|
| Stateless mode can't resolve task names | Each `echo \| tingly-pm` is a fresh process with no memory | Use JSONL fixture files (`cat fixture.jsonl \| tingly-pm`) |
| Session persistence requires `.pm/sessions/` | Files only persist within same `.pm/` dir | Accept: cross-session context is lost |
| Tool schema is auto-generated from Go struct tags | No manual schema control | Change the struct, rebuild |
| Console formatter leaks to stdout in chat mode | `SetConsoleOutputEnabled` only affects `run`/`serve` | Accept: chat mode has ANSI output |
| `ListTasks` only scans `tasks/` dir | Archive is separate directory | `search_tasks` tool covers both; `ListTasks` is active-only by design |

---

## Prompt Do's and Don'ts

Current prompt has 6 sections. Changes must follow these rules:

### Do

- **Keep sections organized** with `##` headers
- **Use `CRITICAL:` prefix** for rules the model repeatedly violates
- **Provide Chinese + English examples** for any user-facing keyword mapping
- **Use positive instructions** ("search first") not negative ("don't ask for ID")
- **Document context dependencies** ("this requires session context")

### Don't

- **Don't flatten the structure** into a single paragraph — sections are attention anchors
- **Don't add instructions that require context the model might not have**
- **Don't duplicate tool descriptions** — they're auto-generated from struct tags
- **Don't add overly specific examples** — they cause overfitting to test inputs
- **Don't add more than ~80 lines** — beyond that, attention degrades

### Current Prompt Sections

```
1. Task Creation     — dedup, slug generation
2. Task Updates      — CRITICAL: no hallucination
3. Task References   — resolve informal names via search
4. Task ID Format    — TASK-YYYYMMDD-HHmmss
5. Status Lifecycle  — active vs terminal states
6. Priority Guide    — explicit Chinese/English mappings
7. Response Style    — language matching, conciseness
```

---

## Tool Design Rules

Current tool count: 12. Rules for any tool changes:

### Before Adding a Tool

- Does `list_tasks` or `search_tasks` with filters already cover this?
- Is this a write-only gap (can write but not read)?
- Will the model reliably choose this tool over similar ones?

### Before Removing a Tool

- Is the prompt strong enough to compensate? (e.g., "only set specified fields" made `assign_task` removable)
- Test the replacement path works for the same use case

### Tool Naming Convention

- Verb phrases: `create_task`, `list_tasks`, `search_tasks`
- Not nouns: `task_creator`, `task_list`
- Not overly specific: `search_tasks_by_title_and_body` → just `search_tasks`

### Output Format Rules

- Include all relevant context in the tool response (assignee, status, age)
- Don't rely on the LLM to reformat raw data — the tool should return scannable text
- Add computed fields the raw data doesn't have (age, grouping headers)
- Keep consistent assignee format: ` → name` or `-> name` (pick one, stick with it)

### Removed Tools (Don't Re-Add)

| Tool | Removed in | Why | Replacement |
|------|-----------|-----|-------------|
| `assign_task` | Round 2 | Subset of `update_task(assignee=...)` | Prompt says "only set specified fields" |
| `show_blockers` | Round 4 | Merged into `list_tasks(show_blockers=true)` | Filter field on existing tool |

---

## Current Tool Inventory

| Tool | Args | Purpose |
|------|------|---------|
| `create_task` | title, slug, priority?, assignee?, labels?, description? | Create task with dedup |
| `update_task` | task_id, status?, priority?, assignee?, labels?, title?, slug? | Update fields (no hallucination) |
| `get_task` | task_id | Read task detail |
| `list_tasks` | status?, assignee?, priority?, label?, show_blockers? | List/filter with grouping + age |
| `archive_task` | task_id, resolution (done/dropped) | Move to archive |
| `search_tasks` | query | Full-text search (active + archive) |
| `add_comment` | task_id, content, by? | Append comment to task body |
| `register_member` | name, type (human/agent), labels? | Add team member |
| `list_members` | type? | List team members |
| `add_dependency` | task_id, depends_on | Add blocked_by relation |
| `remove_dependency` | task_id, depends_on | Remove relation |
| `list_timeline` | limit? | Read recent timeline events |
| `generate_report` | report_type (daily/weekly) | Generate + save report |
| `summary` | — | Quick status stats |

---

## Execution Reference

### Build & Test

```bash
go build .                          # Must succeed
go test ./...                       # Must pass (board/ has tests)
```

### Run Tests

```bash
# Run a fixture (single or multi-turn)
cat .eval/fixtures/{name}.jsonl | ./tingly-pm -mode run -dir /tmp/test $CONFIG 2>/dev/null

# With output capture
cat .eval/fixtures/{name}.jsonl | ./tingly-pm -mode run -dir /tmp/test $CONFIG 2>/dev/null | tee output.jsonl

# Timeout by turns: 1=30s, 2-3=60s, 4-5=120s, 6+=180s
cat .eval/fixtures/{name}.jsonl | timeout 60 ./tingly-pm -mode run -dir /tmp/test $CONFIG 2>/dev/null
```

### Config

```bash
# Uses .pm/config.json for model settings (contains API key, gitignored)
# .pm/config.json is in .gitignore — don't commit it
CONFIG="-config .pm"
```

### Data Directory Structure

```
.pm/
├── tasks/           # Active tasks only (ls = meaningful)
├── archive/YYYYMM/  # Terminal tasks (done/dropped)
├── members.json     # Team roster
├── timeline.jsonl   # Append-only event log
├── reports/         # Generated reports
└── sessions/        # Session persistence

.eval/
└── fixtures/        # Stream JSON test fixtures (*.jsonl + *.expect.md + INDEX.md)
```

---

## Improvement Backlog

Potential improvements identified but not yet attempted:

| Priority | Improvement | Complexity | Notes |
|----------|------------|------------|-------|
| Medium | Merge `summary` into `list_tasks` | Low | Summary is just counts; list_tasks with `limit=0` could show stats |
| Medium | Proactive behavior prompt | Low | "Suggest checking blocked tasks when listing" |
| Low | Batch task creation | Medium | Model-dependent; prompt-only |
| Low | Multi-project support | High | Architectural change |
| Low | Task age-based alerts | Medium | "Task X has been todo for 5 days" |
