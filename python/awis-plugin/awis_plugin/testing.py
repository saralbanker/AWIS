"""awis_plugin.testing — offline test helpers for AWIS plugin authors (FR-PS-15).

Provides:
    mock_request(capability, inputs, plugin=plugin) → dict outputs
    mock_handshake(manifest_path=None) → manifest echo dict

No subprocess, no stdin/stdout I/O, no runtime required.  Dispatch goes
through the SAME _dispatch() code path that serve() uses so behaviour is
identical (FR-PS-15; TDS-05 §3).
"""
from __future__ import annotations

from pathlib import Path
from typing import Any

from awis_plugin import _PluginRegistry, _dispatch, _load_manifest, plugin as _default_plugin


class PluginCallError(Exception):
    """Raised by mock_request when the capability dispatch returns an error.

    Attributes:
        code:   data.code from the JSON-RPC error (e.g. "capability_not_found").
        detail: human-readable detail string.
        data:   full error data dict.
    """

    def __init__(self, data: dict[str, Any]) -> None:
        self.code: str = data.get("code", "unknown")
        self.detail: str = data.get("detail", "")
        self.data: dict[str, Any] = data
        super().__init__(f"{self.code}: {self.detail}")


def mock_request(
    capability: str,
    inputs: dict[str, Any],
    plugin: _PluginRegistry | None = None,
) -> dict[str, Any]:
    """Invoke *capability* with *inputs* through the same dispatch path as serve().

    Args:
        capability: the capability id to invoke (e.g. "git.context.assemble").
        inputs:     the inputs dict passed to the handler.
        plugin:     registry to use; defaults to the module-level singleton.

    Returns:
        The outputs dict returned by the handler.

    Raises:
        PluginCallError: if the capability is not found or the handler raises.
            error.code matches the JSON-RPC error data.code
            ("capability_not_found" or "handler_error").
    """
    if plugin is None:
        plugin = _default_plugin

    outputs, err_data = plugin._do_dispatch(capability, inputs)
    if err_data is not None:
        raise PluginCallError(err_data)
    return outputs  # type: ignore[return-value]


def mock_handshake(
    manifest_path: str | Path | None = None,
) -> dict[str, Any]:
    """Return the manifest echo dict the plugin would emit during handshake.

    This is what the runtime expects as the JSON-RPC result for the handshake
    method (TDS-05 §2).  No process is started.

    Args:
        manifest_path: path to awis-plugin.yaml; if None, searched beside the
                       awis_plugin package directory.

    Returns:
        dict with keys: name (str), version (str), capabilities (list[dict]).

    Raises:
        FileNotFoundError: if awis-plugin.yaml cannot be located.
    """
    module_dir = Path(__file__).resolve().parent
    mf = _load_manifest(manifest_path, module_dir)
    return {
        "name": mf.get("name", ""),
        "version": mf.get("version", ""),
        "capabilities": mf.get("capabilities", []),
    }


__all__ = ["mock_request", "mock_handshake", "PluginCallError"]
