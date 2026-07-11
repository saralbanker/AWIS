package main

// recall.go — 'awis recall' command (TDS-07 §3 M17; M17-C1; PRD §21).
//
// recall "<query>" [--synthesize] [--namespace=<ns>]
// FTS over execution_events_fts (migration 0006); --synthesize routes through
// configured intelligence (Null → PRD §21 empty-state message).

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/awis/awis/internal/storage"
)

func init() {
	commands["recall"] = command{fn: runRecall, summary: "Query execution history (FTS)"}
}

// recallEntryJSON is one FTS result entry.
type recallEntryJSON struct {
	EventID    string  `json:"event_id"`
	InstanceID string  `json:"instance_id"`
	Namespace  string  `json:"namespace"`
	EventType  string  `json:"event_type"`
	StepID     string  `json:"step_id,omitempty"`
	Payload    string  `json:"payload"`
	EmittedAt  string  `json:"emitted_at"`
	Rank       float64 `json:"rank"`
}

// recallOutputJSON is the full recall JSON schema.
type recallOutputJSON struct {
	Query      string            `json:"query"`
	Synthesize bool              `json:"synthesize"`
	Results    []recallEntryJSON `json:"results"`
	Synthesis  *string           `json:"synthesis,omitempty"`
}

func runRecall(args []string) {
	fs := newFlagSet("recall")
	var synthesize bool
	var limit int
	fs.BoolVar(&synthesize, "synthesize", false, "Route through configured intelligence for synthesis")
	fs.IntVar(&limit, "limit", 20, "Maximum FTS results (default 20)")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 1 {
		fail(2, "recall requires a <query> argument", "", `awis recall "<query>"`)
	}
	query := rest[0]

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("recall: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	rs, ok := store.(storage.RecallStore)
	if !ok {
		fail(1, "recall: storage does not support RecallStore (FTS index missing)", globalDataDir+"/runtime.db", "ensure migration 0006 has been applied")
	}

	ctx := context.Background()
	rows, err := rs.SearchEvents(ctx, query, limit)
	if err != nil {
		fail(1, fmt.Sprintf("recall: FTS query failed: %s", err), query, "check query syntax (FTS5 match expression)")
	}

	entries := make([]recallEntryJSON, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, recallEntryJSON{
			EventID:    r.EventID,
			InstanceID: r.InstanceID,
			Namespace:  r.Namespace,
			EventType:  r.EventType,
			StepID:     r.StepID,
			Payload:    r.Payload,
			EmittedAt:  r.EmittedAt.UTC().Format(time.RFC3339),
			Rank:       r.Rank,
		})
	}

	// --synthesize: check if intelligence is configured.
	var synthesis *string
	if synthesize {
		// PRD §21 empty-state message when intelligence is not configured.
		// In V1, check ANTHROPIC_API_KEY as the indicator of intelligence configuration.
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			if !globalJSON {
				// PRD §21 verbatim empty-state output.
				fmt.Println("Synthesis requires intelligence configured.")
				fmt.Println("Current: intelligence: null (no API key configured)")
				fmt.Println()
				fmt.Println("Configure: export ANTHROPIC_API_KEY=<key> && awis start")
				fmt.Println("Raw FTS results below:")
				fmt.Println()
			} else {
				msg := "Synthesis requires intelligence configured. Set ANTHROPIC_API_KEY."
				synthesis = &msg
			}
		} else {
			msg := "(synthesis: intelligence configured but synthesis routing not yet implemented in V1 FTS path)"
			synthesis = &msg
		}
	}

	if globalJSON {
		out := recallOutputJSON{
			Query:      query,
			Synthesize: synthesize,
			Results:    entries,
			Synthesis:  synthesis,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		return
	}

	// Human output.
	if len(entries) == 0 {
		fmt.Printf("No results for %q\n\n", query)
		fmt.Println("  What: no FTS matches found in execution_events")
		fmt.Println("  What now: try a broader query term, or 'awis history' to browse instances")
		return
	}

	fmt.Printf("RECALL  %q  (%d results)\n\n", query, len(entries))
	for i, e := range entries {
		fmt.Printf("  [%d] %s  %s  %s\n", i+1, e.InstanceID, e.EventType, e.EmittedAt)
		if e.StepID != "" {
			fmt.Printf("       step: %s\n", e.StepID)
		}
		payload := e.Payload
		if len(payload) > 120 {
			payload = payload[:120] + "..."
		}
		fmt.Printf("       %s\n\n", payload)
	}
}
