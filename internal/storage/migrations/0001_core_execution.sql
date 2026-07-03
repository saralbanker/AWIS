CREATE TABLE execution_events (
  event_id          TEXT PRIMARY KEY,
  instance_id       TEXT NOT NULL,
  namespace         TEXT NOT NULL,
  event_type        TEXT NOT NULL,
  step_id           TEXT,
  payload           TEXT NOT NULL,  -- JSON
  emitted_at        TEXT NOT NULL,
  sequence_num      INTEGER NOT NULL,
  schema_version    INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE workflow_definitions (
  id                TEXT NOT NULL,
  version           TEXT NOT NULL,
  namespace         TEXT NOT NULL,
  definition        TEXT NOT NULL,  -- JSON serialized WorkflowDefinition
  registered_at     TEXT NOT NULL,
  PRIMARY KEY (id, version)
);
CREATE TABLE step_results_cache (
  idempotency_key   TEXT PRIMARY KEY,  -- hash(instance_id + step_id + attempt)
  result            TEXT NOT NULL,     -- JSON
  cached_at         TEXT NOT NULL,
  expires_at        TEXT NOT NULL
);
CREATE INDEX idx_events_instance_seq ON execution_events(instance_id, sequence_num);
CREATE INDEX idx_events_ns_time ON execution_events(namespace, emitted_at);
