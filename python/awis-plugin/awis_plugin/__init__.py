"""awis_plugin — Python reference client for the AWIS plugin protocol (TDS-05).

Public API:
    from awis_plugin import plugin

    @plugin.capability(id="my.capability.id")
    def handle(inputs: dict) -> dict:
        return {"result": inputs["value"]}

    plugin.serve()          # blocking NDJSON JSON-RPC loop (stdin/stdout)
"""
from __future__ import annotations

import json
import sys
import traceback
from pathlib import Path
from typing import Any, Callable

# ---------------------------------------------------------------------------
# Protocol constant (TDS-05 §2)
# ---------------------------------------------------------------------------
_PROTOCOL_VERSION = "awis-plugin/1"


# ---------------------------------------------------------------------------
# _dispatch — shared dispatch path (used by both serve() and mock_request)
# ---------------------------------------------------------------------------

def _dispatch(
    handlers: dict[str, Callable[[dict], dict]],
    capability: str,
    inputs: dict[str, Any],
) -> tuple[dict[str, Any] | None, dict[str, Any] | None]:
    """Invoke the handler for *capability* with *inputs*.

    Returns (outputs, error_data) where exactly one is not None.
    error_data has keys: code (str), detail (str), and optionally traceback (str).

    This is the canonical dispatch function; both serve() and mock_request() call
    this to guarantee identical behaviour (FR-PS-15, TDS-05 §3).
    """
    fn = handlers.get(capability)
    if fn is None:
        return None, {
            "code": "capability_not_found",
            "detail": "no handler registered for capability",
        }
    try:
        result = fn(inputs)
        return result, None
    except Exception:  # noqa: BLE001
        tb = traceback.format_exc()
        return None, {
            "code": "handler_error",
            "detail": f"handler {capability!r} raised an unexpected exception",
            "traceback": tb,
        }


# ---------------------------------------------------------------------------
# _load_manifest — read awis-plugin.yaml and return the manifest dict
# ---------------------------------------------------------------------------

def _load_manifest(manifest_path: str | Path | None, module_dir: Path) -> dict[str, Any]:
    """Load and return the awis-plugin.yaml manifest as a dict.

    If *manifest_path* is given it is used directly; otherwise the file is
    located beside the directory of the calling module (awis_plugin package dir
    falls back to module_dir).
    """
    import importlib.util  # stdlib only

    if manifest_path is not None:
        p = Path(manifest_path)
    else:
        p = module_dir / "awis-plugin.yaml"

    if not p.exists():
        raise FileNotFoundError(f"awis-plugin.yaml not found at {p}")

    # stdlib yaml-ish parsing: we only need name, version, capabilities[].id
    # Use the json-compatible subset via a very small hand-rolled YAML reader.
    # Since we cannot depend on pyyaml we use a minimal approach.
    return _parse_manifest_yaml(p.read_text(encoding="utf-8"))


def _parse_manifest_yaml(text: str) -> dict[str, Any]:
    """Minimal manifest YAML parser (stdlib only; covers the TDS-05 §9 schema).

    Supports only the subset of YAML used in awis-plugin.yaml: scalar key:value
    pairs and the capabilities list with id sub-keys.  Sufficient for the manifest
    echo required by the handshake response (TDS-05 §2).
    """
    result: dict[str, Any] = {}
    capabilities: list[dict[str, Any]] = []
    in_capabilities = False
    current_cap: dict[str, Any] | None = None

    for raw_line in text.splitlines():
        line = raw_line.rstrip()
        stripped = line.lstrip()

        # Skip blank lines and comments.
        if not stripped or stripped.startswith("#"):
            continue

        indent = len(line) - len(stripped)

        if stripped.startswith("capabilities:"):
            in_capabilities = True
            continue

        if in_capabilities:
            if indent == 0:
                # A top-level key signals end of capabilities block.
                if current_cap is not None:
                    capabilities.append(current_cap)
                    current_cap = None
                in_capabilities = False
                # Fall through to parse this top-level key.
            elif stripped.startswith("- "):
                # New capability entry.
                if current_cap is not None:
                    capabilities.append(current_cap)
                content = stripped[2:]
                if ":" in content:
                    k, v = content.split(":", 1)
                    current_cap = {k.strip(): v.strip()}
                else:
                    current_cap = {}
                continue
            elif indent >= 4 and current_cap is not None:
                # Sub-key of current capability (id, inputs, outputs, timeout_ms).
                if ":" in stripped:
                    k, v = stripped.split(":", 1)
                    k = k.strip()
                    v = v.strip()
                    if k == "id":
                        current_cap[k] = v
                continue
            else:
                continue

        # Top-level key: value
        if ":" in stripped:
            k, v = stripped.split(":", 1)
            k = k.strip()
            v = v.strip()
            if k not in ("capabilities", "runtime") and indent == 0:
                result[k] = v

    if in_capabilities and current_cap is not None:
        capabilities.append(current_cap)

    result["capabilities"] = [{"id": c["id"]} for c in capabilities if "id" in c]
    return result


# ---------------------------------------------------------------------------
# _PluginRegistry — the `plugin` singleton
# ---------------------------------------------------------------------------

