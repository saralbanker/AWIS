package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/validate"
)

// Submit starts a new workflow instance for the definition identified by
// (definitionID, version) and returns its id. It:
//
//   - loads the definition from the registry at the EXPLICIT version (latest
//     selection is M08's; Submit requires a version at M06);
//   - validates it (any Issue ⇒ rejection, FR-WD-04);
//   - mints a new instance id and emits WorkflowStarted (seq 1), which creates
//     the running projection row (EDR-007: status running, StartedAt=emitted_at,
//     variables={"inputs": inputs}, current_steps=[]).
//
// The InitialStep is NOT dispatched here; it becomes activatable on the next tick
// (derived statelessly from the projection — see transition.go).
func (e *Engine) Submit(ctx context.Context, definitionID string, version core.SemVer, inputs map[string]any) (core.InstanceID, error) {
	def, err := e.storage.GetWorkflow(ctx, definitionID, version)
	if err != nil {
		return "", fmt.Errorf("engine: Submit load %s@%s: %w", definitionID, version, err)
	}

	if issues := validate.Validate(def); len(issues) > 0 {
		return "", fmt.Errorf("engine: Submit rejected %s@%s: %s", definitionID, version, formatIssues(issues))
	}

	dv, err := buildDefView(def)
	if err != nil {
		return "", err
	}
	e.mu.Lock()
	e.defs[defKey{id: def.ID, version: def.Version}] = dv
	e.mu.Unlock()

	iid := core.InstanceID(e.newID())
	e.mu.Lock()
	e.seq[iid] = 0
	e.ver[iid] = 0
	e.mu.Unlock()

	if inputs == nil {
		inputs = map[string]any{}
	}
	if err := e.emitWorkflowStarted(ctx, iid, def.Namespace, def.ID, def.Version, inputs); err != nil {
		return "", err
	}
	return iid, nil
}

// formatIssues renders validation issues into a single error string (FR-WD-04).
// Rendering with file/line + fix suggestion (PRD §18 format) is M10/M14; this is
// the engine's coarse rejection message.
func formatIssues(issues []validate.Issue) string {
	parts := make([]string, 0, len(issues))
	for _, is := range issues {
		loc := is.Field
		if is.StepID != "" {
			loc = is.StepID + "." + loc
		}
		parts = append(parts, fmt.Sprintf("[%s] %s: %s", is.Code, loc, is.Message))
	}
	return strings.Join(parts, "; ")
}
