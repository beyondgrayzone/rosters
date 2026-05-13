package commands

import (
	"fmt"
	"strings"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/filter"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/plan"
	"rosters/pkg/sort"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

func RegisterReadyCommand(rootCmd *cobra.Command) {
	readyCmd := &cobra.Command{
		Use:   "ready",
		Short: "Show open issues with no unresolved blockers",
		RunE:  runReady,
	}

	readyCmd.Flags().String("type", "", "Filter by type (task|bug|feature|epic)")
	readyCmd.Flags().String("assignee", "", "Filter by assignee")
	readyCmd.Flags().String("label", "", "Filter: must have ALL labels (comma-separated, AND)")
	readyCmd.Flags().String("label-any", "", "Filter: must have any label (comma-separated, OR)")
	readyCmd.Flags().Bool("unlabeled", false, "Filter: issues with no labels")
	readyCmd.Flags().String("priority", "", "Filter by priority (comma-separated, e.g. 0,1 or P0,P1)")
	readyCmd.Flags().String("priority-max", "", "Filter to priority <= n (e.g. --priority-max 1 = P0+P1)")
	readyCmd.Flags().Int("limit", 50, "Max issues to show")
	readyCmd.Flags().String("sort", "priority", "Sort order (priority|created|updated|id)")
	readyCmd.Flags().Bool("respect-schedule", false, "Exclude issues with extensions.queued=true or future extensions.scheduledFor")

	rootCmd.AddCommand(readyCmd)
}

func isScheduledOut(issue models.Issue, now int64) bool {
	if issue.Extensions == nil {
		return false
	}
	if q, ok := issue.Extensions["queued"].(bool); ok && q {
		return true
	}
	if s, ok := issue.Extensions["scheduledFor"].(string); ok {
		t, err := time.Parse(time.RFC3339, s)
		if err == nil && t.UnixMilli() > now {
			return true
		}
	}
	return false
}

func runReady(cmd *cobra.Command, args []string) error {
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	issues, err := store.ReadIssues(rostersDir)
	if err != nil {
		return err
	}

	planCtx, err := plan.LoadPlanContext(rostersDir)
	if err != nil {
		return err
	}

	closedIds := make(map[string]bool)
	for _, i := range issues {
		if i.Status == "closed" {
			closedIds[i.ID] = true
		}
	}

	var ready []models.Issue
	for _, i := range issues {
		if i.Status != "open" {
			continue
		}

		if i.RequiresPlan != nil && *i.RequiresPlan {
			sub, ok := planCtx.PlansByRoster[i.ID]
			if !ok || sub.Status == models.PlanStatusDraft {
				continue
			}
		} else {
			p := plan.PlanForIssue(planCtx, i)
			if p != nil && p.Status == models.PlanStatusDraft {
				ready = append(ready, i)
				continue
			}
		}

		blocked := false
		for _, bid := range i.BlockedBy {
			if !closedIds[bid] {
				blocked = true
				break
			}
		}

		if !blocked {
			ready = append(ready, i)
		}
	}

	fOpts := filter.IssueFilterOptions{}
	limit := 50
	sortMode := "priority"
	respectSchedule := false

	if cmd != nil {
		if val, err := cmd.Flags().GetString("type"); err == nil && val != "" {
			fOpts.Type = &val
		}
		if val, err := cmd.Flags().GetString("assignee"); err == nil && val != "" {
			fOpts.Assignee = &val
		}
		if val, err := cmd.Flags().GetString("label"); err == nil && val != "" {
			fOpts.Label = &val
		}
		if val, err := cmd.Flags().GetString("label-any"); err == nil && val != "" {
			fOpts.LabelAny = &val
		}
		if val, err := cmd.Flags().GetBool("unlabeled"); err == nil {
			fOpts.Unlabeled = val
		}
		if val, err := cmd.Flags().GetString("priority"); err == nil && val != "" {
			fOpts.Priority = make(map[int]bool)
			for _, p := range strings.Split(val, ",") {
				if n, err := filter.ParsePriorityToken(p); err == nil {
					fOpts.Priority[n] = true
				}
			}
		}
		if val, err := cmd.Flags().GetString("priority-max"); err == nil && val != "" {
			if n, err := filter.ParsePriorityToken(val); err == nil {
				fOpts.PriorityMax = &n
			}
		}
		if val, err := cmd.Flags().GetInt("limit"); err == nil {
			limit = val
		}
		if val, err := cmd.Flags().GetString("sort"); err == nil {
			sortMode = val
		}
		if val, err := cmd.Flags().GetBool("respect-schedule"); err == nil {
			respectSchedule = val
		}
	}

	ready = filter.ApplyIssueFilters(ready, fOpts)

	if respectSchedule {
		now := time.Now().UnixMilli()
		var filtered []models.Issue
		for _, i := range ready {
			if !isScheduledOut(i, now) {
				filtered = append(filtered, i)
			}
		}
		ready = filtered
	}

	if !sort.IsSortMode(sortMode) {
		return fmt.Errorf("invalid --sort value: %s. Valid: %s", sortMode, strings.Join(sort.ValidSortModes, "|"))
	}
	ready = sort.SortIssues(ready, sort.SortMode(sortMode))

	if len(ready) > limit {
		ready = ready[:limit]
	}

	return outputReady(ready, closedIds, planCtx)
}

func outputReady(ready []models.Issue, closedIds map[string]bool, ctx *plan.PlanContext) error {
	fmtMode := format.GetFormat()

	if fmtMode == "json" {
		var list []any
		for _, i := range ready {
			item := map[string]any{
				"id":        i.ID,
				"title":     i.Title,
				"status":    i.Status,
				"type":      i.Type,
				"priority":  i.Priority,
				"assignee":  i.Assignee,
				"createdAt": i.CreatedAt,
				"updatedAt": i.UpdatedAt,
			}
			p := plan.PlanForIssue(ctx, i)
			if p != nil {
				item["plan_status"] = p.Status
				item["plan_children"] = p.Children
			}
			list = append(list, item)
		}
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "ready",
			"issues":  list,
			"count":   len(ready),
		})
		return nil
	}

	if len(ready) == 0 {
		if fmtMode != "ids" {
			fmt.Println("No ready issues.")
		}
		return nil
	}

	for _, i := range ready {
		p := plan.PlanForIssue(ctx, i)
		suffix := plan.PlanLineSuffix(p)

		switch fmtMode {
		case "ids":
			fmt.Println(i.ID)
		case "compact":
			fmt.Println(format.FormatIssueOneLineCompact(i, closedIds))
		case "plain":
			fmt.Println(format.StripAnsi(format.FormatIssueOneLine(i, closedIds) + suffix))
		default:
			fmt.Printf("%s%s\n", format.FormatIssueOneLine(i, closedIds), suffix)
		}
	}

	if fmtMode != "ids" && fmtMode != "compact" {
		fmt.Printf("\n%d ready issue(s)\n", len(ready))
	}

	return nil
}
