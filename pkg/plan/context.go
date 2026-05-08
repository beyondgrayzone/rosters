package plan

import (
	"fmt"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/store"
)

type PlanContext struct {
	PlansByID     map[string]models.Plan
	PlansByRoster map[string]models.Plan
}

func LoadPlanContext(rostersDir string) (*PlanContext, error) {
	plans, err := store.ReadPlans(rostersDir)
	if err != nil {
		return nil, err
	}

	plansByID := make(map[string]models.Plan)
	plansByRoster := make(map[string]models.Plan)

	for _, p := range plans {
		plansByID[p.ID] = p
		existing, ok := plansByRoster[p.Roster]
		if !ok || p.UpdatedAt > existing.UpdatedAt {
			plansByRoster[p.Roster] = p
		}
	}

	return &PlanContext{
		PlansByID:     plansByID,
		PlansByRoster: plansByRoster,
	}, nil
}

func PlanForIssue(ctx *PlanContext, issue models.Issue) *models.Plan {
	if issue.PlanID == nil {
		return nil
	}
	plan, ok := ctx.PlansByID[*issue.PlanID]
	if !ok {
		return nil
	}
	return &plan
}

func PlanLineSuffix(p *models.Plan) string {
	if p == nil {
		return ""
	}
	if p.Status == models.PlanStatusDraft {
		return fmt.Sprintf(" %s", format.Accent.Sprint("[plan in draft - run rt plan submit]"))
	}
	return fmt.Sprintf(" %s", format.Muted.Sprintf("[plan %s]", p.Status))
}

type ChildSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Adopted bool   `json:"adopted"`
}

func SummarisePlanChildren(plan models.Plan, issues []models.Issue) []ChildSummary {
	adoptedSet := make(map[string]bool)
	for _, id := range plan.AdoptedChildren {
		adoptedSet[id] = true
	}

	var result []ChildSummary
	for _, id := range plan.Children {
		var title, status string
		found := false
		for _, iss := range issues {
			if iss.ID == id {
				title = iss.Title
				status = iss.Status
				found = true
				break
			}
		}
		if !found {
			title = "(missing)"
			status = "missing"
		}
		result = append(result, ChildSummary{
			ID:      id,
			Title:   title,
			Status:  status,
			Adopted: adoptedSet[id],
		})
	}
	return result
}
