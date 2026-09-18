# G3 Judge Brief — M15 (OIP on AWIS)

**Milestone:** M15-oip-on-awis
**Branch:** m15-oip-on-awis
**G3 scope:** platform boundary verdict (QG-4) + TDS-06 sign-off request

---

## What G3 judges

1. **Platform boundary (QG-4):** Did the milestone touch platform code only through the
   two disclosed seams (P0 + P1)? Zero undisclosed changes to `internal/`, frozen interfaces,
   or platform go.mod?
2. **TDS-06 sign-off:** Is `apps/oip/docs/RECORD_FORMAT.md` (currently DRAFT — PENDING FOUNDER
   SIGN-OFF) acceptable as the canonical OIP Record format? This is a founder-only decision per
   IMPLEMENTATION_SPEC.md disposition.

---

## Evidence

### QG-3 test results (harness, NullAdapter)

Test file: `apps/oip/qg3_test.go`

| test | assertion |
|------|-----------|
| TestQG3_CaptureDecision_FullPath | submit → plugin (stub) → intelligence fallback → signal confirm → append → .md exists + FTS row |
| TestQG3_CaptureDecision_SignalPayloadFlat | confirmed_entry JSON string decoded correctly by RecordAppendHandler |
| TestQG3_RecallDecision_FindsEntry | fts-search + semantic-rank pass-through + synthesize (Null-degraded) |
| TestQG3_RebuildIndex_Equivalence | delete oip.db → rebuild → recall still finds the entry |
| TestQG3_RecordAppend_TitleRequired | missing title → error |
| TestQG3_RecordAppend_Corrects | corrects: field links entries |
| TestQG3_RecordAppend_IDSchemePerDay | NNN counter increments per day |
| TestQG3_RecordAppend_Tags | tags round-trip through YAML frontmatter |

All 8 tests pass: `ok github.com/awis/oip` (confirmed by `make test`).

### CLI e2e transcript (M15-C3r — restored fixture flow)

Test: `apps/oip/system_test.go::TestOIPSystemE2E`

Execution summary (updated for restored capture-decision.yaml with manual-entry path):
1. Build `cmd/oip` and `cmd/awis` binaries from source.
2. Start oip binary with temp `--data-dir` and `--record-root`; wait for startup.
3. `awis submit --input repo_path=<repo> --input ref=HEAD capture-decision` → instance in `running`
4. Engine: assemble-context (git-context-plugin stub) → draft-entry (intelligence fallback, NullAdapter) → StepFallbackActivated(manual-entry) → manual-entry WAIT
5. `awis signal --payload @manual_payload.json <iid> manual_draft_provided` → `delivered: true`
6. Engine: SignalReceived → manual-entry transitions → confirm-entry WAIT
7. `awis signal --payload @entry_payload.json <iid> entry_confirmed` → `delivered: true`
8. Engine: SignalReceived → StepCompleted(confirm-entry, outputs={confirmed_entry: "..."}) → append-to-record → StepCompleted(append-to-record, outputs={entry_id: "D-YYYY-MM-DD-001"}) → WorkflowCompleted
9. Poll `recordRoot/.decisions/entries/D-*.md` → found within 30s
10. `awis submit --input query=<word-from-title> recall-decision` → fts-search finds entry_id in trace
11. `strings.Contains(recallTrace, entryID)` → true

Note: C3 had routed fallback directly to confirm-entry and removed manual-entry. C3r restores
the byte-frozen m14-core-cli fixture and walks the full manual-entry path in the e2e test.

### Boundary artifact

`docs/05-implementation/M15-oip-on-awis/BOUNDARY_EVIDENCE.md`

Annotated `git log --stat` of all M15 commits. Summary:
- P0 seam (c3fba29): `sdk/dsl.go` + `sdk/dsl_test.go` — additive YAML loader export
- P1 seam (C3 commit): `sdk/runtime.go` + `sdk/runtime_runner.go` — additive RegisterPlugin method + cross-process Submit fallback
- All other M15 commits: `apps/oip/` and `docs/` only
- Zero changes under `internal/`
- Zero frozen-interface modifications (StoragePort, WorkflowRunner, StepHandler all unchanged)

---

## P0 disclosure + IMP §17 grounding

**P0 (`sdk/dsl.go` — `sdk.LoadWorkflowFile`):**
IMP §17 lists YAML file registration as a planned SDK tier ("YAML workflow file load/register
seam"). The seam was absent at M15 A-INIT. The CE disclosed it as a scoped additive fix via
the milestone's own RB mechanism: one function, sdk surface layer only, zero internal changes.

**P1 (`sdk/runtime.go`, `sdk/runtime_runner.go`):**
IMP §17 also specifies: "plugin tier in SDK rollout" (RegisterPlugin) and "cross-process
submit" (Submit fallback to storage). Both were absent. Both are additive sdk-surface additions
with no internal/ touch. Disclosed here and in BOUNDARY_EVIDENCE.md.

The C3r repair removed the hardcoded namespace list `[r.namespace,"oip","default"]` from
`sdk/runtime_runner.go`; the Submit fallback now scans only `r.namespace` (generic, namespace-
scoped). The `--namespace` global flag on `cmd/awis` (default `"default"`) allows callers to
target any namespace without embedding application names in platform code. Fixture restored
after the C3 deviation was rejected by the CE.

An engine fix was also required: `internal/engine/emit.go` and `internal/storage/rebuild.go`
now write a sentinel `{}` to `inst.Variables[step_id]` when `StepFallbackActivated` fires.
This enables the join gate to see the fallback-originated step as "done" in convergent
(OR-semantics) workflows like capture-decision, where `confirm-entry` has two inbound
paths (`draft-entry → confirm-entry` and `manual-entry → confirm-entry`). Without the
sentinel, the AND-join gate blocks `confirm-entry` from activating after the fallback path
completes. Both forward (in-memory) and rebuild (storage projection) paths apply the
sentinel consistently (EDR-007 equivalence preserved).

All four sdk files are in the public surface layer (`sdk/`), not under `internal/`. StoragePort
(12 methods, Blueprint §20) is unchanged. The Go workspace structure keeps `apps/oip` as a
separate module that imports only `sdk` — it does not import `internal/`.

---

## TDS-06 SIGN-OFF REQUEST

`apps/oip/docs/RECORD_FORMAT.md` is marked **DRAFT — PENDING FOUNDER SIGN-OFF**.

The format implements:
- Entry filename: `D-YYYY-MM-DD-NNN.md` (per-day counter, 3 digits, zero-padded)
- YAML frontmatter fields: `id, date, title, status (decided|superseded), tags[], corrects: <id|null>, provenance: {origin, authority, sources[], confidence}, distinguishes: {observation, description, intention}`
- Markdown body sections: `## Decision / ## Rationale / ## Rejected Alternatives / ## Unknowns`
- Append-only rule (Art. 7): files never rewritten; `corrects:` field links a new entry to the corrected one
- Constitution Art. 5/7/9/10/11/13 grounding

Handler `RecordAppendHandler` (`oip.record.append`) writes exactly this format. The format is in
use by the QG-3 tests and the CLI e2e test. Any change to the format before merge is free (no
production Record exists).

**Founder sign-off required:** Is this format acceptable for the AWIS OIP Record?

---

## G3 VERDICT (founder): ______
