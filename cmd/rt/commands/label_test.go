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

	"github.com/spf13/cobra"
)

func TestLabelCommands(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)
	issue := models.Issue{ID: "test-1", Title: "Issue One", Status: "open", CreatedAt: "now", UpdatedAt: "now"}
	store.AppendIssue(rostersDir, issue)

	cmd := &cobra.Command{}

	t.Run("add labels", func(t *testing.T) {
		err := runLabelAdd(cmd, []string{"test-1", "backend", "UI"})
		if err != nil {
			t.Fatal(err)
		}

		issues, _ := store.ReadIssues(rostersDir)
		labels := issues[0].Labels
		if len(labels) != 2 || labels[0] != "backend" || labels[1] != "ui" {
			t.Errorf("unexpected labels: %v", labels)
		}
	})

	t.Run("list labels", func(t *testing.T) {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runLabelList(cmd, []string{"test-1"})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "backend") || !strings.Contains(string(out), "ui") {
			t.Errorf("list output missing labels: %s", string(out))
		}
	})

	t.Run("list-all labels", func(t *testing.T) {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runLabelListAll(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "backend") || !strings.Contains(string(out), "ui") {
			t.Errorf("list-all output missing labels: %s", string(out))
		}
		if !strings.Contains(string(out), "2 label(s)") {
			t.Errorf("list-all output missing summary: %s", string(out))
		}
	})

	t.Run("remove labels", func(t *testing.T) {
		err := runLabelRemove(cmd, []string{"test-1", "backend"})
		if err != nil {
			t.Fatal(err)
		}

		issues, _ := store.ReadIssues(rostersDir)
		labels := issues[0].Labels
		if len(labels) != 1 || labels[0] != "ui" {
			t.Errorf("unexpected labels after remove: %v", labels)
		}
	})

	t.Run("json output", func(t *testing.T) {
		format.SetFormat("json")
		defer format.SetFormat("markdown")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runLabelListAll(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		if err := json.Unmarshal(out, &res); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if res["success"] != true || res["command"] != "label list-all" {
			t.Errorf("unexpected json result: %v", res)
		}
	})
}
