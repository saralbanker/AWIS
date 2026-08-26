package main

// metrics.go — 'awis metrics' command (TDS-07 §3 M17; M17-C1; PRD §20).
//
// metrics [--namespace=<ns>] [--workflow=<id>]
// Aggregates counts and durations from workflow_instances: instances by status,
// average/min/max duration for completed instances.

import (
	"context"
	"fmt"
	"time"

	"github.com/awis/awis/internal/core"
)

func init() {
	commands["metrics"] = command{fn: runMetrics, summary: "Aggregate execution statistics"}
}

// metricsOutputJSON is the JSON schema for 'awis metrics --json'.
type metricsOutputJSON struct {
	Namespace         string                    `json:"namespace"`
	WorkflowFilter    string                    `json:"workflow_filter,omitempty"`
	ByStatus          map[string]int            `json:"by_status"`
	TotalInstances    int                       `json:"total_instances"`
	CompletedMetrics  *completedMetricsJSON     `json:"completed_metrics,omitempty"`
	WorkflowBreakdown []workflowMetricEntryJSON `json:"workflow_breakdown"`
	GeneratedAt       string                    `json:"generated_at"`
}

// completedMetricsJSON holds duration statistics for completed instances.
type completedMetricsJSON struct {
	Count int `json:"count"`
	AvgMs int `json:"avg_ms"`
	MinMs int `json:"min_ms"`
	MaxMs int `json:"max_ms"`
}

// workflowMetricEntryJSON is per-workflow breakdown.
type workflowMetricEntryJSON struct {
	WorkflowID string `json:"workflow_id"`
	Total      int    `json:"total"`
	Completed  int    `json:"completed"`
	Failed     int    `json:"failed"`
}

func runMetrics(args []string) {
	fs := newFlagSet("metrics")
	var ns string
	var workflowID string
	fs.StringVar(&ns, "namespace", "", "Filter by namespace")
	fs.StringVar(&workflowID, "workflow", "", "Filter by workflow ID")
	mustParse(fs, args)

	if ns == "" {
		ns = globalNamespace
	}

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("metrics: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	ctx := context.Background()
	now := time.Now()

	allStatuses := []core.InstanceStatus{
		core.InstanceStatusPending,
		core.InstanceStatusRunning,
		core.InstanceStatusWaiting,
		core.InstanceStatusCompensating,
		core.InstanceStatusCompleted,
		core.InstanceStatusFailed,
		core.InstanceStatusCancelled,
		core.InstanceStatusCompensated,
		core.InstanceStatusCompensationFailed,
	}

	byStatus := make(map[string]int)
	var allInstances []core.WorkflowInstance

	for _, st := range allStatuses {
		filter := core.InstanceFilter{Status: st}
		// Only apply namespace filter if not "default" (default = all namespaces in metrics).
		if ns != "" && ns != "default" {
			filter.Namespace = ns
		}
		insts, err := store.ListInstances(ctx, filter)
		if err != nil {
			continue
		}
		for _, inst := range insts {
			if workflowID != "" && inst.DefinitionID != workflowID {
				continue
			}
			byStatus[string(inst.Status)]++
			allInstances = append(allInstances, inst)
		}
	}

	// Compute duration stats for completed instances.
	var completedMetrics *completedMetricsJSON
	var completedDurations []int
	for _, inst := range allInstances {
		if inst.Status == core.InstanceStatusCompleted && inst.CompletedAt != nil {
			ms := int(inst.CompletedAt.Sub(inst.StartedAt).Milliseconds())
			completedDurations = append(completedDurations, ms)
		}
	}
	if len(completedDurations) > 0 {
		sum, minV, maxV := 0, completedDurations[0], completedDurations[0]
		for _, d := range completedDurations {
			sum += d
			if d < minV {
				minV = d
			}
			if d > maxV {
				maxV = d
			}
		}
		completedMetrics = &completedMetricsJSON{
			Count: len(completedDurations),
			AvgMs: sum / len(completedDurations),
			MinMs: minV,
			MaxMs: maxV,
		}
	}

	// Per-workflow breakdown.
	wfCounts := make(map[string]*workflowMetricEntryJSON)
	for _, inst := range allInstances {
		e, ok := wfCounts[inst.DefinitionID]
		if !ok {
			e = &workflowMetricEntryJSON{WorkflowID: inst.DefinitionID}
			wfCounts[inst.DefinitionID] = e
		}
		e.Total++
		switch inst.Status {
		case core.InstanceStatusCompleted:
			e.Completed++
		case core.InstanceStatusFailed, core.InstanceStatusCompensationFailed:
			e.Failed++
		}
	}
	breakdown := make([]workflowMetricEntryJSON, 0, len(wfCounts))
	for _, e := range wfCounts {
		breakdown = append(breakdown, *e)
	}

	out := metricsOutputJSON{
		Namespace:         ns,
		WorkflowFilter:    workflowID,
		ByStatus:          byStatus,
		TotalInstances:    len(allInstances),
		CompletedMetrics:  completedMetrics,
		WorkflowBreakdown: breakdown,
		GeneratedAt:       now.UTC().Format(time.RFC3339),
	}

	if globalJSON {
		emitJSON(out)
		return
	}

	// Human output.
	fmt.Printf("METRICS  %s\n\n", now.Format("2006-01-02 15:04:05"))
	if ns != "" && ns != "default" {
		fmt.Printf("Namespace:  %s\n", ns)
	}
	if workflowID != "" {
		fmt.Printf("Workflow:   %s\n", workflowID)
	}
	fmt.Println()
	fmt.Printf("Total instances: %d\n\n", out.TotalInstances)

	if out.TotalInstances == 0 {
		fmt.Println("  No instances found.")
		fmt.Println("  What now: awis submit <workflow-id>")
		return
	}

	fmt.Println("By status:")
	statusOrder := []string{"running", "waiting", "pending", "completed", "failed", "cancelled", "compensating", "compensated", "compensation_failed"}
	for _, st := range statusOrder {
		if c, ok := byStatus[st]; ok && c > 0 {
			fmt.Printf("  %-22s %d\n", st, c)
		}
	}
	fmt.Println()

	if completedMetrics != nil {
		fmt.Println("Completed duration:")
		fmt.Printf("  Count: %d  Avg: %s  Min: %s  Max: %s\n",
			completedMetrics.Count,
			formatDurationMs(completedMetrics.AvgMs),
			formatDurationMs(completedMetrics.MinMs),
			formatDurationMs(completedMetrics.MaxMs),
		)
		fmt.Println()
	}

	if len(breakdown) > 0 {
		fmt.Println("By workflow:")
		fmt.Printf("  %-24s %-8s %-10s %-8s\n", "WORKFLOW", "TOTAL", "COMPLETED", "FAILED")
		for _, e := range breakdown {
			fmt.Printf("  %-24s %-8d %-10d %-8d\n", e.WorkflowID, e.Total, e.Completed, e.Failed)
		}
	}
}
