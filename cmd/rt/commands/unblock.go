package commands

import (
	"fmt"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

func RegisterUnblockCommand(rootCmd *cobra.Command) {
	var blockerID string
	var allFlag bool

	unblockCmd := &cobra.Command{
		Use:   "unblock <id> [--from <blocker-id> | --all]",
		Short: "Remove blockers from an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !allFlag && blockerID == "" {
				return fmt.Errorf("usage: rt unblock <id> [--from <blocker-id> | --all]")
			}
			return runUnblock(args[0], blockerID, allFlag)
		},
	}

	unblockCmd.Flags().StringVar(&blockerID, "from", "", "Remove a specific blocker")
	unblockCmd.Flags().BoolVar(&allFlag, "all", false, "Remove all closed blockers")
	rootCmd.AddCommand(unblockCmd)
}

func runUnblock(issueID, blockerID string, all bool) error {
	dir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	var removed []string

	_, err = store.WithLock(store.IssuesPath(dir), func() (any, error) {
		issues, err := store.ReadIssues(dir)
		if err != nil {
			return nil, err
		}

		issueIdx := -1
		for i := range issues {
			if issues[i].ID == issueID {
				issueIdx = i
				break
			}
		}

		if issueIdx == -1 {
			return nil, fmt.Errorf("issue not found: %s", issueID)
		}

		issue := &issues[issueIdx]
		currentBlockers := issue.BlockedBy
		now := time.Now().Format(time.RFC3339)

		if all {
			closedIDs := make(map[string]bool)
			for _, iss := range issues {
				if iss.Status == "closed" {
					closedIDs[iss.ID] = true
				}
			}

			var remaining []string
			for _, bid := range currentBlockers {
				if closedIDs[bid] {
					removed = append(removed, bid)
				} else {
					remaining = append(remaining, bid)
				}
			}

			issue.BlockedBy = remaining
			if len(issue.BlockedBy) == 0 {
				issue.BlockedBy = nil
			}
			issue.UpdatedAt = now

			for _, rid := range removed {
				for i := range issues {
					if issues[i].ID == rid {
						var newBlocks []string
						for _, b := range issues[i].Blocks {
							if b != issueID {
								newBlocks = append(newBlocks, b)
							}
						}
						issues[i].Blocks = newBlocks
						if len(issues[i].Blocks) == 0 {
							issues[i].Blocks = nil
						}
						issues[i].UpdatedAt = now
						break
					}
				}
			}
		} else {
			found := false
			var remaining []string
			for _, bid := range currentBlockers {
				if bid == blockerID {
					found = true
					removed = append(removed, bid)
				} else {
					remaining = append(remaining, bid)
				}
			}

			if !found {
				return nil, fmt.Errorf("%s is not blocked by %s", issueID, blockerID)
			}

			issue.BlockedBy = remaining
			if len(issue.BlockedBy) == 0 {
				issue.BlockedBy = nil
			}
			issue.UpdatedAt = now

			for i := range issues {
				if issues[i].ID == blockerID {
					var newBlocks []string
					for _, b := range issues[i].Blocks {
						if b != issueID {
							newBlocks = append(newBlocks, b)
						}
					}
					issues[i].Blocks = newBlocks
					if len(issues[i].Blocks) == 0 {
						issues[i].Blocks = nil
					}
					issues[i].UpdatedAt = now
					break
				}
			}
		}

		return nil, store.WriteIssues(dir, issues)
	})

	if err != nil {
		return err
	}

	if format.GetFormat() == "json" {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "unblock",
			"issueId": issueID,
			"removed": removed,
		})
	} else {
		if len(removed) == 0 {
			fmt.Printf("%s %s %s.\n", format.Muted.Sprint("No closed blockers to remove from"), format.AccentBold(issueID), "")
		} else {
			for _, rid := range removed {
				fmt.Printf("%s %s %s\n", format.AccentBold(issueID), format.Muted.Sprint("unblocked from"), format.AccentBold(rid))
			}
		}
	}

	return nil
}
