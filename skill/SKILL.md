---
name: pm
description: Manage tasks, team members, dependencies, timelines, and reports on a local file-based project board via the `pm` CLI. Trigger when the user asks to track work in the current project — create/update/list/search/archive tasks, assign people, add comments, record dependencies, or generate standup/weekly reports. Looks for a `.pm/` directory in the project root.
---

# pm — Project Manager CLI

`pm` is a CLI front-end for the tingly-pm file-based board. Each project keeps
its state in `.pm/` (tasks, members, timeline, reports) inside the project
directory. Use `pm` instead of editing those files by hand — it keeps the
timeline log consistent.

## When to use

- User says "create a task", "add a todo", "track this", "记一个任务".
- User asks "what's in progress?", "list p0 tasks", "who's on what?".
- User asks for a standup, daily, or weekly report.
- User wants to assign work, set dependencies, or archive completed work.
- The current project has (or should have) a `.pm/` directory.

If the project has no `.pm/`, run `pm init` first.

## Output modes

- **Default**: human-readable text. Use this for confirmations the user reads.
- **`--json`**: structured JSON. Use this when you need to extract a field
  (task ID, status, etc.) for a follow-up command. Pipe through `jq`.

## Common recipes

### Initialize a board (once per project)
```
pm init
```

### Create a task — always search first to avoid duplicates
```
pm task search "login"
# If nothing relevant, then:
pm task create --title "Fix login bug" --slug fix-login --priority p0 \
  --assignee alice --labels backend,auth \
  --description "Session token not refreshing on /me endpoint."
```
Slug must be kebab-case English (≤50 chars). Translate non-English titles
into a slug before calling.

### Resolve "that login task" → task ID
```
ID=$(pm task search "login" --json | jq -r '.[0].id')
pm task update "$ID" --status in_progress --assignee alice
```

### Update only the fields the user asked about
`pm task update` only changes flags you pass — never invent values for
fields the user didn't mention.
```
pm task update TASK-20260506-094928 --status in_progress
pm task update TASK-20260506-094928 --priority p0
pm task update TASK-20260506-094928 --assignee bob
```

### Add a comment
```
pm task comment TASK-20260506-094928 \
  --content "Reproduced on staging, root cause is stale JWT" \
  --by claude
```

### Block / unblock
```
pm task block   TASK-add-tests --on TASK-refactor-auth
pm task unblock TASK-add-tests --on TASK-refactor-auth
pm task list --blockers          # show only currently-blocked tasks
```

### Archive completed work
```
pm task archive TASK-20260506-094928 --resolution done
# or
pm task archive TASK-20260506-094928 --resolution dropped
```

### Members
```
pm member add alice --type human --labels frontend,react
pm member add alice --labels frontend,react,typescript   # upsert: type omitted is OK for existing
pm member list
pm member search --labels react
pm member remove alice
```
**Important**: when the user asks to assign a task to someone unknown, ask
whether to register them first — don't auto-register, since typos persist.

### Reports & overview
```
pm summary                       # quick board snapshot
pm report daily                  # daily progress report (default)
pm report weekly                 # last 7 days
pm timeline --limit 20           # recent timeline events, newest first
```

## Reference

### Task lifecycle
- Active (in `.pm/tasks/`): `todo`, `in_progress`, `blocked`, `review`
- Terminal (in `.pm/archive/YYYYMM/`): `done`, `dropped` — reach via `pm task archive`

### Priority guide
- `p0`: critical / blocking ("紧急", "urgent", "blocker")
- `p1`: important
- `p2`: normal (default if user says nothing about priority)
- `p3`: low / nice-to-have

### Task ID format
`TASK-YYYYMMDD-HHmmss` — auto-generated at creation time.

### Global flags (every subcommand accepts)
- `--dir DIR` — project directory (default: current directory)
- `--json` — emit JSON instead of human-readable text
- `--by NAME` — actor recorded in the timeline event (default: `agent`); set
  to a more specific name like `claude` or the user's handle when relevant.

### Behaviors to follow
1. **Search before create** — call `pm task search` with key keywords from the
   proposed title to detect duplicates. Only create if nothing matches.
2. **Resolve references silently** — when the user says "the auth task" or
   "任务A", run `pm task search` to find the ID instead of asking the user.
3. **Update narrowly** — pass only the flags the user explicitly mentioned.
4. **Confirm assignments** — if assignee isn't a known member, ask whether to
   register before assigning.
5. **Match the user's language** — task titles stay in the user's original
   language; only the slug gets translated to English.

### Full command list
Run `pm` (no args) or `pm <command> -h` for the latest signature.

```
pm init
pm task create | update | get | list | search | archive | comment | block | unblock
pm member add | list | search | remove
pm report [daily|weekly]
pm summary
pm timeline [--limit N]
```
