package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rosters/pkg/config"
	"rosters/pkg/format"

	"github.com/spf13/cobra"
)

const primeFile = "PRIME.md"

type PrimeCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type PrimeCommandGroup struct {
	Name     string         `json:"name"`
	Commands []PrimeCommand `json:"commands"`
	Notes    []string       `json:"notes,omitempty"`
}

type PrimeWorkflow struct {
	Name     string   `json:"name"`
	Commands []string `json:"commands"`
}

type PrimeSectionsFull struct {
	Mode            string              `json:"mode"`
	Title           string              `json:"title"`
	ContextRecovery string              `json:"contextRecovery"`
	CloseProtocol   CloseProtocol       `json:"closeProtocol"`
	Rules           []string            `json:"rules"`
	CommandGroups   []PrimeCommandGroup `json:"commandGroups"`
	Workflows       []PrimeWorkflow     `json:"workflows"`
}

type CloseProtocol struct {
	Warning string   `json:"warning"`
	Steps   []string `json:"steps"`
	Footer  string   `json:"footer"`
}

type PrimeSectionsCompact struct {
	Mode         string         `json:"mode"`
	Title        string         `json:"title"`
	Commands     []PrimeCommand `json:"commands"`
	PlanningNote string         `json:"planningNote"`
	ClosingNote  string         `json:"closingNote"`
}

var fullSections = PrimeSectionsFull{
	Mode:            "full",
	Title:           "Rosters Workflow Context",
	ContextRecovery: "Run `rt prime` after compaction, clear, or new session",
	CloseProtocol: CloseProtocol{
		Warning: "Before saying \"done\" or \"complete\", you MUST run this checklist:",
		Steps: []string{
			"Close completed issues:    rt close <id1> <id2> ...",
			"File issues for remaining:  rt create --title \"...\"",
			"Run quality gates:          bun test && bun run lint && bun run typecheck",
			"Sync and push:              rt sync && git push",
			"Verify:                     git status (must show \"up to date with origin\")",
		},
		Footer: "**NEVER skip this.** Work is not done until pushed.",
	},
	Rules: []string{
		"**Default**: Use rosters for ALL task tracking (`rt create`, `rt ready`, `rt close`)",
		"**Prohibited**: Do NOT use TodoWrite, TaskCreate, or markdown files for task tracking",
		"**Workflow**: Create issues BEFORE writing code, mark in_progress when starting",
		"Git workflow: run `rt sync` at session end",
	},
	CommandGroups: []PrimeCommandGroup{
		{
			Name: "Finding Work",
			Commands: []PrimeCommand{
				{Command: "rt ready", Description: "Show issues ready to work (no blockers)"},
				{Command: "rt list --status=open", Description: "All open issues"},
				{Command: "rt list --status=in_progress", Description: "Your active work"},
				{
					Command:     "rt show <id> [<id2> ...]",
					Description: "Detailed issue view; multi-id shows each separated by a divider (`--json` returns `issues: [...]`)",
				},
			},
		},
		{
			Name: "Creating & Updating",
			Commands: []PrimeCommand{
				{
					Command:     "rt create --title=\"...\" --type=task|bug|feature|epic --priority=2",
					Description: "New issue\n  - Priority: 0-4 or P0-P4 (0=critical, 2=medium, 4=backlog)",
				},
				{Command: "rt update <id> --status=in_progress", Description: "Claim work"},
				{Command: "rt update <id> --assignee=username", Description: "Assign to someone"},
				{Command: "rt close <id>", Description: "Mark complete"},
				{Command: "rt close <id1> <id2> ...", Description: "Close multiple issues at once"},
			},
		},
		{
			Name: "Dependencies & Blocking",
			Commands: []PrimeCommand{
				{Command: "rt dep add <issue> <depends-on>", Description: "Add dependency"},
				{Command: "rt dep remove <issue> <depends-on>", Description: "Remove dependency"},
				{Command: "rt blocked", Description: "Show all blocked issues"},
			},
		},
		{
			Name: "Labels",
			Commands: []PrimeCommand{
				{Command: "rt label add <id> bug ui", Description: "Add labels to an issue"},
				{Command: "rt label remove <id> bug", Description: "Remove labels"},
				{Command: "rt label list <id>", Description: "List labels on an issue"},
				{Command: "rt label list-all", Description: "Show all labels in project"},
				{
					Command:     "rt list --label=bug",
					Description: "Filter by label (AND, comma-separated)",
				},
				{Command: "rt list --label-any=bug,ui", Description: "Filter by label (OR)"},
				{Command: "rt list --unlabeled", Description: "Issues with no labels"},
				{Command: "rt create --title=\"...\" --labels=bug,ui", Description: "Create with labels"},
			},
		},
		{
			Name: "Sync & Project Health",
			Commands: []PrimeCommand{
				{Command: "rt sync", Description: "Stage and commit .rosters/ changes"},
				{Command: "rt sync --status", Description: "Check without committing"},
				{Command: "rt stats", Description: "Project statistics"},
				{Command: "rt doctor", Description: "Check for data integrity issues"},
			},
		},
		{
			Name: "Planning",
			Notes: []string{
				"Use `rt plan` when work is large or ambiguous enough to benefit from structured decomposition. The plan spawns one child roster per step; `step.blocks` uses forward semantics (step i with `blocks: [j]` means step i blocks step j). For small, well-scoped tasks, just `rt create` directly.",
			},
			Commands: []PrimeCommand{
				{
					Command:     "rt plan templates",
					Description: "List built-in templates (`feature`, `bug`, `refactor`) plus custom ones",
				},
				{
					Command:     "rt plan prompt <roster-id>",
					Description: "Emit prompt JSON for the LLM to fill",
				},
				{
					Command:     "rt plan submit <roster-id> --plan <file>",
					Description: "Validate + spawn children",
				},
				{Command: "rt plan show <pl-id>", Description: "Sections, children, nested sub-plans"},
				{
					Command:     "rt plan outcome <pl-id> --result success|partial|failure",
					Description: "Storage-only outcome",
				},
				{
					Command:     "rt plan review <pl-id> --by <name>",
					Description: "Optional reviewer (informational)",
				},
			},
		},
	},
	Workflows: []PrimeWorkflow{
		{
			Name: "Starting work",
			Commands: []string{
				"rt ready                              # Find available work",
				"rt show <id>                          # Review issue details",
				"rt update <id> --status=in_progress   # Claim it",
			},
		},
		{
			Name: "Completing work",
			Commands: []string{
				"rt close <id1> <id2> ...    # Close all completed issues at once",
				"rt sync                     # Stage + commit .rosters/",
				"git push                    # Push to remote",
			},
		},
		{
			Name: "Creating dependent work",
			Commands: []string{
				"rt create --title=\"Implement feature X\" --type=feature",
				"rt create --title=\"Write tests for X\" --type=task",
				"rt dep add <test-id> <feature-id>   # Tests depend on feature",
			},
		},
	},
}

