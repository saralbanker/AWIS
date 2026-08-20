"""test_protocol.py — pytest suite for awis_plugin against the shared golden files.

All golden files are read from internal/plugin/testdata/protocol/ via the
``goldens_dir`` fixture in conftest.py — no copies kept here
(M12-C4 acceptance criterion; TDS-05 §11).

Coverage:
- Handshake echo vs handshake-response.json golden (T11)
- Execute dispatch vs execute-request / execute-response goldens (T11)
- Error shape vs execute-error.json golden: capability_not_found (T11)
- Error shape: handler exception → handler_error + traceback (T11)
- Shutdown notification → clean loop exit (T11)
- mock_request ↔ serve dispatch parity (T12, FR-PS-15)
- mock_request capability_not_found raises PluginCallError (T12)
- mock_handshake returns correct manifest echo (T12)
"""
from __future__ import annotations

import io
import json
import pathlib

import pytest

from awis_plugin import _PluginRegistry, _parse_manifest_yaml
from awis_plugin.testing import PluginCallError, mock_handshake, mock_request

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

FIXTURES_DIR = pathlib.Path(__file__).resolve().parent / "fixtures"


def _read_golden(goldens_dir: pathlib.Path, name: str) -> dict:
    """Read and decode a golden JSON file."""
    return json.loads((goldens_dir / name).read_text(encoding="utf-8"))


def _json_equal(a: dict, b: dict) -> bool:
    """Return True iff two dicts round-trip to identical JSON (key-order agnostic)."""
    return json.dumps(a, sort_keys=True) == json.dumps(b, sort_keys=True)


def _make_registry(*capability_ids: str) -> _PluginRegistry:
    """Return a registry with no-op handlers for each *capability_ids*."""
    reg = _PluginRegistry()
    for cap_id in capability_ids:
        def _handle(inputs: dict, _cap=cap_id) -> dict:
            return {"echo": _cap}
        reg._handlers[cap_id] = _handle
    return reg


def _serve_line(registry: _PluginRegistry, line: str, manifest_path: str | None = None) -> dict:
    """Feed one NDJSON line into serve() and return the parsed response."""
    stdin = io.StringIO(line + "\n")
    stdout = io.StringIO()
    registry.serve(manifest_path=manifest_path, _stdin=stdin, _stdout=stdout, _exit=False)
    raw = stdout.getvalue().strip()
    return json.loads(raw)


# ---------------------------------------------------------------------------
# Handshake tests (T11, TDS-05 §2)
# ---------------------------------------------------------------------------

class TestHandshake:
    """Plugin echoes the manifest on a handshake request (TDS-05 §2)."""

    def test_handshake_echo_matches_golden(self, goldens_dir: pathlib.Path) -> None:
        """serve() responding to handshake-request.json must equal handshake-response.json."""
        request_golden = _read_golden(goldens_dir, "handshake-request.json")
        response_golden = _read_golden(goldens_dir, "handshake-response.json")

        # The fixture manifest matches the golden handshake-response.json
        manifest_path = str(FIXTURES_DIR / "awis-plugin.yaml")
        reg = _PluginRegistry()
        response = _serve_line(reg, json.dumps(request_golden), manifest_path=manifest_path)

        assert response["jsonrpc"] == "2.0"
        assert response["id"] == request_golden["id"]
        assert "result" in response

        result = response["result"]
        expected = response_golden["result"]

        assert result["name"] == expected["name"], (
            f"name mismatch: got {result['name']!r}, want {expected['name']!r}"
        )
        assert result["version"] == expected["version"], (
            f"version mismatch: got {result['version']!r}, want {expected['version']!r}"
        )

        got_ids = {c["id"] for c in result["capabilities"]}
        want_ids = {c["id"] for c in expected["capabilities"]}
        assert got_ids == want_ids, (
            f"capabilities mismatch: got {got_ids}, want {want_ids}"
        )

    def test_handshake_result_has_required_fields(self, goldens_dir: pathlib.Path) -> None:
        """Handshake result must have name, version, capabilities (TDS-05 §2 table)."""
        request_golden = _read_golden(goldens_dir, "handshake-request.json")
        manifest_path = str(FIXTURES_DIR / "awis-plugin.yaml")
        reg = _PluginRegistry()
        response = _serve_line(reg, json.dumps(request_golden), manifest_path=manifest_path)

        result = response["result"]
        assert "name" in result
        assert "version" in result
        assert "capabilities" in result
        assert isinstance(result["capabilities"], list)
        for cap in result["capabilities"]:
            assert "id" in cap


# ---------------------------------------------------------------------------
# Execute dispatch tests (T11, TDS-05 §3)
# ---------------------------------------------------------------------------

