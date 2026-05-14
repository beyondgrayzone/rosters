package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/plan"
	"rosters/pkg/store"
	"rosters/pkg/util"

	"github.com/spf13/cobra"
)

func RegisterPlanCommand(rootCmd *cobra.Command) {
	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Plan management",
	}

	planCmd.AddCommand(templatesCmd())
	planCmd.AddCommand(promptCmd())
	planCmd.AddCommand(submitCmd())
	planCmd.AddCommand(showPlanCmd())
	planCmd.AddCommand(listPlansCmd())
	planCmd.AddCommand(adoptCmd())
	planCmd.AddCommand(releaseCmd())
	planCmd.AddCommand(validatePlanCmd())
	planCmd.AddCommand(outcomePlanCmd())
	planCmd.AddCommand(reviewPlanCmd())

	planCmd.Run = func(cmd *cobra.Command, args []string) {
		cmd.Help()
		os.Exit(1)
	}

	rootCmd.AddCommand(planCmd)
}

func templatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "templates",
		Short: "List available plan templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := config.FindRostersDir("")
			tpls, _ := config.LoadPlanTemplates(dir)

			var names []string
			for k := range tpls {
				names = append(names, k)
			}
			sort.Strings(names)

			var items []map[string]string
			for _, n := range names {
				desc := ""
				if tpls[n].Description != nil {
					desc = *tpls[n].Description
				}
				items = append(items, map[string]string{"name": n, "description": desc})
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{
					"success":   true,
					"command":   "plan templates",
					"templates": items,
					"count":     len(items),
				})
				return nil
			}

			format.Brand.Println("Available templates:")
			for _, item := range items {
				desc := ""
				if item["description"] != "" {
					desc = "  " + format.Muted.Sprint(item["description"])
				}
				fmt.Printf("  %s%s\n", format.AccentBold(item["name"]), desc)
			}
			return nil
		},
	}
}

func promptCmd() *cobra.Command {
	var tplOverride, domOverride string
	c := &cobra.Command{
		Use:   "prompt <roster-id>",
		Short: "Emit structured planning prompt JSON for a roster",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := config.FindRostersDir("")
			issues, _ := store.ReadIssues(dir)
			id := args[0]
			var s *models.Issue
			for _, iss := range issues {
				if iss.ID == id {
					s = &iss
					break
				}
			}
			if s == nil {
				return fmt.Errorf("roster not found: %s", id)
			}

			tpls, _ := config.LoadPlanTemplates(dir)
			name := tplOverride
			if name == "" {
				name = "feature"
				if s.Type == "bug" {
					name = "bug"
				}
			}

			t, ok := tpls[name]
			if !ok {
				return fmt.Errorf("unknown template: %s", name)
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{
					"plan_request": map[string]any{
						"roster":       id,
						"template":     name,
						"instructions": "Fill every section...",
						"sections":     t.Sections,
					},
				})
			} else {
				format.Brand.Printf("Plan prompt for %s\n", id)
				fmt.Printf("Template: %s\n\n", name)
				for k, spec := range t.Sections {
					status := "optional"
					if spec.Required {
						status = "required"
					}
					fmt.Printf("  %s (%v) %s\n", format.AccentBold(k), spec.Kind, status)
					fmt.Printf("    %s\n", format.Muted.Sprint(spec.Prompt))
				}
			}
			return nil
		},
	}
	c.Flags().StringVar(&tplOverride, "template", "", "Override inferred template")
	c.Flags().StringVar(&domOverride, "domain", "", "Force lore domain")
	return c
}

