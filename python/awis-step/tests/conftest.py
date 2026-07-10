"""conftest.py — pytest configuration for awis-step tests.

Provides the ``goldens_dir`` fixture that locates the shared golden protocol
files at ``internal/runner/subprocess/testdata/protocol/`` relative to the
repository root.  The pytest suite must read goldens from there — no copies
(M11-C3 acceptance criterion).
"""
import os
import pathlib

import pytest


def _find_repo_root() -> pathlib.Path:
    """Walk up from this file to find the repo root (contains go.mod)."""
    here = pathlib.Path(__file__).resolve()
    for parent in [here, *here.parents]:
        if (parent / "go.mod").exists():
            return parent
    raise RuntimeError(
        "Could not locate repository root (no go.mod found above this file)"
    )


REPO_ROOT = _find_repo_root()
GOLDENS_DIR = REPO_ROOT / "internal" / "runner" / "subprocess" / "testdata" / "protocol"


@pytest.fixture
def goldens_dir() -> pathlib.Path:
    """Return the path to the canonical golden protocol files."""
    assert GOLDENS_DIR.is_dir(), f"goldens dir not found: {GOLDENS_DIR}"
    return GOLDENS_DIR
