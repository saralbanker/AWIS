# M18 — Validation Checklist (binary; exit list)
<!-- Written at A-INIT (2026-10-10). Nothing is ticked yet — no card has run. Ticked by
     M18-V1 (awis-verifier) at C-VERIFY. -->

### Quality gates (PRD §31)
- [ ] QG-1: `awis init` → `awis start` → `awis submit hello-world` → `awis trace <id>`
      succeeds (instance reaches a *completed*, not just terminal-failed, status) on clean
      macOS and Linux with no prior setup, in ≤ 5 minutes. (C2; macOS leg via real CI timing,
      not a local macOS verifier — none exists.)
- [ ] QG-2: `awis trace` output is parsed correctly by a developer who has not read any AWIS
      documentation. **Human-subject measurement — founder-administered, not a card
      deliverable.** Recorded here as a founder ruling when performed (M17 DEC-4 pattern).
- [ ] QG-3: all workflows (incl. OIP's) complete with `NullAdapter` as sole intelligence
      provider. Re-confirm at M18 HEAD against the existing AWIS-E1 CI gate (from M6) — do
      not re-derive from scratch. (C4)
- [ ] QG-4: OIP's `capture-decision`/`recall-decision` workflows run on AWIS with zero
      modification to AWIS platform code. Re-confirm against the M15 G3 boundary demo
      (`git log --stat` artifact) at M18 HEAD. (C4)
- [ ] QG-5: `WorkflowTestHarness` runs a complete workflow (incl. WAIT/signal delivery) in a
      Go unit test in < 1s with zero external dependencies. Re-confirm against the existing
      M09-derived test. (C4)

### NFR benchmarks (PRD §31 NFR-P-01..09, 100K-event fixtures where IMP §27.M18 specifies)
- [ ] NFR-P-01: execution loop tick latency (no I/O) < 5ms P99. (C1)
- [ ] NFR-P-02: native step dispatch latency < 10ms P95 (engineering target) / ≤ 50ms P95
      (PRD §32 ship gate) — **CONTRA-2, record both numbers.** (C1)
- [ ] NFR-P-03: intelligence step latency (Anthropic Haiku) < 5s P95 — **UNMEASURABLE, no
      credential in any verification environment.** Record disposition only, same as PRD §32
      rows 28/32 (M17 DEC-4). Not a blocker.
- [ ] NFR-P-04: signal delivery latency < 200ms. (C1)
- [ ] NFR-P-05: `awis trace` query time (100K events) < 500ms. (C1, 100K-event fixture)
- [ ] NFR-P-06: `awis rebuild-state` (100K events) < 30s. (C1, 100K-event fixture)
- [ ] NFR-P-07: workflow instance startup latency (submit → first step started) < 300ms. (C1)
- [ ] NFR-P-08: plugin spawn time (Python, cold start) < 3s. (C2)
- [ ] NFR-P-09: plugin capability call latency (excl. handler logic) < 50ms. (C2)

### Release tooling
- [ ] `make bench` wired (not the `"NOT-YET"` stub) and green. (C1/C2)
- [ ] `make release-dry` wired (not the `"NOT-YET"` stub); `.goreleaser.yaml` produces a
      4-platform artifact set (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64) +
      checksums + release notes from milestone PR titles. (C3)
- [ ] `docs/V1_BENCHMARKS.md` committed with every NFR-P-01..09 row + CONTRA-2 disposition +
      QG-3/4/5 re-confirmation citations. (C4)

### Acceptance re-measurement
- [ ] Every PRD §32 Must-Have row re-measured at M18 HEAD; no new regression versus M17's
      DEC-4 disposition (49 SATISFIED / 6 NOT SATISFIED / 1 PARTIAL / 3 UNMEASURABLE at
      `9c477b5`). Any row that changed status is called out explicitly, not silently
      absorbed. (C4)

### Fix-only discipline
- [ ] Any NFR miss found in C1/C2 either (a) has a scoped, targeted fix landed in C4 with the
      re-measured number now passing, or (b) is explicitly waived by founder ruling with the
      miss amount recorded — never silently redesigned, never silently dropped.

### Independent verification
- [ ] M18-V1 (awis-verifier) independently reproduces every number above on a clean
      checkout — not transcribed from the implementer's report. macOS legs cross-checked
      against real GitHub Actions run timings.

---
**Explicitly NOT on this checklist (advisory / non-gating / human-only — do not add):**
one-week OIP dogfood metrics (IMP §27.M18 Obj, IR-6: "recorded not gating"); `v1.0.0` tag push
and the G4 ship/no-ship verdict (human-only, happens at D-CLOSE/E-MERGE, not a checklist row
a card can tick).
