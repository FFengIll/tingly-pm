package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FFengIll/tingly-pm/board"
)

// commonFlags holds the global flags every subcommand accepts.
type commonFlags struct {
	dir  string
	json bool
}

// addCommonFlags wires --dir and --json onto a subcommand FlagSet.
func addCommonFlags(fs *flag.FlagSet) *commonFlags {
	c := &commonFlags{}
	fs.StringVar(&c.dir, "dir", ".", "project directory containing .pm/")
	fs.BoolVar(&c.json, "json", false, "emit JSON output")
	return c
}

// parseInterspersed parses args allowing flags to appear after positional
// arguments — e.g. `pm task update TASK-xxx --status in_progress`. The
// stdlib `flag` package stops at the first non-flag, which would force
// awkward orderings. We pre-split args into flag tokens and positional tokens,
// using the FlagSet to detect which flags consume a value.
func parseInterspersed(fs *flag.FlagSet, args []string) []string {
	var flagArgs, positional []string
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			positional = append(positional, a)
			i++
			continue
		}
		// It's a flag token.
		flagArgs = append(flagArgs, a)

		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") {
			// `--flag=value` embeds the value.
			i++
			continue
		}
		f := fs.Lookup(name)
		if f == nil {
			// Unknown flag — let fs.Parse emit the error.
			i++
			continue
		}
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			i++
			continue
		}
		// Non-bool: consume the next token as the value.
		if i+1 < len(args) {
			flagArgs = append(flagArgs, args[i+1])
			i += 2
		} else {
			i++
		}
	}

	if err := fs.Parse(flagArgs); err != nil {
		fail(err)
	}
	return positional
}

// pmDir resolves the absolute .pm/ path for the project directory.
func (c *commonFlags) pmDir() string {
	abs, err := filepath.Abs(c.dir)
	if err != nil {
		fail(err)
	}
	return filepath.Join(abs, ".pm")
}

// requireInit fails fast with a clear hint when .pm/ is missing.
func (c *commonFlags) requireInit() string {
	pm := c.pmDir()
	if _, err := os.Stat(filepath.Join(pm, "tasks")); err != nil {
		failf(".pm/ not initialized in %s — run `pm init` first", c.dir)
	}
	return pm
}

// --- JSON shapes ---

// taskJSON mirrors board.Task but exposes Body so `pm task get --json` is useful.
type taskJSON struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Slug      string   `json:"slug"`
	Status    string   `json:"status"`
	Priority  string   `json:"priority"`
	Assignee  string   `json:"assignee"`
	Created   string   `json:"created"`
	Updated   string   `json:"updated"`
	Labels    []string `json:"labels"`
	Blocks    []string `json:"blocks"`
	BlockedBy []string `json:"blocked_by"`
	ClosedAt  string   `json:"closed_at,omitempty"`
	Body      string   `json:"body,omitempty"`
}

func toTaskJSON(t *board.Task) taskJSON {
	return taskJSON{
		ID:        t.ID,
		Title:     t.Title,
		Slug:      t.Slug,
		Status:    t.Status,
		Priority:  t.Priority,
		Assignee:  t.Assignee,
		Created:   t.Created,
		Updated:   t.Updated,
		Labels:    t.Labels,
		Blocks:    t.Blocks,
		BlockedBy: t.BlockedBy,
		ClosedAt:  t.ClosedAt,
		Body:      t.Body,
	}
}

// --- Writers ---

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fail(err)
	}
}

func printTasks(tasks []*board.Task, jsonMode bool) {
	if jsonMode {
		out := make([]taskJSON, 0, len(tasks))
		for _, t := range tasks {
			out = append(out, toTaskJSON(t))
		}
		writeJSON(out)
		return
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}

	now := time.Now()
	currentPriority := ""
	for i, t := range tasks {
		if t.Priority != currentPriority {
			currentPriority = t.Priority
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("=== %s ===\n", strings.ToUpper(currentPriority))
		}
		assignee := ""
		if t.Assignee != "" {
			assignee = " -> " + t.Assignee
		}
		fmt.Printf("[%s] %s: %s (%s, %s ago)%s\n",
			t.Priority, t.ID, t.Title, t.Status, ageSince(t.Created, now), assignee)
	}
}

