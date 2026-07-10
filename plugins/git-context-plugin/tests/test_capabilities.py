"""test_capabilities.py — offline pytest suite for git-context-plugin.

Tests both capabilities via mock_request (FR-PS-15):
    - git.context.assemble: happy path, bad ref, non-repo dir
    - git.diff.fetch:       happy path, bad ref

The `temp_git_repo` fixture is defined in conftest.py; the entire suite is
skipped when the `git` binary is absent (also via conftest.py).
"""
from __future__ import annotations

import pathlib

import pytest

# Register the plugin capabilities by importing the package.
import git_context_plugin  # noqa: F401

from awis_plugin.testing import PluginCallError, mock_request


# ---------------------------------------------------------------------------
# git.context.assemble — happy path
# ---------------------------------------------------------------------------

class TestAssembleHappy:
    """Verify every pinned key is present in the returned context object."""

    def test_context_keys_present(self, temp_git_repo: pathlib.Path) -> None:
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD"},
        )
        ctx = outputs["context"]

        # Every pinned key from SPEC / CE pins must be present.
        assert "ref" in ctx
        assert "sha" in ctx
        assert "author" in ctx
        assert "email" in ctx
        assert "date" in ctx
        assert "message" in ctx
        assert "parents" in ctx
        assert "files" in ctx
        assert "stats" in ctx

    def test_ref_echo(self, temp_git_repo: pathlib.Path) -> None:
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD"},
        )
        assert outputs["context"]["ref"] == "HEAD"

    def test_sha_is_40_hex(self, temp_git_repo: pathlib.Path) -> None:
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD"},
        )
        sha = outputs["context"]["sha"]
        assert len(sha) == 40
        assert all(c in "0123456789abcdef" for c in sha)

    def test_author_and_email(self, temp_git_repo: pathlib.Path) -> None:
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD"},
        )
        ctx = outputs["context"]
        assert ctx["author"] == "AWIS Test"
        assert ctx["email"] == "test@awis.example"

    def test_date_iso_format(self, temp_git_repo: pathlib.Path) -> None:
        """Date must be non-empty and contain at least a 'T' ISO separator."""
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD"},
        )
        date = outputs["context"]["date"]
        assert date  # non-empty
        assert "T" in date or "+" in date or "-" in date  # ISO 8601 marker

    def test_message_non_empty(self, temp_git_repo: pathlib.Path) -> None:
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD"},
        )
        assert outputs["context"]["message"].strip()

    def test_parents_is_list(self, temp_git_repo: pathlib.Path) -> None:
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD"},
        )
        parents = outputs["context"]["parents"]
        assert isinstance(parents, list)
        # HEAD has one parent (it is the second commit).
        assert len(parents) == 1

    def test_files_list_contains_changed_paths(self, temp_git_repo: pathlib.Path) -> None:
        """Commit 2 (HEAD) changes beta.txt and adds gamma.txt."""
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD"},
        )
        files = outputs["context"]["files"]
        assert isinstance(files, list)
        paths = [f["path"] for f in files]
        assert "beta.txt" in paths or "gamma.txt" in paths

    def test_files_entries_have_path_and_status(self, temp_git_repo: pathlib.Path) -> None:
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD"},
        )
        for f in outputs["context"]["files"]:
            assert "path" in f
            assert "status" in f

    def test_stats_shape(self, temp_git_repo: pathlib.Path) -> None:
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD"},
        )
        stats = outputs["context"]["stats"]
        assert "files_changed" in stats
        assert "insertions" in stats
        assert "deletions" in stats
        assert isinstance(stats["files_changed"], int)
        assert isinstance(stats["insertions"], int)
        assert isinstance(stats["deletions"], int)

    def test_first_commit_has_no_parents(self, temp_git_repo: pathlib.Path) -> None:
        """The initial commit (HEAD~1) must report an empty parents list."""
        outputs = mock_request(
            "git.context.assemble",
            {"repo_path": str(temp_git_repo), "ref": "HEAD~1"},
        )
        assert outputs["context"]["parents"] == []


# ---------------------------------------------------------------------------
# git.context.assemble — error paths
# ---------------------------------------------------------------------------

class TestAssembleErrors:

    def test_bad_ref_raises_handler_error(self, temp_git_repo: pathlib.Path) -> None:
        with pytest.raises(PluginCallError) as exc_info:
            mock_request(
                "git.context.assemble",
                {"repo_path": str(temp_git_repo), "ref": "nonexistent-branch-xyz"},
            )
        assert exc_info.value.code == "handler_error"

    def test_non_repo_dir_raises_handler_error(self, tmp_path: pathlib.Path) -> None:
        """A plain directory that is not a git repo must trigger handler_error."""
        non_repo = tmp_path / "not-a-repo"
        non_repo.mkdir()
        with pytest.raises(PluginCallError) as exc_info:
            mock_request(
                "git.context.assemble",
                {"repo_path": str(non_repo), "ref": "HEAD"},
            )
        assert exc_info.value.code == "handler_error"


# ---------------------------------------------------------------------------
# git.diff.fetch — happy path
# ---------------------------------------------------------------------------

class TestDiffFetchHappy:

    def test_diff_contains_changed_path(self, temp_git_repo: pathlib.Path) -> None:
        """Diff between HEAD~1 and HEAD must reference at least one changed file."""
        outputs = mock_request(
            "git.diff.fetch",
            {
                "repo_path": str(temp_git_repo),
                "from_ref": "HEAD~1",
                "to_ref": "HEAD",
            },
        )
        diff = outputs["diff"]
        assert isinstance(diff, str)
        # The diff between commit 1 and commit 2 touches beta.txt and gamma.txt.
        assert "beta.txt" in diff or "gamma.txt" in diff

    def test_diff_key_present(self, temp_git_repo: pathlib.Path) -> None:
        outputs = mock_request(
            "git.diff.fetch",
            {
                "repo_path": str(temp_git_repo),
                "from_ref": "HEAD~1",
                "to_ref": "HEAD",
            },
        )
        assert "diff" in outputs

    def test_diff_same_ref_is_empty(self, temp_git_repo: pathlib.Path) -> None:
        """A diff from HEAD to HEAD should produce an empty diff string."""
        outputs = mock_request(
            "git.diff.fetch",
            {
                "repo_path": str(temp_git_repo),
                "from_ref": "HEAD",
                "to_ref": "HEAD",
            },
        )
        assert outputs["diff"] == ""


# ---------------------------------------------------------------------------
# git.diff.fetch — error path
# ---------------------------------------------------------------------------

class TestDiffFetchErrors:

    def test_bad_ref_raises_handler_error(self, temp_git_repo: pathlib.Path) -> None:
        with pytest.raises(PluginCallError) as exc_info:
            mock_request(
                "git.diff.fetch",
                {
                    "repo_path": str(temp_git_repo),
                    "from_ref": "no-such-ref",
                    "to_ref": "HEAD",
                },
            )
        assert exc_info.value.code == "handler_error"
