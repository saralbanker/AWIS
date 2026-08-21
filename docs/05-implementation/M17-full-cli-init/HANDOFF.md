# M17 → M18 Handoff
**Status: IN PROGRESS — C1/C2/C3 DONE; V1 (verification) not yet dispatched.**
This file is created at C3 completion (module previously had no HANDOFF.md/TRACEABILITY.md);
it will be revised at D-CLOSE once V1 runs.

## Guaranteed outputs (contract — to be confirmed as actuals at D-CLOSE)
- Migration `0006_recall_fts` + `RecallStore` (additive; StoragePort method set untouched).
- `cmd/awis` full command tree: `history`, `logs --tail`, `metrics`, `recall (+--synthesize)`,
  `replay`, `audit`, `export`, `prune-events --dry-run`, `config show|set|validate|edit`,
  `plugin status|remove`, `rebuild-state`, cron trigger scanner in `start`, `init` (go:embed
  scaffold + `--force`).
- `docs/CLI.md` complete for the full tree; `docs/CLI_CONTRACT.md` (TDS-07) section per command.
- init→start→submit→trace rehearsal system test (real binary, not `-short`).

## What M18 may assume (drafted; confirm at D-CLOSE)
- The full CLI surface through M17 is mechanically wired end-to-end: `awis init` produces a
  working scaffold that `start`/`submit`/`status`/`trace` operate against out of the box.
  Formal QG-1/QG-2 measurement against real application handlers remains M18 scope — the M17
  rehearsal test proves CLI plumbing, not business-handler execution (no handlers are compiled
  into the generic `awis` binary).
- Submitting to the scaffolded `hello-world` workflow requires `--namespace=examples`
  (the workflow's declared namespace) — there is no cross-namespace submit fallback.
- `trace --json` has a known defect M18 should account for or fix (see Known limitations).

## Known limitations
- **`trace --json` bug (found, not fixed — out of M17-C3 scope):** silently swallows a
  JSON-encode error when an event's `Payload` is a non-nil zero-length `json.RawMessage`,
  producing empty stdout at exit 0 instead of an error. Discovered while writing the M17-C3
  rehearsal test; worked around there by polling via `status --json` instead. Needs a decision
  at M17-V1 (in-scope fix via revision card) or deferral to M18.
- Destructive `prune-events` (only `--dry-run` shipped), remote plugin install, TUI, and
  `--synthesize` quality tuning are explicitly out of M17 scope (see
  IMPLEMENTATION_SPEC.md `## Non-scope`).
- `config edit` opens `$EDITOR`; its test is skipped in CI (documented at the card level, not a
  new limitation).

## Actuals (filled as cards complete)
- C1: a342394 · C2: 744d3d0 · C3: afe526d
- C3 was a **P2 salvage re-dispatch**: STATE had left it `DISPATCHED` with no commit (a prior
  run died before finishing). Preserved WIP on branch `m17-c3-wip` (9 files, explicitly marked
  non-green) was used as source material only — kept on its branch, not merged, not deleted.
  4 real defects found in that material were corrected (go:embed missing the `all:` prefix so
  `.gitignore` was silently excluded from the binary; the byte-identical test's source paths
  were 2 directory levels too deep; no init goldens existed; no rehearsal system test existed),
  plus 2 remaining gaps closed (`docs/CLI.md` full-tree coverage; TDS-07 `init` section in
  `docs/CLI_CONTRACT.md`). Full defect-by-defect record: module `TRACEABILITY.md`.
  Independently re-verified this session (not solely the implementer's self-report): `make
  verify` full green at afe526d; rehearsal test confirmed running un-skipped and passing;
  `go.mod`/`go.sum` unchanged.
- V1: not yet dispatched. This session's scope was explicitly limited to C3 completion and its
  records — no verification dispatch, no merge work, no M18 work was performed.