class _PluginRegistry:
    """Registry and decorator-factory; also exposes serve().

    Usage::

        from awis_plugin import plugin

        @plugin.capability(id="my.capability.id")
        def handle(inputs: dict) -> dict:
            return {"result": inputs["value"]}

        plugin.serve()
    """

    def __init__(self) -> None:
        self._handlers: dict[str, Callable[[dict], dict]] = {}

    # -- decorator factory ---------------------------------------------------

    def capability(self, *, id: str) -> Callable:  # noqa: A002
        """Return a decorator that registers the wrapped function under *id*."""
        def decorator(fn: Callable[[dict], dict]) -> Callable[[dict], dict]:
            self._handlers[id] = fn
            return fn
        return decorator

    # -- internal dispatch (shared with testing.py) --------------------------

    def _do_dispatch(
        self,
        capability: str,
        inputs: dict[str, Any],
    ) -> tuple[dict[str, Any] | None, dict[str, Any] | None]:
        """Delegate to module-level _dispatch using this registry's handlers."""
        return _dispatch(self._handlers, capability, inputs)

    # -- serve ---------------------------------------------------------------

    def serve(
        self,
        manifest_path: str | Path | None = None,
        *,
        _stdin: Any = None,
        _stdout: Any = None,
        _exit: bool = True,
    ) -> None:
        """Run the NDJSON JSON-RPC 2.0 loop (TDS-05 §1–§4).

        Reads messages from stdin (one JSON object per line), dispatches them,
        writes responses to stdout.  Exits cleanly (exit 0) on shutdown
        notification (TDS-05 §4).

        Args:
            manifest_path: path to awis-plugin.yaml; if None, searched beside
                           the package directory.
            _stdin:  injectable stdin (default: sys.stdin).
            _stdout: injectable stdout (default: sys.stdout).
            _exit:   if False, return instead of calling sys.exit(0); used by
                     tests to avoid terminating the process.
        """
        if _stdin is None:
            _stdin = sys.stdin
        if _stdout is None:
            _stdout = sys.stdout

        # Locate the awis-plugin.yaml for handshake echo.
        # Use the directory of the entry module (sys.argv[0]) as the default
        # search location — this is the plugin author's script, not the library.
        # Fall back to cwd if argv[0] is empty or not a real path.
        import sys as _sys
        _argv0 = _sys.argv[0] if _sys.argv else ""
        _argv0_path = Path(_argv0).resolve() if _argv0 else Path.cwd()
        module_dir = _argv0_path.parent if _argv0_path.is_file() else _argv0_path

        manifest_data: dict[str, Any] | None = None

        def _get_manifest() -> dict[str, Any]:
            nonlocal manifest_data
            if manifest_data is None:
                manifest_data = _load_manifest(manifest_path, module_dir)
            return manifest_data

        def _write(obj: dict) -> None:
            _stdout.write(json.dumps(obj) + "\n")
            _stdout.flush()

        for line in _stdin:
            line = line.rstrip("\n")
            if not line:
                continue

            # Parse the JSON-RPC object.
            try:
                msg: dict[str, Any] = json.loads(line)
            except json.JSONDecodeError as exc:
                # Malformed line: emit a parse error response (best-effort; no id).
                _write({
                    "jsonrpc": "2.0",
                    "id": None,
                    "error": {
                        "code": -32700,
                        "message": "Parse error",
                        "data": {"detail": str(exc)},
                    },
                })
                continue

            method = msg.get("method")
            msg_id = msg.get("id")  # None for notifications

            if method == "handshake":
                # Echo the manifest (TDS-05 §2).
                try:
                    mf = _get_manifest()
                except Exception as exc:  # noqa: BLE001
                    _write({
                        "jsonrpc": "2.0",
                        "id": msg_id,
                        "error": {
                            "code": -32000,
                            "message": "handshake_error",
                            "data": {"detail": str(exc)},
                        },
                    })
                    continue
                _write({
                    "jsonrpc": "2.0",
                    "id": msg_id,
                    "result": {
                        "name": mf.get("name", ""),
                        "version": mf.get("version", ""),
                        "capabilities": mf.get("capabilities", []),
                    },
                })

            elif method == "execute":
                params: dict[str, Any] = msg.get("params", {})
                cap = params.get("capability", "")
                inputs = params.get("inputs", {})

                outputs, err_data = self._do_dispatch(cap, inputs)

                if err_data is None:
                    _write({
                        "jsonrpc": "2.0",
                        "id": msg_id,
                        "result": {
                            "outputs": outputs,
                            "duration_ms": 0,
                        },
                    })
                else:
                    _write({
                        "jsonrpc": "2.0",
                        "id": msg_id,
                        "error": {
                            "code": -32000,
                            "message": "handler_error",
                            "data": err_data,
                        },
                    })

            elif method == "shutdown":
                # Notification (no id); exit cleanly (TDS-05 §4).
                if _exit:
                    sys.exit(0)
                else:
                    return

            else:
                # Unknown method.
                if msg_id is not None:
                    _write({
                        "jsonrpc": "2.0",
                        "id": msg_id,
                        "error": {
                            "code": -32601,
                            "message": "Method not found",
                            "data": {"method": method},
                        },
                    })


# ---------------------------------------------------------------------------
# Module-level `plugin` singleton
# ---------------------------------------------------------------------------
plugin = _PluginRegistry()

__all__ = ["plugin", "_PluginRegistry", "_dispatch", "_load_manifest", "_parse_manifest_yaml"]
