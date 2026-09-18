// WorkflowRunner implementation on *Runtime (Blueprint §12; IMP §27.M8 T5).
// Each method is a thin wrapper over the internal engine or storage.
package sdk

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
	// submit process (e.g. the awis CLI) to target workflows registered by a
	// separate runtime process against the same DB (M15-P1; IMP §17 cross-process
	// submit).
	//
	// The scan is two-stage (B-23). A workflow's namespace comes from its own
	// YAML, while the Runtime's namespace comes from a flag, and the two are
	// unrelated: `awis start` registers definitions from every namespace it
	// discovers regardless of --namespace. Scanning only r.namespace therefore
	// made the documented quickstart fail outright — the scaffolded workflows
	// declare `namespace: examples` while the CLI defaults to "default", so
	// `awis submit hello-world` reported "no registered workflow with id
	// \"hello-world\"" on a freshly initialised project.
	//
	// Stage 1 prefers the Runtime's own namespace, so an explicit --namespace
	// still wins and stays unambiguous. Stage 2 widens to every namespace. An id
	// that resolves in more than one namespace is an ERROR naming the candidates,
	// never a silent pick: guessing which of two same-named workflows the operator
	// meant is worse than making them say.
	// Stage 1: the Runtime's own namespace.
	if latestVersion == "" {
		latestVersion, _ = latestInNamespace(ctx, r.storage, r.namespace, definitionID)
	}
	// Stage 2: every namespace. ListWorkflows with an empty namespace means "no
	// predicate", matching ListInstances' filter idiom.
	//
	// Note that workflow_definitions is keyed by (id, version) GLOBALLY —
	// namespace is not part of the primary key — so an id can only appear in two
	// namespaces at two different versions. The ambiguity branch below therefore
	// fires rarely, but when it does the two definitions are genuinely unrelated
	// workflows that happen to share an id, and picking one would be a guess.
	if latestVersion == "" {
		ver, nsCandidates := latestInNamespace(ctx, r.storage, "", definitionID)
		if len(nsCandidates) > 1 {
			return "", fmt.Errorf(
				"sdk: Submit: workflow id %q is registered in %d namespaces (%s); "+
					"re-run with an explicit namespace to disambiguate",
				definitionID, len(nsCandidates), strings.Join(nsCandidates, ", "))
		}
		latestVersion = ver
	}

	if latestVersion == "" {
		return "", fmt.Errorf("sdk: Submit: no registered workflow with id %q", definitionID)
	}
	return r.eng.Submit(ctx, definitionID, latestVersion, inputs)
}

// latestInNamespace returns the highest registered version of definitionID
// within namespace (empty namespace = every namespace), along with the sorted
// distinct namespaces the id was found in. A storage error yields no match
// rather than an error: the caller treats "not found here" and "could not look
// here" identically and falls through to its next stage.
func latestInNamespace(ctx context.Context, store core.StoragePort, namespace, definitionID string) (core.SemVer, []string) {
	defs, err := store.ListWorkflows(ctx, namespace)
	if err != nil {
		return "", nil
	}
	var latest core.SemVer
	seen := make(map[string]bool)
	var namespaces []string
	for _, def := range defs {
		if def.ID != definitionID {
			continue
		}
		if !seen[def.Namespace] {
			seen[def.Namespace] = true
			namespaces = append(namespaces, def.Namespace)
		}
		if latest == "" || def.Version > latest {
			latest = def.Version
		}
	}
	sort.Strings(namespaces)
	return latest, namespaces
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
// Variables contains both the workflow inputs (under the "inputs" key) and
// each step's accumulated outputs (under its step id); splitVariables (B-11b)
// separates them so Inputs and Outputs are never the same map.
func instanceToStatus(inst core.WorkflowInstance) core.WorkflowStatus {
	inputs, outputs := splitVariables(inst.Variables)
	return core.WorkflowStatus{
		InstanceID:   inst.InstanceID,
		DefinitionID: inst.DefinitionID,
		Version:      inst.DefinitionVersion,
		Status:       inst.Status,
		CurrentSteps: inst.CurrentSteps,
		Inputs:       inputs,
		Outputs:      outputs,
		CreatedAt:    inst.StartedAt,
		UpdatedAt:    inst.UpdatedAt,
	}
}
