# tingly-pm — Product Design & Technical Specification

## Overview

**tingly-pm** is an AI project manager agent. It serves as a project secretary — recording tasks, tracking progress, managing team assignments, maintaining timelines, and generating reports. It is **not** a decision-maker.

**Key traits**: NL-first, file-based storage, single project, small teams of humans + agents.

## Architecture

```
                  ┌── HTTP Server ──┐
  Human/Agent ───►│   tingly-pm     │───► .pm/ (file system, independent git repo)
  (chat mode) ───►│  (ReAct Agent)  │
                  └── Stdio JSON ───┘
                         │
                    pkg/agent.ReActAgent
                    pkg/model/anthropic
                    pkg/tool.Registry
```

## Design Decisions Summary

| Dimension | Decision |
|-----------|----------|
| **Core Role** | Background service / AI project secretary |
| **Interaction** | NL-first bot, three modes: chat / stdio / HTTP |
| **Storage** | File + directory structured in `.pm/` |
| **Task ID** | Timestamp-based `TASK-YYYYMMDD-HHmmss` |
| **File Naming** | `{priority}-{id}-{slug}.md` — meaningful at a glance |
| **Status Lifecycle** | Active states stay in `tasks/`, terminal states archive to `archive/YYYYMM/` |
| **Team Members** | Typed: `human` vs `agent` |
| **Reporting** | PM provides tool, externally triggered |
| **Init** | Auto-create on first use, no explicit init |
| **Git** | `.pm/` is an independent git repo inside project dir |
| **Config** | `-config <dir>` to read `config.json` for model settings |
| **Model** | Anthropic Claude only |

---

## Three Interaction Modes

### 1. Chat Mode (default)

Interactive REPL for human developers:

```bash
tingly-pm -dir /path/to/project -config /path/to/config/dir

tingly-pm — AI Project Manager
Type your message and press Enter. /quit to exit.

> 创建一个高优任务：实现JWT令牌刷新，分配给 agent-1

Created TASK-20260327-163000: 实现JWT令牌刷新 [p0]

> 现在项目什么情况？

Total active: 1 tasks
  todo: 1
Active members: agent-1

> /quit
Bye.
```

### 2. Stdio JSON Stream (`-mode run`)

For agent/programmatic access:

```jsonl
{"role": "user", "content": "创建任务：用户认证"}
{"role": "assistant", "content": "Created TASK-20260327-143022: 用户认证 [p1]"}
```

### 3. HTTP Server (`-mode serve`)

For remote access:

```
POST /message  {"content": "..."}  →  {"content": "..."}
GET  /health                       →  {"status": "ok"}
```

All three modes wrap the same `ReActAgent.Reply()` call.

---

## Configuration

### CLI Flags

```bash
tingly-pm \
  -mode chat \                        # chat | run | serve
  -dir /path/to/project \             # project directory
  -config /path/to/config/dir \       # config directory
  -addr :8080 \                       # HTTP listen address (serve mode)
```

### config.json

Located at `<config-dir>/config.json`. Provides model connection settings:

```json
{
  "base_url": "https://api.tingly.dev/tingly/claude_code",
  "api_key": "sk-xxx",
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 8192
}
```

Priority: config.json `api_key` > `ANTHROPIC_API_KEY` env var. Without `-config`, only env var is read.

---

## Storage: .pm/ Directory

`.pm/` lives inside the project directory as an **independent git repo**. The parent project's git naturally ignores nested `.git/` directories — no `.gitignore` entry needed.

Auto-created on first use.

### Directory Structure

```
.pm/
├── .git/                                          # Independent git repo
├── tasks/                                         # Active tasks only
│   ├── p0-TASK-20260327-143022-jwt-token-refresh.md
│   ├── p1-TASK-20260328-100000-email-notification.md
│   └── p2-TASK-20260326-091500-code-review-tool.md
├── archive/                                       # Terminal tasks (done/dropped)
│   ├── 202603/
│   │   ├── p0-TASK-20260320-110000-db-migration.md
│   │   └── p1-TASK-20260315-140000-old-feature.md
│   └── 202604/
├── members.json                                   # Team roster (typed: human/agent)
├── timeline.jsonl                                 # Append-only event log
└── reports/                                       # Generated reports
    └── 20260327-daily.md
```

### Design Rationale

- **`tasks/` only contains active tasks** — `ls` is always meaningful, never cluttered
- **`archive/YYYYMM/`** — monthly archives for done/dropped tasks
- **Independent git repo** — PM data decoupled from project code, no pollution of project commit history
- **File naming with priority prefix** — `ls`一眼看出该关注什么（code-naming style: meaningful at a glance）

