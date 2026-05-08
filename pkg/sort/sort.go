package sort

import (
	"sort"

	"rosters/pkg/models"
)

type SortMode string

const (
	SortPriority SortMode = "priority"
	SortCreated  SortMode = "created"
	SortUpdated  SortMode = "updated"
	SortID       SortMode = "id"
)

var ValidSortModes = []string{"priority", "created", "updated", "id"}

func IsSortMode(v string) bool {
	for _, m := range ValidSortModes {
		if m == v {
			return true
		}
	}
	return false
}

func SortIssues(issues []models.Issue, mode SortMode) []models.Issue {
	sorted := make([]models.Issue, len(issues))
	copy(sorted, issues)

	sort.SliceStable(sorted, func(i, j int) bool {
		switch mode {
		case SortPriority:
			if sorted[i].Priority != sorted[j].Priority {
				return sorted[i].Priority < sorted[j].Priority
			}
			return sorted[i].CreatedAt > sorted[j].CreatedAt
		case SortCreated:
			return sorted[i].CreatedAt > sorted[j].CreatedAt
		case SortUpdated:
			return sorted[i].UpdatedAt > sorted[j].UpdatedAt
		case SortID:
			return sorted[i].ID < sorted[j].ID
		default:
			return false
		}
	})

	return sorted
}
