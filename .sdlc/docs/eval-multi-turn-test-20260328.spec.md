# Eval Multi-Turn Test Support

**Category:** eval
**Feature:** Stream JSON test fixtures for multi-turn conversation testing
**Date:** 2026-03-28
**Status:** Draft

---

## Overview

The eval loop's test infrastructure currently uses ad-hoc `echo '...' | tingly-pm` for single-turn tests. While the agent's `run` mode already supports **stream JSON I/O** (newline-delimited JSON over stdin/stdout, one turn per line), the eval framework doesn't leverage this as a first-class testing primitive.

**This spec formalizes the stream JSON protocol as the standard test interface** and introduces **JSONL fixture files** as a persistent, replayable representation of stream JSON sessions.

### The Protocol

```
stdin  → {"content": "user message 1"}    \n  →  stdout: {"role": "assistant", "content": "reply 1"}    \n
stdin  → {"content": "user message 2"}    \n  →  stdout: {"role": "assistant", "content": "reply 2"}    \n
stdin  → {"content": "user message 3"}    \n  →  stdout: {"role": "assistant", "content": "reply 3"}    \n
...
```

Both sides are **line-delimited JSON (JSONL/NDJSON)**. Input and output are interleaved — the agent processes one request, responds, then reads the next. A file is just a captured stream you can replay via `cat file.jsonl | tingly-pm -mode run`.

---

## Current State

### What exists
- `runStdio` already implements stream JSON I/O (`bufio.Scanner` reads JSON lines from stdin, writes JSON lines to stdout)
- `printf '...\n...\n...' | tingly-pm -mode run` — ad-hoc multi-turn via piped stdin
- Baseline test suite uses an inline 11-step sequence embedded in playbook markdown

### What's missing
- **No persistent test fixtures** — tests are reconstructed from memory each round
- **No per-turn evaluation** — output is a blob, can't grade individual turns
- **No fixture discovery** — subagents don't know what test sessions exist
- **Pool items don't indicate turn count** — single vs multi-turn is implicit

---

## Design

### 1. Stream JSON as Test Interface

The stream JSON protocol is the foundation. A **test session** is a pair of JSONL streams:

```
input.jsonl  →  tingly-pm -mode run  →  output.jsonl
```

Both sides use the same format: one JSON object per line.

**Input stream** (what you send):
```jsonl
{"content": "创建任务A：用户认证，p0"}
{"content": "创建任务B：登录页面，p1"}
{"content": "登录页面依赖用户认证"}
```

**Output stream** (what you get back):
```jsonl
{"role": "assistant", "content": "已创建 TASK-20260328-100000: 用户认证 [p0]"}
{"role": "assistant", "content": "已创建 TASK-20260328-100001: 登录页面 [p1]"}
{"role": "assistant", "content": "已添加依赖：登录页面 blocked_by 用户认证"}
```

For testing, the **input stream** is the fixture. The output stream is captured and evaluated.

### 2. JSONL Fixture Files

A fixture file is simply a saved input stream. Placed in `.eval/fixtures/`:

**Naming**: `{category}-{description}.jsonl`

**Single-turn** — `.eval/fixtures/create-task-chinese.jsonl`:
```jsonl
{"content": "创建一个任务：修复用户登录超时问题，高优先级"}
```

**Multi-turn** — `.eval/fixtures/workflow-create-dep-archive.jsonl`:
```jsonl
{"content": "创建任务A：用户认证，p0"}
{"content": "创建任务B：登录页面，p1"}
{"content": "登录页面依赖用户认证"}
{"content": "用户认证完成了"}
{"content": "列出所有任务"}
```

One line = one turn in the stream. No special schema.

### 3. Capturing Output

Capture the output stream alongside for reproducibility:

```bash
cat .eval/fixtures/workflow-create-dep-archive.jsonl \
  | timeout 120 ./tingly-pm -mode run -dir /tmp/test-eval-r1-s1 -config .pm 2>/dev/null \
  | tee /tmp/test-eval-r1-s1.output.jsonl
```

The `.output.jsonl` is the recorded output stream — N lines in, N lines out (assuming no errors).

### 4. Expectation File (Optional)

An `.expect.md` file provides per-turn grading guidance for subagent evaluators:

**`.eval/fixtures/workflow-create-dep-archive.expect.md`**:
```markdown
# workflow-create-dep-archive

TURN 1: task created, p0 priority
TURN 2: task created, p1 priority
TURN 3: dependency added, resolves both tasks by name
TURN 4: task archived as done, resolves by name
TURN 5: list shows remaining tasks, archived task not in active list
```

Prose, not machine-parseable — subagents read it and grade each turn.

### 5. Fixture Index

`.eval/fixtures/INDEX.md` — the manifest that replaces the inline test feature pool:

