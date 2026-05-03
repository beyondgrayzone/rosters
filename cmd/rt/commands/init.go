package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/models"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func RegisterInitCommand(rootCmd *cobra.Command) {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize .rosters/ in current directory",
		RunE:  runInit,
	}
	initCmd.Flags().Bool("json", false, "Output as JSON")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	jsonMode, _ := cmd.Flags().GetBool("json")

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	rostersDir := filepath.Join(cwd, models.SeedsDirName)
	configPath := filepath.Join(rostersDir, models.ConfigFile)

	if _, err := os.Stat(configPath); err == nil {
		if jsonMode {
			format.OutputJSON(map[string]any{
				"success": true,
				"command": "init",
				"dir":     rostersDir,
			})
		} else {
			format.PrintSuccess(fmt.Sprintf("Already initialized: %s", rostersDir))
		}
		return nil
	}

	if err := os.MkdirAll(rostersDir, 0755); err != nil {
		return err
	}

	projectName := filepath.Base(cwd)
	configData := map[string]any{
		"project":        projectName,
		"version":        "1",
		"max_plan_depth": models.DefaultMaxPlanDepth,
	}

	configContent, err := yaml.Marshal(configData)
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, configContent, 0600); err != nil {
		return err
	}

	for _, file := range []string{models.IssuesFile, models.TemplatesFile, models.PlansFile} {
		if err := os.WriteFile(filepath.Join(rostersDir, file), []byte(""), 0600); err != nil {
			return err
		}
	}

	gitignorePath := filepath.Join(rostersDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("*.lock\n"), 0600); err != nil {
		return err
	}

	projectRoot := config.ProjectRootFromRostersDir(rostersDir)
	gitattrsPath := filepath.Join(projectRoot, ".gitattributes")
	entry := fmt.Sprintf("%s/%s merge=union\n%s/%s merge=union\n%s/%s merge=union\n",
		models.SeedsDirName, models.IssuesFile,
		models.SeedsDirName, models.TemplatesFile,
		models.SeedsDirName, models.PlansFile)

	if existing, err := os.ReadFile(gitattrsPath); err == nil {
		if !strings.Contains(string(existing), fmt.Sprintf("%s/%s", models.SeedsDirName, models.IssuesFile)) {
			if err := os.WriteFile(gitattrsPath, append(existing, []byte("\n"+entry)...), 0600); err != nil {
				return err
			}
		}
	} else if os.IsNotExist(err) {
		if err := os.WriteFile(gitattrsPath, []byte(entry), 0600); err != nil {
			return err
		}
	}

	if jsonMode {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "init",
			"dir":     rostersDir,
		})
	} else {
		format.PrintSuccess(fmt.Sprintf("Initialized .rosters/ in %s", cwd))
	}

	return nil
}
