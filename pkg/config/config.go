package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"rosters/pkg/models"
	"rosters/pkg/plan"

	"gopkg.in/yaml.v3"
)

func FindRostersDir(startDir string) (string, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	dir := startDir
	for {
		configPath := filepath.Join(dir, models.SeedsDirName, models.ConfigFile)
		if _, err := os.Stat(configPath); err == nil {
			return resolveWorktreeRoot(filepath.Join(dir, models.SeedsDirName)), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func resolveWorktreeRoot(candidateRostersDir string) string {
	candidateRoot := filepath.Dir(candidateRostersDir)
	common := gitCommonDir(candidateRoot)
	if common == "" {
		return candidateRostersDir
	}
	var mainRoot string
	if strings.HasSuffix(common, ".git") {
		mainRoot = filepath.Dir(common)
	} else {
		mainRoot = filepath.Dir(filepath.Dir(common))
	}
	mainResolved, err := filepath.Abs(mainRoot)
	if err != nil {
		return candidateRostersDir
	}
	candidateResolved, err := filepath.Abs(candidateRoot)
	if err != nil {
		return candidateRostersDir
	}
	if mainResolved == candidateResolved {
		return candidateRostersDir
	}
	mainRostersDir := filepath.Join(mainResolved, models.SeedsDirName)
	if _, err := os.Stat(filepath.Join(mainRostersDir, models.ConfigFile)); err == nil {
		return mainRostersDir
	}
	return candidateRostersDir
}

func gitCommonDir(cwd string) string {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return ""
	}
	resolved, err := filepath.Abs(filepath.Join(cwd, raw))
	if err != nil {
		return ""
	}
	return resolved
}

func IsInsideWorktree(dir string) bool {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return false
		}
	}
	gd := gitDir(dir)
	common := gitCommonDir(dir)
	if gd == "" || common == "" {
		return false
	}
	return gd != common
}

func gitDir(cwd string) string {
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--git-dir")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	raw := strings.TrimSpace(string(output))
	if raw == "" {
		return ""
	}
	resolved, err := filepath.Abs(filepath.Join(cwd, raw))
	if err != nil {
		return ""
	}
	return resolved
}

func ProjectRootFromRostersDir(rostersDir string) string {
	return filepath.Dir(rostersDir)
}

func ReadConfig(rostersDir string) (*models.Config, error) {
	content, err := os.ReadFile(filepath.Join(rostersDir, models.ConfigFile))
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	config := &models.Config{}
	if project, ok := data["project"].(string); ok && project != "" {
		config.Project = project
	} else {
		config.Project = "rosters"
	}
	if version, ok := data["version"].(string); ok {
		config.Version = version
	} else {
		config.Version = "1"
	}
	if maxDepth, ok := data["max_plan_depth"].(int); ok {
		config.MaxPlanDepth = &maxDepth
	}
	return config, nil
}

func MaxPlanDepth(config *models.Config) int {
	if config.MaxPlanDepth != nil {
		return *config.MaxPlanDepth
	}
	return models.DefaultMaxPlanDepth
}

func LoadPlanTemplates(rostersDir string) (map[string]*models.PlanTemplate, error) {
	configPath := filepath.Join(rostersDir, models.ConfigFile)
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cloneBuiltinTemplates(), nil
		}
		return nil, err
	}
	var data map[string]any
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, err
	}
	builtins := cloneBuiltinTemplates()
	userBlockRaw, ok := data["plan_templates"]
	if !ok {
		return builtins, nil
	}
	userBlock, ok := userBlockRaw.(map[string]any)
	if !ok {
		return builtins, nil
	}
	result := cloneBuiltinTemplates()
	for name, raw := range userBlock {
		rawMap, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("plan_templates.%s must be a mapping", name)
		}
		sectionsRaw, ok := rawMap["sections"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("plan_templates.%s.sections must be a mapping", name)
		}
		sections := make(map[string]models.SectionSpec)
		for secName, secRaw := range sectionsRaw {
			spec, err := parseSectionSpec(secRaw, fmt.Sprintf("plan_templates.%s.sections.%s", name, secName))
			if err != nil {
				return nil, err
			}
			sections[secName] = spec
		}
		tpl := &models.PlanTemplate{Name: name, Sections: sections}
		if desc, ok := rawMap["description"].(string); ok {
			tpl.Description = &desc
		}
		result[name] = tpl
	}
	return result, nil
}

func cloneBuiltinTemplates() map[string]*models.PlanTemplate {
	result := make(map[string]*models.PlanTemplate)
	for name, tpl := range plan.BuiltinPlanTemplates {
		cloned := *tpl
		cloned.Sections = make(map[string]models.SectionSpec)
		for k, v := range tpl.Sections {
			cloned.Sections[k] = v
		}
		result[name] = &cloned
	}
	return result
}

func parseSectionSpec(raw any, path string) (models.SectionSpec, error) {
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return models.SectionSpec{}, fmt.Errorf("%s: must be a mapping", path)
	}
	required, ok := rawMap["required"].(bool)
	if !ok {
		return models.SectionSpec{}, fmt.Errorf("%s.required: must be a boolean", path)
	}
	prompt, ok := rawMap["prompt"].(string)
	if !ok {
		return models.SectionSpec{}, fmt.Errorf("%s.prompt: must be a string", path)
	}
	kind := rawMap["kind"]
	var spec models.SectionSpec
	spec.Required = required
	spec.Kind = kind
	spec.Prompt = prompt
	if minLen, ok := rawMap["min_length"].(int); ok {
		spec.MinLength = &minLen
	}
	if min, ok := rawMap["min"].(int); ok {
		spec.Min = &min
	}
	if item, ok := rawMap["item"]; ok {
		spec.Item = item
	}
	if mulchSrc, ok := rawMap["mulch_source"].(string); ok {
		spec.MulchSource = &mulchSrc
	}
	return spec, nil
}
