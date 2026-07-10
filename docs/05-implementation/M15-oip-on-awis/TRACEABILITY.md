# M15 — Traceability
| T | Task | Sources | Card |
|---|---|---|---|
| T0 | Platform seam: `sdk.LoadWorkflowFile` (re-export of dsl.ParseFile) | IMP §17 L324 (YAML tier in SDK rollout); Blueprint §12 (both paths → same struct); RB row mechanism | P0 |
| T1 | TDS-06 `apps/oip/docs/RECORD_FORMAT.md` (DRAFT banner; Art. 7/9/10/11/13 rules; ID scheme; corrects:) | Constitution Title II; Blueprint §10/§18; IMP §12 | C1 |
| T2 | apps/oip module scaffolding + oip.db schema (entries/entries_fts/entry_vectors) | Finalization B5 Option A verbatim | C1 |
| T3 | RecordAppendHandler (md file + db index, one operation) | Blueprint §29; B5; Art. 5/7 | C2 |
| T4 | IndexFTSHandler + SemanticRankHandler (pass-through, CONTRA-3) | Blueprint §29; CONTRA-3 | C2 |
| T5 | rebuild-index workflow + handler (index disposable, Record irreplaceable) | Art. 5; IMP §27.M15 Obj | C2 |
| T6 | OIP entrypoint (embedded runtime; sdk-only; registers YAMLs unchanged via T0 seam) | IMP §27.M15 Obj; Blueprint §12 Mode 1 | C2 |
| T7 | QG-3 harness tests (both workflows + rebuild equivalence, NullAdapter) | QG-3; IMP §27.M15 Val | C2 |
| T8 | CLI e2e (submit → signal entry_confirmed → append → recall finds it) | IMP §27.M15 Val | C3 |
| T9 | BOUNDARY_EVIDENCE.md (git log --stat proof) + G3 brief | QG-4; IMP §20.M15; O-2 | C3 |

## Dispositions (CE, Fable, 2026-07-10)
- **Platform gap found at A-INIT:** no sdk-visible YAML loading; OIP's embedded runtime
  cannot register the frozen YAML files sdk-only. Resolution per the milestone's own RB
  mechanism: scoped platform fix P0 (`sdk.LoadWorkflowFile`, additive, one function),
  justified by IMP §17 (YAML tier listed in SDK rollout) — the boundary moves, not the
  application. Disclosed as its own commit + G3-brief line item for the founder's verdict.
- **TDS-06 sign-off + G3 are founder-only:** built at-risk under the 2026-07-10 directive;
  DRAFT banner stays until sign-off; E-MERGE brief carries both.
- entry_vectors table created but EMPTY in V1 (CONTRA-3); semantic-rank passes FTS order
  through unchanged.

## Execution record
- P0 (c3fba29) / C1 (80ef917), 2026-07-10: sdk.LoadWorkflowFile seam (2 files, disclosed);
  TDS-06 DRAFT + OIP_DB.md + apps/oip scaffolding (modernc sqlite dep, OIP-internal stubs,
  .decisions/.index gitignore). C2 run interrupted mid-card; uncommitted partials on tree →
  P2 salvage re-dispatch (EEOS §15 D2).
