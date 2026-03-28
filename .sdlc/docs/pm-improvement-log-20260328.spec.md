# tingly-pm Improvement Log

**Agent:** tingly-pm (AI project manager)
**Driver:** Claude Code
**Date:** 2026-03-28
**Method:** [Agent Iterative Improvement Methodology](agent-improvement-methodology-20260328.spec.md)

---

## Summary

4 rounds of iterative improvement. 14 tools → 12. Prompt rewritten from scratch.

| Round | Focus | Experiments | Pass Rate | Approach |
|-------|-------|-------------|-----------|----------|
| 1 | Fix fundamentals | 8 tests | 8/8 (100%) | Sequential |
| 2 | Priority, tools, search | 6 tests | 6/6 (100%) | Sequential |
| 3 | Contextual reasoning | 4 tests | 4/4 (100%) | Sequential |
| 4 | Output, timeline, consolidation | 3 parallel + integration | 3/3 + 1/1 | **Parallel** |

## Starting State

```
tools: 14 (create_task, update_task, get_task, list_tasks, archive_task,
       search_tasks, add_comment, register_member, list_members,
       assign_task, add_dependency, remove_dependency, show_blockers,
       generate_report, summary, save_session)
prompt: 27 lines, flat unstructured
stdio mode: broken (ANSI escape codes in JSON output)
```

## Ending State

```
tools: 12 (removed assign_task, show_blockers; added list_timeline)
prompt: ~60 lines, 6 structured sections
stdio mode: clean JSON
```

---

## Round 1 — Fix Fundamentals

**Commit:** `01802b9`

| # | Test | Before | After | Verdict |
|---|------|--------|-------|---------|
| 1 | Create task (Chinese) | Title English, ANSI leak | Title Chinese, clean JSON | PASS |
| 2 | Duplicate creation | Created 3rd duplicate | Detected existing, upgraded | PASS |
| 3 | Update + assign | Hallucinated 3 fields | Only changed assignee | PASS |
| 4 | List tasks | Plain | Better formatted | PASS |
| 5 | Archive | OK | OK | PASS |
| 6 | Error handling | N/A | Graceful with suggestion | PASS |
| 7 | Report generation | N/A | Good formatted | PASS |
| 8 | Language matching | Mixed | Matches input | PASS |

**Changes:**
- `main.go`: disable console output for run/serve modes
- `prompt/prompt.go`: rewrite with 6 structured sections

**Learnings:**
- Prompt structure > length. Sections with headers beat flat lists.
- `CRITICAL:` prefix stops persistent hallucination.

## Round 2 — Priority, Tool Consolidation, Search

**Commit:** `157ef7e`

| # | Test | Before | After | Verdict |
|---|------|--------|-------|---------|
| 1 | "紧急" priority | p1 | p0 | PASS |
| 2 | Assign via update_task | N/A | Works | PASS |
| 3 | Search archived | Active only | Active + archive | PASS |
| 4 | show_blockers logic | Status filter only | Field-presence scan | PASS |
| 5 | Status-only update | N/A | No contamination | PASS |
| 6 | Member ops | OK | OK | PASS |

**Changes:**
- Prompt: explicit Chinese→priority mappings
- Removed `assign_task` (14→13)
- `search_tasks`: scan archive dir
- `show_blockers`: scan by `blocked_by != []`

**Learnings:**
- Explicit Chinese mappings beat English definitions.
- Tool reduction is safe when prompt compensates.

## Round 3 — Contextual Reasoning

**Commit:** `2d098ec`

| # | Test | Before | After | Verdict |
|---|------|--------|-------|---------|
| 1 | "登录页面依赖用户认证" (session) | Asked for IDs | Resolved both + added dep | PASS |
| 2 | Multi-assign | N/A | Both assigned | PASS |
| 3 | Complete by name | N/A | Resolved → archived | PASS |
| 4 | Smart summary | N/A | Contextual (alice done, bob active) | PASS |

**Changes:**
- Prompt: "Task References" section — search before asking for ID

**Learnings:**
- Reference resolution requires session context (architectural, not prompt-fixable).
- Multi-message sessions unlock complex workflows.

## Round 4 — Parallel Batch

**Commits:** `ab5cc03`, `a1f14bc`

3 experiments launched in parallel via subagents.

| Exp | Change | Build | Tests | Result |
|-----|--------|-------|-------|--------|
| A | Merge `show_blockers` into `list_tasks` filter | PASS | PASS | PASS |
| B | Add `list_timeline` tool | PASS | PASS | PASS |
| C | `list_tasks` age + priority grouping | PASS | PASS | PASS |

Integration test (8-step multi-message session): all features combined — PASS.

**Changes:**
- `list_tasks`: group by priority headers, include human-readable age
- `list_tasks`: `show_blockers` filter (replaces standalone tool)
- Removed `show_blockers` (13→12)
- Added `list_timeline` + `board.ListEvents()`

**Learnings:**
- Parallel subagents: 3 experiments in ~2.5min vs ~7.5min sequential.
- File conflicts manageable when changes target different functions.
- Integration testing after merge is essential.

---

## Tool Evolution

| Version | Count | Tools |
|---------|-------|-------|
| v0 (initial) | 14 | create_task, update_task, get_task, list_tasks, archive_task, search_tasks, add_comment, register_member, list_members, **assign_task**, add_dependency, remove_dependency, **show_blockers**, generate_report, summary, save_session |
| v1 (round 2) | 13 | removed assign_task |
| v2 (round 4) | 12 | removed show_blockers, added **list_timeline** |
