package main

// history.go — 'awis history' command (TDS-07 §3 M17; M17-C1).
//
// history [--workflow=<id>] [--n=20] [--namespace=<ns>] [--status=failed|completed]
// Lists completed/failed/cancelled instances with durations.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/awis/awis/internal/core"
)

func init() {
	commands["history"] = command{fn: runHistory, summary: "Recent completed instances"}
}

// historyEntryJSON is one entry in the TDS-07 history JSON output.
type historyEntryJSON struct {
	InstanceID  string  `json:"instance_id"`
	WorkflowID  string  `json:"workflow_id"`
	Namespace   string  `json:"namespace"`
	Status      string  `json:"status"`
	DurationMs  int     `json:"duration_ms"`
	FailedStep  *string `json:"failed_step"`
	CompletedAt string  `json:"completed_at"`
}

// historyOutputJSON is the full history JSON schema.
type historyOutputJSON struct {
	Instances []historyEntryJSON `json:"instances"`
}

func runHistory(args []string) {
	fs := newFlagSet("history")
	var workflowID string
	var n int
	var ns string
	var statusFilter string
	fs.StringVar(&workflowID, "workflow", "", "Filter by workflow ID")
	fs.IntVar(&n, "n", 20, "Number of instances to return (default 20)")
	fs.StringVar(&ns, "namespace", "", "Filter by namespace")
	fs.StringVar(&statusFilter, "status", "", "Filter by status: failed|completed")
	mustParse(fs, args)

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("history: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	ctx := context.Background()
	if ns == "" {
		ns = globalNamespace
	}

	// Collect terminal instances from multiple statuses.
	terminalStatuses := []core.InstanceStatus{
		core.InstanceStatusCompleted,
		core.InstanceStatusFailed,
		core.InstanceStatusCancelled,
		core.InstanceStatusCompensated,
		core.InstanceStatusCompensationFailed,
	}

	// If a status filter is provided, restrict to that status.
	if statusFilter != "" {
		terminalStatuses = []core.InstanceStatus{core.InstanceStatus(statusFilter)}
	}

	var all []core.WorkflowInstance
	for _, st := range terminalStatuses {
		filter := core.InstanceFilter{Status: st}
		if ns != "default" {
			filter.Namespace = ns
		}
		insts, err := store.ListInstances(ctx, filter)
		if err != nil {
			continue
		}
		// Filter by workflow ID if specified.
		for _, inst := range insts {
			if workflowID != "" && inst.DefinitionID != workflowID {
				continue
			}
			all = append(all, inst)
		}
	}

	// Sort descending by UpdatedAt (most recent first).
	sortByUpdatedAt(all)
	if len(all) > n {
		all = all[:n]
	}

	entries := make([]historyEntryJSON, 0, len(all))
	for _, inst := range all {
		var durMs int
		var completedAt string
		if inst.CompletedAt != nil {
			durMs = int(inst.CompletedAt.Sub(inst.StartedAt).Milliseconds())
			completedAt = inst.CompletedAt.UTC().Format(time.RFC3339)
		} else {
			completedAt = inst.UpdatedAt.UTC().Format(time.RFC3339)
		}
		entries = append(entries, historyEntryJSON{
			InstanceID:  string(inst.InstanceID),
			WorkflowID:  inst.DefinitionID,
			Namespace:   inst.Namespace,
			Status:      string(inst.Status),
			DurationMs:  durMs,
			CompletedAt: completedAt,
		})
	}

	if globalJSON {
		out := historyOutputJSON{Instances: entries}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		return
	}

	// Human output.
	if len(entries) == 0 {
		fmt.Println("No completed instances found.")
		fmt.Println()
		fmt.Println("  What: no completed or failed workflow instances in storage")
		fmt.Println("  What now: awis submit <workflow-id>")
		return
	}
	fmt.Printf("HISTORY (last %d)\n\n", len(entries))
	fmt.Printf("  %-12s %-20s %-12s %-10s %-10s %s\n", "INSTANCE", "WORKFLOW", "NAMESPACE", "STATUS", "DURATION", "COMPLETED AT")
	for _, e := range entries {
		sym := historySymbol(e.Status)
		dur := formatDurationMs(e.DurationMs)
		fmt.Printf("  %s %-11s %-20s %-12s %-10s %-10s %s\n",
			sym, shortInstanceID(e.InstanceID), e.WorkflowID, e.Namespace, e.Status, dur, e.CompletedAt)
	}
}

func historySymbol(status string) string {
	switch status {
	case "completed", "compensated":
		return "✓"
	case "failed", "compensation_failed":
		return "✗"
	case "cancelled":
		return "✗"
	default:
		return "·"
	}
}

func shortInstanceID(id string) string {
	if len(id) > 10 {
		return id[:10]
	}
	return id
}

func formatDurationMs(ms int) string {
	if ms == 0 {
		return "0ms"
	}
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
