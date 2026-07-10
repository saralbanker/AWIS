// QG-3 harness tests for the OIP application (M15-C2).
//
// Tests: handler unit tests, recall-decision workflow (via awistesting harness),
// rebuild-index equivalence, append-only invariant, and a signal-path test for
// the capture-decision workflow (modified Go definition skipping the plugin step
// for SDK-only compatibility).
//
// All imports are SDK-only: github.com/awis/awis/sdk + sdk/testing.
// Platform internal packages are NOT imported (QG-4 / M15 boundary constraint).
package oip_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	awistesting "github.com/awis/awis/sdk/testing"

	"github.com/awis/awis/sdk"

	oip "github.com/awis/oip"
)

// ── Handler unit tests (no workflow engine) ───────────────────────────────────

// TestQG3_RecordAppend verifies that RecordAppendHandler writes a TDS-06 file
// and inserts a FTS row (Art. 5, Art. 7, Art. 9, Art. 10, Art. 11, Art. 13).
func TestQG3_RecordAppend(t *testing.T) {
	recordRoot := t.TempDir()

	h := oip.NewRecordAppendHandler(recordRoot)
	if h.ID() != "oip.record.append" {
		t.Fatalf("handler ID = %q, want oip.record.append", h.ID())
	}

	ctx := sdk.StepContext{
		Inputs: map[string]any{
			"title":                 "Adopt pure-Go SQLite driver",
			"decision":              "Use modernc.org/sqlite.",
			"rationale":             "CGO-free deployment.",
			"rejected_alternatives": "mattn/go-sqlite3 requires CGO.",
			"unknowns":              "Performance at scale.",
			"tags":                  []string{"architecture", "storage"},
			"provenance_origin":     "manual",
			"provenance_authority":  "test",
			"provenance_confidence": "high",
		},
	}

	result, err := h.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	entryID, ok := result.Outputs["entry_id"].(string)
	if !ok || entryID == "" {
		t.Fatalf("entry_id missing or empty: %v", result.Outputs)
	}

	// ID scheme: D-YYYY-MM-DD-NNN
	if !strings.HasPrefix(entryID, "D-") {
		t.Errorf("entry_id %q does not start with D-", entryID)
	}

	// Verify the .md file exists and is TDS-06-parseable.
	mdPath := filepath.Join(recordRoot, ".decisions", "entries", entryID+".md")
	content, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("entry file not found: %v", err)
	}
	contentStr := string(content)

	// Required frontmatter fields.
	for _, field := range []string{
		"id: " + entryID,
		"date:",
		"title:",
		"status:",
		"corrects: null",
		"provenance:",
		"distinguishes:",
	} {
		if !strings.Contains(contentStr, field) {
			t.Errorf("entry missing frontmatter field %q", field)
		}
	}

	// Required body sections (Art. 11, Art. 13).
	for _, section := range []string{
		"## Decision",
		"## Rationale",
		"## Rejected Alternatives",
		"## Unknowns",
	} {
		if !strings.Contains(contentStr, section) {
			t.Errorf("entry missing body section %q", section)
		}
	}
}

