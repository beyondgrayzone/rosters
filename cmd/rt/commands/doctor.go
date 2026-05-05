package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"rosters/pkg/config"
	"rosters/pkg/format"
	"rosters/pkg/models"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"` // pass, warn, fail
	Message string   `json:"message"`
	Details []string `json:"details"`
	Fixable bool     `json:"fixable"`
}

type rawLine struct {
	LineNumber int
	Text       string
	Parsed     any
	Error      string
}

func RegisterDoctorCommand(rootCmd *cobra.Command) {
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check project health and data integrity",
		RunE:  runDoctor,
	}

	doctorCmd.Flags().Bool("fix", false, "Auto-fix fixable issues")
	doctorCmd.Flags().Bool("verbose", false, "Show all check results including passes")
	doctorCmd.Flags().Bool("json", false, "Output as JSON")

	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	fixMode, _ := cmd.Flags().GetBool("fix")
	verbose, _ := cmd.Flags().GetBool("verbose")
	isJSON, _ := cmd.Flags().GetBool("json")

	dir, err := config.FindRostersDir("")
	if err != nil {
		return err
	}

	checks := performChecks(dir)

	if fixMode {
		hasFixable := false
		for _, c := range checks {
			if c.Status != "pass" && c.Fixable {
				hasFixable = true
				break
			}
		}

		if hasFixable {
			fixedItems := applyDoctorFixes(dir, checks)
			reChecks := performChecks(dir)
			reportDoctorResults(reChecks, isJSON, verbose, fixedItems)
			return nil
		}
	}

	reportDoctorResults(checks, isJSON, verbose, nil)
	return nil
}

func performChecks(dir string) []doctorCheck {
	var checks []doctorCheck

	cfg, cfgErr := config.ReadConfig(dir)
	checks = append(checks, checkConfig(dir, cfg, cfgErr))

	if checks[0].Status == "fail" {
		return checks
	}

	checks = append(checks, checkJsonlIntegrity(dir))
	checks = append(checks, checkSchemaValidation(dir))
	checks = append(checks, checkDuplicateIDs(dir))

	issues := readValidIssues(dir)
	checks = append(checks, checkReferentialIntegrity(issues))
	checks = append(checks, checkBidirectionalConsistency(issues))
	checks = append(checks, checkCircularDependencies(issues))
	checks = append(checks, checkLabelSchema(dir))
	checks = append(checks, checkExtensionsSchema(dir))
	checks = append(checks, checkClosedFieldsConsistency(dir))
	checks = append(checks, checkStaleLocks(dir))
	checks = append(checks, checkGitattributes(dir))

	return checks
}

func readRawLines(filePath string) []rawLine {
	file, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var lines []rawLine
	scanner := bufio.NewScanner(file)
	ln := 0
	for scanner.Scan() {
		ln++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var parsed any
		err := json.Unmarshal([]byte(text), &parsed)
		rl := rawLine{LineNumber: ln, Text: text}
		if err != nil {
			rl.Error = err.Error()
		} else {
			rl.Parsed = parsed
		}
		lines = append(lines, rl)
	}
	return lines
}

func readValidIssues(dir string) []models.Issue {
	lines := readRawLines(filepath.Join(dir, models.IssuesFile))
	idMap := make(map[string]models.Issue)
	for _, line := range lines {
		if line.Error != "" || line.Parsed == nil {
			continue
		}
		var iss models.Issue
		if err := json.Unmarshal([]byte(line.Text), &iss); err == nil && iss.ID != "" {
			idMap[iss.ID] = iss
		}
	}
	var issues []models.Issue
	for _, iss := range idMap {
		issues = append(issues, iss)
	}
	return issues
}

