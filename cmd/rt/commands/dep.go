package commands

import (
	"fmt"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

func RegisterDepCommand(rootCmd *cobra.Command) {
	depCmd := &cobra.Command{
		Use:   "dep",
		Short: "Manage issue dependencies",
	}

	addCmd := &cobra.Command{
		Use:   "add <issue> <depends-on>",
		Short: "Add a dependency (issue depends on depends-on)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDepAddRemove(args[0], args[1], true)
		},
	}

	removeCmd := &cobra.Command{
		Use:   "remove <issue> <depends-on>",
		Short: "Remove a dependency",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDepAddRemove(args[0], args[1], false)
		},
	}

	listCmd := &cobra.Command{
		Use:   "list <issue>",
		Short: "Show dependencies for an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDepList(args[0])
		},
	}

	depCmd.AddCommand(addCmd, removeCmd, listCmd)
	rootCmd.AddCommand(depCmd)
}

func runDepList(issueID string) error {
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	issues, err := store.ReadIssues(rostersDir)
	if err != nil {
		return err
	}

	var issue *models.Issue
	for _, i := range issues {
		if i.ID == issueID {
			issue = &i
			break
		}
	}

	if issue == nil {
		return fmt.Errorf("issue not found: %s", issueID)
	}

	blockedBy := issue.BlockedBy
	blocks := issue.Blocks
	closedBlockerIds := make(map[string]bool)
	for _, i := range issues {
		if i.Status == "closed" {
			closedBlockerIds[i.ID] = true
		}
	}

	if format.GetFormat() == "json" {
		format.OutputJSON(map[string]any{
			"success":   true,
			"command":   "dep list",
			"issueId":   issueID,
			"blockedBy": blockedBy,
			"blocks":    blocks,
		})
		return nil
	}

	fmt.Printf("%s %s\n", format.AccentBold(issueID), format.Muted.Sprint("dependencies:"))
	if len(blockedBy) > 0 {
		fmt.Println(format.Muted.Sprint("  Blocked by:"))
		for _, bid := range blockedBy {
			var b *models.Issue
			for _, i := range issues {
				if i.ID == bid {
					b = &i
					break
				}
			}
			if b != nil {
				fmt.Print("    ")
				format.PrintIssueOneLine(*b, closedBlockerIds)
			} else {
				fmt.Printf("    %s %s\n", format.Accent.Sprint(bid), format.Muted.Sprint("(not found)"))
			}
		}
	}

	if len(blocks) > 0 {
		fmt.Println(format.Muted.Sprint("  Blocks:"))
		for _, bid := range blocks {
			var b *models.Issue
			for _, i := range issues {
				if i.ID == bid {
					b = &i
					break
				}
			}
			if b != nil {
				fmt.Print("    ")
				format.PrintIssueOneLine(*b, closedBlockerIds)
			} else {
				fmt.Printf("    %s %s\n", format.Accent.Sprint(bid), format.Muted.Sprint("(not found)"))
			}
		}
	}

	if len(blockedBy) == 0 && len(blocks) == 0 {
		fmt.Println(format.Muted.Sprint("  No dependencies."))
	}

	return nil
}

func runDepAddRemove(issueID string, dependsOnID string, isAdd bool) error {
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	_, err = store.WithLock(store.IssuesPath(rostersDir), func() (any, error) {
		issues, err := store.ReadIssues(rostersDir)
		if err != nil {
			return nil, err
		}

		issueIdx := -1
		depIdx := -1
		for i, iss := range issues {
			if iss.ID == issueID {
				issueIdx = i
			}
			if iss.ID == dependsOnID {
				depIdx = i
			}
		}

		if issueIdx == -1 {
			return nil, fmt.Errorf("issue not found: %s", issueID)
		}
		if depIdx == -1 {
			return nil, fmt.Errorf("issue not found: %s", dependsOnID)
		}

		now := time.Now().UTC().Format(time.RFC3339)

		if isAdd {
			issues[issueIdx].BlockedBy = addToSet(issues[issueIdx].BlockedBy, dependsOnID)
			issues[depIdx].Blocks = addToSet(issues[depIdx].Blocks, issueID)
		} else {
			issues[issueIdx].BlockedBy = removeFromSet(issues[issueIdx].BlockedBy, dependsOnID)
			issues[depIdx].Blocks = removeFromSet(issues[depIdx].Blocks, issueID)
		}

		issues[issueIdx].UpdatedAt = now
		issues[depIdx].UpdatedAt = now

		return nil, store.WriteIssues(rostersDir, issues)
	})

	if err != nil {
		return err
	}

	if format.GetFormat() == "json" {
		subcmd := "remove"
		if isAdd {
			subcmd = "add"
		}
		format.OutputJSON(map[string]any{
			"success":     true,
			"command":     fmt.Sprintf("dep %s", subcmd),
			"issueId":     issueID,
			"dependsOnId": dependsOnID,
		})
	} else {
		verb := "Removed"
		if isAdd {
			verb = "Added"
		}
		fmt.Printf("%s dependency: %s %s %s\n", verb, format.Accent.Sprint(issueID), format.Muted.Sprint("→"), format.Accent.Sprint(dependsOnID))
	}

	return nil
}

func addToSet(slice []string, val string) []string {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}

func removeFromSet(slice []string, val string) []string {
	var result []string
	for _, v := range slice {
		if v != val {
			result = append(result, v)
		}
	}
	return result
}
