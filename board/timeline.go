package board

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TimelineEvent represents a single event in the timeline
type TimelineEvent struct {
	Timestamp string `json:"ts"`
	Event     string `json:"event"`
	Task      string `json:"task,omitempty"`
	By        string `json:"by,omitempty"`
	Assignee  string `json:"assignee,omitempty"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Content   string `json:"content,omitempty"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Path      string `json:"path,omitempty"`
	Status    string `json:"status,omitempty"`
}

// AppendEvent appends an event to timeline.jsonl
func AppendEvent(pmDir string, event *TimelineEvent) error {
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(pmDir, "timeline.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}

// ReadTimeline reads all events from timeline.jsonl
func ReadTimeline(pmDir string) ([]TimelineEvent, error) {
	data, err := os.ReadFile(filepath.Join(pmDir, "timeline.jsonl"))
	if err != nil {
		return nil, err
	}

	var events []TimelineEvent
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		var e TimelineEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, nil
}
