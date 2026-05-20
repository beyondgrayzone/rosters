package commands

import (
	"fmt"
	"strings"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/git"
	"rosters/pkg/models"

	"github.com/spf13/cobra"
)

func RegisterSyncCommand(rootCmd *cobra.Command) {
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Stage and commit .rosters/ changes",
		RunE:  runSync,
	}

	syncCmd.Flags().Bool("status", false, "Check status without committing")
	syncCmd.Flags().Bool("dry-run", false, "Show what would be committed without committing")

	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	statusOnly, _ := cmd.Flags().GetBool("status")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	isJSON := format.GetFormat() == "json"

	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	projectRoot := config.ProjectRootFromRostersDir(rostersDir)

	if config.IsInsideWorktree("") {
		msg := "Inside a git worktree - skipping commit. Issues are stored in the main repo."
		if isJSON {
			format.OutputJSON(map[string]any{
				"success":   true,
				"command":   "sync",
				"committed": false,
				"worktree":  true,
				"message":   msg,
			})
		} else {
			format.PrintWarning(msg)
		}
		return nil
	}

	changed, err := git.Output(projectRoot, "status", "--porcelain", models.SeedsDirName+"/")
	if err != nil {
		changed = ""
	}
	changed = strings.TrimSpace(changed)

	if statusOnly {
		if isJSON {
			format.OutputJSON(map[string]any{
				"success":    true,
				"command":    "sync",
				"hasChanges": changed != "",
				"changes":    changed,
			})
		} else {
			if changed != "" {
				fmt.Println("Uncommitted .rosters/ changes:")
				fmt.Println(changed)
			} else {
				fmt.Println("No uncommitted .rosters/ changes.")
			}
		}
		return nil
	}

	if changed == "" {
		if isJSON {
			format.OutputJSON(map[string]any{
				"success":   true,
				"command":   "sync",
				"committed": false,
				"message":   "Nothing to commit",
			})
		} else {
			fmt.Println("No changes to commit.")
		}
		return nil
	}

	date := time.Now().Format("2006-01-02")
	msg := fmt.Sprintf("rosters: sync %s", date)

	if dryRun {
		if isJSON {
			format.OutputJSON(map[string]any{
				"success":     true,
				"command":     "sync",
				"dryRun":      true,
				"wouldCommit": true,
				"message":     msg,
				"changes":     changed,
			})
		} else {
			fmt.Println("Dry run - would commit:")
			fmt.Println(changed)
			fmt.Printf("Commit message: %s\n", msg)
		}
		return nil
	}

	if err := git.Run(projectRoot, "add", models.SeedsDirName+"/"); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	if err := git.Run(projectRoot, "commit", "-m", msg); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	if isJSON {
		format.OutputJSON(map[string]any{
			"success":   true,
			"command":   "sync",
			"committed": true,
			"message":   msg,
		})
	} else {
		fmt.Printf("Committed: %s\n", msg)
	}

	return nil
}
