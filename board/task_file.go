package board

import (
	"fmt"
	"os"
	"strings"
)

// ReadTaskFile reads and parses a task markdown file
func ReadTaskFile(path string) (*Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	t, body, err := parseFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	t.Body = body
	t.FilePath = path
	return t, nil
}

// FormatTaskFile formats a Task into markdown with YAML frontmatter
func FormatTaskFile(t *Task) string {
	var b strings.Builder

	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("id: %s\n", t.ID))
	b.WriteString(fmt.Sprintf("title: %s\n", yamlQuote(t.Title)))
	b.WriteString(fmt.Sprintf("slug: %s\n", t.Slug))
	b.WriteString(fmt.Sprintf("status: %s\n", t.Status))
	b.WriteString(fmt.Sprintf("priority: %s\n", t.Priority))
	b.WriteString(fmt.Sprintf("assignee: %s\n", t.Assignee))
	b.WriteString(fmt.Sprintf("created: %s\n", t.Created))
	b.WriteString(fmt.Sprintf("updated: %s\n", t.Updated))
	if t.ClosedAt != "" {
		b.WriteString(fmt.Sprintf("closed_at: %s\n", t.ClosedAt))
	}
	b.WriteString(fmt.Sprintf("labels: [%s]\n", strings.Join(t.Labels, ", ")))
	b.WriteString(fmt.Sprintf("blocks: [%s]\n", strings.Join(t.Blocks, ", ")))
	b.WriteString(fmt.Sprintf("blocked_by: [%s]\n", strings.Join(t.BlockedBy, ", ")))
	b.WriteString("---\n")
	if t.Body != "" {
		b.WriteString(t.Body)
	}

	return b.String()
}

func parseFrontmatter(content string) (*Task, string, error) {
	if !strings.HasPrefix(content, "---\n") {
		return nil, content, fmt.Errorf("no frontmatter found")
	}

	rest := content[4:]
	endIdx := strings.Index(rest, "\n---\n")
	if endIdx == -1 {
		if strings.HasSuffix(rest, "\n---") {
			endIdx = len(rest) - 4
		} else {
			return nil, content, fmt.Errorf("unclosed frontmatter")
		}
	}

	fmContent := rest[:endIdx]
	body := ""
	if endIdx+4 < len(rest) {
		body = strings.TrimPrefix(rest[endIdx+4:], "\n")
	}

	t := &Task{}
	for _, line := range strings.Split(fmContent, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])

		switch key {
		case "id":
			t.ID = val
		case "title":
			t.Title = yamlUnquote(val)
		case "slug":
			t.Slug = val
		case "status":
			t.Status = val
		case "priority":
			t.Priority = val
		case "assignee":
			t.Assignee = val
		case "created":
			t.Created = val
		case "updated":
			t.Updated = val
		case "closed_at":
			t.ClosedAt = val
		case "labels":
			t.Labels = parseYAMLList(val)
		case "blocks":
			t.Blocks = parseYAMLList(val)
		case "blocked_by":
			t.BlockedBy = parseYAMLList(val)
		}
	}

	return t, body, nil
}

func parseYAMLList(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func yamlQuote(s string) string {
	if strings.ContainsAny(s, ":#{}[]|>&*!%@`'\"") {
		return fmt.Sprintf("%q", s)
	}
	return s
}

func yamlUnquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
