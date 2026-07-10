// Package index implements FTS query (ranked, with snippets) and semantic
// pass-through for the OIP recall workflow.
//
// In V1, semantic ranking is a PASS-THROUGH: SemanticRank returns FTS results
// unchanged because entry_vectors is empty (CONTRA-3; no embeddings generated
// in V1). The entry_vectors table is present in the schema for V2 migration
// with zero schema change.
//
// This package is OIP's own internal package and does NOT import any package
// under github.com/awis/awis/internal (platform boundary; M15 QG-4).
package index
