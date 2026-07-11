package main

// rebuild.go — 'awis rebuild-state' command (TDS-07 §3 M17; M17-C2).
//
// rebuild-state [<instance-id>] [--all]
// Rebuilds the workflow_instances projection from the append-only EventLog
// (M03 path; EDR-007). Must be run against a stopped runtime.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

func init() {
	commands["rebuild-state"] = command{fn: runRebuildState, summary: "Rebuild workflow_instances projection from EventLog"}
}

// rebuildStateOutputJSON is the JSON schema for 'awis rebuild-state --json'.
type rebuildStateOutputJSON struct {
	Mode    string `json:"mode"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// rebuildStateStore is the additive interface for rebuild (M03 path).
// Type-asserted from *storage.SQLiteStorage — additive per CONTRA-4; StoragePort untouched.
type rebuildStateStore interface {
	RebuildState(ctx context.Context) error
}

func runRebuildState(args []string) {
	fs := newFlagSet("rebuild-state")
	var all bool
	fs.BoolVar(&all, "all", false, "Rebuild ALL instances (required for full projection rebuild)")
	mustParse(fs, args)

	rest := fs.Args()

	// V1: --all is the only supported mode. An <instance-id> positional arg is
	// accepted syntactically but routes to --all semantics (RebuildState rebuilds all
	// instances in the EventLog in one atomic transaction — EDR-007 algorithm).
	if !all && len(rest) == 0 {
		fail(2,
			"rebuild-state requires --all or <instance-id>",
			"",
			"awis rebuild-state --all",
		)
	}

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("rebuild-state: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	// Type-assert to rebuildStateStore (additive; M03 path).
	rs, ok := store.(rebuildStateStore)
	if !ok {
		fail(1,
			"rebuild-state: storage does not support RebuildState",
			globalDataDir+"/runtime.db",
			"ensure storage is up to date (run migrations)",
		)
	}

	ctx := context.Background()

	var mode string
	if all {
		mode = "all"
	} else {
		mode = rest[0] // instance-id: treated as all in V1 (full-projection-only semantics)
	}

	if err := rs.RebuildState(ctx); err != nil {
		fail(1,
			fmt.Sprintf("rebuild-state: projection rebuild failed: %s", err),
			globalDataDir+"/runtime.db",
			"check storage integrity",
		)
	}

	msg := "projection rebuilt from EventLog"

	if globalJSON {
		out := rebuildStateOutputJSON{
			Mode:    mode,
			Success: true,
			Message: msg,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		return
	}

	fmt.Printf("rebuild-state: %s\n", msg)
	fmt.Printf("  Mode:    %s\n", mode)
	fmt.Printf("  DB:      %s\n", globalDataDir+"/runtime.db")
	fmt.Println()
	fmt.Println("  What now: awis start  (resume engine; rebuild is safe before start)")
}
