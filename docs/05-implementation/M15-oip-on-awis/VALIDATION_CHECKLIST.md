# M15 — Validation Checklist (binary; exit list)
- [ ] V-COMMON all ✅ (build/test/lint/race/e1; clean tree) + pytest regression
- [ ] P0: sdk diff = exactly LoadWorkflowFile (+test); grounded justification in TRACEABILITY
- [ ] TDS-06 exists with DRAFT—PENDING FOUNDER SIGN-OFF banner; format encodes Art. 7
      (corrects:, never rewrite), Art. 9 (provenance block), Art. 10 (distinguishes block),
      Art. 11 (rationale/rejected sections), Art. 13 (unknowns section); ID scheme
      D-YYYY-MM-DD-NNN
- [ ] apps/oip imports: `go list -deps` shows NO github.com/awis/awis/internal/* (sdk only)
- [ ] QG-4: milestone diff outside apps/oip + docs/ledger + P0-commit = EMPTY;
      BOUNDARY_EVIDENCE.md contains the git log --stat proof separating P0 from app commits
- [ ] Both frozen YAMLs registered byte-unchanged (sha256 vs m14 HEAD versions)
- [ ] RecordAppendHandler: .md written per TDS-06 + entries/entries_fts rows in one
      operation; append-only (second entry with corrects: links, never rewrites) — tests
- [ ] IndexFTSHandler returns FTS-ranked results; SemanticRankHandler pass-through unchanged
      (CONTRA-3) — tests
- [ ] rebuild-index: delete oip.db → rebuild from .md files → recall still finds entries
- [ ] QG-3: capture-decision AND recall-decision complete in harness with NullAdapter
      (fallback/degraded paths); WAIT step consumes entry_confirmed signal
- [ ] CLI e2e: real binary submit → signal entry_confirmed → entry file appended → recall
      workflow finds it (evidence transcript)
- [ ] G3 brief exists (docs/05-implementation/M15-oip-on-awis/G3_BRIEF.md): boundary evidence,
      P0 disclosure, TDS-06 sign-off request, QG-3/QG-4 results — verdict line EMPTY (founder)
- [ ] oip.db lives at .decisions/.index/oip.db, git-ignored; entries/*.md in git (layout per
      Finalization B5)
- [ ] No new platform Go dep; apps/oip go.mod may add modernc.org/sqlite only
