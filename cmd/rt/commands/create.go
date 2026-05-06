package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/store"
	"rosters/pkg/util"

	"github.com/spf13/cobra"
)

func RegisterCreateCommand(rootCmd *cobra.Command) {
	var (
		title       string
		issueType   string
		priority    string
		assignee    string
		description string
		labels      string
	)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(title, issueType, priority, assignee, description, labels)
		},
	}

	createCmd.Flags().StringVar(&title, "title", "", "Issue title (required)")
	createCmd.Flags().StringVar(&issueType, "type", "task", "Issue type (task|bug|feature|epic)")
	createCmd.Flags().StringVar(&priority, "priority", "2", "Priority 0-4 or P0-P4")
	createCmd.Flags().StringVar(&assignee, "assignee", "", "Assignee name")
	createCmd.Flags().StringVar(&description, "description", "", "Issue description")
	createCmd.Flags().StringVar(&description, "desc", "", "Issue description (alias)")
	createCmd.Flags().StringVar(&description, "body", "", "Issue description (alias)")
	createCmd.Flags().StringVar(&labels, "labels", "", "Comma-separated labels")

	_ = createCmd.MarkFlagRequired("title")
	rootCmd.AddCommand(createCmd)
}

func runCreate(title, iType, prio, asgn, desc, lbls string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("--title is required")
	}

	validType := false
	for _, vt := range models.ValidTypes {
		if vt == iType {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("--type must be one of: %s", strings.Join(models.ValidTypes, ", "))
	}

	pVal, err := parsePriority(prio)
	if err != nil || pVal < 0 || pVal > 4 {
		return fmt.Errorf("--priority must be 0-4 or P0-P4")
	}

	dir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	cfg, err := config.ReadConfig(dir)
	if err != nil {
		return err
	}

	var createdID string
	_, err = store.WithLock(store.IssuesPath(dir), func() (any, error) {
		existing, err := store.ReadIssues(dir)
		if err != nil {
			return nil, err
		}

		var ids []string
		for _, i := range existing {
			ids = append(ids, i.ID)
		}

		createdID = util.GenerateID(cfg.Project, ids)
		now := time.Now().Format(time.RFC3339)

		issue := models.Issue{
			ID:        createdID,
			Title:     strings.TrimSpace(title),
			Status:    "open",
			Type:      iType,
			Priority:  pVal,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if asgn != "" {
			issue.Assignee = util.Ptr(asgn)
		}
		if desc != "" {
			issue.Description = util.Ptr(desc)
		}
		if lbls != "" {
			parts := strings.Split(lbls, ",")
			var clean []string
			for _, p := range parts {
				if t := strings.TrimSpace(strings.ToLower(p)); t != "" {
					clean = append(clean, t)
				}
			}
			if len(clean) > 0 {
				issue.Labels = clean
			}
		}

		return nil, store.AppendIssue(dir, issue)
	})

	if err != nil {
		return err
	}

	if format.GetFormat() == "json" {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "create",
			"id":      createdID,
		})
	} else {
		format.PrintSuccess(fmt.Sprintf("Created %s", createdID))
	}

	return nil
}

func parsePriority(val string) (int, error) {
	val = strings.ToUpper(val)
	if strings.HasPrefix(val, "P") {
		return strconv.Atoi(val[1:])
	}
	return strconv.Atoi(val)
}
