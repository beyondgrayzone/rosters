package plan

import (
	"testing"

	"rosters/pkg/models"
)

func TestBuiltinTemplatesExist(t *testing.T) {
	expected := []string{"feature", "bug", "refactor"}
	for _, name := range expected {
		if _, ok := BuiltinPlanTemplates[name]; !ok {
			t.Errorf("BuiltinPlanTemplates missing %q", name)
		}
	}
}

func TestBuiltinFeatureTemplate(t *testing.T) {
	tpl := BuiltinFeatureTemplate
	checkRequiredSections(t, tpl, []string{"context", "approach", "steps", "acceptance"})
	optionalSections := []string{"alternatives", "risks"}
	for _, sec := range optionalSections {
		if spec, ok := tpl.Sections[sec]; ok {
			if spec.Required {
				t.Errorf("section %q should be optional", sec)
			}
		} else {
			t.Errorf("optional section %q missing", sec)
		}
	}
}

func TestBuiltinBugTemplate(t *testing.T) {
	tpl := BuiltinBugTemplate
	checkRequiredSections(t, tpl, []string{"context", "reproduction", "root_cause", "approach", "steps", "acceptance"})
}

func TestBuiltinRefactorTemplate(t *testing.T) {
	tpl := BuiltinRefactorTemplate
	checkRequiredSections(t, tpl, []string{"context", "behavior_invariant", "approach", "steps", "acceptance"})
}

func checkRequiredSections(t *testing.T, tpl *models.PlanTemplate, required []string) {
	for _, name := range required {
		spec, ok := tpl.Sections[name]
		if !ok {
			t.Errorf("required section %q missing from template %q", name, tpl.Name)
			continue
		}
		if !spec.Required {
			t.Errorf("section %q should be required", name)
		}
	}
}

func TestTemplateDescriptionNonEmpty(t *testing.T) {
	for name, tpl := range BuiltinPlanTemplates {
		if tpl.Description == nil || *tpl.Description == "" {
			t.Errorf("template %q has empty description", name)
		}
	}
}
