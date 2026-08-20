"""Entry point for `python3 -m git_context_plugin`.

Bootstraps awis_plugin from AWIS_PLUGIN_LIBPATH if set (mirrors the
awis-step fixture pattern established by M12 e2e), then starts the
JSON-RPC serve loop with this package's manifest.
"""
from __future__ import annotations

import os
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# sys.path bootstrap: honour AWIS_PLUGIN_LIBPATH (M12 e2e pattern).
# Insert the awis-plugin library root before any other import.
# ---------------------------------------------------------------------------
_libpath = os.environ.get("AWIS_PLUGIN_LIBPATH")
if _libpath:
    for _p in reversed(_libpath.split(os.pathsep)):
        _p = _p.strip()
        if _p and _p not in sys.path:
            sys.path.insert(0, _p)

# ---------------------------------------------------------------------------
# Import plugin registry and register capabilities by importing the package.
# ---------------------------------------------------------------------------
import git_context_plugin  # noqa: F401, E402  (registers @plugin.capability decorators)

from awis_plugin import plugin  # noqa: E402

# ---------------------------------------------------------------------------
# Locate the manifest beside this package directory.
# ---------------------------------------------------------------------------
_MANIFEST_PATH = Path(__file__).resolve().parent.parent / "awis-plugin.yaml"

if __name__ == "__main__":
    plugin.serve(manifest_path=_MANIFEST_PATH)
