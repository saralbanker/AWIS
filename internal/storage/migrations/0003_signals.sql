CREATE TABLE signal_inbox (signal_id TEXT PRIMARY KEY, instance_id TEXT NOT NULL,
    signal_name TEXT NOT NULL, payload TEXT NOT NULL, delivered_at TEXT, received_at TEXT NOT NULL);
CREATE TABLE wait_records (instance_id TEXT NOT NULL, step_id TEXT NOT NULL,
    signal_name TEXT NOT NULL, created_at TEXT NOT NULL, timeout_at TEXT,
    timeout_action TEXT NOT NULL, PRIMARY KEY (instance_id, step_id));
CREATE INDEX idx_wait_timeout ON wait_records(timeout_at) WHERE timeout_at IS NOT NULL;
CREATE INDEX idx_signals_instance ON signal_inbox(instance_id, signal_name);
