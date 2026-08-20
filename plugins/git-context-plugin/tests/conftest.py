"""conftest.py — pytest configuration for git-context-plugin tests.

Provides the `temp_git_repo` fixture: an initialised git repository with at
least two commits touching at least two files.  Built entirely via subprocess
so the test suite carries zero Python deps beyond the stdlib and awis_plugin.

The suite is skipped automatically if the `git` binary is absent.
"""
from __future__ import annotations

import pathlib
import shutil
import subprocess
import sys

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

# ---------------------------------------------------------------------------
# Bootstrap sys.path so awis_plugin is importable when pytest is run from the
# plugin directory (mirrors python/awis-plugin conftest pattern).
# ---------------------------------------------------------------------------
_AWIS_PLUGIN_SRC = REPO_ROOT / "python" / "awis-plugin"
_PLUGIN_SRC = REPO_ROOT / "plugins" / "git-context-plugin"

for _p in [str(_AWIS_PLUGIN_SRC), str(_PLUGIN_SRC)]:
    if _p not in sys.path:
        sys.path.insert(0, _p)


# ---------------------------------------------------------------------------
# Skip marker: entire suite skipped when git binary is missing.
# ---------------------------------------------------------------------------

def pytest_configure(config: pytest.Config) -> None:  # noqa: ARG001
    """Register custom markers."""


def pytest_collection_modifyitems(
    items: list[pytest.Item],
    config: pytest.Config,  # noqa: ARG001
) -> None:
    """Skip all items when git binary is absent."""
    if shutil.which("git") is None:
        skip = pytest.mark.skip(reason="git binary not found on PATH")
        for item in items:
            item.add_marker(skip)


# ---------------------------------------------------------------------------
# temp_git_repo fixture
# ---------------------------------------------------------------------------

def _git(args: list[str], cwd: str) -> str:
    """Run git with list argv (no shell) and return stdout."""
    result = subprocess.run(
        ["git", *args],
        cwd=cwd,
        capture_output=True,
        text=True,
        check=True,
    )
    return result.stdout.strip()


@pytest.fixture()
def temp_git_repo(tmp_path: pathlib.Path) -> pathlib.Path:
    """Create a temp git repository with >=2 commits touching >=2 files.

    Layout after fixture:
        tmp_path/
            alpha.txt   — created in commit 1
            beta.txt    — created in commit 1, modified in commit 2
            gamma.txt   — created in commit 2
    """
    repo = tmp_path / "repo"
    repo.mkdir()
    repo_s = str(repo)

    _git(["init", "-b", "main"], cwd=repo_s)
    _git(["config", "user.email", "test@awis.example"], cwd=repo_s)
    _git(["config", "user.name", "AWIS Test"], cwd=repo_s)

    # Commit 1 — add alpha.txt and beta.txt
    (repo / "alpha.txt").write_text("alpha line one\nalpha line two\n")
    (repo / "beta.txt").write_text("beta initial\n")
    _git(["add", "alpha.txt", "beta.txt"], cwd=repo_s)
    _git(["commit", "-m", "chore: initial commit\n\nAdds alpha and beta files."], cwd=repo_s)

    # Commit 2 — modify beta.txt and add gamma.txt
    (repo / "beta.txt").write_text("beta initial\nbeta second line\n")
    (repo / "gamma.txt").write_text("gamma content\n")
    _git(["add", "beta.txt", "gamma.txt"], cwd=repo_s)
    _git(["commit", "-m", "feat: add gamma and update beta"], cwd=repo_s)

    return repo