```markdown
# Test Fixtures

## Single-Turn (1 message)

| Fixture | Turns | Category | Description |
|---------|-------|----------|-------------|
| create-task-chinese.jsonl | 1 | create | Chinese input, keyword priority |
| create-task-english.jsonl | 1 | create | English input, explicit priority |
| create-task-duplicate.jsonl | 1 | create | Duplicate detection |
| error-empty-input.jsonl | 1 | error | Empty content handling |
| ... | | | |

## Multi-Turn (2+ messages)

| Fixture | Turns | Category | Description |
|---------|-------|----------|-------------|
| context-resolve-by-name.jsonl | 3 | context | Create then reference by name |
| context-implicit-reference.jsonl | 4 | context | "this task"/"that task" resolution |
| workflow-create-dep-archive.jsonl | 5 | workflow | Full lifecycle |
| ... | | | |
```

### 6. Execution Pattern

Same command for single or multi-turn — the fixture file is the stream:

```bash
# Run any fixture
cat .eval/fixtures/{name}.jsonl | timeout {N} ./tingly-pm -mode run -dir /tmp/test-{id} -config .pm 2>/dev/null

# With output capture
cat .eval/fixtures/{name}.jsonl | timeout {N} ./tingly-pm -mode run -dir /tmp/test-{id} -config .pm 2>/dev/null | tee /tmp/test-{id}.output.jsonl
```

Timeout by turn count:
- 1 turn: `timeout 30`
- 2-3 turns: `timeout 60`
- 4-5 turns: `timeout 120`
- 6+ turns: `timeout 180`

### 7. Updated Subagent Prompt Template

```
=== TEST FIXTURES (Stream JSON) ===

Test fixtures are JSONL files in .eval/fixtures/.
Each line is one JSON message in the stream-json protocol.
The agent reads from stdin line by line, responds to stdout line by line.
Read .eval/fixtures/INDEX.md for available fixtures.

=== PROTOCOL ===

Input (stdin)  — one JSON per line:  {"content": "user message"}
Output (stdout) — one JSON per line: {"role": "assistant", "content": "reply"}

The streams are interleaved: agent processes one input, emits one output,
then reads the next input. Session state persists across the stream.

=== EXECUTION ===

cat .eval/fixtures/{name}.jsonl | timeout {N} ./tingly-pm -mode run -dir /tmp/test-{id} -config .pm 2>/dev/null | tee /tmp/test-{id}.output.jsonl

=== MULTI-TURN GRADING ===

Output is N JSON lines — one per input turn.
Grade EACH turn independently. Overall PASS = all turns pass.

If .expect.md exists: use its per-turn criteria.
Otherwise: grade by fixture description from INDEX.md.
Report: "X/N turns passed" for multi-turn fixtures.
```

### 8. Initial Fixture Set

**Single-turn (16 fixtures):**

| Fixture | Input |
|---------|-------|
| `create-task-chinese.jsonl` | 创建一个任务：修复用户登录超时问题，高优先级 |
| `create-task-english.jsonl` | Create a task: implement OAuth2, p1 |
| `create-task-duplicate.jsonl` | Create a task: implement OAuth2 |
| `create-task-priority-keyword.jsonl` | 创建一个紧急任务：API限流 |
| `create-task-assign.jsonl` | 创建任务：代码审查，分配给alice |
| `update-task-single-field.jsonl` | update {task_id} priority to p0 |
| `update-task-nonexistent.jsonl` | update TASK-NONEXIST-12345 status to in_progress |
| `list-tasks-empty.jsonl` | 列出所有任务 |
| `search-tasks-by-title.jsonl` | 搜索"认证" |
| `member-register-list.jsonl` | 注册成员alice，然后列出所有成员 |
| `error-empty-input.jsonl` | {"content": ""} |
| `error-invalid-taskid.jsonl` | 查看 TASK-NONEXIST-12345 |
| `language-english-input.jsonl` | List all tasks with p0 priority |
| `report-daily.jsonl` | 生成今日报告 |
| `timeline-recent.jsonl` | 查看最近活动 |
| `summary-stats.jsonl` | 项目概况 |

**Multi-turn (9 fixtures):**

| Fixture | Turns | Description |
|---------|-------|-------------|
| `context-resolve-by-name.jsonl` | 3 | Create "用户认证" → update "用户认证" status → list |
| `context-implicit-reference.jsonl` | 4 | Create two tasks → "第一个任务完成了" → list |
| `workflow-create-dep-archive.jsonl` | 5 | Create A, B → add dep → archive A → list |
| `workflow-create-assign-list.jsonl` | 4 | Create task → register member → assign → list |
| `context-cross-language.jsonl` | 3 | 创建中文任务 → English: "update the first task" → list |
| `context-error-recovery.jsonl` | 3 | Create task → invalid op → continue normal use |
| `workflow-dependency-add-remove.jsonl` | 4 | Create two → add dep → remove dep → verify |
| `workflow-create-comment-list.jsonl` | 3 | Create task → add comment → get detail |
| `workflow-register-assign-multi.jsonl` | 4 | Register 2 members → create 2 tasks → assign → list |

### 9. Discovery & Mutation via Fixtures

