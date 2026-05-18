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

	"github.com/spf13/cobra"
)

const (
	onboardSchema             = 1
	piPackageName             = "@bgz/rosters-cli"
	startMarker               = "<!-- rosters:start -->"
	endMarker                 = "<!-- rosters:end -->"
	legacyVersionMarkerPrefix = "<!-- rosters-onboard-v:"
)

var candidateFiles = []string{"CLAUDE.md", ".claude/CLAUDE.md", "AGENTS.md"}

type onboardOptions struct {
	stdout bool
	check  bool
	json   bool
}

func RegisterOnboardCommand(rootCmd *cobra.Command) {
	opts := &onboardOptions{}
	onboardCmd := &cobra.Command{
		Use:   "onboard",
		Short: "Add rosters section to CLAUDE.md / AGENTS.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOnboard(opts)
		},
	}

	onboardCmd.Flags().BoolVar(&opts.stdout, "stdout", false, "Print what would be written to stdout")
	onboardCmd.Flags().BoolVar(&opts.check, "check", false, "Check status without modifying files")
	onboardCmd.Flags().BoolVar(&opts.json, "json", false, "Output as JSON")

	rootCmd.AddCommand(onboardCmd)
}

func getVersionMarker() string {
	return fmt.Sprintf("<!-- rosters-onboard:v%s -->", models.Version)
}

func getSchemaMarker(variant string) string {
	suffix := ""
	if variant == "pi" {
		suffix = ":pi"
	}
	return fmt.Sprintf("<!-- rosters-onboard-schema:%d%s -->", onboardSchema, suffix)
}

func buildStandardSnippet() string {
	v := models.Version
	return fmt.Sprintf(`## Issue Tracking (Rosters)
%s
%s

This project uses [Rosters](https://github.com/beebomed/rosters) v%s for git-native issue tracking.

**At the start of every session**, run:
`+"```"+`
rt prime
`+"```"+`

This injects session context: rules, command reference, and workflows. Pass `+"`--format json|compact|markdown|plain|ids`"+` on any command for agent-friendly output.

**Quick reference:**
- `+"`rt ready`"+` - Find unblocked work
- `+"`rt search <query>`"+` - Full-text search across titles + descriptions
- `+"`rt create --title \"...\" --type task --priority 2`"+` - Create issue
- `+"`rt update <id> --status in_progress`"+` - Claim work
- `+"`rt close <id>`"+` - Complete work
- `+"`rt dep add <id> <depends-on>`"+` - Add dependency between issues
- `+"`rt sync`"+` - Sync with git (run before pushing)

### Planning
Use `+"`rt plan`"+` when work is large or ambiguous enough that an LLM benefits from structured decomposition. Submit spawns one child roster per step; `+"`step.blocks`"+` uses forward semantics (step i with `+"`blocks: [j]`"+` means step i blocks step j, and step j gets step i's id in its `+"`blockedBy`"+`).

- `+"`rt plan templates`"+` - List built-ins (`+"`feature`"+`, `+"`bug`"+`, `+"`refactor`"+`) plus custom templates
- `+"`rt plan prompt <roster-id>`"+` - Emit a structured prompt the LLM fills in
- `+"`rt plan submit <roster-id> --plan <file>`"+` - Validate + spawn child rosters
- `+"`rt plan show <pl-id>`"+` - View sections, children, sub-plans
- `+"`rt plan outcome <pl-id> --result success|partial|failure`"+` - Record outcome (storage-only)
- `+"`rt plan review <pl-id> --by <name>`"+` - Record reviewer (informational)

### Before You Finish
1. Close completed issues: `+"`rt close <id>`"+`
2. File issues for remaining work: `+"`rt create --title \"...\"`"+`
3. Sync and push: `+"`rt sync && git push`"+` `, getVersionMarker(), getSchemaMarker(""), v)
}

func buildPiSnippet() string {
	v := models.Version
	return fmt.Sprintf(`## Issue Tracking (Rosters)
%s
%s

This project uses [Rosters](https://github.com/beebomed/rosters) v%s via the in-tree
`+"`@bgz/pi-rosters`"+` pi-coding-agent extension. The extension auto-primes on `+"`session_start`"+`,
renders a `+"`sd: <n> ready / <n> in-progress / <n> blocked`"+` status widget, registers
`+"`sd_create`"+` / `+"`sd_ready`"+` / `+"`sd_show`"+` / `+"`sd_update`"+` / `+"`sd_close`"+` / `+"`sd_dep`"+` / `+"`sd_search`"+`
custom tools, expands `+"`#sd-<id>`"+` references on send, and ships `+"`/sd`"+`, `+"`/sd:ready`"+`,
`+"`/sd:create`"+`, `+"`/sd:show`"+`, `+"`/sd:close`"+`, `+"`/sd:claim`"+` slash commands.

**Manual escape hatches** (rarely needed - the extension handles the rituals):

- `+"`rt ready`"+` - Find unblocked work from the shell.
- `+"`rt create --title \"...\"`"+` / `+"`rt close <id>`"+` - Create or close from the shell.
- `+"`rt sync`"+` - Stage and commit `+"`.rosters/`"+` changes before `+"`git push`"+`.

Configuration lives under `+"`pi.*`"+` in `+"`.rosters/config.yaml`"+`. Run `+"`rt setup pi --check`"+` to verify
the install state; `+"`rt setup pi --remove`"+` reverts to the standalone CLI snippet.

### Before You Finish
1. Close completed issues: `+"`rt close <id>`"+`
2. File issues for remaining work: `+"`rt create --title \"...\"`"+`
3. Sync and push: `+"`rt sync && git push`"+` `, getVersionMarker(), getSchemaMarker("pi"), v)
}

