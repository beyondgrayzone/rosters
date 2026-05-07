package commands

import (
	"fmt"
	"os"
	"strings"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/plan"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

const humanDivider = "------------------------------------------------------------"

func RegisterShowCommand(rootCmd *cobra.Command) {
	showCmd := &cobra.Command{
		Use:   "show <id> [ids...]",
		Short: "Show one or more issues",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runShow,
	}
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	issues, err := store.ReadIssues(rostersDir)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		return renderSingle(args[0], issues, rostersDir)
	}

	return renderMultiple(args, issues, rostersDir)
}

func renderSingle(id string, issues []models.Issue, rostersDir string) error {
	var issue *models.Issue
	for _, iss := range issues {
		if iss.ID == id {
			issue = &iss
			break
		}
	}

	if issue == nil {
		if strings.HasPrefix(id, "pl-") {
			return renderPlanShow(id, rostersDir)
		}
		return fmt.Errorf("issue not found: %s", id)
	}

	planCtx, _ := plan.LoadPlanContext(rostersDir)
	var p *models.Plan
	var planChildren []plan.ChildSummary
	if planCtx != nil {
		p = plan.PlanForIssue(planCtx, *issue)
		if p != nil {
			planChildren = plan.SummarisePlanChildren(*p, issues)
		}
	}

	if format.GetFormat() == "json" {
		out := map[string]any{
			"success": true,
			"command": "show",
			"issue":   issue,
		}
		if p != nil {
			out["plan"] = map[string]any{
				"id":       p.ID,
				"status":   p.Status,
				"revision": p.Revision,
				"template": p.Template,
				"children": p.Children,
			}
			out["plan_children"] = planChildren
		}
		format.OutputJSON(out)
		return nil
	}

	if format.GetFormat() == "ids" {
		fmt.Println(issue.ID)
		return nil
	}

	output := format.FormatIssueFull(*issue)
	if p != nil {
		var anyChildren []any
		for _, c := range planChildren {
			anyChildren = append(anyChildren, map[string]any{
				"id":      c.ID,
				"title":   c.Title,
				"status":  c.Status,
				"adopted": c.Adopted,
			})
		}
		output += format.RenderPlanBlock(p, anyChildren)
	}

	if format.GetFormat() == "plain" {
		fmt.Print(format.StripAnsi(output))
	} else {
		fmt.Print(output)
	}
	fmt.Println()

	return nil
}

func renderMultiple(ids []string, issues []models.Issue, rostersDir string) error {
	type result struct {
		id    string
		issue *models.Issue
		err   string
	}

	var results []result
	for _, id := range ids {
		var found *models.Issue
		for _, iss := range issues {
			if iss.ID == id {
				found = &iss
				break
			}
		}

		if found != nil {
			results = append(results, result{id: id, issue: found})
		} else if strings.HasPrefix(id, "pl-") {
			results = append(results, result{id: id, err: fmt.Sprintf("Plan id %s not supported in multi-id 'rt show'; use 'rt plan show %s'", id, id)})
		} else {
			results = append(results, result{id: id, err: fmt.Sprintf("Issue not found: %s", id)})
		}
	}

	isJSON := format.GetFormat() == "json"
	anyMissing := false
	for _, r := range results {
		if r.err != "" {
			anyMissing = true
			break
		}
	}

	if isJSON {
		planCtx, _ := plan.LoadPlanContext(rostersDir)
		var items []any
		var errs []any

		for _, r := range results {
			if r.issue != nil {
				item := map[string]any{"issue": r.issue}
				if planCtx != nil {
					p := plan.PlanForIssue(planCtx, *r.issue)
					if p != nil {
						item["plan"] = map[string]any{
							"id":       p.ID,
							"status":   p.Status,
							"revision": p.Revision,
							"template": p.Template,
							"children": p.Children,
						}
						item["plan_children"] = plan.SummarisePlanChildren(*p, issues)
					}
				}
				items = append(items, item)
			} else {
				errs = append(errs, map[string]string{"id": r.id, "error": r.err})
			}
		}

		out := map[string]any{
			"success": !anyMissing,
			"command": "show",
			"results": items,
		}
		if len(errs) > 0 {
			out["errors"] = errs
		}
		format.OutputJSON(out)
		if anyMissing {
			os.Exit(1)
		}
		return nil
	}

	first := true
	for _, r := range results {
		if r.issue != nil {
			if !first {
				fmt.Printf("\n%s\n\n", format.Muted.Sprint(humanDivider))
			}
			first = false

			planCtx, _ := plan.LoadPlanContext(rostersDir)
			output := format.FormatIssueFull(*r.issue)
			if planCtx != nil {
				p := plan.PlanForIssue(planCtx, *r.issue)
				if p != nil {
					children := plan.SummarisePlanChildren(*p, issues)
					var anyChildren []any
					for _, c := range children {
						anyChildren = append(anyChildren, map[string]any{
							"id":      c.ID,
							"title":   c.Title,
							"status":  c.Status,
							"adopted": c.Adopted,
						})
					}
					output += format.RenderPlanBlock(p, anyChildren)
				}
			}

			if format.GetFormat() == "plain" {
				fmt.Print(format.StripAnsi(output))
			} else {
				fmt.Print(output)
			}
			fmt.Println()
		}
	}

	for _, r := range results {
		if r.err != "" {
			format.PrintError(fmt.Sprintf("%s: %s", r.id, r.err))
		}
	}

	if anyMissing {
		os.Exit(1)
	}

	return nil
}

func renderPlanShow(id string, rostersDir string) error {
	plans, err := store.ReadPlans(rostersDir)
	if err != nil {
		return err
	}

	var found *models.Plan
	for _, p := range plans {
		if p.ID == id {
			found = &p
			break
		}
	}

	if found == nil {
		return fmt.Errorf("plan not found: %s", id)
	}

	if format.GetFormat() == "json" {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "plan show",
			"plan":    found,
		})
		return nil
	}

	fmt.Printf("%s  %s  rev %d\n", format.AccentBold(found.ID), format.Brand.Sprint(found.Status), found.Revision)
	fmt.Printf("Roster:   %s\n", format.Accent.Sprint(found.Roster))
	fmt.Printf("Template: %s\n", found.Template)
	return nil
}