func submitCmd() *cobra.Command {
	var planFile, nameFlag, domainFlag string
	var overwrite, recordDecision bool
	c := &cobra.Command{
		Use:   "submit <roster-id>",
		Short: "Validate a plan, spawn children, write plans.jsonl row",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rosterID := args[0]
			dir, _ := config.FindRostersDir("")

			var raw []byte
			var err error
			if planFile == "-" {
				raw, err = io.ReadAll(os.Stdin)
			} else {
				raw, err = os.ReadFile(planFile)
			}
			if err != nil {
				return err
			}

			var submitted models.SubmittedPlan
			if err := json.Unmarshal(raw, &submitted); err != nil {
				return err
			}

			tpls, _ := config.LoadPlanTemplates(dir)
			tpl, ok := tpls[submitted.Template]
			if !ok {
				return fmt.Errorf("unknown template: %s", submitted.Template)
			}

			vRes := plan.ValidatePlan(submitted, *tpl)
			if !vRes.Valid {
				b, _ := json.MarshalIndent(vRes, "", "  ")
				fmt.Fprintln(os.Stderr, string(b))
				os.Exit(1)
			}

			now := time.Now().Format(time.RFC3339)
			var planID string
			var childIDs []string
			var revision int
			var rosterSnapshot models.Issue

			_, err = store.WithLock(store.PlansPath(dir), func() (any, error) {
				return store.WithLock(store.IssuesPath(dir), func() (any, error) {
					issues, _ := store.ReadIssues(dir)
					plans, _ := store.ReadPlans(dir)

					var s *models.Issue
					var sIdx int
					for i, iss := range issues {
						if iss.ID == rosterID {
							s = &iss
							sIdx = i
							break
						}
					}
					if s == nil {
						return nil, fmt.Errorf("roster not found: %s", rosterID)
					}
					rosterSnapshot = *s

					for _, p := range plans {
						if p.Roster == rosterID && p.Status != models.PlanStatusDraft && !overwrite {
							fmt.Fprintf(os.Stderr, "✗ plan %s already exists for %s (status: %s, revision: %d)\n  Use --overwrite to replace it.\n", p.ID, rosterID, p.Status, p.Revision)
							os.Exit(1)
						}
					}

					pIDs := make([]string, len(plans))
					for i, p := range plans {
						pIDs[i] = p.ID
					}
					planID = util.GenerateID("pl", pIDs)
					revision = 1

					steps, _ := submitted.Sections["steps"].([]any)
					for i, st := range steps {
						step := st.(map[string]any)
						issIDs := make([]string, len(issues))
						for j, iss := range issues {
							issIDs[j] = iss.ID
						}
						cfg, _ := config.ReadConfig(dir)
						cID := util.GenerateID(cfg.Project, append(issIDs, childIDs...))
						childIDs = append(childIDs, cID)

						newIss := models.Issue{
							ID:            cID,
							Title:         step["title"].(string),
							Status:        "open",
							Type:          "task",
							Priority:      2,
							PlanID:        &planID,
							PlanStepIndex: util.Ptr(i),
							CreatedAt:     now,
							UpdatedAt:     now,
						}

						args := plan.BackrefArgs{
							StepIndex:         util.Ptr(i),
							PlanID:            planID,
							ParentRosterID:    s.ID,
							ParentRosterTitle: s.Title,
							TemplateName:      submitted.Template,
							Approach:          submitted.Sections["approach"],
						}
						newIss.Description = util.Ptr(plan.BuildPlanBackref(args))
						newIss.Blocks = []string{s.ID}
						issues = append(issues, newIss)
						s.BlockedBy = append(s.BlockedBy, cID)
					}

					s.PlanID = &planID
					s.UpdatedAt = now
					issues[sIdx] = *s

					pName := nameFlag
					if pName == "" && submitted.Name != nil {
						pName = *submitted.Name
					}

					newPlan := models.Plan{
						ID:        planID,
						Roster:    rosterID,
						Template:  submitted.Template,
						Status:    models.PlanStatusApproved,
						Revision:  revision,
						Sections:  submitted.Sections,
						Children:  childIDs,
						CreatedAt: now,
						UpdatedAt: now,
					}
					if pName != "" {
						newPlan.Name = &pName
					}

					plans = append(plans, newPlan)
					store.WriteIssues(dir, issues)
					store.WritePlans(dir, plans)
					return nil, nil
				})
			})

			if err != nil {
				return err
			}

			var loreID string
			if recordDecision {
				loreID = runOutboundDecision(rosterSnapshot, planID, submitted.Sections["approach"], domainFlag, dir)
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{
					"success":       true,
					"command":       "plan submit",
					"plan_id":       planID,
					"children":      childIDs,
					"parent_roster": rosterID,
					"revision":      revision,
					"overwritten":   overwrite,
				})
			} else {
				format.PrintSuccess(fmt.Sprintf("plan %s created (status: approved)", format.Accent.Sprint(planID)))
				format.PrintSuccess(fmt.Sprintf("%d child rosters: %v", len(childIDs), childIDs))
				if loreID != "" {
					format.PrintSuccess(fmt.Sprintf("recorded lore decision %s", format.Accent.Sprint(loreID)))
				}
				writeNextHints(planID)
			}
			return nil
		},
	}
	c.Flags().StringVar(&planFile, "plan", "", "Path to plan JSON")
	c.MarkFlagRequired("plan")
	c.Flags().StringVar(&nameFlag, "name", "", "Human-readable label")
	c.Flags().BoolVar(&overwrite, "overwrite", false, "Replace existing plan")
	c.Flags().BoolVar(&recordDecision, "record-decision", false, "Record approach as lore decision")
	c.Flags().StringVar(&domainFlag, "domain", "", "Force lore domain")
	return c
}