func checkConfig(dir string, cfg *models.Config, err error) doctorCheck {
	c := doctorCheck{Name: "config", Status: "pass", Message: "Config is valid"}
	if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
		c.Status, c.Message = "fail", ".rosters/ directory not found"
		return c
	}
	if err != nil || cfg == nil {
		c.Status, c.Message = "fail", "config.yaml is missing or unparseable"
		return c
	}
	if cfg.Project == "" {
		c.Status, c.Message = "fail", "config.yaml missing required 'project' field"
		return c
	}
	return c
}

func checkJsonlIntegrity(dir string) doctorCheck {
	var details []string
	for _, f := range []string{models.IssuesFile, models.TemplatesFile} {
		lines := readRawLines(filepath.Join(dir, f))
		for _, l := range lines {
			if l.Error != "" {
				details = append(details, fmt.Sprintf("%s line %d: %s", f, l.LineNumber, l.Error))
			}
		}
	}
	if len(details) > 0 {
		return doctorCheck{
			Name:    "jsonl-integrity",
			Status:  "fail",
			Message: fmt.Sprintf("%d malformed line(s) in JSONL files", len(details)),
			Details: details,
			Fixable: true,
		}
	}
	return doctorCheck{Name: "jsonl-integrity", Status: "pass", Message: "All JSONL lines parse correctly"}
}

func checkSchemaValidation(dir string) doctorCheck {
	var details []string
	lines := readRawLines(filepath.Join(dir, models.IssuesFile))
	for _, l := range lines {
		if l.Parsed == nil {
			continue
		}
		m, ok := l.Parsed.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		if id == "" {
			id = fmt.Sprintf("line %d", l.LineNumber)
			details = append(details, id+": missing or invalid 'id'")
		}
		if v, ok := m["title"].(string); !ok || v == "" {
			details = append(details, id+": missing or invalid 'title'")
		}
		if v, ok := m["createdAt"].(string); !ok || v == "" {
			details = append(details, id+": missing or invalid 'createdAt'")
		}
		if v, ok := m["status"].(string); ok {
			valid := false
			for _, s := range models.ValidStatuses {
				if s == v {
					valid = true
					break
				}
			}
			if !valid {
				details = append(details, id+": invalid status '"+v+"'")
			}
		}
	}
	if len(details) > 0 {
		return doctorCheck{Name: "schema-validation", Status: "fail", Message: fmt.Sprintf("%d schema violation(s)", len(details)), Details: details}
	}
	return doctorCheck{Name: "schema-validation", Status: "pass", Message: "All issues have valid schema"}
}

func checkDuplicateIDs(dir string) doctorCheck {
	var details []string
	for _, f := range []string{models.IssuesFile, models.TemplatesFile} {
		lines := readRawLines(filepath.Join(dir, f))
		counts := make(map[string]int)
		for _, l := range lines {
			if l.Parsed == nil {
				continue
			}
			m := l.Parsed.(map[string]any)
			if id, ok := m["id"].(string); ok && id != "" {
				counts[id]++
			}
		}
		for id, count := range counts {
			if count > 1 {
				details = append(details, fmt.Sprintf("%s appears %d times in %s", id, count, f))
			}
		}
	}
	if len(details) > 0 {
		return doctorCheck{Name: "duplicate-ids", Status: "warn", Message: fmt.Sprintf("%d duplicate ID(s) found", len(details)), Details: details, Fixable: true}
	}
	return doctorCheck{Name: "duplicate-ids", Status: "pass", Message: "No duplicate IDs"}
}

func checkReferentialIntegrity(issues []models.Issue) doctorCheck {
	ids := make(map[string]bool)
	for _, iss := range issues {
		ids[iss.ID] = true
	}
	var details []string
	for _, iss := range issues {
		for _, ref := range iss.BlockedBy {
			if !ids[ref] {
				details = append(details, fmt.Sprintf("%s.blockedBy → %s (not found)", iss.ID, ref))
			}
		}
		for _, ref := range iss.Blocks {
			if !ids[ref] {
				details = append(details, fmt.Sprintf("%s.blocks → %s (not found)", iss.ID, ref))
			}
		}
	}
	if len(details) > 0 {
		return doctorCheck{Name: "referential-integrity", Status: "warn", Message: fmt.Sprintf("%d dangling dependency reference(s)", len(details)), Details: details, Fixable: true}
	}
	return doctorCheck{Name: "referential-integrity", Status: "pass", Message: "All dependency references are valid"}
}

