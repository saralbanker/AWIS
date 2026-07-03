# M00 — Validation Checklist (all binary; all must pass)
Verified by awis-verifier card M00-V1 on clean clone at HEAD `1042939` (2026-07-03).
- [x] Clean clone → `make build` exits 0
- [x] Clean clone → `make test` exits 0 (zero tests is acceptable; harness must run)
- [x] Clean clone → `make lint` exits 0 with pinned golangci-lint (2.12.2)
- [x] `make race contract e1 bench release-dry` targets exist (stubs exit 0 with a NOT-YET message; `race` is real, not a stub)
- [ ] CI green on Linux AND macOS on `main` — **PENDING first push: no GitHub remote configured yet; workflow statically verified (matrix ubuntu+macos, e1 job slot present); local Linux equivalents green**
- [x] `go.work` lists both modules; `cd apps/oip && go build ./...` succeeds independently
- [x] Compile-error proof: a scratch file in `apps/oip` importing `github.com/awis/awis/internal/x` FAILS to build (delete after demonstrating; record output in PR) — error: `use of internal package github.com/awis/awis/internal/scratch not allowed`; transcript in HANDOFF actuals
- [x] `.gitignore` blocks: `*.db*`, `.awis/`, `.decisions/.index/`, venvs, Go artifacts
- [x] `docs/edr/edr-001-module-path.md`, `edr-002-cli-framework.md`, `edr-003-sqlite-driver.md` committed (+ edr-004-aeo-model-mapping.md)
- [x] Root `README.md` states the platform boundary in one paragraph
- [x] No product code, no schema, no type definitions anywhere (only two comment-only doc.go files)
- [x] Global DoD (IMP §24) items 1, 2, 6, 7 verified (3–5 vacuous at M00) — item 2 macOS leg pending CI, Linux verified; item 7 (merge approval) pending human squash-merge
