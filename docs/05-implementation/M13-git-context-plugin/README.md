# M13 — git-context-plugin (reference plugin)
**Status:** Partitioned — materialize at entry · **Effort:** 1d · **Window:** Week 4 (parallel tail)
**Objective:** Python reference plugin: git.context.assemble + git.diff.fetch; pinned deps; per-plugin venv (PR-5 mitigation).
**Depends on:** M12 · **Blocks:** M15
**Primary sources:** IMP §27.M13; Blueprint §11 manifest example; PRD FR-PS-13 · **Compilation spec:** IKB §4/M13
**Key ACs:** offline plugin-harness tests pass; real call against this repo's git history succeeds.
