# Spec: PM Agent Iterative Improvement Loop

**Date:** 2026-03-28
**Scope:** tingly-pm self-improvement via external driver (Claude Code)
**Status:** Completed (3 rounds)

---

## Overview

A workflow where Claude Code iteratively improves tingly-pm by treating it as the subject being improved — analogous to "model training" but at the agent level.

```
Modify → Build → Execute → Observe → Evaluate → Commit or Revert → Repeat
```

This is **not** a feature of tingly-pm itself. It is an external improvement process driven by Claude Code.

---

## The Loop

```
┌──────────────────────────────────────────────────────┐
│              Claude Code (the driver)                 │
│                                                       │
│  1. Identify what to improve                          │
│  2. Hypothesis: what change will fix it               │
│  3. Edit prompt.go / tools.go / main.go               │
│  4. go build . && go test ./...                       │
│  5. Run tingly-pm with test input via stdio           │
│  6. Observe output                                   │
│  7. Good? → git commit                                │
│     Bad? → git revert, try different approach         │
│  8. Repeat                                           │
└──────────────────────────────────────────────────────┘
         │                            ▲
         ▼                            │
   ┌───────────┐              ┌─────────────┐
   │ tingly-pm │──stdout────▶│ evaluation  │
   │ (subject) │              │             │
   └───────────┘              └─────────────┘
         │
         ▼
   .pm/ (task board state change)
```

## Execution Interface

tingly-pm supports `-mode run` (stdio JSON). This is the primary execution interface.

```bash
# Single request (stateless — fresh session each time)
echo '{"content":"创建一个任务：修复登录bug"}' | ./tingly-pm -mode run -dir /tmp/test-pm -config .pm

# Multi-message session (agent retains context across messages)
printf '{"content":"创建任务A：认证"}\n{"content":"创建任务B：登录"}\n{"content":"B依赖A"}\n' \
  | ./tingly-pm -mode run -dir /tmp/test-pm -config .pm
```

**Key insight**: Single-request mode tests raw tool correctness. Multi-message session tests contextual reasoning (task reference resolution, multi-step workflows).

## What Gets Improved

### 1. System Prompt (`prompt/prompt.go`) — highest impact

Prompt changes directly affect all behavior. Organized into sections:
- Task Creation (dedup, slug generation)
- Task Updates (CRITICAL: no field hallucination)
- Task References (resolve informal names via search)
- Status Lifecycle
- Priority Guide (explicit Chinese/English mappings)
- Response Style (language matching, conciseness)

### 2. Tool Behavior (`tools/tools.go`)

- Remove redundant tools (fewer tools = less confusion)
- Fix tool logic bugs (search scope, blocker detection)
- Improve output formatting (include assignee, status in search results)

### 3. Agent Configuration (`main.go`)

- Disable console output for programmatic modes (run/serve)

## Evaluation Criteria

| Criterion | How to Check |
|-----------|-------------|
| Tool correctness | Agent calls the right tool with valid args |
| No hallucination | Doesn't fabricate field values, task IDs, member names |
| Response quality | Output is concise, accurate, actionable |
| Language consistency | Chinese input → Chinese response |
| Error handling | Graceful error with helpful suggestion |
| Stateful reasoning | Resolves informal task references in session context |

## Safety Protocol

1. **Always build before run** — `go build .` must succeed
2. **Always test before commit** — `go test ./...` must pass
3. **Isolated test environment** — `-dir /tmp/test-pm-<round>`, never real project data
4. **One logical change at a time** — Edit, test, evaluate before next change
5. **Revert if tests fail** — `git checkout -- <file>` immediately
6. **Commit with intent** — Each commit message explains what behavior improved
7. **Branch per round** — `git checkout -b experiment/round-N`; merge to main only if all pass

## Test Isolation

Each improvement iteration uses a clean `.pm/` directory:

```bash
mkdir -p /tmp/test-pm-rN        # N = round number
echo '{"content":"..."}' | ./tingly-pm -mode run -dir /tmp/test-pm-rN -config .pm
```

---

## Round 1 — Fix Fundamentals

**Branch:** `experiment/round-1-improvements`
**Commit:** `01802b9`

### Baseline Problems Found

| # | Test | Observation | Severity |
|---|------|-------------|----------|
| 1 | Create task (Chinese) | Title translated to English; ANSI escape codes leaked into JSON stdout | **Critical** |
| 2 | Duplicate creation | Created 3 identical tasks with no dedup check | **High** |
| 3 | Update + assign | Hallucinated title ("Update user profile UI"), slug, priority when only assignee was requested | **Critical** |
| 4 | List tasks | Plain format, acceptable | OK |
| 5 | Summary | OK | OK |
| 6 | Archive | OK | OK |

### Changes

1. **`main.go`**: `ag.SetConsoleOutputEnabled(false)` for `run` and `serve` modes
2. **`prompt/prompt.go`**: Complete rewrite with structured sections:
   - Task Creation: "Before creating, always search existing tasks to avoid duplicates"
   - Task Updates: "CRITICAL: ONLY include fields the user explicitly asked to change"
   - Priority Guide: explicit p0–p3 definitions
   - Response Style: language matching, title language preservation

### Results

