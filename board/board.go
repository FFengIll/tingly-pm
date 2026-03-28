package board

import (
	"os"
	"os/exec"
	"path/filepath"
)

// EnsureInit creates the .pm/ directory structure if it doesn't exist.
func EnsureInit(pmDir string) error {
	if _, err := os.Stat(filepath.Join(pmDir, "tasks")); err == nil {
		return nil // Already initialized
	}

	dirs := []string{
		filepath.Join(pmDir, "tasks"),
		filepath.Join(pmDir, "archive"),
		filepath.Join(pmDir, "reports"),
		filepath.Join(pmDir, "sessions"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	// Create empty members.json
	if err := os.WriteFile(filepath.Join(pmDir, "members.json"), []byte("{\"members\":[]}\n"), 0644); err != nil {
		return err
	}

	// Create empty timeline
	if err := os.WriteFile(filepath.Join(pmDir, "timeline.jsonl"), []byte(""), 0644); err != nil {
		return err
	}

	// Init git repo (ignore errors if git not available)
	exec.Command("git", "-C", pmDir, "init").Run()
	exec.Command("git", "-C", pmDir, "add", ".").Run()
	exec.Command("git", "-C", pmDir, "commit", "-m", "init: tingly-pm board").Run()

	return nil
}
