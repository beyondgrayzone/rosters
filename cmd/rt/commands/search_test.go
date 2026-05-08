package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/store"
	"rosters/pkg/util"

	"github.com/spf13/cobra"
)

func TestSearchCommand(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	issue1 := models.Issue{ID: "test-1", Title: "Database connection error", Description: util.Ptr("Happens on startup"), Status: "open", Type: "bug", Priority: 1, CreatedAt: "2023-01-01", UpdatedAt: "2023-01-01"}
	issue2 := models.Issue{ID: "test-2", Title: "Update README", Status: "closed", Type: "task", Priority: 3, CreatedAt: "2023-01-02", UpdatedAt: "2023-01-02"}
	store.AppendIssue(rostersDir, issue1)
	store.AppendIssue(rostersDir, issue2)

	rootCmd := &cobra.Command{}
	RegisterSearchCommand(rootCmd)

	t.Run("basic text search", func(t *testing.T) {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rootCmd.SetArgs([]string{"search", "database"})
		rootCmd.Execute()

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "test-1") || !strings.Contains(string(out), "Database connection error") {
			t.Errorf("output missing match: %s", string(out))
		}
		if strings.Contains(string(out), "test-2") {
			t.Error("output should not contain non-matching issue")
		}
	})

	t.Run("search in description", func(t *testing.T) {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rootCmd.SetArgs([]string{"search", "startup"})
		rootCmd.Execute()

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "test-1") {
			t.Errorf("output missing match in description: %s", string(out))
		}
	})

	t.Run("search with status filter", func(t *testing.T) {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rootCmd.SetArgs([]string{"search", "Update", "--status", "open"})
		rootCmd.Execute()

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if strings.Contains(string(out), "test-2") {
			t.Error("output should not contain closed issue when filtering for open")
		}
	})

	t.Run("json mode returns valid results", func(t *testing.T) {
		format.SetFormat("json")
		defer format.SetFormat("markdown")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rootCmd.SetArgs([]string{"search", "database"})
		rootCmd.Execute()

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		if err := json.Unmarshal(out, &res); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if res["success"] != true || res["count"].(float64) != 1 {
			t.Errorf("unexpected json structure: %v", res)
		}
	})
}
