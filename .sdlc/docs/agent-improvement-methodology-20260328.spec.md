# Agent Iterative Improvement Methodology

**Type:** Reusable framework
**Scope:** Any ReAct-based agent (Go, Claude Code as driver)
**Extracted from:** tingly-pm improvement loop (4 rounds, 2026-03-28)

---

## Core Idea

Treat an agent (prompt + tools + config) as a trainable subject. An external driver (Claude Code) iteratively modifies the agent, executes it, observes output, evaluates behavior, and commits or reverts — analogous to model training at the agent level.

```
Modify → Build → Execute → Observe → Evaluate → Commit or Revert → Repeat
```

## Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                    Claude Code (the driver)                    │
│                                                                 │
│  Per Round:                                                     │
│  ┌──────────────────────────────────────────────────────┐      │
│  │ 1. Baseline: run existing agent, record problems     │      │
│  │ 2. Hypothesize: N improvements based on problems     │      │
│  │ 3. Launch N subagents in parallel                    │      │
│  │    Each: edit → build → test → execute → record      │      │
│  │ 4. Collect results, evaluate all together            │      │
│  │ 5. Apply winners, commit                             │      │
│  │ 6. Record learnings → feed into next round           │      │
│  └──────────────────────────────────────────────────────┘      │
│           ▲                                                      │
│           └──────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Baseline

Before any changes, run the agent with a comprehensive test suite to establish ground truth.

### Test Categories

| Category | What to Test | Example |
|----------|-------------|---------|
| Core operations | Every tool individually | create, update, archive, search |
| Edge cases | Error inputs, empty inputs, invalid IDs | `{"content":""}`, nonexistent task ID |
| Language | Input in different languages | Chinese, English, mixed |
| Contextual | Multi-turn conversations | "create A", "B depends on A", "A is done" |
| Output quality | Response formatting, conciseness | Does it group? Show key info? |
| Hallucination | Fabricated fields, invented data | Update task → only changes specified fields |

### Execution Modes

| Mode | Command | What it Tests |
|------|---------|---------------|
| Single request | `echo '...' \| agent -mode run` | Raw tool correctness (stateless) |
| Multi-message | `printf '...\n...\n...' \| agent -mode run` | Contextual reasoning (stateful session) |

**Key distinction**: Single-request mode isolates tool behavior. Multi-message mode tests whether the agent can reason across turns (reference resolution, multi-step workflows).

### Isolation

Every test needs a clean state directory:

```bash
mkdir -p /tmp/test-agent-rN    # N = round number
```

---

## Phase 2: Hypothesize

From the baseline, identify problems and form hypotheses.

### Problem Severity Scale

| Severity | Meaning | Action |
|----------|---------|--------|
| Critical | Agent produces wrong data or crashes | Fix immediately |
| High | Agent behaves incorrectly but doesn't crash | Fix in this round |
| Medium | Agent could be better but works | Fix if easy |
| Low | Nice to have | Queue for future |

### Improvement Targets

Ranked by impact:

1. **System prompt** — highest impact, zero code risk. Changes affect all behavior immediately.
2. **Tool behavior** — medium impact, requires code changes. Fix bugs, improve output, remove redundancy.
3. **Tool set** — structural impact. Add missing tools, remove redundant ones, merge overlapping ones.
4. **Agent configuration** — low impact. MaxIterations, memory size, model params.

### Hypothesis Format

For each experiment, state clearly:

```
Exp [ID]: [Title]
Hypothesis: [What change will fix what problem]
Target: [prompt.go / tools.go / main.go]
Expected: [What the output should look like after]
```

---

## Phase 3: Experiment (Parallel)

Launch N subagents in parallel, one per hypothesis.

### Subagent Protocol

Each subagent receives:

```
1. What file(s) to edit
2. What change to make
3. How to verify: go build + go test
4. How to execute: test input via stdio
5. Report format: structured pass/fail/observation
```

### Subagent Isolation

- Each subagent works in the same directory (shared filesystem)
- Changes are NOT committed by subagents
- Main context collects results and decides

> **Note**: Git worktree isolation is ideal but adds complexity. For non-conflicting file changes (different files), shared directory works fine. For conflicting changes (same file), serialize or merge manually.

### Subagent Report Format

```
EXPERIMENT: [ID] - [Title]
BUILD: pass/fail
TESTS: pass/fail
RESULT: pass/fail
OUTPUT: <actual JSON output from test>
OBSERVATIONS: <what you noticed>
```

### What to Watch For

- **False positives**: Test passes but agent used wrong tool or fabricated data
- **Invisible regressions**: New behavior breaks something not in the test suite
- **Over-fitting**: Prompt tuned for specific test inputs but fails on variations

---

## Phase 4: Evaluate

Collect all reports. Compare. Decide.

### Decision Matrix

