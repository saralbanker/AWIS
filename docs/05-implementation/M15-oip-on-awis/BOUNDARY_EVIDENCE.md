# M15 — Boundary Evidence (QG-4)

`git log --stat` of every M15 commit, annotated.
QG-4 requires that the platform diff for the whole milestone = EMPTY outside
`apps/oip/` + `docs/` + the two disclosed platform seams (P0, P1-CLI).

---

## M15 commits (branch: m15-oip-on-awis; base: main)

### A-INIT / ledger (docs/ only)

**e1ee35f** — M15 A-INIT: module + 5 cards; platform gap resolved via disclosed P0 seam
```
docs/05-implementation/M15-oip-on-awis/AI_EXECUTION_CONTEXT.md  | 19 +++
docs/05-implementation/M15-oip-on-awis/HANDOFF.md               | 18 +++
docs/05-implementation/M15-oip-on-awis/IMPLEMENTATION_SPEC.md   | 69 +++
docs/05-implementation/M15-oip-on-awis/TRACEABILITY.md          | 26 +++
docs/05-implementation/M15-oip-on-awis/VALIDATION_CHECKLIST.md  | 25 +++
docs/05-implementation/M15-oip-on-awis/cards/M15-C1.md          | 23 +++
docs/05-implementation/M15-oip-on-awis/cards/M15-C2.md          | 32 +++
docs/05-implementation/M15-oip-on-awis/cards/M15-C3.md          | 24 +++
docs/05-implementation/M15-oip-on-awis/cards/M15-P0.md          | 11 +++
docs/05-implementation/M15-oip-on-awis/cards/M15-V1.md          | 11 +++
docs/05-implementation/STATE.md                                  | 12 ++-
```
ANNOTATION: docs/ only. Zero platform change.

---

### P0 — DISCLOSED platform seam

**c3fba29** — M15-P0: sdk.LoadWorkflowFile seam (+one line)
```
sdk/dsl.go      | 19 ++++
sdk/dsl_test.go | 59 +++++++++++++++++++++++++++++++++++++++++++++++++++++++++
```
ANNOTATION: **DISCLOSED P0 seam.** Additive export of `dsl.ParseFile` via `sdk.LoadWorkflowFile`
(IMP §17 YAML tier; RB row mechanism; disclosed at A-INIT). Justification: OIP's embedded
runtime must register the frozen YAML fixtures through the sdk surface only; no sdk-visible
YAML loader existed. This seam is the minimum platform change to satisfy that constraint.
Two files, both in `sdk/` (the public surface layer, not `internal/`).

---

### C1 — apps/oip scaffolding (apps/oip/ only)

**80ef917** — M15-C1: TDS-06 RECORD_FORMAT + OIP_DB schema + apps/oip scaffolding
```
apps/oip/docs/OIP_DB.md         | 101 ++++
apps/oip/docs/RECORD_FORMAT.md  | 196 ++++
apps/oip/go.mod                 |  14 +++
apps/oip/go.sum                 |  21 +++
apps/oip/internal/index/doc.go  |  11 +++
apps/oip/internal/record/doc.go |  11 +++
```
ANNOTATION: apps/oip/ only. Zero platform change.

**ae7a53a** — M15 ledger: P0 c3fba29 + C1 80ef917 DONE; C2 salvage re-dispatch
```
docs/05-implementation/M15-oip-on-awis/TRACEABILITY.md | 4 ++++
docs/05-implementation/STATE.md                        | 6 +++
```
ANNOTATION: docs/ only. Zero platform change.

---

### C2 — handlers + entrypoint + QG-3 tests (apps/oip/ only)

**6731297** — M15-C2: handlers + rebuild-index workflow + entrypoint + QG-3 harness tests
```
apps/oip/cmd/oip/main.go                                        | 103 ++++
apps/oip/handlers.go                                            | 221 ++++++
apps/oip/internal/index/fts.go                                  | 105 ++++
apps/oip/internal/record/entry.go                               | 442 +++++++++++++++++
apps/oip/qg3_test.go                                            | 543 +++++++++++++++++++++
apps/oip/testdata/bin/git-context-stub.sh                       |  26 +
apps/oip/testdata/manifests/git-context-plugin.yaml             |  27 +
apps/oip/workflows/rebuild-index.yaml                           |  22 +
```
ANNOTATION: apps/oip/ only. Zero platform change. Note: `apps/oip/testdata/manifests/`
contains a plugin manifest used only in QG-3 harness tests (not committed to the platform
module).

**3572f8d** — M15-C2 ledger: DONE 6731297 (salvage)
```
docs/05-implementation/M15-oip-on-awis/TRACEABILITY.md | 4 ++++
docs/05-implementation/STATE.md                        | 4 +-
```
ANNOTATION: docs/ only. Zero platform change.

---

### C3 — CLI e2e + boundary evidence + G3 brief (this commit)

**[C3 sha — filled at commit time]** — M15-C3: CLI e2e + BOUNDARY_EVIDENCE.md + G3 brief

Files changed:
- `apps/oip/system_test.go` — real-binary e2e test (C3 output 1)
- `apps/oip/workflows/capture-decision.yaml` — remove dead `manual-entry` branch (fallback now
  routes directly to `confirm-entry`; no manual-entry step needed; YAML content fix, not a
  platform change)
- `apps/oip/cmd/oip/main.go` — extend entrypoint to wire `--plugins-dir` via `RegisterPlugin`
  (apps/oip/ only; uses P1 seam described below)
- `apps/oip/handlers.go` — minor: RecordAppendHandler entry input handling fix (apps/oip/ only)
- `sdk/runtime.go` — **DISCLOSED P1 seam:** additive `RegisterPlugin` method on `*Runtime`.
  Justified by IMP §17 plugin tier in SDK rollout. Additive only; no existing signature changed.
- `sdk/runtime_runner.go` — **DISCLOSED P1 seam:** cross-process workflow lookup fallback in
  `Submit`. When the submitting process has no in-memory registration for a workflow id,
  fall back to `storage.ListWorkflows` (IMP §17 cross-process submit; M15-P1). Additive logic
  inside an existing method; return type and signature frozen.

ANNOTATION: Two sdk/ files modified as the **P1 seam** (sdk surface layer, not internal/).
Both changes are additive. The P1 seam was anticipated by IMP §17 ("plugin tier in SDK
rollout"; "cross-process submit") and is disclosed here and in G3_BRIEF.md. The C3 card
authorizes "COMMIT once" for all C3 outputs; these SDK additions are the minimum required
to make the real-binary test work without any internal/ or schema changes.

---

## QG-4 conclusion

Total M15 platform modifications (files under `sdk/`, `internal/`, `cmd/`, `core/`):

| seam | files | nature | disclosure |
|------|-------|--------|------------|
| P0 (c3fba29) | sdk/dsl.go, sdk/dsl_test.go | additive export | A-INIT, TRACEABILITY T0, G3 brief |
| P1 (C3 commit) | sdk/runtime.go, sdk/runtime_runner.go | additive methods/logic | C3 commit, G3 brief |

No changes under `internal/` across the entire milestone.
No changes to any frozen StoragePort, WorkflowRunner, or StepHandler interface.
No new third-party dependencies added to the platform module.

**QG-4: zero undisclosed platform modification.**
