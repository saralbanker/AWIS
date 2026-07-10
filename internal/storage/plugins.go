package storage

// Plugin registry storage accessor (M12-C2, T4/T5/T6).
//
// PluginStore is an additive interface type-asserted from a StoragePort
// implementation — the same pattern as AuditLog (audit.go / M07-C1). The
// StoragePort 12-method set is NOT modified (SPEC §2 constraint).
//
// The F-4 audit write site (PluginRegistered) is written in the same
// transaction as the plugin upsert (RegisterPlugin), following the
// transactional pattern established in sqlite.go (UpsertInstance tx block).
//
// TRACEABILITY: rows T4 (0005_plugins.sql), T5 (PluginStore + SQLite impl),
// T6 (PluginRegistered audit write site).

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrPluginNotFound is returned by GetPlugin and LookupCapability when no
// matching row exists.
var ErrPluginNotFound = errors.New("storage: plugin not found")

// ErrCapabilityNotFound is returned by LookupCapability when no matching
// capability row exists.
var ErrCapabilityNotFound = errors.New("storage: capability not found")

// PluginRow is the projection returned by GetPlugin and ListPlugins.
type PluginRow struct {
	PluginID     string
	Name         string
	Version      string
	Manifest     string // raw JSON
	Status       string
	RegisteredAt time.Time
}

// PluginStore is the additive storage interface for the plugin registry
// (Blueprint §11 tables: plugins, plugin_capabilities). It is type-asserted
// from the storage implementation — it is NOT part of core.StoragePort.
type PluginStore interface {
	// RegisterPlugin upserts the plugin row (status='registered',
	// registered_at=clock) and replaces all capability rows for the plugin.
	// capabilityIDs lists every capability id declared by the plugin.
	// The PluginRegistered audit row (F-4) is written in the same transaction.
	RegisterPlugin(ctx context.Context, name, version, manifestJSON string, capabilityIDs []string) error

	// GetPlugin returns the plugin row for the given name.
	// Returns ErrPluginNotFound when absent.
	GetPlugin(ctx context.Context, name string) (PluginRow, error)

	// ListPlugins returns all plugin rows ordered by name.
	ListPlugins(ctx context.Context) ([]PluginRow, error)

	// SetPluginStatus updates the status column for the named plugin.
	// Returns ErrPluginNotFound when absent.
	SetPluginStatus(ctx context.Context, name, status string) error

	// LookupCapability returns the plugin name that owns capabilityID.
	// Returns ErrCapabilityNotFound when absent.
	LookupCapability(ctx context.Context, capabilityID string) (string, error)
}

// ── SQLiteStorage implementation ─────────────────────────────────────────────

