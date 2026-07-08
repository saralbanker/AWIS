// Package engine is the AWIS pull-based execution engine (Blueprint §8; IMP
// §27.M6). It drives registered WorkflowDefinitions to completion by replaying a
// fixed tick pipeline against the StoragePort; the EventLog is the source of
// truth and the workflow_instances projection is derived from it (EDR-007).
//
// # Tick pipeline (Blueprint §8 order)
//
// Each Tick runs these stages:
//
//	SCAN_RUNNABLE     ListInstances(status=running); for each, derive the set of
//	                  activatable steps (join gate, see transition.go).
//	SCAN_TRIGGERABLE  Domain-event → workflow trigger matching. STUB at C1;
//	                  filled by C3 (T5, F-2).
//	CLAIM             storage.ClaimStep gives at-most-once ownership of a step
//	                  (EDR-006). A worker that loses the claim skips silently.
//	DISPATCH          Claimed steps run through their Runner concurrently, bounded
//	                  by MaxParallelSteps. Handler execution is the only concurrent
//	                  work; all event emission is serialized so sequence numbers
//	                  and event ordering are deterministic (V1 single-process).
//	SETTLE            Emit StepCompleted / StepFailed, then check for workflow
//	                  completion (final step reached, no steps left to run).
//	SIGNAL_SCAN       Signal delivery. STUB at C1; filled by M07.
//
// # Event emission and projection
//
// The engine assigns sequence_num (EDR-005): a per-instance counter guarded by a
// mutex, reconciled against the storage monotonicity guard (on
// storage.ErrSequenceViolation the counter is reloaded from MAX+1 and the append
// retried once). Every emit appends the event and THEN applies the same
// forward projection RebuildState applies on replay (EDR-007 §2 status map),
// upserting the workflow_instances row under optimistic concurrency control.
// applyEventToInstance mirrors RebuildState exactly: the M06 event-sequence
// fixtures prove forward-path ≡ rebuild equivalence.
//
// # Scope (C1)
//
// C1 implements the tick pipeline, Submit, event emission, forward projection,
// the NativeRunner dispatch path, and transition fan-out/join (T1 + T2 native +
// T3). Retry, fallback, compensation, and cancellation (T4), triggers (T5), and
// the E1 gate (T6) are C2/C3; the runner map and the two scan stubs are the
// seams where they slot in. applyEventToInstance already handles all 12 event
// types so C2/C3 need only add emission sites.
package engine
