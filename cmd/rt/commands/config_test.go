package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rosters/pkg/models"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestConfigCommands(t *testing.T) {
	tmpDir := t.TempDir()
	rostersDir := filepath.Join(tmpDir, ".rosters")
	os.MkdirAll(rostersDir, 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	configPath := filepath.Join(rostersDir, models.ConfigFile)
	initialConfig := map[string]any{
		"project":        "test-proj",
		"version":        "1",
		"max_plan_depth": 3,
	}
	b, _ := yaml.Marshal(initialConfig)
	os.WriteFile(configPath, b, 0644)

	t.Run("config show returns whole config", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("path", "", "")
		cmd.Flags().Bool("json", false, "")

		err := runConfigShow(cmd, []string{})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("config show returns specific path", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().String("path", "project", "")
		cmd.Flags().Bool("json", false, "")

		err := runConfigShow(cmd, []string{})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("config set updates value", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("json", false, "")

		err := runConfigSet(cmd, []string{"max_plan_depth", "10"})
		if err != nil {
			t.Fatal(err)
		}

		content, _ := os.ReadFile(configPath)
		if !strings.Contains(string(content), "max_plan_depth: 10") {
			t.Errorf("config not updated: %s", string(content))
		}
	})

	t.Run("config set creates nested objects", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("json", false, "")

		err := runConfigSet(cmd, []string{"pi.auto_prime", "false"})
		if err != nil {
			t.Fatal(err)
		}

		content, _ := os.ReadFile(configPath)
		if !strings.Contains(string(content), "auto_prime: false") {
			t.Errorf("nested config not created: %s", string(content))
		}
	})

	t.Run("config unset removes key", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("json", false, "")

		err := runConfigUnset(cmd, []string{"pi.auto_prime"})
		if err != nil {
			t.Fatal(err)
		}

		content, _ := os.ReadFile(configPath)
		if strings.Contains(string(content), "auto_prime") {
			t.Errorf("key not removed: %s", string(content))
		}
	})

	t.Run("config schema returns valid map", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.Flags().Bool("json", false, "")
		err := runConfigSchema(cmd, []string{})
		if err != nil {
			t.Fatal(err)
		}
	})
}
