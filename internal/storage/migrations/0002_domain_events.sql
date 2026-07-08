CREATE TABLE domain_events (
  event_id     TEXT PRIMARY KEY,
  namespace    TEXT NOT NULL,
  event_type   TEXT NOT NULL,
  source       TEXT NOT NULL,
  payload      TEXT NOT NULL,  -- JSON
  emitted_at   TEXT NOT NULL,
  consumed_at  TEXT
);
CREATE INDEX idx_domain_events_ns_type_time ON domain_events(namespace, event_type, emitted_at);