func checkBidirectionalConsistency(issues []models.Issue) doctorCheck {
	byId := make(map[string]models.Issue)
	for _, iss := range issues {
		byId[iss.ID] = iss
	}
	var details []string
	contains := func(list []string, s string) bool {
		for _, item := range list {
			if item == s {
				return true
			}
		}
		return false
	}
	for _, iss := range issues {
		for _, ref := range iss.BlockedBy {
			if target, ok := byId[ref]; ok && !contains(target.Blocks, iss.ID) {
				details = append(details, fmt.Sprintf("%s.blockedBy has %s, but %s.blocks missing %s", iss.ID, ref, ref, iss.ID))
			}
		}
		for _, ref := range iss.Blocks {
			if target, ok := byId[ref]; ok && !contains(target.BlockedBy, iss.ID) {
				details = append(details, fmt.Sprintf("%s.blocks has %s, but %s.blockedBy missing %s", iss.ID, ref, ref, iss.ID))
			}
		}
	}
	if len(details) > 0 {
		return doctorCheck{Name: "bidirectional-consistency", Status: "warn", Message: fmt.Sprintf("%d bidirectional mismatch(es)", len(details)), Details: details, Fixable: true}
	}
	return doctorCheck{Name: "bidirectional-consistency", Status: "pass", Message: "All dependency links are bidirectional"}
}

func checkCircularDependencies(issues []models.Issue) doctorCheck {
	graph := make(map[string][]string)
	for _, iss := range issues {
		graph[iss.ID] = iss.BlockedBy
	}
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var cycles [][]string

	var dfs func(node string, path []string)
	dfs = func(node string, path []string) {
		if inStack[node] {
			for i, p := range path {
				if p == node {
					cycle := append([]string{}, path[i:]...)
					cycles = append(cycles, append(cycle, node))
					break
				}
			}
			return
		}
		if visited[node] {
			return
		}
		visited[node] = true
		inStack[node] = true
		for _, dep := range graph[node] {
			dfs(dep, append(path, node))
		}
		inStack[node] = false
	}

	for id := range graph {
		dfs(id, nil)
	}

	if len(cycles) > 0 {
		var details []string
		for _, c := range cycles {
			details = append(details, strings.Join(c, " → "))
		}
		return doctorCheck{Name: "circular-dependencies", Status: "warn", Message: fmt.Sprintf("%d circular dependency chain(s) found", len(cycles)), Details: details}
	}
	return doctorCheck{Name: "circular-dependencies", Status: "pass", Message: "No circular dependencies"}
}

func checkLabelSchema(dir string) doctorCheck {
	var details []string
	lines := readRawLines(filepath.Join(dir, models.IssuesFile))
	for _, l := range lines {
		if l.Parsed == nil {
			continue
		}
		m := l.Parsed.(map[string]any)
		id, _ := m["id"].(string)
		if labels, ok := m["labels"]; ok && labels != nil {
			if slice, ok := labels.([]any); ok {
				for _, lbl := range slice {
					if s, ok := lbl.(string); !ok || strings.TrimSpace(s) == "" {
						details = append(details, fmt.Sprintf("%s: invalid label entry", id))
					}
				}
			} else {
				details = append(details, fmt.Sprintf("%s: labels is not an array", id))
			}
		}
	}
	if len(details) > 0 {
		return doctorCheck{Name: "label-schema", Status: "warn", Message: fmt.Sprintf("%d label schema issue(s)", len(details)), Details: details, Fixable: true}
	}
	return doctorCheck{Name: "label-schema", Status: "pass", Message: "All label arrays are valid"}
}

