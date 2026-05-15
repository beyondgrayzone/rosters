package commands

import (
	"fmt"
	"strings"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/store"
	"rosters/pkg/util"

	"github.com/spf13/cobra"
)

func RegisterTplCommand(rootCmd *cobra.Command) {
	tplCmd := &cobra.Command{
		Use:   "tpl",
		Short: "Manage issue templates (molecules)",
	}

	tplCmd.AddCommand(tplCreateCmd())
	tplCmd.AddCommand(tplStepCmd())
	tplCmd.AddCommand(tplListCmd())
	tplCmd.AddCommand(tplShowCmd())
	tplCmd.AddCommand(tplPourCmd())
	tplCmd.AddCommand(tplStatusCmd())

	rootCmd.AddCommand(tplCmd)
}

func tplCreateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new template",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := config.FindRostersDir("")
			if err != nil {
				return err
			}

			var createdID string
			_, err = store.WithLock(store.TemplatesPath(dir), func() (any, error) {
				templates, _ := store.ReadTemplates(dir)
				var ids []string
				for _, t := range templates {
					ids = append(ids, t.ID)
				}
				createdID = util.GenerateID("tpl", ids)
				tpl := models.Template{ID: createdID, Name: name, Steps: []models.TemplateStep{}}
				return nil, store.AppendTemplate(dir, tpl)
			})
			if err != nil {
				return err
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{"success": true, "command": "tpl create", "id": createdID})
			} else {
				format.PrintSuccess(fmt.Sprintf("Created template %s: %s", createdID, name))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Template name")
	cmd.MarkFlagRequired("name")
	return cmd
}

func tplStepCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "step",
		Short: "Manage template steps",
	}
	cmd.AddCommand(tplStepAddCmd())
	return cmd
}

func tplStepAddCmd() *cobra.Command {
	var title, stepType string
	var priority int
	cmd := &cobra.Command{
		Use:   "add <id>",
		Short: "Add a step to a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := config.FindRostersDir("")
			if err != nil {
				return err
			}
			templateID := args[0]

			validType := false
			for _, v := range models.ValidTypes {
				if v == stepType {
					validType = true
					break
				}
			}
			if !validType {
				return fmt.Errorf("type must be one of: %s", strings.Join(models.ValidTypes, ", "))
			}

			var stepCount int
			_, err = store.WithLock(store.TemplatesPath(dir), func() (any, error) {
				templates, _ := store.ReadTemplates(dir)
				idx := -1
				for i, t := range templates {
					if t.ID == templateID {
						idx = i
						break
					}
				}
				if idx == -1 {
					return nil, fmt.Errorf("template not found: %s", templateID)
				}
				step := models.TemplateStep{
					Title:    title,
					Type:     &stepType,
					Priority: &priority,
				}
				templates[idx].Steps = append(templates[idx].Steps, step)
				stepCount = len(templates[idx].Steps)
				return nil, store.WriteTemplates(dir, templates)
			})
			if err != nil {
				return err
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{"success": true, "command": "tpl step add", "id": templateID, "stepCount": stepCount})
			} else {
				format.PrintSuccess(fmt.Sprintf("Added step %d to %s: \"%s\"", stepCount, templateID, title))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Step title")
	cmd.Flags().StringVar(&stepType, "type", "task", "Step type")
	cmd.Flags().IntVar(&priority, "priority", 2, "Step priority")
	cmd.MarkFlagRequired("title")
	return cmd
}

func tplListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := config.FindRostersDir("")
			if err != nil {
				return err
			}
			templates, _ := store.ReadTemplates(dir)

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{"success": true, "command": "tpl list", "templates": templates, "count": len(templates)})
				return nil
			}

			if len(templates) == 0 {
				fmt.Println("No templates.")
				return nil
			}
			for _, tpl := range templates {
				fmt.Printf("%s  %s  %s\n", format.AccentBold(tpl.ID), tpl.Name, format.Muted.Sprintf("(%d steps)", len(tpl.Steps)))
			}
			return nil
		},
	}
}

func tplShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show template with steps",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := config.FindRostersDir("")
			if err != nil {
				return err
			}
			templates, _ := store.ReadTemplates(dir)
			var found *models.Template
			for _, t := range templates {
				if t.ID == args[0] {
					found = &t
					break
				}
			}
			if found == nil {
				return fmt.Errorf("template not found: %s", args[0])
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{"success": true, "command": "tpl show", "template": found})
				return nil
			}

			fmt.Printf("%s  %s\n", format.AccentBold(found.ID), found.Name)
			fmt.Println(format.Muted.Sprintf("Steps (%d):", len(found.Steps)))
			for i, step := range found.Steps {
				sType := "task"
				if step.Type != nil {
					sType = *step.Type
				}
				sPriority := 2
				if step.Priority != nil {
					sPriority = *step.Priority
				}
				fmt.Printf("  %d. %s  %s\n", i+1, step.Title, format.Muted.Sprintf("[%s P%d]", sType, sPriority))
			}
			return nil
		},
	}
}

