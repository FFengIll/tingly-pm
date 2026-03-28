# Spec: PM Agent Iterative Improvement Loop

**Date:** 2026-03-28
**Scope:** tingly-pm self-improvement via external driver (Claude Code)
**Status:** Draft

---

## Overview

Create a workflow where Claude Code iteratively improves tingly-pm by:

1. **Modify** — Edit tingly-pm source code (prompt, tools, logic)
2. **Execute** — Run tingly-pm, interact with it via stdio mode
3. **Observe** — Check if it behaves correctly (response quality, tool usage)
4. **Evaluate** — Judge good or bad based on observed behavior
5. **Decide** — Commit if good, discard if bad, iterate

This is **not** a feature of tingly-pm itself. It is an external improvement process driven by Claude Code, treating tingly-pm as the subject being improved.

## The Loop

```
┌──────────────────────────────────────────────┐
│           Claude Code (the driver)           │
│                                              │
│  1. Read current tingly-pm code              │
│  2. Identify what to improve                 │
│  3. Edit prompt.go / tools.go / main.go      │
│  4. Build & run tingly-pm                    │
│  5. Send test input via stdio                │
│  6. Observe output                           │
│  7. Good? → commit                           │
│     Bad? → revert, try different approach    │
└──────────────────────────────────────────────┘
         │                            ▲
         ▼                            │
   ┌───────────┐              ┌─────────────┐
   │ tingly-pm │──stdout────▶│ evaluation  │
   │ (subject) │              │ (manual or  │
   └───────────┘              │  automated) │
         │                    └─────────────┘
         ▼
   .pm/ (task board state change)
```

## Execution Interface

tingly-pm already supports `-mode run` (stdio JSON):

```bash
echo '{"content":"创建一个任务：修复登录bug"}' | go run main.go -mode run -dir /tmp/test-pm
# → {"role":"assistant","content":"Created TASK-..."}
```

This is the primary execution interface. Claude Code:
1. Spawns tingly-pm as a subprocess
2. Sends JSON input via stdin
3. Reads JSON output from stdout
4. Evaluates the response

## What Gets Improved

### 1. System Prompt (`prompt/prompt.go`)

The most impactful target. Prompt changes directly affect behavior.

Current prompt is short (~27 lines). Can be improved to:
- Better task slug generation (less hallucination)
- Proactive status tracking (ask about blocked tasks)
- Smarter report generation
- Better Chinese/English handling
- More concise responses
- Correct tool argument usage

### 2. Tool Behavior (`tools/tools.go`)

- Error messages clarity
- Validation robustness
- Output formatting
- Edge case handling

### 3. Agent Configuration (`main.go`)

- MaxIterations tuning
- Memory size tuning
- Model parameters

## Improvement Scenarios

### Scenario: Prompt Engineering

```
Goal: PM should proactively ask about blockers when listing tasks

1. Edit prompt/prompt.go — add instruction about checking blockers
2. Build: go build .
3. Run: echo '{"content":"列出当前任务"}' | ./tingly-pm -mode run -dir /tmp/test-pm
4. Observe: Does the agent mention blocked tasks or show_blockers?
5. Good → commit
6. Bad → try different wording, revert if stuck
```

### Scenario: Tool Output Improvement

```
Goal: list_tasks output should include task age for better prioritization

1. Edit tools/tools.go — add age calculation to ListTasks
2. Build + Test: go test ./board/ ./tools/...
3. Run: echo '{"content":"有哪些任务"}' | ./tingly-pm -mode run -dir /tmp/test-pm
4. Observe: Does output show task age?
5. Verify: go test still passes
6. Good → commit
```

### Scenario: Behavior Correction

```
Goal: Agent creates duplicate tasks instead of checking existing ones first

1. Run: echo '{"content":"创建任务：修复登录"}' then echo '{"content":"创建任务：修复登录"}'
2. Observe: Agent creates two identical tasks (bad)
3. Edit prompt/prompt.go — add instruction to search existing tasks before creating
4. Re-run same sequence
5. Observe: Agent now checks and warns about duplicates
6. Good → commit
```

## Test Isolation

Each improvement iteration needs a clean `.pm/` directory to avoid state contamination:

```bash
# Before each test run
rm -rf /tmp/test-pm
mkdir -p /tmp/test-pm

# Run with isolated state
echo '{"content":"..."}' | go run . -mode run -dir /tmp/test-pm
```

## Evaluation Criteria

| Criterion | How to Check |
|-----------|-------------|
| Tool correctness | Agent calls the right tool with valid args |
| Response quality | Output is concise, accurate, actionable |
| Error handling | Agent handles edge cases gracefully |
| Prompt following | Agent follows system prompt instructions |
| Language consistency | Chinese input → Chinese response (or appropriate mix) |
| No hallucination | Doesn't invent task IDs, member names, etc. |

## Safety Protocol

1. **Always build before run** — `go build .` must succeed
2. **Always test before commit** — `go test ./...` must pass
3. **Isolated test environment** — Use `-dir /tmp/test-pm`, never real project data
4. **One change at a time** — Edit one thing, test, evaluate
5. **Revert if tests fail** — If `go test` breaks, revert immediately
6. **Commit with intent** — Each commit message explains what behavior was improved

## Workflow Template

For each improvement iteration:

