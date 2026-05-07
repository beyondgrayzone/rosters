package models

type Issue struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	Status        string         `json:"status"`
	Type          string         `json:"type"`
	Priority      int            `json:"priority"`
	Assignee      *string        `json:"assignee,omitempty"`
	Description   *string        `json:"description,omitempty"`
	CloseReason   *string        `json:"closeReason,omitempty"`
	Blocks        []string       `json:"blocks,omitempty"`
	BlockedBy     []string       `json:"blockedBy,omitempty"`
	Labels        []string       `json:"labels,omitempty"`
	Convoy        *string        `json:"convoy,omitempty"`
	PlanID        *string        `json:"plan_id,omitempty"`
	PlanStepIndex *int           `json:"plan_step_index,omitempty"`
	RequiresPlan  *bool          `json:"requires_plan,omitempty"`
	Extensions    map[string]any `json:"extensions,omitempty"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
	ClosedAt      *string        `json:"closedAt,omitempty"`
}

type PlanStatus string

const (
	PlanStatusDraft    PlanStatus = "draft"
	PlanStatusApproved PlanStatus = "approved"
	PlanStatusActive   PlanStatus = "active"
	PlanStatusDone     PlanStatus = "done"
)

type PlanOutcome string

const (
	PlanOutcomeSuccess PlanOutcome = "success"
	PlanOutcomePartial PlanOutcome = "partial"
	PlanOutcomeFailure PlanOutcome = "failure"
)

type Plan struct {
	ID              string         `json:"id"`
	Roster          string         `json:"roster"`
	Template        string         `json:"template"`
	Name            *string        `json:"name,omitempty"`
	Status          PlanStatus     `json:"status"`
	Revision        int            `json:"revision"`
	Sections        map[string]any `json:"sections"`
	Children        []string       `json:"children"`
	AdoptedChildren []string       `json:"adoptedChildren,omitempty"`
	Outcome         *PlanOutcome   `json:"outcome,omitempty"`
	OutcomeNote     *string        `json:"outcomeNote,omitempty"`
	ReviewedBy      *string        `json:"reviewedBy,omitempty"`
	CreatedAt       string         `json:"createdAt"`
	UpdatedAt       string         `json:"updatedAt"`
}

type TemplateStep struct {
	Title        string  `json:"title"`
	Type         *string `json:"type,omitempty"`
	Priority     *int    `json:"priority,omitempty"`
	PlanTemplate *string `json:"plan_template,omitempty"`
}

type Template struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Steps []TemplateStep `json:"steps"`
}

type Config struct {
	Project      string `yaml:"project"`
	Version      string `yaml:"version"`
	MaxPlanDepth *int   `yaml:"max_plan_depth,omitempty"`
}

type SectionKind string

const (
	SectionKindText  SectionKind = "text"
	SectionKindList  SectionKind = "list"
	SectionKindSteps SectionKind = "steps"
)

type SectionSpec struct {
	Required    bool    `json:"required"`
	Kind        any     `json:"kind"`
	Prompt      string  `json:"prompt"`
	MinLength   *int    `json:"min_length,omitempty"`
	Min         *int    `json:"min,omitempty"`
	Item        any     `json:"item,omitempty"`
	MulchSource *string `json:"mulch_source,omitempty"`
}

type PlanTemplate struct {
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Sections    map[string]SectionSpec `json:"sections"`
}

type SubmittedStep struct {
	Title        string  `json:"title"`
	Type         *string `json:"type,omitempty"`
	Priority     *int    `json:"priority,omitempty"`
	Blocks       []int   `json:"blocks,omitempty"`
	PlanTemplate *string `json:"plan_template,omitempty"`
}

const (
	SeedsDirName        = ".rosters"
	IssuesFile          = "issues.jsonl"
	TemplatesFile       = "templates.jsonl"
	PlansFile           = "plans.jsonl"
	ConfigFile          = "config.yaml"
	LockStaleMS         = 30000
	LockRetryMS         = 100
	LockTimeoutMS       = 30000
	DefaultMaxPlanDepth = 3
)

var (
	ValidTypes     = []string{"task", "bug", "feature", "epic"}
	ValidStatuses  = []string{"open", "in_progress", "closed"}
	PriorityLabels = map[int]string{
		0: "Critical",
		1: "High",
		2: "Medium",
		3: "Low",
		4: "Backlog",
	}
)
