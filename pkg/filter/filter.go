package filter

import (
	"fmt"
	"strconv"
	"strings"

	"rosters/pkg/models"
)

type IssueFilterOptions struct {
	Type        *string
	Assignee    *string
	Label       *string
	LabelAny    *string
	Unlabeled   bool
	Priority    map[int]bool
	PriorityMax *int
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

func ParsePriorityToken(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	stripped := trimmed
	if strings.HasPrefix(strings.ToUpper(trimmed), "P") {
		stripped = trimmed[1:]
	}
	n, err := strconv.Atoi(stripped)
	if err != nil || n < 0 || n > 4 {
		return 0, fmt.Errorf("invalid priority %q: must be 0-4 or P0-P4", trimmed)
	}
	return n, nil
}

func ApplyIssueFilters(issues []models.Issue, opts IssueFilterOptions) []models.Issue {
	var result []models.Issue
	for _, i := range issues {
		if opts.Type != nil && i.Type != *opts.Type {
			continue
		}
		if opts.Assignee != nil && (i.Assignee == nil || *i.Assignee != *opts.Assignee) {
			continue
		}
		if opts.Label != nil {
			required := splitLabels(*opts.Label)
			labels := make(map[string]bool)
			for _, l := range i.Labels {
				labels[l] = true
			}
			match := true
			for _, r := range required {
				if !labels[r] {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		if opts.LabelAny != nil {
			anyOf := splitLabels(*opts.LabelAny)
			labels := make(map[string]bool)
			for _, l := range i.Labels {
				labels[l] = true
			}
			match := false
			for _, r := range anyOf {
				if labels[r] {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if opts.Unlabeled && len(i.Labels) > 0 {
			continue
		}
		if len(opts.Priority) > 0 {
			if !opts.Priority[i.Priority] {
				continue
			}
		}
		if opts.PriorityMax != nil && i.Priority > *opts.PriorityMax {
			continue
		}
		result = append(result, i)
	}
	return result
}
