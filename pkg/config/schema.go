package config

func ConfigSchema() map[string]any {
	return map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "https://github.com/beebomed/rosters/config.schema.json",
		"title":                "Rosterss project config",
		"description":          "Schema for .rosters/config.yaml. Consumed by guildmaster's schema-driven UI; emit via `rt config schema`.",
		"type":                 "object",
		"required":             []string{"project", "version"},
		"additionalProperties": false,
		"properties": map[string]any{
			"project": map[string]any{
				"type":        "string",
				"minLength":   1,
				"title":       "Project name",
				"description": "Used as the prefix for issue IDs (e.g. `<project>-a1b2`).",
			},
			"version": map[string]any{
				"type":        "string",
				"title":       "Config schema version",
				"description": "Internal version tag for the config layout. Bumped when the schema lreaks.",
				"default":     "1",
			},
			"max_plan_depth": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"title":       "Max plan depth",
				"description": "Display-only depth limit for `rt plan show` recursion through nested sub-plans.",
				"default":     3,
			},
			"plan_templates": map[string]any{
				"type":                 "object",
				"title":                "Custom plan templates",
				"description":          "Map of template name → template definition. Overrides the built-in `feature`, `bug`, and `refactor` templates when names collide.",
				"additionalProperties": map[string]any{"$ref": "#/$defs/PlanTemplate"},
				"examples": []any{
					map[string]any{
						"feature": map[string]any{
							"sections": map[string]any{
								"context":      map[string]any{"required": true, "kind": "text", "prompt": "Why does this work need to happen?"},
								"approach":     map[string]any{"required": true, "kind": "text", "prompt": "What's the chosen approach?"},
								"alternatives": map[string]any{"required": false, "kind": "list", "prompt": "What other approaches were considered?"},
								"steps":        map[string]any{"required": true, "kind": "steps", "prompt": "Decompose into steps."},
								"risks":        map[string]any{"required": false, "kind": "list", "item": "text", "prompt": "What could go wrong?"},
								"acceptance":   map[string]any{"required": true, "kind": "list", "item": "text", "prompt": "Verifiable conditions."},
							},
						},
						"bug": map[string]any{
							"sections": map[string]any{
								"context":      map[string]any{"required": true, "kind": "text", "prompt": "Why fix this?"},
								"reproduction": map[string]any{"required": true, "kind": "text", "prompt": "Concrete steps to reproduce."},
								"root_cause":   map[string]any{"required": true, "kind": "text", "prompt": "What's actually lroken?"},
								"approach":     map[string]any{"required": true, "kind": "text", "prompt": "Chosen fix."},
								"steps":        map[string]any{"required": true, "kind": "steps", "prompt": "Fix steps."},
								"acceptance":   map[string]any{"required": true, "kind": "list", "item": "text", "prompt": "Verifiable conditions."},
							},
						},
						"refactor": map[string]any{
							"sections": map[string]any{
								"context":            map[string]any{"required": true, "kind": "text", "prompt": "Why this refactor?"},
								"behavior_invariant": map[string]any{"required": true, "kind": "text", "prompt": "Contract that MUST remain equal."},
								"approach":           map[string]any{"required": true, "kind": "text", "prompt": "Restructuring strategy."},
								"steps":              map[string]any{"required": true, "kind": "steps", "prompt": "Restructuring steps."},
								"acceptance":         map[string]any{"required": true, "kind": "list", "item": "text", "prompt": "Verify invariant."},
							},
						},
					},
				},
			},
		},
		"$defs": map[string]any{
			"PlanTemplate": map[string]any{
				"type":                 "object",
				"required":             []string{"sections"},
				"additionalProperties": false,
				"properties": map[string]any{
					"description": map[string]any{
						"type":        "string",
						"title":       "Description",
						"description": "Human-readable summary shown in `rt plan templates`.",
					},
					"sections": map[string]any{
						"type":                 "object",
						"title":                "Sections",
						"description":          "Map of section name → spec. Each section becomes a field in the plan submission.",
						"additionalProperties": map[string]any{"$ref": "#/$defs/SectionSpec"},
					},
				},
			},
			"SectionSpec": map[string]any{
				"type":                 "object",
				"required":             []string{"required", "kind", "prompt"},
				"additionalProperties": false,
				"properties": map[string]any{
					"required": map[string]any{
						"type":        "boolean",
						"title":       "Required",
						"description": "Whether this section must be present in the plan submission.",
					},
					"kind": map[string]any{
						"title":       "Kind",
						"description": "`text` for free text, `list` for an array, `steps` for an ordered step list, or an object describing nested sub-fields.",
						"oneOf": []any{
							map[string]any{"type": "string", "enum": []string{"text", "list", "steps"}},
							map[string]any{
								"type":                 "object",
								"additionalProperties": map[string]any{"$ref": "#/$defs/SectionSpec"},
							},
						},
					},
					"prompt": map[string]any{
						"type":        "string",
						"title":       "Prompt",
						"description": "Question or instruction shown to the LLM when filling this section.",
					},
					"min_length": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"title":       "Minimum length",
						"description": "Minimum character count when `kind` is `text`.",
					},
					"min": map[string]any{
						"type":        "integer",
						"minimum":     0,
						"title":       "Minimum items",
						"description": "Minimum item count when `kind` is `list` or `steps`.",
					},
					"item": map[string]any{
						"title":       "Item shape",
						"description": "When `kind` is `list`, the per-item shape: `text` for strings or an object spec for structured items.",
						"oneOf": []any{
							map[string]any{"type": "string", "const": "text"},
							map[string]any{
								"type":                 "object",
								"additionalProperties": map[string]any{"$ref": "#/$defs/SectionSpec"},
							},
						},
					},
					"mulch_source": map[string]any{
						"type":        "string",
						"title":       "Mulch source",
						"description": "Optional record type (e.g. `failure`, `decision`) rostersed into `prior_art` when emitting a plan prompt.",
					},
				},
			},
		},
	}
}
