package filter

import (
	"rosters/pkg/models"
	"testing"
)

func TestApplyIssueFilters(t *testing.T) {
	alice := "alice"
	issues := []models.Issue{
		{ID: "1", Type: "bug", Priority: 0, Labels: []string{"ui", "fix"}, Assignee: &alice},
		{ID: "2", Type: "task", Priority: 2, Labels: []string{"backend"}},
		{ID: "3", Type: "bug", Priority: 1, Labels: []string{"ui"}},
	}

	t.Run("filter by type", func(t *testing.T) {
		bug := "bug"
		res := ApplyIssueFilters(issues, IssueFilterOptions{Type: &bug})
		if len(res) != 2 {
			t.Errorf("expected 2 bugs, got %d", len(res))
		}
	})

	t.Run("filter by priority max", func(t *testing.T) {
		max := 1
		res := ApplyIssueFilters(issues, IssueFilterOptions{PriorityMax: &max})
		if len(res) != 2 {
			t.Errorf("expected 2 issues with priority <= 1, got %d", len(res))
		}
	})

	t.Run("filter by labels (AND)", func(t *testing.T) {
		lab := "ui,fix"
		res := ApplyIssueFilters(issues, IssueFilterOptions{Label: &lab})
		if len(res) != 1 || res[0].ID != "1" {
			t.Errorf("expected issue 1, got %v", res)
		}
	})

	t.Run("filter by labels (ANY)", func(t *testing.T) {
		lab := "backend,fix"
		res := ApplyIssueFilters(issues, IssueFilterOptions{LabelAny: &lab})
		if len(res) != 2 {
			t.Errorf("expected 2 issues, got %d", len(res))
		}
	})

	t.Run("filter by priority set", func(t *testing.T) {
		pri := map[int]bool{0: true, 2: true}
		res := ApplyIssueFilters(issues, IssueFilterOptions{Priority: pri})
		if len(res) != 2 {
			t.Errorf("expected 2 issues, got %d", len(res))
		}
	})
}
