# pm skill

A Claude Code skill that teaches the agent to drive the `pm` CLI for
file-based project management.

## Install

Build the CLI and put it on your `$PATH`:

```bash
go build -o /usr/local/bin/pm ./cmd/pm
# or, for a user-local install:
go build -o ~/bin/pm ./cmd/pm
```

Then symlink (or copy) this directory into your Claude Code skills folder:

```bash
ln -s "$(pwd)/skill" ~/.claude/skills/pm
# or:
cp -r skill ~/.claude/skills/pm
```

Confirm the agent picks it up — start a new Claude Code session in any
project and ask "create a high-priority task to fix login". The agent
should run `pm init` (if needed) and `pm task create ...`.

## Verify

```bash
mkdir /tmp/pm-demo && cd /tmp/pm-demo
pm init
pm task create --title "demo" --slug demo --priority p1
pm task list
```

## Updating the skill

Edit `SKILL.md` in this directory; if you symlinked, the changes are picked
up immediately by new Claude Code sessions. If you copied, re-copy.