func isPiInstalled(projectRoot string) bool {
	settingsPath := filepath.Join(projectRoot, ".pi", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return false
	}
	var parsed struct {
		Packages []any `json:"packages"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return false
	}
	for _, p := range parsed.Packages {
		if s, ok := p.(string); ok && s == piPackageName {
			return true
		}
		if m, ok := p.(map[string]any); ok {
			if src, ok := m["source"].(string); ok && src == piPackageName {
				return true
			}
		}
	}
	return false
}

func findTargetFile(projectRoot string) string {
	for _, c := range candidateFiles {
		p := filepath.Join(projectRoot, c)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func detectStatus(content string, variant string) string {
	if !strings.Contains(content, startMarker) || !strings.Contains(content, endMarker) {
		return "missing"
	}
	if strings.Contains(content, legacyVersionMarkerPrefix) {
		return "outdated"
	}
	if strings.Contains(content, getSchemaMarker(variant)) {
		return "current"
	}
	return "outdated"
}

func wrapInMarkers(snippet string) string {
	return fmt.Sprintf("%s\n%s\n%s", startMarker, snippet, endMarker)
}

func replaceMarkerSection(content, snippet string) string {
	startIdx := strings.Index(content, startMarker)
	endIdx := strings.Index(content, endMarker)
	if startIdx == -1 || endIdx == -1 {
		return ""
	}
	return content[:startIdx] + wrapInMarkers(snippet) + content[endIdx+len(endMarker):]
}

func runOnboard(opts *onboardOptions) error {
	rostersDir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}
	projectRoot := config.ProjectRootFromRostersDir(rostersDir)

	variant := ""
	if isPiInstalled(projectRoot) {
		variant = "pi"
	}

	snippet := buildStandardSnippet()
	if variant == "pi" {
		snippet = buildPiSnippet()
	}

	targetPath := findTargetFile(projectRoot)

	if opts.check {
		status := "missing"
		if targetPath != "" {
			content, _ := os.ReadFile(targetPath)
			status = detectStatus(string(content), variant)
		}
		if opts.json {
			format.OutputJSON(map[string]any{
				"success": true,
				"command": "onboard",
				"status":  status,
				"file":    targetPath,
			})
		} else {
			if targetPath == "" {
				fmt.Println("Status: missing (no CLAUDE.md found)")
			} else {
				fmt.Printf("Status: %s (%s)\n", status, targetPath)
			}
		}
		return nil
	}

	if opts.stdout {
		fmt.Println(wrapInMarkers(snippet))
		return nil
	}

	filePath := targetPath
	if filePath == "" {
		filePath = filepath.Join(projectRoot, "AGENTS.md")
	}

	wrapped := wrapInMarkers(snippet)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := os.WriteFile(filePath, []byte(wrapped+"\n"), 0644); err != nil {
			return err
		}
		if opts.json {
			format.OutputJSON(map[string]any{"success": true, "command": "onboard", "action": "created", "file": filePath})
		} else {
			format.PrintSuccess(fmt.Sprintf("Created %s with rosters section", filePath))
		}
		return nil
	}

	data, _ := os.ReadFile(filePath)
	content := string(data)
	status := detectStatus(content, variant)

	if status == "current" {
		if opts.json {
			format.OutputJSON(map[string]any{"success": true, "command": "onboard", "action": "unchanged", "file": filePath})
		} else {
			format.PrintSuccess("Rosters section is already up to date")
		}
		return nil
	}

	if status == "outdated" {
		updated := replaceMarkerSection(content, snippet)
		if err := os.WriteFile(filePath, []byte(updated), 0644); err != nil {
			return err
		}
		if opts.json {
			format.OutputJSON(map[string]any{"success": true, "command": "onboard", "action": "updated", "file": filePath})
		} else {
			format.PrintSuccess(fmt.Sprintf("Updated rosters section in %s", filePath))
		}
		return nil
	}

	sep := "\n"
	if !strings.HasSuffix(content, "\n") {
		sep = "\n\n"
	}
	if err := os.WriteFile(filePath, []byte(content+sep+wrapped+"\n"), 0644); err != nil {
		return err
	}
	if opts.json {
		format.OutputJSON(map[string]any{"success": true, "command": "onboard", "action": "appended", "file": filePath})
	} else {
		format.PrintSuccess(fmt.Sprintf("Added rosters section to %s", filePath))
	}

	return nil
}
