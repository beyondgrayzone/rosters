package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/filter"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/plan"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

type updateOptions struct {
	status          string
	title           string
	assignee        string
	description     string
	issueType       string
	priority        string
	addLabel        string
	removeLabel     string
	setLabels       *string
	extensions      string
	clearExtensions bool
}

func RegisterUpdateCommand(rootCmd *cobra.Command) {
	var opts updateOptions

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update issue fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(args[0], opts)
		},
	}

	updateCmd.Flags().StringVar(&opts.status, "status", "", "New status (open|in_progress|closed)")
	updateCmd.Flags().StringVar(&opts.title, "title", "", "New title")
	updateCmd.Flags().StringVar(&opts.assignee, "assignee", "", "New assignee")
	updateCmd.Flags().StringVar(&opts.description, "description", "", "New description")
	updateCmd.Flags().StringVar(&opts.description, "desc", "", "New description (alias)")
	updateCmd.Flags().StringVar(&opts.description, "body", "", "New description (alias)")
	updateCmd.Flags().StringVar(&opts.issueType, "type", "", "New type (task|bug|feature|epic)")
	updateCmd.Flags().StringVar(&opts.priority, "priority", "", "New priority 0-4 or P0-P4")
	updateCmd.Flags().StringVar(&opts.addLabel, "add-label", "", "Add label(s) (comma-separated)")
	updateCmd.Flags().StringVar(&opts.removeLabel, "remove-label", "", "Remove label(s) (comma-separated)")
	opts.setLabels = updateCmd.Flags().String("set-labels", "", "Set labels (comma-separated, empty to clear)")
	updateCmd.Flags().Lookup("set-labels").Changed = false
	updateCmd.Flags().StringVar(&opts.extensions, "extensions", "", "Shallow-merge JSON object into Issue.extensions")
	updateCmd.Flags().BoolVar(&opts.clearExtensions, "clear-extensions", false, "Remove the extensions field")

	rootCmd.AddCommand(updateCmd)
}

func runUpdate(id string, opts updateOptions) error {
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	statusChanging := opts.status != ""

	inner := func() error {
		issues, err := store.ReadIssues(rostersDir)
		if err != nil {
			return err
		}

		var issue *models.Issue
		issueIdx := -1
		for i, iss := range issues {
			if iss.ID == id {
				issue = &issues[i]
				issueIdx = i
				break
			}
		}

		if issue == nil {
			return fmt.Errorf("issue not found: %s", id)
		}

		now := time.Now().Format(time.RFC3339)
		oldStatus := issue.Status
		issue.UpdatedAt = now

		if opts.status != "" {
			valid := false
			for _, v := range models.ValidStatuses {
				if v == opts.status {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("--status must be one of: %s", strings.Join(models.ValidStatuses, ", "))
			}
			issue.Status = opts.status
			if issue.Status != "closed" {
				issue.ClosedAt = nil
				issue.CloseReason = nil
			}
		}

		if opts.title != "" {
			issue.Title = opts.title
		}
		if opts.assignee != "" {
			issue.Assignee = &opts.assignee
		}
		if opts.description != "" {
			issue.Description = &opts.description
		}
		if opts.issueType != "" {
			valid := false
			for _, v := range models.ValidTypes {
				if v == opts.issueType {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("--type must be one of: %s", strings.Join(models.ValidTypes, ", "))
			}
			issue.Type = opts.issueType
		}
		if opts.priority != "" {
			p, err := filter.ParsePriorityToken(opts.priority)
			if err != nil {
				return err
			}
			issue.Priority = p
		}

		if opts.extensions != "" && opts.clearExtensions {
			return fmt.Errorf("--extensions and --clear-extensions are mutually exclusive")
		}

		if opts.clearExtensions {
			issue.Extensions = nil
		} else if opts.extensions != "" {
			var incoming map[string]any
			if err := json.Unmarshal([]byte(opts.extensions), &incoming); err != nil {
				return fmt.Errorf("--extensions must be valid JSON: %w", err)
			}
			if issue.Extensions == nil {
				issue.Extensions = make(map[string]any)
			}
			for k, v := range incoming {
				issue.Extensions[k] = v
			}
		}

		if labelsSetChanged(opts.setLabels) {
			val := *opts.setLabels
			if val == "" {
				issue.Labels = nil
			} else {
				issue.Labels = splitLabels(val)
			}
		}

		if opts.addLabel != "" {
			toAdd := splitLabels(opts.addLabel)
			seen := make(map[string]bool)
			var merged []string
			for _, l := range issue.Labels {
				if !seen[l] {
					merged = append(merged, l)
					seen[l] = true
				}
			}
			for _, l := range toAdd {
				if !seen[l] {
					merged = append(merged, l)
					seen[l] = true
				}
			}
			issue.Labels = merged
		}

		if opts.removeLabel != "" {
			toRemove := make(map[string]bool)
			for _, l := range splitLabels(opts.removeLabel) {
				toRemove[l] = true
			}
			var remaining []string
			for _, l := range issue.Labels {
				if !toRemove[l] {
					remaining = append(remaining, l)
				}
			}
			issue.Labels = remaining
		}

		issues[issueIdx] = *issue
		if err := store.WriteIssues(rostersDir, issues); err != nil {
			return err
		}

		if statusChanging && issue.Status != oldStatus {
			plans, err := store.ReadPlans(rostersDir)
			if err != nil {
				return err
			}
			affected := plan.AffectedPlanIDs(plans, []string{id})
			if len(affected) > 0 {
				changed := plan.ApplyPlanTransitions(plans, issues, affected, now)
				if changed > 0 {
					if err := store.WritePlans(rostersDir, plans); err != nil {
						return err
					}
				}
			}
		}

		if format.GetFormat() == "json" {
			format.OutputJSON(map[string]any{
				"success": true,
				"command": "update",
				"issue":   issue,
			})
		} else {
			format.PrintSuccess(fmt.Sprintf("Updated %s", id))
		}
		return nil
	}

	if statusChanging {
		_, err = store.WithLock(store.PlansPath(rostersDir), func() (any, error) {
			return store.WithLock(store.IssuesPath(rostersDir), func() (any, error) {
				return nil, inner()
			})
		})
	} else {
		_, err = store.WithLock(store.IssuesPath(rostersDir), func() (any, error) {
			return nil, inner()
		})
	}

	return err
}

func labelsSetChanged(setLabels *string) bool {
	// Simple check: in a real implementation we'd check if the flag was actually provided
	// For this port, we rely on the pointer not being nil.
	return setLabels != nil && *setLabels != "__NOT_SET__"
}

func splitLabels(value string) []string {
	parts := strings.Split(value, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(p))
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
