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
- B-BUILD (2026-10-11; Opus CE orchestration, Haiku 5.5 execution per founder MODEL-DIRECTIVE):
  C1 16cf735 · C2 WITHDRAWN (stale QG-1 premise) → C2r f895a12 · C3 e15497b · C3r 99f7b13 ·
  C3r2 34894e7 (release.yml token hardening, from background security review) · C4 54fcf15
  (fix-only: stmt cache → NFR-P-01; buffered trace → NFR-P-05 single-instance) · C4r 3addeb9
  (trace adapter/tokens suffix, PRD §32 row 41) · C5 29cb521 (V1 remediation, below).
  CE rulings and per-card review evidence: STATE.md M18 block.

## M18-V1 verification report (run 1, awis-verifier/Haiku 5.5, clean worktree @ 4092406)
Recorded by CE from the verifier's verbatim report (the verifier's identity forbids doc edits,
so it did not write this file; see STATE.md V1-RESULT). VERDICT: **FAIL**.

| Row | Verdict | Evidence (verifier-measured) |
|---|---|---|
| QG-1 | PASS | linux `TestQG1RehearsalCompletesHelloWorld` 249ms, status completed; macOS real CI job 114508620323 `ok github.com/awis/awis/cmd/awis 42.292s` (no skip/GOOS gate) |
| QG-2 | STOP | founder-administered human-subject measurement |
| QG-3 | FAIL | local `make e1` 8/8 PASS; CI 38152488419 ubuntu E1 success, macOS E1 **skipped** (job failed at Test) |
| QG-4 | STOP | 9 disclosed platform paths in M15 range (P0 c3fba29, P1 0ef33c8, C3r 0e452f8); `TestOIPSystemE2E` + 8 `TestQG3_*` PASS at HEAD; verifier declines to adopt G3 founder ruling — founder confirms |
| QG-5 | PASS | `TestQG5_WaitSignalWithin1s` 0.01s; `TestCheckpointM9_OIPCaptureWorkflow` 0.01s; macOS CI sdk/testing ok |
| NFR-P-01 | PASS | P99 4.792 / 4.668 / 4.533ms (<5ms) — MATCH |
| NFR-P-02 | PASS | P95 1.143 / 1.177 / 1.160ms (gate ≤50ms; target <10ms informational) — MATCH-ON-VERDICT |
| NFR-P-03 | UNMEASURABLE | no `ANTHROPIC_API_KEY`; `TestLiveSmoke` SKIP |
| NFR-P-04 | PASS | max 99.594 / 99.812 / 99.550ms (<200ms) — MATCH-ON-VERDICT |
| NFR-P-05 | PASS | multi max 2.082 / 2.414 / 1.832ms; single max 464.648 / 461.568 / 469.400ms (<500ms) — MATCH-ON-VERDICT |
| NFR-P-06 | PASS | max 404.998 / 409.603 / 407.595ms (<30s) — MATCH |
| NFR-P-07 | PASS | max 101.742 / 101.421 / 101.064ms (<300ms) — MATCH |
| NFR-P-08 | PASS | max 40.593 / 41.608 / 42.252ms (<3s) — MATCH |
| NFR-P-09 | PASS | binding max 5.502 / 6.645 / 5.661ms (<50ms); floor 0.72–0.75ms — MATCH |
| make bench | PASS | 3/3 exit 0, eight benchmarks |
| make release-dry | PASS | 4 archives; `file`: Mach-O x86_64/arm64, ELF x86-64/aarch64 static; `(cd dist && sha256sum -c checksums.txt)` OK×4; snapshot `awis version 0.0.0-SNAPSHOT-4092406` |
| release notes | PASS | throwaway clone, push URL nonexistent, token unset, `GORELEASER_CURRENT_TAG=v1.0.0`: `awis version 1.0.0`; 100 `M##` entries (41 bookkeeping — backlog M18-B1) |
| V1_BENCHMARKS.md | PASS | all rows present (stale macOS-pending sentence → fixed in C5) |
| PRD §32 re-measure | FAIL | row 14 SAT→not satisfied as written, not called out; row 13 STOP (PRD vs TDS-07) → recorded in C5 |
| fix-only discipline | PASS | C1 misses fixed by scoped C4 diffs, re-measured PASS 3/3 |
| frozen-surface diff | PASS | `git diff main...4092406`: 46 files, no violation; no existing test modified |
| `make verify` / `contract` / `e1` | PASS | exit 0 each |
| `git tag -l 'v*'` | PASS | empty |

FAILs: (1) CI 38152488419 macOS Test: `--- FAIL: TestDeliverSignal_CrashPreCommit … 13 signal(s)
marked delivered, want 0` (pre-existing M07 test race; main 4977de7 also 5/30 under load);
(2) consequent macOS E1 skipped; (3) PRD row 14; (4) card text: checksum command must run in
dist/; (5) card text omits AWIS-E1 by name. PRD §32 at 4092406 per verifier: 50 SAT · 3 NOT
(14, 37, 40) · 2 PARTIAL (41, 44) · 3 UNMEASURABLE (5, 28, 32) · 1 STOP (13).
CE disposition: STATE.md CE-CLASSIFICATION; remediation card M18-C5 (29cb521).
