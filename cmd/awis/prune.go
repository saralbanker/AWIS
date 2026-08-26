package main

// prune.go — 'awis prune-events' command (TDS-07 §3 M17; M17-C1).
//
// prune-events --dry-run [--before=<date>]
// TTL scan report (domain_events 7d TTL per F-2). ONLY dry-run in V1;
// destructive prune is post-V1 (PRD Should-Have).

import (
	"context"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

func init() {
	commands["prune-events"] = command{fn: runPruneEvents, summary: "Prune EventLog (dry-run only in V1)"}
}

// pruneReportJSON is the JSON schema for 'awis prune-events --json'.
type pruneReportJSON struct {
	DryRun        bool   `json:"dry_run"`
	Before        string `json:"before"`
	EligibleCount int    `json:"eligible_count"`
	TotalScanned  int    `json:"total_scanned"`
	Note          string `json:"note"`
	ScannedAt     string `json:"scanned_at"`
}

func runPruneEvents(args []string) {
	fs := newFlagSet("prune-events")
	var dryRun bool
	var before string
	fs.BoolVar(&dryRun, "dry-run", false, "Report what would be pruned (required in V1)")
	fs.StringVar(&before, "before", "", "Prune events older than this date (RFC3339 or YYYY-MM-DD)")
	mustParse(fs, args)

	if !dryRun {
		fail(2,
			"prune-events requires --dry-run in V1",
			"",
			"awis prune-events --dry-run  (destructive prune is post-V1)")
	}

	// Determine cutoff: default = 7 days ago (F-2 TTL).
	var cutoff time.Time
	if before != "" {
		t, err := time.Parse(time.RFC3339, before)
		if err != nil {
			// Try date-only.
			t, err = time.Parse("2006-01-02", before)
			if err != nil {
				fail(2, fmt.Sprintf("prune-events: cannot parse --before %q", before), "", "use RFC3339 or YYYY-MM-DD format")
			}
		}
		cutoff = t
	} else {
		cutoff = time.Now().UTC().Add(-7 * 24 * time.Hour)
	}

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("prune-events: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	// Scan all instances and count events older than cutoff.
	ctx := context.Background()
	allStatuses := []core.InstanceStatus{
		core.InstanceStatusCompleted,
		core.InstanceStatusFailed,
		core.InstanceStatusCancelled,
		core.InstanceStatusCompensated,
		core.InstanceStatusCompensationFailed,
	}

	var totalScanned int
	var eligible int

	for _, st := range allStatuses {
		filter := core.InstanceFilter{Status: st}
		insts, err := store.ListInstances(ctx, filter)
		if err != nil {
			continue
		}
		for _, inst := range insts {
			events, err := store.ReadEvents(ctx, inst.InstanceID, 0)
			if err != nil {
				continue
			}
			for _, ev := range events {
				totalScanned++
				if ev.EmittedAt.Before(cutoff) {
					eligible++
				}
			}
		}
	}

	report := pruneReportJSON{
		DryRun:        true,
		Before:        cutoff.UTC().Format(time.RFC3339),
		EligibleCount: eligible,
		TotalScanned:  totalScanned,
		Note:          "Destructive prune is post-V1. This is a dry-run report only.",
		ScannedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	if globalJSON {
		emitJSON(report)
		return
	}

	// Human output.
	fmt.Println("PRUNE-EVENTS (dry-run)")
	fmt.Println()
	fmt.Printf("  Cutoff:         %s\n", report.Before)
	fmt.Printf("  Events scanned: %d\n", report.TotalScanned)
	fmt.Printf("  Eligible:       %d  (would be pruned)\n", report.EligibleCount)
	fmt.Println()
	fmt.Println("  NOTE: Destructive prune is post-V1. No data was deleted.")
	fmt.Println("        Re-run without --dry-run when the feature ships.")
}
