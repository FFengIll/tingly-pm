package board

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Task represents a task's structured data
type Task struct {
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
	Body      string   `json:"-"`
	FilePath  string   `json:"-"`
}

var (
	ActiveStatuses   = []string{"todo", "in_progress", "blocked", "review"}
	TerminalStatuses = []string{"done", "dropped"}
	AllStatuses      = append(ActiveStatuses, TerminalStatuses...)
)

// GenerateID creates a new task ID from current time
func GenerateID() string {
	return fmt.Sprintf("TASK-%s", time.Now().Format("20060102-150405"))
}

// TaskFilename builds the filename for a task
func TaskFilename(priority, id, slug string) string {
	return fmt.Sprintf("%s-%s-%s.md", priority, id, slug)
}

// CreateTask creates a new task file in tasks/
func CreateTask(pmDir string, t *Task) error {
	if t.ID == "" {
		t.ID = GenerateID()
	}
	if t.Status == "" {
		t.Status = "todo"
	}
	if t.Priority == "" {
		t.Priority = "p1"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if t.Created == "" {
		t.Created = now
	}
	t.Updated = now

	filename := TaskFilename(t.Priority, t.ID, t.Slug)
	t.FilePath = filepath.Join(pmDir, "tasks", filename)

	content := FormatTaskFile(t)
	return os.WriteFile(t.FilePath, []byte(content), 0644)
}

// GetTask reads a task by ID from tasks/ or archive/
func GetTask(pmDir, taskID string) (*Task, error) {
	// Search in tasks/
	task, err := findTaskInDir(filepath.Join(pmDir, "tasks"), taskID)
	if err == nil {
		return task, nil
	}

	// Search in archive/
	archiveDir := filepath.Join(pmDir, "archive")
	entries, _ := os.ReadDir(archiveDir)
	for _, e := range entries {
		if e.IsDir() {
			task, err = findTaskInDir(filepath.Join(archiveDir, e.Name()), taskID)
			if err == nil {
				return task, nil
			}
		}
	}

	return nil, fmt.Errorf("task %s not found", taskID)
}

func findTaskInDir(dir, taskID string) (*Task, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.Contains(e.Name(), taskID) {
			return ReadTaskFile(filepath.Join(dir, e.Name()))
		}
	}
	return nil, fmt.Errorf("task %s not found in %s", taskID, dir)
}

// ListTasks lists tasks from tasks/ with optional filters
func ListTasks(pmDir string, status, assignee, priority, label string) ([]*Task, error) {
	tasksDir := filepath.Join(pmDir, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, err
	}

	var tasks []*Task
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		t, err := ReadTaskFile(filepath.Join(tasksDir, e.Name()))
		if err != nil {
			continue
		}
		if status != "" && t.Status != status {
			continue
		}
		if assignee != "" && t.Assignee != assignee {
			continue
		}
		if priority != "" && t.Priority != priority {
			continue
		}
		if label != "" && !ContainsStr(t.Labels, label) {
			continue
		}
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority < tasks[j].Priority
		}
		return tasks[i].ID < tasks[j].ID
	})

	return tasks, nil
}

// UpdateTask updates a task's fields and rewrites the file
func UpdateTask(pmDir string, taskID string, updates map[string]any) (*Task, error) {
	t, err := GetTask(pmDir, taskID)
	if err != nil {
		return nil, err
	}

	oldPath := t.FilePath

	if v, ok := updates["status"].(string); ok {
		if !ContainsStr(AllStatuses, v) {
			return nil, fmt.Errorf("invalid status: %s", v)
		}
		t.Status = v
	}
	if v, ok := updates["priority"].(string); ok {
		t.Priority = v
	}
	if v, ok := updates["assignee"].(string); ok {
		t.Assignee = v
	}
	if v, ok := updates["labels"].([]string); ok {
		t.Labels = v
	}
	if v, ok := updates["title"].(string); ok {
		t.Title = v
	}
	if v, ok := updates["slug"].(string); ok {
		t.Slug = v
	}

	t.Updated = time.Now().UTC().Format(time.RFC3339)

	newFilename := TaskFilename(t.Priority, t.ID, t.Slug)
	newPath := filepath.Join(filepath.Dir(oldPath), newFilename)

	content := FormatTaskFile(t)
	if newPath != oldPath {
		if err := os.WriteFile(newPath, []byte(content), 0644); err != nil {
			return nil, err
		}
		os.Remove(oldPath)
		t.FilePath = newPath
	} else {
		if err := os.WriteFile(t.FilePath, []byte(content), 0644); err != nil {
			return nil, err
		}
	}

	return t, nil
}

// ArchiveTask moves a task to archive/YYYYMM/
func ArchiveTask(pmDir, taskID, resolution string) (*Task, error) {
	t, err := GetTask(pmDir, taskID)
	if err != nil {
		return nil, err
	}

	if !ContainsStr(TerminalStatuses, resolution) {
		return nil, fmt.Errorf("resolution must be 'done' or 'dropped', got: %s", resolution)
	}

	now := time.Now().UTC()
	t.Status = resolution
	t.ClosedAt = now.Format(time.RFC3339)
	t.Updated = now.Format(time.RFC3339)

	monthDir := filepath.Join(pmDir, "archive", now.Format("200601"))
	if err := os.MkdirAll(monthDir, 0755); err != nil {
		return nil, err
	}

	filename := TaskFilename(t.Priority, t.ID, t.Slug)
	archivePath := filepath.Join(monthDir, filename)
	content := FormatTaskFile(t)
	if err := os.WriteFile(archivePath, []byte(content), 0644); err != nil {
		return nil, err
	}

	os.Remove(t.FilePath)
	t.FilePath = archivePath

	return t, nil
}

// DeleteTask removes a task file
func DeleteTask(pmDir, taskID string) error {
	t, err := GetTask(pmDir, taskID)
	if err != nil {
		return err
	}
	return os.Remove(t.FilePath)
}

func ContainsStr(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}