---

## Task ID

Format: `TASK-YYYYMMDD-HHmmss`

```
TASK-20260327-143022
     ├── date ──┘└── time
```

```go
id := fmt.Sprintf("TASK-%s", time.Now().Format("20060102-150405"))
```

- Naturally unique (no counter, no lock, no scan) — simplest approach
- Human-readable with embedded creation time
- Used for all inter-task references (decoupled from filename)

---

## Task File

### Filename Convention

Format: `{priority}-{id}-{slug}.md`

| Part | Value | Example |
|------|-------|---------|
| **priority** | `p0` \| `p1` \| `p2` \| `p3` (lower = higher) | `p0` |
| **id** | `TASK-YYYYMMDD-HHmmss` | `TASK-20260327-143022` |
| **slug** | LLM-generated kebab-case English ≤50 chars | `jwt-token-refresh` |

Example: `p0-TASK-20260327-143022-jwt-token-refresh.md`

**Slug generation**: When user creates a task with a Chinese title (e.g., "实现JWT令牌刷新机制"), the LLM translates and generates an English kebab-case slug (`jwt-token-refresh`). This makes the filename meaningful at a glance, like good code variable names.

### Content Format

YAML frontmatter + Markdown body:

```markdown
---
id: TASK-20260327-143022
title: 实现JWT令牌刷新机制
slug: jwt-token-refresh
status: in_progress
priority: p0
assignee: agent-1
created: 2026-03-27T14:30:22Z
updated: 2026-03-27T16:00:00Z
closed_at:
labels: [auth, backend]
blocks: []
blocked_by: [TASK-20260325-090000]
---

## Description

实现基于 JWT 的令牌刷新机制，支持 refresh token rotation。

## Activity

- [2026-03-27 14:30] Created by @yz
- [2026-03-27 14:35] Assigned to agent-1
- [2026-03-27 16:00] Status: todo → in_progress by agent-1
```

**Source of truth**: Frontmatter. Filename is a projection for human browsing.

### Frontmatter Schema

```go
type Task struct {
    ID        string   `json:"id" yaml:"id"`
    Title     string   `json:"title" yaml:"title"`
    Slug      string   `json:"slug" yaml:"slug"`
    Status    string   `json:"status" yaml:"status"`       // todo|in_progress|blocked|review|done|dropped
    Priority  string   `json:"priority" yaml:"priority"`   // p0|p1|p2|p3
    Assignee  string   `json:"assignee" yaml:"assignee"`   // member name or empty
    Created   string   `json:"created" yaml:"created"`     // RFC3339
    Updated   string   `json:"updated" yaml:"updated"`     // RFC3339
    ClosedAt  string   `json:"closed_at,omitempty" yaml:"closed_at,omitempty"`
    Labels    []string `json:"labels" yaml:"labels"`
    Blocks    []string `json:"blocks" yaml:"blocks"`           // task IDs this task blocks
    BlockedBy []string `json:"blocked_by" yaml:"blocked_by"`   // task IDs blocking this task
    Body      string   `json:"-" yaml:"-"`
    FilePath  string   `json:"-" yaml:"-"`
}
```

### File Rename on Change

- **Priority change**: `p1-TASK-...-foo.md` → `p0-TASK-...-foo.md`
- **Title change**: LLM regenerates slug, file renamed
- **Inter-task references use ID only** — renames are safe, no cascade updates needed

---

## Status Model

### State Machine

```
todo ──► in_progress ──► review ──► done ──────► archive/YYYYMM/
  │          │              │
  │          ▼              │
  │       blocked ──────────┘
  │
  └──► dropped ─────────────────────────────────► archive/YYYYMM/
```

### Active States (stay in `tasks/`)

| Status | Meaning |
|--------|---------|
| `todo` | Not started |
| `in_progress` | Being worked on |
| `blocked` | Waiting on dependency |
| `review` | Awaiting review/confirmation |

### Terminal States (moved to `archive/YYYYMM/`)

| Status | Meaning |
|--------|---------|
| `done` | Completed |
| `dropped` | Abandoned |

When a task reaches terminal state, PM:
1. Updates frontmatter (`status`, `updated`, `closed_at`)
2. Moves file to `archive/YYYYMM/` (based on close date)
3. Appends event to `timeline.jsonl`

---

## Team Members

`members.json` — typed roster of humans and agents:

```json
{
  "members": [
    {"name": "yz", "type": "human", "labels": ["backend", "infra"]},
    {"name": "agent-1", "type": "agent", "labels": ["coding", "golang"]}
  ]
}
```

