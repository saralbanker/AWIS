"""test_protocol.py — pytest suite for awis_step against the shared golden files.

All golden files are read from internal/runner/subprocess/testdata/protocol/
via the ``goldens_dir`` fixture in conftest.py — no copies kept here
(M11-C3 acceptance criterion; TDS-04 §9).

Coverage:
- Request parsing: request-basic.json → StepContext fields (no deadline)
- Request parsing: request-deadline.json → StepContext with deadline datetime
- Response emission: StepResult → JSON-equal to response-ok.json
- Error response: StepError raised → JSON-equal to response-error.json
- handler_not_found envelope when handler id is unregistered
- handler_error envelope with traceback when handler raises unexpected exception
- StepError raise maps verbatim to error envelope
- serve() smoke test over real pipes (subprocess.run of a tiny script)
"""
from __future__ import annotations

import io
import json
import pathlib
import subprocess
import sys
import textwrap

import pytest

from awis_step import StepContext, StepError, StepResult, _StepRegistry, _parse_context, _serve


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _read_golden(goldens_dir: pathlib.Path, name: str) -> dict:
    """Read and decode a golden JSON file."""
    return json.loads((goldens_dir / name).read_text(encoding="utf-8"))


def _json_equal(a: dict, b: dict) -> bool:
    """Return True iff two dicts round-trip to identical JSON (key-order agnostic)."""
    return json.dumps(a, sort_keys=True) == json.dumps(b, sort_keys=True)


# ---------------------------------------------------------------------------
# Request parsing tests
# ---------------------------------------------------------------------------

class TestRequestParsing:
    """Parse request envelopes into StepContext (TDS-04 §2)."""

    def test_basic_request_fields(self, goldens_dir: pathlib.Path) -> None:
        """request-basic.json → StepContext with correct fields and no deadline."""
        envelope = _read_golden(goldens_dir, "request-basic.json")
        ctx = _parse_context(envelope)

        assert ctx.instance_id == "inst-001"
        assert ctx.step_id == "compute-hrv"
        assert ctx.attempt == 1
        assert ctx.inputs == {"records": [1, 2, 3]}
        assert ctx.deadline is None

    def test_deadline_request_fields(self, goldens_dir: pathlib.Path) -> None:
        """request-deadline.json → StepContext with an aware datetime deadline."""
        from datetime import datetime, timezone

        envelope = _read_golden(goldens_dir, "request-deadline.json")
        ctx = _parse_context(envelope)

        assert ctx.instance_id == "inst-001"
        assert ctx.step_id == "compute-hrv"
        assert ctx.attempt == 1
        assert ctx.inputs == {"records": [1, 2, 3]}

        # deadline must be parsed to a timezone-aware datetime
        assert ctx.deadline is not None
        assert ctx.deadline.tzinfo is not None
        # The golden value is 2099-01-01T00:00:00.000000000Z
        assert ctx.deadline == datetime(2099, 1, 1, 0, 0, 0, tzinfo=timezone.utc)


# ---------------------------------------------------------------------------
# Response emission tests
# ---------------------------------------------------------------------------

class TestResponseEmission:
    """Handler return values produce protocol-correct response envelopes."""

    def test_success_response_json_equal_to_golden(self, goldens_dir: pathlib.Path) -> None:
        """StepResult(outputs={"hrv": 42.5}) → JSON-equal to response-ok.json."""
        golden = _read_golden(goldens_dir, "response-ok.json")

        registry = _StepRegistry()

        @registry(id="python my_step.py")
        def handle(ctx: StepContext) -> StepResult:
            return StepResult(outputs={"hrv": 42.5})

        # Build request envelope from the basic golden
        request_env = _read_golden(goldens_dir, "request-basic.json")
        stdin = io.StringIO(json.dumps(request_env))
        stdout = io.StringIO()

        _serve(registry._handlers, stdin=stdin, stdout=stdout)

        emitted = json.loads(stdout.getvalue())
        assert _json_equal(emitted, golden), (
            f"Emitted response does not match golden response-ok.json\n"
            f"got:  {emitted}\nwant: {golden}"
        )

    def test_error_response_json_equal_to_golden(self, goldens_dir: pathlib.Path) -> None:
        """StepError(code, message, details) → JSON-equal to response-error.json."""
        golden = _read_golden(goldens_dir, "response-error.json")

        registry = _StepRegistry()

        @registry(id="python my_step.py")
        def handle(ctx: StepContext) -> StepResult:
            raise StepError(
                code="validation_error",
                message="records field must not be empty",
                details={"field": "records"},
            )

        request_env = _read_golden(goldens_dir, "request-basic.json")
        stdin = io.StringIO(json.dumps(request_env))
        stdout = io.StringIO()

        _serve(registry._handlers, stdin=stdin, stdout=stdout)

        emitted = json.loads(stdout.getvalue())
        assert _json_equal(emitted, golden), (
            f"Emitted error response does not match golden response-error.json\n"
            f"got:  {emitted}\nwant: {golden}"
        )


