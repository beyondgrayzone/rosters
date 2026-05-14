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
)

func TestPlanListCommand(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	plan1 := models.Plan{ID: "pl-1", Roster: "s1", Template: "feature", Status: "approved", CreatedAt: "2023-01-02T10:00:00Z"}
	plan2 := models.Plan{ID: "pl-2", Roster: "s2", Template: "bug", Status: "draft", CreatedAt: "2023-01-01T10:00:00Z"}
	store.WritePlans(rostersDir, []models.Plan{plan1, plan2})

	t.Run("list with status filter", func(t *testing.T) {
		cmd := listPlansCmd()
		cmd.SetArgs([]string{"--status", "approved"})

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd.Execute()

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "pl-1") || strings.Contains(string(out), "pl-2") {
			t.Errorf("list output incorrect for status filter: %s", string(out))
		}
	})

	t.Run("list as json", func(t *testing.T) {
		format.SetFormat("json")
		defer format.SetFormat("markdown")

		cmd := listPlansCmd()
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd.Execute()

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		json.Unmarshal(out, &res)
		if res["count"].(float64) != 2 {
			t.Errorf("expected 2 plans in json, got %v", res["count"])
		}
	})
}
