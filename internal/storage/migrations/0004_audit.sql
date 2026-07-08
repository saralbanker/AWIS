-- audit_log: append-only audit trail, separate from the EventLog and
-- independently auditable (PRD §26; F-4). Records administrative facts —
-- WorkflowRegistered, PluginRegistered, ConfigChanged, SignalDelivered — and is
-- never pruned without explicit operator action (PRD NFR-S-05). Call sites are
-- added by later cards (C2 adds the first); this migration lands the table only.
--
-- Columns realise the PRD §26 render shape (FR-OB-08 / §15): timestamp, event
-- type, actor, payload summary. `id` is a DB-internal monotonic surrogate for a
-- stable append order — like workflow_instances.version it is not a domain ID
-- and is exposed by no evented path. DDL is the E0 for this card, kept within
-- the §26 shape (CONTRA-4: additive, unenumerated, StoragePort-internal): the
-- enumerated event types are NOT constrained by a CHECK so the set stays open.
CREATE TABLE audit_log (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  timestamp       TEXT NOT NULL,
  event_type      TEXT NOT NULL,
  actor           TEXT NOT NULL,
  payload_summary TEXT NOT NULL
);
CREATE INDEX idx_audit_timestamp ON audit_log(timestamp);