# ---------------------------------------------------------------------------
# Error path tests
# ---------------------------------------------------------------------------

class TestErrorPaths:
    """Protocol error envelopes for various failure modes (TDS-04 §4, SPEC §3)."""

    def test_handler_not_found(self, goldens_dir: pathlib.Path) -> None:
        """Unregistered handler id → handler_not_found error envelope."""
        registry = _StepRegistry()
        # register nothing

        request_env = _read_golden(goldens_dir, "request-basic.json")
        stdin = io.StringIO(json.dumps(request_env))
        stdout = io.StringIO()

        _serve(registry._handlers, stdin=stdin, stdout=stdout)

        emitted = json.loads(stdout.getvalue())
        assert emitted["protocol"] == "awis-subprocess/1"
        assert "error" in emitted
        assert emitted["error"]["code"] == "handler_not_found"
        assert "python my_step.py" in emitted["error"]["message"]

    def test_handler_error_carries_traceback(self, goldens_dir: pathlib.Path) -> None:
        """Handler raising an unexpected exception → handler_error with details.traceback."""
        registry = _StepRegistry()

        @registry(id="python my_step.py")
        def handle(ctx: StepContext) -> StepResult:
            raise ValueError("something went wrong unexpectedly")

        request_env = _read_golden(goldens_dir, "request-basic.json")
        stdin = io.StringIO(json.dumps(request_env))
        stdout = io.StringIO()

        _serve(registry._handlers, stdin=stdin, stdout=stdout)

        emitted = json.loads(stdout.getvalue())
        assert emitted["protocol"] == "awis-subprocess/1"
        assert "error" in emitted
        assert emitted["error"]["code"] == "handler_error"
        assert "traceback" in emitted["error"]["details"]
        assert "ValueError" in emitted["error"]["details"]["traceback"]

    def test_step_error_maps_verbatim(self, goldens_dir: pathlib.Path) -> None:
        """StepError raised by handler → its code/message/details passed verbatim."""
        registry = _StepRegistry()

        @registry(id="python my_step.py")
        def handle(ctx: StepContext) -> StepResult:
            raise StepError(
                code="my_custom_code",
                message="custom message text",
                details={"key": "value"},
            )

        request_env = _read_golden(goldens_dir, "request-basic.json")
        stdin = io.StringIO(json.dumps(request_env))
        stdout = io.StringIO()

        _serve(registry._handlers, stdin=stdin, stdout=stdout)

        emitted = json.loads(stdout.getvalue())
        assert emitted["error"]["code"] == "my_custom_code"
        assert emitted["error"]["message"] == "custom message text"
        assert emitted["error"]["details"] == {"key": "value"}


# ---------------------------------------------------------------------------
# serve() smoke test over real pipes
# ---------------------------------------------------------------------------

class TestServeSmoke:
    """serve() smoke test: run a real script as a subprocess (TDS-04 §1)."""

    def test_serve_via_real_pipes(self, goldens_dir: pathlib.Path, tmp_path: pathlib.Path) -> None:
        """Run a tiny awis_step script as a subprocess; verify it emits the golden response."""
        # Write the script inline into tmp_path so the test is self-contained.
        script = tmp_path / "smoke_step.py"
        # The script sets PYTHONPATH at import time via sys.path manipulation.
        # The parent conftest locates the repo root; we pass it as an env var.
        awis_step_src = str(pathlib.Path(__file__).resolve().parent.parent)
        script.write_text(
            textwrap.dedent(f"""\
                import sys
                sys.path.insert(0, {awis_step_src!r})
                from awis_step import step, StepContext, StepResult

                @step(id="python my_step.py")
                def handle(ctx: StepContext) -> StepResult:
                    return StepResult(outputs={{"hrv": 42.5}})

                step.serve()
            """),
            encoding="utf-8",
        )

        # Use the request-basic.json golden as stdin.
        request_bytes = (goldens_dir / "request-basic.json").read_bytes()
        golden_ok = _read_golden(goldens_dir, "response-ok.json")

        result = subprocess.run(
            [sys.executable, str(script)],
            input=request_bytes,
            capture_output=True,
            timeout=10,
        )

        assert result.returncode == 0, (
            f"Script exited {result.returncode}\nstderr: {result.stderr.decode()}"
        )
        emitted = json.loads(result.stdout)
        assert _json_equal(emitted, golden_ok), (
            f"serve() smoke output mismatch\ngot:  {emitted}\nwant: {golden_ok}"
        )
