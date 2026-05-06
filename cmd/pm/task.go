package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FFengIll/tingly-pm/board"
)

const taskUsage = `Usage: pm task <subcommand> [flags]

Subcommands:
  create     Create a new task
  update     Update task fields (only fields you pass are changed)
  get        Show one task with body
  list       List active tasks (filterable)
  search     Full-text search across active + archived
  archive    Move a task to archive (done|dropped)
  comment    Append a timestamped comment
  block      Add a blocking dependency
  unblock    Remove a blocking dependency
`

func runTask(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, taskUsage)
		os.Exit(2)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "-h", "--help", "help":
		fmt.Print(taskUsage)
	case "create":
		taskCreate(rest)
	case "update":
		taskUpdate(rest)
	case "get":
		taskGet(rest)
	case "list":
		taskList(rest)
	case "search":
		taskSearch(rest)
	case "archive":
		taskArchive(rest)
	case "comment":
		taskComment(rest)
	case "block":
		taskBlock(rest)
	case "unblock":
		taskUnblock(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown task subcommand: %s\n\n%s", sub, taskUsage)
		os.Exit(2)
	}
}

func taskCreate(args []string) {
	fs := flag.NewFlagSet("task create", flag.ExitOnError)
	common := addCommonFlags(fs)
	title := fs.String("title", "", "task title (required)")
	slug := fs.String("slug", "", "kebab-case English slug (required, ≤50 chars)")
	priority := fs.String("priority", "p1", "priority: p0|p1|p2|p3")
	assignee := fs.String("assignee", "", "assignee name")
	labels := fs.String("labels", "", "comma-separated labels")
	desc := fs.String("description", "", "task description (markdown)")
	by := fs.String("by", "agent", "actor recorded in the timeline event")
	parseInterspersed(fs, args)
	if *title == "" || *slug == "" {
		failf("--title and --slug are required")
	}

	pm := common.requireInit()
	t := &board.Task{
		Title:    *title,
		Slug:     *slug,
		Priority: *priority,
		Assignee: *assignee,
		Labels:   splitLabels(*labels),
	}
	if *desc != "" {
		t.Body = fmt.Sprintf("## Description\n\n%s\n", *desc)
	}
	if err := board.CreateTask(pm, t); err != nil {
		fail(err)
	}
	board.AppendEvent(pm, &board.TimelineEvent{
		Event: "task_created",
		Task:  t.ID,
		By:    *by,
	})

	printOK(
		fmt.Sprintf("Created %s: %s [%s]", t.ID, t.Title, t.Priority),
		common.json,
		map[string]any{"id": t.ID, "task": toTaskJSON(t)},
	)
}

func taskUpdate(args []string) {
	fs := flag.NewFlagSet("task update", flag.ExitOnError)
	common := addCommonFlags(fs)
	status := fs.String("status", "", "new status: todo|in_progress|blocked|review")
	priority := fs.String("priority", "", "new priority: p0|p1|p2|p3")
	assignee := fs.String("assignee", "", "new assignee")
	title := fs.String("title", "", "new title")
	slug := fs.String("slug", "", "new slug (required if title changed)")
	labels := fs.String("labels", "", "comma-separated labels (replaces existing)")
	by := fs.String("by", "agent", "actor recorded in the timeline event")
	pos := parseInterspersed(fs, args)
	if len(pos) < 1 {
		failf("task ID is required: pm task update <id> [flags]")
	}
	id := pos[0]

	pm := common.requireInit()
	old, err := board.GetTask(pm, id)
	if err != nil {
		fail(err)
	}
	oldStatus := old.Status
	oldPriority := old.Priority

	updates := map[string]any{}
	if *status != "" {
		updates["status"] = *status
	}
	if *priority != "" {
		updates["priority"] = *priority
	}
	if *assignee != "" {
		updates["assignee"] = *assignee
	}
	if *title != "" {
		updates["title"] = *title
	}
	if *slug != "" {
		updates["slug"] = *slug
	}
	if *labels != "" {
		updates["labels"] = splitLabels(*labels)
	}
	if len(updates) == 0 {
		failf("no fields to update")
	}

	t, err := board.UpdateTask(pm, id, updates)
	if err != nil {
		fail(err)
	}

	if *status != "" && *status != oldStatus {
		board.AppendEvent(pm, &board.TimelineEvent{
			Event: "status_changed", Task: t.ID, From: oldStatus, To: *status, By: *by,
		})
	}
	if *priority != "" && *priority != oldPriority {
		board.AppendEvent(pm, &board.TimelineEvent{
			Event: "priority_changed", Task: t.ID, From: oldPriority, To: *priority, By: *by,
		})
	}

	printOK(
		fmt.Sprintf("Updated %s: status=%s priority=%s assignee=%s",
			t.ID, t.Status, t.Priority, t.Assignee),
		common.json,
		map[string]any{"task": toTaskJSON(t)},
	)
}

