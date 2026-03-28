package board

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestBoard(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	pmDir := filepath.Join(dir, ".pm")
	if err := EnsureInit(pmDir); err != nil {
		t.Fatal(err)
	}
	return pmDir
}

func TestEnsureInit(t *testing.T) {
	dir := t.TempDir()
	pmDir := filepath.Join(dir, ".pm")

	err := EnsureInit(pmDir)
	if err != nil {
		t.Fatalf("EnsureInit failed: %v", err)
	}

	// Check directories
	for _, sub := range []string{"tasks", "archive", "reports"} {
		if _, err := os.Stat(filepath.Join(pmDir, sub)); os.IsNotExist(err) {
			t.Errorf("directory %s not created", sub)
		}
	}

	// Check files
	for _, f := range []string{"members.json", "timeline.jsonl"} {
		if _, err := os.Stat(filepath.Join(pmDir, f)); os.IsNotExist(err) {
			t.Errorf("file %s not created", f)
		}
	}

	// Idempotent
	if err := EnsureInit(pmDir); err != nil {
		t.Errorf("EnsureInit should be idempotent: %v", err)
	}
}

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if len(id) < 10 {
		t.Errorf("ID too short: %s", id)
	}
	if id[:5] != "TASK-" {
		t.Errorf("ID should start with TASK-: %s", id)
	}
}

func TestTaskFilename(t *testing.T) {
	got := TaskFilename("p0", "TASK-20260327-143022", "jwt-token-refresh")
	want := "p0-TASK-20260327-143022-jwt-token-refresh.md"
	if got != want {
		t.Errorf("TaskFilename = %q, want %q", got, want)
	}
}

func TestCreateAndGetTask(t *testing.T) {
	pmDir := setupTestBoard(t)

	task := &Task{
		Title:    "Test Task",
		Slug:     "test-task",
		Priority: "p1",
		Assignee: "agent-1",
		Labels:   []string{"test"},
	}

	err := CreateTask(pmDir, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if task.ID == "" {
		t.Fatal("task ID should be set")
	}
	if task.Created == "" {
		t.Fatal("task Created should be set")
	}

	// Get it back
	got, err := GetTask(pmDir, task.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if got.Title != "Test Task" {
		t.Errorf("title = %q, want %q", got.Title, "Test Task")
	}
	if got.Priority != "p1" {
		t.Errorf("priority = %q, want %q", got.Priority, "p1")
	}
	if got.Status != "todo" {
		t.Errorf("status = %q, want %q", got.Status, "todo")
	}
}

func TestListTasks(t *testing.T) {
	pmDir := setupTestBoard(t)

	CreateTask(pmDir, &Task{Title: "Task A", Slug: "task-a", Priority: "p0"})
	CreateTask(pmDir, &Task{Title: "Task B", Slug: "task-b", Priority: "p1", Assignee: "agent-1"})

	tasks, err := ListTasks(pmDir, "", "", "", "")
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}

	// Sorted by priority
	if tasks[0].Priority != "p0" {
		t.Errorf("first task priority = %q, want p0", tasks[0].Priority)
	}

	// Filter by assignee
	filtered, _ := ListTasks(pmDir, "", "agent-1", "", "")
	if len(filtered) != 1 {
		t.Errorf("filtered count = %d, want 1", len(filtered))
	}
}

func TestUpdateTask(t *testing.T) {
	pmDir := setupTestBoard(t)

	task := &Task{Title: "Original", Slug: "original", Priority: "p1"}
	CreateTask(pmDir, task)

	// Update status
	updated, err := UpdateTask(pmDir, task.ID, map[string]any{"status": "in_progress"})
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}
	if updated.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", updated.Status)
	}

	// Update priority (should rename file)
	updated, err = UpdateTask(pmDir, task.ID, map[string]any{"priority": "p0"})
	if err != nil {
		t.Fatalf("UpdateTask with priority change failed: %v", err)
	}
	if updated.Priority != "p0" {
		t.Errorf("priority = %q, want p0", updated.Priority)
	}

	// Verify file renamed
	filename := filepath.Base(updated.FilePath)
	if filename[:2] != "p0" {
		t.Errorf("file should start with p0: %s", filename)
	}
}

func TestArchiveTask(t *testing.T) {
	pmDir := setupTestBoard(t)

	task := &Task{Title: "Archive Me", Slug: "archive-me", Priority: "p1"}
	CreateTask(pmDir, task)

	archived, err := ArchiveTask(pmDir, task.ID, "done")
	if err != nil {
		t.Fatalf("ArchiveTask failed: %v", err)
	}
	if archived.Status != "done" {
		t.Errorf("status = %q, want done", archived.Status)
	}
	if archived.ClosedAt == "" {
		t.Error("closed_at should be set")
	}

	// Should not be in tasks/ anymore
	_, err = GetTask(pmDir, task.ID)
	// GetTask also searches archive, so we should still find it
	// Check it's in archive
	if _, err := os.Stat(filepath.Join(pmDir, "tasks")); err == nil {
		entries, _ := os.ReadDir(filepath.Join(pmDir, "tasks"))
		for _, e := range entries {
			if filepath.Ext(e.Name()) == ".md" {
				t.Errorf("task should be removed from tasks/, found: %s", e.Name())
			}
		}
	}

	// Verify it's still findable (in archive)
	found, err := GetTask(pmDir, task.ID)
	if err != nil {
		t.Fatalf("should find task in archive: %v", err)
	}
	if found.Status != "done" {
		t.Errorf("archived task status = %q, want done", found.Status)
	}
}

