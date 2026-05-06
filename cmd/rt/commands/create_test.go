package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rosters/pkg/models"
)

func TestCreateCommand(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, models.SeedsDirName)
	os.MkdirAll(rostersDir, 0755)
	os.WriteFile(filepath.Join(rostersDir, models.ConfigFile), []byte("project: test\nversion: \"1\""), 0644)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	t.Run("creates an issue", func(t *testing.T) {
		err := runCreate("My Title", "task", "P1", "", "Some body", "bug,ui")
		if err != nil {
			t.Fatalf("runCreate failed: %v", err)
		}

		issuesBody, _ := os.ReadFile(filepath.Join(rostersDir, models.IssuesFile))
		if !strings.Contains(string(issuesBody), "My Title") {
			t.Error("issue not found in file")
		}
		if !strings.Contains(string(issuesBody), "test-") {
			t.Error("ID prefix missing")
		}
	})

	t.Run("validates priority", func(t *testing.T) {
		err := runCreate("Title", "task", "P9", "", "", "")
		if err == nil {
			t.Error("should have failed with invalid priority")
		}
	})
}
