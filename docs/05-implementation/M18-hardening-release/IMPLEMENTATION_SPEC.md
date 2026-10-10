# M18 — Implementation Spec
**Sources:** IMP §27.M18; §20 (M18 verification checkpoint: QG-1..5 literal on clean
macOS+Linux; NFR report committed); §21 (CD: tags `v*` run goreleaser, 4-platform static
binaries, checksums, release notes from milestone PR titles); §23 Gate G4 ("Do QG-1..5 pass
and are NFR benchmarks within targets on clean machines?"); §25 F-6 (week-6 load, LOW,
already absorbed — moot, M16/M17 are long closed); §26 (post-tag: migrations append-only
forever); PRD §31 (QG-1..5, NFR-P-01..09 table); PRD §32 (Must-Have acceptance rows);
CONTRA-2 (NFR-P-02 dual threshold: 10ms engineering target / 50ms ship gate).

## CE pins
- **Benchmark suite, not redesign.** IMP §27.M18 Risk line is binding: "benchmark misses →
  targeted fixes only, no redesign." A miss gets a scoped fix card or a founder-ruled waiver
  (M17 DEC-pattern), never a rewrite.
- **CONTRA-2 is dual, not single.** NFR-P-02 (native dispatch) records BOTH the 10ms
  engineering target and the 50ms PRD §32 ship-gate number in `docs/V1_BENCHMARKS.md`; the
  ship gate is 50ms P95, 10ms is informational.
- **100K-event fixtures** back NFR-P-05 (`trace` query) and NFR-P-06 (`rebuild-state`) only —
  the two NFRs IMP §27.M18 names the fixture size against. Other NFRs use workload-
  appropriate fixture sizes (documented per-benchmark, not invented ad hoc).
- **QG-1 must be a real walkthrough, not the M17 rehearsal's "measured loosely" proxy.**
  M17's `TestSystemRehearsalInitStartSubmitTrace` legitimately ends the instance `failed`
  because the generic `awis` binary has no compiled-in handlers (M17 HANDOFF.md, confirmed
  this document). QG-1's "succeeds" wording requires a workflow that actually completes —
  use the `awis init` scaffold's own embedded `hello-world` example with its
  `handlers/example_handler.go`, compiled into a throwaway test binary, so the timed path is
  init → start → submit → trace on a *completing* instance.
- **QG-2 is a human-subject measurement** ("parsed correctly by a developer who has not read
  any AWIS documentation") — no model can self-administer this. Record it as a founder-
  measured checklist row (M17 DEC-4 pattern: a human ruling restated as a measurable
  condition), not a card deliverable.
- **QG-3/4/5 likely already hold** on existing evidence (E1 gate from M6 for QG-3; the M15
  G3 boundary demo for QG-4; the M09 `WorkflowTestHarness` sub-1s test for QG-5) — M18's job
  is to **re-confirm these citations still hold at M18 HEAD**, not re-derive them from
  scratch. Re-confirmation is still a reportable step: the PRD's own citations have been
  wrong before (CURRENT_TRUTH_LEDGER.md §5, "claims disproven by evidence").
- **macOS leg of QG-1/NFR runs through real CI, not a local verifier.** No macOS verifier
  agent exists in this environment. `.github/workflows/ci.yml`'s `macos-latest` leg (now
  green as of `4977de7`, this session) is the clean-macOS evidence; capture its step timings
  rather than requiring a second, redundant manual macOS run.
- **The OIP one-week dogfood (IMP §27.M18 Obj) is advisory, not gating** — IMP says
  explicitly "recorded not gating" (IR-6, E1-equivalent metrics). It does not block B-BUILD,
  C-VERIFY, or D-CLOSE. It is tracked in HANDOFF.md as a standing founder-run activity, not a
  dispatchable card.
- **Release tooling starts from zero.** `make bench` and `make release-dry` are both stubs
  today (`Makefile`: `"NOT-YET (wired at M02/M06/M18): bench"` / `release-dry`); no
  `.goreleaser.yaml` exists. This is new wiring, not a fix to existing wiring.

## Card split
C1: NFR benchmark harness + `make bench` wiring + 100K-event fixtures; engine/storage/CLI
    NFRs — NFR-P-01 (tick latency), NFR-P-02 (native dispatch, CONTRA-2 dual), NFR-P-04
    (signal delivery), NFR-P-05 (trace query, 100K), NFR-P-06 (rebuild-state, 100K), NFR-P-07
    (submit→first-step).
C2: Plugin NFRs — NFR-P-08 (plugin spawn cold start), NFR-P-09 (plugin capability-call
    latency); formal QG-1 rehearsal (completing instance, real handler, timed ≤5min);
    `docs/V1_BENCHMARKS.md` skeleton (tables, no numbers yet).
C3: `.goreleaser.yaml` (4 platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64) +
    checksums + release notes from milestone PR titles; wire `make release-dry`.
C4: Compile final `docs/V1_BENCHMARKS.md` (C1+C2 numbers, CONTRA-2 disposition); re-confirm
    QG-3/QG-4/QG-5 citations at current HEAD; re-measure PRD §32 Must-Have rows against
    M17's DEC-4 disposition for drift; any benchmark miss from C1/C2 gets a targeted fix here
    (fix-only, no redesign) or an explicit founder-waiver note if a fix isn't scoped-safe.
V1: verify — independent re-run of `make verify` + `make bench` + `make release-dry` on a
    clean checkout; cross-check macOS timings against the real CI run; confirm
    `docs/V1_BENCHMARKS.md` numbers are independently reproduced, not transcribed; tick
    VALIDATION_CHECKLIST.md.

NOT in any card (human/advisory, tracked in README.md/HANDOFF.md only):
- QG-2 human-subject measurement (founder-administered).
- One-week OIP dogfood (advisory, non-gating, IR-6).
- `v1.0.0` tag push + G4 ship/no-ship verdict (human-only, EEOS rule 6/IMP §28).

## Non-scope
Fixing pre-existing non-blocking defects D-13/D-16/D-18/D-19 (CURRENT_TRUTH_LEDGER.md §2) —
tracked separately, not M18's objective. Redesigning any NFR-missed subsystem. Live-API
intelligence benchmarking (NFR-P-03, Anthropic Haiku draft latency) — no credential exists in
any verification environment (CURRENT_TRUTH_LEDGER.md §7.6); record as UNMEASURABLE with the
same disposition PRD §32 rows 28/32 already carry, not a blocker invented for M18. Engine,
core, SDK, StoragePort changes beyond what a scoped fix-only card requires — frozen surfaces
stay frozen.
