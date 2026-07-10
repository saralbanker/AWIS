"""awis_step — Python reference client for the AWIS subprocess protocol (TDS-04).

Public API (Blueprint §25 verbatim):
    from awis_step import step, StepContext, StepResult, StepError

    @step(id="my-handler")
    def handle(ctx: StepContext) -> StepResult:
        ...

    step.serve()
"""
from __future__ import annotations

import json
import sys
import traceback
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Callable

# ---------------------------------------------------------------------------
# Protocol constant (TDS-04 §2)
# ---------------------------------------------------------------------------
_PROTOCOL = "awis-subprocess/1"


# ---------------------------------------------------------------------------
# Public data types (TDS-04; Blueprint §25 example)
# ---------------------------------------------------------------------------

@dataclass
class StepContext:
    """Parsed request envelope presented to handler functions."""
    instance_id: str
    step_id: str
    attempt: int
    inputs: dict[str, Any]
    deadline: datetime | None = None


@dataclass
class StepResult:
    """Successful handler return value; serialised as a success response envelope."""
    outputs: dict[str, Any] = field(default_factory=dict)


class StepError(Exception):
    """Handler-raised typed error; serialised as an error response envelope (TDS-04 §4)."""

    def __init__(
        self,
        code: str,
        message: str,
        details: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(message)
        self.code = code
        self.message = message
        self.details = details


# ---------------------------------------------------------------------------
# _StepRegistry — the `step` object exported by this module
# ---------------------------------------------------------------------------

class _StepRegistry:
    """Registry and decorator-factory; also exposes serve().

    Usage::

        @step(id="my-handler")
        def handle(ctx: StepContext) -> StepResult:
            ...

        step.serve()
    """

    def __init__(self) -> None:
        self._handlers: dict[str, Callable[[StepContext], StepResult]] = {}

    # -- decorator factory ---------------------------------------------------

    def __call__(self, *, id: str) -> Callable:  # noqa: A002  (id is the API name)
        """Return a decorator that registers the wrapped function under *id*."""
        def decorator(fn: Callable[[StepContext], StepResult]) -> Callable[[StepContext], StepResult]:
            self._handlers[id] = fn
            return fn
        return decorator

    # -- serve ---------------------------------------------------------------

    def serve(
        self,
        *,
        _stdin: Any = None,
        _stdout: Any = None,
    ) -> None:
        """Read one request envelope from stdin, dispatch, write one response to stdout.

        Args:
            _stdin:  injectable stdin-like object (default: sys.stdin); used by tests.
            _stdout: injectable stdout-like object (default: sys.stdout); used by tests.

        The function always exits (via sys.exit(0)) on the success path, as required
        by the one-shot model (TDS-04 §1).  Tests that inject _stdin/_stdout
        should NOT rely on sys.exit — they should call _serve_once instead if
        they want to capture the return rather than exit.  But serve() itself
        always calls sys.exit(0) so that the process terminates cleanly.
        """
        _serve(self._handlers, stdin=_stdin, stdout=_stdout)
        sys.exit(0)


# ---------------------------------------------------------------------------
# Internal implementation
# ---------------------------------------------------------------------------

def _parse_deadline(raw: str | None) -> datetime | None:
    """Parse an RFC3339Nano deadline string to an aware datetime, or None."""
    if raw is None:
        return None
    # Python's fromisoformat handles most RFC3339 variants; the trailing 'Z'
    # must be replaced with '+00:00' on Python < 3.11.
    normalized = raw.replace("Z", "+00:00")
    return datetime.fromisoformat(normalized)


def _parse_context(envelope: dict[str, Any]) -> StepContext:
    """Build a StepContext from a decoded request envelope dict."""
    return StepContext(
        instance_id=envelope["instance_id"],
        step_id=envelope["step_id"],
        attempt=envelope["attempt"],
        inputs=envelope.get("inputs", {}),
        deadline=_parse_deadline(envelope.get("deadline")),
    )


def _write_success(outputs: dict[str, Any], stdout: Any) -> None:
    """Write a success response envelope (TDS-04 §3)."""
    envelope = {
        "protocol": _PROTOCOL,
        "outputs": outputs,
    }
    stdout.write(json.dumps(envelope) + "\n")
    stdout.flush()


def _write_error(
    code: str,
    message: str,
    details: dict[str, Any] | None,
    stdout: Any,
) -> None:
    """Write an error response envelope (TDS-04 §4)."""
    error_block: dict[str, Any] = {
        "code": code,
        "message": message,
    }
    if details is not None:
        error_block["details"] = details
    envelope = {
        "protocol": _PROTOCOL,
        "error": error_block,
    }
    stdout.write(json.dumps(envelope) + "\n")
    stdout.flush()


def _serve(
    handlers: dict[str, Callable[[StepContext], StepResult]],
    *,
    stdin: Any = None,
    stdout: Any = None,
) -> None:
    """Core one-shot dispatch logic; separated for testability."""
    if stdin is None:
        stdin = sys.stdin
    if stdout is None:
        stdout = sys.stdout

    # Read the full stdin content (runtime closes stdin after writing — TDS-04 §1).
    raw = stdin.read()
    envelope: dict[str, Any] = json.loads(raw)

    handler_id: str = envelope["handler"]
    fn = handlers.get(handler_id)

    if fn is None:
        _write_error(
            "handler_not_found",
            f"no handler registered for id {handler_id!r}",
            None,
            stdout,
        )
        return

    ctx = _parse_context(envelope)
    try:
        result = fn(ctx)
        _write_success(result.outputs, stdout)
    except StepError as exc:
        _write_error(exc.code, exc.message, exc.details, stdout)
    except Exception:  # noqa: BLE001
        tb = traceback.format_exc()
        _write_error(
            "handler_error",
            f"handler {handler_id!r} raised an unexpected exception",
            {"traceback": tb},
            stdout,
        )


# ---------------------------------------------------------------------------
# Module-level `step` singleton (Blueprint §25: `from awis_step import step`)
# ---------------------------------------------------------------------------
step = _StepRegistry()

__all__ = ["step", "StepContext", "StepResult", "StepError"]
