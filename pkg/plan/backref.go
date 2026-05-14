package plan

import (
	"fmt"
	"strings"
)

const (
	BackrefStart = "<!-- rosters:plan-backref:start -->"
	BackrefEnd   = "<!-- rosters:plan-backref:end -->"
	MaxExcerpt   = 240
)

type BackrefArgs struct {
	StepIndex         *int
	PlanID            string
	ParentRosterID    string
	ParentRosterTitle string
	TemplateName      string
	Approach          any
}

func BuildPlanBackref(args BackrefArgs) string {
	var lines []string
	if args.StepIndex != nil {
		lines = append(lines, fmt.Sprintf("Step %d of plan %s.", *args.StepIndex+1, args.PlanID))
	} else {
		lines = append(lines, fmt.Sprintf("Adopted into plan %s.", args.PlanID))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Parent roster: %s - %s", args.ParentRosterID, args.ParentRosterTitle))
	lines = append(lines, fmt.Sprintf("Plan template: %s", args.TemplateName))

	excerpt := approachExcerpt(args.Approach)
	if excerpt != "" {
		lines = append(lines, fmt.Sprintf("Plan approach: %s", excerpt))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Run `rt plan show %s` for the full plan (context, alternatives, sibling steps, acceptance criteria).", args.PlanID))

	body := strings.Join(lines, "\n")
	return fmt.Sprintf("%s\n%s\n%s", BackrefStart, body, BackrefEnd)
}

func ApplyPlanBackref(existing string, args BackrefArgs) string {
	block := BuildPlanBackref(args)
	if strings.Contains(existing, BackrefStart) && strings.Contains(existing, BackrefEnd) {
		return replaceBackrefSection(existing, block)
	}
	if strings.TrimSpace(existing) == "" {
		return block
	}
	return fmt.Sprintf("%s\n\n%s", block, existing)
}

func StripPlanBackref(existing string) string {
	startIdx := strings.Index(existing, BackrefStart)
	endIdx := strings.Index(existing, BackrefEnd)
	if startIdx == -1 || endIdx == -1 {
		return existing
	}

	before := strings.TrimRight(existing[:startIdx], " \n\r\t")
	after := strings.TrimLeft(existing[endIdx+len(BackrefEnd):], " \n\r\t")

	if before == "" && after == "" {
		return ""
	}
	if before == "" {
		return after
	}
	if after == "" {
		return before
	}
	return before + "\n\n" + after
}

func replaceBackrefSection(existing, block string) string {
	startIdx := strings.Index(existing, BackrefStart)
	endIdx := strings.Index(existing, BackrefEnd)
	if startIdx == -1 || endIdx == -1 {
		return block
	}
	return existing[:startIdx] + block + existing[endIdx+len(BackrefEnd):]
}

func approachExcerpt(value any) string {
	s, ok := value.(string)
	if !ok {
		return ""
	}
	collapsed := strings.Join(strings.Fields(s), " ")
	if len(collapsed) <= MaxExcerpt {
		return collapsed
	}
	slice := collapsed[:MaxExcerpt]
	lastSpace := strings.LastIndex(slice, " ")
	if lastSpace > MaxExcerpt/2 {
		return strings.TrimRight(slice[:lastSpace], " ") + "…"
	}
	return strings.TrimRight(slice, " ") + "…"
}
