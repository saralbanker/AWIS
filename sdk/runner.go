package sdk

import "github.com/awis/awis/internal/core"

// WorkflowRunner is the application-facing runtime control surface for workflow
// instances: Submit starts one, Signal delivers to a waiting one, Status reads
// current state, Cancel requests cancellation, and List queries instances.
type WorkflowRunner = core.WorkflowRunner

// WorkflowStatus is the summarized current state of an instance returned by the
// runner. Its shape is completed at M08; it is not part of the G1 format freeze
// (sdk surface mutable until M08).
type WorkflowStatus = core.WorkflowStatus

// InstanceFilter selects instances for List. Its shape is completed at M08; it is
// not part of the G1 format freeze (sdk surface mutable until M08).
type InstanceFilter = core.InstanceFilter
