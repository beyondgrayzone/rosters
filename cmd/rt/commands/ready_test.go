package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/store"
	"rosters/pkg/util"

	"github.com/spf13/cobra"
)

func TestReadyCommand(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	os.WriteFile(filepath.Join(rostersDir, "config.yaml"), []byte("project: test\nversion: \"1\""), 0644)

	i1 := models.Issue{ID: "test-1", Title: "Ready", Status: "open", Priority: 1, CreatedAt: "2023-01-01T00:00:00Z", UpdatedAt: "2023-01-01T00:00:00Z"}
	i2 := models.Issue{ID: "test-2", Title: "Blocked", Status: "open", Priority: 1, BlockedBy: []string{"test-3"}, CreatedAt: "2023-01-01T00:00:01Z", UpdatedAt: "2023-01-01T00:00:01Z"}
	i3 := models.Issue{ID: "test-3", Title: "Open Blocker", Status: "open", Priority: 1, CreatedAt: "2023-01-01T00:00:02Z", UpdatedAt: "2023-01-01T00:00:02Z"}

	store.WriteIssues(rostersDir, []models.Issue{i1, i2, i3})

	setupCmd := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().String("type", "", "")
		cmd.Flags().String("assignee", "", "")
		cmd.Flags().String("label", "", "")
		cmd.Flags().String("label-any", "", "")
		cmd.Flags().Bool("unlabeled", false, "")
		cmd.Flags().String("priority", "", "")
		cmd.Flags().String("priority-max", "", "")
		cmd.Flags().Int("limit", 50, "")
		cmd.Flags().String("sort", "priority", "")
		cmd.Flags().Bool("respect-schedule", false, "")
		return cmd
	}

	t.Run("shows only unblocked open issues", func(t *testing.T) {
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		format.SetFormat("ids")
		runReady(setupCmd(), nil)

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) != 2 {
			t.Errorf("expected 2 ready issues (test-1, test-3), got %d: %v", len(lines), lines)
		}
	})

	t.Run("includes draft plan issues regardless of blockers", func(t *testing.T) {
		i2WithPlan := i2
		i2WithPlan.PlanID = util.Ptr("pl-draft")
		store.WriteIssues(rostersDir, []models.Issue{i1, i2WithPlan, i3})

		plan := models.Plan{
			ID:        "pl-draft",
			Roster:    "test-2",
			Status:    models.PlanStatusDraft,
			UpdatedAt: "2023-01-01T00:00:00Z",
		}
		store.WritePlans(rostersDir, []models.Plan{plan})

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		format.SetFormat("ids")
		runReady(setupCmd(), nil)

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		if !strings.Contains(string(out), "test-2") {
			t.Error("issue with draft plan should be visible even if blocked")
		}
	})

	t.Run("respects schedule logic", func(t *testing.T) {
		future := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
		iScheduled := i1
		iScheduled.Extensions = map[string]any{"scheduledFor": future}

		if !isScheduledOut(iScheduled, time.Now().UnixMilli()) {
			t.Error("isScheduledOut should return true for future schedule")
		}

		iQueued := i1
		iQueued.Extensions = map[string]any{"queued": true}
		if !isScheduledOut(iQueued, time.Now().UnixMilli()) {
			t.Error("isScheduledOut should return true for queued=true")
		}
	})

	t.Run("JSON format output", func(t *testing.T) {
		format.SetFormat("json")
		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		outputReady([]models.Issue{i1}, nil, nil)

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = rescueStdout

		var res map[string]any
		if err := json.Unmarshal(out, &res); err != nil {
			t.Fatal(err)
		}
		if res["count"].(float64) != 1 {
			t.Errorf("expected count 1 in JSON, got %v", res["count"])
		}
	})
}
