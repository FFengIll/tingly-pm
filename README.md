# tingly-pm

> **Self-evolving** — this agent iteratively improves its own prompt, tools, and behavior through usage.

AI project manager that runs a file-based task board. Built with Go, [Anthropic Claude](https://docs.anthropic.com/) and [tingly-agentscope](https://github.com/tingly-dev/tingly-agentscope).

## Features

- **Task Management** — Create, update, search, and archive tasks with priority levels (p0–p3), status tracking, and labels
- **Team Members** — Register human and agent team members with capability labels
- **Dependency Tracking** — Block/unblock relationships between tasks
- **Timeline** — Append-only event log recording all board activity
- **Reports** — Daily and weekly progress reports with activity summaries
- **Session Persistence** — Conversations auto-save and restore across runs
- **Bilingual** — Supports both Chinese and English interactions
- **Multiple Modes** — Interactive chat, JSON stdio, and HTTP server

## Quick Start

### Prerequisites

- Go 1.25+
- An Anthropic API key

### Install

```bash
go install github.com/FFengIll/tingly-pm@latest
```

### Run

```bash
# Interactive chat mode (default)
tingly-pm -dir ./my-project

# JSON stdio mode (for programmatic use)
tingly-pm -mode run -dir ./my-project

# HTTP server mode
tingly-pm -mode serve -addr :8080 -dir ./my-project
```

### Configuration

Set the API key via environment variable:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
tingly-pm
```

Or provide a config directory with a `config.json`:

```json
{
  "base_url": "https://api.anthropic.com",
  "api_key": "sk-ant-...",
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 8192
}
```

```bash
tingly-pm -config /path/to/config
```

## Usage

Once running, interact with the agent in natural language:

```
> Create a task: implement user login
> What's blocked right now?
> Assign the auth task to alice
> Generate a daily report
> /quit
```

Informal task references ("that login task", "任务A") are resolved automatically.

## Board Structure

All data lives in a `.pm/` directory inside the project folder:

```
.pm/
├── tasks/              # Active tasks (markdown with YAML frontmatter)
│   └── p1-TASK-20260328-143022-user-login.md
├── archive/            # Archived tasks, organized by month
│   └── 202603/
├── reports/            # Generated reports
├── sessions/           # Persisted conversation state
├── members.json        # Team member registry
└── timeline.jsonl      # Append-only event log
```

### Task Format

Each task is a markdown file with YAML frontmatter:

```markdown
---
id: TASK-20260328-143022
title: Implement user login
slug: user-login
status: in_progress
priority: p1
assignee: alice
created: 2026-03-28T14:30:22Z
updated: 2026-03-28T15:00:00Z
labels: [backend, auth]
blocks: [TASK-20260328-150000]
blocked_by: []
---

## Description

Add OAuth2 login flow with session management.
```

### Task Status Lifecycle

| Status        | Location         | Description               |
|---------------|------------------|---------------------------|
| `todo`        | `tasks/`         | Not yet started           |
| `in_progress` | `tasks/`         | Currently being worked on |
| `blocked`     | `tasks/`         | Waiting on a dependency   |
| `review`      | `tasks/`         | Ready for review          |
| `done`        | `archive/YYYYMM/`| Completed                 |
| `dropped`     | `archive/YYYYMM/`| Will not be completed     |

### Priority Levels

| Priority | Meaning             |
|----------|---------------------|
| `p0`     | Critical / blocking |
| `p1`     | Important (default) |
| `p2`     | Normal              |
| `p3`     | Low / nice to have  |

## HTTP API (serve mode)

| Method | Endpoint   | Description        |
|--------|------------|--------------------|
| POST   | `/message` | Send a message     |
| GET    | `/health`  | Health check       |

**POST /message** request body:

```json
{ "content": "list all p0 tasks" }
```

## CLI Flags

| Flag      | Default | Description               |
|-----------|---------|---------------------------|
| `-mode`   | `chat`  | `chat`, `run`, or `serve`|
| `-addr`   | `:8080` | HTTP address (serve mode) |
| `-dir`    | `.`     | Project directory         |
| `-config` |         | Path to `config.json`     |

## Architecture

```
main.go          # Entry point, modes (chat/run/serve), config loading
├── board/       # File-based task board logic
│   ├── board.go      # Directory initialization
│   ├── task.go       # Task CRUD, archive, status lifecycle
│   ├── task_file.go  # Markdown frontmatter parsing/formatting
│   ├── member.go     # Team member registry
│   ├── report.go     # Summary and report generation
│   └── timeline.go   # Append-only event log
├── prompt/      # System prompt definition
└── tools/       # Agent tools (task ops, search, reports, session)
```

Built on a **ReAct loop** (reason + act) with tool calling via [tingly-agentscope](https://github.com/tingly-dev/tingly-agentscope).

## Eval Loop

The eval loop iteratively improves the agent through seed-guided exploration and per-experiment verification. Each round runs a baseline, forms hypotheses, and experiments one at a time — each independently evaluated before the next begins. Good changes accumulate; bad changes are discarded immediately.

### Quick Start

```bash
# Run 4 rounds (auto-detects next round number from existing artifacts)
./eval-loop.sh

# Dry run — preview prompts without executing
./eval-loop.sh --dry-run
```

### CLI Reference

| Flag | Default | Description |
|------|---------|-------------|
| `-n, --rounds <N>` | `4` | Number of improvement rounds to run |
| `-s, --start <N>` | auto | Starting round number (auto-detects highest existing + 1) |
| `-j, --jobs <N>` | parallel | Subagent parallelism: `0` or `serial` for sequential execution |
| `-m, --model` | default | Claude model to use |
| `-p, --prompt` | built-in | Custom prompt file (replaces the default prompt) |
| `-d, --desc` | none | Exploration seed text to guide improvement focus |
| `--dry-run` | off | Print commands without executing |
| `-v, --verbose` | on | Show Claude's intermediate steps |

### Examples

```bash
# Continue from where you left off, 3 more rounds
./eval-loop.sh -n 3

# Explicit start round (e.g., rounds 10–12)
./eval-loop.sh -s 10 -n 3

# Serial mode — run subagents one at a time (lower API load)
./eval-loop.sh -j serial
./eval-loop.sh -j 0

# Focus exploration on a specific area
./eval-loop.sh -d "focus on cross-language duplicate detection"

# Full control
./eval-loop.sh -s 5 -n 3 -m opus -j serial -d "explore edge cases in task dependencies"
```

### Round Numbering

Round numbers are persistent. The script scans `.eval/` for existing artifacts (log files, directories, reports) and starts at `max(existing) + 1`. This prevents overwriting previous results. Use `-s` to override.

### Subagent Parallelism

By default, subagents run in parallel during baseline and verification phases. When rate limits or resource constraints are a concern, use `-j serial` to run subagents sequentially.

### Artifacts

Each round writes to `.eval/`:

```
.eval/
├── round-{N}.log              # Full Claude output for that round
├── round-{N}/                 # Per-round subdirectory
│   ├── baseline.md            # Baseline pass rate and hypotheses
│   ├── experiments.md         # All experiments with keep/discard decisions
│   └── final-report.md        # Summary and learnings
└── fixtures/                  # Test fixtures (JSONL streams)
    └── INDEX.md               # Fixture manifest
```

## License

[MPL-2.0](LICENSE.txt)