func printTask(t *board.Task, jsonMode bool) {
	if jsonMode {
		writeJSON(toTaskJSON(t))
		return
	}
	fmt.Printf("ID:        %s\n", t.ID)
	fmt.Printf("Title:     %s\n", t.Title)
	fmt.Printf("Slug:      %s\n", t.Slug)
	fmt.Printf("Status:    %s\n", t.Status)
	fmt.Printf("Priority:  %s\n", t.Priority)
	fmt.Printf("Assignee:  %s\n", t.Assignee)
	fmt.Printf("Created:   %s\n", t.Created)
	fmt.Printf("Updated:   %s\n", t.Updated)
	if t.ClosedAt != "" {
		fmt.Printf("ClosedAt:  %s\n", t.ClosedAt)
	}
	if len(t.Labels) > 0 {
		fmt.Printf("Labels:    %s\n", strings.Join(t.Labels, ", "))
	}
	if len(t.BlockedBy) > 0 {
		fmt.Printf("BlockedBy: %s\n", strings.Join(t.BlockedBy, ", "))
	}
	if len(t.Blocks) > 0 {
		fmt.Printf("Blocks:    %s\n", strings.Join(t.Blocks, ", "))
	}
	if t.Body != "" {
		fmt.Println()
		fmt.Println("--- Body ---")
		fmt.Print(t.Body)
		if !strings.HasSuffix(t.Body, "\n") {
			fmt.Println()
		}
	}
}

func printMembers(members []board.Member, jsonMode bool) {
	if jsonMode {
		writeJSON(members)
		return
	}
	if len(members) == 0 {
		fmt.Println("No members found.")
		return
	}
	for _, m := range members {
		labels := ""
		if len(m.Labels) > 0 {
			labels = " [" + strings.Join(m.Labels, ", ") + "]"
		}
		fmt.Printf("- %s (%s)%s\n", m.Name, m.Type, labels)
	}
}

func printEvents(events []board.TimelineEvent, jsonMode bool) {
	if jsonMode {
		writeJSON(events)
		return
	}
	if len(events) == 0 {
		fmt.Println("No timeline events yet.")
		return
	}
	for _, e := range events {
		var b strings.Builder
		fmt.Fprintf(&b, "[%s] %s", e.Timestamp, e.Event)
		if e.Task != "" {
			fmt.Fprintf(&b, " task=%s", e.Task)
		}
		if e.By != "" {
			fmt.Fprintf(&b, " by=%s", e.By)
		}
		if e.Name != "" {
			fmt.Fprintf(&b, " name=%s", e.Name)
		}
		if e.From != "" || e.To != "" {
			fmt.Fprintf(&b, " %s→%s", e.From, e.To)
		}
		if e.Content != "" {
			fmt.Fprintf(&b, " %s", e.Content)
		}
		fmt.Println(b.String())
	}
}

// printText prints a free-form string (report/summary). In JSON mode, wraps it
// as {"output": "..."} so callers can still parse a single object.
func printText(s string, jsonMode bool) {
	if jsonMode {
		writeJSON(map[string]string{"output": s})
		return
	}
	fmt.Print(s)
	if !strings.HasSuffix(s, "\n") {
		fmt.Println()
	}
}

// printOK prints a confirmation. JSON mode emits a status object including any
// payload (e.g., the created task ID) for programmatic callers.
func printOK(msg string, jsonMode bool, payload map[string]any) {
	if jsonMode {
		out := map[string]any{"ok": true, "message": msg}
		for k, v := range payload {
			out[k] = v
		}
		writeJSON(out)
		return
	}
	fmt.Println(msg)
}

// ageSince computes a human-readable age string from an RFC3339 timestamp.
// Examples: "<1h", "3h", "2d", "1w", "4w".
func ageSince(created string, now time.Time) string {
	t, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return "?"
	}
	dur := now.Sub(t)
	if dur < 0 {
		return "<1h"
	}
	hours := int(dur.Hours())
	days := hours / 24
	if hours < 1 {
		return "<1h"
	}
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	if days < 7 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dw", days/7)
}

// splitLabels parses a comma-separated label string, trimming whitespace and
// dropping empty entries. Returns nil for an empty input so callers can
// distinguish "no change" from "clear all".
func splitLabels(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