func checkExtensionsSchema(dir string) doctorCheck {
	var details []string
	lines := readRawLines(filepath.Join(dir, models.IssuesFile))
	for _, l := range lines {
		if l.Parsed == nil {
			continue
		}
		m := l.Parsed.(map[string]any)
		id, _ := m["id"].(string)
		if ext, ok := m["extensions"]; ok && ext != nil {
			if _, ok := ext.(map[string]any); !ok {
				details = append(details, fmt.Sprintf("%s: extensions must be a plain object", id))
			}
		}
	}
	if len(details) > 0 {
		return doctorCheck{Name: "extensions-schema", Status: "warn", Message: fmt.Sprintf("%d malformed extensions field(s)", len(details)), Details: details, Fixable: true}
	}
	return doctorCheck{Name: "extensions-schema", Status: "pass", Message: "All extensions fields are valid"}
}

func checkClosedFieldsConsistency(dir string) doctorCheck {
	var details []string
	lines := readRawLines(filepath.Join(dir, models.IssuesFile))
	for _, l := range lines {
		if l.Parsed == nil {
			continue
		}
		m := l.Parsed.(map[string]any)
		id, _ := m["id"].(string)
		status, _ := m["status"].(string)
		closedAt, _ := m["closedAt"].(string)
		reason, _ := m["closeReason"].(string)

		hasClosedMeta := closedAt != "" || reason != ""
		if status != "closed" && hasClosedMeta {
			details = append(details, fmt.Sprintf("%s: status=%s but closed metadata set", id, status))
		} else if status == "closed" && closedAt == "" {
			details = append(details, fmt.Sprintf("%s: status=closed but closedAt missing", id))
		}
	}
	if len(details) > 0 {
		return doctorCheck{Name: "closed-fields-consistency", Status: "warn", Message: fmt.Sprintf("%d status/close-metadata mismatch(es)", len(details)), Details: details, Fixable: true}
	}
	return doctorCheck{Name: "closed-fields-consistency", Status: "pass", Message: "All status/close-metadata pairs are consistent"}
}

func checkStaleLocks(dir string) doctorCheck {
	var details []string
	for _, f := range []string{models.IssuesFile, models.TemplatesFile, models.PlansFile} {
		lockPath := filepath.Join(dir, f+".lock")
		if st, err := os.Stat(lockPath); err == nil {
			age := time.Since(st.ModTime())
			if age.Milliseconds() > int64(models.LockStaleMS) {
				details = append(details, fmt.Sprintf("%s.lock is stale (%ds old)", f, int(age.Seconds())))
			}
		}
	}
	if len(details) > 0 {
		return doctorCheck{Name: "stale-locks", Status: "warn", Message: fmt.Sprintf("%d stale lock file(s) found", len(details)), Details: details, Fixable: true}
	}
	return doctorCheck{Name: "stale-locks", Status: "pass", Message: "No stale lock files"}
}

func checkGitattributes(dir string) doctorCheck {
	root := config.ProjectRootFromRostersDir(dir)
	path := filepath.Join(root, ".gitattributes")
	var details []string
	content, err := os.ReadFile(path)
	entries := []string{
		models.SeedsDirName + "/" + models.IssuesFile + " merge=union",
		models.SeedsDirName + "/" + models.TemplatesFile + " merge=union",
	}
	if err != nil {
		details = append(details, ".gitattributes file not found")
	} else {
		text := string(content)
		for _, e := range entries {
			if !strings.Contains(text, e) {
				details = append(details, "Missing: "+e)
			}
		}
	}
	if len(details) > 0 {
		return doctorCheck{Name: "gitattributes", Status: "warn", Message: "Missing merge=union gitattributes entries", Details: details, Fixable: true}
	}
	return doctorCheck{Name: "gitattributes", Status: "pass", Message: "Gitattributes configured correctly"}
}

