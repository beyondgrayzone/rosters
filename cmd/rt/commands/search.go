package commands

import (
	"fmt"
	"strings"

	"rosters/pkg/config"
	"rosters/pkg/filter"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/sort"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
)

var (
	searchStatus      string
	searchType        string
	searchAssignee    string
	searchLabel       string
	searchLabelAny    string
	searchUnlabeled   bool
	searchPriority    string
	searchPriorityMax string
	searchLimit       int
	searchSort        string
)

func RegisterSearchCommand(rootCmd *cobra.Command) {
	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search title + description",
		Args:  cobra.ExactArgs(1),
		RunE:  runSearch,
	}

	searchCmd.Flags().StringVar(&searchStatus, "status", "", "Filter by status (open|in_progress|closed)")
	searchCmd.Flags().StringVar(&searchType, "type", "", "Filter by type (task|bug|feature|epic)")
	searchCmd.Flags().StringVar(&searchAssignee, "assignee", "", "Filter by assignee")
	searchCmd.Flags().StringVar(&searchLabel, "label", "", "Filter: must have ALL labels (comma-separated, AND)")
	searchCmd.Flags().StringVar(&searchLabelAny, "label-any", "", "Filter: must have any label (comma-separated, OR)")
	searchCmd.Flags().BoolVar(&searchUnlabeled, "unlabeled", false, "Filter: issues with no labels")
	searchCmd.Flags().StringVar(&searchPriority, "priority", "", "Filter by priority (comma-separated, e.g. 0,1 or P0,P1)")
	searchCmd.Flags().StringVar(&searchPriorityMax, "priority-max", "", "Filter to priority <= n (e.g. --priority-max 1 = P0+P1)")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 50, "Max issues to show")
	searchCmd.Flags().StringVar(&searchSort, "sort", "priority", "Sort order (priority|created|updated|id)")

	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.ToLower(args[0])
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

	opts, err := buildFilterOptions()
	if err != nil {
		return err
	}

	var filtered []models.Issue
	for _, i := range allIssues {
		if searchStatus != "" && i.Status != searchStatus {
			continue
		}

		match := strings.Contains(strings.ToLower(i.Title), query)
		if !match && i.Description != nil {
			match = strings.Contains(strings.ToLower(*i.Description), query)
		}

		if match {
			filtered = append(filtered, i)
		}
	}

	issues := filter.ApplyIssueFilters(filtered, opts)

	if !sort.IsSortMode(searchSort) {
		return fmt.Errorf("invalid --sort value: %s. Valid: %s", searchSort, strings.Join(sort.ValidSortModes, "|"))
	}
	issues = sort.SortIssues(issues, sort.SortMode(searchSort))

	if len(issues) > searchLimit {
		issues = issues[:searchLimit]
	}

	return outputSearchResults(issues, query, closedBlockerIds)
}

func buildFilterOptions() (filter.IssueFilterOptions, error) {
	opts := filter.IssueFilterOptions{
		Unlabeled: searchUnlabeled,
	}

	if searchType != "" {
		opts.Type = &searchType
	}
	if searchAssignee != "" {
		opts.Assignee = &searchAssignee
	}
	if searchLabel != "" {
		opts.Label = &searchLabel
	}
	if searchLabelAny != "" {
		opts.LabelAny = &searchLabelAny
	}

	if searchPriority != "" {
		opts.Priority = make(map[int]bool)
		for _, p := range strings.Split(searchPriority, ",") {
			val, err := filter.ParsePriorityToken(p)
			if err != nil {
				return opts, err
			}
			opts.Priority[val] = true
		}
	}

	if searchPriorityMax != "" {
		val, err := filter.ParsePriorityToken(searchPriorityMax)
		if err != nil {
			return opts, err
		}
		opts.PriorityMax = &val
	}

	return opts, nil
}

func outputSearchResults(issues []models.Issue, query string, closedBlockerIds map[string]bool) error {
	fmtMode := format.GetFormat()

	if fmtMode == "json" {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "search",
			"query":   query,
			"issues":  issues,
			"count":   len(issues),
		})
		return nil
	}

	if len(issues) == 0 {
		if fmtMode != "ids" {
			fmt.Printf("No issues match \"%s\".\n", query)
		}
		return nil
	}

	for _, issue := range issues {
		switch fmtMode {
		case "ids":
			fmt.Println(issue.ID)
		case "compact":
			fmt.Println(format.FormatIssueOneLineCompact(issue, closedBlockerIds))
		case "plain":
			fmt.Println(format.StripAnsi(format.FormatIssueOneLine(issue, closedBlockerIds)))
		default:
			format.PrintIssueOneLine(issue, closedBlockerIds)
		}
	}

	if fmtMode != "ids" && fmtMode != "compact" {
		fmt.Printf("\n%d match(es)\n", len(issues))
	}

	return nil
}