func runOutboundDecision(roster models.Issue, planID string, approach any, domainOverride string, dir string) string {
	if _, err := exec.LookPath("lr"); err != nil {
		fmt.Fprintln(os.Stderr, "⚠ --record-decision: lr not found on PATH; skipping")
		return ""
	}
	projectRoot := filepath.Dir(dir)
	domain, _ := plan.InferDomain(util.Deref(roster.Description), roster.Labels, domainOverride, projectRoot)
	if domain == "" {
		fmt.Fprintln(os.Stderr, "⚠ --record-decision: no lore domain inferred (skipping)")
		return ""
	}

	appStr, _ := approach.(string)
	cmd := exec.Command("lr", "decision", "add", "--domain", domain, "--title", roster.Title, "--body", appStr, "--meta", fmt.Sprintf("plan_id=%s", planID))
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ --record-decision: failed (%v)\n", err)
		return ""
	}
	return string(out)
}

func writeNextHints(planID string) {
	fmt.Fprintf(os.Stderr, "\nNext:\n  rt plan show %s\n  rt ready\n", planID)
}

func showPlanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := config.FindRostersDir("")
			plans, _ := store.ReadPlans(dir)
			id := args[0]
			var p *models.Plan
			for _, pl := range plans {
				if pl.ID == id || pl.Roster == id {
					p = &pl
					break
				}
			}
			if p == nil {
				return fmt.Errorf("plan not found")
			}

			issues, _ := store.ReadIssues(dir)
			children := plan.SummarisePlanChildren(*p, issues)

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{
					"success":  true,
					"command":  "plan show",
					"plan":     p,
					"children": children,
				})
				return nil
			}

			fmt.Printf("%s  %s  rev %d\n", format.AccentBold(p.ID), format.Brand.Sprint(p.Status), p.Revision)
			if p.Name != nil {
				fmt.Printf("Name:     %s\n", *p.Name)
			}
			fmt.Printf("Roster:   %s\n", format.Accent.Sprint(p.Roster))
			fmt.Printf("Template: %s\n", p.Template)
			fmt.Printf("\n%s\n", format.Brand.Sprint("Sections:"))
			for k, v := range p.Sections {
				fmt.Printf("  %s\n    %v\n", format.AccentBold(k), v)
			}
			fmt.Printf("\n%s\n", format.Brand.Sprint(fmt.Sprintf("Children (%d):", len(children))))
			for _, c := range children {
				tag := ""
				if c.Adopted {
					tag = " (adopted)"
				}
				fmt.Printf("  %s  [%s]  %s%s\n", format.Accent.Sprint(c.ID), format.Muted.Sprint(c.Status), c.Title, format.Muted.Sprint(tag))
			}
			return nil
		},
	}
}

func listPlansCmd() *cobra.Command {
	var rosterFilter, statusFilter, outcomeFilter, templateFilter string
	c := &cobra.Command{
		Use:   "list",
		Short: "List plans with optional filters",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := config.FindRostersDir("")
			plans, _ := store.ReadPlans(dir)

			var filtered []models.Plan
			for _, p := range plans {
				if rosterFilter != "" && p.Roster != rosterFilter {
					continue
				}
				if statusFilter != "" && string(p.Status) != statusFilter {
					continue
				}
				if outcomeFilter != "" && (p.Outcome == nil || string(*p.Outcome) != outcomeFilter) {
					continue
				}
				if templateFilter != "" && p.Template != templateFilter {
					continue
				}
				filtered = append(filtered, p)
			}

			sort.SliceStable(filtered, func(i, j int) bool {
				return filtered[i].CreatedAt > filtered[j].CreatedAt
			})

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{
					"success": true,
					"command": "plan list",
					"plans":   filtered,
					"count":   len(filtered),
				})
				return nil
			}

			if len(filtered) == 0 {
				format.Muted.Println("No plans match.")
				return nil
			}

			for _, p := range filtered {
				namePart := ""
				if p.Name != nil {
					namePart = fmt.Sprintf("  %s", *p.Name)
				}
				fmt.Printf("%s  %s  rev %d%s  %s  roster=%s\n", format.AccentBold(p.ID), format.Muted.Sprint(p.Status), p.Revision, namePart, p.Template, p.Roster)
			}
			return nil
		},
	}
	c.Flags().StringVar(&rosterFilter, "roster", "", "Filter by parent roster ID")
	c.Flags().StringVar(&statusFilter, "status", "", "Filter by status")
	c.Flags().StringVar(&outcomeFilter, "outcome", "", "Filter by outcome")
	c.Flags().StringVar(&templateFilter, "template", "", "Filter by template name")
	return c
}