func applyDoctorFixes(dir string, checks []doctorCheck) []string {
	var fixed []string
	for _, c := range checks {
		if c.Status == "pass" || !c.Fixable {
			continue
		}
		switch c.Name {
		case "jsonl-integrity":
			for _, f := range []string{models.IssuesFile, models.TemplatesFile} {
				path := filepath.Join(dir, f)
				lines := readRawLines(path)
				var valid []string
				for _, l := range lines {
					if l.Error == "" {
						valid = append(valid, l.Text)
					}
				}
				if len(valid) < len(lines) {
					os.WriteFile(path, []byte(strings.Join(valid, "\n")+"\n"), 0644)
					fixed = append(fixed, fmt.Sprintf("Cleaned malformed lines from %s", f))
				}
			}
		case "duplicate-ids":
			for _, f := range []string{models.IssuesFile, models.TemplatesFile} {
				path := filepath.Join(dir, f)
				lines := readRawLines(path)
				seen := make(map[string]string)
				for _, l := range lines {
					if l.Parsed == nil {
						continue
					}
					m := l.Parsed.(map[string]any)
					if id, ok := m["id"].(string); ok {
						seen[id] = l.Text
					}
				}
				if len(seen) < len(lines) {
					var output []string
					for _, txt := range seen {
						output = append(output, txt)
					}
					os.WriteFile(path, []byte(strings.Join(output, "\n")+"\n"), 0644)
					fixed = append(fixed, "Deduplicated "+f)
				}
			}
		case "stale-locks":
			for _, f := range []string{models.IssuesFile, models.TemplatesFile, models.PlansFile} {
				lockPath := filepath.Join(dir, f+".lock")
				if st, err := os.Stat(lockPath); err == nil {
					if time.Since(st.ModTime()).Milliseconds() > int64(models.LockStaleMS) {
						os.Remove(lockPath)
						fixed = append(fixed, "Removed stale "+f+".lock")
					}
				}
			}
		case "gitattributes":
			root := config.ProjectRootFromRostersDir(dir)
			path := filepath.Join(root, ".gitattributes")
			entry := fmt.Sprintf("%s/%s merge=union\n%s/%s merge=union\n", models.SeedsDirName, models.IssuesFile, models.SeedsDirName, models.TemplatesFile)
			if existing, err := os.ReadFile(path); err == nil {
				os.WriteFile(path, append(existing, []byte("\n"+entry)...), 0644)
			} else {
				os.WriteFile(path, []byte(entry), 0644)
			}
			fixed = append(fixed, "Updated .gitattributes")
		}
	}
	return fixed
}

func reportDoctorResults(checks []doctorCheck, isJSON bool, verbose bool, fixed []string) {
	summary := map[string]int{"pass": 0, "warn": 0, "fail": 0}
	for _, c := range checks {
		summary[c.Status]++
	}

	if isJSON {
		format.OutputJSON(map[string]any{
			"success": summary["fail"] == 0,
			"checks":  checks,
			"summary": summary,
			"fixed":   fixed,
		})
	} else {
		fmt.Printf("\n%s\n\n", color.New(color.Bold).Sprint("Rosters Doctor"))
		for _, c := range checks {
			if c.Status == "pass" && !verbose {
				continue
			}
			icon := color.HiGreenString("✓")
			if c.Status == "warn" {
				icon = color.YellowString("!")
			} else if c.Status == "fail" {
				icon = color.RedString("✗")
			}
			fmt.Printf("  %s %s\n", icon, c.Message)
			for _, d := range c.Details {
				format.Muted.Printf("      %s\n", d)
			}
		}
		fmt.Printf("\n")
		format.Muted.Printf("%d passed, %d warning(s), %d failure(s)\n", summary["pass"], summary["warn"], summary["fail"])
		if len(fixed) > 0 {
			fmt.Printf("\n%s\n", color.New(color.Bold).Sprint("Fixed:"))
			for _, f := range fixed {
				format.Brand.Printf("  ✓ %s\n", f)
			}
		}
	}

	if summary["fail"] > 0 {
		os.Exit(1)
	}
}
