# Engine Hardening

The engine-hardening program: `engine-hardening` branch, base `7146214`,
29 commits.

Its objective was to stop auditing AWIS abstractly and *fix* the engine,
producing a trustworthy integrated runtime that is GUI-ready. It closed 25
CRITICAL and IMPORTANT defects, built the binary-level integration tier from
nothing, and ended with both gates green.

## The documents

| File | What it is |
|---|---|
| [01-PRE-REVIEW-REPORT.md](01-PRE-REVIEW-REPORT.md) | **Report 1** — everything fixed in the main program, with live evidence. State at commit `4c483dd`. |
| [02-POST-REVIEW-REPORT.md](02-POST-REVIEW-REPORT.md) | **Report 2** — the three-agent adversarial review, the eight further defects it found, and the correction it forced on Report 1's own evidence. |
| [ENGINE_HARDENING_PLAN.md](ENGINE_HARDENING_PLAN.md) | Root-cause analysis, the defect→commit map, the dependency graph and execution order, and the verification rule every fix had to satisfy. |
| [ENGINE_FREEZE_REPORT.md](ENGINE_FREEZE_REPORT.md) | The freeze verdict, full command and feature re-audit, GUI-readiness assessment, and the defect evidence table. |

Read Report 1 for what was wrong and how it was fixed; Report 2 for what
survived that fixing and had to be caught by someone else.

## Standing conclusions

**The engine is ready to freeze.** Zero open CRITICAL or IMPORTANT defects;
`make verify` and `make integration` both green; 21 binary-level integration
tests; every fix carrying a regression test verified load-bearing by reverting
the fix and observing red.

**The GUI is not ready to start**, and is blocked on exactly two things: there
is no global event cursor (`sequence_num` is per-instance, `event_id` is an
unordered UUID, and `VACUUM` can renumber `rowid`), and no
definition → YAML serializer anywhere in the repo, so a visual editor cannot
round-trip. Hot registration, which looked like a third blocker, is not one —
the engine loads definitions from storage on demand.

**No `v1.0.0` tag**, and three decisions belong to the founder: the `"default"`
namespace overloading, whether the `global_seq` migration lands in V1, and
whether to ship the intelligence path without a live-provider smoke test.

## The two findings worth remembering

**Concurrent `awis submit` failed with `SQLITE_BUSY`** despite WAL and
`busy_timeout` both being set — which is exactly why nobody had looked there. A
read-then-write transaction under DEFERRED locking begins as a *reader*, and in
WAL mode the write upgrade fails immediately without honouring `busy_timeout`.
No in-process unit test can reach this; it needs separate processes contending
for one file, and it surfaced within minutes of the concurrency probe existing.

**A green test suite proved nothing, four different ways.** `TestMain` in a
non-test file (so it never ran), polling on a status that does not change
across the transition under test, tests that encoded a defect as the expected
behaviour, and `t.Fatalf` from a spawned goroutine (which does not fail a
test). A fifth was found by the reviewers: a B-4 fixture whose failing step was
the *initial* step, making it structurally immune to the defect it was named
after.
