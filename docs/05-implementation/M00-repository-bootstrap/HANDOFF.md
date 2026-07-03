# M00 → M01 Handoff
**Status: PENDING EXECUTION** — the *Guaranteed outputs* below are the contract M01 may rely on;
the *Actuals* section is completed when M00 merges (per 05-implementation/README.md policy).

## Guaranteed outputs (contract)
- Green `main` with two-module workspace (`github.com/awis/awis`, `github.com/awis/oip`)
- Working `make build/test/lint`; CI on Linux+macOS; all eight Makefile targets present
- `docs/` + `docs/edr/` directories with edr-001..003 recorded
- QG-4 boundary mechanically demonstrated (compile-error proof in PR record)

## What M01 may assume
- It can add `sdk/` package + `docs/EVENTLOG_FORMAT.md`, `WORKFLOW_SCHEMA.md`, `EXPRESSION_GRAMMARS.md` without touching CI or Makefile.
- The SQLite driver decision (edr-003) is settled; TDS-01 may reference WAL semantics.

## Known limitations (by design)
- `e1`, `contract`, `bench` targets are stubs. No executable behavior exists anywhere.

## Actuals (fill at merge)
- Merge commit: _
- Deviations from spec: _
- Open issues created: _
