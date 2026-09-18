// Package sdk re-exports the platform's YAML DSL loader so applications can
// register workflows from YAML files without importing the platform's internal
// packages. This seam is justified by IMP §17 L324 (YAML tier in SDK rollout)
// and by the M15 disposition recorded in M15 TRACEABILITY.md.
package sdk

import (
	"github.com/awis/awis/internal/dsl"
)

// LoadWorkflowFile parses a YAML workflow definition file and returns a
// *WorkflowDefinition ready for registration with [Runtime.RegisterWorkflow].
// It is a thin re-export of internal/dsl.ParseFile; no transformation is
// applied. The returned struct is identical to what [WorkflowBuilder.Build]
// produces (IMP §17 L324; M15 disposition: scoped platform seam, disclosed
// to G3).
func LoadWorkflowFile(path string) (*WorkflowDefinition, error) {
	return dsl.ParseFile(path)
}