**Mutation** — copy existing fixture, modify stream lines:

```bash
# Mutate: add error mid-stream
cp .eval/fixtures/workflow-create-dep-archive.jsonl \
   .eval/fixtures/mutated-dep-archive-with-error.jsonl
# Insert an invalid turn between turns 2 and 3
```

**Discovery** — write new stream from scratch:

```bash
cat > .eval/fixtures/discover-conflicting-update.jsonl << 'EOF'
{"content": "创建任务：数据迁移，p0"}
{"content": "把数据迁移优先级改为p2"}
{"content": "把数据迁移优先级改为p0"}
{"content": "查看数据迁移"}
EOF
```

### 10. Selection & Weighting

```
Per subagent budget:
  - Single-turn fixtures: 3-5
  - Multi-turn fixtures: 1-2
  - Mutation/Discovery: 0-2 new fixtures created
  - Total turns cap: ~20 turns per subagent

Selection mode:
  - POOL_SAMPLE (60%): pick random fixtures from INDEX.md
  - MUTATE (30%): copy + modify an existing fixture's stream
  - DISCOVER (10%): create a new fixture stream from scratch
```

---

## Fixture Lifecycle

```
Round 0 (manual):
  Write initial fixture set → .eval/fixtures/*.jsonl + INDEX.md

Round N (automated, via subagents):
  Read INDEX.md → select fixtures → stream them to agent → capture output → grade
  MUTATE/DISCOVER → write new .jsonl → update INDEX.md
  Part 2 verification: reshuffled fixture selection from INDEX.md
  New fixtures committed as eval artifacts
  Pool grows organically
```

Fixtures are first-class eval artifacts in `.eval/`, versioned in git.

---

## Protocol Diagram

```
┌─────────────────────────────────────────────────┐
│  .eval/fixtures/{name}.jsonl                    │
│  ┌───────────────────────────────────────────┐  │
│  │ {"content": "msg 1"}                     │  │
│  │ {"content": "msg 2"}                     │  │  Input stream
│  │ {"content": "msg 3"}                     │  │  (JSONL)
│  └──────────────┬────────────────────────────┘  │
└─────────────────┼───────────────────────────────┘
                  │ cat | pipe
                  ▼
┌─────────────────────────────────────────────────┐
│  tingly-pm -mode run                            │
│                                                 │
│  ┌─────────────┐    ┌──────────────┐            │
│  │  Stdin       │───→│  Agent REPL  │──→ stdout  │  Output stream
│  │  (JSONL)     │    │  (stateful)  │    (JSONL)  │  (JSONL)
│  └─────────────┘    └──────────────┘            │
└─────────────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│  output.jsonl (captured)                        │
│  ┌───────────────────────────────────────────┐  │
│  │ {"role": "assistant", "content": "..."}   │  │
│  │ {"role": "assistant", "content": "..."}   │  │
│  │ {"role": "assistant", "content": "..."}   │  │
│  └───────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│  .eval/fixtures/{name}.expect.md (optional)     │
│  Per-turn grading criteria for subagent          │
└─────────────────────────────────────────────────┘
```

---

## Implementation Plan

### Phase 1: Create fixture files
1. Create `.eval/fixtures/` directory
2. Write 16 single-turn `.jsonl` files
3. Write 9 multi-turn `.jsonl` files
4. Write `.expect.md` for multi-turn fixtures
5. Write `INDEX.md`

### Phase 2: Update eval-loop.md
1. Replace inline test pool with fixture reference
2. Update subagent prompt with stream JSON protocol + fixture execution
3. Update mutation/discovery to operate on fixture files
4. Update report format

### Phase 3: Update methodology spec
1. Document stream JSON as standard test execution mode
2. Update mutation rules (copy + edit stream)
3. Update discovery protocol (create new stream + update index)

### Phase 4: Update playbook
1. Point baseline to fixture files
2. Update execution reference with `cat fixture.jsonl | ...`
3. Remove inline test sequences

---

## Files Changed

| File | Change |
|------|--------|
| `.eval/fixtures/*.jsonl` | **New** — 25 initial test fixtures |
| `.eval/fixtures/*.expect.md` | **New** — expectations for multi-turn fixtures |
| `.eval/fixtures/INDEX.md` | **New** — fixture manifest |
| `eval-loop.md` | Replace pool with fixture refs, update subagent template |
| `.sdlc/docs/agent-improvement-methodology-20260328.spec.md` | Stream JSON docs, fixture-based mutation/discovery |
| `.sdlc/docs/pm-improvement-playbook-20260328.spec.md` | Point baseline to fixtures |

**No Go code changes.** The stream JSON protocol already exists in `runStdio`.

---

## Open Questions

1. **`.expect.md` prose vs structured?** Proposing prose — LLM subagents read/write it naturally.
2. **Fixture commit strategy?** Propose: commit with round artifacts (same commit if code changes pass, separate if discover-only).
3. **Max turns per fixture?** Propose cap at 8. Longer sequences → split into sub-fixtures.
