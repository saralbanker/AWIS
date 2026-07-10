"""echo_plugin.py — minimal awis_plugin fixture for M12-C4 Go e2e test.

Registers one capability (echo.call) that returns the input message as reply.
Uses awis_plugin via sys.path bootstrap (the Go test injects AWIS_PLUGIN_PATH).

The Go e2e test (e2e_python_test.go) sets sys.path so awis_plugin is importable
without a pip install.
"""
import os
import sys

# Bootstrap: the Go test sets AWIS_PLUGIN_LIBPATH to python/awis-plugin so
# awis_plugin is importable without a pip install.
_lib_path = os.environ.get("AWIS_PLUGIN_LIBPATH", "")
if _lib_path and _lib_path not in sys.path:
    sys.path.insert(0, _lib_path)

from awis_plugin import plugin  # noqa: E402


@plugin.capability(id="echo.call")
def echo(inputs: dict) -> dict:
    return {"reply": inputs.get("message", "")}


if __name__ == "__main__":
    # manifest_path is passed as the first arg by the e2e test.
    manifest_path = sys.argv[1] if len(sys.argv) > 1 else None
    plugin.serve(manifest_path=manifest_path)
