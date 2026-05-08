package commands

import (
	"os"
	"path/filepath"
	"testing"

	"rosters/pkg/models"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

func TestListCommand(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	i1 := models.Issue{ID: "test-1", Title: "Issue 1", Status: "open", Type: "bug", Priority: 1}
	i2 := models.Issue{ID: "test-2", Title: "Issue 2", Status: "closed", Type: "task", Priority: 2}
	store.AppendIssue(rostersDir, i1)
	store.AppendIssue(rostersDir, i2)

	rootCmd := &cobra.Command{}
	RegisterListCommand(rootCmd)

	t.Run("lists open issues by default", func(t *testing.T) {
		cmd := rootCmd.Commands()[0]
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		cmd := rootCmd.Commands()[0]
		cmd.SetArgs([]string{"--status", "closed"})
		err := cmd.Execute()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("filters by priority-max", func(t *testing.T) {
		cmd := rootCmd.Commands()[0]
		cmd.SetArgs([]string{"--priority-max", "1"})
		err := cmd.Execute()
		if err != nil {
			t.Fatal(err)
		}
	})
}