func adoptCmd() *cobra.Command {
	var step int
	c := &cobra.Command{
		Use:   "adopt <plan-id> <roster-ids...>",
		Short: "Adopt existing open rosters into a plan",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			planID := args[0]
			toAdopt := args[1:]
			dir, _ := config.FindRostersDir("")

			var updatedPlan models.Plan
			_, err := store.WithLock(store.PlansPath(dir), func() (any, error) {
				return store.WithLock(store.IssuesPath(dir), func() (any, error) {
					plans, _ := store.ReadPlans(dir)
					issues, _ := store.ReadIssues(dir)

					var pIdx = -1
					for i, pl := range plans {
						if pl.ID == planID {
							pIdx = i
							break
						}
					}
					if pIdx == -1 {
						return nil, fmt.Errorf("plan not found")
					}

					now := time.Now().Format(time.RFC3339)
					for _, id := range toAdopt {
						for i, iss := range issues {
							if iss.ID == id {
								iss.PlanID = &planID
								if step > 0 {
									iss.PlanStepIndex = util.Ptr(step - 1)
								}
								iss.UpdatedAt = now
								issues[i] = iss
								plans[pIdx].Children = append(plans[pIdx].Children, id)
								plans[pIdx].AdoptedChildren = append(plans[pIdx].AdoptedChildren, id)
								break
							}
						}
					}
					plans[pIdx].Revision++
					plans[pIdx].UpdatedAt = now
					updatedPlan = plans[pIdx]
					store.WriteIssues(dir, issues)
					store.WritePlans(dir, plans)
					return nil, nil
				})
			})

			if err != nil {
				return err
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{
					"success":  true,
					"command":  "plan adopt",
					"plan_id":  planID,
					"adopted":  toAdopt,
					"revision": updatedPlan.Revision,
				})
			} else {
				for _, id := range toAdopt {
					format.PrintSuccess(fmt.Sprintf("%s adopted into plan %s", format.Accent.Sprint(id), format.Accent.Sprint(planID)))
				}
				format.PrintSuccess(fmt.Sprintf("plan %s revision bumped to %d", format.Accent.Sprint(planID), updatedPlan.Revision))
			}
			return nil
		},
	}
	c.Flags().IntVar(&step, "step", 0, "1-based step index")
	return c
}

func releaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "release <plan-id> <roster-ids...>",
		Short: "Release rosters from a plan",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			planID := args[0]
			toRelease := make(map[string]bool)
			for _, id := range args[1:] {
				toRelease[id] = true
			}
			dir, _ := config.FindRostersDir("")

			var updatedPlan models.Plan
			_, err := store.WithLock(store.PlansPath(dir), func() (any, error) {
				return store.WithLock(store.IssuesPath(dir), func() (any, error) {
					plans, _ := store.ReadPlans(dir)
					issues, _ := store.ReadIssues(dir)

					var pIdx = -1
					for i, pl := range plans {
						if pl.ID == planID {
							pIdx = i
							break
						}
					}
					if pIdx == -1 {
						return nil, fmt.Errorf("plan not found")
					}

					now := time.Now().Format(time.RFC3339)
					var nextC []string
					for _, cid := range plans[pIdx].Children {
						if !toRelease[cid] {
							nextC = append(nextC, cid)
						}
					}

					for i, iss := range issues {
						if toRelease[iss.ID] && iss.PlanID != nil && *iss.PlanID == planID {
							iss.PlanID = nil
							iss.PlanStepIndex = nil
							iss.UpdatedAt = now
							issues[i] = iss
						}
					}

					plans[pIdx].Children = nextC
					plans[pIdx].Revision++
					plans[pIdx].UpdatedAt = now
					updatedPlan = plans[pIdx]
					store.WriteIssues(dir, issues)
					store.WritePlans(dir, plans)
					return nil, nil
				})
			})

			if err != nil {
				return err
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{
					"success":  true,
					"command":  "plan release",
					"plan_id":  planID,
					"released": args[1:],
					"revision": updatedPlan.Revision,
				})
			} else {
				for _, id := range args[1:] {
					format.PrintSuccess(fmt.Sprintf("%s released from plan %s", format.Accent.Sprint(id), format.Accent.Sprint(planID)))
				}
				format.PrintSuccess(fmt.Sprintf("plan %s revision bumped to %d", format.Accent.Sprint(planID), updatedPlan.Revision))
			}
			return nil
		},
	}
}

func validatePlanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <id>",
		Short: "Re-run validation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := config.FindRostersDir("")
			plans, _ := store.ReadPlans(dir)
			id := args[0]
			var p *models.Plan
			for _, pl := range plans {
				if pl.ID == id || pl.Roster == id {
					p = &pl
					break
				}
			}
			if p == nil {
				return fmt.Errorf("plan not found")
			}

			tpls, _ := config.LoadPlanTemplates(dir)
			tpl, _ := tpls[p.Template]

			submitted := models.SubmittedPlan{Template: p.Template, Sections: p.Sections}
			res := plan.ValidatePlan(submitted, *tpl)

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{
					"success": res.Valid,
					"command": "plan validate",
					"valid":   res.Valid,
					"plan_id": p.ID,
					"errors":  res.Errors,
				})
			} else if res.Valid {
				format.PrintSuccess(fmt.Sprintf("plan %s valid", format.Accent.Sprint(p.ID)))
			} else {
				b, _ := json.MarshalIndent(res.Errors, "", "  ")
				fmt.Fprintln(os.Stderr, string(b))
				os.Exit(1)
			}
			return nil
		},
	}
}

func outcomePlanCmd() *cobra.Command {
	var result, note string
	c := &cobra.Command{
		Use:   "outcome <id>",
		Short: "Record a plan outcome",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := config.FindRostersDir("")
			id := args[0]
			var updated models.Plan
			var openChildren int

			_, err := store.WithLock(store.PlansPath(dir), func() (any, error) {
				return store.WithLock(store.IssuesPath(dir), func() (any, error) {
					plans, _ := store.ReadPlans(dir)
					issues, _ := store.ReadIssues(dir)
					for i, p := range plans {
						if p.ID == id || p.Roster == id {
							outcome := models.PlanOutcome(result)
							plans[i].Outcome = &outcome
							if note != "" {
								plans[i].OutcomeNote = &note
							}
							plans[i].UpdatedAt = time.Now().Format(time.RFC3339)
							updated = plans[i]

							for _, cid := range p.Children {
								for _, iss := range issues {
									if iss.ID == cid && iss.Status != "closed" {
										openChildren++
										break
									}
								}
							}

							return nil, store.WritePlans(dir, plans)
						}
					}
					return nil, fmt.Errorf("plan not found")
				})
			})

			if err != nil {
				return err
			}

			if openChildren > 0 {
				fmt.Fprintf(os.Stderr, "⚠ plan %s has %d open children\n", updated.ID, openChildren)
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{
					"success":       true,
					"command":       "plan outcome",
					"plan_id":       updated.ID,
					"outcome":       updated.Outcome,
					"outcomeNote":   updated.OutcomeNote,
					"open_children": openChildren,
				})
			} else {
				noteStr := ""
				if updated.OutcomeNote != nil {
					noteStr = fmt.Sprintf(" - %s", *updated.OutcomeNote)
				}
				format.PrintSuccess(fmt.Sprintf("plan %s outcome recorded: %s%s", format.Accent.Sprint(updated.ID), *updated.Outcome, noteStr))
			}
			return nil
		},
	}
	c.Flags().StringVar(&result, "result", "", "success|partial|failure")
	c.MarkFlagRequired("result")
	c.Flags().StringVar(&note, "note", "", "Optional note")
	return c
}

func reviewPlanCmd() *cobra.Command {
	var by string
	c := &cobra.Command{
		Use:   "review <id>",
		Short: "Record a reviewer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := config.FindRostersDir("")
			id := args[0]
			var updated models.Plan
			_, err := store.WithLock(store.PlansPath(dir), func() (any, error) {
				plans, _ := store.ReadPlans(dir)
				for i, p := range plans {
					if p.ID == id || p.Roster == id {
						plans[i].ReviewedBy = &by
						plans[i].UpdatedAt = time.Now().Format(time.RFC3339)
						updated = plans[i]
						return nil, store.WritePlans(dir, plans)
					}
				}
				return nil, fmt.Errorf("plan not found")
			})

			if err != nil {
				return err
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{
					"success":    true,
					"command":    "plan review",
					"plan_id":    updated.ID,
					"reviewedBy": updated.ReviewedBy,
				})
			} else {
				format.PrintSuccess(fmt.Sprintf("plan %s reviewed by %s", format.Accent.Sprint(updated.ID), *updated.ReviewedBy))
			}
			return nil
		},
	}
	c.Flags().StringVar(&by, "by", "", "Reviewer name")
	c.MarkFlagRequired("by")
	return c
}