| Test | Before | After | Verdict |
|------|--------|-------|---------|
| Create task (Chinese) | Title English, ANSI leak | Title Chinese, clean JSON | PASS |
| Duplicate creation | Created 3rd duplicate | Detected existing, upgraded instead | PASS |
| Update + assign | Hallucinated 3 fields | Only changed assignee | PASS |
| List tasks | Plain | Better formatted | PASS |
| Archive | OK | OK | PASS |
| Error handling (invalid ID) | N/A | Graceful with suggestion | PASS |
| Report generation | N/A | Good formatted report | PASS |
| Language matching | Mixed | Matches input | PASS |

### Learnings

- **Prompt structure > length**: Organized sections with headers beat a flat instruction list
- **"CRITICAL" prefix works**: A single `CRITICAL:` paragraph stopped the hallucination
- **Console leak was silent**: Chat mode worked fine, stdio mode was broken for programmatic use

---

## Round 2 — Priority, Tool Consolidation, Search

**Branch:** `experiment/round-1-improvements` (continued)
**Commit:** `157ef7e`

### Baseline Problems Found

| # | Test | Observation | Severity |
|---|------|-------------|----------|
| 1 | "高优先级" input | Mapped to p1 instead of p0 | **High** |
| 2 | assign_task tool | Redundant with update_task (just sets assignee) | Medium |
| 3 | search_tasks | Only searched active tasks, not archive | Low |
| 4 | show_blockers | Only listed tasks with status=blocked, missed tasks with blocked_by relations | Medium |

### Changes

1. **`prompt/prompt.go`**: Added explicit Chinese keyword→priority mappings:
   - p0: "高优先级", "紧急", "最高优先级"
   - p1: "重要", "尽快"
   - p2: "一般", "普通"
   - p3: "低优先级", "有空再做"

2. **`tools/tools.go`**: Removed `assign_task` tool (14→13 tools)

3. **`tools/tools.go`**: `search_tasks` now scans both `tasks/` and `archive/` directories

4. **`tools/tools.go`**: `show_blockers` scans all tasks with non-empty `blocked_by` field regardless of status

### Results

| Test | Before | After | Verdict |
|------|--------|-------|---------|
| "紧急" → p0 | p1 | p0 | PASS |
| Assign via update_task only | N/A | Works correctly | PASS |
| Search archived tasks | Only active | Both active + archive | PASS |
| show_blockers (in_progress + blocked_by) | "No blocked tasks" | Shows relation correctly | PASS |
| Status-only update | N/A | No field contamination | PASS |
| Member operations | OK | OK | PASS |

### Learnings

- **Explicit Chinese mappings beat English definitions**: "critical/blocking/must fix" didn't help; "紧急" → p0 did
- **Tool reduction is safe**: Removing assign_task didn't hurt — prompt guidance made the convenience wrapper unnecessary
- **Field-presence scan > status filter**: show_blockers by `blocked_by != []` is more useful than by `status == "blocked"`

---

## Round 3 — Contextual Reasoning

**Branch:** `experiment/round-1-improvements` (continued)
**Commit:** `2d098ec`

### Baseline Problems Found

| # | Test | Observation | Severity |
|---|------|-------------|----------|
| 1 | "B依赖A" (stateless) | Asked user for task IDs instead of resolving | **High** |
| 2 | Multi-assign in one message | N/A | Untested |
| 3 | Complete by informal name | N/A | Untested |

### Changes

1. **`prompt/prompt.go`**: Added "Task References" section:
   - "When users refer to tasks by informal names, use search_tasks to find them first"
   - "NEVER ask the user to provide a task ID — resolve it yourself"

### Results

| Test | Before | After | Verdict |
|------|--------|-------|---------|
| "登录页面依赖用户认证" (session) | Asked for IDs | Resolved both names + added dependency | PASS |
| Multi-assign ("认证给alice，登录页面给bob") | N/A | Assigned both correctly | PASS |
| Complete by name ("用户认证完成了") | N/A | Resolved → archived | PASS |
| Smart summary | N/A | Noted alice finished, bob still active | PASS |

### Learnings

- **Session context is essential for reference resolution**: Stateless mode (separate processes) fundamentally can't resolve informal names — this is architectural, not fixable by prompt
- **Multi-message sessions unlock complex workflows**: Piping multiple JSON lines into one process lets the agent build context
- **The prompt works as intended when context exists**: The instruction is correct; the limitation is the execution mode

---

## Final State

### Commits on main

```
5596371 docs: record round 2-3 experiment results in feedback spec
2d098ec feat: add task reference resolution instructions to prompt
157ef7e feat: tune priority mapping, remove redundant tool, fix search + blockers
01802b9 feat: improve agent behavior — fix console leak, enhance prompt
```

### Files Changed

| File | Changes |
|------|---------|
| `main.go` | +2 lines: disable console output for run/serve |
| `prompt/prompt.go` | Rewritten: 27 lines → ~60 lines, 6 structured sections |
| `tools/tools.go` | Removed assign_task, fixed search_tasks (archive), fixed show_blockers |

### Tools: 14 → 13

| Removed | Reason |
|---------|--------|
| `assign_task` | Redundant with `update_task(assignee=...)` |

### Remaining Improvement Opportunities

1. **Batch operations** — create multiple tasks in one message (prompt-only, model dependent)
2. **`summary` vs `list_tasks` overlap** — summary provides counts, list provides details; could merge
3. **`show_blockers` vs `list_tasks(status=blocked)`** — now overlapping; could merge
4. **Proactive behavior** — agent could suggest actions (e.g., "task X has been blocked for 3 days")
5. **Timeline queries** — no tool to read timeline events (only append)
6. **Multi-project support** — agent managing multiple `.pm/` directories
