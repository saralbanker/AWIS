# Refactoring Specialist — Playbook

## Fowler's smell taxonomy, with detection thresholds

| Smell | Detect by | Threshold | Fix |
|---|---|---|---|
| Long method | line count | >20 lines | extract named sub-functions |
| Large class | line/method count | >200 lines or >7 public methods | split by responsibility |
| Duplicate code | same logic in N places | 3+ occurrences (rule of three) | extract a shared function |
| Long parameter list | param count | >4 | introduce a parameter object |
| Divergent change | reasons a class changes | 2+ unrelated reasons | split by reason for change |
| Shotgun surgery | files touched per feature | 6+ across modules | consolidate into one module |
| Feature envy | external vs. own data used | >50% external | move the method to the data owner |
| Primitive obsession | domain concept as a raw string/number | e.g. email/money/status as `string` | introduce a value object |
| Data clumps | same fields passed together | 3+ functions share the same 3+ params | extract a type |

## Detection commands
```bash
find src/ -name "*.ts" | xargs wc -l | sort -rn | head -10   # long files
npx jscpd src/ --format "typescript" --min-lines 5           # duplicate blocks
```

## Safe-refactoring principles
Tests first (write characterization tests if none exist); one refactoring
per commit so a bad step is a one-line revert; behavior must not change —
if a test fails, the behavior changed, full stop; for large legacy
rewrites, use the strangler-fig pattern (build the new path alongside the
old, migrate callers incrementally, delete the old path only once nothing
uses it).

## When things go sideways

| Symptom | Likely cause | What to do |
|---|---|---|
| Tests fail after a refactor | behavior changed, even if unintentionally | `git revert` that step, re-examine what the code actually did |
| Extracted name is unclear | named after mechanism, not intent | rename: `doProcessing()` → `validateAndNormalizeInput()` |
| Code got more complex, not less | over-abstraction, wrong pattern | revert — duplication is cheaper than the wrong abstraction |
| No tests exist to refactor against | no safety net | write characterization tests first, don't refactor blind |
