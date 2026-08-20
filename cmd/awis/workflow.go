package main

// workflow.go — 'awis workflow' sub-commands (TDS-07 §4; M14-C3 T9).
//
// workflow validate <file> — dsl.ValidateFile + Report.Render; exit 3 on invalid.
// workflow list            — registered definitions from storage (WorkflowRegistry).
// workflow show <id>       — definition summary per TDS-07.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/awis/awis/internal/core"
	"github.com/awis/awis/internal/dsl"
)

func init() {
	commands["workflow"] = command{fn: runWorkflow, summary: "Workflow sub-commands (validate, list, show)"}
}

func runWorkflow(args []string) {
	if len(args) == 0 {
		fail(2, "workflow requires a sub-command", "", "awis workflow [validate|list|show]")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "validate":
		runWorkflowValidate(rest)
	case "list":
		runWorkflowList(rest)
	case "show":
		runWorkflowShow(rest)
	default:
		fail(2, fmt.Sprintf("workflow: unknown sub-command %q", sub), "", "awis workflow [validate|list|show]")
	}
}

// ── workflow validate ─────────────────────────────────────────────────────────

// validateOutputJSON is the TDS-07 §4 JSON schema for workflow validate.
type validateOutputJSON struct {
	File   string              `json:"file"`
	Valid  bool                `json:"valid"`
	Errors []validateErrorJSON `json:"errors"`
}

type validateErrorJSON struct {
	Line    *int   `json:"line"`
	Message string `json:"message"`
}

func runWorkflowValidate(args []string) {
	fs := newFlagSet("workflow validate")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 1 {
		fail(2, "workflow validate requires a <file> argument", "", "awis workflow validate <file.yaml>")
	}
	path := rest[0]

	report, err := dsl.ValidateFile(path)
	if err != nil {
		fail(1, fmt.Sprintf("workflow validate: cannot read file: %s", err), path, "check the file path and permissions")
	}

	if globalJSON {
		errs := make([]validateErrorJSON, 0, len(report.Issues))
		for _, iss := range report.Issues {
			var linePtr *int
			if iss.Line > 0 {
				l := iss.Line
				linePtr = &l
			}
			errs = append(errs, validateErrorJSON{Line: linePtr, Message: iss.Message})
		}
		out := validateOutputJSON{
			File:   path,
			Valid:  report.Valid(),
			Errors: errs,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		if !report.Valid() {
			os.Exit(3)
		}
		return
	}

	// Human output.
	if report.Valid() {
		// TDS-07 §4: workflows/capture-decision.yaml  valid ✓
		fmt.Printf("%s  valid ✓\n", path)
		return
	}

	// Invalid: use Report.Render() which produces PRD §18 format.
	// TDS-07 §4:
	// awis: workflow validation failed: workflows/capture-decision.yaml
	//   Line 45: step 'confirm-entry' declares fallback 'manual-entry'...
	fmt.Fprintf(os.Stderr, "awis: workflow validation failed: %s\n", path)
	rendered := report.Render()
	// Render() produces "✗ Validation failed: <file>\n..." — strip the first line
	// since we already printed the header, then print the rest.
	lines := strings.SplitN(rendered, "\n", 2)
	if len(lines) > 1 {
		fmt.Fprintln(os.Stderr, lines[1])
	}
	os.Exit(3)
}

// ── workflow list ─────────────────────────────────────────────────────────────

// workflowListOutputJSON is the TDS-07 §4 JSON schema for workflow list.
type workflowListOutputJSON struct {
	Workflows []workflowEntryJSON `json:"workflows"`
}

type workflowEntryJSON struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	Namespace string `json:"namespace"`
	StepCount int    `json:"step_count"`
}

func runWorkflowList(args []string) {
	fs := newFlagSet("workflow list")
	var namespace string
	fs.StringVar(&namespace, "namespace", "", "Filter by namespace")
	mustParse(fs, args)

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("workflow list: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	ctx := context.Background()
	defs, err := store.ListWorkflows(ctx, namespace)
	if err != nil {
		fail(1, fmt.Sprintf("workflow list: storage query failed: %s", err), globalDataDir+"/runtime.db", "check storage integrity")
	}

	if globalJSON {
		entries := make([]workflowEntryJSON, 0, len(defs))
		for _, d := range defs {
			entries = append(entries, workflowEntryJSON{
				ID:        d.ID,
				Version:   string(d.Version),
				Namespace: d.Namespace,
				StepCount: len(d.Steps),
			})
		}
		out := workflowListOutputJSON{Workflows: entries}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		return
	}

	// TDS-07 §4 human output:
	// REGISTERED WORKFLOWS
	//
	//   ID                    VERSION   NAMESPACE   STEPS
	//   capture-decision      1.0.0     oip         5
	//   recall-decision       1.0.0     oip         4
	fmt.Println("REGISTERED WORKFLOWS")
	fmt.Println()
	fmt.Printf("  %-24s %-10s %-14s %s\n", "ID", "VERSION", "NAMESPACE", "STEPS")
	for _, d := range defs {
		fmt.Printf("  %-24s %-10s %-14s %d\n", d.ID, d.Version, d.Namespace, len(d.Steps))
	}
}

// ── workflow show ─────────────────────────────────────────────────────────────

// workflowShowOutputJSON is the TDS-07 §4 JSON schema for workflow show.
type workflowShowOutputJSON struct {
	ID        string             `json:"id"`
	Version   string             `json:"version"`
	Namespace string             `json:"namespace"`
	Steps     []workflowStepJSON `json:"steps"`
}

type workflowStepJSON struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Next       *string `json:"next"`
	Fallback   *string `json:"fallback"`
	WaitSignal *string `json:"wait_signal"`
}

