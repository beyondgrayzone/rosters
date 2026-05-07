package format

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"rosters/pkg/models"

	"github.com/fatih/color"
)

var (
	Brand  = color.New(color.FgHiGreen)
	Accent = color.New(color.FgHiYellow)
	Muted  = color.New(color.FgHiBlack)
	Error  = color.New(color.FgHiRed)

	quietMode  = false
	jsonMode   = false
	formatMode = "markdown"
	ansiRegex  = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

func StripAnsi(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func AccentBold(s string) string {
	return Accent.Add(color.Bold).Sprint(s)
}

func SetQuiet(v bool) {
	quietMode = v
}

func SetJSONMode(v bool) {
	jsonMode = v
}

func SetFormat(mode string) {
	formatMode = mode
	if mode == "plain" {
		color.NoColor = true
	}
	if mode == "json" {
		jsonMode = true
	}
}

func GetFormat() string {
	return formatMode
}

func OutputJSON(data any) {
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))
}

func PrintSuccess(msg string) {
	if quietMode || jsonMode {
		return
	}
	Brand.Printf("✓ %s\n", msg)
}

func PrintError(msg string) {
	color.New(color.FgRed).Printf("✗ %s\n", msg)
}

func PrintWarning(msg string) {
	if jsonMode {
		return
	}
	color.New(color.FgYellow).Printf("! %s\n", msg)
}

func PrintTiming(d time.Duration) {
	if !quietMode {
		Muted.Fprintf(os.Stderr, "Done in %dms\n", d.Milliseconds())
	}
}

func FormatIssueFull(issue models.Issue) string {
	statusColor := Brand
	if issue.Status == "closed" {
		statusColor = Muted
	} else if issue.Status == "in_progress" {
		statusColor = color.New(color.FgCyan)
	}

	priorityLabel, ok := models.PriorityLabels[issue.Priority]
	if !ok {
		priorityLabel = fmt.Sprintf("%d", issue.Priority)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("%s  %s", AccentBold(issue.ID), statusColor.Sprint(issue.Status)))
	lines = append(lines, fmt.Sprintf("Title:    %s", issue.Title))
	lines = append(lines, fmt.Sprintf("Type:     %s   Priority: %s", Muted.Sprint(issue.Type), Muted.Sprint(priorityLabel)))

	if issue.Assignee != nil {
		lines = append(lines, fmt.Sprintf("Assignee: %s", *issue.Assignee))
	}
	if len(issue.Labels) > 0 {
		var colored []string
		for _, l := range issue.Labels {
			colored = append(colored, Accent.Sprint(l))
		}
		lines = append(lines, fmt.Sprintf("Labels:   %s", strings.Join(colored, ", ")))
	}

	if issue.Description != nil && *issue.Description != "" {
		lines = append(lines, fmt.Sprintf("\n%s", *issue.Description))
	}

	if len(issue.BlockedBy) > 0 {
		var ids []string
		for _, id := range issue.BlockedBy {
			ids = append(ids, Accent.Sprint(id))
		}
		lines = append(lines, fmt.Sprintf("Blocked by: %s", strings.Join(ids, ", ")))
	}
	if len(issue.Blocks) > 0 {
		var ids []string
		for _, id := range issue.Blocks {
			ids = append(ids, Accent.Sprint(id))
		}
		lines = append(lines, fmt.Sprintf("Blocks:     %s", strings.Join(ids, ", ")))
	}

	if issue.Convoy != nil {
		lines = append(lines, fmt.Sprintf("Convoy:   %s", Muted.Sprint(*issue.Convoy)))
	}
	if issue.CloseReason != nil {
		lines = append(lines, fmt.Sprintf("Reason:   %s", *issue.CloseReason))
	}

	lines = append(lines, fmt.Sprintf("Created:  %s", Muted.Sprint(issue.CreatedAt)))
	lines = append(lines, fmt.Sprintf("Updated:  %s", Muted.Sprint(issue.UpdatedAt)))
	if issue.ClosedAt != nil {
		lines = append(lines, fmt.Sprintf("Closed:   %s", Muted.Sprint(*issue.ClosedAt)))
	}

	return strings.Join(lines, "\n")
}

func RenderPlanBlock(plan *models.Plan, children []any) string {
	if plan == nil {
		return ""
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s %s  %s", Brand.Sprint("Plan:"), Accent.Sprint(plan.ID), Muted.Sprintf("[%s]", plan.Status)))

	if plan.Status == models.PlanStatusDraft {
		lines = append(lines, Accent.Sprint("plan in draft  run rt plan submit"))
	} else if len(children) > 0 {
		lines = append(lines, Muted.Sprintf("Plan steps (%d):", len(children)))
		for _, c := range children {
			child := c.(map[string]any)
			tag := ""
			if adopted, ok := child["adopted"].(bool); ok && adopted {
				tag = fmt.Sprintf(" %s", Muted.Sprint("(adopted)"))
			}
			lines = append(lines, fmt.Sprintf("  %s  %s  %s%s",
				Accent.Sprint(child["id"]),
				Muted.Sprintf("[%s]", child["status"]),
				child["title"],
				tag))
		}
	}

	return "\n" + strings.Join(lines, "\n") + "\n"
}
