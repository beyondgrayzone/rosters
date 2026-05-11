package commands

import (
	"os"
	"path/filepath"
	"testing"

	"rosters/pkg/models"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

func TestBlockCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)
	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	issues := []models.Issue{
		{ID: "TEST-1", Title: "Issue 1", Status: "open", CreatedAt: "2023-01-01", UpdatedAt: "2023-01-01"},
		{ID: "TEST-2", Title: "Issue 2", Status: "open", CreatedAt: "2023-01-01", UpdatedAt: "2023-01-01"},
	}
	store.WriteIssues(rostersDir, issues)

	rootCmd := &cobra.Command{}
	RegisterBlockCommand(rootCmd)

	t.Run("successfully blocks an issue", func(t *testing.T) {
		rootCmd.SetArgs([]string{"block", "TEST-1", "--by", "TEST-2"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Command failed: %v", err)
		}

		updated, _ := store.ReadIssues(rostersDir)
		var i1, i2 models.Issue
		for _, iss := range updated {
			if iss.ID == "TEST-1" {
				i1 = iss
			}
			if iss.ID == "TEST-2" {
				i2 = iss
			}
		}

		if len(i1.BlockedBy) != 1 || i1.BlockedBy[0] != "TEST-2" {
			t.Errorf("TEST-1 should be blocked by TEST-2, got: %v", i1.BlockedBy)
		}
		if len(i2.Blocks) != 1 || i2.Blocks[0] != "TEST-1" {
			t.Errorf("TEST-2 should block TEST-1, got: %v", i2.Blocks)
		}
	})

	t.Run("fails on missing issue", func(t *testing.T) {
		rootCmd.SetArgs([]string{"block", "NONEXISTENT", "--by", "TEST-2"})
		err := rootCmd.Execute()
		if err == nil {
			t.Error("Expected error for non-existent issue")
		}
	})
}
