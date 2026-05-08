package plan

import (
	"rosters/pkg/models"
)

func ComputeNextPlanStatus(plan models.Plan, planChildren []models.Issue) models.PlanStatus {
	if plan.Status == models.PlanStatusDraft {
		return models.PlanStatusDraft
	}
	if len(planChildren) == 0 {
		return plan.Status
	}

	allClosed := true
	for _, c := range planChildren {
		if c.Status != "closed" {
			allClosed = false
			break
		}
	}

	if allClosed {
		return models.PlanStatusDone
	}

	if plan.Status == models.PlanStatusActive || plan.Status == models.PlanStatusDone {
		return models.PlanStatusActive
	}

	anyInProgress := false
	for _, c := range planChildren {
		if c.Status == "in_progress" {
			anyInProgress = true
			break
		}
	}

	if anyInProgress {
		return models.PlanStatusActive
	}

	return models.PlanStatusApproved
}

func ApplyPlanTransitions(
	plans []models.Plan,
	allIssues []models.Issue,
	affectedPlanIDs []string,
	now string,
) int {
	targets := make(map[string]bool)
	for _, id := range affectedPlanIDs {
		targets[id] = true
	}

	changed := 0
	for i := range plans {
		p := &plans[i]
		if !targets[p.ID] {
			continue
		}

		var children []models.Issue
		for _, cid := range p.Children {
			for _, iss := range allIssues {
				if iss.ID == cid {
					children = append(children, iss)
					break
				}
			}
		}

		next := ComputeNextPlanStatus(*p, children)
		if next != p.Status {
			p.Status = next
			p.UpdatedAt = now
			changed++
		}
	}
	return changed
}

func AffectedPlanIDs(plans []models.Plan, issueIDs []string) []string {
	ids := make(map[string]bool)
	for _, id := range issueIDs {
		ids[id] = true
	}

	var out []string
	for _, p := range plans {
		hasChild := false
		for _, cid := range p.Children {
			if ids[cid] {
				hasChild = true
				break
			}
		}
		if hasChild {
			out = append(out, p.ID)
		}
	}
	return out
}