PM can distinguish behavior between humans and agents (e.g., agents may auto-claim tasks in future).

---

## Timeline

`timeline.jsonl` — append-only event log:

```jsonl
{"ts":"2026-03-27T14:30:22Z","event":"task_created","task":"TASK-20260327-143022","by":"yz"}
{"ts":"2026-03-27T14:35:00Z","event":"task_assigned","task":"TASK-20260327-143022","assignee":"agent-1","by":"yz"}
{"ts":"2026-03-27T16:00:00Z","event":"status_changed","task":"TASK-20260327-143022","from":"todo","to":"in_progress","by":"agent-1"}
{"ts":"2026-03-28T10:00:00Z","event":"task_archived","task":"TASK-20260327-143022","status":"done","by":"agent-1"}
```

### Event Types

| Event | Fields |
|-------|--------|
| `task_created` | task, by |
| `task_assigned` | task, assignee, by |
| `status_changed` | task, from, to, by |
| `priority_changed` | task, from, to, by |
| `comment_added` | task, by, content |
| `task_archived` | task, status, by |
| `member_registered` | name, type, by |
| `report_generated` | type, path, by |

---

## PM Agent Tools

The PM agent (ReActAgent) has 14 internal tools:

### Task Management

| Tool | Args | Description |
|------|------|-------------|
| `create_task` | title, slug, priority?, assignee?, labels?, description? | Create task |
| `update_task` | task_id, status?, priority?, assignee?, labels?, title?, slug? | Update fields, rename file if needed |
| `get_task` | task_id | Read task detail |
| `list_tasks` | status?, assignee?, priority?, label? | List/filter active tasks |
| `archive_task` | task_id, resolution (done/dropped) | Move to archive |
| `search_tasks` | query | Full-text search |

### Collaboration

| Tool | Args | Description |
|------|------|-------------|
| `add_comment` | task_id, content, by? | Append comment + timeline |
| `register_member` | name, type, labels? | Add team member |
| `list_members` | type? | List members |
| `assign_task` | task_id, assignee | Update assignee |

### Relations

| Tool | Args | Description |
|------|------|-------------|
| `add_dependency` | task_id, depends_on | Add blocked_by relation |
| `remove_dependency` | task_id, depends_on | Remove relation |
| `show_blockers` | task_id? | Show all blocked tasks or specific |

### Reporting

| Tool | Args | Description |
|------|------|-------------|
| `generate_report` | report_type (daily/weekly) | Generate + save report |
| `summary` | — | Quick status stats |

---

## Interaction Examples

```
> 创建一个高优任务：实现JWT令牌刷新机制，分配给 agent-1
PM: ✓ Created TASK-20260327-143022: 实现JWT令牌刷新机制 [p0] → agent-1

> 现在项目什么情况？
PM: Total active: 15 tasks
    todo: 4 | in_progress: 6 | blocked: 2 | review: 3
    Blocked: TASK-001 等待 TASK-003 完成
    Active members: yz, agent-1, agent-2

> 出一份今天的日报
PM: ## Daily Report - 2026-03-27
    [saved to .pm/reports/20260327-daily.md]

> TASK-003 完成了，PR #45 已合并
PM: ✓ TASK-003 → done, recorded completion
```

---

## Project Structure (Implementation)

```
tingly-pm/
├── go.mod                   # module: github.com/FFengIll/tingly-pm
├── main.go                  # Entry: parse flags, select mode, create agent
├── board/                   # Core task board logic
│   ├── board.go             # EnsureInit: create .pm/ dirs + git init
│   ├── task.go              # Task CRUD, status management
│   ├── task_file.go         # Frontmatter parse/format, filename generation
│   ├── member.go            # Members CRUD
│   ├── timeline.go          # Append/read event log
│   └── report.go            # Summary + report generation
├── tools/
│   └── tools.go             # 14 PM agent tools (PMTools struct)
├── prompt/
│   └── prompt.go            # System prompt for ReActAgent
└── board/
    └── board_test.go        # 16 tests
```

### Key Dependencies

- `pkg/agent.ReActAgent` — Agent loop (LLM + tools)
- `pkg/model/anthropic` — Anthropic Claude model
- `pkg/tool.Registry` — Tool registration + execution
- `pkg/memory.History` — Conversation history

---

## Future Considerations

- **Webhook notifications**: Push task events to Slack/Discord
- **Multiple projects**: PM managing multiple .pm/ directories
- **Web UI**: Dashboard reading from .pm/ files
- **Agent auto-assignment**: Match member labels to task labels
- **Config profile support**: Multiple named configs per project
