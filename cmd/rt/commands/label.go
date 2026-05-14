package commands

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

func RegisterLabelCommand(rootCmd *cobra.Command) {
	labelCmd := &cobra.Command{
		Use:   "label",
		Short: "Manage issue labels",
	}

	addCmd := &cobra.Command{
		Use:   "add <id> <label...>",
		Short: "Add labels to an issue",
		Args:  cobra.MinimumNArgs(2),
		RunE:  runLabelAdd,
	}

	removeCmd := &cobra.Command{
		Use:   "remove <id> <label...>",
		Short: "Remove labels from an issue",
		Args:  cobra.MinimumNArgs(2),
		RunE:  runLabelRemove,
	}

	listCmd := &cobra.Command{
		Use:   "list <id>",
		Short: "List labels on an issue",
		Args:  cobra.ExactArgs(1),
		RunE:  runLabelList,
	}

	listAllCmd := &cobra.Command{
		Use:   "list-all",
		Short: "List all labels used in the project",
		RunE:  runLabelListAll,
	}

	labelCmd.AddCommand(addCmd, removeCmd, listCmd, listAllCmd)
	rootCmd.AddCommand(labelCmd)
}

func normalizeLabels(raw []string) []string {
	var result []string
	for _, l := range raw {
		trimmed := strings.ToLower(strings.TrimSpace(l))
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func runLabelAdd(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	newLabels := normalizeLabels(args[1:])
	if len(newLabels) == 0 {
		return fmt.Errorf("no valid labels provided")
	}

	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	_, err = store.WithLock(store.IssuesPath(rostersDir), func() (any, error) {
		issues, err := store.ReadIssues(rostersDir)
		if err != nil {
			return nil, err
		}

		idx := -1
		for i, iss := range issues {
			if iss.ID == issueID {
				idx = i
				break
			}
		}

		if idx == -1 {
			return nil, fmt.Errorf("issue not found: %s", issueID)
		}

		labelMap := make(map[string]bool)
		for _, l := range issues[idx].Labels {
			labelMap[l] = true
		}
		for _, l := range newLabels {
			labelMap[l] = true
		}

		var merged []string
		for l := range labelMap {
			merged = append(merged, l)
		}
		sort.Strings(merged)

		issues[idx].Labels = merged
		issues[idx].UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		return nil, store.WriteIssues(rostersDir, issues)
	})

	if err != nil {
		return err
	}

	if format.GetFormat() == "json" {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "label add",
			"issueId": issueID,
			"labels":  newLabels,
		})
	} else {
		var colored []string
		for _, l := range newLabels {
			colored = append(colored, format.Accent.Sprint(l))
		}
		format.PrintSuccess(fmt.Sprintf("Added label(s) %s to %s", strings.Join(colored, ", "), format.Accent.Sprint(issueID)))
	}

	return nil
}

func runLabelRemove(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	toRemove := normalizeLabels(args[1:])
	removeSet := make(map[string]bool)
	for _, l := range toRemove {
		removeSet[l] = true
	}

	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	_, err = store.WithLock(store.IssuesPath(rostersDir), func() (any, error) {
		issues, err := store.ReadIssues(rostersDir)
		if err != nil {
			return nil, err
		}

		idx := -1
		for i, iss := range issues {
			if iss.ID == issueID {
				idx = i
				break
			}
		}

		if idx == -1 {
			return nil, fmt.Errorf("issue not found: %s", issueID)
		}

		var remaining []string
		for _, l := range issues[idx].Labels {
			if !removeSet[l] {
				remaining = append(remaining, l)
			}
		}

		issues[idx].Labels = remaining
		issues[idx].UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		return nil, store.WriteIssues(rostersDir, issues)
	})

	if err != nil {
		return err
	}

	if format.GetFormat() == "json" {
		var labels []string
		for l := range removeSet {
			labels = append(labels, l)
		}
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "label remove",
			"issueId": issueID,
			"labels":  labels,
		})
	} else {
		format.PrintSuccess(fmt.Sprintf("Removed label(s) from %s", format.Accent.Sprint(issueID)))
	}

	return nil
}

func runLabelList(cmd *cobra.Command, args []string) error {
	issueID := args[0]
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	issues, err := store.ReadIssues(rostersDir)
	if err != nil {
		return err
	}

	var found *models.Issue
	for _, iss := range issues {
		if iss.ID == issueID {
			found = &iss
			break
		}
	}

	if found == nil {
		return fmt.Errorf("issue not found: %s", issueID)
	}

	labels := found.Labels
	if labels == nil {
		labels = []string{}
	}

	if format.GetFormat() == "json" {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "label list",
			"issueId": issueID,
			"labels":  labels,
		})
	} else {
		if len(labels) == 0 {
			fmt.Printf("%s has no labels.\n", format.AccentBold(issueID))
		} else {
			fmt.Printf("%s %s\n", format.AccentBold(issueID), format.Muted.Sprint("labels:"))
			for _, l := range labels {
				fmt.Printf("  %s\n", format.Accent.Sprint(l))
			}
		}
	}

	return nil
}

func runLabelListAll(cmd *cobra.Command, args []string) error {
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	issues, err := store.ReadIssues(rostersDir)
	if err != nil {
		return err
	}

	counts := make(map[string]int)
	for _, iss := range issues {
		for _, l := range iss.Labels {
			counts[l]++
		}
	}

	var labels []string
	for l := range counts {
		labels = append(labels, l)
	}
	sort.Strings(labels)

	if format.GetFormat() == "json" {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "label list-all",
			"labels":  labels,
			"counts":  counts,
		})
	} else {
		if len(labels) == 0 {
			fmt.Println("No labels found.")
			return nil
		}
		for _, l := range labels {
			fmt.Printf("  %-20s %s\n", format.Accent.Sprint(l), format.Muted.Sprint(counts[l]))
		}
		fmt.Printf("\n%d label(s)\n", len(labels))
	}

	return nil
}
