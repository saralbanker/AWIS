// Package record implements the OIP Record writer and oip.db access layer.
//
// It writes TDS-06-formatted decision entries to .decisions/entries/<id>.md,
// computes per-day NNN counters from existing files, enforces the append-only
// invariant (never overwrites; collision advances NNN), and inserts into the
// entries + entries_fts tables of oip.db (located at
// .decisions/.index/oip.db relative to the configured record root).
//
// This package is OIP's own internal package and does NOT import any package
// under github.com/awis/awis/internal (platform boundary; M15 QG-4).
package record
