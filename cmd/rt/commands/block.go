package commands

import (
	"fmt"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

func RegisterBlockCommand(rootCmd *cobra.Command) {
	var blockerID string

	blockCmd := &cobra.Command{
		Use:   "block <id> --by <blocker-id>",
		Short: "Add a blocker to an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if blockerID == "" {
				return fmt.Errorf("usage: rt block <id> --by <blocker-id>")
			}
			return runBlock(args[0], blockerID)
		},
	}

	blockCmd.Flags().StringVar(&blockerID, "by", "", "Issue that blocks this issue")
	blockCmd.MarkFlagRequired("by")
	rootCmd.AddCommand(blockCmd)
}

func runBlock(issueID, blockerID string) error {
	dir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	_, err = store.WithLock(store.IssuesPath(dir), func() (any, error) {
		issues, err := store.ReadIssues(dir)
		if err != nil {
			return nil, err
		}

		issueIdx := -1
		blockerIdx := -1
		for i := range issues {
			if issues[i].ID == issueID {
				issueIdx = i
			}
			if issues[i].ID == blockerID {
				blockerIdx = i
			}
		}

		if issueIdx == -1 {
			return nil, fmt.Errorf("issue not found: %s", issueID)
		}
		if blockerIdx == -1 {
			return nil, fmt.Errorf("issue not found: %s", blockerID)
		}

		now := time.Now().Format(time.RFC3339)

		foundBy := false
		for _, id := range issues[issueIdx].BlockedBy {
			if id == blockerID {
				foundBy = true
				break
			}
		}
		if !foundBy {
			issues[issueIdx].BlockedBy = append(issues[issueIdx].BlockedBy, blockerID)
			issues[issueIdx].UpdatedAt = now
		}

		foundBlocks := false
		for _, id := range issues[blockerIdx].Blocks {
			if id == issueID {
				foundBlocks = true
				break
			}
		}
		if !foundBlocks {
			issues[blockerIdx].Blocks = append(issues[blockerIdx].Blocks, issueID)
			issues[blockerIdx].UpdatedAt = now
		}

		return nil, store.WriteIssues(dir, issues)
	})

	if err != nil {
		return err
	}

	if format.GetFormat() == "json" {
		format.OutputJSON(map[string]any{
			"success":   true,
			"command":   "block",
			"issueId":   issueID,
			"blockerId": blockerID,
		})
	} else {
		fmt.Printf("%s %s %s\n", format.AccentBold(issueID), format.Muted.Sprint("is now blocked by"), format.AccentBold(blockerID))
	}

	return nil
}
