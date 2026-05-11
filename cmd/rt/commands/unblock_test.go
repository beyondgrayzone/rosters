package commands

import (
	"os"
	"path/filepath"
	"testing"

	"rosters/pkg/models"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

func TestUnblockCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)
	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	rootCmd := &cobra.Command{}
	RegisterUnblockCommand(rootCmd)

	t.Run("successfully unblocks an issue from specific blocker", func(t *testing.T) {
		issues := []models.Issue{
			{ID: "T-1", BlockedBy: []string{"T-2"}},
			{ID: "T-2", Blocks: []string{"T-1"}},
		}
		store.WriteIssues(rostersDir, issues)

		rootCmd.SetArgs([]string{"unblock", "T-1", "--from", "T-2"})
		rootCmd.Execute()

		updated, _ := store.ReadIssues(rostersDir)
		for _, iss := range updated {
			if iss.ID == "T-1" && len(iss.BlockedBy) != 0 {
				t.Errorf("T-1 should have no blockers, got: %v", iss.BlockedBy)
			}
			if iss.ID == "T-2" && len(iss.Blocks) != 0 {
				t.Errorf("T-2 should block nothing, got: %v", iss.Blocks)
			}
		}
	})

	t.Run("unblocks all closed issues", func(t *testing.T) {
		issues := []models.Issue{
			{ID: "T-1", BlockedBy: []string{"T-CLOSED", "T-OPEN"}},
			{ID: "T-CLOSED", Status: "closed", Blocks: []string{"T-1"}},
			{ID: "T-OPEN", Status: "open", Blocks: []string{"T-1"}},
		}
		store.WriteIssues(rostersDir, issues)

		rootCmd.SetArgs([]string{"unblock", "T-1", "--all"})
		rootCmd.Execute()

		updated, _ := store.ReadIssues(rostersDir)
		var t1 models.Issue
		for _, iss := range updated {
			if iss.ID == "T-1" {
				t1 = iss
			}
		}

		if len(t1.BlockedBy) != 1 || t1.BlockedBy[0] != "T-OPEN" {
			t.Errorf("T-1 should only be blocked by T-OPEN, got: %v", t1.BlockedBy)
		}
	})
}
