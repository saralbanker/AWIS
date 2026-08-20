-- Plugin registry tables (Blueprint §11 "Plugin Registry" SQL verbatim).
-- Migration number: 0005 (F-4 renumber: IMP §14 originally named this
-- 0003_plugins.sql; signals/audit landed at 0003/0004 in M07, so plugins
-- registry is 0005). See TRACEABILITY.md disposition row T4.
CREATE TABLE plugins (
  plugin_id    TEXT PRIMARY KEY,
  name         TEXT NOT NULL,
  version      TEXT NOT NULL,
  manifest     TEXT NOT NULL,    -- JSON
  status       TEXT NOT NULL,    -- registered | active | suspended | failed
  registered_at TEXT NOT NULL
);
CREATE TABLE plugin_capabilities (
  plugin_id    TEXT NOT NULL REFERENCES plugins(plugin_id),
  capability_id TEXT NOT NULL,
  PRIMARY KEY (plugin_id, capability_id)
);
