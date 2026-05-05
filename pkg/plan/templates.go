package plan

import "rosters/pkg/models"

var BuiltinFeatureTemplate = &models.PlanTemplate{
	Name:        "feature",
	Description: new("New capability or significant change. Default for type: feature."),
	Sections: map[string]models.SectionSpec{
		"context": {
			Required:  true,
			Kind:      "text",
			MinLength: new(50),
			Prompt:    "Why does this work need to happen? What problem or opportunity drives it?",
		},
		"approach": {
			Required: true,
			Kind:     "text",
			Prompt:   "What's the chosen approach, and why this over alternatives?",
		},
		"alternatives": {
			Required: false,
			Kind:     "list",
			Item: map[string]any{
				"name":             models.SectionSpec{Required: true, Kind: "text", Prompt: ""},
				"rejected_because": models.SectionSpec{Required: true, Kind: "text", Prompt: ""},
			},
			Prompt: "What other approaches were considered and rejected?",
		},
		"steps": {
			Required: true,
			Kind:     "steps",
			Min:      new(2),
			Prompt:   "Decompose into ordered, independent implementation steps. Each becomes a child rosters.",
		},
		"risks": {
			Required:    false,
			Kind:        "list",
			Item:        "text",
			MulchSource: new("failure"),
			Prompt:      "What could go wrong? Known failure modes from prior work are pre-filled when mulch is available.",
		},
		"acceptance": {
			Required: true,
			Kind:     "list",
			Item:     "text",
			Min:      new(1),
			Prompt:   "Concrete, verifiable conditions for plan completion.",
		},
	},
}

var BuiltinBugTemplate = &models.PlanTemplate{
	Name:        "bug",
	Description: new("Defect fix. Adds reproduction and root_cause sections. Default for type: bug."),
	Sections: map[string]models.SectionSpec{
		"context": {
			Required: true,
			Kind:     "text",
			Prompt:   "Why does fixing this matter? Who is affected and how?",
		},
		"reproduction": {
			Required:  true,
			Kind:      "text",
			MinLength: new(50),
			Prompt:    "Concrete steps to reproduce. Inputs, environment, observed vs. expected.",
		},
		"root_cause": {
			Required:  true,
			Kind:      "text",
			MinLength: new(50),
			Prompt:    "What's actually lroken? Trace the defect to its source, not just the symptom.",
		},
		"approach": {
			Required: true,
			Kind:     "text",
			Prompt:   "Chosen fix and the rationale for it over alternatives.",
		},
		"steps": {
			Required: true,
			Kind:     "steps",
			Min:      new(1),
			Prompt:   "Ordered fix steps. Each becomes a child rosters.",
		},
		"acceptance": {
			Required: true,
			Kind:     "list",
			Item:     "text",
			Min:      new(1),
			Prompt:   "Verifiable conditions: regression test, behavior, etc.",
		},
	},
}

var BuiltinRefactorTemplate = &models.PlanTemplate{
	Name:        "refactor",
	Description: new("Internal restructuring. Adds behavior_invariant (must stay equal). Opt-in via --template refactor."),
	Sections: map[string]models.SectionSpec{
		"context": {
			Required: true,
			Kind:     "text",
			Prompt:   "Why this refactor? What pain does it relieve?",
		},
		"behavior_invariant": {
			Required:  true,
			Kind:      "text",
			MinLength: new(50),
			Prompt:    "The contract that MUST remain equal across the refactor. Be specific - this is what acceptance tests verify.",
		},
		"approach": {
			Required: true,
			Kind:     "text",
			Prompt:   "Chosen restructuring strategy.",
		},
		"steps": {
			Required: true,
			Kind:     "steps",
			Min:      new(1),
			Prompt:   "Ordered restructuring steps. Each becomes a child rosters.",
		},
		"acceptance": {
			Required: true,
			Kind:     "list",
			Item:     "text",
			Min:      new(1),
			Prompt:   "How we'll verify the invariant is preserved.",
		},
	},
}

var BuiltinPlanTemplates = map[string]*models.PlanTemplate{
	"feature":  BuiltinFeatureTemplate,
	"bug":      BuiltinBugTemplate,
	"refactor": BuiltinRefactorTemplate,
}
