"""Entry point for `python3 -m git_context_plugin`.

Starts the JSON-RPC serve loop with this package's manifest.

The AWIS_PLUGIN_LIBPATH sys.path bootstrap deliberately does NOT live here.
`python3 -m git_context_plugin` imports the package — running __init__.py in
full — before executing this file, and __init__.py imports awis_plugin at
module level. A bootstrap here would run after that import had already failed
(B-21), so it lives at the top of __init__.py instead.
"""
from __future__ import annotations

from pathlib import Path

# Importing the package runs the AWIS_PLUGIN_LIBPATH bootstrap and registers
# the @plugin.capability decorators.
import git_context_plugin  # noqa: F401

from awis_plugin import plugin

# ---------------------------------------------------------------------------
# Locate the manifest beside this package directory.
# ---------------------------------------------------------------------------
_MANIFEST_PATH = Path(__file__).resolve().parent.parent / "awis-plugin.yaml"

if __name__ == "__main__":
    plugin.serve(manifest_path=_MANIFEST_PATH)
