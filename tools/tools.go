package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FFengIll/tingly-pm/board"

	"github.com/tingly-dev/tingly-agentscope/pkg/session"
	"github.com/tingly-dev/tingly-agentscope/pkg/tool"
)

// PMTools provides all PM agent tools
type PMTools struct {
	pmDir       string
	sessionMgr  *session.SessionManager
	sessionID   string
}

// NewPMTools creates a new PMTools instance
func NewPMTools(pmDir string) *PMTools {
	return &PMTools{pmDir: pmDir}
}

// SetSessionManager sets the session manager for session persistence tools
func (p *PMTools) SetSessionManager(mgr *session.SessionManager, sessionID string) {
	p.sessionMgr = mgr
	p.sessionID = sessionID
}

// --- Arg Structs ---

type CreateTaskArgs struct {
	Title       string `json:"title" description:"Task title" required:"true"`
	Slug        string `json:"slug" description:"Kebab-case English slug for filename, max 50 chars, translate Chinese to English" required:"true"`
	Priority    string `json:"priority" description:"Priority: p0, p1, p2, or p3 (lower number = higher priority)"`
	Assignee    string `json:"assignee" description:"Assignee name"`
	Labels      string `json:"labels" description:"Comma-separated labels"`
	Description string `json:"description" description:"Task description in markdown"`
}

type UpdateTaskArgs struct {
	TaskID   string `json:"task_id" description:"Task ID e.g. TASK-20260327-143022" required:"true"`
	Status   string `json:"status" description:"New status: todo, in_progress, blocked, review"`
	Priority string `json:"priority" description:"New priority: p0, p1, p2, p3"`
	Assignee string `json:"assignee" description:"New assignee"`
	Labels   string `json:"labels" description:"Comma-separated labels, replaces existing"`
	Title    string `json:"title" description:"New title"`
	Slug     string `json:"slug" description:"New slug if title changed"`
}

type GetTaskArgs struct {
	TaskID string `json:"task_id" description:"Task ID" required:"true"`
}

type ListTasksArgs struct {
	Status       string `json:"status" description:"Filter by status"`
	Assignee     string `json:"assignee" description:"Filter by assignee"`
	Priority     string `json:"priority" description:"Filter by priority"`
	Label        string `json:"label" description:"Filter by label"`
	ShowBlockers bool   `json:"show_blockers" description:"If true, only return tasks that have blocked_by relations"`
}

type ArchiveTaskArgs struct {
	TaskID     string `json:"task_id" description:"Task ID to archive" required:"true"`
	Resolution string `json:"resolution" description:"Resolution: done or dropped" required:"true"`
}

type AddCommentArgs struct {
	TaskID  string `json:"task_id" description:"Task ID" required:"true"`
	Content string `json:"content" description:"Comment text" required:"true"`
	By      string `json:"by" description:"Who is commenting"`
}

type RegisterMemberArgs struct {
	Name       string `json:"name" description:"Member name" required:"true"`
	MemberType string `json:"member_type" description:"Type: human or agent" required:"true"`
	Labels     string `json:"labels" description:"Comma-separated capability labels"`
}

type RemoveMemberArgs struct {
	Name string `json:"name" description:"Member name to remove" required:"true"`
}

type UpdateMemberArgs struct {
	Name       string `json:"name" description:"Member name to update" required:"true"`
	MemberType string `json:"member_type" description:"New type: human or agent"`
	Labels     string `json:"labels" description:"Comma-separated capability labels (replaces existing)"`
}

type ListMembersArgs struct {
	MemberType string `json:"member_type" description:"Filter by type: human or agent"`
}

type SearchMembersArgs struct {
	Labels string `json:"labels" description:"Comma-separated labels to search for (fuzzy match)"`
}

type AssignTaskArgs struct {
	TaskID   string `json:"task_id" description:"Task ID" required:"true"`
	Assignee string `json:"assignee" description:"Assignee name" required:"true"`
}

type AddDependencyArgs struct {
	TaskID    string `json:"task_id" description:"Task that is blocked" required:"true"`
	DependsOn string `json:"depends_on" description:"Task ID that blocks it" required:"true"`
}

type RemoveDependencyArgs struct {
	TaskID    string `json:"task_id" description:"Task to remove dependency from" required:"true"`
	DependsOn string `json:"depends_on" description:"Task ID to unblock" required:"true"`
}

/*
type ShowBlockersArgs struct {
	TaskID string `json:"task_id" description:"Task ID, if empty shows all blocked tasks"`
}
*/

type GenerateReportArgs struct {
	ReportType string `json:"report_type" description:"Report type: daily or weekly"`
}

