# AWIS Beta Baseline — Operating Protocol

Status: ACTIVE
Established: 2026-09-18
Scope: repository stabilization, feature verification, defect elimination,
milestone closure, documentation consolidation, Beta Baseline declaration.
Not in scope: architecture redesign, scope expansion, new features.

This document governs the Beta Baseline program only. It does not amend the
PRD, the architecture, or any milestone contract.

---

## 1. Authority

The Director decides. Subagents supply evidence and perform approved changes.
No subagent may declare a milestone closed, a defect resolved, or a feature
verified. Those are Director decisions made against evidence.

### Evidence hierarchy (highest first)

1. Current repository source code
2. Current repository tests
3. Current runtime behavior
4. Current CI behavior
5. Current milestone contracts
6. Current implementation specifications
7. Current architecture documents
8. Current PRD
9. Historical reports
10. Assumptions

Rules:
- Repository reality overrides report claims.
- Runtime reality overrides documentation claims.
- A passing test is evidence of a passing test, not of correctness.
- Reports are evidence, not truth.
- A document is not correct because it exists, is newer, or is confident.
- Missing evidence is an UNKNOWN, not a failure and not a pass.

### Claim rule

Every entry in the Current Truth Ledger carries a proof token: a file:line, a
command and its output, a commit SHA, or a test name. An entry without a proof
token is an UNKNOWN by definition.

---

## 2. Roles

### Director (this session)
- Decides. Holds the ledger. Assigns work.
- Does not perform bulk repository searches.
- Does not edit code or run implementation.
- Authors only governance artifacts: this protocol and the Current Truth Ledger.
- Merges subagent evidence into one place. Never spawns an investigation into a
  question already answered in the ledger.

### Investigator — Haiku, low effort, READ-ONLY
Purpose: discovery. Search, inventory, traceability, dependency mapping,
evidence gathering.

Constraints:
- MUST NOT create, edit, or delete any file.
- MUST NOT write a report file. Findings return inline to the Director.
- MUST NOT evaluate correctness, recommend action, or draw conclusions.
- MUST return facts with proof tokens, and write UNKNOWN rather than guess.
- Output is bounded; briefs state a line cap.

### Executor — Sonnet, medium effort, WRITE
Purpose: the only role permitted to modify the repository.

Constraints:
- Acts only on an explicit, Director-approved change specification.
- MUST NOT expand scope, refactor opportunistically, or fix unrelated issues
  found in passing. Such findings are reported back, not acted on.
- MUST NOT modify tests to make a failure disappear unless the test itself is
  the approved defect.
- Reports exactly what it changed, file by file, and what it did not change.

### Verifier — Sonnet, high effort, READ-ONLY
Purpose: proof. Testing, validation, milestone verification, regression
analysis, reproduction of defects.

Constraints:
- MUST NOT edit production code or tests.
- MUST run commands and report real output, including failures, verbatim.
- MUST return a binary evidence table: each claim PASS / FAIL / UNKNOWN with
  the command and output that establishes it.
- MUST NOT report PASS on the basis of a document. Only on execution.
- A verification that could not be run is UNKNOWN, never PASS.

---

## 3. Prohibited

- Audit chains and audit-of-audit cycles.
- Any report whose only purpose is reporting.
- Duplicate investigations; reinvestigating a proven question.
- Architecture redesign without evidence.
- Implementation before verification.
- Treating a report as reality.

There is exactly one living state document for this program: the Current Truth
Ledger. Subagents return findings inline; the Director merges them into the
ledger. New standalone status/verdict/audit documents MUST NOT be created.

---

## 4. Phase gates

- Phase 0 — Truth: maps built, state validated, ledger created.
- Phase 1 — Product health: feature verification matrix; each feature VERIFIED
  WORKING / VERIFIED FAILING / UNVERIFIED.
- Phase 2 — Defect elimination: locate -> reproduce -> fix -> revalidate +
  regression. Repeat until P0 and P1 are empty.
- Phase 3 — Milestone closure: M15, M16, M17 actual state; closure only where
  evidence justifies it.
- Phase 4 — Consolidation: authority drift eliminated, stale operational
  reports retired from active decision making.
- Phase 5 — Beta Baseline: declared only if evidence supports it.

A phase does not open until the prior phase's gate is recorded in the ledger.

## 5. Severity

- P0: data loss, corruption, silent wrong results, engine/CLI/API unusable for
  a core workflow, security exposure.
- P1: a named Beta surface is broken or materially incorrect; recovery,
  cancellation, retry, signal, or rebuild-state semantics violated.
- P2: degraded behavior with a workaround. Not a Beta blocker.
- P3: cosmetic, cleanup, deferred debt. Not a Beta blocker.

Beta Baseline requires P0 = 0 and P1 = 0, each closure backed by a Verifier
regression pass.