class TestExecuteDispatch:
    """Plugin dispatches execute requests by capability id (TDS-05 §3)."""

    def test_execute_success_response_shape(self, goldens_dir: pathlib.Path) -> None:
        """Execute success response must have jsonrpc, id, result.outputs, result.duration_ms."""
        request_golden = _read_golden(goldens_dir, "execute-request.json")
        response_golden = _read_golden(goldens_dir, "execute-response.json")

        cap_id = request_golden["params"]["capability"]

        reg = _PluginRegistry()

        @reg.capability(id=cap_id)
        def handle(inputs: dict) -> dict:
            return {"context": {}}

        response = _serve_line(reg, json.dumps(request_golden))

        assert response["jsonrpc"] == "2.0"
        assert response["id"] == request_golden["id"]
        assert "result" in response
        assert "outputs" in response["result"]
        assert "duration_ms" in response["result"]

        # Outputs must match the golden.
        assert response["result"]["outputs"] == response_golden["result"]["outputs"], (
            f"outputs mismatch\ngot:  {response['result']['outputs']}\n"
            f"want: {response_golden['result']['outputs']}"
        )

    def test_execute_inputs_delivered_to_handler(self, goldens_dir: pathlib.Path) -> None:
        """Inputs from execute-request.json are delivered to the handler verbatim."""
        request_golden = _read_golden(goldens_dir, "execute-request.json")
        cap_id = request_golden["params"]["capability"]
        expected_inputs = request_golden["params"]["inputs"]

        received: dict = {}

        reg = _PluginRegistry()

        @reg.capability(id=cap_id)
        def handle(inputs: dict) -> dict:
            received.update(inputs)
            return {"context": {}}

        _serve_line(reg, json.dumps(request_golden))

        assert received == expected_inputs, (
            f"inputs delivered to handler mismatch\ngot:  {received}\nwant: {expected_inputs}"
        )

    def test_execute_capability_not_found_matches_error_golden(
        self, goldens_dir: pathlib.Path
    ) -> None:
        """Missing capability → error with data.code=capability_not_found (TDS-05 §3 error table)."""
        execute_error_golden = _read_golden(goldens_dir, "execute-error.json")
        request_golden = _read_golden(goldens_dir, "execute-request.json")

        # Register nothing → capability_not_found.
        reg = _PluginRegistry()
        response = _serve_line(reg, json.dumps(request_golden))

        assert "error" in response, "Expected error response"
        err = response["error"]
        assert err["code"] == execute_error_golden["error"]["code"]
        assert "data" in err
        assert err["data"]["code"] == "capability_not_found"

    def test_execute_handler_exception_produces_handler_error(
        self, goldens_dir: pathlib.Path
    ) -> None:
        """Handler exception → error with data.code=handler_error + traceback (TDS-05 §3)."""
        request_golden = _read_golden(goldens_dir, "execute-request.json")
        cap_id = request_golden["params"]["capability"]

        reg = _PluginRegistry()

        @reg.capability(id=cap_id)
        def handle(inputs: dict) -> dict:
            raise ValueError("test exception from handler")

        response = _serve_line(reg, json.dumps(request_golden))

        assert "error" in response
        err = response["error"]
        assert err["data"]["code"] == "handler_error"
        assert "traceback" in err["data"]
        assert "ValueError" in err["data"]["traceback"]


# ---------------------------------------------------------------------------
# Shutdown tests (T11, TDS-05 §4)
# ---------------------------------------------------------------------------

class TestShutdown:
    """Shutdown notification causes clean loop exit (TDS-05 §4)."""

    def test_shutdown_exits_loop(self, goldens_dir: pathlib.Path) -> None:
        """serve() returns cleanly on shutdown notification (no exception, no id in output)."""
        shutdown_golden = _read_golden(goldens_dir, "shutdown-notification.json")

        reg = _PluginRegistry()
        stdin = io.StringIO(json.dumps(shutdown_golden) + "\n")
        stdout = io.StringIO()

        # Must return without exception when _exit=False.
        reg.serve(_stdin=stdin, _stdout=stdout, _exit=False)

        # No response is emitted for a notification (no id field in shutdown).
        assert stdout.getvalue().strip() == ""

    def test_shutdown_after_execute(self, goldens_dir: pathlib.Path) -> None:
        """serve() handles an execute then shutdown notification in sequence."""
        request_golden = _read_golden(goldens_dir, "execute-request.json")
        shutdown_golden = _read_golden(goldens_dir, "shutdown-notification.json")
        cap_id = request_golden["params"]["capability"]

        reg = _PluginRegistry()

        @reg.capability(id=cap_id)
        def handle(inputs: dict) -> dict:
            return {"context": {}}

        lines = json.dumps(request_golden) + "\n" + json.dumps(shutdown_golden) + "\n"
        stdin = io.StringIO(lines)
        stdout = io.StringIO()

        reg.serve(_stdin=stdin, _stdout=stdout, _exit=False)

        # One response line should have been emitted (for the execute).
        lines_out = [l for l in stdout.getvalue().strip().splitlines() if l]
        assert len(lines_out) == 1
        resp = json.loads(lines_out[0])
        assert "result" in resp


# ---------------------------------------------------------------------------
# mock_request / mock_handshake parity tests (T12, FR-PS-15)
# ---------------------------------------------------------------------------

