# tingly-pm Improvement Playbook

**Scope**: tingly-pm specific improvement guidance
**Based on**: 11 rounds of iterative improvement (2026-03-28 ~ 2026-03-30)
**Method**: [Agent Iterative Improvement Methodology v2](agent-improvement-methodology-20260328.spec.md) — parallel fuzzing eval + two-part verify-before-commit loop

---

## How to Use This Document

Before starting a new improvement round on tingly-pm:

1. Read the **Baseline Test Suite** — run `eval-assert.sh smoke` first
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

### Experimental Branches (`.worktree/`)

Experiments and eval rounds run in git worktrees under `.worktree/`:

```
.worktree/
├── exp-round-N-name/    # Per-round experiment branch
├── exp-feature-x/       # Feature experiment
└── ...
```

- `.worktree/` is gitignored — worktrees are ephemeral
- Merge to main only after Part 2 verification passes
- Clean up after merge: `git worktree remove .worktree/exp-name`

---

## Architecture Overview

```
tingly-pm/
├── main.go              # Entry point, mode selection, agent creation
├── prompt/prompt.go     # System prompt (highest impact for behavior)
├── tools/tools.go       # 16 agent tools (medium impact)
├── board/               # Data layer (task CRUD, members, timeline, reports)
│   ├── task.go          # Task struct, CRUD, status lifecycle
│   ├── task_file.go     # Markdown frontmatter parse/format
│   ├── member.go        # Member CRUD + QueryMembers (name/label search)
│   ├── timeline.go      # Append + read events (JSONL)
│   ├── report.go        # Summary + daily/weekly report generation
│   └── board.go         # EnsureInit (directory + git setup)
└── .pm/                 # Runtime data (tasks/, archive/, members.json, timeline.jsonl)
├── .eval/               # Eval loop artifacts (round-N/, fixtures/)
├── .worktree/           # Experimental branches (gitignored)
├── eval-loop.sh         # Outer loop driver (triggers claude -p per round)
└── eval-assert.sh       # Automated fixture regression testing
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

Run this before every improvement round.

```bash
# Automated assertions (recommended)
./eval-assert.sh smoke          # 5 smoke tests only
./eval-assert.sh                # all 56 fixtures
./eval-assert.sh -v workflow-create-dep-archive  # verbose single fixture

# Manual fixture execution
cat .eval/fixtures/{name}.jsonl \
  | timeout {N} ./tingly-pm -mode run -dir /tmp/test-{id} -config .pm 2>/dev/null
