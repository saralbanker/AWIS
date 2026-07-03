# M00 — Implementation Specification
**Authority:** IMP §4, §5, §21, §27.M0. This spec transcribes; it does not design.

## Objective
An empty-but-green repository: skeleton only, trivially releasable, CI passing from the first commit. Nothing else exists after M00.

## Tasks (ordered, from IMP §4)
1. **Repo init.** `git init`; default branch `main`; `.gitignore` covering Go build artifacts, SQLite artifacts (`*.db`, `*.db-wal`, `*.db-shm`), `.awis/`, `.decisions/.index/`, Python venvs.
2. **Two-module Go workspace.** `go.work` tying:
   - `github.com/awis/awis` (repo root) — platform: runtime, CLI, SDK package. Module path per CONTRA-1 disposition: one module, public package `github.com/awis/awis/sdk`.
   - `github.com/awis/oip` at `apps/oip/` — empty scaffold (go.mod + doc.go only) until M15. Separate module path makes importing `awis/internal/*` a compile error (mechanical QG-4 enforcement).
3. **Makefile** targets: `build`, `test`, `lint`, `race`, `contract`, `e1` (zero-AI gate — stub until M6), `bench`, `release-dry`.
4. **CI skeleton** (GitHub Actions, `.github/workflows/ci.yml`): fmt/vet/lint + build + test on Linux and macOS. Red-to-green from the first commit.
5. **Docs skeleton.** `docs/` (TDS pack lands in M01), `CODEBASE.md` stub, `README.md` stating the platform boundary in one paragraph (AWIS = runtime; OIP = first application; boundary is the SDK).
6. **Toolchain pinning.** `go.mod` toolchain directive; `golangci-lint` config; `pyproject.toml` stubs for `python/awis-step/` and `python/awis-plugin/`.
7. **EDR decisions (required by IMP §27.M0 risk note).** Decide NOW and record in `docs/edr/`:
   - `edr-001-module-path.md` — the CONTRA-1 disposition (transcribe from IMP §25).
   - `edr-002-cli-framework.md` — CLI framework choice.
   - `edr-003-sqlite-driver.md` — `modernc.org/sqlite` (pure Go, CGO disabled) per frozen architecture; record WAL-mode default.

## Scope walls (FORBIDDEN in M00)
- No product code, no types, no schemas (those are M01 under Gate G1).
- No third-party dependencies beyond the linter and (pinned, unused-yet) SQLite driver entry.
- No directory beyond IMP §5's enumeration.

## Acceptance criteria
- `make build test lint` passes on a clean clone.
- CI green on both OSes on `main`.
- EDR notes committed (DoD addition).

## Merge / Rollback
**Merge:** CI green. **RB:** n/a — first commit. **Repo after:** skeleton only, releasable trivially.