func taskGet(args []string) {
	fs := flag.NewFlagSet("task get", flag.ExitOnError)
	common := addCommonFlags(fs)
	pos := parseInterspersed(fs, args)
	if len(pos) < 1 {
		failf("task ID is required: pm task get <id>")
	}
	pm := common.requireInit()
	t, err := board.GetTask(pm, pos[0])
	if err != nil {
		fail(err)
	}
	printTask(t, common.json)
}

func taskList(args []string) {
	fs := flag.NewFlagSet("task list", flag.ExitOnError)
	common := addCommonFlags(fs)
	status := fs.String("status", "", "filter by status")
	assignee := fs.String("assignee", "", "filter by assignee")
	priority := fs.String("priority", "", "filter by priority")
	label := fs.String("label", "", "filter by label")
	blockers := fs.Bool("blockers", false, "only tasks that have blocked_by relations")
	parseInterspersed(fs, args)

	pm := common.requireInit()
	tasks, err := board.ListTasks(pm, *status, *assignee, *priority, *label)
	if err != nil {
		fail(err)
	}
	if *blockers {
		filtered := tasks[:0]
		for _, t := range tasks {
			if len(t.BlockedBy) > 0 {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	printTasks(tasks, common.json)
}

func taskSearch(args []string) {
	fs := flag.NewFlagSet("task search", flag.ExitOnError)
	common := addCommonFlags(fs)
	pos := parseInterspersed(fs, args)
	if len(pos) < 1 {
		failf("search query is required: pm task search <query>")
	}
	q := strings.ToLower(pos[0])
	pm := common.requireInit()

	active, err := board.ListTasks(pm, "", "", "", "")
	if err != nil {
		fail(err)
	}
	var matches []*board.Task
	for _, t := range active {
		if matchTask(t, q) {
			matches = append(matches, t)
		}
	}

	// Walk archive/YYYYMM/
	archiveDir := filepath.Join(pm, "archive")
	entries, _ := os.ReadDir(archiveDir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		monthEntries, err := os.ReadDir(filepath.Join(archiveDir, e.Name()))
		if err != nil {
			continue
		}
		for _, me := range monthEntries {
			if me.IsDir() || !strings.HasSuffix(me.Name(), ".md") {
				continue
			}
			t, err := board.ReadTaskFile(filepath.Join(archiveDir, e.Name(), me.Name()))
			if err != nil {
				continue
			}
			if matchTask(t, q) {
				matches = append(matches, t)
			}
		}
	}

	printTasks(matches, common.json)
}

func matchTask(t *board.Task, q string) bool {
	return strings.Contains(strings.ToLower(t.Title), q) ||
		strings.Contains(strings.ToLower(t.Body), q) ||
		strings.Contains(strings.ToLower(t.ID), q)
}

func taskArchive(args []string) {
	fs := flag.NewFlagSet("task archive", flag.ExitOnError)
	common := addCommonFlags(fs)
	resolution := fs.String("resolution", "", "resolution: done|dropped (required)")
	by := fs.String("by", "agent", "actor recorded in the timeline event")
	pos := parseInterspersed(fs, args)
	if len(pos) < 1 {
		failf("task ID is required: pm task archive <id> --resolution done|dropped")
	}
	if *resolution == "" {
		failf("--resolution is required (done or dropped)")
	}

	pm := common.requireInit()
	t, err := board.ArchiveTask(pm, pos[0], *resolution)
	if err != nil {
		fail(err)
	}
	board.AppendEvent(pm, &board.TimelineEvent{
		Event: "task_archived", Task: t.ID, Status: *resolution, By: *by,
	})
	printOK(
		fmt.Sprintf("Archived %s as %s", t.ID, *resolution),
		common.json,
		map[string]any{"id": t.ID, "resolution": *resolution},
	)
}

func taskComment(args []string) {
	fs := flag.NewFlagSet("task comment", flag.ExitOnError)
	common := addCommonFlags(fs)
	content := fs.String("content", "", "comment text (required)")
	by := fs.String("by", "agent", "comment author")
	pos := parseInterspersed(fs, args)
	if len(pos) < 1 {
		failf("task ID is required: pm task comment <id> --content ...")
	}
	if *content == "" {
		failf("--content is required")
	}

	pm := common.requireInit()
	t, err := board.GetTask(pm, pos[0])
	if err != nil {
		fail(err)
	}
	now := time.Now().UTC().Format("2006-01-02 15:04")
	t.Body += fmt.Sprintf("\n- [%s] %s: %s\n", now, *by, *content)

	if err := os.WriteFile(t.FilePath, []byte(board.FormatTaskFile(t)), 0644); err != nil {
		fail(err)
	}
	board.AppendEvent(pm, &board.TimelineEvent{
		Event: "comment_added", Task: t.ID, By: *by, Content: *content,
	})
	printOK(
		fmt.Sprintf("Comment added to %s", t.ID),
		common.json,
		map[string]any{"id": t.ID},
	)
}

func taskBlock(args []string) {
	fs := flag.NewFlagSet("task block", flag.ExitOnError)
	common := addCommonFlags(fs)
	on := fs.String("on", "", "task ID that blocks this one (required)")
	pos := parseInterspersed(fs, args)
	if len(pos) < 1 || *on == "" {
		failf("usage: pm task block <id> --on <other-id>")
	}
	id := pos[0]
	pm := common.requireInit()

	t, err := board.GetTask(pm, id)
	if err != nil {
		fail(err)
	}
	if !board.ContainsStr(t.BlockedBy, *on) {
		t.BlockedBy = append(t.BlockedBy, *on)
	}
	if err := os.WriteFile(t.FilePath, []byte(board.FormatTaskFile(t)), 0644); err != nil {
		fail(err)
	}
	if blocker, err := board.GetTask(pm, *on); err == nil && !board.ContainsStr(blocker.Blocks, id) {
		blocker.Blocks = append(blocker.Blocks, id)
		_ = os.WriteFile(blocker.FilePath, []byte(board.FormatTaskFile(blocker)), 0644)
	}
	printOK(
		fmt.Sprintf("%s is now blocked by %s", id, *on),
		common.json,
		map[string]any{"id": id, "blocked_by": *on},
	)
}

func taskUnblock(args []string) {
	fs := flag.NewFlagSet("task unblock", flag.ExitOnError)
	common := addCommonFlags(fs)
	on := fs.String("on", "", "task ID to unblock from (required)")
	pos := parseInterspersed(fs, args)
	if len(pos) < 1 || *on == "" {
		failf("usage: pm task unblock <id> --on <other-id>")
	}
	id := pos[0]
	pm := common.requireInit()

	t, err := board.GetTask(pm, id)
	if err != nil {
		fail(err)
	}
	t.BlockedBy = removeStr(t.BlockedBy, *on)
	if err := os.WriteFile(t.FilePath, []byte(board.FormatTaskFile(t)), 0644); err != nil {
		fail(err)
	}
	if blocker, err := board.GetTask(pm, *on); err == nil {
		blocker.Blocks = removeStr(blocker.Blocks, id)
		_ = os.WriteFile(blocker.FilePath, []byte(board.FormatTaskFile(blocker)), 0644)
	}
	printOK(
		fmt.Sprintf("Removed dependency: %s no longer blocked by %s", id, *on),
		common.json,
		map[string]any{"id": id, "unblocked_from": *on},
	)
}

func removeStr(s []string, v string) []string {
	var out []string
	for _, item := range s {
		if item != v {
			out = append(out, item)
		}
	}
	return out
}