// RegisterPlugin upserts the plugins row and replaces all plugin_capabilities
// rows for the plugin, then writes a PluginRegistered audit entry — all in one
// transaction (F-4 write site; TRACEABILITY T6).
//
// plugin_id equals manifest name (V1 one-installed-version-per-name pin;
// TRACEABILITY disposition).
func (s *SQLiteStorage) RegisterPlugin(ctx context.Context, name, version, manifestJSON string, capabilityIDs []string) error {
	now := s.now().UTC().Format(time.RFC3339Nano)

	tx, err := s.db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("storage: RegisterPlugin begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Upsert the plugin row (plugin_id = name; V1 pin).
	_, err = tx.ExecContext(ctx, `
		INSERT INTO plugins (plugin_id, name, version, manifest, status, registered_at)
		VALUES (?, ?, ?, ?, 'registered', ?)
		ON CONFLICT(plugin_id) DO UPDATE SET
			name          = excluded.name,
			version       = excluded.version,
			manifest      = excluded.manifest,
			status        = 'registered',
			registered_at = excluded.registered_at`,
		name, name, version, manifestJSON, now,
	)
	if err != nil {
		return fmt.Errorf("storage: RegisterPlugin upsert plugins: %w", err)
	}

	// Replace all capability rows: delete existing, insert new.
	if _, err = tx.ExecContext(ctx,
		`DELETE FROM plugin_capabilities WHERE plugin_id = ?`, name,
	); err != nil {
		return fmt.Errorf("storage: RegisterPlugin delete capabilities: %w", err)
	}
	for _, capID := range capabilityIDs {
		if _, err = tx.ExecContext(ctx,
			`INSERT INTO plugin_capabilities (plugin_id, capability_id) VALUES (?, ?)`,
			name, capID,
		); err != nil {
			return fmt.Errorf("storage: RegisterPlugin insert capability %q: %w", capID, err)
		}
	}

	// Write PluginRegistered audit row in the same transaction (F-4).
	payloadBytes, _ := json.Marshal(map[string]string{"name": name, "version": version})
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO audit_log (timestamp, event_type, actor, payload_summary)
		VALUES (?, 'PluginRegistered', 'system', ?)`,
		now,
		string(payloadBytes),
	); err != nil {
		return fmt.Errorf("storage: RegisterPlugin audit insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("storage: RegisterPlugin commit: %w", err)
	}
	return nil
}

// GetPlugin returns the plugin row for the given name (plugin_id = name).
// Returns ErrPluginNotFound when absent.
func (s *SQLiteStorage) GetPlugin(ctx context.Context, name string) (PluginRow, error) {
	var p PluginRow
	var registeredAtStr string
	err := s.db.db.QueryRowContext(ctx, `
		SELECT plugin_id, name, version, manifest, status, registered_at
		FROM plugins WHERE plugin_id = ?`,
		name,
	).Scan(&p.PluginID, &p.Name, &p.Version, &p.Manifest, &p.Status, &registeredAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PluginRow{}, fmt.Errorf("%w: %s", ErrPluginNotFound, name)
		}
		return PluginRow{}, fmt.Errorf("storage: GetPlugin query: %w", err)
	}
	t, err := parseTimeStr(registeredAtStr)
	if err != nil {
		return PluginRow{}, fmt.Errorf("storage: GetPlugin parse registered_at: %w", err)
	}
	p.RegisteredAt = t
	return p, nil
}

// ListPlugins returns all plugin rows ordered by name.
func (s *SQLiteStorage) ListPlugins(ctx context.Context) ([]PluginRow, error) {
	rows, err := s.db.db.QueryContext(ctx, `
		SELECT plugin_id, name, version, manifest, status, registered_at
		FROM plugins ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("storage: ListPlugins query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var plugins []PluginRow
	for rows.Next() {
		var p PluginRow
		var registeredAtStr string
		if err := rows.Scan(&p.PluginID, &p.Name, &p.Version, &p.Manifest, &p.Status, &registeredAtStr); err != nil {
			return nil, fmt.Errorf("storage: ListPlugins scan: %w", err)
		}
		t, err := parseTimeStr(registeredAtStr)
		if err != nil {
			return nil, fmt.Errorf("storage: ListPlugins parse registered_at: %w", err)
		}
		p.RegisteredAt = t
		plugins = append(plugins, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: ListPlugins rows: %w", err)
	}
	return plugins, nil
}

// SetPluginStatus updates the status column for the named plugin.
// Returns ErrPluginNotFound when absent.
func (s *SQLiteStorage) SetPluginStatus(ctx context.Context, name, status string) error {
	res, err := s.db.db.ExecContext(ctx,
		`UPDATE plugins SET status = ? WHERE plugin_id = ?`,
		status, name,
	)
	if err != nil {
		return fmt.Errorf("storage: SetPluginStatus update: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("storage: SetPluginStatus rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("%w: %s", ErrPluginNotFound, name)
	}
	return nil
}

// LookupCapability returns the plugin name (= plugin_id) that owns capabilityID.
// Returns ErrCapabilityNotFound when absent.
func (s *SQLiteStorage) LookupCapability(ctx context.Context, capabilityID string) (string, error) {
	var pluginID string
	err := s.db.db.QueryRowContext(ctx,
		`SELECT plugin_id FROM plugin_capabilities WHERE capability_id = ?`,
		capabilityID,
	).Scan(&pluginID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%w: %s", ErrCapabilityNotFound, capabilityID)
		}
		return "", fmt.Errorf("storage: LookupCapability query: %w", err)
	}
	return pluginID, nil
}