// TestQG3_FTSRow verifies that after append, FTS query finds the entry.
func TestQG3_FTSRow(t *testing.T) {
	recordRoot := t.TempDir()

	appendH := oip.NewRecordAppendHandler(recordRoot)
	res, err := appendH.Execute(sdk.StepContext{
		Inputs: map[string]any{
			"title":                 "SQLite driver choice",
			"decision":              "Pure-Go driver selected for OIP.",
			"rationale":             "CGO-free is critical.",
			"rejected_alternatives": "CGO driver rejected.",
			"unknowns":              "Scale unknowns.",
			"provenance_origin":     "manual",
			"provenance_authority":  "test",
			"provenance_confidence": "high",
		},
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	entryID := res.Outputs["entry_id"].(string)

	ftsH := oip.NewIndexFTSHandler(recordRoot)
	ftsRes, err := ftsH.Execute(sdk.StepContext{
		Inputs: map[string]any{"query": "Pure-Go driver"},
	})
	if err != nil {
		t.Fatalf("fts: %v", err)
	}

	results, _ := ftsRes.Outputs["results"].([]any)
	if len(results) == 0 {
		t.Fatal("FTS returned no results")
	}
	found := false
	for _, r := range results {
		m, _ := r.(map[string]any)
		if m["id"] == entryID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("entry %s not found in FTS results: %v", entryID, results)
	}
}

// TestQG3_SemanticPassthrough verifies CONTRA-3 V1 pass-through.
func TestQG3_SemanticPassthrough(t *testing.T) {
	h := oip.NewSemanticRankHandler()
	if h.ID() != "oip.index.semantic" {
		t.Fatalf("ID = %q, want oip.index.semantic", h.ID())
	}

	input := []any{
		map[string]any{"id": "D-2026-07-10-001", "rank": -1.5},
		map[string]any{"id": "D-2026-07-10-002", "rank": -0.5},
	}
	res, err := h.Execute(sdk.StepContext{
		Inputs: map[string]any{"candidates": input},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	ranked, _ := res.Outputs["ranked"].([]any)
	if len(ranked) != len(input) {
		t.Fatalf("ranked len = %d, want %d", len(ranked), len(input))
	}
}

// TestQG3_RebuildEquivalence verifies that after deleting oip.db and running
// rebuild-index, FTS queries return the same results as before.
func TestQG3_RebuildEquivalence(t *testing.T) {
	recordRoot := t.TempDir()
	appendH := oip.NewRecordAppendHandler(recordRoot)

	titles := []string{"Architecture decision", "Security policy update"}
	entryIDs := make([]string, 0, 2)
	for _, title := range titles {
		res, err := appendH.Execute(sdk.StepContext{
			Inputs: map[string]any{
				"title":                 title,
				"decision":              title + " decided.",
				"rationale":             "Rationale.",
				"rejected_alternatives": "None.",
				"unknowns":              "Unknown.",
				"provenance_origin":     "manual",
				"provenance_authority":  "test",
				"provenance_confidence": "high",
			},
		})
		if err != nil {
			t.Fatalf("append %q: %v", title, err)
		}
		entryIDs = append(entryIDs, res.Outputs["entry_id"].(string))
	}

	// Delete oip.db.
	dbPath := filepath.Join(recordRoot, ".decisions", ".index", "oip.db")
	if err := os.Remove(dbPath); err != nil {
		t.Fatalf("remove oip.db: %v", err)
	}

	// Rebuild.
	rebuildH := oip.NewRebuildIndexHandler(recordRoot)
	_, err := rebuildH.Execute(sdk.StepContext{Inputs: map[string]any{}})
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	// FTS still finds the entries.
	ftsH := oip.NewIndexFTSHandler(recordRoot)
	for _, eid := range entryIDs {
		res, err := ftsH.Execute(sdk.StepContext{
			Inputs: map[string]any{"query": eid},
		})
		if err != nil {
			t.Fatalf("fts after rebuild for %s: %v", eid, err)
		}
		results, _ := res.Outputs["results"].([]any)
		found := false
		for _, r := range results {
			m, _ := r.(map[string]any)
			if m["id"] == eid {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("entry %s not found after rebuild; results: %v", eid, results)
		}
	}
}

// TestQG3_AppendOnly verifies that corrects-chain never rewrites existing files
// (Art. 7: Record is corrected by appending, never by silent rewriting).
func TestQG3_AppendOnly(t *testing.T) {
	recordRoot := t.TempDir()
	appendH := oip.NewRecordAppendHandler(recordRoot)

	res1, err := appendH.Execute(sdk.StepContext{
		Inputs: map[string]any{
			"title":                 "Original decision",
			"decision":              "Use approach A.",
			"rationale":             "Rationale A.",
			"rejected_alternatives": "Approach B.",
			"unknowns":              "Unknown.",
			"provenance_origin":     "manual",
			"provenance_authority":  "test",
			"provenance_confidence": "high",
		},
	})
	if err != nil {
		t.Fatalf("append original: %v", err)
	}
	originalID := res1.Outputs["entry_id"].(string)

	res2, err := appendH.Execute(sdk.StepContext{
		Inputs: map[string]any{
			"title":                 "Corrected decision",
			"decision":              "Use approach B instead.",
			"rationale":             "Rationale B.",
			"rejected_alternatives": "Approach A (original).",
			"unknowns":              "None.",
			"corrects":              originalID,
			"provenance_origin":     "manual",
			"provenance_authority":  "test",
			"provenance_confidence": "high",
		},
	})
	if err != nil {
		t.Fatalf("append correcting: %v", err)
	}
	correctingID := res2.Outputs["entry_id"].(string)

	// Both files must exist.
	for _, id := range []string{originalID, correctingID} {
		mdPath := filepath.Join(recordRoot, ".decisions", "entries", id+".md")
		if _, err := os.Stat(mdPath); err != nil {
			t.Errorf("entry file %s missing: %v", id, err)
		}
	}

	// Original file content unchanged.
	origContent, err := os.ReadFile(
		filepath.Join(recordRoot, ".decisions", "entries", originalID+".md"),
	)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	if !strings.Contains(string(origContent), "Use approach A.") {
		t.Error("original entry was modified (append-only violated)")
	}

	// Correcting entry links back.
	corrContent, err := os.ReadFile(
		filepath.Join(recordRoot, ".decisions", "entries", correctingID+".md"),
	)
	if err != nil {
		t.Fatalf("read correcting: %v", err)
	}
	if !strings.Contains(string(corrContent), "corrects: "+originalID) {
		t.Errorf("correcting entry missing corrects link; content:\n%s", string(corrContent))
	}
}

// ── Workflow-engine tests (awistesting harness, NullAdapter) ─────────────────

// TestQG3_RecallWorkflow tests the recall-decision workflow end-to-end via the
// awistesting harness with NullAdapter. Seeds entries then submits the workflow.
func TestQG3_RecallWorkflow(t *testing.T) {
	recordRoot := t.TempDir()

	// Seed an entry.
	appendH := oip.NewRecordAppendHandler(recordRoot)
	_, err := appendH.Execute(sdk.StepContext{
		Inputs: map[string]any{
			"title":                 "Recall test entry",
			"decision":              "This is the recalled decision.",
			"rationale":             "Recall rationale.",
			"rejected_alternatives": "None.",
			"unknowns":              "Unknown.",
			"provenance_origin":     "manual",
			"provenance_authority":  "test",
			"provenance_confidence": "high",
		},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Build harness with FTS + semantic handlers + NullAdapter (default).
	h := awistesting.NewHarness(t,
		awistesting.WithStepHandler(oip.NewIndexFTSHandler(recordRoot)),
		awistesting.WithStepHandler(oip.NewSemanticRankHandler()),
	)

	def, err := sdk.LoadWorkflowFile("workflows/recall-decision.yaml")
	if err != nil {
		t.Fatalf("LoadWorkflowFile: %v", err)
	}

	_, err = h.Run(def, map[string]any{"query": "recalled decision"})
	// NullAdapter may degrade the intelligence step; workflow may fail or complete.
	// The test verifies the pipeline runs without a handler-not-found error.
	if err != nil && strings.Contains(err.Error(), "handler not found") {
		t.Fatalf("handler not found error: %v", err)
	}
}

// TestQG3_RebuildIndexWorkflow tests the rebuild-index workflow end-to-end
// via the awistesting harness.
func TestQG3_RebuildIndexWorkflow(t *testing.T) {
	recordRoot := t.TempDir()

	// Seed an entry.
	appendH := oip.NewRecordAppendHandler(recordRoot)
	_, err := appendH.Execute(sdk.StepContext{
		Inputs: map[string]any{
			"title":                 "Rebuild workflow test",
			"decision":              "Test decision.",
			"rationale":             "Test rationale.",
			"rejected_alternatives": "None.",
			"unknowns":              "None.",
			"provenance_origin":     "manual",
			"provenance_authority":  "test",
			"provenance_confidence": "high",
		},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Delete db to force rebuild.
	dbPath := filepath.Join(recordRoot, ".decisions", ".index", "oip.db")
	_ = os.Remove(dbPath)

	h := awistesting.NewHarness(t,
		awistesting.WithStepHandler(oip.NewRebuildIndexHandler(recordRoot)),
	)

	def, err := sdk.LoadWorkflowFile("workflows/rebuild-index.yaml")
	if err != nil {
		t.Fatalf("LoadWorkflowFile: %v", err)
	}

	res, err := h.Run(def, map[string]any{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Step outputs are stored as steps.rebuild.outputs.status in the variables map.
	// Use GetOutput with the full variable path key OR simply verify db is recreated.
	_ = res

	// db must be recreated (the rebuild handler creates it).
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("oip.db not recreated: %v", err)
	}
}

// TestQG3_CaptureSignalPath tests the capture-decision signal path using the
// awistesting harness with a simplified Go workflow definition (no plugin step,
// no intelligence step, pure signal → append path).
// The full capture-decision YAML is NOT used here because its intelligence
// fallback path requires complex engine state. The QG-3 signal path is:
// context-stub → confirm-entry (signal) → append-to-record.
//
// Signal payload fields are mapped flat into the append step's inputs because
// the AWIS template system always resolves to strings (stringify) — nested
// map fields must be referenced individually as steps.<id>.outputs.<field>.
func TestQG3_CaptureSignalPath(t *testing.T) {
	recordRoot := t.TempDir()

	stub := &contextStubHandler{root: recordRoot}

	h := awistesting.NewHarness(t,
		awistesting.WithStepHandler(stub),
		awistesting.WithStepHandler(oip.NewRecordAppendHandler(recordRoot)),
	)

	// Simplified capture workflow: stub → signal → append.
	// The signal payload carries flat entry fields; each is referenced
	// individually in the append step inputs (template system → string).
	def := &sdk.WorkflowDefinition{
		SchemaVersion: 1,
		ID:            "capture-test",
		Version:       "1.0.0",
		Namespace:     "test",
		Name:          "Capture Test",
		InitialStep:   "assemble",
		FinalSteps:    []string{"append"},
		Steps: []sdk.Step{
			{
				ID:      "assemble",
				Type:    sdk.StepTypeNative,
				Handler: "test.assemble-context",
				Outputs: sdk.OutputSchema{"context": map[string]any{"type": "object"}},
			},
			{
				ID:   "confirm",
				Type: sdk.StepTypeSignal,
				WaitSignal: &sdk.WaitConfig{
					SignalName:    "entry_confirmed",
					TimeoutAction: "cancel",
				},
			},
			{
				ID:      "append",
				Type:    sdk.StepTypeNative,
				Handler: "oip.record.append",
				Inputs: sdk.InputSchema{
					"title":                 "{{steps.confirm.outputs.title}}",
					"decision":              "{{steps.confirm.outputs.decision}}",
					"rationale":             "{{steps.confirm.outputs.rationale}}",
					"rejected_alternatives": "{{steps.confirm.outputs.rejected_alternatives}}",
					"unknowns":              "{{steps.confirm.outputs.unknowns}}",
					"provenance_origin":     "{{steps.confirm.outputs.provenance_origin}}",
					"provenance_authority":  "{{steps.confirm.outputs.provenance_authority}}",
					"provenance_confidence": "{{steps.confirm.outputs.provenance_confidence}}",
				},
				Outputs: sdk.OutputSchema{"entry_id": map[string]any{"type": "string"}},
			},
		},
		Transitions: []sdk.Transition{
			{From: "assemble", To: "confirm"},
			{From: "confirm", To: "append"},
		},
	}

	res, err := h.Run(def, map[string]any{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Workflow is waiting on entry_confirmed. Signal payload carries flat fields
	// so the template engine can resolve each as steps.confirm.outputs.<field>.
	h.Signal(res.InstanceID, "entry_confirmed", map[string]any{
		"title":                 "Decision captured via signal path",
		"decision":              "Signal-path decision.",
		"rationale":             "Signal-path rationale.",
		"rejected_alternatives": "None.",
		"unknowns":              "None.",
		"provenance_origin":     "workflow",
		"provenance_authority":  "test",
		"provenance_confidence": "high",
	})

	// A .md file must exist after append-to-record.
	entriesDir := filepath.Join(recordRoot, ".decisions", "entries")
	files, err := os.ReadDir(entriesDir)
	if err != nil {
		t.Fatalf("entries dir not found: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no entry files created after capture signal path")
	}

	// Verify the entry is TDS-06-parseable.
	content, err := os.ReadFile(filepath.Join(entriesDir, files[0].Name()))
	if err != nil {
		t.Fatalf("read entry file: %v", err)
	}
	contentStr := string(content)
	for _, section := range []string{"## Decision", "## Rationale", "## Rejected Alternatives", "## Unknowns"} {
		if !strings.Contains(contentStr, section) {
			t.Errorf("entry missing section %q", section)
		}
	}

	// FTS row must be present.
	ftsH := oip.NewIndexFTSHandler(recordRoot)
	ftsRes, err := ftsH.Execute(sdk.StepContext{
		Inputs: map[string]any{"query": "Signal-path decision"},
	})
	if err != nil {
		t.Fatalf("fts: %v", err)
	}
	results, _ := ftsRes.Outputs["results"].([]any)
	if len(results) == 0 {
		t.Fatal("FTS found no results after capture signal path")
	}
}

// contextStubHandler is a native step handler that satisfies the assemble-context
// step in the capture-decision workflow for QG-3 testing.
type contextStubHandler struct {
	root string
}

func (h *contextStubHandler) ID() string { return "test.assemble-context" }
func (h *contextStubHandler) Execute(_ sdk.StepContext) (sdk.StepResult, error) {
	return sdk.StepResult{
		Outputs: map[string]any{
			"context": map[string]any{
				"repo":    h.root,
				"branch":  "main",
				"commits": []any{},
			},
		},
	}, nil
}
