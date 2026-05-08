package commands

import (
	"os"
	"path/filepath"
	"testing"

	"rosters/pkg/models"
	"rosters/pkg/store"
)

func TestUpdateCommand(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	issue := models.Issue{ID: "test-1", Title: "Original", Status: "open", Priority: 2}
	store.AppendIssue(rostersDir, issue)

	t.Run("updates basic fields", func(t *testing.T) {
		opts := updateOptions{
			title:    "New Title",
			priority: "P1",
			status:   "in_progress",
		}
		err := runUpdate("test-1", opts)
		if err != nil {
			t.Fatal(err)
		}

		issues, _ := store.ReadIssues(rostersDir)
		if issues[0].Title != "New Title" || issues[0].Priority != 1 || issues[0].Status != "in_progress" {
			t.Errorf("fields not updated correctly: %+v", issues[0])
		}
	})

	t.Run("clears closed metadata on reopen", func(t *testing.T) {
		closedAt := "some-time"
		reason := "done"
		issue2 := models.Issue{ID: "test-2", Status: "closed", ClosedAt: &closedAt, CloseReason: &reason}
		store.AppendIssue(rostersDir, issue2)

		runUpdate("test-2", updateOptions{status: "open"})

		issues, _ := store.ReadIssues(rostersDir)
		var updated *models.Issue
		for _, iss := range issues {
			if iss.ID == "test-2" {
				updated = &iss
			}
		}
		if updated.ClosedAt != nil || updated.CloseReason != nil {
			t.Error("closed metadata should be cleared")
		}
	})

	t.Run("handles labels", func(t *testing.T) {
		runUpdate("test-1", updateOptions{addLabel: "a,b"})
		issues, _ := store.ReadIssues(rostersDir)
		if len(issues[0].Labels) != 2 {
			t.Errorf("expected 2 labels, got %v", issues[0].Labels)
		}

		runUpdate("test-1", updateOptions{removeLabel: "a"})
		issues, _ = store.ReadIssues(rostersDir)
		if len(issues[0].Labels) != 1 || issues[0].Labels[0] != "b" {
			t.Errorf("expected label [b], got %v", issues[0].Labels)
		}
	})

	t.Run("merges extensions", func(t *testing.T) {
		runUpdate("test-1", updateOptions{extensions: `{"foo": "bar"}`})
		issues, _ := store.ReadIssues(rostersDir)
		if issues[0].Extensions["foo"] != "bar" {
			t.Error("extension not merged")
		}
	})
}
