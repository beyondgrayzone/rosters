package commands

import (
	"fmt"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/plan"
	"rosters/pkg/store"
	"rosters/pkg/util"

	"github.com/spf13/cobra"
)

func RegisterCloseCommand(rootCmd *cobra.Command) {
	var reason string

	closeCmd := &cobra.Command{
		Use:   "close <id> [ids...]",
		Short: "Close one or more issues",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClose(args, reason)
		},
	}

	closeCmd.Flags().StringVar(&reason, "reason", "", "Reason for closing the issue")
	rootCmd.AddCommand(closeCmd)
}

func runClose(ids []string, reason string) error {
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	isJSON := format.GetFormat() == "json"
	now := time.Now().UTC().Format(time.RFC3339)

	var closedIDs []string

	plansPath := store.PlansPath(rostersDir)
	issuesPath := store.IssuesPath(rostersDir)

	if err := store.AcquireLock(plansPath); err != nil {
		return err
	}
	defer store.ReleaseLock(plansPath)

	if err := store.AcquireLock(issuesPath); err != nil {
		return err
	}
	defer store.ReleaseLock(issuesPath)

	issues, err := store.ReadIssues(rostersDir)
	if err != nil {
		return err
	}

	for _, id := range ids {
		foundIdx := -1
		for i, iss := range issues {
			if iss.ID == id {
				foundIdx = i
				break
			}
		}

		if foundIdx == -1 {
			return fmt.Errorf("issue not found: %s", id)
		}

		issue := &issues[foundIdx]
		issue.Status = "closed"
		issue.ClosedAt = util.Ptr(now)
		issue.UpdatedAt = now
		if reason != "" {
			issue.CloseReason = util.Ptr(reason)
		}

		closedIDs = append(closedIDs, id)

		for _, blockedID := range issue.Blocks {
			for i := range issues {
				if issues[i].ID == blockedID {
					var remaining []string
					for _, bid := range issues[i].BlockedBy {
						if bid != id {
							remaining = append(remaining, bid)
						}
					}
					if len(remaining) > 0 {
						issues[i].BlockedBy = remaining
					} else {
						issues[i].BlockedBy = nil
					}
					issues[i].UpdatedAt = now
					break
				}
			}
		}
	}

	if err := store.WriteIssues(rostersDir, issues); err != nil {
		return err
	}

	plans, err := store.ReadPlans(rostersDir)
	if err != nil {
		return err
	}

	affected := plan.AffectedPlanIDs(plans, closedIDs)
	if len(affected) > 0 {
		changed := plan.ApplyPlanTransitions(plans, issues, affected, now)
		if changed > 0 {
			if err := store.WritePlans(rostersDir, plans); err != nil {
				return err
			}
		}
	}

	if isJSON {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "close",
			"closed":  closedIDs,
		})
	} else {
		for _, id := range closedIDs {
			msg := fmt.Sprintf("Closed %s", id)
			if reason != "" {
				msg += fmt.Sprintf(" - %s", reason)
			}
			format.PrintSuccess(msg)
		}
	}

	return nil
}