type SummaryArgs struct{}

type SearchTasksArgs struct {
	Query string `json:"query" description:"Search query" required:"true"`
}

type SaveSessionArgs struct {
	Label string `json:"label" description:"Optional label for this save (e.g., 'before-refactor')"`
}

type ListTimelineArgs struct {
	Limit int `json:"limit" description:"Max number of events to return (default 20, newest first)"`
}

// --- Tool Methods ---

func (p *PMTools) CreateTask(ctx context.Context, args CreateTaskArgs) (*tool.ToolResponse, error) {
	var labels []string
	if args.Labels != "" {
		for _, l := range strings.Split(args.Labels, ",") {
			labels = append(labels, strings.TrimSpace(l))
		}
	}

	t := &board.Task{
		Title:    args.Title,
		Slug:     args.Slug,
		Priority: args.Priority,
		Assignee: args.Assignee,
		Labels:   labels,
	}
	if args.Description != "" {
		t.Body = fmt.Sprintf("## Description\n\n%s\n", args.Description)
	}

	if err := board.CreateTask(p.pmDir, t); err != nil {
		return nil, err
	}

	board.AppendEvent(p.pmDir, &board.TimelineEvent{
		Event: "task_created",
		Task:  t.ID,
		By:    "pm",
	})

	return tool.TextResponse(fmt.Sprintf("Created %s: %s [%s]", t.ID, t.Title, t.Priority)), nil
}

func (p *PMTools) UpdateTask(ctx context.Context, args UpdateTaskArgs) (*tool.ToolResponse, error) {
	oldTask, err := board.GetTask(p.pmDir, args.TaskID)
	if err != nil {
		return nil, err
	}
	oldStatus := oldTask.Status
	oldPriority := oldTask.Priority

	updates := make(map[string]any)
	if args.Status != "" {
		updates["status"] = args.Status
	}
	if args.Priority != "" {
		updates["priority"] = args.Priority
	}
	if args.Assignee != "" {
		updates["assignee"] = args.Assignee
	}
	if args.Title != "" {
		updates["title"] = args.Title
	}
	if args.Slug != "" {
		updates["slug"] = args.Slug
	}
	if args.Labels != "" {
		var labels []string
		for _, l := range strings.Split(args.Labels, ",") {
			labels = append(labels, strings.TrimSpace(l))
		}
		updates["labels"] = labels
	}

	t, err := board.UpdateTask(p.pmDir, args.TaskID, updates)
	if err != nil {
		return nil, err
	}

	if args.Status != "" && args.Status != oldStatus {
		board.AppendEvent(p.pmDir, &board.TimelineEvent{
			Event: "status_changed",
			Task:  t.ID,
			From:  oldStatus,
			To:    args.Status,
			By:    "pm",
		})
	}
	if args.Priority != "" && args.Priority != oldPriority {
		board.AppendEvent(p.pmDir, &board.TimelineEvent{
			Event: "priority_changed",
			Task:  t.ID,
			From:  oldPriority,
			To:    args.Priority,
			By:    "pm",
		})
	}

	return tool.TextResponse(fmt.Sprintf("Updated %s: status=%s priority=%s assignee=%s", t.ID, t.Status, t.Priority, t.Assignee)), nil
}

func (p *PMTools) GetTask(ctx context.Context, args GetTaskArgs) (*tool.ToolResponse, error) {
	t, err := board.GetTask(p.pmDir, args.TaskID)
	if err != nil {
		return nil, err
	}

	data, _ := json.MarshalIndent(t, "", "  ")
	result := string(data)
	if t.Body != "" {
		result += "\n\n--- Body ---\n" + t.Body
	}
	return tool.TextResponse(result), nil
}

func (p *PMTools) ListTasks(ctx context.Context, args ListTasksArgs) (*tool.ToolResponse, error) {
	tasks, err := board.ListTasks(p.pmDir, args.Status, args.Assignee, args.Priority, args.Label)
	if err != nil {
		return nil, err
	}

	// Filter to only tasks with blockers if requested
	if args.ShowBlockers {
		var blocked []*board.Task
		for _, t := range tasks {
			if len(t.BlockedBy) > 0 {
				blocked = append(blocked, t)
			}
		}
		tasks = blocked
	}

	if len(tasks) == 0 {
		return tool.TextResponse("No tasks found."), nil
	}

	var b strings.Builder
	now := time.Now()
	currentPriority := ""
	for _, t := range tasks {
		// Print group header when priority changes
		if t.Priority != currentPriority {
			currentPriority = t.Priority
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("=== %s ===\n", strings.ToUpper(currentPriority)))
		}
		assignee := ""
		if t.Assignee != "" {
			assignee = " -> " + t.Assignee
		}
		age := ageSince(t.Created, now)
		b.WriteString(fmt.Sprintf("[%s] %s: %s (%s, %s ago)%s\n", t.Priority, t.ID, t.Title, t.Status, age, assignee))
	}
	return tool.TextResponse(b.String()), nil
}

