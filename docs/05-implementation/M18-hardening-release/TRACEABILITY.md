# M18 — Traceability
| T | Task | Source coordinates | Card |
|---|---|---|---|
| T1 | NFR benchmark harness + `make bench` wiring | IMP §21 (bench: informational until M18, then thresholded); Makefile stub | C1 |
| T2 | NFR-P-01 tick latency benchmark | PRD §31 NFR-P-01 | C1 |
| T3 | NFR-P-02 native dispatch benchmark (CONTRA-2 dual) | PRD §31 NFR-P-02; IMP §25 CONTRA-2; PRD §32 Performance row | C1 |
| T4 | NFR-P-04 signal delivery benchmark | PRD §31 NFR-P-04 | C1 |
| T5 | NFR-P-05 trace query benchmark (100K-event fixture) | PRD §31 NFR-P-05; IMP §27.M18 Obj (fixture size) | C1 |
| T6 | NFR-P-06 rebuild-state benchmark (100K-event fixture) | PRD §31 NFR-P-06; IMP §27.M18 Obj (fixture size) | C1 |
| T7 | NFR-P-07 submit→first-step benchmark | PRD §31 NFR-P-07 | C1 |
| T8 | NFR-P-08 plugin spawn benchmark | PRD §31 NFR-P-08 | C2 |
| T9 | NFR-P-09 plugin capability-call benchmark | PRD §31 NFR-P-09 | C2 |
| T10 | QG-1 formal rehearsal (completing instance, real handler, timed) | PRD §31 QG-1; distinguished from M17's proxy per M17 HANDOFF.md | C2 |
| T11 | `docs/V1_BENCHMARKS.md` skeleton | IMP §27.M18 Obj | C2 |
| T12 | `.goreleaser.yaml` (4 platforms) + checksums + PR-title release notes | IMP §21 CD | C3 |
| T13 | `make release-dry` wiring | Makefile stub; IMP §27.M18 Obj | C3 |
| T14 | `docs/V1_BENCHMARKS.md` final compilation | IMP §27.M18 Obj; CONTRA-2 | C4 |
| T15 | QG-3/4/5 re-confirmation at M18 HEAD | PRD §31 QG-3/4/5; M6 E1 gate, M15 G3 demo, M09 harness test | C4 |
| T16 | PRD §32 Must-Have re-measurement vs. M17 DEC-4 disposition | PRD §32; CURRENT_TRUTH_LEDGER.md (M17 disposition) | C4 |
| T17 | Fix-only remediation for any C1/C2 benchmark miss | IMP §27.M18 Risk line (binding: no redesign) | C4 |
| T18 | Independent re-verification: benchmarks, release-dry, QG checklist | IMP §27.M18 Val (G4 checklist) | V1 |

## Notes / dispositions
- **Not a task, by design:** QG-2 (human-subject trace comprehension), the one-week OIP
  dogfood (advisory/non-gating, IR-6), and the `v1.0.0` tag + G4 verdict (human-only). See
  IMPLEMENTATION_SPEC.md "NOT in any card."
- **CONTRA-2 disposition:** both the 10ms engineering target and the 50ms PRD §32 ship gate
  are recorded for NFR-P-02 — neither number is dropped in favor of the other.
- **NFR-P-03 disposition:** UNMEASURABLE, no credential exists in any verification
  environment that has touched this repository. Same class of gap as PRD §32 rows 28/32
  (M17's own DEC-4 disposition) — not a new M18 blocker.
- **A-INIT predecessor action (this session, same conversation, before M18 was entered):**
  fixed and verified, on real GitHub Actions, the `TestResolvePluginUser_UnknownNumericUID`
  macOS CI failure (commit `4977de7`, run `38041177134`, both OS legs green). This was the
  entry-gate precondition the founder's task set for opening M18 — recorded here because it
  is the fact that made this A-INIT possible, even though it is not itself an M18 task.

## Execution record (appended during B-BUILD/C-VERIFY)
- A-INIT (this document, 2026-10-10): module materialized (six files compiled against
  IMP §27.M18, IKB §4 row 18, PRD §31/§32, CONTRA-2) + 5 cards (C1–C4, V1) written READY.
  No card dispatched yet.
