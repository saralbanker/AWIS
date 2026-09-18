-- FTS5 virtual table over execution_events payloads (IMP §14; CONTRA-4 additive;
-- StoragePort untouched). Platform-owned, inside runtime.db.
-- Migration number: 0006 (F-4 sequence; see M17 IMPLEMENTATION_SPEC §CE pins).
--
-- Non-content external table: does not store raw content, joins back to
-- execution_events for emitted_at. No trigger — the index is rebuilt via
-- INSERT INTO ... SELECT on each SearchEvents call (lazy sync; V1 search is
-- developer-facing, not latency-sensitive; avoids per-insert trigger cost that
-- is prohibitive at 10K-100K event scale under race detector).
CREATE VIRTUAL TABLE IF NOT EXISTS execution_events_fts
USING fts5(
    event_id,
    instance_id,
    namespace,
    event_type,
    step_id,
    payload
);