// ageSince computes a human-readable age string from an RFC3339 timestamp.
// Examples: "<1h", "3h", "2d", "1w", "4w"
func ageSince(created string, now time.Time) string {
	t, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return "?"
	}
	dur := now.Sub(t)
	if dur < 0 {
		return "<1h"
	}
	days := int(dur.Hours() / 24)
	hours := int(dur.Hours())
	if hours < 1 {
		return "<1h"
	}
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	if days < 7 {
		return fmt.Sprintf("%dd", days)
	}
	weeks := days / 7
	return fmt.Sprintf("%dw", weeks)
}

func (p *PMTools) ArchiveTask(ctx context.Context, args ArchiveTaskArgs) (*tool.ToolResponse, error) {
	t, err := board.ArchiveTask(p.pmDir, args.TaskID, args.Resolution)
	if err != nil {
		return nil, err
	}

	board.AppendEvent(p.pmDir, &board.TimelineEvent{
		Event:  "task_archived",
		Task:   t.ID,
		Status: args.Resolution,
		By:     "pm",
	})

	return tool.TextResponse(fmt.Sprintf("Archived %s as %s", t.ID, args.Resolution)), nil
}

func (p *PMTools) AddComment(ctx context.Context, args AddCommentArgs) (*tool.ToolResponse, error) {
	t, err := board.GetTask(p.pmDir, args.TaskID)
	if err != nil {
		return nil, err
	}

	by := args.By
	if by == "" {
		by = "pm"
	}

	now := time.Now().UTC().Format("2006-01-02 15:04")
	t.Body += fmt.Sprintf("\n- [%s] %s: %s\n", now, by, args.Content)

	content := board.FormatTaskFile(t)
	if err := os.WriteFile(t.FilePath, []byte(content), 0644); err != nil {
		return nil, err
	}

	board.AppendEvent(p.pmDir, &board.TimelineEvent{
		Event:   "comment_added",
		Task:    t.ID,
		By:      by,
		Content: args.Content,
	})

	return tool.TextResponse(fmt.Sprintf("Comment added to %s", t.ID)), nil
}

func (p *PMTools) RegisterMember(ctx context.Context, args RegisterMemberArgs) (*tool.ToolResponse, error) {
	var labels []string
	if args.Labels != "" {
		for _, l := range strings.Split(args.Labels, ",") {
			labels = append(labels, strings.TrimSpace(l))
		}
	}

	if err := board.RegisterMember(p.pmDir, args.Name, args.MemberType, labels); err != nil {
		return nil, err
	}

	board.AppendEvent(p.pmDir, &board.TimelineEvent{
		Event: "member_registered",
		Name:  args.Name,
		Type:  args.MemberType,
		By:    "pm",
	})

	return tool.TextResponse(fmt.Sprintf("Registered %s (%s)", args.Name, args.MemberType)), nil
}

func (p *PMTools) UpdateMember(ctx context.Context, args UpdateMemberArgs) (*tool.ToolResponse, error) {
	var labels []string
	if args.Labels != "" {
		for _, l := range strings.Split(args.Labels, ",") {
			labels = append(labels, strings.TrimSpace(l))
		}
	}

	if err := board.UpdateMember(p.pmDir, args.Name, args.MemberType, labels); err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Updated %s", args.Name)
	if args.MemberType != "" {
		msg += fmt.Sprintf(" (type: %s)", args.MemberType)
	}
	if len(labels) > 0 {
		msg += fmt.Sprintf(" (labels: %s)", strings.Join(labels, ", "))
	}

	board.AppendEvent(p.pmDir, &board.TimelineEvent{
		Event: "member_updated",
		Name:  args.Name,
		By:    "pm",
	})

	return tool.TextResponse(msg), nil
}

func (p *PMTools) ListMembers(ctx context.Context, args ListMembersArgs) (*tool.ToolResponse, error) {
	members, err := board.ListMembers(p.pmDir, args.MemberType)
	if err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return tool.TextResponse("No members found."), nil
	}

	var b strings.Builder
	for _, m := range members {
		labels := ""
		if len(m.Labels) > 0 {
			labels = " [" + strings.Join(m.Labels, ", ") + "]"
		}
		b.WriteString(fmt.Sprintf("- %s (%s)%s\n", m.Name, m.Type, labels))
	}
	return tool.TextResponse(b.String()), nil
}

