# M18 → v1.0.0 Handoff
**Status: NOT STARTED.** This file is created at A-INIT (2026-10-10), per the module
template — module previously had no HANDOFF.md/TRACEABILITY.md (it had only a 6-line
README.md, `Status: Partitioned — materialize at entry`). It remains a draft, with every
"guaranteed output" below unconfirmed, until B-BUILD/C-VERIFY run and D-CLOSE records actuals.

## Guaranteed outputs (contract — to be confirmed as actuals at D-CLOSE)
- `make bench` wired: NFR-P-01, 02, 04, 05, 06, 07 (engine/storage/CLI) + NFR-P-08, 09
  (plugin) benchmarked against real fixtures (100K-event for P-05/P-06).
- `.goreleaser.yaml` + `make release-dry`: 4-platform artifact set (linux/amd64,
  linux/arm64, darwin/amd64, darwin/arm64), checksums, PR-title-derived release notes.
- `docs/V1_BENCHMARKS.md`: every NFR-P-01..09 row, CONTRA-2's two numbers, QG-3/4/5
  re-confirmation citations, NFR-P-03's UNMEASURABLE disposition.
- QG-1 formally satisfied (completing instance, not the M17 rehearsal's legitimate `failed`
  proxy) on both Linux (local) and macOS (real CI timing).
- Any benchmark miss: a targeted fix (not a redesign) or an explicit founder waiver with the
  miss amount on record.

## What a post-M18 session may assume (drafted; confirm at D-CLOSE)
- `main` is CI-green on real GitHub Actions for both OS legs — not a local simulation. (This
  A-INIT's immediate predecessor action verified this directly: run `38041177134` at commit
  `4977de7`, both `ubuntu-latest` and `macos-latest` green.)
- `v1.0.0` has NOT been tagged as of this document. No card in this module tags it; that is
  G4's human-only ship/no-ship act at D-CLOSE/E-MERGE.
- Migrations become append-only forever only after the tag lands (IMP §27.M18 Post-tag rule)
  — until then, migration 0008 (the latest, per CURRENT_TRUTH_LEDGER.md) is not yet under
  that permanent constraint.
- V2 planning stays frozen until OIP dogfood evidence exists (Blueprint §33) — the dogfood
  week this module starts (advisory, non-gating) is what produces that evidence, but M18
  itself does not wait on the dogfood week to complete.

## Known limitations (carried in from upstream, not M18's to fix)
- D-13 (deferred feature work), D-16 (P3, doc-only), D-18 (documented goroutine leak under a
  permanently stalled writer), D-19 (P2, lexicographic `started_at` ordering at second
  boundaries) — all open, all explicitly non-blocking per CURRENT_TRUTH_LEDGER.md §2/§4.
  M18's Risk line (IMP §27.M18) authorizes fixes only for benchmark misses M18 itself
  discovers — these four are not that.
- NFR-P-03 (live Anthropic latency) is UNMEASURABLE: no credential exists in any
  verification environment that has touched this repository (CURRENT_TRUTH_LEDGER.md §7.6).
  A future session with a real credential could close this; M18 cannot.
- The macOS CI leg has now run exactly once for real (this session, run `38041177134`). One
  green run is evidence, not a long-run guarantee — CURRENT_TRUTH_LEDGER.md's own load-
  campaign discipline (DEC-22: 10-iteration clean runs before trusting a result) has not yet
  been applied to the macOS leg specifically.

## Actuals (filled as cards complete)
- Not yet dispatched. A-INIT (this session) performed: branch creation, six-file module
  compilation, STATE.md M18 entry. No card has run.
