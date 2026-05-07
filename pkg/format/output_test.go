package format

import (
	"strings"
	"testing"

	"rosters/pkg/models"
)

func TestStripAnsi(t *testing.T) {
	colored := Brand.Sprint("Success")
	stripped := StripAnsi(colored)
	if stripped != "Success" {
		t.Errorf("expected Success, got %q", stripped)
	}
}

func TestFormatIssueFull(t *testing.T) {
	assignee := "alice"
	issue := models.Issue{
		ID:        "proj-123",
		Title:     "Fix bug",
		Status:    "open",
		Type:      "bug",
		Priority:  1,
		Assignee:  &assignee,
		Labels:    []string{"ui", "high-pri"},
		CreatedAt: "2023-01-01",
		UpdatedAt: "2023-01-02",
	}

	out := StripAnsi(FormatIssueFull(issue))

	expected := []string{
		"proj-123",
		"Fix bug",
		"Assignee: alice",
		"Labels:   ui, high-pri",
		"Type:     bug",
		"Priority: High",
	}

	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("output missing expected string %q", s)
		}
	}
}

func TestRenderPlanBlock(t *testing.T) {
	t.Run("renders draft hint", func(t *testing.T) {
		p := &models.Plan{ID: "pl-1", Status: models.PlanStatusDraft}
		out := StripAnsi(RenderPlanBlock(p, nil))
		if !strings.Contains(out, "plan in draft") {
			t.Errorf("output missing draft hint: %s", out)
		}
	})

	t.Run("renders steps", func(t *testing.T) {
		p := &models.Plan{ID: "pl-1", Status: models.PlanStatusApproved}
		children := []any{
			map[string]any{"id": "c1", "title": "Step 1", "status": "open", "adopted": false},
			map[string]any{"id": "c2", "title": "Step 2", "status": "closed", "adopted": true},
		}
		out := StripAnsi(RenderPlanBlock(p, children))

		if !strings.Contains(out, "Plan steps (2):") {
			t.Error("missing steps header")
		}
		if !strings.Contains(out, "c1") || !strings.Contains(out, "Step 1") {
			t.Error("missing child 1")
		}
		if !strings.Contains(out, "(adopted)") {
			t.Error("missing adopted tag")
		}
	})
}
