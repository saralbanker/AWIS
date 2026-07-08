// WorkflowBuilder provides a fluent Go API for constructing WorkflowDefinitions
// (IMP §27.M8; Blueprint §12; PRD FR-SDK-01/11). No new dependencies; semver
// validation uses stdlib regexp only.
package sdk

import (
	"fmt"
	"regexp"

	"github.com/awis/awis/internal/core"
)

// semverRE validates a semver version string per IMP §27.M8.
// Accepts pre-release and build-metadata suffixes (e.g. "2.1.3-alpha").
var semverRE = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)`)

// validateSemver returns a non-nil error if version is not a valid semver.
// It is used by both WorkflowBuilder.Build and RegisterWorkflow.
func validateSemver(version string) error {
	if version == "" {
		return fmt.Errorf("version must not be empty")
	}
	if !semverRE.MatchString(version) {
		return fmt.Errorf("version %q does not match semver format (e.g. \"1.0.0\")", version)
	}
	return nil
}

// WorkflowBuilder is a fluent builder for core.WorkflowDefinition.
// Construct via NewWorkflowBuilder; call Build() to validate and produce
// the definition.
type WorkflowBuilder struct {
	id          string
	version     string
	name        string
	description string
	namespace   string
	steps       []core.Step
	transitions []core.Transition
	initialStep string
	finalSteps  []string
	triggers    []core.Trigger
	metadata    map[string]any
}

// NewWorkflowBuilder returns a WorkflowBuilder seeded with the given id and
// version. Additional fields are set via the builder methods.
func NewWorkflowBuilder(id, version string) *WorkflowBuilder {
	return &WorkflowBuilder{
		id:      id,
		version: version,
	}
}

// SetName sets the human-readable name of the workflow.
func (b *WorkflowBuilder) SetName(name string) *WorkflowBuilder {
	b.name = name
	return b
}

// SetDescription sets the optional human-readable description.
func (b *WorkflowBuilder) SetDescription(desc string) *WorkflowBuilder {
	b.description = desc
	return b
}

// SetNamespace sets the owning namespace of the workflow.
func (b *WorkflowBuilder) SetNamespace(ns string) *WorkflowBuilder {
	b.namespace = ns
	return b
}

// AddStep appends a step to the workflow definition.
func (b *WorkflowBuilder) AddStep(s core.Step) *WorkflowBuilder {
	b.steps = append(b.steps, s)
	return b
}

// AddTransition appends a transition to the workflow definition.
func (b *WorkflowBuilder) AddTransition(t core.Transition) *WorkflowBuilder {
	b.transitions = append(b.transitions, t)
	return b
}

// SetInitialStep sets the id of the first step to execute.
func (b *WorkflowBuilder) SetInitialStep(id string) *WorkflowBuilder {
	b.initialStep = id
	return b
}

// AddFinalStep marks a step id as a terminal step.
func (b *WorkflowBuilder) AddFinalStep(id string) *WorkflowBuilder {
	b.finalSteps = append(b.finalSteps, id)
	return b
}

// AddTrigger appends a trigger declaration to the workflow definition.
func (b *WorkflowBuilder) AddTrigger(t core.Trigger) *WorkflowBuilder {
	b.triggers = append(b.triggers, t)
	return b
}

// SetMetadata sets the application-defined metadata map.
func (b *WorkflowBuilder) SetMetadata(m map[string]any) *WorkflowBuilder {
	b.metadata = m
	return b
}

// Build validates the builder state and returns a *core.WorkflowDefinition.
// Returns an error if id is empty or version is not a valid semver string.
func (b *WorkflowBuilder) Build() (*core.WorkflowDefinition, error) {
	if b.id == "" {
		return nil, fmt.Errorf("awis: WorkflowBuilder.Build: id must not be empty")
	}
	if err := validateSemver(b.version); err != nil {
		return nil, fmt.Errorf("awis: WorkflowBuilder.Build: %w", err)
	}
	meta := b.metadata
	if meta == nil {
		meta = map[string]any{}
	}
	triggers := b.triggers
	if triggers == nil {
		triggers = []core.Trigger{}
	}
	def := &core.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            b.id,
		Version:       core.SemVer(b.version),
		Namespace:     b.namespace,
		Name:          b.name,
		Description:   b.description,
		Triggers:      triggers,
		Steps:         b.steps,
		Transitions:   b.transitions,
		InitialStep:   b.initialStep,
		FinalSteps:    b.finalSteps,
		Metadata:      meta,
	}
	return def, nil
}
