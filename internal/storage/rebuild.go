// Package storage — RebuildState projection library (M03-C2 / EDR-007).
//
// RebuildState replays the append-only EventLog and writes a fresh
// workflow_instances projection. It is a library function that predates the
// engine (IMP §26) and is safe to call at any time.
//
// This file intentionally does NOT route through UpsertInstance: that method
// carries runtime version/claim semantics (optimistic-lock coupling, claim
// release on terminal status) that are inappropriate for a batch projection
// write. RebuildState issues direct INSERTs inside a single transaction so
// the projection is atomic from the perspective of any concurrent reader.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

// RebuildState wipes the workflow_instances and step_claims tables and
// replays ALL events in the EventLog to reconstruct the projection from
// scratch (NFR-R-03 / EDR-007).
//
// Algorithm:
//  1. Snapshot pre-wipe (instance_id → definition_id, definition_version,
//     cancellation_requested) triples. These fields are not evented
//     (EDR-007 definition-identity gap; cancellation_requested is a
//     non-evented runtime request flag, cancellation.go); preserving them
//     from the existing rows avoids losing identity or a pending
//     cancellation request on rebuild. Instances absent from the snapshot
//     receive empty strings / false.
//  2. Snapshot the durable wait_records table (migration 0003_signals.sql;
//     NOT wiped by this rebuild) to learn which instances are parked on a
//     live wait. `waiting` is the other non-evented projection status
//     (EDR-007 §9 gap): entering a wait is a tick-time decision that emits
//     no event, so a pure event replay can never reconstruct it. Deriving
//     it from wait_records instead of the old projection row is strictly
//     more correct because wait_records is the same durable source the
//     engine itself recovers waits from after a restart (signal.go's
//     recoverWait) — rebuild and runtime agree by construction.
//  3. Open a single transaction; DELETE workflow_instances + step_claims;
//     replay every event ordered by sequence_num per instance; for any
//     instance the replay leaves in a NON-terminal status that also has a
//     live wait record, override the projected status to `waiting` (a
//     terminal status from the EventLog always wins over a stale
//     wait_records row — an instance can never be resurrected out of
//     completed/failed/cancelled/compensated/compensation_failed); INSERT
//     the projected rows. step_claims remain wiped — they are runtime
//     state, not history.
//  4. Commit atomically.
//
// Rebuilt rows are written with version = 1 (fresh projection generation,
// documented in EDR-007).
//
// Defensive rule (pre-authorised by card): if an instance's event stream
// begins without a WorkflowStarted event the projection starts from zero
// values, taking the namespace from the first event's envelope.
//
// Concurrency assumption: RebuildState is a recovery operation and assumes no
// concurrent writer (local mode has none — Blueprint §9; `awis rebuild-state`
// runs against a stopped runtime). Events are read before the write
// transaction opens; an append interleaved between read and commit would not
// be reflected until the next rebuild.
func (s *SQLiteStorage) RebuildState(ctx context.Context) error {
	// ── Step 0: refuse to rebuild across an unsupported schema version (B-22) ──
	//
	// AppendEvent, ReadEvents and ReadEventRange all reject rows above
	// MaxSupportedSchemaVersion on the reasoning that mis-projecting the
	// append-only source of truth is worse than a failed read. This replay
	// path reads execution_events with its own raw SQL and so bypassed all
	// three, which made `awis rebuild-state` the one command that would
	// silently reinterpret a newer build's events under v1 field assumptions —
	// and it does so while WIPING the projection, so the mis-projection
	// replaces correct state rather than merely accompanying it.
	//
	// The check is pre-flight, before any destructive work: a rebuild is
	// all-or-nothing, so discovering this halfway through would leave the
	// operator with a wiped projection and an error.
	var maxSchema sql.NullInt64
	if err := s.db.db.QueryRowContext(ctx,
		`SELECT MAX(schema_version) FROM execution_events`).Scan(&maxSchema); err != nil {
		return fmt.Errorf("storage: RebuildState schema-version check: %w", err)
	}
	if maxSchema.Valid && maxSchema.Int64 > MaxSupportedSchemaVersion {
		return fmt.Errorf("%w: event log contains schema_version %d, max supported is %d",
			ErrUnsupportedSchemaVersion, maxSchema.Int64, MaxSupportedSchemaVersion)
	}

	// ── Step 1: snapshot definition identity + cancellation flag (EDR-007 gap) ─
	type defIdentity struct {
		definitionID          string
		definitionVersion     string
		cancellationRequested bool
	}
	snapshot := make(map[string]defIdentity)

	identRows, err := s.db.db.QueryContext(ctx,
		`SELECT instance_id, definition_id, definition_version, cancellation_requested FROM workflow_instances`)
	if err != nil {
		return fmt.Errorf("storage: RebuildState snapshot definition identity: %w", err)
	}
	var snapErr error
	func() {
		defer func() { _ = identRows.Close() }()
		for identRows.Next() {
			var iid, defID, defVer string
			var cancelFlag int
			if snapErr = identRows.Scan(&iid, &defID, &defVer, &cancelFlag); snapErr != nil {
				return
			}
			snapshot[iid] = defIdentity{
				definitionID:          defID,
				definitionVersion:     defVer,
				cancellationRequested: cancelFlag != 0,
			}
		}
		snapErr = identRows.Err()
	}()
	if snapErr != nil {
		return fmt.Errorf("storage: RebuildState snapshot rows: %w", snapErr)
	}

	// ── Step 1b: snapshot live wait_records (EDR-007 §9 gap) ─────────────────
	// wait_records is durable and is NOT wiped by this rebuild; an instance
	// present here is parked on a live wait. Only used to RESURRECT a
	// non-terminal replayed status to `waiting` — see Step 4's ordering rule.
	waitingInstances := make(map[string]bool)
	waitRows, err := s.db.db.QueryContext(ctx,
		`SELECT DISTINCT instance_id FROM wait_records`)
	if err != nil {
		return fmt.Errorf("storage: RebuildState snapshot wait_records: %w", err)
	}
	var waitErr error
	func() {
		defer func() { _ = waitRows.Close() }()
		for waitRows.Next() {
			var iid string
			if waitErr = waitRows.Scan(&iid); waitErr != nil {
				return
			}
			waitingInstances[iid] = true
		}
		waitErr = waitRows.Err()
	}()
	if waitErr != nil {
		return fmt.Errorf("storage: RebuildState wait_records rows: %w", waitErr)
	}

	// ── Step 2: collect distinct instance_ids from the event log ─────────────
	iidRows, err := s.db.db.QueryContext(ctx,
		`SELECT DISTINCT instance_id FROM execution_events ORDER BY instance_id`)
	if err != nil {
		return fmt.Errorf("storage: RebuildState list instances: %w", err)
	}
	var instanceIDs []string
	var iidErr error
	func() {
		defer func() { _ = iidRows.Close() }()
		for iidRows.Next() {
			var iid string
			if iidErr = iidRows.Scan(&iid); iidErr != nil {
				return
			}
			instanceIDs = append(instanceIDs, iid)
		}
		iidErr = iidRows.Err()
	}()
	if iidErr != nil {
		return fmt.Errorf("storage: RebuildState list instance rows: %w", iidErr)
	}

	// ── Step 3: project each instance from its event stream ───────────────────
	rows := make([]instanceProjection, 0, len(instanceIDs))

	for _, iid := range instanceIDs {
		evRows, err := s.db.db.QueryContext(ctx, `
			SELECT event_type, step_id, payload, emitted_at, namespace
			FROM execution_events
			WHERE instance_id = ?
			ORDER BY sequence_num`,
			iid,
		)
		if err != nil {
			return fmt.Errorf("storage: RebuildState events for %q: %w", iid, err)
		}

		proj, projErr := projectInstance(evRows)
		_ = evRows.Close()
		if projErr != nil {
			return fmt.Errorf("storage: RebuildState project %q: %w", iid, projErr)
		}

		proj.instanceID = iid
		rows = append(rows, proj)
	}

	// ── Step 4: wipe + insert inside ONE transaction ───────────────────────────
	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storage: RebuildState begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM step_claims`); err != nil {
		return fmt.Errorf("storage: RebuildState delete step_claims: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM workflow_instances`); err != nil {
		return fmt.Errorf("storage: RebuildState delete workflow_instances: %w", err)
	}

	for _, r := range rows {
		ident := snapshot[r.instanceID] // empty strings / false when absent

		// Ordering rule (card): a wait record must NOT resurrect an instance
		// the event stream shows as TERMINAL. Terminal status from the
		// EventLog always wins over a stale wait_records row.
		status := r.status
		if !isTerminalStatus(core.InstanceStatus(status)) && waitingInstances[r.instanceID] {
			status = string(core.InstanceStatusWaiting)
		}

		currentSteps := r.currentSteps
		if currentSteps == nil {
			currentSteps = []string{}
		}
		stepsJSON, err := json.Marshal(currentSteps)
		if err != nil {
			return fmt.Errorf("storage: RebuildState marshal steps %q: %w", r.instanceID, err)
		}

		variables := r.variables
		if variables == nil {
			variables = map[string]any{}
		}
		varsJSON, err := json.Marshal(variables)
		if err != nil {
			return fmt.Errorf("storage: RebuildState marshal vars %q: %w", r.instanceID, err)
		}

		startedAt := r.startedAt.UTC().Format(time.RFC3339Nano)
		updatedAt := r.updatedAt.UTC().Format(time.RFC3339Nano)

		var completedAt *string
		if r.completedAt != nil {
			v := r.completedAt.UTC().Format(time.RFC3339Nano)
			completedAt = &v
		}

		cancellationRequested := 0
		if ident.cancellationRequested {
			cancellationRequested = 1
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO workflow_instances
				(instance_id, definition_id, definition_version, namespace, status,
				 current_steps, variables, started_at, updated_at, completed_at,
				 version, cancellation_requested)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)`,
			r.instanceID,
			ident.definitionID,
			ident.definitionVersion,
			r.namespace,
			status,
			string(stepsJSON),
			string(varsJSON),
			startedAt,
			updatedAt,
			completedAt,
			cancellationRequested,
		); err != nil {
			return fmt.Errorf("storage: RebuildState insert %q: %w", r.instanceID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("storage: RebuildState commit: %w", err)
	}
	return nil
}

// instanceProjection is the mutable projection state built while replaying
// one instance's event stream.
type instanceProjection struct {
	instanceID   string
	namespace    string
	status       string
	currentSteps []string
	variables    map[string]any
	startedAt    time.Time
	updatedAt    time.Time
	completedAt  *time.Time
}

// projectInstance replays event rows (already ordered by sequence_num) and
// returns the final projected state. The caller must Close rows after return.
func projectInstance(rows *sql.Rows) (instanceProjection, error) {
	var p instanceProjection
	p.status = string(core.InstanceStatusRunning) // zero-value default
	p.currentSteps = []string{}
	p.variables = map[string]any{}

	started := false

	for rows.Next() {
		var (
			eventType    string
			stepIDNull   sql.NullString
			payloadStr   string
			emittedAtStr string
			namespace    string
		)
		if err := rows.Scan(&eventType, &stepIDNull, &payloadStr, &emittedAtStr, &namespace); err != nil {
			return instanceProjection{}, fmt.Errorf("scan event row: %w", err)
		}

		emittedAt, err := parseTimeStr(emittedAtStr)
		if err != nil {
			return instanceProjection{}, fmt.Errorf("parse emitted_at %q: %w", emittedAtStr, err)
		}

		// First event seeds namespace (defensive rule: no WorkflowStarted).
		if !started {
			p.namespace = namespace
		}

		var payload map[string]json.RawMessage
		if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
			return instanceProjection{}, fmt.Errorf("unmarshal payload for %q: %w", eventType, err)
		}

		stepID := stepIDNull.String

		switch core.EventType(eventType) {

		case core.EventTypeWorkflowStarted:
			// status running, StartedAt=emitted_at, variables={"inputs": payload.inputs}
			p.namespace = namespace
			p.status = string(core.InstanceStatusRunning)
			p.startedAt = emittedAt
			p.currentSteps = []string{}
			var inputs any
			if raw, ok := payload["inputs"]; ok {
				if err := json.Unmarshal(raw, &inputs); err != nil {
					return instanceProjection{}, fmt.Errorf("unmarshal WorkflowStarted.inputs: %w", err)
				}
			}
			p.variables = map[string]any{"inputs": inputs}
			started = true

		case core.EventTypeStepStarted:
			// current_steps += envelope step_id
			pid := stepID
			if pid == "" {
				// fallback: read step_id from payload
				if err := unmarshalField(payload, "step_id", &pid); err != nil {
					return instanceProjection{}, fmt.Errorf("StepStarted.step_id: %w", err)
				}
			}
			if pid != "" {
				p.currentSteps = appendUniq(p.currentSteps, pid)
			}

		case core.EventTypeStepCompleted:
			// current_steps -= step_id; variables[step_id] = payload.outputs
			pid := stepID
			if pid == "" {
				if err := unmarshalField(payload, "step_id", &pid); err != nil {
					return instanceProjection{}, fmt.Errorf("StepCompleted.step_id: %w", err)
				}
			}
			if pid != "" {
				p.currentSteps = removeStep(p.currentSteps, pid)
				var outputs any
				if err := unmarshalField(payload, "outputs", &outputs); err != nil {
					return instanceProjection{}, fmt.Errorf("StepCompleted.outputs: %w", err)
				}
				p.variables[pid] = outputs
			}

		case core.EventTypeStepFailed:
			// retrying:true → keep in current_steps; retrying:false → remove
			var retrying bool
			if err := unmarshalField(payload, "retrying", &retrying); err != nil {
				return instanceProjection{}, fmt.Errorf("StepFailed.retrying: %w", err)
			}
			if !retrying {
				pid := stepID
				if pid == "" {
					if err := unmarshalField(payload, "step_id", &pid); err != nil {
						return instanceProjection{}, fmt.Errorf("StepFailed.step_id: %w", err)
					}
				}
				if pid != "" {
					p.currentSteps = removeStep(p.currentSteps, pid)
				}
			}

		case core.EventTypeStepFallbackActivated:
			// remove payload.step_id from current_steps; record a sentinel in
			// variables so convergent join gates treat the fallback step as done.
			var pid string
			if err := unmarshalField(payload, "step_id", &pid); err != nil {
				return instanceProjection{}, fmt.Errorf("StepFallbackActivated.step_id: %w", err)
			}
			if pid != "" {
				p.currentSteps = removeStep(p.currentSteps, pid)
				if _, exists := p.variables[pid]; !exists {
					p.variables[pid] = map[string]any{}
				}
			}

		case core.EventTypeSignalReceived:
			// waiting → running
			p.status = string(core.InstanceStatusRunning)

		case core.EventTypeWorkflowCompleted:
			p.status = string(core.InstanceStatusCompleted)
			p.currentSteps = []string{}
			p.completedAt = &emittedAt

		case core.EventTypeWorkflowFailed:
			p.status = string(core.InstanceStatusFailed)
			p.completedAt = &emittedAt

		case core.EventTypeWorkflowCancelled:
			p.status = string(core.InstanceStatusCancelled)
			p.completedAt = &emittedAt

		case core.EventTypeWorkflowCompensating:
			p.status = string(core.InstanceStatusCompensating)

		case core.EventTypeWorkflowCompensated:
			p.status = string(core.InstanceStatusCompensated)
			p.completedAt = &emittedAt

		case core.EventTypeWorkflowCompensationFailed:
			p.status = string(core.InstanceStatusCompensationFailed)
			p.completedAt = &emittedAt
		}

		// UpdatedAt = last event's emitted_at
		p.updatedAt = emittedAt
	}

	if err := rows.Err(); err != nil {
		return instanceProjection{}, fmt.Errorf("rows error: %w", err)
	}

	return p, nil
}

// unmarshalField decodes payload[key] into dst when the key is present; a
// malformed value is an error (frozen-format payloads must never be silently
// projected as missing — TDS-01 §2). An absent key leaves dst untouched.
func unmarshalField(payload map[string]json.RawMessage, key string, dst any) error {
	raw, ok := payload[key]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("field %q: %w", key, err)
	}
	return nil
}

// appendUniq appends s to slice only if not already present.
func appendUniq(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// removeStep removes the first occurrence of s from slice.
func removeStep(slice []string, s string) []string {
	for i, v := range slice {
		if v == s {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
