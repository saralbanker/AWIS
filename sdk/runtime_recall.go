// RecallAPI implementation on *Runtime (Blueprint §12; IMP §27.M8 T7).
// QueryHistory, ReplayInstance, and StepStats are best-effort reads over the
// storage layer.
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

// compile-time assertion: *Runtime satisfies core.RecallAPI.
var _ core.RecallAPI = (*Runtime)(nil)

// QueryHistory returns execution records for instances matching the query.
// It reads workflow_instances via ListInstances (filtered by HistoryQuery
// fields) and derives StartedAt/CompletedAt from the instance fields.
func (r *Runtime) QueryHistory(ctx context.Context, query core.HistoryQuery) ([]core.ExecutionRecord, error) {
	filter := core.InstanceFilter{
		Namespace: query.Namespace,
		Status:    query.Status,
	}
	instances, err := r.storage.ListInstances(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("sdk: QueryHistory: %w", err)
	}

	var records []core.ExecutionRecord
	for _, inst := range instances {
		if query.DefinitionID != "" && inst.DefinitionID != query.DefinitionID {
			continue
		}
		inputs, outputs := splitVariables(inst.Variables)
		rec := core.ExecutionRecord{
			InstanceID:   inst.InstanceID,
			DefinitionID: inst.DefinitionID,
			Version:      inst.DefinitionVersion,
			Status:       inst.Status,
			StartedAt:    inst.StartedAt,
			CompletedAt:  inst.CompletedAt,
			Inputs:       inputs,
			Outputs:      outputs,
		}
		records = append(records, rec)
	}
	return records, nil
}

// ReplayInstance reads the instance and its full EventLog (from sequence 0)
// and returns a ReplayTrace carrying the instance's identity/lifecycle fields
// plus the complete ordered event list. It returns a non-nil error — never a
// zero-value ReplayTrace — when instanceID does not exist or either storage
// read fails.
func (r *Runtime) ReplayInstance(ctx context.Context, instanceID core.InstanceID) (core.ReplayTrace, error) {
	inst, err := r.storage.GetInstance(ctx, instanceID)
	if err != nil {
		return core.ReplayTrace{}, fmt.Errorf("sdk: ReplayInstance: %w", err)
	}
	events, err := r.storage.ReadEvents(ctx, instanceID, 0)
	if err != nil {
		return core.ReplayTrace{}, fmt.Errorf("sdk: ReplayInstance: %w", err)
	}
	return core.ReplayTrace{
		InstanceID:   inst.InstanceID,
		DefinitionID: inst.DefinitionID,
		Version:      inst.DefinitionVersion,
		Namespace:    inst.Namespace,
		Status:       inst.Status,
		StartedAt:    inst.StartedAt,
		CompletedAt:  inst.CompletedAt,
		Events:       events,
	}, nil
}

// StepStats returns aggregate statistics for stepID across instances of
// definitionID in r.namespace (B-11c). It reads StepStarted/StepCompleted/
// StepFailed events from the storage EventLog over a fixed 10-year lookback
// window (see below) to derive the counts and average duration.
//
// definitionID scoping: ExecutionEvent carries an InstanceID but not a
// definition id, so StepStats first calls ListInstances(r.namespace) to build
// an instance→definition map, then only considers events whose instance
// belongs to definitionID. This prevents two workflows that both have a step
// called stepID from having their statistics silently merged. If definitionID
// is "" (empty), no instance filter is applied and StepStats aggregates
// stepID across every workflow in the namespace — this is the pre-existing
// cross-workflow behaviour, preserved intentionally for that one case.
//
// Field definitions (exact, since a GUI consumer reads these directly):
//   - TotalRuns counts every StepStarted event for stepID, i.e. every
//     dispatch attempt including retries. A step that fails once and
//     succeeds on retry contributes 2 to TotalRuns (one per attempt).
//   - SuccessCount counts StepCompleted events for stepID (one per
//     successful attempt; at most one per step activation, since a
//     completed step does not retry).
//   - FailureCount counts only *terminal* StepFailed events for stepID,
//     i.e. payload {retrying:false} — an attempt that failed but is about
//     to be retried (payload {retrying:true}) is NOT counted here.
//   - Consequently SuccessCount+FailureCount <= TotalRuns, with equality
//     only when no step activation in scope ever retried; each retried
//     attempt adds to TotalRuns without adding to either count until the
//     step activation reaches a terminal (success or non-retrying failure)
//     outcome.
//   - AvgDurationMs is the mean of duration_ms across StepCompleted events
//     (0 when SuccessCount is 0).
//
// Note: the 10-year ReadEventRange window is scanned on every call; this is
// acceptable at V1 scale but not optimized for large histories.
func (r *Runtime) StepStats(ctx context.Context, definitionID, stepID string) (core.StepStatistics, error) {
	// Read events for a wide time range covering all history (V1: 10 years back).
	now := time.Now()
	from := now.Add(-10 * 365 * 24 * time.Hour)
	events, err := r.storage.ReadEventRange(ctx, r.namespace, from, now)
	if err != nil {
		return core.StepStatistics{}, fmt.Errorf("sdk: StepStats: %w", err)
	}

	// When definitionID is non-empty, restrict events to instances that
	// belong to that definition (B-11c). allowedInstances stays nil for
	// definitionID=="" so the filter below is a no-op, preserving the
	// pre-existing cross-workflow behaviour for that case.
	var allowedInstances map[core.InstanceID]bool
	if definitionID != "" {
		instances, lerr := r.storage.ListInstances(ctx, core.InstanceFilter{Namespace: r.namespace})
		if lerr != nil {
			return core.StepStatistics{}, fmt.Errorf("sdk: StepStats: %w", lerr)
		}
		allowedInstances = make(map[core.InstanceID]bool, len(instances))
		for _, inst := range instances {
			if inst.DefinitionID == definitionID {
				allowedInstances[inst.InstanceID] = true
			}
		}
	}

	stats := core.StepStatistics{
		DefinitionID: definitionID,
		StepID:       stepID,
	}
	var totalDurationMs float64
	for _, ev := range events {
		if allowedInstances != nil && !allowedInstances[ev.InstanceID] {
			continue
		}

		// Filter by step_id in JSON payload.
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			continue
		}

		// Check step_id field in payload if present; also check the StepID column.
		sid := ev.StepID
		if raw, ok := payload["step_id"]; ok {
			var s string
			if jsonErr := json.Unmarshal(raw, &s); jsonErr == nil {
				sid = s
			}
		}
		if sid != stepID {
			continue
		}

		switch ev.EventType {
		case core.EventTypeStepStarted:
			stats.TotalRuns++
		case core.EventTypeStepCompleted:
			stats.SuccessCount++
			if raw, ok := payload["duration_ms"]; ok {
				var d float64
				if jsonErr := json.Unmarshal(raw, &d); jsonErr == nil {
					totalDurationMs += d
				}
			}
		case core.EventTypeStepFailed:
			// Only count final failures (retrying == false).
			var retrying bool
			if raw, ok := payload["retrying"]; ok {
				_ = json.Unmarshal(raw, &retrying)
			}
			if !retrying {
				stats.FailureCount++
			}
		}
	}
	if stats.SuccessCount > 0 {
		stats.AvgDurationMs = totalDurationMs / float64(stats.SuccessCount)
	}
	return stats, nil
}
