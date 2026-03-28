package board

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GenerateSummary generates a quick status summary
func GenerateSummary(pmDir string) (string, error) {
	tasks, err := ListTasks(pmDir, "", "", "", "")
	if err != nil {
		return "", err
	}

	counts := make(map[string]int)
	assignees := make(map[string]bool)
	var blocked []string

	for _, t := range tasks {
		counts[t.Status]++
		if t.Assignee != "" {
			assignees[t.Assignee] = true
		}
		if t.Status == "blocked" {
			blocked = append(blocked, fmt.Sprintf("  - %s: %s (blocked by %s)", t.ID, t.Title, strings.Join(t.BlockedBy, ", ")))
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Total active: %d tasks\n", len(tasks)))
	for _, s := range ActiveStatuses {
		if c := counts[s]; c > 0 {
			b.WriteString(fmt.Sprintf("  %s: %d\n", s, c))
		}
	}

	if len(assignees) > 0 {
		names := make([]string, 0, len(assignees))
		for a := range assignees {
			names = append(names, a)
		}
		b.WriteString(fmt.Sprintf("Active members: %s\n", strings.Join(names, ", ")))
	}

	if len(blocked) > 0 {
		b.WriteString("Blocked:\n")
		for _, line := range blocked {
			b.WriteString(line + "\n")
		}
	}

	return b.String(), nil
}

// GenerateReport generates a daily or weekly report
func GenerateReport(pmDir, reportType string) (string, error) {
	tasks, err := ListTasks(pmDir, "", "", "", "")
	if err != nil {
		return "", err
	}

	events, _ := ReadTimeline(pmDir)

	now := time.Now()
	var since time.Time
	switch reportType {
	case "weekly":
		since = now.AddDate(0, 0, -7)
	default:
		since = now.AddDate(0, 0, -1)
		reportType = "daily"
	}

	// Filter recent events
	var recentEvents []TimelineEvent
	for _, e := range events {
		t, err := time.Parse(time.RFC3339, e.Timestamp)
		if err != nil {
			continue
		}
		if t.After(since) {
			recentEvents = append(recentEvents, e)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("## %s Report - %s\n\n", strings.ToUpper(reportType[:1])+reportType[1:], now.Format("2006-01-02")))

	// Summary
	counts := make(map[string]int)
	for _, t := range tasks {
		counts[t.Status]++
	}
	b.WriteString("### Summary\n")
	b.WriteString(fmt.Sprintf("Active tasks: %d\n", len(tasks)))
	for _, s := range ActiveStatuses {
		if c := counts[s]; c > 0 {
			b.WriteString(fmt.Sprintf("  %s: %d\n", s, c))
		}
	}
	b.WriteString("\n")

	// Recent activity
	if len(recentEvents) > 0 {
		b.WriteString("### Recent Activity\n")
		for _, e := range recentEvents {
			switch e.Event {
			case "task_created":
				b.WriteString(fmt.Sprintf("- Created %s (by %s)\n", e.Task, e.By))
			case "status_changed":
				b.WriteString(fmt.Sprintf("- %s: %s → %s (by %s)\n", e.Task, e.From, e.To, e.By))
			case "task_assigned":
				b.WriteString(fmt.Sprintf("- %s assigned to %s\n", e.Task, e.Assignee))
			case "task_archived":
				b.WriteString(fmt.Sprintf("- %s archived (%s)\n", e.Task, e.Status))
			case "comment_added":
				b.WriteString(fmt.Sprintf("- Comment on %s by %s\n", e.Task, e.By))
			}
		}
	} else {
		b.WriteString("### Recent Activity\nNo activity in this period.\n")
	}

	// Save report
	reportFilename := fmt.Sprintf("%s-%s.md", now.Format("20060102"), reportType)
	reportPath := filepath.Join(pmDir, "reports", reportFilename)
	reportContent := b.String()
	os.WriteFile(reportPath, []byte(reportContent), 0644)

	return reportContent, nil
}
