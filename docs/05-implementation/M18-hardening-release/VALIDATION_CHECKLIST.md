# M18 — Validation Checklist (binary; exit list)
<!-- Written at A-INIT (2026-10-10). Ticked 2026-10-11 by the CE from M18-V1r's verdicts
     (awis-verifier, clean worktree @ b78ae44, VERDICT PASS; report in TRACEABILITY.md) — the
     verifier's identity forbids doc edits, so it reported and the CE recorded. Unticked rows
     carry their disposition inline. Proposed G4 rulings (CE-recorded 2026-10-12; pending founder G4 VERDICT) noted inline;
     full record in TRACEABILITY.md "Proposed G4 rulings". -->

### Quality gates (PRD §31)
- [x] QG-1: `awis init` → `awis start` → `awis submit hello-world` → `awis trace <id>`
      succeeds (instance reaches a *completed*, not just terminal-failed, status) on clean
      macOS and Linux with no prior setup, in ≤ 5 minutes. (C2; macOS leg via real CI timing,
      not a local macOS verifier — none exists.)
      **V1r:** PASS — linux 249ms/0.83s test, completed; macOS real CI 38155115251 @b78ae44 `ok cmd/awis 45.541s`.
- [~] QG-2: `awis trace` output is parsed correctly by a developer who has not read any AWIS
      documentation. **Human-subject measurement — founder-administered, not a card
      deliverable.** Recorded here as a founder ruling when performed (M17 DEC-4 pattern).
      **V1r: NOT TICKED —** STOP — founder-administered human-subject measurement; pending founder.
      **Proposed G4 ruling (CE-recorded 2026-10-12; pending founder G4 VERDICT): WAIVER for v1.0.0 — unmeasured; NOT PASS.**
      Measure post-release when an independent developer is available (backlog M18-B3).