func TestMembers(t *testing.T) {
	pmDir := setupTestBoard(t)

	err := RegisterMember(pmDir, "yz", "human", []string{"backend"})
	if err != nil {
		t.Fatalf("RegisterMember failed: %v", err)
	}

	// Duplicate
	err = RegisterMember(pmDir, "yz", "human", nil)
	if err == nil {
		t.Error("should reject duplicate member")
	}

	// Invalid type
	err = RegisterMember(pmDir, "x", "invalid", nil)
	if err == nil {
		t.Error("should reject invalid member type")
	}

	members, _ := ListMembers(pmDir, "")
	if len(members) != 1 {
		t.Fatalf("len(members) = %d, want 1", len(members))
	}

	agents, _ := ListMembers(pmDir, "agent")
	if len(agents) != 0 {
		t.Errorf("no agents expected, got %d", len(agents))
	}
}

func TestTimeline(t *testing.T) {
	pmDir := setupTestBoard(t)

	AppendEvent(pmDir, &TimelineEvent{
		Event: "task_created",
		Task:  "TASK-001",
		By:    "yz",
	})

	events, err := ReadTimeline(pmDir)
	if err != nil {
		t.Fatalf("ReadTimeline failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].Event != "task_created" {
		t.Errorf("event = %q, want task_created", events[0].Event)
	}
}

func TestTaskFileParsing(t *testing.T) {
	content := `---
id: TASK-20260327-143022
title: "实现JWT令牌刷新"
slug: jwt-token-refresh
status: in_progress
priority: p0
assignee: agent-1
created: 2026-03-27T14:30:22Z
updated: 2026-03-27T14:30:22Z
labels: [auth, backend]
blocks: [TASK-002]
blocked_by: [TASK-003]
---

## Description

This is a test.
`

	task, body, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("parseFrontmatter failed: %v", err)
	}

	if task.ID != "TASK-20260327-143022" {
		t.Errorf("id = %q", task.ID)
	}
	if task.Title != "实现JWT令牌刷新" {
		t.Errorf("title = %q", task.Title)
	}
	if task.Status != "in_progress" {
		t.Errorf("status = %q", task.Status)
	}
	if task.Priority != "p0" {
		t.Errorf("priority = %q", task.Priority)
	}
	if len(task.Labels) != 2 || task.Labels[0] != "auth" {
		t.Errorf("labels = %v", task.Labels)
	}
	if len(task.Blocks) != 1 || task.Blocks[0] != "TASK-002" {
		t.Errorf("blocks = %v", task.Blocks)
	}
	if len(task.BlockedBy) != 1 || task.BlockedBy[0] != "TASK-003" {
		t.Errorf("blocked_by = %v", task.BlockedBy)
	}
	if body == "" {
		t.Error("body should not be empty")
	}
	// Body may have leading newline from the double-\n after ---\n
	if !strings.Contains(body, "## Description") {
		t.Errorf("body should contain '## Description', got: %q", body)
	}
}

func TestFormatAndParseRoundTrip(t *testing.T) {
	original := &Task{
		ID:        "TASK-20260327-143022",
		Title:     "测试任务",
		Slug:      "test-task",
		Status:    "in_progress",
		Priority:  "p0",
		Assignee:  "agent-1",
		Created:   "2026-03-27T14:30:22Z",
		Updated:   "2026-03-27T14:30:22Z",
		Labels:    []string{"auth"},
		Blocks:    []string{"TASK-002"},
		BlockedBy: []string{"TASK-003"},
		Body:      "## Description\n\nTest body.\n",
	}

	formatted := FormatTaskFile(original)
	parsed, body, err := parseFrontmatter(formatted)
	if err != nil {
		t.Fatalf("round trip parse failed: %v", err)
	}

	if parsed.ID != original.ID {
		t.Errorf("id mismatch: %q vs %q", parsed.ID, original.ID)
	}
	if parsed.Title != original.Title {
		t.Errorf("title mismatch: %q vs %q", parsed.Title, original.Title)
	}
	if parsed.Status != original.Status {
		t.Errorf("status mismatch: %q vs %q", parsed.Status, original.Status)
	}
	if parsed.Priority != original.Priority {
		t.Errorf("priority mismatch: %q vs %q", parsed.Priority, original.Priority)
	}
	if body != original.Body {
		t.Errorf("body mismatch:\ngot:  %q\nwant: %q", body, original.Body)
	}
}

func TestGenerateSummary(t *testing.T) {
	pmDir := setupTestBoard(t)

	CreateTask(pmDir, &Task{Title: "Task A", Slug: "task-a", Priority: "p0", Assignee: "agent-1"})
	CreateTask(pmDir, &Task{Title: "Task B", Slug: "task-b", Priority: "p1"})

	summary, err := GenerateSummary(pmDir)
	if err != nil {
		t.Fatalf("GenerateSummary failed: %v", err)
	}

	if summary == "" {
		t.Error("summary should not be empty")
	}
}

func TestGenerateReport(t *testing.T) {
	pmDir := setupTestBoard(t)

	report, err := GenerateReport(pmDir, "daily")
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	if report == "" {
		t.Error("report should not be empty")
	}
}

func TestInvalidStatus(t *testing.T) {
	pmDir := setupTestBoard(t)

	_, err := UpdateTask(pmDir, "TASK-00000000-000000", map[string]any{"status": "invalid"})
	if err == nil {
		t.Error("should reject invalid status")
	}
}

func TestInvalidArchiveResolution(t *testing.T) {
	pmDir := setupTestBoard(t)

	task := &Task{Title: "Test", Slug: "test"}
	CreateTask(pmDir, task)

	_, err := ArchiveTask(pmDir, task.ID, "invalid")
	if err == nil {
		t.Error("should reject invalid resolution")
	}
}