func (p *PMTools) SearchMembers(ctx context.Context, args SearchMembersArgs) (*tool.ToolResponse, error) {
	members, err := board.SearchMembers(p.pmDir, args.Labels)
	if err != nil {
		return nil, err
	}

	if len(members) == 0 {
		return tool.TextResponse("No members found matching those labels."), nil
	}

	var b strings.Builder
	for _, m := range members {
		labels := ""
		if len(m.Labels) > 0 {
			labels = " [" + strings.Join(m.Labels, ", ") + "]"
		}
		b.WriteString(fmt.Sprintf("- %s (%s)%s\n", m.Name, m.Type, labels))
	}
	return tool.TextResponse(b.String()), nil
}

// Note: assign_task removed in Round 2 — update_task with assignee field covers this.
/*
func (p *PMTools) AssignTask(ctx context.Context, args AssignTaskArgs) (*tool.ToolResponse, error) {
	_, err := board.UpdateTask(p.pmDir, args.TaskID, map[string]any{
		"assignee": args.Assignee,
	})
	if err != nil {
		return nil, err
	}

	board.AppendEvent(p.pmDir, &board.TimelineEvent{
		Event:    "task_assigned",
		Task:     args.TaskID,
		Assignee: args.Assignee,
		By:       "pm",
	})

	return tool.TextResponse(fmt.Sprintf("Assigned %s to %s", args.TaskID, args.Assignee)), nil
}
*/

func (p *PMTools) AddDependency(ctx context.Context, args AddDependencyArgs) (*tool.ToolResponse, error) {
	t, err := board.GetTask(p.pmDir, args.TaskID)
	if err != nil {
		return nil, err
	}

	if !board.ContainsStr(t.BlockedBy, args.DependsOn) {
		t.BlockedBy = append(t.BlockedBy, args.DependsOn)
	}

	// Write updated blocked_by via file rewrite
	content := board.FormatTaskFile(t)
	if err := os.WriteFile(t.FilePath, []byte(content), 0644); err != nil {
		return nil, err
	}

	// Update blocker's blocks list too
	blocker, err := board.GetTask(p.pmDir, args.DependsOn)
	if err == nil && !board.ContainsStr(blocker.Blocks, args.TaskID) {
		blocker.Blocks = append(blocker.Blocks, args.TaskID)
		content := board.FormatTaskFile(blocker)
		os.WriteFile(blocker.FilePath, []byte(content), 0644)
	}

	return tool.TextResponse(fmt.Sprintf("%s is now blocked by %s", args.TaskID, args.DependsOn)), nil
}

func (p *PMTools) RemoveDependency(ctx context.Context, args RemoveDependencyArgs) (*tool.ToolResponse, error) {
	t, err := board.GetTask(p.pmDir, args.TaskID)
	if err != nil {
		return nil, err
	}

	t.BlockedBy = removeStr(t.BlockedBy, args.DependsOn)
	content := board.FormatTaskFile(t)
	if err := os.WriteFile(t.FilePath, []byte(content), 0644); err != nil {
		return nil, err
	}

	return tool.TextResponse(fmt.Sprintf("Removed dependency: %s no longer blocked by %s", args.TaskID, args.DependsOn)), nil
}

/*
func (p *PMTools) ShowBlockers(ctx context.Context, args ShowBlockersArgs) (*tool.ToolResponse, error) {
	if args.TaskID != "" {
		t, err := board.GetTask(p.pmDir, args.TaskID)
		if err != nil {
			return nil, err
		}
		if len(t.BlockedBy) == 0 {
			return tool.TextResponse(fmt.Sprintf("%s has no blockers", t.ID)), nil
		}
		return tool.TextResponse(fmt.Sprintf("%s is blocked by: %s", t.ID, strings.Join(t.BlockedBy, ", "))), nil
	}

	// Show all tasks that have blocked_by relations (regardless of status)
	tasks, err := board.ListTasks(p.pmDir, "", "", "", "")
	if err != nil {
		return nil, err
	}

	var blocked []*board.Task
	for _, t := range tasks {
		if len(t.BlockedBy) > 0 {
			blocked = append(blocked, t)
		}
	}

	if len(blocked) == 0 {
		return tool.TextResponse("No tasks with blockers."), nil
	}

	var b strings.Builder
	b.WriteString("Tasks with blockers:\n")
	for _, t := range blocked {
		b.WriteString(fmt.Sprintf("  %s: %s (%s) blocked by %s\n", t.ID, t.Title, t.Status, strings.Join(t.BlockedBy, ", ")))
	}
	return tool.TextResponse(b.String()), nil
}
*/

