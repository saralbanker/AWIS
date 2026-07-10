# M13 — Traceability
| T | Task | Source coordinates | Card |
|---|---|---|---|
| T1 | `git_context_plugin` package: git.context.assemble + git.diff.fetch (zero deps, git argv subprocess) | FR-PS-13; Blueprint §11 capabilities; SPEC pins | C1 |
| T2 | Manifest `awis-plugin.yaml` (Blueprint §11 verbatim modulo pinned command/args) | Blueprint §11; FR-PS-02 | C1 |
| T3 | Offline pytest via mock_request (temp git repo fixture; happy + error paths) | FR-PS-15; IMP §27.M13 Val | C1 |
| T4 | Plugin README incl. per-plugin venv install path | IMP §27.M13 DoD; PR-5 | C1 |
| T5 | Go e2e: plugin-name handler (resolution rule) + capability-id handler vs THIS repo's history | IMP §27.M13 Val ("real call"); frozen §7 fixture form | C2 |

## Notes / dispositions (CE, A-INIT 2026-07-10)
- Zero-dep pin: git binary via subprocess satisfies "pinned dependencies" (PR-5) trivially;
  venv flow remains documented in README for dep-bearing plugins.
- Checked-in manifest uses `python3` (not Blueprint's `python`) — environments per QG-1
  (macOS/Linux) ship `python3`; recorded as the only manifest delta.
- `context` object shape is a V1 pin (SPEC); OIP (M15) consumes it opaquely (`{type: object}`
  in the frozen fixture), so the shape is not frozen surface.

## Execution record (appended during B-BUILD/C-VERIFY)
- C1 (76574ff, 2026-07-10): git_context_plugin package (2 capabilities, zero deps, argv git);
  manifest Blueprint-verbatim modulo python3 delta; 17 offline pytest vs temp git repo; README
  with PR-5 venv path; Makefile third suite. pytest 41 total green; all gates green.
- C2 (080a367, 2026-07-10): e2e_gitcontext_test.go — plugin-name resolution form proven against
  THIS repo's history (context.sha/message real); capability-id diff form proven (1503-byte
  diff HEAD~1..HEAD). Diff = one new test file. All gates green. No deviations.