func tplPourCmd() *cobra.Command {
	var prefix string
	cmd := &cobra.Command{
		Use:   "pour <id>",
		Short: "Instantiate template into issues",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := config.FindRostersDir("")
			if err != nil {
				return err
			}
			templateID := args[0]

			templates, _ := store.ReadTemplates(dir)
			var tpl *models.Template
			for _, t := range templates {
				if t.ID == templateID {
					tpl = &t
					break
				}
			}
			if tpl == nil {
				return fmt.Errorf("template not found: %s", templateID)
			}
			if len(tpl.Steps) == 0 {
				return fmt.Errorf("template %s has no steps", templateID)
			}

			cfg, _ := config.ReadConfig(dir)
			var createdIDs []string

			_, err = store.WithLock(store.IssuesPath(dir), func() (any, error) {
				issues, _ := store.ReadIssues(dir)
				var existingIDs []string
				for _, iss := range issues {
					existingIDs = append(existingIDs, iss.ID)
				}
				now := time.Now().Format(time.RFC3339)

				var newIssues []models.Issue
				for _, step := range tpl.Steps {
					allIDs := append(existingIDs, createdIDs...)
					id := util.GenerateID(cfg.Project, allIDs)
					title := strings.ReplaceAll(step.Title, "{prefix}", prefix)

					sType := "task"
					if step.Type != nil {
						sType = *step.Type
					}
					sPriority := 2
					if step.Priority != nil {
						sPriority = *step.Priority
					}

					issue := models.Issue{
						ID:        id,
						Title:     title,
						Status:    "open",
						Type:      sType,
						Priority:  sPriority,
						CreatedAt: now,
						UpdatedAt: now,
						Convoy:    &templateID,
					}
					newIssues = append(newIssues, issue)
					createdIDs = append(createdIDs, id)
				}

				for i := 1; i < len(newIssues); i++ {
					newIssues[i].BlockedBy = []string{newIssues[i-1].ID}
					newIssues[i-1].Blocks = []string{newIssues[i].ID}
				}

				allIssues := append(issues, newIssues...)
				return nil, store.WriteIssues(dir, allIssues)
			})
			if err != nil {
				return err
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{"success": true, "command": "tpl pour", "ids": createdIDs})
			} else {
				format.PrintSuccess(fmt.Sprintf("Poured template %s - created %d issues", format.Accent.Sprint(templateID), len(createdIDs)))
				for _, id := range createdIDs {
					fmt.Printf("  %s\n", format.Accent.Sprint(id))
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&prefix, "prefix", "", "Prefix for issue titles")
	cmd.MarkFlagRequired("prefix")
	return cmd
}

func tplStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <id>",
		Short: "Show convoy status for a template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := config.FindRostersDir("")
			if err != nil {
				return err
			}
			templateID := args[0]
			issues, _ := store.ReadIssues(dir)

			var convoyIssues []models.Issue
			for _, iss := range issues {
				if iss.Convoy != nil && *iss.Convoy == templateID {
					convoyIssues = append(convoyIssues, iss)
				}
			}

			if len(convoyIssues) == 0 {
				if format.GetFormat() == "json" {
					format.OutputJSON(map[string]any{"success": true, "command": "tpl status", "templateId": templateID, "total": 0, "issues": []string{}})
				} else {
					fmt.Printf("No issues found for convoy %s\n", templateID)
				}
				return nil
			}

			closedIDs := make(map[string]bool)
			for _, iss := range issues {
				if iss.Status == "closed" {
					closedIDs[iss.ID] = true
				}
			}

			var completed, inProgress, blocked int
			var convoyIDs []string
			for _, iss := range convoyIssues {
				convoyIDs = append(convoyIDs, iss.ID)
				if iss.Status == "closed" {
					completed++
				} else {
					if iss.Status == "in_progress" {
						inProgress++
					}
					isBlocked := false
					for _, bid := range iss.BlockedBy {
						if !closedIDs[bid] {
							isBlocked = true
							break
						}
					}
					if isBlocked {
						blocked++
					}
				}
			}

			status := models.ConvoyStatus{
				TemplateID: templateID,
				Total:      len(convoyIssues),
				Completed:  completed,
				InProgress: inProgress,
				Blocked:    blocked,
				Issues:     convoyIDs,
			}

			if format.GetFormat() == "json" {
				format.OutputJSON(map[string]any{"success": true, "command": "tpl status", "status": status})
			} else {
				fmt.Printf("%s %s\n", format.Muted.Sprint("Convoy:"), format.Accent.Sprint(templateID))
				fmt.Printf("  %s       %d\n", format.Muted.Sprint("Total:"), status.Total)
				fmt.Printf("  %s   %d\n", format.Muted.Sprint("Completed:"), status.Completed)
				fmt.Printf("  %s %d\n", format.Muted.Sprint("In progress:"), status.InProgress)
				fmt.Printf("  %s     %d\n", format.Muted.Sprint("Blocked:"), status.Blocked)
				fmt.Println(format.Muted.Sprint("  Issues:"))
				for _, iss := range convoyIssues {
					fmt.Print("    ")
					format.PrintIssueOneLine(iss, closedIDs)
				}
			}
			return nil
		},
	}
}
