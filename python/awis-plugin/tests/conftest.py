"""conftest.py — pytest configuration for awis-plugin tests.

Provides the ``goldens_dir`` fixture that locates the shared golden plugin
protocol files at ``internal/plugin/testdata/protocol/`` relative to the
repository root.  The pytest suite reads goldens from there — no copies
(M12-C4 acceptance criterion; TDS-05 §11).
"""
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
GOLDENS_DIR = REPO_ROOT / "internal" / "plugin" / "testdata" / "protocol"


@pytest.fixture
def goldens_dir() -> pathlib.Path:
    """Return the path to the canonical golden plugin protocol files."""
    assert GOLDENS_DIR.is_dir(), f"goldens dir not found: {GOLDENS_DIR}"
    return GOLDENS_DIR