class TestMockRequestParity:
    """mock_request goes through the same dispatch path as serve() (FR-PS-15)."""

    def test_mock_request_returns_same_outputs_as_serve(
        self, goldens_dir: pathlib.Path
    ) -> None:
        """mock_request and serve() return identical outputs for the same capability+inputs."""
        request_golden = _read_golden(goldens_dir, "execute-request.json")
        cap_id = request_golden["params"]["capability"]
        inputs = request_golden["params"]["inputs"]

        reg = _PluginRegistry()

        @reg.capability(id=cap_id)
        def handle(ins: dict) -> dict:
            return {"context": {}}

        # via serve
        response = _serve_line(reg, json.dumps(request_golden))
        serve_outputs = response["result"]["outputs"]

        # via mock_request
        mock_outputs = mock_request(cap_id, inputs, plugin=reg)

        assert serve_outputs == mock_outputs, (
            f"serve outputs {serve_outputs!r} != mock_request outputs {mock_outputs!r}"
        )

    def test_mock_request_capability_not_found_raises(self) -> None:
        """mock_request raises PluginCallError with code=capability_not_found."""
        reg = _PluginRegistry()  # no handlers
        with pytest.raises(PluginCallError) as exc_info:
            mock_request("nonexistent.capability", {}, plugin=reg)

        err = exc_info.value
        assert err.code == "capability_not_found"

    def test_mock_request_handler_error_raises(self) -> None:
        """mock_request raises PluginCallError with code=handler_error on handler exception."""
        reg = _PluginRegistry()

        @reg.capability(id="test.raise")
        def handle(inputs: dict) -> dict:
            raise RuntimeError("deliberate test error")

        with pytest.raises(PluginCallError) as exc_info:
            mock_request("test.raise", {"x": 1}, plugin=reg)

        err = exc_info.value
        assert err.code == "handler_error"
        assert "traceback" in err.data

    def test_mock_request_uses_same_dispatch_as_serve(
        self, goldens_dir: pathlib.Path
    ) -> None:
        """serve() and mock_request agree on error code for missing capability."""
        request_golden = _read_golden(goldens_dir, "execute-request.json")
        cap_id = request_golden["params"]["capability"]
        inputs = request_golden["params"]["inputs"]

        reg = _PluginRegistry()  # no handlers registered

        # serve path → error in response
        resp = _serve_line(reg, json.dumps(request_golden))
        serve_code = resp["error"]["data"]["code"]

        # mock_request path → raises
        with pytest.raises(PluginCallError) as exc_info:
            mock_request(cap_id, inputs, plugin=reg)

        assert exc_info.value.code == serve_code

    def test_mock_handshake_returns_manifest_echo(self) -> None:
        """mock_handshake returns the manifest echo dict for the fixture manifest."""
        manifest_path = str(FIXTURES_DIR / "awis-plugin.yaml")
        result = mock_handshake(manifest_path=manifest_path)

        assert result["name"] == "git-context-plugin"
        assert result["version"] == "1.0.0"
        cap_ids = {c["id"] for c in result["capabilities"]}
        assert "git.context.assemble" in cap_ids
        assert "git.diff.fetch" in cap_ids

    def test_mock_handshake_matches_handshake_response_golden(
        self, goldens_dir: pathlib.Path
    ) -> None:
        """mock_handshake result matches the handshake-response.json golden result."""
        response_golden = _read_golden(goldens_dir, "handshake-response.json")
        manifest_path = str(FIXTURES_DIR / "awis-plugin.yaml")
        result = mock_handshake(manifest_path=manifest_path)

        expected = response_golden["result"]
        assert result["name"] == expected["name"]
        assert result["version"] == expected["version"]
        got_ids = {c["id"] for c in result["capabilities"]}
        want_ids = {c["id"] for c in expected["capabilities"]}
        assert got_ids == want_ids


# ---------------------------------------------------------------------------
# Manifest YAML parser tests
# ---------------------------------------------------------------------------

class TestManifestParser:
    """_parse_manifest_yaml correctly parses the TDS-05 §9 manifest format."""

    def test_parse_fixture_manifest(self) -> None:
        """Parse the test fixture manifest; verify all top-level fields."""
        text = (FIXTURES_DIR / "awis-plugin.yaml").read_text(encoding="utf-8")
        mf = _parse_manifest_yaml(text)

        assert mf["name"] == "git-context-plugin"
        assert mf["version"] == "1.0.0"
        assert mf["description"] == "Assembles git context for capture workflows"
        assert mf["author"] == "awis"
        cap_ids = {c["id"] for c in mf["capabilities"]}
        assert cap_ids == {"git.context.assemble", "git.diff.fetch"}

    def test_parse_minimal_manifest(self) -> None:
        """Minimal manifest with one capability parses correctly."""
        text = """\
name: echo-plugin
version: 0.1.0
description: minimal
author: test

capabilities:
  - id: echo.call
    inputs:
      msg: string
    outputs:
      msg: string
    timeout_ms: 1000

runtime:
  command: python3
  args: ["-m", "echo_plugin"]
  idle_timeout_s: 60
"""
        mf = _parse_manifest_yaml(text)
        assert mf["name"] == "echo-plugin"
        assert mf["version"] == "0.1.0"
        assert len(mf["capabilities"]) == 1
        assert mf["capabilities"][0]["id"] == "echo.call"
