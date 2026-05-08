package plan

import (
	"rosters/pkg/models"
	"testing"
)

func TestComputeNextPlanStatus(t *testing.T) {
	tests := []struct {
		name     string
		plan     models.Plan
		children []models.Issue
		want     models.PlanStatus
	}{
		{
			name: "stays draft",
			plan: models.Plan{Status: models.PlanStatusDraft},
			want: models.PlanStatusDraft,
		},
		{
			name:     "all closed to done",
			plan:     models.Plan{Status: models.PlanStatusApproved},
			children: []models.Issue{{Status: "closed"}, {Status: "closed"}},
			want:     models.PlanStatusDone,
		},
		{
			name:     "one in progress to active",
			plan:     models.Plan{Status: models.PlanStatusApproved},
			children: []models.Issue{{Status: "open"}, {Status: "in_progress"}},
			want:     models.PlanStatusActive,
		},
		{
			name:     "done back to active if child reopens",
			plan:     models.Plan{Status: models.PlanStatusDone},
			children: []models.Issue{{Status: "closed"}, {Status: "open"}},
			want:     models.PlanStatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeNextPlanStatus(tt.plan, tt.children); got != tt.want {
				t.Errorf("ComputeNextPlanStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAffectedPlanIDs(t *testing.T) {
	plans := []models.Plan{
		{ID: "pl1", Children: []string{"iss1", "iss2"}},
		{ID: "pl2", Children: []string{"iss3"}},
	}
	affected := AffectedPlanIDs(plans, []string{"iss2"})
	if len(affected) != 1 || affected[0] != "pl1" {
		t.Errorf("expected [pl1], got %v", affected)
	}
}
