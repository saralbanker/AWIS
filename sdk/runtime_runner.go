// WorkflowRunner implementation on *Runtime (Blueprint §12; IMP §27.M8 T5).
// Each method is a thin wrapper over the internal engine or storage.
package sdk

import (
	"context"
	"fmt"

	"github.com/awis/awis/internal/core"
)

// compile-time assertion: *Runtime satisfies core.WorkflowRunner.
var _ core.WorkflowRunner = (*Runtime)(nil)

// Submit starts a new workflow instance. It looks up the latest registered
// version for definitionID from the in-memory registry (r.defs) and falls
// back to storage if the id is not found in-process (e.g. when submitting
// to a workflow registered by another process against the same DB — M15-P1
// seam; IMP §17 cross-process submit).
func (r *Runtime) Submit(ctx context.Context, definitionID string, inputs map[string]any) (core.InstanceID, error) {
	r.mu.Lock()
	// Find the latest registered version for this definitionID.
	// We track registrations in r.defs keyed as "id@version"; iterate to find
	// a match. V1: last-registered-wins for "latest".
	var latestVersion core.SemVer
	for key, ver := range r.defs {
		if len(key) > len(definitionID)+1 && key[:len(definitionID)+1] == definitionID+"@" {
			latestVersion = ver
		}
	}
	r.mu.Unlock()

	// Fallback: query storage when not in the in-memory registry. This allows a
	// submit process (e.g. awis CLI) to target workflows registered by a separate
	// runtime process against the same DB (M15-P1; IMP §17 cross-process submit).
	// Scan only r.namespace — callers must set the correct namespace; no
	// application names may appear in platform code (QG-4 boundary).
	if latestVersion == "" {
		if defs, lerr := r.storage.ListWorkflows(ctx, r.namespace); lerr == nil {
			for _, def := range defs {
				if def.ID == definitionID {
					if latestVersion == "" || def.Version > latestVersion {
						latestVersion = def.Version
					}
				}
			}
		}
	}

	if latestVersion == "" {
		return "", fmt.Errorf("sdk: Submit: no registered workflow with id %q", definitionID)
	}
	return r.eng.Submit(ctx, definitionID, latestVersion, inputs)
}

// Signal delivers a named signal to a waiting workflow instance.
func (r *Runtime) Signal(ctx context.Context, instanceID core.InstanceID, signalName string, payload map[string]any) error {
	return r.eng.Signal(ctx, instanceID, signalName, payload)
}

// Status returns the current state of a workflow instance as a WorkflowStatus.
func (r *Runtime) Status(ctx context.Context, instanceID core.InstanceID) (core.WorkflowStatus, error) {
	inst, err := r.storage.GetInstance(ctx, instanceID)
	if err != nil {
		return core.WorkflowStatus{}, fmt.Errorf("sdk: Status: %w", err)
	}
	return instanceToStatus(inst), nil
}

// Cancel requests cancellation of a running workflow instance.
func (r *Runtime) Cancel(ctx context.Context, instanceID core.InstanceID, reason string) error {
	return r.eng.Cancel(ctx, instanceID, reason, false)
}

// List returns workflow instances matching the given filter, mapped to
// []WorkflowStatus.
func (r *Runtime) List(ctx context.Context, filter core.InstanceFilter) ([]core.WorkflowStatus, error) {
	instances, err := r.storage.ListInstances(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("sdk: List: %w", err)
	}
	out := make([]core.WorkflowStatus, len(instances))
	for i, inst := range instances {
		out[i] = instanceToStatus(inst)
	}
	return out, nil
}

// instanceToStatus maps a core.WorkflowInstance to a core.WorkflowStatus.
// Variables contains both inputs and accumulated outputs; we expose the full
// map as both Inputs and Outputs at V1 (IMP §27.M8: shape freeze only).
func instanceToStatus(inst core.WorkflowInstance) core.WorkflowStatus {
	return core.WorkflowStatus{
		InstanceID:   inst.InstanceID,
		DefinitionID: inst.DefinitionID,
		Version:      inst.DefinitionVersion,
		Status:       inst.Status,
		CurrentSteps: inst.CurrentSteps,
		Inputs:       inst.Variables,
		Outputs:      inst.Variables,
		CreatedAt:    inst.StartedAt,
		UpdatedAt:    inst.UpdatedAt,
	}
}
