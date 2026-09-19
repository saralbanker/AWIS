-- Migration 0008: step_claims lease (D-6).
-- A claim committed by ClaimStep's own transaction had no release path other
-- than terminal-status UpsertInstance or a RebuildState wipe; if the
-- StepStarted emission that should follow a successful claim failed, the
-- claim was orphaned forever. expires_at bounds an orphan's lifetime so
-- ClaimStep can reclaim it — but ONLY together with the absence of a
-- StepStarted event for the same (instance, step); see ClaimStep (sqlite.go).
-- Column typing mirrors claimed_at (TEXT, RFC3339Nano via the injectable
-- clock) and step_results_cache.expires_at's existing convention.
ALTER TABLE step_claims ADD COLUMN expires_at TEXT;
