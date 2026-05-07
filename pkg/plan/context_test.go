package plan

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"rosters/pkg/models"
	"rosters/pkg/util"
)

func TestPlanContext(t *testing.T) {
	tmpDir := t.TempDir()

	// Create issues file
	i1 := models.Issue{ID: "iss-1", Title: "Parent", Status: "open", PlanID: util.Ptr("pl-1")}
	i2 := models.Issue{ID: "iss-2", Title: "Child", Status: "open"}
	issuesPath := filepath.Join(tmpDir, "issues.jsonl")
	d1, _ := json.Marshal(i1)
	d2, _ := json.Marshal(i2)
	os.WriteFile(issuesPath, append(append(d1, '\n'), append(d2, '\n')...), 0644)

	// Create plans file
	p1 := models.Plan{
		ID:        "pl-1",
		Roster:    "iss-1",
		Status:    models.PlanStatusApproved,
		Children:  []string{"iss-2"},
		UpdatedAt: "2023-01-01",
	}
	plansPath := filepath.Join(tmpDir, "plans.jsonl")
	dp, _ := json.Marshal(p1)
	os.WriteFile(plansPath, append(dp, '\n'), 0644)

	ctx, err := LoadPlanContext(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("loads plans accurately", func(t *testing.T) {
		if len(ctx.PlansByID) != 1 || ctx.PlansByID["pl-1"].Roster != "iss-1" {
			t.Error("PlansByID mapping failed")
		}
		if len(ctx.PlansByRoster) != 1 || ctx.PlansByRoster["iss-1"].ID != "pl-1" {
			t.Error("PlansByRoster mapping failed")
		}
	})

	t.Run("PlanForIssue returns correct plan", func(t *testing.T) {
		p := PlanForIssue(ctx, i1)
		if p == nil || p.ID != "pl-1" {
			t.Error("PlanForIssue failed to find plan")
		}

		pNone := PlanForIssue(ctx, i2)
		if pNone != nil {
			t.Error("PlanForIssue should return nil for issue without PlanID")
		}
	})

	t.Run("SummarisePlanChildren handles missing and adopted status", func(t *testing.T) {
		plan := models.Plan{
			Children:        []string{"iss-2", "iss-missing"},
			AdoptedChildren: []string{"iss-2"},
		}

		summaries := SummarisePlanChildren(plan, []models.Issue{i2})

		if len(summaries) != 2 {
			t.Fatalf("expected 2 summaries, got %d", len(summaries))
		}

		// Check iss-2 (found and adopted)
		if summaries[0].ID != "iss-2" || !summaries[0].Adopted || summaries[0].Status != "open" {
			t.Errorf("iss-2 summary incorrect: %+v", summaries[0])
		}

		// Check iss-missing (not found)
		if summaries[1].ID != "iss-missing" || summaries[1].Status != "missing" {
			t.Errorf("iss-missing summary incorrect: %+v", summaries[1])
		}
	})
}
