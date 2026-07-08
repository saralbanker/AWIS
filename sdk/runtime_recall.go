// RecallAPI implementation on *Runtime (Blueprint §12; IMP §27.M8 T7).
// QueryHistory and StepStats are best-effort reads over the storage layer.
// ReplayInstance is a stub (M17).
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
		rec := core.ExecutionRecord{
			InstanceID:   inst.InstanceID,
			DefinitionID: inst.DefinitionID,
			Version:      inst.DefinitionVersion,
			Status:       inst.Status,
			StartedAt:    inst.StartedAt,
			CompletedAt:  inst.CompletedAt,
			Inputs:       inst.Variables,
			Outputs:      inst.Variables,
		}
		records = append(records, rec)
	}
	return records, nil
}

// ReplayInstance is a stub; full implementation is deferred to M17.
// TODO(M17): implement replay trace construction.
func (r *Runtime) ReplayInstance(_ context.Context, _ core.InstanceID) (core.ReplayTrace, error) {
	return core.ReplayTrace{}, nil
}

// StepStats returns aggregate statistics for a step across all instances.
// It reads StepStarted and StepCompleted events from the storage EventLog
// to derive totals, success/failure counts, and average duration.
func (r *Runtime) StepStats(ctx context.Context, definitionID, stepID string) (core.StepStatistics, error) {
	// Read events for a wide time range covering all history (V1: 10 years back).
	now := time.Now()
	from := now.Add(-10 * 365 * 24 * time.Hour)
	events, err := r.storage.ReadEventRange(ctx, r.namespace, from, now)
	if err != nil {
		return core.StepStatistics{}, fmt.Errorf("sdk: StepStats: %w", err)
	}

	stats := core.StepStatistics{
		DefinitionID: definitionID,
		StepID:       stepID,
	}
	var totalDurationMs float64
	for _, ev := range events {
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
