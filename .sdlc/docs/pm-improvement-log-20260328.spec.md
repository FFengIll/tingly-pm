# tingly-pm Improvement Log

**Agent:** tingly-pm (AI project manager)
**Driver:** Claude Code
**Date:** 2026-03-28
**Method:** [Agent Iterative Improvement Methodology](agent-improvement-methodology-20260328.spec.md)

---

## Summary

5 rounds of iterative improvement. 14 tools → 12. Prompt rewritten from scratch twice.

| Round | Focus | Experiments | Pass Rate | Approach |
|-------|-------|-------------|-----------|----------|
| 1 | Fix fundamentals | 8 tests | 8/8 (100%) | Sequential |
| 2 | Priority, tools, search | 6 tests | 6/6 (100%) | Sequential |
| 3 | Contextual reasoning | 4 tests | 4/4 (100%) | Sequential |
| 4 | Output, timeline, consolidation | 3 parallel + integration | 3/3 + 1/1 | **Parallel** |
| 5 | Context resolution, fuzzy dupes | 2 parallel + verify | 2/2 + verified | **v2 Parallel** |

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
prompt: ~73 lines, 8 structured sections (added Context Resolution)
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

---

## Round 5 — v2 Parallel Fuzzing (First Loop)

**Commit:** `8629fe9`

**Method:** v2 two-part loop with parallel fuzzing evaluation

**Part 1 — Baseline:**
- Launched 4 parallel subagents with randomized test selection
- Baseline: 19.5/20 passed (97.5%)
- Identified gaps: implicit reference resolution, fuzzy duplicate detection

**Part 1 — Experiments:**

| Exp | Change | Build | Tests | Result |
|-----|--------|-------|-------|--------|
| A | Add "Context Resolution" section | PASS | PASS | PASS - Resolves "this task", "that task", "那个任务" |
| B | Enhanced fuzzy duplicate detection | PASS | PASS | PASS - Detects "fix login bug" vs "fix the login page bug" |

**Part 2 — Verification:**
- Launched 3 fresh subagents with reshuffled tests
- Verified new features work without regressions
- Result: Context resolution PASS, duplicate detection PASS (conservative but functional)
- No regressions in existing functionality

**Changes:**
- `prompt/prompt.go`: Added 8th section "Context Resolution"
- `prompt/prompt.go`: Enhanced Task Creation with keyword-based fuzzy matching
- Prompt: 6 sections → 8 sections (~60 lines → ~73 lines)

**Learnings:**
- v2 parallel fuzzing is faster and catches more edge cases than sequential
- Context resolution enables natural workflows like "create task A. assign THIS to bob"
- Fuzzy duplicate detection works for same-language similar phrasings
- Cross-language duplicate detection remains a future enhancement
- Two-part verification loop prevents confirmation bias

---

## Round 6 — New Loop Round 1 (Personnel Focus)

**Commit:** `0f91c42`

**Method:** v2 two-part loop with parallel fuzzing
**Exploration Seed:** "留意人员配置" (Pay attention to personnel configuration)

**Part 1 — Baseline:**
- Launched 4 parallel subagents with member/personnel focus
- Baseline: 21/23 passed (91%)
- Identified gaps: auto-registration ambiguity, label edge cases, reassignment workflow

**Part 1 — Experiments:**

| Exp | Change | Build | Tests | Result |
|-----|--------|-------|-------|--------|
| A | Add "Member Assignment" validation | PASS | 3/3 PASS | PASS - Asks before auto-registering members |
| B | Add "Member Labels" validation | PASS | 4/4 PASS | PASS - Handles special chars, empty, cross-language |
| C | Add "Task Reassignment" workflow | PASS | 10/10 PASS | PASS - "from X to Y" confirmation |

**Part 2 — Verification:**
- Launched 3 fresh subagents with reshuffled tests
- Verified: 21.5/22 passed (98%)
- No regressions detected
- Cross-language member support verified

**Changes:**
- `prompt/prompt.go`: Added "## Member Assignment" section (prevents silent auto-registration)
- `prompt/prompt.go`: Added "## Member Labels" section (handles edge cases)
- `prompt/prompt.go`: Added "## Task Reassignment" section (fills coverage gap)
- Prompt: 8 sections → 11 sections (~73 lines → ~98 lines)

**New Test Fixtures:**
- member-error-scenarios.jsonl
- member-labels-types.jsonl
- mutated-assign-nonexistent-member.jsonl
- mutated-assign-empty-members.jsonl
- mutated-cross-language-members.jsonl
- verify-special-chars-label.jsonl
- verify-overflow-title.jsonl

**Learnings:**
- Exploration seed successfully directed testing toward personnel configuration
- Auto-registration was a silent bug causing typo risk - now fixed
- Cross-language member names (张三) and labels (前端) work correctly
- Member label edge cases (special chars, empty, cross-language) now handled
- Pass rate improved: 91% → 98% (+7 percentage points)

---

## Round 7 — New Loop Round 2 (Personnel CRUD)

**Commit:** [pending]

**Method:** v2 two-part loop with parallel fuzzing
**Exploration Seed:** "留意人员配置" (Pay attention to personnel configuration)

**Part 1 — Baseline:**
- Launched 4 parallel subagents with member/personnel focus
- Baseline: 17/17 passed (100%)
- Identified gaps: No member search, update, or removal functionality

**Part 1 — Experiments:**

| Exp | Change | Build | Tests | Result |
|-----|--------|-------|-------|--------|
| A | Add search_members tool | PASS | PASS | PASS - Fuzzy label matching |
| B | Add update_member tool | PASS | PASS | PASS - Update type and labels |
| C | Add remove_member tool | PASS | PASS | PASS - Remove from registry |

**Part 2 — Verification:**
- Launched 3 fresh subagents with reshuffled tests
- Verified: 21/22 passed (95%)
- No functional regressions (one test setup issue)
- All new features working correctly

**Changes:**
- `board/member.go`: Added SearchMembers, UpdateMember, RemoveMember functions
- `tools/tools.go`: Added search_members, update_member, remove_member tools
- Total tools: 12 → 15

**New Test Fixtures:**
- mutated-member-typos.jsonl
- mutated-empty-member-name.jsonl
- mutated-member-special-chars.jsonl
- mutated-member-label-special-chars.jsonl
- discovered-conflicting-member-names.jsonl
- discovered-member-missing-fields.jsonl
- discovered-member-label-edge-cases.jsonl
- discovered-member-type-validation.jsonl
- discovered-member-case-sensitivity.jsonl

**Learnings:**
- Complete CRUD operations for members significantly improve usability
- Fuzzy label matching (case-insensitive substring) works well
- Cross-language support verified (Chinese names and labels)
- Edge case handling is good (helpful errors, empty filters)
- Integration with existing workflows is seamless
