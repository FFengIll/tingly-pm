# tingly-pm Improvement Playbook

**Scope**: tingly-pm specific improvement guidance
**Based on**: 4 rounds of iterative improvement (2026-03-28)
**Method**: [Agent Iterative Improvement Methodology](agent-improvement-methodology-20260328.spec.md)

---

## How to Use This Document

Before starting a new improvement round on tingly-pm:

1. Read the **Baseline Test Suite** — run it first to find current problems
2. Check **Known Constraints** — don't waste time on architectural limitations
3. Check **Prompt Do's and Don'ts** — don't regress on solved problems
4. Check **Tool Design Rules** — don't re-introduce removed patterns
5. Run your experiments, then **update this document** with new learnings

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

Run this before every improvement round. Uses multi-message session for contextual tests.

```bash
# Setup
mkdir -p /tmp/test-pm-baseline
CONFIG="-config .pm"

printf '{
  "content": "创建一个任务：修复用户登录超时问题，高优先级"
}
{
  "content": "再创建一个：修复用户登录超时问题，高优先级"
}
{
  "content": "创建任务A：用户认证，p0"
}
{
  "content": "创建任务B：登录页面，p1"
}
{
  "content": "登录页面依赖用户认证"
}
{
  "content": "把用户认证分配给alice，登录页面分配给bob"
}
{
  "content": "用户认证完成了"
}
{
  "content": "列出所有任务"
}
{
  "content": "查看最近活动"
}
{
  "content": "有哪些任务被阻塞了"
}
{
  "content": "项目概况"
}
' | timeout 120 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG
```

### Expected Results

| Step | Expected Behavior | What to Check |
|------|-------------------|---------------|
| 1. Create (Chinese, 紧急) | Title stays Chinese, priority = p0 | No English title, p0 not p1 |
| 2. Duplicate | Refuses, references existing task | No second task created |
| 3-4. Create A & B | Two tasks created with correct priorities | p0 and p1 respectively |
| 5. Dependency | Resolves both by name, adds blocked_by | No "please provide task ID" |
| 6. Multi-assign | Assigns both correctly | No field hallucination |
| 7. Complete by name | Resolves → archives | Not asking for ID |
| 8. List tasks | Priority groups (=== P0 ===), age shown | Scannable format |
| 9. Timeline | Events listed newest-first | Timestamps present |
| 10. Blocked tasks | Shows tasks with blocked_by | Uses list_tasks not removed tool |
| 11. Summary | Contextual (mentions completed tasks) | Not generic template |

### Single-Request Quick Tests

```bash
# Error handling
echo '{"content":"update TASK-NONEXIST-12345 status to in_progress"}' \
  | timeout 30 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null
# Expected: graceful error with suggestion

# Empty input
echo '{"content":""}' \
  | timeout 30 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null
# Expected: helpful intro message

# English input
echo '{"content":"Create a task: implement OAuth2, assign to bob, p1"}' \
  | timeout 30 ./tingly-pm -mode run -dir /tmp/test-pm-baseline $CONFIG 2>/dev/null
# Expected: English response, correct priority
```

---

## Known Constraints

These are architectural limitations. Don't try to fix them via prompt.

| Constraint | Why | Workaround |
|------------|-----|-----------|
| Stateless mode can't resolve task names | Each `echo \| tingly-pm` is a fresh process with no memory | Use multi-message session (`printf '...\n...'`) |
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
# Single request (stateless)
echo '{"content":"..."}' | ./tingly-pm -mode run -dir /tmp/test $CONFIG 2>/dev/null

# Multi-message session (stateful)
printf '{"content":"..."}\n{"content":"..."}\n' | ./tingly-pm -mode run -dir /tmp/test $CONFIG 2>/dev/null
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