var compactSections = PrimeSectionsCompact{
	Mode:  "compact",
	Title: "Rosters Quick Reference",
	Commands: []PrimeCommand{
		{Command: "rt ready", Description: "Find unblocked work"},
		{Command: "rt show <id> [id...]", Description: "View one or more issues"},
		{Command: "rt create --title \"...\"", Description: "Create issue (--type, --priority)"},
		{Command: "rt update <id> --status in_progress", Description: "Claim work"},
		{Command: "rt close <id>", Description: "Complete work"},
		{Command: "rt dep add <a> <b>", Description: "a depends on b"},
		{Command: "rt blocked", Description: "Show blocked issues"},
		{Command: "rt label add <id> <l...>", Description: "Add labels"},
		{Command: "rt list --label=bug", Description: "Filter by label"},
		{
			Command:     "rt plan prompt <roster>",
			Description: "Plan large/ambiguous work; spawns child rosters",
		},
		{Command: "rt plan submit <roster> --plan <file>", Description: "Submit + spawn children"},
		{Command: "rt sync", Description: "Stage + commit .rosters/"},
	},
	PlanningNote: "**Planning:** Use `rt plan` for ambiguous or large work - built-in templates: `feature`, `bug`, `refactor`.",
	ClosingNote:  "**Before finishing:** `rt close <ids> && rt sync && git push`",
}

