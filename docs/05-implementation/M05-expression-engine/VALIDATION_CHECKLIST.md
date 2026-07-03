# M05 — Validation Checklist (all binary; all must pass)

- [ ] Clean clone: `make build test lint` exit 0; `go test -race ./internal/expr/... ./internal/validate/...` exit 0
- [ ] Corpus conformance: 42/42 rows (15 template + 27 condition) — every Valid row parses, every invalid row rejected; test iterates corpus.Template + corpus.Condition directly
- [ ] Every invalid corpus row's error carries a precise Position within source bounds
- [ ] corpus.go untouched (`git diff main -- internal/expr/corpus` empty)
- [ ] FuzzParseTemplate + FuzzParseCondition exist with corpus seeds; seed corpus passes in CI mode; longer local fuzz run recorded in HANDOFF with zero panics
- [ ] FR-WD-05: missing path resolves to "" with a Warning; not-yet-completed step ref likewise; Resolve has NO error return
- [ ] Frozen null rules tested verbatim: ordering-op vs null ⇒ false; null==null ⇒ true
- [ ] EDR-010 semantics matrix tested (no coercion; numeric-only ordering; mismatch ⇒ false/==, true/!=)
- [ ] Hyphenated identifiers lex as single tokens (steps.draft-entry.status parses; a-b is not subtraction)
- [ ] Validator: every PRD §18 required check has ≥1 failing-case test + ≥1 passing-case test (initial/final refs, orphans, cycles, transition refs, fallback refs, condition grammar, event-scope-in-transition rejected, trigger filter grammar, intelligence fields, signal/intelligence config presence, handler non-empty)
- [ ] Validator collects ALL issues (multi-defect definition yields multiple Issues, no first-fail)
- [ ] internal/core, sdk/, go.mod unchanged (`git diff main -- internal/core sdk go.mod go.sum` empty)
- [ ] Package docs cross-link docs/EXPRESSION_GRAMMARS.md + EDR-010 (IMP DoD "grammar docs cross-linked")
- [ ] edr-010 committed; module TRACEABILITY rows complete; global DoD (IMP §24) items 1–7 (macOS CI leg pending remote)