func runWorkflowShow(args []string) {
	fs := newFlagSet("workflow show")
	var namespace string
	fs.StringVar(&namespace, "namespace", "", "Namespace to look in (default: all)")
	mustParse(fs, args)

	rest := fs.Args()
	if len(rest) < 1 {
		fail(2, "workflow show requires an <id> argument", "", "awis workflow show <id>")
	}
	workflowID := rest[0]

	store, err := OpenStorage(globalDataDir)
	if err != nil {
		fail(1, fmt.Sprintf("workflow show: cannot open storage: %s", err), globalDataDir+"/runtime.db", "check file permissions")
	}

	ctx := context.Background()

	// Find the workflow: list all and match by ID (ListWorkflows takes namespace).
	allDefs, err := store.ListWorkflows(ctx, namespace)
	if err != nil {
		fail(1, fmt.Sprintf("workflow show: storage query failed: %s", err), globalDataDir+"/runtime.db", "check storage integrity")
	}

	var def *core.WorkflowDefinition
	for i := range allDefs {
		if allDefs[i].ID == workflowID {
			d := allDefs[i]
			def = &d
			break
		}
	}
	if def == nil {
		fail(1, fmt.Sprintf("workflow %q not found", workflowID),
			"--data-dir "+globalDataDir,
			"awis workflow list  # to see registered workflows")
	}

	// Build transition map: from → to.
	nextMap := make(map[string]string)
	for _, t := range def.Transitions {
		if !t.OnError && t.Condition == "" {
			nextMap[t.From] = t.To
		}
	}

	if globalJSON {
		steps := make([]workflowStepJSON, 0, len(def.Steps))
		for _, s := range def.Steps {
			next := nextMap[s.ID]
			var nextPtr *string
			if next != "" {
				nextPtr = &next
			}
			var fallbackPtr *string
			if s.Fallback != "" {
				fallbackPtr = &s.Fallback
			}
			var waitSignalPtr *string
			if s.WaitSignal != nil {
				sig := s.WaitSignal.SignalName
				waitSignalPtr = &sig
			}
			steps = append(steps, workflowStepJSON{
				ID:         s.ID,
				Type:       string(s.Type),
				Next:       nextPtr,
				Fallback:   fallbackPtr,
				WaitSignal: waitSignalPtr,
			})
		}
		out := workflowShowOutputJSON{
			ID:        def.ID,
			Version:   string(def.Version),
			Namespace: def.Namespace,
			Steps:     steps,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(out)
		return
	}

	// TDS-07 §4 human output:
	// Workflow: capture-decision v1.0.0  (namespace: oip)
	//
	// Steps:
	//   draft-entry       intelligence  → confirm-entry
	//   confirm-entry     signal        → append-to-record  (signal: entry_confirmed)
	//   append-to-record  native        → publish-entry
	//   publish-entry     native        → [end]
	//
	// Fallbacks:
	//   draft-entry  →  manual-entry (fallback when intelligence unavailable)
	fmt.Printf("Workflow: %s v%s  (namespace: %s)\n", def.ID, def.Version, def.Namespace)
	fmt.Println()
	fmt.Println("Steps:")
	for _, s := range def.Steps {
		next := nextMap[s.ID]
		nextStr := "[end]"
		if next != "" {
			nextStr = next
		}
		extra := ""
		if s.WaitSignal != nil {
			extra = fmt.Sprintf("  (signal: %s)", s.WaitSignal.SignalName)
		}
		fmt.Printf("  %-24s %-14s → %s%s\n", s.ID, string(s.Type), nextStr, extra)
	}

	// Fallbacks section.
	var hasFallbacks bool
	for _, s := range def.Steps {
		if s.Fallback != "" {
			hasFallbacks = true
			break
		}
	}
	if hasFallbacks {
		fmt.Println()
		fmt.Println("Fallbacks:")
		for _, s := range def.Steps {
			if s.Fallback != "" {
				fmt.Printf("  %-20s →  %s (fallback when intelligence unavailable)\n", s.ID, s.Fallback)
			}
		}
	}
}