```
1. Identify: What behavior needs improvement?
2. Hypothesis: What change (prompt/code) will fix it?
3. Implement: Edit the file
4. Verify: go build . && go test ./...
5. Execute: Run tingly-pm with test input
6. Observe: Does the output match expectation?
7. Decide:
   - go test fails → revert
   - output wrong → revert, try different approach
   - output correct → commit with descriptive message
8. Repeat
```

## Current State & First Targets

Based on current code analysis, likely improvement areas:

1. **Prompt is minimal** (27 lines) — room for much better behavior guidance
2. **No duplicate checking** — agent can create identical tasks
3. **list_tasks output is plain** — no age, no grouping
4. **No proactive behavior** — agent only reacts, doesn't suggest
5. **Error messages are raw** — tool errors not user-friendly
6. **No task priority guidance** — prompt doesn't say when to use p0 vs p3

## Dependencies

- None. Uses existing tingly-pm `-mode run` interface
- Claude Code provides the driver loop (edit, build, run, evaluate)
- `.pm/` directory provides clean state for each test

---

## Round 1 Results (2026-03-28)

### Changes Made

1. **`main.go`**: Disable console output for `run`/`serve` modes via `SetConsoleOutputEnabled(false)`
2. **`prompt/prompt.go`**: Rewrite system prompt with structured sections

### Before vs After

| Test | Before | After | Result |
|------|--------|-------|--------|
| Create task (Chinese input) | Title translated to English, ANSI leak in JSON | Title stays Chinese, clean JSON | PASS |
| Duplicate creation | Created 3 identical tasks | Detected existing, upgraded priority | PASS |
| Update + assign | Hallucinated title/slug/priority fields | Only changed specified fields | PASS |
| List tasks | Plain format | Better formatted | PASS |
| Archive | OK | OK | PASS |
| Error handling (invalid ID) | N/A | Graceful error with suggestion | PASS |
| Report generation | N/A | Good formatted report | PASS |
| Language matching | Mixed | Matches input language | PASS |

### Remaining Issues

1. **"高优先级" interpreted as p1 not p0** — prompt says p0=critical, model chose p1 for "high priority". Need to map "高优先级" → p0 explicitly.
2. **`search_tasks` only searches active tasks** — doesn't check archive. Minor.
3. **14 tools may be too many** — some could be merged (e.g., `assign_task` is a subset of `update_task`). Future investigation.
4. **No batch operations** — can't create multiple tasks at once. Future consideration.

### Commit

`01802b9` — feat: improve agent behavior — fix console leak, enhance prompt

### Key Learnings

- **Prompt structure matters more than length** — organized sections (Task Creation, Task Updates, Status, Priority, Response Style) work better than a flat list
- **"CRITICAL" labels in prompt work** — the hallucination problem was fixed with a single `CRITICAL:` prefixed paragraph
- **Console leak was a silent bug** — stdio mode was broken for programmatic use, but chat mode worked fine

---

## Round 2 Results (2026-03-28)

### Changes Made

1. **`prompt/prompt.go`**: Added explicit Chinese→priority mappings ("高优先级"/"紧急" → p0)
2. **`tools/tools.go`**: Removed `assign_task` tool (redundant with `update_task`)
3. **`tools/tools.go`**: `search_tasks` now also searches archived tasks
4. **`tools/tools.go`**: `show_blockers` scans all tasks with `blocked_by` regardless of status

### Before vs After

| Test | Before | After | Result |
|------|--------|-------|--------|
| "紧急" priority | Mapped to p1 | Maps to p0 | PASS |
| Assign without assign_task | N/A | Works via update_task | PASS |
| Search archived tasks | Only active | Both active + archive | PASS |
| Show blockers (in_progress + blocked_by) | "No blocked tasks" | Correctly shows relation | PASS |

### Commit

`157ef7e` — feat: tune priority mapping, remove redundant tool, fix search + blockers

### Key Learnings

- **Tool count reduction works** — removing `assign_task` (13→12 tools) didn't hurt behavior; the prompt guidance about "only set specified fields" made the convenience wrapper unnecessary
- **`show_blockers` was too narrow** — filtering by `status=blocked` missed tasks that had `blocked_by` relations but different status; scanning by field presence is more useful
- **Explicit Chinese mappings beat English-only definitions** — "critical/blocking/must fix immediately" didn't help the Chinese model translate "紧急"; explicit Chinese examples solved it

---

## Round 3 Results (2026-03-28)

### Changes Made

1. **`prompt/prompt.go`**: Added "Task References" section — instruct agent to resolve informal task names via search instead of asking for IDs

### Before vs After

| Test | Before | After | Result |
|------|--------|-------|--------|
| "B依赖A" (stateless) | Asked for task IDs | Still asks (expected — no session context) | N/A |
| "登录页面依赖用户认证" (session) | N/A | Correctly resolves both + adds dependency | PASS |
| Multi-assign in one msg | N/A | Assigns both correctly | PASS |
| Complete by name | N/A | Resolves name → ID → archives | PASS |

### Commit

`2d098ec` — feat: add task reference resolution instructions to prompt

### Key Learnings

- **Session context is critical** — task reference resolution only works when the agent has conversation history. Stateless stdio mode (separate processes) can't resolve references.
- **Multi-message sessions unlock powerful workflows** — piping multiple JSON messages into one `tingly-pm` process allows the agent to build context and handle complex multi-step operations
- **The prompt alone can't solve statelessness** — this is an architectural property, not a prompt issue
