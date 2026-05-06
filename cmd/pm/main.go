// pm is a CLI front-end for the tingly-pm file-based project board.
//
// It exposes the same operations as the ReAct agent in the parent binary,
// but as plain subcommands so any external agent (Claude Code, scripts, etc.)
// can drive the board directly.
package main

import (
	"fmt"
	"os"
)

const usage = `pm — file-based project board CLI

Usage:
  pm <command> [subcommand] [flags]

Commands:
  init                       Create .pm/ in the project directory
  task create                Create a task
  task update <id>           Update task fields
  task get <id>              Show a task (with body)
  task list                  List active tasks (filterable)
  task search <query>        Full-text search across active + archived tasks
  task archive <id>          Archive a task as done|dropped
  task comment <id>          Append a timestamped comment to a task
  task block <id>            Add a blocking dependency
  task unblock <id>          Remove a blocking dependency
  member add <name>          Create or update a team member (upsert)
  member list                List members (filterable by --type)
  member search              Search members by labels or name
  member remove <name>       Remove a member
  report [daily|weekly]      Generate progress report (default: daily)
  summary                    Quick board status summary
  timeline                   Recent timeline events (newest first)

Global flags (accepted by every subcommand):
  --dir DIR                  Project directory containing .pm/ (default: ".")
  --json                     Emit structured JSON instead of human-readable text

Run "pm <command> -h" for command-specific flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "-h", "--help", "help":
		fmt.Print(usage)
	case "init":
		runInit(args)
	case "task":
		runTask(args)
	case "member":
		runMember(args)
	case "report":
		runReport(args)
	case "summary":
		runSummary(args)
	case "timeline":
		runTimeline(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", cmd, usage)
		os.Exit(2)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