func RegisterPrimeCommand(rootCmd *cobra.Command) {
	primeCmd := &cobra.Command{
		Use:   "prime",
		Short: "Output AI agent context",
		RunE:  runPrime,
	}

	primeCmd.Flags().Bool("compact", false, "Condensed quick-reference output")
	primeCmd.Flags().Bool("export", false, "Output the default template")
	primeCmd.Flags().Bool("json", false, "Output as JSON")

	rootCmd.AddCommand(primeCmd)
}

func runPrime(cmd *cobra.Command, args []string) error {
	compact, _ := cmd.Flags().GetBool("compact")
	export, _ := cmd.Flags().GetBool("export")
	isJSON, _ := cmd.Flags().GetBool("json")

	if export {
		return outputDefault(compact, isJSON)
	}

	var customContent string
	rostersDir, err := config.FindRostersDir("")
	if err == nil {
		path := filepath.Join(rostersDir, primeFile)
		if b, err := os.ReadFile(path); err == nil {
			customContent = string(b)
		}
	}

	if customContent != "" {
		if isJSON {
			format.OutputJSON(map[string]any{
				"success":  true,
				"command":  "prime",
				"sections": nil,
				"content":  customContent,
			})
		} else {
			fmt.Print(customContent)
		}
		return nil
	}

	return outputDefault(compact, isJSON)
}

func outputDefault(compact bool, isJSON bool) error {
	var sections any
	if compact {
		sections = compactSections
	} else {
		sections = fullSections
	}

	content := renderSections(sections)

	if isJSON {
		format.OutputJSON(map[string]any{
			"success":  true,
			"command":  "prime",
			"sections": sections,
			"content":  content,
		})
	} else {
		fmt.Print(content)
	}
	return nil
}

func renderSections(sections any) string {
	switch s := sections.(type) {
	case PrimeSectionsFull:
		return renderFull(s)
	case PrimeSectionsCompact:
		return renderCompact(s)
	default:
		return ""
	}
}

func renderFull(s PrimeSectionsFull) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", s.Title))
	sb.WriteString(fmt.Sprintf("> **Context Recovery**: %s\n\n", s.ContextRecovery))

	sb.WriteString("# Session Close Protocol\n\n")
	sb.WriteString(fmt.Sprintf("**CRITICAL**: %s\n\n", s.CloseProtocol.Warning))
	sb.WriteString("```\n")
	for i, step := range s.CloseProtocol.Steps {
		sb.WriteString(fmt.Sprintf("[ ] %d. %s\n", i+1, step))
	}
	sb.WriteString("```\n\n")
	sb.WriteString(s.CloseProtocol.Footer + "\n\n")

	sb.WriteString("## Core Rules\n")
	for _, rule := range s.Rules {
		sb.WriteString(fmt.Sprintf("- %s\n", rule))
	}
	sb.WriteString("\n")

	sb.WriteString("## Essential Commands\n\n")
	for _, group := range s.CommandGroups {
		sb.WriteString(fmt.Sprintf("### %s\n", group.Name))
		if len(group.Notes) > 0 {
			for _, note := range group.Notes {
				sb.WriteString(note + "\n\n")
			}
		}
		for _, cmd := range group.Commands {
			sb.WriteString(fmt.Sprintf("- `%s` - %s\n", cmd.Command, cmd.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Common Workflows\n\n")
	for _, wf := range s.Workflows {
		sb.WriteString(fmt.Sprintf("**%s:**\n", wf.Name))
		sb.WriteString("```bash\n")
		for _, c := range wf.Commands {
			sb.WriteString(c + "\n")
		}
		sb.WriteString("```\n\n")
	}

	return sb.String()
}

func renderCompact(s PrimeSectionsCompact) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", s.Title))
	sb.WriteString("```\n")

	pad := 26
	for _, c := range s.Commands {
		cmdStr := c.Command
		if len(cmdStr) < pad {
			cmdStr = fmt.Sprintf("%-26s", cmdStr)
		} else {
			cmdStr = cmdStr + " "
		}
		sb.WriteString(fmt.Sprintf("%s# %s\n", cmdStr, c.Description))
	}
	sb.WriteString("```\n\n")
	sb.WriteString(s.PlanningNote + "\n\n")
	sb.WriteString(s.ClosingNote + "\n")

	return sb.String()
}
