# M13 — Implementation Spec
**Canonical sources:** IMP §27.M13; Blueprint §11 (the awis-plugin.yaml example IS this
plugin's manifest — verbatim oracle); PRD FR-PS-13; TDS-05 `docs/PLUGIN_PROTOCOL.md`;
M12 HANDOFF (awis-plugin lib + mock_request + Manager.Register); IMP §25 PR-5 (Python deps →
pinned deps, per-plugin venv).

## Scope
`plugins/git-context-plugin/`: the reference Python plugin implementing `git.context.assemble`
and `git.diff.fetch` with `awis_plugin`; offline pytest via `mock_request`; Go e2e calling it
from a harness workflow; real-repo integration proof (this repository's git history); plugin
README (DoD) with the per-plugin venv install path (PR-5).

## CE pins
- **Zero Python runtime dependencies** — git operations shell out to the `git` binary
  (`subprocess.run`, list argv, no shell). This satisfies "pinned deps" trivially and keeps
  PR-5 mitigation documentation honest (README still documents the venv flow for plugins that
  DO have deps).
- **Capability outputs:**
  - `git.context.assemble(repo_path, ref)` → `context` object:
    `{ref, sha, author, email, date (ISO), message, parents[], files[{path,status}],
    stats{files_changed, insertions, deletions}}` (from `git show --numstat --format=…`).
  - `git.diff.fetch(repo_path, from_ref, to_ref)` → `diff` (unified diff string,
    `git diff from..to`).
- **Errors:** invalid repo/ref → raised exception carrying the git stderr → awis_plugin maps
  to `handler_error` (data.code) — no custom error protocol.
- **Manifest** `plugins/git-context-plugin/awis-plugin.yaml`: Blueprint §11 example verbatim
  EXCEPT `runtime.command: python3` and `runtime.args: ["-m", "git_context_plugin"]`
  unchanged; `GIT_TERMINAL_PROMPT: "0"` env kept; `idle_timeout_s: 300` kept. (PYTHONPATH for
  spawn is supplied by the caller's manifest env at registration/e2e time — the checked-in
  manifest stays canonical; the e2e writes a temp manifest with absolute PYTHONPATH, the
  established C4 pattern.)
- Package layout: `plugins/git-context-plugin/git_context_plugin/__init__.py` + `__main__.py`
  (so `-m git_context_plugin` serves), `tests/`, `pyproject.toml` (zero deps), `README.md`.

## 1. Plugin + offline tests (M13-C1)
Implementation per pins; pytest: temp git repo fixture (git init/config/commit via
subprocess); mock_request for both capabilities (happy + bad-ref + bad-repo); manifest parses
(reuse awis_plugin manifest reader if present, else assert via yaml stdlib-absent → just Go
side covers manifest validity — keep pytest to capability behavior); README (manifest
authoring pointer, venv install: `python3 -m venv .venv && .venv/bin/pip install -e .` +
awis-plugin path install, capability docs, TDS-05/PLUGIN_GUIDE citations).

## 2. e2e + real-history integration (M13-C2)
Go test `internal/plugin/e2e_gitcontext_test.go` (new file only): temp manifest (absolute
PYTHONPATH covering python/awis-plugin + plugins/git-context-plugin), Manager.Register,
harness workflow: plugin step `handler: git-context-plugin` (the frozen §7 fixture form —
exercises the input-key-set resolution rule with TWO capabilities registered) with inputs
{repo_path: <this repo>, ref: HEAD} → native step consuming `context`; assert sha/message
non-empty and resolution picked git.context.assemble. Second test: capability-id handler
`git.diff.fetch` with two refs from this repo's history (use `git rev-parse HEAD~1/HEAD`,
skip if history too shallow). t.Skip without python3 or git.

## Non-scope
- No venv automation in code (README documentation only; `awis plugin install` deps flow is
  M14/M17 CLI).
- No new Go code outside the new test file; no changes to internal/plugin, awis_plugin,
  goldens, TDS-05; no Go/Python deps.
