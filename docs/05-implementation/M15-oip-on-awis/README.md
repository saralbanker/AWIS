# M15 — OIP on AWIS  → GATE G3 (platform boundary verdict)
**Status:** Partitioned — materialize at entry · **Effort:** 4d · **Window:** Week 5 · **Phases:** A, B, C
**Phase A:** TDS-06 (apps/oip/docs/RECORD_FORMAT.md — OIP's irreversible artifact; HUMAN SIGN-OFF before any handler code; Constitution Articles 5–13, append-only, corrects: linkage).
**Phase B:** apps/oip module (separate go.mod — github.com/awis/oip): RecordAppendHandler (.md file + oip.db entries/FTS in one operation), IndexFTSHandler, SemanticRankHandler (pass-through when no vectors — CONTRA-3), oip.db mini-migrations, rebuild-index workflow. [O-5] add apps/oip to CI matrix.
**Phase C:** capture-decision + recall-decision YAML exactly per Blueprint §7/§10; register via sdk ONLY; end-to-end via harness AND CLI (submit → signal entry_confirmed → entry appended → recall finds it); produce G3 boundary evidence — [O-2] platform diffs categorized defect-fix vs surface-change; only surface-change fails QG-4.
**Depends on:** M09, M10, M13, M14 · **Blocks:** M18 · **Gate:** G3
**Primary sources:** IMP §27.M15; Blueprint §10/§18; Finalization Blockers 5–6; Constitution Title II/III; PRD QG-3/QG-4 · **Compilation spec:** IKB §4/M15
