//go:build integration

package integration

import "encoding/json"

// submitOutput mirrors 'awis submit --json' (TDS-07 §4).
type submitOutput struct {
	InstanceID string `json:"instance_id"`
	WorkflowID string `json:"workflow_id"`
	Status     string `json:"status"`
}

// traceEventJSON mirrors one element of the 'events' array in
// 'awis --json trace <id>' (TDS-07 §4).
type traceEventJSON struct {
	Seq       int             `json:"seq"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

// traceOutputJSON mirrors 'awis --json trace <id>' (TDS-07 §4).
type traceOutputJSON struct {
	InstanceID string           `json:"instance_id"`
	Status     string           `json:"status"`
	Events     []traceEventJSON `json:"events"`
}

// findEvent returns the first event of the given event_type, or nil.
func findEvent(events []traceEventJSON, eventType string) *traceEventJSON {
	for i := range events {
		if events[i].EventType == eventType {
			return &events[i]
		}
	}
	return nil
}

// hasRetryingStepFailed reports whether events contains a
// StepFailed{retrying:true} entry (TDS-01 §2 StepFailed payload).
func hasRetryingStepFailed(events []traceEventJSON) bool {
	for _, ev := range events {
		if ev.EventType != "StepFailed" {
			continue
		}
		var p struct {
			Retrying bool `json:"retrying"`
		}
		if json.Unmarshal(ev.Payload, &p) == nil && p.Retrying {
			return true
		}
	}
	return false
}
