package commands

import (
	"fmt"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

func RegisterBlockedCommand(rootCmd *cobra.Command) {
	blockedCmd := &cobra.Command{
		Use:   "blocked",
		Short: "Show all blocked issues",
		RunE:  runBlocked,
	}
	rootCmd.AddCommand(blockedCmd)
}

func runBlocked(cmd *cobra.Command, args []string) error {
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	issues, err := store.ReadIssues(rostersDir)
	if err != nil {
		return err
	}

	closedIds := make(map[string]bool)
	for _, i := range issues {
		if i.Status == "closed" {
			closedIds[i.ID] = true
		}
	}

	var blocked []models.Issue
	for _, i := range issues {
		if i.Status == "closed" {
			continue
		}
		isBlocked := false
		for _, bid := range i.BlockedBy {
			if !closedIds[bid] {
				isBlocked = true
				break
			}
		}
		if isBlocked {
			blocked = append(blocked, i)
		}
	}

	mode := format.GetFormat()

	switch mode {
	case "json":
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "blocked",
			"issues":  blocked,
			"count":   len(blocked),
		})
		return nil
	case "ids":
		for _, issue := range blocked {
			fmt.Println(issue.ID)
		}
		return nil
	case "compact":
		for _, issue := range blocked {
			fmt.Println(format.FormatIssueOneLineCompact(issue, closedIds))
		}
		return nil
	case "plain":
		if len(blocked) == 0 {
			fmt.Println("No blocked issues.")
			return nil
		}
		for _, issue := range blocked {
			fmt.Println(format.StripAnsi(format.FormatIssueOneLine(issue, closedIds)))
		}
		fmt.Printf("\n%d blocked issue(s)\n", len(blocked))
		return nil
	default:
		if len(blocked) == 0 {
			fmt.Println("No blocked issues.")
			return nil
		}
		for _, issue := range blocked {
			format.PrintIssueOneLine(issue, closedIds)
		}
		fmt.Printf("\n%d blocked issue(s)\n", len(blocked))
		return nil
	}
}