```

Timeout by turn count: 1=15s, 2-3=30s, 4-5=60s, 6+=90s.

### Smoke Tests (Mandatory)

| Fixture | Category | Description |
|---------|----------|-------------|
| create-task-english | create | English input, explicit priority |
| create-task-chinese | create | Chinese input, keyword priority detection |
| update-task-single-field | update | No hallucination on update |
| create-task-duplicate | create | Dedup detection |
| error-empty-input | error | Empty content handling |

### Key Workflow Fixtures

| Fixture | Turns | Description |
|---------|-------|-------------|
| workflow-create-dep-archive | 5 | Full lifecycle: create → dep → archive → list |
| workflow-create-assign-list | 4 | Create → register member → assign → list |
| context-ordinal-reference | 4 | Create tasks, resolve ordinal reference ("第一个任务") |
| context-cross-language | 3 | Chinese create, English update by reference |
| discovered-member-removal-workflow | 7 | Register → create tasks → remove member → verify |
| report-types-session | 4 | Weekly report + summary + session save |

See `.eval/fixtures/INDEX.md` for the full fixture manifest.

---

## Known Constraints

These are architectural limitations. Don't try to fix them via prompt.

| Constraint | Why | Workaround |
|------------|-----|-----------|
| Stateless mode can't resolve task names | Each `echo \| tingly-pm` is a fresh process with no memory | Use JSONL fixture files (`cat fixture.jsonl \| tingly-pm`) |
| Session persistence requires `.pm/sessions/` | Files only persist within same `.pm/` dir | Accept: cross-session context is lost |
| Tool schema is auto-generated from Go struct tags | No manual schema control | Change the struct, rebuild |
| Console formatter leaks to stdout in chat mode | `SetConsoleOutputEnabled` only affects `run`/`serve` | Accept: chat mode has ANSI output |
| `ListTasks` only scans `tasks/` dir | Archive is separate directory | `SearchTasks` covers both; `ListTasks` is active-only by design |

---

## Prompt Do's and Don'ts

Current prompt has 12 sections (~148 lines). Changes must follow these rules:

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
- **Don't exceed 150 lines** — use Prompt Size Management (see eval-loop.md)

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

---

## Tool Design Rules

Current tool count: 16 (Go methods). Rules for any tool changes:

### Before Adding a Tool

- Does `ListTasks`/`SearchTasks` with filters already cover this?
- Is this a write-only gap (can write but not read)?
- Will the model reliably choose this tool over similar ones?

### Before Removing a Tool

- Is the prompt strong enough to compensate?
- Test the replacement path works for the same use case

### Tool Naming Convention

- PascalCase: `CreateTask`, `ListTasks`, `SearchTasks`
- Tool names are derived from Go method names via `RegisterAll`

### Output Format Rules

- Include all relevant context in the tool response (assignee, status, age)
- Don't rely on the LLM to reformat raw data — return scannable text
- Add computed fields the raw data doesn't have (age, grouping headers)

### Removed Tools (Don't Re-Add)

| Tool | Removed in | Why | Replacement |
|------|-----------|-----|-------------|
| `assign_task` | Round 2 | Subset of `update_task(assignee=...)` | Prompt says "only set specified fields" |
| `show_blockers` | Round 4 | Merged into `list_tasks(show_blockers=true)` | Filter field |
| `register_member` | Round 8 | Merged into `upsert_member` | Upsert handles create + update |
| `update_member` | Round 8 | Merged into `upsert_member` | Upsert handles create + update |
| `add_dependency` | Round 9 | Merged into `manage_dependency(action="add")` | Action parameter |
| `remove_dependency` | Round 9 | Merged into `manage_dependency(action="remove")` | Action parameter |
| `add_comment` | Round 11 | Merged into `update_task(body_append)` | Comment is just body append |
| `archive_task` | Round 11 | Merged into `update_task(status="done"/"dropped")` | Auto-archive on terminal status |
| `summary` | Round 11 | Merged into `generate_report(type="summary")` | Unified report generation |
| `list_timeline` | Round 11 | Merged into `generate_report(type="timeline")` | Unified report generation |

---

## Current Tool Inventory (16 Go methods)

| Tool | Args | Purpose |
|------|------|---------|
| `CreateTask` | title, slug, priority?, assignee?, labels?, description? | Create task with dedup |
| `UpdateTask` | task_id, status?, priority?, assignee?, labels?, title?, slug? | Update fields (no hallucination) |
| `GetTask` | task_id | Read task detail |
| `ListTasks` | status?, assignee?, priority?, label?, show_blockers? | List/filter with grouping + age |
| `ArchiveTask` | task_id, resolution (done/dropped) | Move to archive |
| `SearchTasks` | query | Full-text search (active + archive) |
| `AddComment` | task_id, content, by? | Append comment to task body |
| `UpsertMember` | name, member_type?, labels? | Create or update team member |
| `ListMembers` | member_type? | List team members |
| `SearchMembers` | labels | Search members by labels |
| `RemoveMember` | name | Remove team member |
| `AddDependency` | task_id, depends_on | Add blocked_by relation |
| `RemoveDependency` | task_id, depends_on | Remove relation |
| `GenerateReport` | report_type (daily/weekly) | Generate + save report |
| `Summary` | — | Quick status stats |
| `SaveSession` | label? | Save conversation state |

---

## Execution Reference

### Build & Test

```bash
go build -o tingly-pm .            # Must succeed
go test ./...                      # Must pass (board/ has tests)
./eval-assert.sh smoke             # Automated smoke tests
```

### Run Fixtures

```bash
# Automated assertions
./eval-assert.sh                    # all fixtures
./eval-assert.sh smoke              # smoke tests only
./eval-assert.sh -v {name}          # verbose single fixture

# Manual execution
cat .eval/fixtures/{name}.jsonl | timeout {N} ./tingly-pm -mode run -dir /tmp/test-{id} -config .pm 2>/dev/null
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
├── fixtures/        # Test fixtures (*.jsonl + *.expect.md + INDEX.md)
├── round-{N}/       # Per-round reports and decisions
└── round-{N}.log    # Full Claude output per round

.worktree/           # Experimental branches (gitignored, ephemeral)
```

---

## Improvement Backlog

| Priority | Improvement | Complexity | Notes |
|----------|------------|------------|-------|
| High | Cross-language duplicate detection | Medium | LLM semantic gap — Chinese/English dedup |
| Medium | Aggressive duplicate threshold tuning | Low | Fuzzy match sensitivity |
| Low | Fix "list members" to show both humans and agents | Low | Default ListMembers behavior |
| Future | Stateful context in `run` mode | High | Architectural change |