- [x] QG-3: all workflows (incl. OIP's) complete with `NullAdapter` as sole intelligence
      provider. Re-confirm at M18 HEAD against the existing AWIS-E1 CI gate (from M6) — do
      not re-derive from scratch. (C4)
      **V1r:** PASS — `make e1` 8/8; macOS + ubuntu `Zero-AI gate (E1)` success in CI 38155115251.
- [~] QG-4: OIP's `capture-decision`/`recall-decision` workflows run on AWIS with zero
      modification to AWIS platform code. Re-confirm against the M15 G3 boundary demo
      (`git log --stat` artifact) at M18 HEAD. (C4)
      **V1r: NOT TICKED —** STOP — 9 disclosed platform paths in M15 range; founder G3 Q1 PASS (2026-09-18) on these seams; verifier does not adopt — founder confirms at G4.
      **Proposed G4 ruling (CE-recorded 2026-10-12; pending founder G4 VERDICT): ACCEPTANCE, not literal PASS** — G4 would extend the G3 Q1 ruling
      (P0/P1/P2 seams) to the disclosed `--namespace` flag in `cmd/awis/{main,errors,submit}.go`
      (M15 0e452f8); flag coverage gap → M18-B6.
- [x] QG-5: `WorkflowTestHarness` runs a complete workflow (incl. WAIT/signal delivery) in a
      Go unit test in < 1s with zero external dependencies. Re-confirm against the existing
      M09-derived test. (C4)
      **V1r:** PASS — TestQG5_WaitSignalWithin1s 0.01s; macOS CI sdk/testing ok.

### NFR benchmarks (PRD §31 NFR-P-01..09, 100K-event fixtures where IMP §27.M18 specifies)
- [x] NFR-P-01: execution loop tick latency (no I/O) < 5ms P99. (C1)
      **V1r:** PASS — P99 4.561ms (V1 run 1: 4.792/4.668/4.533ms).
- [x] NFR-P-02: native step dispatch latency < 10ms P95 (engineering target) / ≤ 50ms P95
      (PRD §32 ship gate) — **CONTRA-2, record both numbers.** (C1)
      **V1r:** PASS — P95 1.115ms; 50ms gate PASS, 10ms target PASS (both recorded).
- [ ] NFR-P-03: intelligence step latency (Anthropic Haiku) < 5s P95 — **UNMEASURABLE, no
      credential in any verification environment.** Record disposition only, same as PRD §32
      rows 28/32 (M17 DEC-4). Not a blocker.
      **V1r: NOT TICKED —** UNMEASURABLE (no ANTHROPIC_API_KEY); disposition recorded in docs/V1_BENCHMARKS.md; non-blocking per module spec.
      **Proposed G4 ruling (CE-recorded 2026-10-12; pending founder G4 VERDICT):** stays UNMEASURABLE — not verified; non-blocking.
- [x] NFR-P-04: signal delivery latency < 200ms. (C1)
      **V1r:** PASS — max 99.635ms.
- [x] NFR-P-05: `awis trace` query time (100K events) < 500ms. (C1, 100K-event fixture)
      **V1r:** PASS — multi max 1.914ms; single-instance (100K in one) max 465.672ms.
- [x] NFR-P-06: `awis rebuild-state` (100K events) < 30s. (C1, 100K-event fixture)
      **V1r:** PASS — max 404.603ms.
- [x] NFR-P-07: workflow instance startup latency (submit → first step started) < 300ms. (C1)
      **V1r:** PASS — max 100.775ms.
- [x] NFR-P-08: plugin spawn time (Python, cold start) < 3s. (C2)
      **V1r:** PASS — max 43.236ms.
- [x] NFR-P-09: plugin capability call latency (excl. handler logic) < 50ms. (C2)
      **V1r:** PASS — binding max 5.519ms (floor 0.740ms informational).

### Release tooling
- [x] `make bench` wired (not the `"NOT-YET"` stub) and green. (C1/C2)
      **V1r:** PASS — exit 0, 8 benchmarks, all RESULT PASS.
- [x] `make release-dry` wired (not the `"NOT-YET"` stub); `.goreleaser.yaml` produces a
      4-platform artifact set (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64) +
      checksums + release notes from milestone PR titles. (C3)
      **V1r:** PASS — 4 archives, `(cd dist && sha256sum -c checksums.txt)` OK×4; v1.0.0 notes + `awis version 1.0.0` proven in throwaway clone (V1 run 1).
- [x] `docs/V1_BENCHMARKS.md` committed with every NFR-P-01..09 row + CONTRA-2 disposition +
      QG-3/4/5 re-confirmation citations. (C4)
      **V1r:** PASS.

### Acceptance re-measurement
- [x] Every PRD §32 Must-Have row re-measured at M18 HEAD; no new regression versus M17's
      DEC-4 disposition (49 SATISFIED / 6 NOT SATISFIED / 1 PARTIAL / 3 UNMEASURABLE at
      `9c477b5`). Any row that changed status is called out explicitly, not silently
      absorbed. (C4)
      **V1r:** PASS by card rule — all 59 rows re-measured; status changes called out explicitly: rows 8/15/38/53 NOT→SAT; rows 13, 14, 41 were mis-scored SAT at M17 DEC-4 (not code regressions) and are now 13 FOUNDER DECISION, 14 NOT SATISFIED as written, 41 PARTIAL — founder decisions at G4. Totals 50/3/2/3/1.
      **Proposed G4 rulings (CE-recorded 2026-10-12; pending founder G4 VERDICT):** rows 13, 14 — TDS-07 governs; PRD wording corrected,
      implementation unchanged → SATISFIED BY AMENDMENT. Row 41 — PARTIAL accepted for
      v1.0.0 (outputs via `--json --full`; human format post-release, backlog M18-B4). Rows 37/40/44 and
      5/28/32 carried forward (DEC-11 / DEC-4). Totals now 52 SAT / 2 NOT / 2 PARTIAL / 3 UNMEASURABLE.

### Fix-only discipline
- [x] Any NFR miss found in C1/C2 either (a) has a scoped, targeted fix landed in C4 with the
      re-measured number now passing, or (b) is explicitly waived by founder ruling with the
      miss amount recorded — never silently redesigned, never silently dropped.
      **V1r:** PASS — both C1 misses fixed by scoped C4 diffs, re-measured PASS.

### Independent verification
- [x] M18-V1 (awis-verifier) independently reproduces every number above on a clean
      checkout — not transcribed from the implementer's report. macOS legs cross-checked
      against real GitHub Actions run timings.
      **V1r:** PASS — V1 run 1 (3 bench runs) + V1r; macOS legs from real CI.

---
**Explicitly NOT on this checklist (advisory / non-gating / human-only — do not add):**
one-week OIP dogfood metrics (IMP §27.M18 Obj, IR-6: "recorded not gating"); `v1.0.0` tag push
and the G4 ship/no-ship verdict (human-only, happens at D-CLOSE/E-MERGE, not a checklist row
a card can tick).
