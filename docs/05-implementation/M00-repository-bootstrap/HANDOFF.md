# M00 → M01 Handoff
**Status: MERGED (`0277c9d`)**** — the *Guaranteed outputs* below are the contract M01 may rely on;
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

## Actuals (drafted at execution 2026-07-03; merge commit filled at merge)
- Merge commit: `0277c9d` (squash of `m00-repository-bootstrap`; work `1042939`, record `79436ed`; approved by founder 2026-07-03)
- Toolchain (environment, not repo): Go 1.26.4 + golangci-lint 2.12.2, user-installed at `~/toolchains/`
- EDR decisions: edr-001 CONTRA-1 module path; edr-002 CLI = spf13/cobra (require lands at M14); edr-003 modernc.org/sqlite + WAL default (require lands at M02); edr-004 AEO model mapping
- Boundary proof (executed twice — builder, then independent verifier on a clean clone):
  `cd apps/oip && go build ./...` → `boundarycheck.go:3:8: use of internal package github.com/awis/awis/internal/scratch not allowed` (exit 1); scratch files deleted; post-cleanup builds green
- Verification: awis-verifier card M00-V1 → VERDICT PASS, 12/12 items (CI-on-GitHub item static-only), HEAD `1042939`
- Deviations from spec: (1) CI green on GitHub Linux+macOS unverifiable locally — no remote configured; pending first push. (2) `.claude/agents/` (4 AEO subagents) added — org tooling additive to IMP §5 tree, recorded in edr-004. (3) This session invoked builder/verifier as general-purpose agents pinned to Sonnet carrying the `.claude/agents` identities verbatim (named agents register at next session start). (4) CE trivial review patches: golangci-lint-action v6→v8, EDR-002 canonical command names. (5) Checklist ticks + this actuals draft written by CE directly (sub-trivial volume) rather than a scribe batch.
- Open issues created: none
