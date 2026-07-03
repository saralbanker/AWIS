# M05 — Validation Checklist (all binary; all must pass)
Verified by awis-verifier card M05-V1 on clean clone at HEAD `5bc33fe`: **15/15 PASS**, no fix
cycle required. Corpus 42/42 (6 rows independently re-probed); fuzz 20s×2 clean (verifier) on top
of 30s×2 clean (builder).

- [x] Clean clone: `make build test lint` exit 0; `go test -race ./internal/expr/... ./internal/validate/...` exit 0
- [x] Corpus conformance: 42/42 rows (15 template + 27 condition) — every Valid row parses, every invalid row rejected; test iterates corpus.Template + corpus.Condition directly
- [x] Every invalid corpus row's error carries a precise Position within source bounds
- [x] corpus.go untouched (`git diff main -- internal/expr/corpus` empty)
- [x] FuzzParseTemplate + FuzzParseCondition exist with corpus seeds; seed corpus passes in CI mode; local fuzz runs recorded in HANDOFF with zero panics
- [x] FR-WD-05: missing path resolves to "" with a Warning; not-yet-completed step ref likewise; Resolve has NO error return
- [x] Frozen null rules tested verbatim: ordering-op vs null ⇒ false; null==null ⇒ true
- [x] EDR-010 semantics matrix tested (no coercion; numeric-only ordering; mismatch ⇒ false/==, true/!=)
- [x] Hyphenated identifiers lex as single tokens (steps.draft-entry.status parses; a-b is not subtraction)
- [x] Validator: every PRD §18 required check has ≥1 failing-case test + ≥1 passing-case test (initial/final refs, orphans, cycles, transition refs, fallback refs, condition grammar, event-scope-in-transition rejected, trigger filter grammar, intelligence fields, signal/intelligence config presence, handler non-empty)
- [x] Validator collects ALL issues (multi-defect definition yields multiple Issues, no first-fail)
- [x] internal/core, sdk/, go.mod unchanged (`git diff main -- internal/core sdk go.mod go.sum` empty)
- [x] Package docs cross-link docs/EXPRESSION_GRAMMARS.md + EDR-010 (IMP DoD "grammar docs cross-linked")
- [x] edr-010 committed; module TRACEABILITY rows complete; global DoD (IMP §24) items 1–7 (macOS CI leg pending remote)
