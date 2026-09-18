"""git_context_plugin — AWIS reference plugin for git integration.

Capabilities:
    git.context.assemble(repo_path, ref) → context object
    git.diff.fetch(repo_path, from_ref, to_ref) → diff string

Zero Python runtime dependencies; all git operations delegate to the
`git` binary via subprocess (list argv, no shell).
"""
from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path
from typing import Any

# ---------------------------------------------------------------------------
# sys.path bootstrap: honour AWIS_PLUGIN_LIBPATH.
#
# This MUST live here, in __init__.py, and it must run before the awis_plugin
# import below. `python3 -m git_context_plugin` imports this package — running
# this file top to bottom — BEFORE it executes __main__.py. A bootstrap placed
# in __main__.py therefore runs too late: the `from awis_plugin import plugin`
# below has already raised ModuleNotFoundError and the process is dead before
# __main__.py's first line (B-21).
#
# AWIS resolves a relative AWIS_PLUGIN_LIBPATH in the manifest against the
# manifest's own directory, so the shipped plugin works from any working
# directory.
# ---------------------------------------------------------------------------
_libpath = os.environ.get("AWIS_PLUGIN_LIBPATH")
if _libpath:
    for _p in reversed(_libpath.split(os.pathsep)):
        _p = _p.strip()
        if _p and _p not in sys.path:
            sys.path.insert(0, _p)

from awis_plugin import plugin  # noqa: E402  (must follow the bootstrap above)


def _run_git(args: list[str], cwd: str) -> str:
    """Run a git command with list argv (no shell) in cwd.

    Returns stdout text on success.
    Raises RuntimeError carrying git stderr on non-zero exit.
    """
    result = subprocess.run(
        args,
        cwd=cwd,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        stderr = result.stderr.strip()
        raise RuntimeError(stderr or f"git exited with code {result.returncode}")
    return result.stdout


@plugin.capability(id="git.context.assemble")
def assemble(inputs: dict[str, Any]) -> dict[str, Any]:
    """Assemble git context for a given repo_path and ref.

    Returns a context object:
        {ref, sha, author, email, date (ISO), message, parents[], files[{path,status}],
         stats{files_changed, insertions, deletions}}
    """
    repo_path: str = inputs["repo_path"]
    ref: str = inputs["ref"]

    # Validate repo_path is a git repository by running git rev-parse.
    _run_git(["git", "rev-parse", "--git-dir"], cwd=repo_path)

    # Resolve ref to a full SHA.
    sha = _run_git(["git", "rev-parse", ref], cwd=repo_path).strip()

    # Get commit metadata using a custom format:
    # author name, author email, ISO date, parent SHAs, commit message
    fmt = "%aN%n%aE%n%aI%n%P%n%B"
    show_out = _run_git(
        ["git", "show", "--numstat", f"--format={fmt}", sha],
        cwd=repo_path,
    )

    lines = show_out.split("\n")
    author = lines[0] if len(lines) > 0 else ""
    email = lines[1] if len(lines) > 1 else ""
    date = lines[2] if len(lines) > 2 else ""
    parents_line = lines[3] if len(lines) > 3 else ""
    parents: list[str] = [p for p in parents_line.split() if p]

    # Message runs until the first blank line that separates from numstat output.
    # The format is: %B followed by a NUL-terminated block, then numstat lines.
    # In practice git show outputs: header lines, blank line, then numstat lines.
    # We find the end of the message by looking for numstat lines (digit\tdigit\t).
    message_lines: list[str] = []
    numstat_lines: list[str] = []
    in_message = True
    i = 4
    while i < len(lines):
        line = lines[i]
        # numstat lines have the form: digits\tdigits\tpath
        # A blank separator line appears between message and numstat.
        if in_message:
            # Detect transition: blank line followed by numstat-format line.
            if line == "" and i + 1 < len(lines) and _is_numstat(lines[i + 1]):
                in_message = False
            else:
                message_lines.append(line)
        else:
            if line.strip():
                numstat_lines.append(line)
        i += 1

    message = "\n".join(message_lines).strip()

    # Parse numstat lines: additions\tdeletions\tpath
    files: list[dict[str, str]] = []
    total_insertions = 0
    total_deletions = 0
    for nline in numstat_lines:
        parts = nline.split("\t", 2)
        if len(parts) == 3:
            add_s, del_s, path = parts
            # Binary files show '-' for both counts.
            add = int(add_s) if add_s.isdigit() else 0
            dele = int(del_s) if del_s.isdigit() else 0
            total_insertions += add
            total_deletions += dele
            files.append({"path": path.strip(), "status": "M"})

    # Determine per-file status from git diff-tree.
    # git diff-tree --name-status gives A/M/D/etc. per file.
    try:
        diff_tree_out = _run_git(
            ["git", "diff-tree", "--name-status", "-r", sha],
            cwd=repo_path,
        )
        status_map: dict[str, str] = {}
        for dt_line in diff_tree_out.splitlines():
            dt_parts = dt_line.split("\t", 1)
            if len(dt_parts) == 2:
                status_map[dt_parts[1].strip()] = dt_parts[0].strip()
        for f in files:
            if f["path"] in status_map:
                f["status"] = status_map[f["path"]]
    except RuntimeError:
        pass  # fall back to "M" for all

    return {
        "context": {
            "ref": ref,
            "sha": sha,
            "author": author,
            "email": email,
            "date": date,
            "message": message,
            "parents": parents,
            "files": files,
            "stats": {
                "files_changed": len(files),
                "insertions": total_insertions,
                "deletions": total_deletions,
            },
        }
    }


def _is_numstat(line: str) -> bool:
    """Return True if the line looks like a git numstat line (digits TAB digits TAB path)."""
    parts = line.split("\t", 2)
    if len(parts) < 2:
        return False
    return parts[0].isdigit() or parts[0] == "-"


@plugin.capability(id="git.diff.fetch")
def diff_fetch(inputs: dict[str, Any]) -> dict[str, Any]:
    """Fetch unified diff between from_ref and to_ref.

    Returns: {diff: unified diff string}
    """
    repo_path: str = inputs["repo_path"]
    from_ref: str = inputs["from_ref"]
    to_ref: str = inputs["to_ref"]

    # Validate repo.
    _run_git(["git", "rev-parse", "--git-dir"], cwd=repo_path)

    # Resolve both refs to ensure they exist.
    _run_git(["git", "rev-parse", from_ref], cwd=repo_path)
    _run_git(["git", "rev-parse", to_ref], cwd=repo_path)

    diff = _run_git(
        ["git", "diff", f"{from_ref}..{to_ref}"],
        cwd=repo_path,
    )
    return {"diff": diff}