| Build | Tests | Behavior | Decision |
|-------|-------|----------|----------|
| fail | — | — | Reject immediately |
| pass | fail | — | Reject (broken existing tests) |
| pass | pass | wrong | Reject (behavior regression) |
| pass | pass | correct | Accept |

### Patterns to Look For

| Pattern | Meaning | Action |
|---------|---------|--------|
| All experiments pass | Changes are independent | Commit all |
| Some fail, some pass | Incompatible changes or bad hypotheses | Commit winners, analyze failures |
| All fail | Fundamental misunderstanding | Rethink approach, don't force |
| "Works but fragile" | Passes specific test but feels brittle | Commit with TODO for hardening |

### Integration Test

After applying all winning changes, run a comprehensive multi-message session that exercises all features together. Individual passes don't guarantee combined correctness.

---

## Phase 5: Apply

### Commit Strategy

```
- One commit per logical change group (not per experiment)
- Commit message: what improved and why
- Example: "feat: improve task dedup via search-before-create prompt"
```

### Revert Strategy

If integration test fails:

1. `git diff HEAD~1` — review what changed
2. Identify the breaking change
3. Either fix it or `git revert <commit>`
4. Re-run integration test

### Record Learnings

After each round, answer:

```
- What worked that I didn't expect?
- What didn't work that I expected to?
- What pattern is worth repeating?
- What assumption was wrong?
```

These feed into the next round's hypotheses.

---

## Prompt Engineering Patterns

From 4 rounds of prompt improvement:

### Pattern 1: Structured Sections > Flat List

Organize prompt into titled sections with `##` headers. The LLM uses section headers as attention anchors.

```markdown
## Task Creation
...
## Task Updates
...
## Response Style
...
```

### Pattern 2: CRITICAL Prefix for Hard Rules

When the model consistently violates a rule, prefix with `CRITICAL:` and give explicit examples of what NOT to do.

```markdown
CRITICAL: When using update_task, ONLY include fields the user explicitly asked to change.
- If user says "assign to alice", only set assignee. Do NOT change title, slug, priority.
- Never fabricate values for unspecified fields.
```

### Pattern 3: Explicit Language Mappings

If the model operates in multiple languages, provide explicit keyword mappings. English-only definitions don't transfer reliably.

```markdown
- p0: Critical — "高优先级", "紧急", "critical", "urgent"
```

### Pattern 4: Behavioral Instructions Over Constraints

Instead of "don't do X", prefer "do Y instead". Positive instructions are more effective.

```markdown
# Bad
Don't ask the user for task IDs.

# Good
When the user refers to a task by name, use search_tasks to find it first.
```

### Pattern 5: Context-Dependent Behavior

Acknowledge that some behaviors depend on context (session state). Document limitations explicitly so you don't chase unsolvable problems.

```markdown
Note: Task reference resolution requires session context.
Single-request (stateless) mode cannot resolve informal names.
```

---

## Tool Design Patterns

From 4 rounds of tool improvement:

### Pattern 1: Fewer Tools > More Convenience Wrappers

If tool B is just `tool_A(field=X)`, remove B. A well-written prompt that says "only set specified fields" makes convenience wrappers unnecessary.

### Pattern 2: Merge Query + Filter

Don't create separate tools for every filter combination. One tool with optional filter fields:

```go
type ListArgs struct {
    Status       string `json:"status"`
    Assignee     string `json:"assignee"`
    ShowBlockers bool   `json:"show_blockers"`   // composite filter
}
```

### Pattern 3: Read Before Write

If the agent can write data (append events, create tasks), it must also be able to read it back (list timeline, search tasks). Write-only tools create information asymmetry.

### Pattern 4: Output Formatting Lives in the Tool

Don't rely on the LLM to format tool output. The tool should return structured, scannable text. The LLM's job is to interpret and summarize, not to parse raw data.

### Pattern 5: Human-Readable Derivatives

Add computed fields to tool output that the raw data doesn't have: task age, priority grouping headers, status summaries. These make the output immediately useful without LLM processing.

---

## Anti-Patterns

### Don't: Tune Prompt for Single Test Inputs

If you only test "创建一个任务：修复登录", the agent learns to handle that exact input. Test variations: different priorities, missing fields, Chinese/English, duplicates.

### Don't: Change Multiple Things at Once

If you change the prompt AND add a tool AND modify config, you can't attribute the effect. One logical change per experiment (but parallel experiments can touch different things).

### Don't: Trust Single-Request Results for Contextual Features

Stateless execution can't test session-dependent behavior. Always use multi-message sessions for reference resolution, multi-step workflows, and context-aware responses.

### Don't: Keep Redundant Tools "Just in Case"

Every tool adds schema complexity that the LLM must process. More tools = more confusion = higher chance of wrong tool selection.

### Don't: Skip the Integration Test

Parallel experiments pass individually but may conflict when combined. Always run a full integration test after merging.

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
5. **Revert immediately** — if tests break, revert, don't patch forward
6. **Commit with intent** — message explains what behavior improved
