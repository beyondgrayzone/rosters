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
)

func TestShowCommand(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	issue1 := models.Issue{ID: "test-a1", Title: "Issue One", Status: "open", Priority: 2, CreatedAt: "now", UpdatedAt: "now"}
	issue2 := models.Issue{ID: "test-b2", Title: "Issue Two", Status: "closed", Priority: 1, CreatedAt: "now", UpdatedAt: "now"}
	store.AppendIssue(rostersDir, issue1)
	store.AppendIssue(rostersDir, issue2)

	t.Run("shows single issue", func(t *testing.T) {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := renderSingle("test-a1", []models.Issue{issue1, issue2}, rostersDir)

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), "test-a1") || !strings.Contains(string(out), "Issue One") {
			t.Errorf("output missing issue details: %s", string(out))
		}
	})

	t.Run("shows multiple issues", func(t *testing.T) {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := renderMultiple([]string{"test-a1", "test-b2"}, []models.Issue{issue1, issue2}, rostersDir)

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), "test-a1") || !strings.Contains(string(out), "test-b2") {
			t.Errorf("output missing one of the issues: %s", string(out))
		}
		if !strings.Contains(string(out), humanDivider) {
			t.Error("output missing divider between multiple issues")
		}
	})

	t.Run("json mode returns valid json", func(t *testing.T) {
		format.SetFormat("json")
		defer format.SetFormat("markdown")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		renderSingle("test-a1", []models.Issue{issue1}, rostersDir)

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		if err := json.Unmarshal(out, &res); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}
		if res["success"] != true || res["command"] != "show" {
			t.Errorf("unexpected json structure: %v", res)
		}
	})

	t.Run("shows plan block for issue with plan", func(t *testing.T) {
		format.SetFormat("markdown")
		planID := "pl-123"
		issueWithPlan := issue1
		issueWithPlan.PlanID = util.Ptr(planID)

		planData := models.Plan{
			ID:       planID,
			Roster:   "test-a1",
			Status:   models.PlanStatusApproved,
			Children: []string{"test-b2"},
		}
		b, _ := json.Marshal(planData)
		os.WriteFile(filepath.Join(rostersDir, "plans.jsonl"), append(b, '\n'), 0644)

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		renderSingle("test-a1", []models.Issue{issueWithPlan, issue2}, rostersDir)

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "Plan:") || !strings.Contains(string(out), "pl-123") {
			t.Errorf("output missing plan information: %s", string(out))
		}
	})
}
