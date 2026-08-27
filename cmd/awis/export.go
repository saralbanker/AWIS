package main

// export.go — 'awis export' command (TDS-07 §3 M17; M17-C1; PRD §25 portability).
//
// export [--out <dir>] [--namespace=<ns>]
// Exports runtime.db events + definitions to JSON files.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/awis/awis/internal/core"
)

func init() {
	commands["export"] = command{fn: runExport, summary: "Export execution history to JSON"}
}

// exportOutputJSON is the JSON schema for 'awis export --json' (summary to stdout).
type exportOutputJSON struct {
	OutDir          string `json:"out_dir"`
	InstancesFile   string `json:"instances_file"`
	DefinitionsFile string `json:"definitions_file"`
	InstanceCount   int    `json:"instance_count"`
	DefinitionCount int    `json:"definition_count"`
	ExportedAt      string `json:"exported_at"`
}

func runExport(args []string) {
	fs := newFlagSet("export")
	var outDir string
	var ns string
	fs.StringVar(&outDir, "out", ".", "Output directory for exported JSON files")
	fs.StringVar(&ns, "namespace", "", "Filter by namespace")
	mustParse(fs, args)

	if ns == "" {
		ns = globalNamespace
	}

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("export: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	ctx := context.Background()
	now := time.Now()

	// Collect all instances.
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

	// One predicate for BOTH halves of the export. Previously the instance
	// half honoured --namespace while the definition half was hardcoded to
	// ListWorkflows(ctx, ""), so `awis export --namespace X` wrote X's
	// instances alongside EVERY namespace's workflow definitions — an export
	// that does not describe one namespace and does not describe all of them
	// (B-26).
	nsPredicate := namespacePredicate(ns)

	var allInstances []core.WorkflowInstance
	for _, st := range allStatuses {
		filter := core.InstanceFilter{Status: st, Namespace: nsPredicate}
		insts, err := store.ListInstances(ctx, filter)
		if err != nil {
			continue
		}
		allInstances = append(allInstances, insts...)
	}

	// Collect workflow definitions under the SAME predicate.
	defs, err := store.ListWorkflows(ctx, nsPredicate)
	if err != nil {
		fail(1, fmt.Sprintf("export: cannot list workflows: %s", err), globalDataDir+"/runtime.db", "check storage integrity")
	}

	// Resolve output directory.
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fail(1, fmt.Sprintf("export: cannot create output directory: %s", err), outDir, "check directory permissions")
	}
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		fail(1, fmt.Sprintf("export: cannot resolve output path: %s", err), outDir, "")
	}

	// Write instances.json.
	instancesFile := filepath.Join(absOut, "instances.json")
	instancesData, err := json.MarshalIndent(allInstances, "", "  ")
	if err != nil {
		fail(1, fmt.Sprintf("export: cannot marshal instances: %s", err), "", "")
	}
	if err := os.WriteFile(instancesFile, instancesData, 0o644); err != nil {
		fail(1, fmt.Sprintf("export: cannot write instances.json: %s", err), instancesFile, "check directory permissions")
	}

	// Write definitions.json.
	defsFile := filepath.Join(absOut, "definitions.json")
	defsData, err := json.MarshalIndent(defs, "", "  ")
	if err != nil {
		fail(1, fmt.Sprintf("export: cannot marshal definitions: %s", err), "", "")
	}
	if err := os.WriteFile(defsFile, defsData, 0o644); err != nil {
		fail(1, fmt.Sprintf("export: cannot write definitions.json: %s", err), defsFile, "check directory permissions")
	}

	summary := exportOutputJSON{
		OutDir:          absOut,
		InstancesFile:   instancesFile,
		DefinitionsFile: defsFile,
		InstanceCount:   len(allInstances),
		DefinitionCount: len(defs),
		ExportedAt:      now.UTC().Format(time.RFC3339),
	}

	if globalJSON {
		emitJSON(summary)
		return
	}

	// Human output.
	fmt.Printf("Export complete  %s\n\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Instances:   %d → %s\n", len(allInstances), instancesFile)
	fmt.Printf("  Definitions: %d → %s\n", len(defs), defsFile)
}

// namespacePredicate maps a CLI namespace argument to a storage predicate,
// where "" means "no predicate — every namespace".
//
// The global --namespace flag defaults to the string "default", and the read
// commands have always treated that value as a wildcard rather than as the
// literal namespace named "default". That overloading is preserved here
// because changing it would silently alter what `awis history`, `awis
// metrics` and `awis export` return for every existing project — including
// the scaffolded one, whose config.yaml says `namespace: default` while its
// workflows declare `namespace: examples`.
//
// The consequence is worth stating plainly: an instance genuinely in a
// namespace called "default" cannot be selected on its own, because asking
// for it returns everything. Resolving that needs a distinct sentinel (an
// empty default, or an explicit --all-namespaces flag) and is a product
// decision, not a refactor.
//
// This helper exists so the predicate is derived in ONE place; the bug it
// replaces was two halves of a single command disagreeing about it.
func namespacePredicate(ns string) string {
	if ns == "default" {
		return ""
	}
	return ns
}
