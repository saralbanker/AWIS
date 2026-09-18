package api

import (
	"net/http"

	"github.com/awis/awis/internal/core"
)

// workflowEntry is the response-DTO convention for one row of
// GET /api/v1/workflows, mirroring cmd/awis/workflow.go's own
// workflowEntryJSON shape (id/version/namespace/step_count) rather than
// marshaling core.WorkflowDefinition's full body for a list view.
type workflowEntry struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	Namespace string `json:"namespace"`
	StepCount int    `json:"step_count"`
}

// handleListWorkflows implements GET /api/v1/workflows, backed by
// StoragePort.ListWorkflows. An optional ?namespace= query param filters;
// omitted or empty means every namespace — StoragePort.ListWorkflows'
// documented behaviour, and the same default cmd/awis/workflow.go's own
// `workflow list` command uses.
func handleListWorkflows(store core.StoragePort) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		namespace := r.URL.Query().Get("namespace")

		defs, err := store.ListWorkflows(r.Context(), namespace)
		if err != nil {
			return err
		}

		entries := make([]workflowEntry, 0, len(defs))
		for _, d := range defs {
			entries = append(entries, workflowEntry{
				ID:        d.ID,
				Version:   string(d.Version),
				Namespace: d.Namespace,
				StepCount: len(d.Steps),
			})
		}
		return writeJSON(w, http.StatusOK, entries)
	}
}

// handleGetWorkflow implements GET /api/v1/workflows/{id}/{version}, backed
// by StoragePort.GetWorkflow. A missing (id, version) pair surfaces
// storage.ErrWorkflowNotFound (E-G1-3) through wrap()'s error-mapping
// middleware as a 404, not a 500 or an empty 200 — the card's acceptance
// criterion. core.WorkflowDefinition is returned in full: it already carries
// complete json tags (internal/core/workflow.go) and, unlike the list view,
// a single-definition fetch is exactly what a canvas/editor view needs.
func handleGetWorkflow(store core.StoragePort) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		version := r.PathValue("version")

		def, err := store.GetWorkflow(r.Context(), id, core.SemVer(version))
		if err != nil {
			return err
		}
		return writeJSON(w, http.StatusOK, def)
	}
}
