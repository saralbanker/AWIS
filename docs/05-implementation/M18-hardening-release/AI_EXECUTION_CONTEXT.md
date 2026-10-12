# M18 — AI Execution Context
**Model:** Mixed + Human (IMP §28: "Benchmarks/fixes are Sonnet; ship/no-ship is human").
Cards to `awis-builder` (C1, C2, C3, C4); V1 to `awis-verifier`. Dispatch order per
IMPLEMENTATION_SPEC.md "Card split": C1 → C2 → C3 → C4 → V1. Branch: `m18-hardening-release`
(branched from `main` at `4977de7`, the CI-fix commit this A-INIT session verified green on
real GitHub Actions — see TRACEABILITY.md).

**Gate:** **G4 — Release** (IMP §23). Question the human answers: "Do QG-1..5 pass and are
NFR benchmarks within targets on clean machines?" Evidence: benchmark report, quality-gate
checklist, one week of dogfood metrics. Failure action: ship blocks, hardening continues.
G4 fires at D-CLOSE/E-MERGE, not at any card — no card in this module may tag `v1.0.0` or
declare ship/no-ship.

**STATUS AS OF THIS DOCUMENT (2026-10-10, A-INIT session): M18 was never entered before now.**
`docs/05-implementation/STATE.md` had no M18 block at all — DONE-MILESTONES stopped at M17.
This session's immediate predecessor fixed a real, reproduced macOS CI failure
(`TestResolvePluginUser_UnknownNumericUID`, `internal/plugin/isolation_test.go`) and
confirmed green on actual GitHub Actions (run `38041177134`, commit `4977de7`) before this
A-INIT began — that CI-green state is the entry gate this module's founder directive required
before M18 could open. No card has been dispatched. No benchmark has been run. No release
artifact exists. Treat every number in this module as **not yet measured** until V1 reports.

**Key constraints (IMPLEMENTATION_SPEC.md "CE pins"):**
1. Benchmark misses get a targeted fix, never a redesign (IMP §27.M18 Risk line, binding).
2. CONTRA-2 is recorded as two numbers (10ms target, 50ms gate), not collapsed to one.
3. 100K-event fixtures are only required for NFR-P-05/P-06; other NFRs use workload-
   appropriate fixture sizes, stated per-benchmark.
4. QG-1 requires a workflow that actually *completes* — the M17 rehearsal's "legitimately
   ends `failed`" result does not satisfy QG-1's literal "succeeds" wording.
5. QG-2 is human-subject; no card may claim to have satisfied it by itself.
6. The OIP dogfood week is advisory/non-gating (IR-6) — it must not appear as a blocking row
   in any card's ACCEPTANCE criteria.
7. NFR-P-03 (live Anthropic latency) is UNMEASURABLE in this environment (no credential) —
   record that disposition, do not fabricate a number and do not treat its absence as a
   blocker invented for this milestone.
8. No engine/core/sdk/StoragePort change outside a scoped, benchmark-driven fix-only card.
9. `v1.0.0` tag and the G4 ship/no-ship verdict are human-only — no card performs either.

**Milestone escalation deltas:** none yet — this module has no execution history before this
A-INIT. (This section exists per the M15/M17 module template; it will accumulate real entries
as B-BUILD/C-VERIFY run.)