func (p *PMTools) GenerateReport(ctx context.Context, args GenerateReportArgs) (*tool.ToolResponse, error) {
	reportType := args.ReportType
	if reportType == "" {
		reportType = "daily"
	}

	report, err := board.GenerateReport(p.pmDir, reportType)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(report), nil
}

func (p *PMTools) Summary(ctx context.Context, args SummaryArgs) (*tool.ToolResponse, error) {
	summary, err := board.GenerateSummary(p.pmDir)
	if err != nil {
		return nil, err
	}
	return tool.TextResponse(summary), nil
}

func (p *PMTools) SearchTasks(ctx context.Context, args SearchTasksArgs) (*tool.ToolResponse, error) {
	query := strings.ToLower(args.Query)

	// Search active tasks
	tasks, err := board.ListTasks(p.pmDir, "", "", "", "")
	if err != nil {
		return nil, err
	}

	var matches []*board.Task
	for _, t := range tasks {
		if strings.Contains(strings.ToLower(t.Title), query) ||
			strings.Contains(strings.ToLower(t.Body), query) ||
			strings.Contains(strings.ToLower(t.ID), query) {
			matches = append(matches, t)
		}
	}

	// Also search archived tasks
	archiveDir := filepath.Join(p.pmDir, "archive")
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
			if strings.Contains(strings.ToLower(t.Title), query) ||
				strings.Contains(strings.ToLower(t.Body), query) ||
				strings.Contains(strings.ToLower(t.ID), query) {
				matches = append(matches, t)
			}
		}
	}

	if len(matches) == 0 {
		return tool.TextResponse(fmt.Sprintf("No tasks matching '%s'", args.Query)), nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d task(s) matching '%s':\n", len(matches), args.Query))
	for _, t := range matches {
		assignee := ""
		if t.Assignee != "" {
			assignee = " → " + t.Assignee
		}
		b.WriteString(fmt.Sprintf("  [%s] %s: %s (%s)%s\n", t.Priority, t.ID, t.Title, t.Status, assignee))
	}
	return tool.TextResponse(b.String()), nil
}

func (p *PMTools) SaveSession(ctx context.Context, args SaveSessionArgs) (*tool.ToolResponse, error) {
	if p.sessionMgr == nil {
		return tool.TextResponse("Session persistence not configured"), nil
	}

	saveID := p.sessionID
	if args.Label != "" {
		saveID = args.Label
	}

	if err := p.sessionMgr.Save(ctx, saveID); err != nil {
		return nil, err
	}
	return tool.TextResponse(fmt.Sprintf("Session saved as '%s'", saveID)), nil
}

func (p *PMTools) ListTimeline(ctx context.Context, args ListTimelineArgs) (*tool.ToolResponse, error) {
	events, err := board.ListEvents(p.pmDir, args.Limit)
	if err != nil {
		// If file doesn't exist yet, return empty
		return tool.TextResponse("No timeline events yet."), nil
	}

	if len(events) == 0 {
		return tool.TextResponse("No timeline events yet."), nil
	}

	var b strings.Builder
	for _, e := range events {
		b.WriteString(fmt.Sprintf("[%s] %s", e.Timestamp, e.Event))
		if e.Task != "" {
			b.WriteString(fmt.Sprintf(" task=%s", e.Task))
		}
		if e.By != "" {
			b.WriteString(fmt.Sprintf(" by=%s", e.By))
		}
		if e.Content != "" {
			b.WriteString(fmt.Sprintf(" %s", e.Content))
		}
		if e.Name != "" {
			b.WriteString(fmt.Sprintf(" name=%s", e.Name))
		}
		if e.From != "" {
			b.WriteString(fmt.Sprintf(" %s→%s", e.From, e.To))
		}
		b.WriteString("\n")
	}
	return tool.TextResponse(b.String()), nil
}

func removeStr(s []string, v string) []string {
	var result []string
	for _, item := range s {
		if item != v {
			result = append(result, item)
		}
	}
	return result
}

func (p *PMTools) RemoveMember(ctx context.Context, args RemoveMemberArgs) (*tool.ToolResponse, error) {
	if err := board.RemoveMember(p.pmDir, args.Name); err != nil {
		return nil, err
	}

	board.AppendEvent(p.pmDir, &board.TimelineEvent{
		Event: "member_removed",
		Name:  args.Name,
		By:    "pm",
	})

	return tool.TextResponse(fmt.Sprintf("Removed member %s", args.Name)), nil
}
