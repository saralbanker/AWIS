package main

// audit.go — 'awis audit' command (TDS-07 §3 M17; M17-C1).
//
// audit [--limit N]
// Reads audit_log table via RecallStore.ListAudit.

import (
	"context"
	"fmt"
	"time"

	"github.com/awis/awis/internal/storage"
)

func init() {
	commands["audit"] = command{fn: runAudit, summary: "View audit log entries"}
}

// auditEntryJSON is one audit log entry in JSON output.
type auditEntryJSON struct {
	ID             int64  `json:"id"`
	Timestamp      string `json:"timestamp"`
	EventType      string `json:"event_type"`
	Actor          string `json:"actor"`
	PayloadSummary string `json:"payload_summary"`
}

// auditOutputJSON is the JSON schema for 'awis audit --json'.
type auditOutputJSON struct {
	Entries []auditEntryJSON `json:"entries"`
}

func runAudit(args []string) {
	fs := newFlagSet("audit")
	var limit int
	fs.IntVar(&limit, "limit", 50, "Maximum number of audit entries (default 50)")
	mustParse(fs, args)

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("audit: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	rs, ok := store.(storage.RecallStore)
	if !ok {
		fail(1, "audit: storage does not support RecallStore", globalDataDir+"/runtime.db", "ensure storage migration is current")
	}

	ctx := context.Background()
	rows, err := rs.ListAudit(ctx, limit)
	if err != nil {
		fail(1, fmt.Sprintf("audit: cannot read audit log: %s", err), globalDataDir+"/runtime.db", "check storage integrity")
	}

	entries := make([]auditEntryJSON, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, auditEntryJSON{
			ID:             r.ID,
			Timestamp:      r.Timestamp.UTC().Format(time.RFC3339),
			EventType:      r.EventType,
			Actor:          r.Actor,
			PayloadSummary: r.PayloadSummary,
		})
	}

	if globalJSON {
		out := auditOutputJSON{Entries: entries}
		emitJSON(out)
		return
	}

	// Human output.
	if len(entries) == 0 {
		fmt.Println("No audit log entries found.")
		fmt.Println()
		fmt.Println("  What: audit_log is empty")
		fmt.Println("  What now: awis start  (registers workflows, writes audit rows)")
		return
	}

	fmt.Printf("AUDIT LOG (last %d)\n\n", len(entries))
	fmt.Printf("  %-4s %-28s %-24s %-10s %s\n", "ID", "TIMESTAMP", "EVENT TYPE", "ACTOR", "SUMMARY")
	for _, e := range entries {
		summary := e.PayloadSummary
		if len(summary) > 40 {
			summary = summary[:40] + "..."
		}
		fmt.Printf("  %-4d %-28s %-24s %-10s %s\n",
			e.ID, e.Timestamp, e.EventType, e.Actor, summary)
	}
}
