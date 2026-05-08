package sort

import (
	"rosters/pkg/models"
	"testing"
)

func TestSortIssues(t *testing.T) {
	issues := []models.Issue{
		{ID: "B", Priority: 2, CreatedAt: "2023-01-01"},
		{ID: "A", Priority: 1, CreatedAt: "2023-01-02"},
		{ID: "C", Priority: 2, CreatedAt: "2023-01-03"},
	}

	t.Run("sort by priority", func(t *testing.T) {
		res := SortIssues(issues, SortPriority)
		if res[0].ID != "A" {
			t.Errorf("expected A first, got %s", res[0].ID)
		}
		// Same priority should sort by CreatedAt desc
		if res[1].ID != "C" {
			t.Errorf("expected C second (newer), got %s", res[1].ID)
		}
	})

	t.Run("sort by id", func(t *testing.T) {
		res := SortIssues(issues, SortID)
		if res[0].ID != "A" || res[1].ID != "B" || res[2].ID != "C" {
			t.Errorf("id sort order failed: %v", res)
		}
	})
}
