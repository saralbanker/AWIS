// Package dsl implements the AWIS YAML workflow DSL parser (M10; IMP §27.M10;
// TDS-02). ParseFile and Parse unmarshal a YAML file into the frozen
// core.WorkflowDefinition type. Graph-semantic validation is the responsibility
// of internal/validate; expression grammars are TDS-03 / internal/expr.
package dsl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/awis/awis/internal/core"
)

// semverRE mirrors sdk/builder.go:15 exactly (coordinate: sdk/builder.go semverRE).
var semverRE = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)`)

// validateSemver mirrors sdk/builder.go:validateSemver (coordinate: sdk/builder.go validateSemver).
func validateSemver(version string) error {
	if version == "" {
		return fmt.Errorf("version must not be empty")
	}
	if !semverRE.MatchString(version) {
		return fmt.Errorf("version %q does not match semver format (e.g. \"1.0.0\")", version)
	}
	return nil
}

// ── intermediate YAML structs (TDS-02 field names verbatim as yaml tags) ─────

type ymlWorkflowDef struct {
	SchemaVersion int             `yaml:"schema_version"`
	ID            string          `yaml:"id"`
	Version       string          `yaml:"version"`
	Namespace     string          `yaml:"namespace"`
	Name          string          `yaml:"name"`
	Description   string          `yaml:"description"`
	Triggers      []ymlTrigger    `yaml:"triggers"`
	Steps         []ymlStep       `yaml:"steps"`
	Transitions   []ymlTransition `yaml:"transitions"`
	InitialStep   string          `yaml:"initial_step"`
	FinalSteps    []string        `yaml:"final_steps"`
	Compensation  *ymlCompPlan    `yaml:"compensation"`
	Timeout       string          `yaml:"timeout"`
	Metadata      map[string]any  `yaml:"metadata"`
}

type ymlTrigger struct {
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:"config"`
}

type ymlStep struct {
	ID           string          `yaml:"id"`
	Name         string          `yaml:"name"`
	Type         string          `yaml:"type"`
	Handler      string          `yaml:"handler"`
	Inputs       map[string]any  `yaml:"inputs"`
	Outputs      map[string]any  `yaml:"outputs"`
	Retry        *ymlRetryPolicy `yaml:"retry"`
	Timeout      string          `yaml:"timeout"`
	Fallback     string          `yaml:"fallback"`
	Compensation *ymlCompRef     `yaml:"compensation"`
	WaitSignal   *ymlWaitConfig  `yaml:"wait_signal"`
	Intelligence *ymlIntelReq    `yaml:"intelligence"`
}

type ymlRetryPolicy struct {
	Attempts        int      `yaml:"attempts"`
	Backoff         string   `yaml:"backoff"`
	InitialDelay    string   `yaml:"initial_delay"`
	MaxDelay        string   `yaml:"max_delay"`
	RetryableErrors []string `yaml:"retryable_errors"`
}

type ymlWaitConfig struct {
	SignalName    string `yaml:"signal_name"`
	Timeout       string `yaml:"timeout"`
	TimeoutAction string `yaml:"timeout_action"`
}

type ymlIntelReq struct {
	Capability    string `yaml:"capability"`
	ModelHint     string `yaml:"model_hint"`
	ContextBudget int    `yaml:"context_budget"`
	Required      bool   `yaml:"required"`
}

type ymlTransition struct {
	From      string `yaml:"from"`
	To        string `yaml:"to"`
	Condition string `yaml:"condition"`
	OnError   bool   `yaml:"on_error"`
}

type ymlCompPlan struct {
	Steps []ymlCompStep `yaml:"steps"`
}

type ymlCompStep struct {
	StepID      string          `yaml:"step_id"`
	UndoHandler string          `yaml:"undo_handler"`
	Retry       *ymlRetryPolicy `yaml:"retry"`
}

type ymlCompRef struct {
	Handler string `yaml:"handler"`
}

// ── line map ──────────────────────────────────────────────────────────────────

// lineMap holds YAML source-line numbers for workflow definition elements,
// captured at parse time. Used by the C2 renderer to attach file/line context
// to validate.Issues (PRD §18; FR-WD-04; T3).
type lineMap struct {
	TopLevel    map[string]int // top-level field name → line
	Steps       map[string]int // step id → line of the step mapping node
	Transitions []int          // transition[i] → line of the mapping node
	Triggers    []int          // trigger[i] → line of the mapping node
}

// ── public API ────────────────────────────────────────────────────────────────

// ParseFile opens path and parses it as a YAML workflow definition.
func ParseFile(path string) (*core.WorkflowDefinition, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	def, _, err := parseInternal(f, path)
	return def, err
}

// Parse reads YAML from r and parses it as a workflow definition.
// filename is used in error messages.
func Parse(r io.Reader, filename string) (*core.WorkflowDefinition, error) {
	def, _, err := parseInternal(r, filename)
	return def, err
}

// ── internal ──────────────────────────────────────────────────────────────────

// parseInternal is the canonical parse path; it returns both the definition
// and the line map so C2's renderer can attach source coordinates to Issues.
func parseInternal(r io.Reader, filename string) (*core.WorkflowDefinition, *lineMap, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: read: %w", filename, err)
	}

	// Pass 1: strict decode — rejects unknown fields (IMP §27.M10 Risk row).
	var raw ymlWorkflowDef
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", filename, err)
	}

	// Pass 2: node decode for line-number capture (C2 renderer, PRD §18).
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", filename, err)
	}
	lm := buildLineMap(&doc)

	def, err := rawToCore(raw, filename)
	if err != nil {
		return nil, nil, err
	}
	return def, lm, nil
}

