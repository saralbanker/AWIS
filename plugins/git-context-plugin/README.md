# git-context-plugin

AWIS reference plugin for git integration.  Implements two capabilities:
`git.context.assemble` and `git.diff.fetch`.  Requires only the `git` binary
on PATH — zero Python runtime dependencies.

**Protocol authority:** `docs/PLUGIN_PROTOCOL.md` (TDS-05)
**Developer guide:** `docs/PLUGIN_GUIDE.md`
**Manifest spec:** Blueprint §11

---

## Capabilities

### `git.context.assemble`

Assembles a rich context object for a given commit reference.

| Input | Type | Description |
|---|---|---|
| `repo_path` | string | Absolute or relative path to the git repository root |
| `ref` | string | Any git ref (branch name, tag, SHA, `HEAD`, `HEAD~1`, etc.) |

| Output | Type | Description |
|---|---|---|
| `context` | object | Full commit context (see shape below) |

**Context object shape:**

```json
{
  "ref":     "HEAD",
  "sha":     "a1b2c3d4...",
  "author":  "Alice",
  "email":   "alice@example.com",
  "date":    "2026-07-10T12:34:56+00:00",
  "message": "feat: add gamma and update beta",
  "parents": ["deadbeef..."],
  "files": [
    {"path": "beta.txt",  "status": "M"},
    {"path": "gamma.txt", "status": "A"}
  ],
  "stats": {
    "files_changed": 2,
    "insertions": 3,
    "deletions": 0
  }
}
```

Timeout: 30 000 ms.

---

### `git.diff.fetch`

Returns the unified diff between two refs.

| Input | Type | Description |
|---|---|---|
| `repo_path` | string | Absolute or relative path to the git repository root |
| `from_ref`  | string | Starting ref (exclusive lower bound of the diff range) |
| `to_ref`    | string | Ending ref (inclusive upper bound) |

| Output | Type | Description |
|---|---|---|
| `diff` | string | Unified diff produced by `git diff from_ref..to_ref` |

Timeout: 10 000 ms.

---

## Manifest (`awis-plugin.yaml`)

The manifest file declares the plugin identity, capabilities, and spawn
command.  It follows Blueprint §11 verbatim with one recorded delta:
`runtime.command` is `python3` (not `python`) for cross-platform
compatibility on Linux/macOS (QG-1).

```yaml
name: git-context-plugin
version: 1.0.0
runtime:
  command: python3
  args: ["-m", "git_context_plugin"]
  env:
    GIT_TERMINAL_PROMPT: "0"
  idle_timeout_s: 300
```

`GIT_TERMINAL_PROMPT: "0"` prevents interactive prompts when the process
receives no tty (NFR-S-02 environment isolation).

---

## Offline Testing with `mock_request`

The `awis_plugin.testing.mock_request` helper exercises the same dispatch
path as `plugin.serve()` without starting a subprocess (FR-PS-15).

```python
import git_context_plugin  # registers @plugin.capability handlers

from awis_plugin.testing import mock_request, PluginCallError

# Happy path
outputs = mock_request(
    "git.context.assemble",
    {"repo_path": "/path/to/repo", "ref": "HEAD"},
)
ctx = outputs["context"]
print(ctx["sha"], ctx["message"])

# Error path
try:
    mock_request(
        "git.context.assemble",
        {"repo_path": "/tmp/not-a-repo", "ref": "HEAD"},
    )
except PluginCallError as e:
    print(e.code)   # "handler_error"
    print(e.detail)
```

Run the test suite:

```sh
cd plugins/git-context-plugin
python3 -m pytest tests/ -q
```

---

## Per-plugin venv install (PR-5)

For plugins that carry Python dependencies, install into a dedicated venv:

```sh
cd plugins/git-context-plugin
python3 -m venv .venv
.venv/bin/pip install -e . -e ../../python/awis-plugin
```

This plugin has zero runtime deps, so the venv step is optional; the pip
command still works and documents the canonical install path for dep-bearing
plugins.

---

## Plugin Registration

In V1, registration is manual.  Call `Manager.Register(manifestPath)` from
application code pointing at `plugins/git-context-plugin/awis-plugin.yaml`,
or use the AWIS SDK runtime.

A CLI command (`awis plugin install`) is planned for M14 (F-5).

---

## References

- **TDS-05** — `docs/PLUGIN_PROTOCOL.md` (wire format, error codes, FSM)
- **PLUGIN_GUIDE** — `docs/PLUGIN_GUIDE.md` (authoring, testing, lifecycle)
- **Blueprint §11** — Plugin Manifest schema and example (canonical oracle)
