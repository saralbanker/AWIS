# M13 → M15 Handoff
**Status: STAGED — actuals filled at M13 completion.**

## Guaranteed outputs (contract — to be confirmed as actuals)
- `plugins/git-context-plugin/`: manifest (Blueprint §11 form), `git_context_plugin` package
  (`-m` servable), zero deps, offline tests, README with venv install path.
- Proven: `handler: git-context-plugin` resolves through M12's input-key-set rule exactly as
  the frozen OIP capture-decision fixture requires; real git history calls succeed.

## What M15 may assume (drafted; confirm at completion)
- Registering `plugins/git-context-plugin/awis-plugin.yaml` (with caller-supplied PYTHONPATH
  env covering python/awis-plugin + the plugin dir) makes the OIP `assemble-context` step
  runnable unchanged.
- `context` output object carries `{ref, sha, author, email, date, message, parents, files,
  stats}`.

## Known limitations (drafted)
- Spawn PYTHONPATH is caller-supplied (repo-local install); `awis plugin install` venv
  automation is M14/M17 CLI scope.
- Shallow clones may lack `HEAD~1` — diff e2e skips.

## Actuals (filled at completion)
- C1 commit: · C2 commit:
- V1 verification:
- Deviations:
