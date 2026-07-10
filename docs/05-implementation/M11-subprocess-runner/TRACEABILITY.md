# M11 — Traceability
Every task below has a card; every AC row maps to a checklist row. Primary: IMP §27.M11.

| T | Task | Source coordinates | Card |
|---|---|---|---|
| T1 | TDS-04 `docs/SUBPROCESS_PROTOCOL.md` authored (envelope, error mapping, timeout, argv rule) | IMP §12 TDS-04 row; Blueprint §25; SPEC §1 pins | C1 |
| T2 | Golden protocol files (single wire truth) | IMP §27.M11 Risk row; SPEC §1 | C1 |
| T3 | `internal/runner/subprocess` SubprocessRunner (spawn, one-shot JSON exchange, timeout kill, error mapping) | IMP §27.M11 Obj; Blueprint §25; engine.Runner (M06) | C2 |
| T4 | sdk runtime wiring: `StepTypeSubprocess` → SubprocessRunner | IMP §6 extension point; sdk/runtime.go:92 | C2 |
| T5 | Go conformance + behavior tests vs goldens + fixture executables | IMP §19 (golden files); SPEC §2 | C2 |
| T6 | `python/awis-step` `awis_step` lib: `@step` + `StepContext/StepResult/StepError` + `step.serve()` | Blueprint §25 example verbatim; FR-SDK-10 | C3 |
| T7 | pytest suite vs the SAME goldens; `make pytest` in CI | IMP §27.M11 Merge row | C3 |
| T8 | e2e keystone: Python step inside harness workflow; outputs flow to next step | IMP §27.M11 Val row; FR-SE-02, FR-SDK-10 | C3 |

## Notes / dispositions
- **FR-SE-02 languages:** protocol is language-neutral (stdin/stdout JSON); Python is the V1
  reference implementation (`awis-step`); shell/TypeScript satisfy the protocol without a
  library (Blueprint §25 language table; PRD FR-SE-02 wording covered by TDS-04 §scope note).
- **One-shot process model** is a TDS-04 decision (CE, A-INIT): Blueprint §25 says "server",
  but request/response envelope + timeout behavior (IMP §12 TDS-04 row) with one process per
  execution is the minimal V1 mechanism; long-lived processes arrive with M12's lifecycle FSM.
  Recorded here as the disposition of that wording tension (not a CONTRA — Blueprint §25 is
  illustrative prose, TDS-04 is the designated normative home).
- **No TypeScript/shell libs in V1** — post-V1 nicety, same status as PyPI publication
  (IMP §18 L325).

## Execution record (appended during B-BUILD/C-VERIFY)
- C1 (5f3cb87, 2026-07-10): docs/SUBPROCESS_PROTOCOL.md (TDS-04) authored — SPEC §1 pins
  transcribed verbatim; 6 goldens at internal/runner/subprocess/testdata/protocol/; doc.go
  package stub. make build/test/lint/race/e1 + docs-lint green. No deviations.
- C2 (febae93, 2026-07-10): internal/runner/subprocess SubprocessRunner (whitespace argv, Setpgid
  process group + SIGKILL watcher, one-shot exchange, full TDS-04 error mapping). 13 tests: 6
  golden-conformance + 7 behavior (timeout kill ≤200ms). sdk/runtime.go = one runners-map line
  + import. Harness route test in sdk/testing/subprocess_route_test.go (NEW file; SPEC §2
  sanctioned alternative — import cycle). deadlineFormat fixed 9-digit nanos (RFC3339Nano trims
  zeros vs golden literal). make build/test/lint/race/e1 green; go.mod diff empty.
- C3 (145a266, 2026-07-10): awis_step lib (step registry/@step/StepContext/StepResult/StepError/
  serve one-shot; stdlib only, py>=3.10); pytest 8/8 vs the shared goldens (conftest repo-root
  traversal, no copies); make pytest target; e2e_python_test.go TestE2E_PythonStepOutputsFlow.
  make build/test/lint/race/e1 + pytest green; go.mod diff empty. Deviation none per report;
  pyproject pytest pythonpath config noted (config, not a dep).
- CE disposition (Fable, 2026-07-10): checklist row "outputs flow to the next (native) step" —
  C3's e2e chains Python→Python; the subprocess→NATIVE flow is proven by C2's harness route
  test (sdk/testing/subprocess_route_test.go). Jointly the row's FR-SE-02/FR-SDK-10 intent
  (IMP §27.M11 Val: "outputs flow to the next step" — no native qualifier in the IMP) is
  satisfied; the "(native)" qualifier was compilation prose, not IMP text. Verifier should
  treat the pair as the row's evidence.
