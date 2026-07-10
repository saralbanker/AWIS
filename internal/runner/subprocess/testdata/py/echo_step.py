"""echo_step.py — awis_step fixture for M11-C3 e2e keystone test.

This script is invoked by the Go e2e test as a subprocess step handler.
The Go runner sets the request envelope's 'handler' field to the full
command string used to spawn this process (TDS-04 §5), i.e.
    "python3 /abs/path/echo_step.py"

The script reconstructs that same string to register its handler, so that
_serve() can dispatch correctly.

The script bootstraps its own import path so that awis_step is importable
without needing PYTHONPATH to be pre-set in the environment — but the Go
e2e test also injects PYTHONPATH via the 'env' command for belt-and-braces.
"""
import os
import pathlib
import sys

# Bootstrap: locate repo root from this file's position and add awis-step
# to the path so this script can be run without PYTHONPATH being pre-set.
# File is at: <repo>/internal/runner/subprocess/testdata/py/echo_step.py
_THIS = pathlib.Path(__file__).resolve()
_REPO_ROOT = _THIS.parents[5]  # 5 levels up: py/ -> testdata/ -> subprocess/ -> runner/ -> internal/ -> repo root
_AWIS_STEP = _REPO_ROOT / "python" / "awis-step"
if str(_AWIS_STEP) not in sys.path:
    sys.path.insert(0, str(_AWIS_STEP))

from awis_step import step, StepContext, StepResult  # noqa: E402

# The envelope 'handler' field equals step.Handler, which is the full argv
# string the Go test uses: "python3 <abs_path>".
# Reconstruct that string so we register under the right key.
_HANDLER_ID = "python3 " + str(_THIS)


@step(id=_HANDLER_ID)
def echo(ctx: StepContext) -> StepResult:
    """Return all inputs as outputs, adding {"echo": true}."""
    outputs = dict(ctx.inputs)
    outputs["echo"] = True
    return StepResult(outputs=outputs)


step.serve()
