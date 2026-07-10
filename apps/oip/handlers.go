// Package oip implements the OIP application handlers for AWIS workflows.
// All four handlers are native StepHandler implementations registered with the
// AWIS runtime via sdk.RegisterHandler.
package oip

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/awis/awis/sdk"

	"github.com/awis/oip/internal/index"
	"github.com/awis/oip/internal/record"
)

// RecordAppendHandler implements the "oip.record.append" native handler.
// It writes a TDS-06 .md entry to disk then inserts into oip.db in one
// operation (Art. 5: file is source of truth; db is index).
type RecordAppendHandler struct {
	writer *record.Writer
}

// NewRecordAppendHandler creates a RecordAppendHandler rooted at recordRoot.
func NewRecordAppendHandler(recordRoot string) *RecordAppendHandler {
	return &RecordAppendHandler{writer: record.NewWriter(recordRoot)}
}

// ID returns the handler identifier "oip.record.append".
func (h *RecordAppendHandler) ID() string { return "oip.record.append" }

// Execute appends a new TDS-06 entry to the Record.
//
// Inputs (from the confirmed_entry map or flat inputs):
//
//	title        string  (required)
//	decision     string
//	rationale    string
//	rejected_alternatives string
//	unknowns     string
//	tags         []string or JSON array string
//	corrects     string (optional)
//	provenance_origin     string
//	provenance_authority  string
//	provenance_sources    []string or JSON array string
//	provenance_confidence string
//	observation  string (optional)
//	description  string (optional)
//	intention    string (optional)
//
// Outputs:
//
//	entry_id string — the generated D-YYYY-MM-DD-NNN identifier.
func (h *RecordAppendHandler) Execute(ctx sdk.StepContext) (sdk.StepResult, error) {
	inputs := ctx.Inputs

	// Support both flat inputs and nested "entry" map (from workflow variable).
	if entryRaw, ok := inputs["entry"]; ok {
		if entryMap, ok := entryRaw.(map[string]any); ok {
			// Merge entry fields into inputs.
			for k, v := range entryMap {
				if _, exists := inputs[k]; !exists {
					inputs[k] = v
				}
			}
		}
	}

	e := record.Entry{
		Title:               strInput(inputs, "title"),
		Status:              strInput(inputs, "status"),
		Corrects:            strInput(inputs, "corrects"),
		Decision:            strInput(inputs, "decision"),
		Rationale:           strInput(inputs, "rationale"),
		RejectedAlternatives: strInput(inputs, "rejected_alternatives"),
		Unknowns:            strInput(inputs, "unknowns"),
		Tags:                sliceInput(inputs, "tags"),
		Provenance: record.Provenance{
			Origin:     strInput(inputs, "provenance_origin"),
			Authority:  strInput(inputs, "provenance_authority"),
			Sources:    sliceInput(inputs, "provenance_sources"),
			Confidence: strInput(inputs, "provenance_confidence"),
		},
		Distinguishes: record.Distinguishes{
			Observation: strInput(inputs, "observation"),
			Description: strInput(inputs, "description"),
			Intention:   strInput(inputs, "intention"),
		},
	}

	if e.Title == "" {
		return sdk.StepResult{}, fmt.Errorf("oip.record.append: title is required")
	}

	id, err := h.writer.Append(context.Background(), e)
	if err != nil {
		return sdk.StepResult{}, fmt.Errorf("oip.record.append: %w", err)
	}

	return sdk.StepResult{
		Outputs: map[string]any{"entry_id": id},
	}, nil
}

// IndexFTSHandler implements the "oip.index.fts" native handler.
// Input: query string. Output: results []map[string]any with id/path/snippet/rank.
type IndexFTSHandler struct {
	querier *index.Querier
}

// NewIndexFTSHandler creates an IndexFTSHandler rooted at recordRoot.
func NewIndexFTSHandler(recordRoot string) *IndexFTSHandler {
	return &IndexFTSHandler{querier: index.NewQuerier(recordRoot)}
}

// ID returns "oip.index.fts".
func (h *IndexFTSHandler) ID() string { return "oip.index.fts" }

// Execute performs a ranked FTS5 query against oip.db.
func (h *IndexFTSHandler) Execute(ctx sdk.StepContext) (sdk.StepResult, error) {
	query := strInput(ctx.Inputs, "query")
	if query == "" {
		return sdk.StepResult{Outputs: map[string]any{"results": []any{}}}, nil
	}

	results, err := h.querier.FTSQuery(context.Background(), query)
	if err != nil {
		return sdk.StepResult{}, fmt.Errorf("oip.index.fts: %w", err)
	}

	out := make([]any, 0, len(results))
	for _, r := range results {
		out = append(out, map[string]any{
			"id":      r.ID,
			"path":    r.Path,
			"snippet": r.Snippet,
			"rank":    r.Rank,
		})
	}
	return sdk.StepResult{Outputs: map[string]any{"results": out}}, nil
}

// SemanticRankHandler implements the "oip.index.semantic" native handler.
// In V1 this is a pass-through (CONTRA-3: no vectors exist).
// Input: results []any (from fts-search). Output: ranked []any (same order).
type SemanticRankHandler struct{}

// NewSemanticRankHandler creates a SemanticRankHandler.
func NewSemanticRankHandler() *SemanticRankHandler { return &SemanticRankHandler{} }

// ID returns "oip.index.semantic".
func (h *SemanticRankHandler) ID() string { return "oip.index.semantic" }

// Execute passes FTS results through unchanged (CONTRA-3 V1 pass-through).
func (h *SemanticRankHandler) Execute(ctx sdk.StepContext) (sdk.StepResult, error) {
	candidates := ctx.Inputs["candidates"]
	return sdk.StepResult{Outputs: map[string]any{"ranked": candidates}}, nil
}

// RebuildIndexHandler implements the "oip.index.rebuild" native handler.
// It drops and recreates oip.db from all .md files in .decisions/entries/.
type RebuildIndexHandler struct {
	writer *record.Writer
}

// NewRebuildIndexHandler creates a RebuildIndexHandler rooted at recordRoot.
func NewRebuildIndexHandler(recordRoot string) *RebuildIndexHandler {
	return &RebuildIndexHandler{writer: record.NewWriter(recordRoot)}
}

// ID returns "oip.index.rebuild".
func (h *RebuildIndexHandler) ID() string { return "oip.index.rebuild" }

// Execute rebuilds oip.db from .decisions/entries/*.md files.
func (h *RebuildIndexHandler) Execute(ctx sdk.StepContext) (sdk.StepResult, error) {
	if err := h.writer.RebuildIndex(context.Background()); err != nil {
		return sdk.StepResult{}, fmt.Errorf("oip.index.rebuild: %w", err)
	}
	return sdk.StepResult{Outputs: map[string]any{"status": "ok"}}, nil
}

// strInput extracts a string from a map[string]any input, returning "" if absent.
func strInput(inputs map[string]any, key string) string {
	v, ok := inputs[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// sliceInput extracts a []string from a map[string]any input, handling
// []string, []any, and JSON-encoded array string forms.
func sliceInput(inputs map[string]any, key string) []string {
	v, ok := inputs[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		result := make([]string, 0, len(t))
		for _, item := range t {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	case string:
		if t == "" {
			return nil
		}
		var arr []string
		if err := json.Unmarshal([]byte(t), &arr); err == nil {
			return arr
		}
		return []string{t}
	}
	return nil
}
