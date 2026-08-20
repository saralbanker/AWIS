# M13 — Validation Checklist (binary; exit list)
Global DoD (IMP §24) + M13 rows (IMP §27.M13; FR-PS-13). Verifier executes via cards/M13-V1.md.

- [x] V-COMMON block all ✅ (`make build test lint race` + `make e1`; clean tree)
- [x] `make pytest` green — now THREE suites (awis-step, awis-plugin, git-context-plugin)
- [x] `plugins/git-context-plugin/awis-plugin.yaml` matches Blueprint §11 example verbatim
      modulo the recorded command/args delta; parses via internal/plugin.ParseManifest
- [x] Both capabilities implemented with zero Python deps (pyproject dependencies = [])
- [x] Offline pytest: mock_request happy + bad-ref + bad-repo for both capabilities against a
      temp git repo fixture (no network, no AWIS runtime)
- [x] Go e2e: `handler: git-context-plugin` (plugin-name form) resolves via input-key-set to
      git.context.assemble and returns real context from THIS repo (sha/message non-empty)
- [x] Go e2e: `git.diff.fetch` capability-id handler returns a real diff between two refs of
      this repo (skips gracefully on shallow history / missing git/python3)
- [x] Plugin README exists: capabilities, manifest, offline testing, per-plugin venv install
      path (PR-5), cites TDS-05 + PLUGIN_GUIDE
- [x] No changes outside plugins/git-context-plugin/, the new Go e2e test file, Makefile
      pytest extension, and module docs — diff proof
- [x] No new Go dependency; no Python dependency; no existing test modified
