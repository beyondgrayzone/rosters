package plan

import (
	"fmt"
	"rosters/pkg/models"
)

type ErrorEntry struct {
	Path string `json:"path"`
	Code string `json:"code"`
	Fix  string `json:"fix"`
}

type ValidationResult struct {
	Valid  bool         `json:"valid"`
	Errors []ErrorEntry `json:"errors"`
}

func ValidatePlan(plan models.SubmittedPlan, template models.PlanTemplate) ValidationResult {
	res := ValidationResult{Valid: true}

	for name, spec := range template.Sections {
		val, exists := plan.Sections[name]
		if !exists || val == nil {
			if spec.Required {
				res.Valid = false
				res.Errors = append(res.Errors, ErrorEntry{
					Path: fmt.Sprintf("sections.%s", name),
					Code: "required",
					Fix:  fmt.Sprintf("add a '%s' field", name),
				})
			}
			continue
		}

		validateSection(fmt.Sprintf("sections.%s", name), val, spec, &res)
	}

	if steps, ok := plan.Sections["steps"].([]any); ok {
		for i, s := range steps {
			if step, ok := s.(map[string]any); ok {
				if blocks, ok := step["blocks"].([]any); ok {
					for _, b := range blocks {
						if bf, ok := b.(float64); ok {
							idx := int(bf)
							if idx == i+1 {
								res.Valid = false
								res.Errors = append(res.Errors, ErrorEntry{
									Path: fmt.Sprintf("sections.steps.%d.blocks", i),
									Code: "self-reference",
									Fix:  fmt.Sprintf("step %d cannot block itself", idx),
								})
							} else if idx < 1 || idx > len(steps) {
								res.Valid = false
								res.Errors = append(res.Errors, ErrorEntry{
									Path: fmt.Sprintf("sections.steps.%d.blocks", i),
									Code: "out-of-range",
									Fix:  fmt.Sprintf("step index %d is out of range (1..%d)", idx, len(steps)),
								})
							}
						}
					}
				}
			}
		}
	}

	return res
}

func validateSection(path string, val any, spec models.SectionSpec, res *ValidationResult) {
	kind, ok := spec.Kind.(string)
	if !ok {
		return
	}

	switch kind {
	case "text":
		s, ok := val.(string)
		if !ok {
			res.Valid = false
			res.Errors = append(res.Errors, ErrorEntry{Path: path, Code: "type", Fix: "must be a string"})
			return
		}
		if spec.MinLength != nil && len(s) < *spec.MinLength {
			res.Valid = false
			res.Errors = append(res.Errors, ErrorEntry{Path: path, Code: "minLength", Fix: fmt.Sprintf("expand to at least %d characters", *spec.MinLength)})
		}
	case "list", "steps":
		l, ok := val.([]any)
		if !ok {
			res.Valid = false
			res.Errors = append(res.Errors, ErrorEntry{Path: path, Code: "type", Fix: "must be an array"})
			return
		}
		if spec.Min != nil && len(l) < *spec.Min {
			res.Valid = false
			res.Errors = append(res.Errors, ErrorEntry{Path: path, Code: "min", Fix: fmt.Sprintf("add at least %d more entries", *spec.Min-len(l))})
		}
	}
}