// buildLineMap walks the yaml.Node document tree and records source line
// numbers for top-level fields, steps (keyed by id), transitions, and triggers.
func buildLineMap(doc *yaml.Node) *lineMap {
	lm := &lineMap{TopLevel: make(map[string]int)}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return lm
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return lm
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		val := mapping.Content[i+1]
		lm.TopLevel[key.Value] = key.Line
		switch key.Value {
		case "steps":
			lm.Steps = make(map[string]int)
			if val.Kind == yaml.SequenceNode {
				for _, n := range val.Content {
					if id := yamlMappingValue(n, "id"); id != "" {
						lm.Steps[id] = n.Line
					}
				}
			}
		case "transitions":
			if val.Kind == yaml.SequenceNode {
				for _, n := range val.Content {
					lm.Transitions = append(lm.Transitions, n.Line)
				}
			}
		case "triggers":
			if val.Kind == yaml.SequenceNode {
				for _, n := range val.Content {
					lm.Triggers = append(lm.Triggers, n.Line)
				}
			}
		}
	}
	return lm
}

// yamlMappingValue returns the scalar string value for key inside a YAML
// MappingNode, or "" if not found.
func yamlMappingValue(node *yaml.Node, key string) string {
	if node.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1].Value
		}
	}
	return ""
}

// rawToCore converts the intermediate YAML struct to a core.WorkflowDefinition
// and enforces schema_version and semver constraints (TDS-02 §1/§5/§6).
func rawToCore(raw ymlWorkflowDef, filename string) (*core.WorkflowDefinition, error) {
	if raw.SchemaVersion != 1 {
		return nil, fmt.Errorf("%s: schema_version must be 1, got %d", filename, raw.SchemaVersion)
	}
	if err := validateSemver(raw.Version); err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}

	triggers := make([]core.Trigger, len(raw.Triggers))
	for i, t := range raw.Triggers {
		cfg := t.Config
		if cfg == nil {
			cfg = map[string]any{}
		}
		triggers[i] = core.Trigger{
			Type:   core.TriggerType(t.Type),
			Config: cfg,
		}
	}

	steps := make([]core.Step, len(raw.Steps))
	for i, s := range raw.Steps {
		step := core.Step{
			ID:       s.ID,
			Name:     s.Name,
			Type:     core.StepType(s.Type),
			Handler:  core.HandlerRef(s.Handler),
			Inputs:   s.Inputs,
			Outputs:  s.Outputs,
			Timeout:  core.Duration(s.Timeout),
			Fallback: s.Fallback,
		}
		if s.Retry != nil {
			step.Retry = &core.RetryPolicy{
				Attempts:        s.Retry.Attempts,
				Backoff:         s.Retry.Backoff,
				InitialDelay:    core.Duration(s.Retry.InitialDelay),
				MaxDelay:        core.Duration(s.Retry.MaxDelay),
				RetryableErrors: s.Retry.RetryableErrors,
			}
		}
		if s.WaitSignal != nil {
			step.WaitSignal = &core.WaitConfig{
				SignalName:    s.WaitSignal.SignalName,
				Timeout:       core.Duration(s.WaitSignal.Timeout),
				TimeoutAction: s.WaitSignal.TimeoutAction,
			}
		}
		if s.Intelligence != nil {
			step.Intelligence = &core.IntelReq{
				Capability:    s.Intelligence.Capability,
				ModelHint:     s.Intelligence.ModelHint,
				ContextBudget: s.Intelligence.ContextBudget,
				Required:      s.Intelligence.Required,
			}
		}
		if s.Compensation != nil {
			step.Compensation = &core.CompensationRef{
				Handler: core.HandlerRef(s.Compensation.Handler),
			}
		}
		steps[i] = step
	}

	transitions := make([]core.Transition, len(raw.Transitions))
	for i, t := range raw.Transitions {
		transitions[i] = core.Transition{
			From:      t.From,
			To:        t.To,
			Condition: core.Condition(t.Condition),
			OnError:   t.OnError,
		}
	}

	var comp *core.CompensationPlan
	if raw.Compensation != nil {
		cs := make([]core.CompensationStep, len(raw.Compensation.Steps))
		for i, s := range raw.Compensation.Steps {
			cs[i] = core.CompensationStep{
				StepID:      s.StepID,
				UndoHandler: core.HandlerRef(s.UndoHandler),
			}
			if s.Retry != nil {
				cs[i].Retry = &core.RetryPolicy{
					Attempts:        s.Retry.Attempts,
					Backoff:         s.Retry.Backoff,
					InitialDelay:    core.Duration(s.Retry.InitialDelay),
					MaxDelay:        core.Duration(s.Retry.MaxDelay),
					RetryableErrors: s.Retry.RetryableErrors,
				}
			}
		}
		comp = &core.CompensationPlan{Steps: cs}
	}

	meta := raw.Metadata
	if meta == nil {
		meta = map[string]any{}
	}

	return &core.WorkflowDefinition{
		SchemaVersion: raw.SchemaVersion,
		ID:            raw.ID,
		Version:       core.SemVer(raw.Version),
		Namespace:     raw.Namespace,
		Name:          raw.Name,
		Description:   raw.Description,
		Triggers:      triggers,
		Steps:         steps,
		Transitions:   transitions,
		InitialStep:   raw.InitialStep,
		FinalSteps:    raw.FinalSteps,
		Compensation:  comp,
		Timeout:       core.Duration(raw.Timeout),
		Metadata:      meta,
	}, nil
}
