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

func TestCloseCommand(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	issue1 := models.Issue{
		ID:        "test-1",
		Title:     "Blocked Issue",
		Status:    "open",
		BlockedBy: []string{"test-2"},
		CreatedAt: "old",
		UpdatedAt: "old",
	}
	issue2 := models.Issue{
		ID:        "test-2",
		Title:     "Blocker Issue",
		Status:    "open",
		Blocks:    []string{"test-1"},
		CreatedAt: "old",
		UpdatedAt: "old",
	}
	plan1 := models.Plan{
		ID:       "pl-1",
		Roster:   "parent",
		Status:   models.PlanStatusActive,
		Children: []string{"test-2"},
	}

	store.WriteIssues(rostersDir, []models.Issue{issue1, issue2})
	store.WritePlans(rostersDir, []models.Plan{plan1})

	t.Run("closes issue and updates dependencies", func(t *testing.T) {
		err := runClose([]string{"test-2"}, "Done")
		if err != nil {
			t.Fatal(err)
		}

		issues, _ := store.ReadIssues(rostersDir)
		var i1, i2 *models.Issue
		for i := range issues {
			if issues[i].ID == "test-1" {
				i1 = &issues[i]
			}
			if issues[i].ID == "test-2" {
				i2 = &issues[i]
			}
		}

		if i2.Status != "closed" || *i2.CloseReason != "Done" || i2.ClosedAt == nil {
			t.Errorf("issue 2 not correctly closed: %+v", i2)
		}

		if len(i1.BlockedBy) != 0 {
			t.Errorf("issue 1 should no longer be blocked: %+v", i1)
		}

		plans, _ := store.ReadPlans(rostersDir)
		if plans[0].Status != models.PlanStatusDone {
			t.Errorf("plan should be done, got %s", plans[0].Status)
		}
	})

	t.Run("json output", func(t *testing.T) {
		format.SetFormat("json")
		defer format.SetFormat("markdown")

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		runClose([]string{"test-1"}, "")

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		json.Unmarshal(out, &res)
		if res["success"] != true || len(res["closed"].([]any)) != 1 {
			t.Errorf("unexpected json: %s", string(out))
		}
	})

	t.Run("fails for missing issue", func(t *testing.T) {
		err := runClose([]string{"non-existent"}, "")
		if err == nil || !strings.Contains(err.Error(), "issue not found") {
			t.Errorf("expected error for missing issue, got %v", err)
		}
	})
}
