package commands

import (
	"fmt"
	"strings"

	"rosters/pkg/config"
	"rosters/pkg/filter"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/plan"
	"rosters/pkg/sort"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

func RegisterListCommand(rootCmd *cobra.Command) {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List issues with filters",
		RunE:  runList,
	}

	listCmd.Flags().String("status", "", "Filter by status (open|in_progress|closed)")
	listCmd.Flags().String("type", "", "Filter by type (task|bug|feature|epic)")
	listCmd.Flags().String("assignee", "", "Filter by assignee")
	listCmd.Flags().Bool("all", false, "Include closed issues")
	listCmd.Flags().String("label", "", "Filter: must have ALL labels (comma-separated)")
	listCmd.Flags().String("label-any", "", "Filter: must have any label (comma-separated)")
	listCmd.Flags().Bool("unlabeled", false, "Filter: issues with no labels")
	listCmd.Flags().String("priority", "", "Filter by priority levels (e.g. 0,1 or P0,P1)")
	listCmd.Flags().String("priority-max", "", "Filter to priority <= n")
	listCmd.Flags().Int("limit", 50, "Max issues to show")
	listCmd.Flags().String("sort", "priority", fmt.Sprintf("Sort order (%s)", strings.Join(sort.ValidSortModes, "|")))

	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	allIssues, err := store.ReadIssues(rostersDir)
	if err != nil {
		return err
	}

	closedBlockerIds := make(map[string]bool)
	for _, i := range allIssues {
		if i.Status == "closed" {
			closedBlockerIds[i.ID] = true
		}
	}

	statusFlag, _ := cmd.Flags().GetString("status")
	showAll, _ := cmd.Flags().GetBool("all")
	typeFlag, _ := cmd.Flags().GetString("type")
	assigneeFlag, _ := cmd.Flags().GetString("assignee")
	labelFlag, _ := cmd.Flags().GetString("label")
	labelAnyFlag, _ := cmd.Flags().GetString("label-any")
	unlabeledFlag, _ := cmd.Flags().GetBool("unlabeled")
	priorityFlag, _ := cmd.Flags().GetString("priority")
	priorityMaxFlag, _ := cmd.Flags().GetString("priority-max")
	limit, _ := cmd.Flags().GetInt("limit")
	sortFlag, _ := cmd.Flags().GetString("sort")

	issues := allIssues
	if statusFlag != "" {
		var filtered []models.Issue
		for _, i := range issues {
			if i.Status == statusFlag {
				filtered = append(filtered, i)
			}
		}
		issues = filtered
	} else if !showAll {
		var filtered []models.Issue
		for _, i := range issues {
			if i.Status != "closed" {
				filtered = append(filtered, i)
			}
		}
		issues = filtered
	}

	opts := filter.IssueFilterOptions{
		Unlabeled: unlabeledFlag,
	}
	if typeFlag != "" {
		opts.Type = &typeFlag
	}
	if assigneeFlag != "" {
		opts.Assignee = &assigneeFlag
	}
	if labelFlag != "" {
		opts.Label = &labelFlag
	}
	if labelAnyFlag != "" {
		opts.LabelAny = &labelAnyFlag
	}
	if priorityFlag != "" {
		opts.Priority = make(map[int]bool)
		for _, t := range strings.Split(priorityFlag, ",") {
			if p, err := filter.ParsePriorityToken(t); err == nil {
				opts.Priority[p] = true
			}
		}
	}
	if priorityMaxFlag != "" {
		if p, err := filter.ParsePriorityToken(priorityMaxFlag); err == nil {
			opts.PriorityMax = &p
		}
	}

	issues = filter.ApplyIssueFilters(issues, opts)

	if !sort.IsSortMode(sortFlag) {
		return fmt.Errorf("invalid --sort value: %s", sortFlag)
	}
	issues = sort.SortIssues(issues, sort.SortMode(sortFlag))

	if len(issues) > limit {
		issues = issues[:limit]
	}

	fmtMode := format.GetFormat()
	planCtx, _ := plan.LoadPlanContext(rostersDir)

	switch fmtMode {
	case "json":
		var items []any
		for _, i := range issues {
			item := map[string]any{"issue": i}
			if p := plan.PlanForIssue(planCtx, i); p != nil {
				item["plan_status"] = p.Status
				item["plan_children"] = p.Children
			}
			items = append(items, item)
		}
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "list",
			"issues":  items,
			"count":   len(issues),
		})
	case "ids":
		for _, i := range issues {
			fmt.Println(i.ID)
		}
	case "compact":
		for _, i := range issues {
			fmt.Println(format.FormatIssueOneLineCompact(i, closedBlockerIds))
		}
	case "plain":
		if len(issues) == 0 {
			fmt.Println("No issues found.")
		} else {
			for _, i := range issues {
				p := plan.PlanForIssue(planCtx, i)
				line := format.FormatIssueOneLine(i, closedBlockerIds) + plan.PlanLineSuffix(p)
				fmt.Println(format.StripAnsi(line))
			}
			fmt.Printf("\n%d issue(s)\n", len(issues))
		}
	default:
		if len(issues) == 0 {
			fmt.Println("No issues found.")
		} else {
			for _, i := range issues {
				p := plan.PlanForIssue(planCtx, i)
				suffix := plan.PlanLineSuffix(p)
				if suffix != "" {
					fmt.Printf("%s%s\n", format.FormatIssueOneLine(i, closedBlockerIds), suffix)
				} else {
					format.PrintIssueOneLine(i, closedBlockerIds)
				}
			}
			fmt.Printf("\n%d issue(s)\n", len(issues))
		}
	}

	return nil
}
