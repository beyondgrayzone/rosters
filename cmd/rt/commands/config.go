package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/models"
	"rosters/pkg/store"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func RegisterConfigCommand(rootCmd *cobra.Command) {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Read, write, and inspect .rosters/config.yaml",
	}

	schemaCmd := &cobra.Command{
		Use:   "schema",
		Short: "Emit the JSON Schema for .rosters/config.yaml",
		RunE:  runConfigSchema,
	}
	schemaCmd.Flags().Bool("json", false, "Compact single-line JSON")

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Print the current config (or a value at --path)",
		RunE:  runConfigShow,
	}
	showCmd.Flags().String("path", "", "Dot-path to read")
	showCmd.Flags().Bool("json", false, "Output as JSON")

	setCmd := &cobra.Command{
		Use:   "set <path> <value>",
		Short: "Set a config value at <path>; <value> is YAML-parsed",
		Args:  cobra.ExactArgs(2),
		RunE:  runConfigSet,
	}
	setCmd.Flags().Bool("json", false, "Output as JSON")

	unsetCmd := &cobra.Command{
		Use:   "unset <path>",
		Short: "Remove the config value at <path>",
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigUnset,
	}
	unsetCmd.Flags().Bool("json", false, "Output as JSON")

	configCmd.AddCommand(schemaCmd, showCmd, setCmd, unsetCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSchema(cmd *cobra.Command, args []string) error {
	jsonMode, _ := cmd.Flags().GetBool("json")
	schema := config.ConfigSchema()

	if jsonMode {
		b, _ := json.Marshal(schema)
		fmt.Println(string(b))
	} else {
		b, _ := json.MarshalIndent(schema, "", "  ")
		fmt.Println(string(b))
	}
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	pathArg, _ := cmd.Flags().GetString("path")
	jsonMode, _ := cmd.Flags().GetBool("json")

	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	raw, err := readRawConfig(rostersDir)
	if err != nil {
		return err
	}

	if pathArg != "" {
		parts := strings.Split(pathArg, ".")
		val, found := getAtPath(raw, parts)
		if !found {
			return fmt.Errorf("path not found: %s", pathArg)
		}

		if jsonMode {
			format.OutputJSON(map[string]any{
				"success": true,
				"command": "config show",
				"path":    pathArg,
				"value":   val,
			})
		} else {
			switch v := val.(type) {
			case map[string]any, []any:
				b, _ := yaml.Marshal(v)
				fmt.Print(string(b))
			default:
				fmt.Println(val)
			}
		}
		return nil
	}

	if jsonMode {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "config show",
			"config":  raw,
		})
	} else {
		b, _ := yaml.Marshal(raw)
		fmt.Print(string(b))
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	pathArg := args[0]
	valueArg := args[1]
	jsonMode, _ := cmd.Flags().GetBool("json")

	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	var parsedValue any
	if err := yaml.Unmarshal([]byte(valueArg), &parsedValue); err != nil {
		return fmt.Errorf("invalid YAML value: %v", err)
	}

	configPath := filepath.Join(rostersDir, models.ConfigFile)
	_, err = store.WithLock(configPath, func() (any, error) {
		raw, err := readRawConfig(rostersDir)
		if err != nil {
			return nil, err
		}

		parts := strings.Split(pathArg, ".")
		if err := setAtPath(raw, parts, parsedValue); err != nil {
			return nil, err
		}

		return nil, writeRawConfig(rostersDir, raw)
	})

	if err != nil {
		return err
	}

	if jsonMode {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "config set",
			"path":    pathArg,
			"value":   parsedValue,
		})
	} else {
		format.PrintSuccess(fmt.Sprintf("Set %s = %v", format.Accent.Sprint(pathArg), parsedValue))
	}

	return nil
}

func runConfigUnset(cmd *cobra.Command, args []string) error {
	pathArg := args[0]
	jsonMode, _ := cmd.Flags().GetBool("json")

	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	configPath := filepath.Join(rostersDir, models.ConfigFile)
	var removed bool
	_, err = store.WithLock(configPath, func() (any, error) {
		raw, err := readRawConfig(rostersDir)
		if err != nil {
			return nil, err
		}

		parts := strings.Split(pathArg, ".")
		removed = unsetAtPath(raw, parts)
		if !removed {
			return nil, nil
		}

		return nil, writeRawConfig(rostersDir, raw)
	})

	if err != nil {
		return err
	}

	if jsonMode {
		format.OutputJSON(map[string]any{
			"success": true,
			"command": "config unset",
			"path":    pathArg,
			"removed": removed,
		})
	} else if removed {
		format.PrintSuccess(fmt.Sprintf("Unset %s", format.Accent.Sprint(pathArg)))
	} else {
		format.PrintWarning(fmt.Sprintf("No such path: %s", pathArg))
	}

	return nil
}

func readRawConfig(rostersDir string) (map[string]any, error) {
	path := filepath.Join(rostersDir, models.ConfigFile)
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	if data == nil {
		data = make(map[string]any)
	}
	return data, nil
}

func writeRawConfig(rostersDir string, data map[string]any) error {
	path := filepath.Join(rostersDir, models.ConfigFile)
	b, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func getAtPath(data any, parts []string) (any, bool) {
	curr := data
	for _, p := range parts {
		m, ok := curr.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := m[p]
		if !exists {
			return nil, false
		}
		curr = val
	}
	return curr, true
}

func setAtPath(data map[string]any, parts []string, value any) error {
	curr := data
	for i := 0; i < len(parts)-1; i++ {
		p := parts[i]
		next, exists := curr[p]
		if !exists || next == nil {
			m := make(map[string]any)
			curr[p] = m
			curr = m
		} else if m, ok := next.(map[string]any); ok {
			curr = m
		} else {
			return fmt.Errorf("cannot set: '%s' is not an object", strings.Join(parts[:i+1], "."))
		}
	}
	curr[parts[len(parts)-1]] = value
	return nil
}

func unsetAtPath(data map[string]any, parts []string) bool {
	curr := data
	for i := 0; i < len(parts)-1; i++ {
		p := parts[i]
		next, exists := curr[p]
		if !exists {
			return false
		}
		m, ok := next.(map[string]any)
		if !ok {
			return false
		}
		curr = m
	}
	last := parts[len(parts)-1]
	if _, exists := curr[last]; exists {
		delete(curr, last)
		return true
	}
	return false
}
