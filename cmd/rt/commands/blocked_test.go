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

func TestBlockedCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)
	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	issues := []models.Issue{
		{ID: "TEST-1", Title: "Open Blocker", Status: "open"},
		{ID: "TEST-2", Title: "Closed Blocker", Status: "closed"},
		{ID: "TEST-3", Title: "Blocked by Open", Status: "open", BlockedBy: []string{"TEST-1"}},
		{ID: "TEST-4", Title: "Blocked by Closed", Status: "open", BlockedBy: []string{"TEST-2"}},
		{ID: "TEST-5", Title: "Closed but has blocker", Status: "closed", BlockedBy: []string{"TEST-1"}},
	}
	store.WriteIssues(rostersDir, issues)

	cmd := &cobra.Command{}

	t.Run("lists blocked issues in default format", func(t *testing.T) {
		format.SetFormat("markdown")
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runBlocked(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		output := string(out)
		if !strings.Contains(output, "TEST-3") {
			t.Errorf("expected TEST-3 in output, got: %s", output)
		}
		if strings.Contains(output, "TEST-4") {
			t.Errorf("did not expect TEST-4 in blocked list, got: %s", output)
		}
		if strings.Contains(output, "TEST-5") {
			t.Errorf("did not expect closed issue TEST-5 in blocked list, got: %s", output)
		}
		if !strings.Contains(output, "1 blocked issue(s)") {
			t.Errorf("footer count incorrect, got: %s", output)
		}
	})

	t.Run("lists IDs in ids format", func(t *testing.T) {
		format.SetFormat("ids")
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runBlocked(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		output := strings.TrimSpace(string(out))
		if output != "TEST-3" {
			t.Errorf("expected only TEST-3 ID, got: %q", output)
		}
	})

	t.Run("outputs JSON", func(t *testing.T) {
		format.SetFormat("json")
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runBlocked(cmd, []string{})

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		json.Unmarshal(out, &res)

		if res["count"].(float64) != 1 {
			t.Errorf("expected count 1, got %v", res["count"])
		}
		issuesArr := res["issues"].([]any)
		if len(issuesArr) != 1 || issuesArr[0].(map[string]any)["id"] != "TEST-3" {
			t.Errorf("unexpected issues in JSON: %v", issuesArr)
		}
	})
}
