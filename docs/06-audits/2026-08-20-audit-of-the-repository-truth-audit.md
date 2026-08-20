# AWIS — Independent Audit of the Repository Truth Audit

**Date:** 2026-08-20 · **Subject:** `docs/06-audits/2026-08-20-repository-truth-audit.md`
(on `origin/claude/awis-repository-audit-0y6xfs`) · **Audit tree:** `main` @ `98350f6`
**Method:** every conclusion re-derived from the git object store, the GitHub Actions API, and
executed gates in throwaway worktrees. The Repository Truth Audit was treated as a hypothesis.
No verdict below rests on its text, on `STATE.md`, or on any commit message.

---

## 1. Executive Summary

The Repository Truth Audit is **directionally right about the records and wrong about the code**.
Its central service — proving that the milestone branches carry no machine-checked evidence — holds
up. Its central engineering judgment does not.

Three findings dominate this review:

> **1. `main` @ `98350f6` ships a P0 workflow-hang defect, and the M15 change the audit rates as its
> highest risk (R-1, HIGH) is the fix for it.**
> Any workflow whose `fallback:` fires on a **non-initial** step never reaches a terminal state on
> `main` today. Both workflows in the repository that use `fallback:` have exactly that shape —
> including `examples/workflows/with-intelligence.yaml`, which is already merged. Proven by
> execution on two worktrees (§3.1).

> **2. The whole M15+M16+M17 stack merges into `main` with zero conflicts and reaches
> `verify: ALL GATES PASSED` after ~20 lines of mechanical fixes.** I executed this end to end. The
> audit rates these milestones 15% / 80% / 10% merge-ready and proposes a 14-step, three-branch
> rebaselining programme to reach the same place (§3.2).

> **3. "Rebaselining" is not an EEOS concept, and the reordering that requires it is the audit's own
> choice.** `rebaseline` appears in no canonical document. Nothing requires a milestone branch to be
> cut from `main`; `STATE.md` records stacking as the actual practice. Because `m16` is stacked on
> `m15`, merging the m16 *branch* drags M15 in regardless — which is precisely why the audit had to
> invent an extraction step (§3.3).

The audit's own headline is also self-superseding: it opens with "GitHub Actions has run exactly 3
times… all 3 on `main`", then discloses in §F-3 that its own push made run #4. There are **5** runs
today, 2 on the audit's branch. The durable claim — *no milestone branch has CI evidence* — survives
intact and is the report's best contribution.

**Bottom line:** the remaining work to a merge-ready trunk is roughly **1.5–2 days of machine work**,
almost all of it documentation and ledger repair, plus founder gate latency. It is not the multi-week
programme the audit's execution table implies.

---

## 2. Findings That Survived Review

| # | Finding | Status | How I verified it |
|---|---|---|---|
| S-1 | No milestone branch has ever been executed by CI | **CONFIRMED** | Actions API: 5 runs total, `head_branch` ∈ {main ×3, audit branch ×2}. Zero on m15/m16/m17. |
| S-2 | `main` @ `98350f6` is genuinely green on Linux **and** macOS, lint included | **CONFIRMED** | Run `32339388733`, step-level: step 8 `golangci-lint` `success` on both legs. |
| S-3 | M16's `[x] V-COMMON all ✅` tick is **false** | **CONFIRMED** | Pinned `golangci-lint@v2.12.2` at m16 tip → 3 issues (`anthropic.go:259` errcheck, `:339`/`:342` QF1012). V-COMMON step 3 requires `make lint`; PASS requires every row ✅. |
| S-4 | Local `golangci-lint` (2.5.0/go1.25.1) cannot lint a go1.26 module; CI's pin can | **CONFIRMED** | Exit 3 locally; v2.12.2 (built go1.26.7) runs clean. R3 is a ~90-second toolchain fix, not a standing blocker. |
| S-5 | M17 issues 1 & 2 (`go:embed` dotfile exclusion; `../../../../` path depth) | **CONFIRMED** | Reproduced, fixed, re-ran green. |
| S-6 | M17 issue 4 (stale version goldens) self-resolves on merge with `main`'s `d5ac004` | **CONFIRMED** | Post-merge test run: `TestVersionHuman`/`TestVersionJSON` absent from the failure set. |
| S-7 | S5 (`apps/oip` module isolation) is genuinely required and belongs with M15 | **CONFIRMED** | `GOWORK=off go build ./...` in the M15-merged tree fails: `no required module provides package github.com/awis/awis/sdk`. Fixed by `require`+`replace`. |
| S-8 | `docs-lint` is an exact-set check that **will** go red for M15/M16/M17 | **CONFIRMED** | `scripts/docs-lint.sh` compares sorted filename sets (`[ "$have" = "$want" ]`); extras fail, not just gaps. Executed on all three merge candidates. |
| S-9 | `main`'s `STATE.md` is materially untruthful | **CONFIRMED** | `git merge-base --is-ancestor f6aa755 origin/main` → yes; ledger still shows M10–M14 `E-MERGE (blocked on founder)` and `DONE-MILESTONES` stopping at M09. |
| S-10 | `m17-c3-wip` is the sole copy of C3 work; preserve it | **CONFIRMED** | One commit `153772c`, branch-only. Tag before touching. |
| S-11 | M18 README-only is compliant by design | **CONFIRMED** | `docs-lint.sh` fires rule 2 only when a dir holds >1 `.md`. |
| S-12 | `cards/` is exempt and is the sanctioned home for gate artifacts | **CONFIRMED** | Script permits exactly one subdirectory, named `cards`; `*.md` count is `-maxdepth 1`. |

---

## 3. Findings That Did Not Survive Review

### 3.1 R-1 — "M15's P2 change is the highest risk in the repository"

**Verdict: INCORRECT in mechanism, INVERTED in direction, and right in severity for a reason the
audit never states.**

The audit describes P2 as injecting a sentinel so that "expressions referencing a fallback-activated
step's outputs now resolve to zero-values instead of failing." Both halves are wrong:

- `FR-WD-05` is frozen and explicit (`internal/expr/template.go:102-110`): template resolution
  **never fails**. There was no error path to convert.
- I traced `resolveRef` → `Env.lookup` for `{{steps.X.outputs.foo}}` on a fallback-activated step.
  **Pre-P2:** `StepOutputs[X]` absent → `(nil,false)` → `""` + one warning. **Post-P2:**
  `StepOutputs[X]` is `{}` → `walk({},["foo"])` → `(nil,false)` → `""` + the *same* warning with the
  *same* reason string. **Zero behavioural change.**

What P2 actually does is repair a liveness defect. `completedSet()` (`internal/engine/transition.go:37`)
derives "completed" from the keys of `inst.Variables`. Pre-P2 a fallback-activated step is removed
from `CurrentSteps` but never enters `Variables`, so it is neither *completed* nor *running*.
`activatableFor()` therefore returns it on every tick forever, `claim()` loses every time, and
`tick.go`'s completion check — `if len(e.activatableFor(dv, inst2)) != 0 { return nil }` — can never
fire.

Measured on two worktrees, same probe, same command:

| Probe topology | `main` (pre-P2) | `m15` (post-P2) |
|---|---|---|
| non-initial step whose `fallback:` fires | **no terminal status in 200 ticks** | `completed` |
| convergent join after fallback (`a→b(fb)`, `b→c`, `fb→c`) | **no terminal status in 200 ticks** | `completed` |

Independently corroborated: reverting only the two P2 hunks on `m15` makes `apps/oip`'s
`TestOIPSystemE2E` fail with `last status="running" step="manual-entry"` and repeated `claim lost` on
`draft-entry` — the same signature.

This is not hypothetical for this repository. `examples/workflows/with-intelligence.yaml`
(**already on `main`**) and `apps/oip/workflows/capture-decision.yaml` both attach `fallback:` to a
non-initial step feeding a convergent join. The existing green test `TestRetryExhaustThenFallback`
misses it only because its failing step is the `InitialStep`, which the zero-inbound guard excludes.

**The real risk, which neither audit names.** P2 contradicts a frozen EDR verbatim.
`docs/edr/edr-007-projection-rules.md` §2 tabulates `StepFallbackActivated → variables: —` and §3
states "**No other event type mutates variables**", and says it is "superseded only by an explicit G2
amendment." No such amendment is recorded. `G3_BRIEF.md`, `BOUNDARY_EVIDENCE.md` and `TRACEABILITY.md`
cite EDR-007's *forward≡rebuild* clause while leaving its *variables* clause unaddressed.

Two observable consequences also go undocumented:
1. `buildEnv` sets `stepStatus[k]="completed"` for every `Variables` key, so a step that **fell back**
   now reports `steps.X.status == 'completed'`. Pre-P2 it was null. Conditions `== 'completed'` flip
   false→true and `== null` (grammar C10/C11) flip true→false.
2. `WorkflowCompleted.outputs` gains a `{"X":{}}` key when a fallback step is also a final step —
   a frozen EDR-011 §2 payload change. Measured directly:
   pre-P2 `{"outputs":{"fb":{...}}}` → post-P2 `{"outputs":{"a":{},"fb":{...}}}`.

**Process defect:** P2 was executed under no card. `cards/M15-C3r.md` does not mention the engine
fix; it is narrated after the fact in `TRACEABILITY.md`/`HANDOFF.md`/`G3_BRIEF.md`. EEOS rule 3
requires a revision card, never an unchartered edit.

**Latent defect found in passing, in neither audit:** both shipped workflows gate their fallback
route on `steps.X.status == 'fallback'`. `docs/EXPRESSION_GRAMMARS.md:35` defines status as
`pending | running | completed | failed | cancelled` — there is no `fallback` value, and `buildEnv`
never emits one. Those transitions are dead in both files; the fallback route only works because
`routeTerminalFailure` adds the target via `addPending`.

**Risk classification: MEDIUM runtime / HIGH governance debt.** Blast radius is bounded and net
beneficial; the exposure is to the *record*, not the runtime. **Not merging P2 is strictly riskier
than merging it**, because `main` keeps the hang.

**Is adversarial review justified?** Yes, but scoped to three questions, not an open-ended pass:
(a) is `completed` the right projected status for a fallback-origin step, or does the enum need a
sixth value; (b) is the `WorkflowCompleted.outputs` delta acceptable under EDR-011 §2; (c) does
EDR-007 get amended or does the sentinel move to a non-`variables` carrier.

**Is founder review necessary?** Yes — but the *ask is wrong*. Today P2 is buried inside a QG-4
platform-boundary question. It should be raised as an **EDR-007 amendment (G2-class)** plus the
status-semantics decision. That is a sharper, faster question than the one currently in front of the
founder.

**A missing alternative the audit never considered.** The sentinel is carried in `Variables` because
that is the only *persisted* per-step map — `e.pending` is in-memory and does not survive rebuild.
A distinguishable carrier (a projected `fallback_activated` set, or a status value the grammar
defines) would deliver the join-gate fix with **no** expression-visible change. That is the correct
long-term design; P2 is the correct Baseline-V1 expedient. Record the debt, do not pretend it is absent.

### 3.2 "M15 15% / M16 80% / M17 10% merge-ready"

**Verdict: OVERSTATED across the board.** I merged the stack into `main` and drove it to green.

`git merge --no-commit --no-ff origin/m17-c3-wip` into `main` → **clean, 0 conflicts, 126 files,
+10,272/−38** — and because the branches are stacked, that single merge carries M15 + M16 + M17.
Applying the complete fix inventory below produced `verify: ALL GATES PASSED`, exit 0:

| # | Fix | Size |
|---|---|---|
| 1 | `gofmt -w .` | 5 files |
| 2 | S5 — `apps/oip/go.mod` `require`+`replace` + tidy | 1 file |
| 3 | S3 — `anthropic.go` errcheck + 2× QF1012 | 3 hunks / 6 lines |
| 4 | `//go:embed all:scaffold` | 1 line |
| 5 | `init_test.go` path depth `../../../../` → `../../` | 3 lines |
| 6 | `go test ./cmd/awis/ -run TestInitGolden -update` | generated, 2 files |

That is the entire code-side remainder. `gofmt-check`, `vet`, `lint` (both modules, 0 issues),
`oip-isolation`, `build`, `test` (incl. the 32 s OIP E2E), `race`, `e1` — all green.

Measured separately, M15 alone merged into `main` also reaches `verify: ALL GATES PASSED` with only
fixes 1 and 2.

`docs-lint` is the only remaining red, and it is documentation: 12 contract files across M15/M16/M17.

### 3.3 "Rebaseline M16 onto main" as the highest-value next action

**Verdict: OVERSTATED, and it solves a problem the audit created.**

- **`rebaseline` is not an EEOS term.** Zero hits across `EEOS.md`, `AWIS_EEOS.md`, the AEO, IKB and
  IMP. AEO §13 requires "one milestone = one branch = one squash-merged PR" and human-only merges.
  It does not require branching from `main`, and `STATE.md` documents the opposite as practice:
  `BRANCH: m15-oip-on-awis (stacked on m14-core-cli)`, "M16 proceeds stacked", "M17 proceeds stacked".
- **"M16 has zero founder dependencies" is true of its delta and false of its branch.** I merged
  `origin/m16-anthropic-adapter` into `main`: clean, 57 files — and `apps/oip/handlers.go` is present.
  Merging the M16 *branch* merges M15.
- **Cherry-pick is the worst of the three paths**, not a better one: the first M16 commit (`3518380`)
  conflicts on `STATE.md` with real conflict markers.
- **Rebaselining does not avoid the S3 fix.** I confirmed on the direct-merge tree that the same 3
  issues appear at `anthropic.go:259/339/342`. (An intermediate report claimed that tree lints clean;
  it was fighting a shared golangci-lint cache across worktrees. Re-run with isolated `GOCACHE`
  and `GOLANGCI_LINT_CACHE`, it does not.)
- **Rebaselining adds work the audit does not cost.** `m16`'s `STATE.md` contains M15's `E-MERGE`
  block, so an M16-first rebaseline needs a hand-edited ledger. And the audit's own instruction to
  exclude `STATE.md` from the patch would land M16 on `main` with **no ledger entry at all** —
  a direct violation of EEOS rule 1 ("no phase or card transition is real until written to STATE.md").

### 3.4 "M15 and M16 verification verdicts should be withdrawn"

**Verdict: PARTIALLY CORRECT — a real defect, over-corrected, resting on a rule that does not exist.**

- **No EEOS text requires withdrawal.** I searched `EEOS.md`, `AWIS_EEOS.md`, the AEO, IKB, IMP and
  `V-COMMON.md` for verdict withdrawal, voiding, revocation or re-verification. **NOT FOUND.**
  `WITHDRAWN` exists only in the *card* lifecycle. The nearest governing rule is
  "ledger contradicts repo → STOP — Fable wake" — which prescribes escalation, not blanket voiding.
  "Treat all branch V1 verdicts as void" is an invented obligation.
- **The two verdicts are not equivalent, and the audit treats them as one.** At each branch tip,
  with the pinned linter:
  - **m16 tip → 3 issues.** Its lint tick is substantively **false**. Confirmed.
  - **m15 tip → 0 issues, both modules.** Its lint tick is **unexecuted but substantively correct**.
  Voiding M15's verdict on M16's evidence is an over-correction.
- **Judging M15-V1 by today's `make verify` is anachronistic.** V-COMMON step 3 is
  `build · test · lint · race`. `gofmt-check` and `oip-isolation` entered the Makefile later
  (`be13cf9`/`a53e810`). M15's 20 gofmt-drifted files are real, but M15-V1 never ticked a gofmt row.
- **V-COMMON was in force.** It is byte-identical on `main`, `m15` and `m16`, was created whole in
  `b691965` (2026-07-08) with `make lint` already in step 3, and has never been edited. The rule
  M15/M16 were measured against is not in dispute.

**Minimum defensible verification strategy** — evidence can be reconstructed without re-running
anything on the branches, because the branch tips will never be merged as-is:

1. **Do not withdraw. Annotate.** Mark M15-V1 and M16-V1 `SUPERSEDED — V-COMMON step 3 unexecuted
   (toolchain defect R3)` in each module's `TRACEABILITY.md`. Keeps the evidence chain honest without
   inventing a rule.
2. **Verify the merge candidate, not the branches.** One CI run on the integration branch is strictly
   stronger evidence than three re-run branch V1s — it proves the tree that will actually land.
   CI on non-main branches is now demonstrated working (2 runs on the audit's own branch).
3. **One V2 card** citing V-COMMON plus all three milestones' checkpoints, dispatched to
   `awis-verifier` on the integration branch.
4. **Pin `golangci-lint` in the Makefile** to CI's `v2.12.2` so local and CI verdicts mean the same
   thing. This is the fix that prevents recurrence; everything else is cleanup.

### 3.5 "M17-C3 is genuinely unfinished, not merely mis-typed"

**Verdict: OVERSTATED.** The missing `init.txt`/`init.json` goldens are generated by the repository's
own mechanism: `main_test.go:15` defines `-update`, and both tests route through `checkGolden()`.
I ran `go test ./cmd/awis/ -run TestInitGolden -update`, then re-ran without it — both pass. This is
a skipped artifact-generation step, not unwritten functionality. Every named acceptance criterion in
`cards/M17-C3.md` — byte-identical embedded examples, `--force` guard, the init→start→submit→trace
rehearsal system test — passes once the two one-line fixes land.

**A blocker both audits missed:** `cmd/awis/errors.go:37` calls `os.Exit` from `fail()`. When
`TestInitCreatesScaffoldFiles` hits the embed error it kills the test binary, so roughly 130 later
tests in `cmd/awis` **never run**. The unpatched branch reports only 2 failures because the rest were
never reached. Neither audit's failure inventory was measured on an unmasked binary.

### 3.6 "M17 is dependent on M15"

**Verdict: INCORRECT.** `docs/05-implementation/M17-full-cli-init/README.md:5` states
`**Depends on:** M14, M12; soft on M16`. M15 appears nowhere in M17's README, IMPLEMENTATION_SPEC or
VALIDATION_CHECKLIST. No `.go` file added by M17's own commits references `apps/oip`, `package oip`,
`OIP_DB` or `record.Entry`. M15 is an ancestor only because milestones are cut sequentially off the
prior close commit. M17's own commits intersect M15's changed-file set in exactly two files:
`cmd/awis/errors.go` (both edit the usage banner, no functional coupling) and `STATE.md`.

### 3.7 "The proposed namespace extraction"

**Verdict: NOT IN EVIDENCE.** No such proposal exists. `namespace extraction` and
`extract.{0,20}namespace` return zero matches across `main`, `m15`, `m16`, `m17-full-cli-init`,
`m17-c3-wip` and `milestone/M14` — and the phrase does not appear in the Repository Truth Audit
either. The only related artifact is the `--namespace` CLI flag added by M15's C3r card, which is
unrelated in both name and content. I cannot validate an architectural solution that is not in the
repository; if it exists it lives outside version control and needs to be produced before it can be
assessed.

### 3.8 "Four divergent STATE.md ledgers" (C-10) and the BOUNDARY_EVIDENCE fold

**C-10: OVERSTATED.** The four copies are not divergent; they are a strict append-only chain. Each
branch's `STATE.md` is `main`'s plus its own milestone blocks — exactly what stacked branches produce.
They merge cleanly *in stack order*. They only conflict if you reorder, which is what the audit
proposes. (C-1, C-2 and C-3 are separately **CONFIRMED**: `main`'s ledger is genuinely false.)

**The fold: INCORRECT, and unnecessary.** The audit proposes renaming `BOUNDARY_EVIDENCE.md` to
`DEPENDENCY_MAP.md`. IKB §2 defines `DEPENDENCY_MAP.md` as "Upstream consumed, downstream produced,
critical-path position" — a QG-4 boundary proof is not that genre, and IKB §3 requires transcription,
never paraphrase, with a determinism claim that two independent compilations differ only in prose.
A folded file breaks that.

The cheaper and contract-exact fix, which I executed and verified:

```
git mv .../BOUNDARY_EVIDENCE.md .../cards/M15-BOUNDARY-EVIDENCE.md
git mv .../G3_BRIEF.md          .../cards/M15-G3.md
compile DEPENDENCY_MAP.md per IKB §3/§4     → docs-lint: OK   (exit 0)
```

Two moves into the already-exempt `cards/` directory plus one properly compiled file. No content
lost, no genre violation, no EEOS §13 amendment. The audit is right that an amendment is the wrong
answer; its own alternative is simply not the cheapest correct one.

### 3.9 "GitHub Actions has run exactly 3 times, all 3 on main"

**Verdict: PARTIALLY CORRECT — superseded by the audit's own §F-3.** There are 5 runs; 2 are on
`claude/awis-repository-audit-0y6xfs` (`32351838137`, `32351890897`), both `success`. The durable
claim — no *milestone* branch has CI evidence — is **CONFIRMED** and is the audit's best finding. The
headline should have been written as the audit's §F-3 already knew it to be.

---

## 4. Hidden Assumptions

| # | Assumption the audit makes silently | Why it matters |
|---|---|---|
| H-1 | A branch's *delta* being independent means its *branch* is mergeable independently | False here: `m16` is stacked on `m15`; merging it merges M15. This single assumption generates the entire rebaselining programme. |
| H-2 | A false verification tick means the underlying gate fails | Holds for M16 (3 issues), fails for M15 (0 issues). Drives an unnecessary M15 re-verification. |
| H-3 | Today's `make verify` is the standard M15/M16-V1 were measured against | `gofmt-check` and `oip-isolation` postdate those verdicts. V-COMMON step 3 is build/test/lint/race. |
| H-4 | EEOS obliges withdrawal of a verdict whose evidence proves defective | No such rule exists anywhere in canon. |
| H-5 | Excluding `STATE.md` from a rebaseline patch is free | It lands a milestone on `main` with no ledger entry — EEOS rule 1 violation. |
| H-6 | Founder gate latency is the binding constraint, so founder-gated work goes last | Inverts IMP's declared critical path and defers a P0 fix that `main` needs now. |
| H-7 | A change touching frozen surfaces is a risk to be contained | Here it is a defect being repaired. The audit never ran the code to find out. |
| H-8 | "Fable" can be remapped to "Opus 5" | `CLAUDE.md`'s frozen policy **forbids** Opus and keeps Fable dormant except on human request. EEOS rule 10 gives Fable A-INIT/D-CLOSE/gate-brief duty — wider than "architecture review". The remap has no textual basis and the audit's own step 10 assigns Opus 5 work. |
| H-9 | The "previous audit" is a citable baseline | No prior repository audit exists in git history on any branch. R2/R3/R4 are referenced, not attributable. |

---

## 5. Audit The Audit — Classification Table

| Audit finding | Verdict | Rationale (evidence) |
|---|---|---|
| F-1 Branch topology matches briefing | **CONFIRMED** | `git ls-remote`; all five SHAs match; common merge-base `827a084`. |
| F-2 `main` green on Linux + macOS incl. lint | **CONFIRMED** | Run `32339388733`, step 8 `golangci-lint` `success` on both legs. |
| F-3 CI never ran on a milestone branch | **CONFIRMED** (headline **PARTIALLY CORRECT**) | True for milestone branches. "Exactly 3 runs, all on main" is superseded by the audit's own §F-3; 5 runs exist. |
| F-4 R3 lint broken locally, works in CI | **CONFIRMED** | Exit 3 locally with 2.5.0; v2.12.2 clean. Audit's two corrections to R3 both hold. |
| F-5 R2 false PASS ticks | **PARTIALLY CORRECT** | M16's tick is false (3 issues). M15's tick is unexecuted but substantively correct (0 issues at tip). Not equivalent. |
| F-6 M16 rebaseline fails lint, passes after fix | **CONFIRMED** | Reproduced independently, including on the direct-merge tree. |
| F-6 "M16 can be rebaselined directly onto main" as best path | **OVERSTATED** | Stack-order merge needs no extraction; cherry-pick conflicts; rebaseline adds a ledger hand-edit and a fresh V2. |
| F-7 M17 issues 1 & 2 | **CONFIRMED** | Reproduced and fixed. |
| F-7 M17 issue 3 "C3 genuinely unfinished" | **OVERSTATED** | Goldens generate mechanically via `-update`; all card acceptance criteria pass. |
| F-7 M17 issue 4 self-resolving | **CONFIRMED** | Absent from post-merge failure set. |
| F-7 "root-cause analysis was 2 of 4 complete" | **MISSING CONTEXT** | It was 2 of 5. `os.Exit` in `fail()` masks ~130 tests; neither audit measured an unmasked binary. |
| F-8 R4 not patch-equivalent | **CONFIRMED** | `m17-full-cli-init`'s `ci.yml` is a distinct, narrower blob; `main`'s is strictly newer. |
| F-8 S3→M16, S5→M15 must be re-derived | **CONFIRMED** | Both proven required by execution. |
| F-9 P2 is symmetric across live and rebuild | **CONFIRMED** | Same key, value and placement. The extra nil-guard in `emit.go` is defensive only — `projectInstance` seeds `p.variables` at `rebuild.go:219`. |
| F-9 P2 risk = "expressions resolve to zero-values instead of failing" | **INCORRECT** | FR-WD-05 is total; resolved value *and* warning are identical pre/post. |
| R-1 P2 is the highest risk (HIGH) | **INCORRECT in direction** | P2 fixes a P0 hang on `main`. Real risk is the undisclosed EDR-007 §2/§3 contradiction and the `status=='completed'` semantic — neither named. |
| R-2 Entire M10–M16 verification history unproven | **CONFIRMED** | No CI, no lint. Mitigation "main itself is proven green" also holds. |
| R-3 `docs-lint` will go red for M15/M16/M17 | **CONFIRMED** | Executed on every merge candidate. |
| R-4 Preserve `m17-c3-wip` | **CONFIRMED** | Sole copy; tag before touching. |
| R-5 Four divergent ledgers | **OVERSTATED** | Append-only chain, not divergence. The underlying C-1/C-2/C-3 falsehoods are confirmed. |
| R-6 Pin golangci-lint | **CONFIRMED** — and under-prioritised | This is the control that prevents recurrence; it is listed as LOW. |
| R-7 Dropping `be13cf9` naively loses S3+S5 | **CONFIRMED** | Both proven required. |
| §7 "Rebaseline, never merge; three fresh branches" | **INCORRECT** | Not an EEOS concept; contradicts documented stacking; the full stack merges clean and goes green. |
| §8 Fold `BOUNDARY_EVIDENCE` → `DEPENDENCY_MAP` | **INCORRECT** | Violates the IKB §2 genre and §3 determinism. Two `git mv`s into `cards/` + a compiled map gives `docs-lint: OK`. |
| §8 `G3_BRIEF` → `cards/M15-G3.md` | **CONFIRMED** | Correct, and verified to satisfy `docs-lint`. |
| §8 M15's two extras violate the contract | **CONFIRMED** | Exact-set check; M00–M14 all carry exactly 7 (the 15/15 precedent holds). |
| §8 Fable → Opus 5 remap | **INCORRECT** | `CLAUDE.md` forbids Opus and restricts Fable to human-invoked review; no canonical text equates them. |
| §9 Order M16 → M17 → M15 | **INCORRECT** | Puts IMP's only critical-path milestone last and defers a P0 fix. |
| §9 "Send the founder the G3/TDS-06 brief on day 0" | **CONFIRMED** — the audit's best recommendation | Highest-latency, human-gated. Should also be re-scoped per §3.1. |
| §10 "M16 is the highest-value next action" (High) | **OVERSTATED** | Rests on H-1. |
| §10 "Why CI never ran" (Medium) | **CONFIRMED as calibrated** | Workflow object registered `06:07:19Z`, 6 s before run #1; all milestone HEADs predate it. Correctly rated inferred. |

---

## 6. Recommended Execution Plan

Ordering principle: **merge in stack order, because the stack is already clean.** Everything the audit
achieves by extraction, sequence achieves for free.

| # | Step | Owner | Clears |
|---|---|---|---|
| **0a** | Tag `preserve/m17-c3-wip` → `153772c`. Never force-push that branch. | Sonnet | R-4 |
| **0b** | Send the founder a **re-scoped** brief: QG-4 + TDS-06 **+ the EDR-007 §2/§3 amendment and the `status=='completed'` decision** (§3.1). Day 0, in parallel. | human | G-1, G-2, and the real P2 question |
| **0c** | Pin `golangci-lint@v2.12.2` in the Makefile to match `ci.yml`. | Sonnet | R-6 — the recurrence control |
| **1** | Reconcile `main`'s `STATE.md` against `f6aa755` (M10–M14 merged; `DONE-MILESTONES` through M14). Per EEOS this is a Fable-wake ledger/repo contradiction — escalate, do not silently repair. | Fable / human | C-1..C-3 |
| **2** | Cut `m15-integration` from `origin/main`; merge `origin/m15-oip-on-awis`. Apply `gofmt -w` (3 files) + S5. Split P2 into its own commit with an EDR-007 note so the founder reviews it in isolation. | Sonnet | T-3 |
| **3** | Add the two missing engine tests: a **mid-workflow fallback + convergent join** case (the gap that hid the hang), routed through the existing `assertProjectionEquivalence` harness (`helpers_test.go:330`) — the replay-equivalence harness already exists, only the case is missing. | **Opus/Fable-grade** | T-4, R-1 |
| **4** | M15 docs: two `git mv`s into `cards/`, compile `DEPENDENCY_MAP.md` per IKB §3/§4. Verified to give `docs-lint: OK`. | Sonnet | P-1 |
| **5** | Push → first real CI on M15. Dispatch one **M15-V2** card citing V-COMMON + checkpoints. | verifier | P-3 |
| **6** | Founder: G3 + TDS-06 + EDR-007 verdict; squash-merge M15. | human | G-1..G-3 |
| **7** | Merge `origin/m16-anthropic-adapter`; apply S3 (3 hunks/6 lines); compile M16's 4 missing contract files. | Sonnet | T-1, P-1 |
| **8** | Push → CI → M16-V2 → founder squash-merge. | verifier / human | P-3, G-3 |
| **9** | Merge `origin/m17-c3-wip`; apply `all:scaffold` + path depth; `-update` the init goldens; compile M17's 4 contract files. | Sonnet | T-2, P-1 |
| **10** | Push → CI → M17-V1 → founder squash-merge. | verifier / human | P-3, G-3 |
| **11** | M18 A-INIT — materialize the 7 contract files. | Sonnet | — |

**Annotate, do not withdraw** (§3.4 step 1) folds into steps 4, 7 and 9 as one line each in
`TRACEABILITY.md`.

**Contingency, if the founder gate cannot clear soon.** Extract P2 alone — 15 lines, no OIP, no
`apps/oip` dependency — as a standalone engine hotfix to `main` with the EDR-007 amendment and the
mid-fallback-join test. That stops `main` shipping a hang without waiting on QG-4, which is a
question about OIP's boundary, not about the engine defect. This option does not exist in the audit's
plan and is the single largest available de-risking.

---

## 7. Recommended Merge Order

**M15 → M16 → M17 → M18.** Stack order. Three sequential squash merges, AEO §13 compliant, zero
extraction work, zero invented branches.

Reject `M16 → M17 → M15`: it requires rebaselining that exists only to escape it, hand-edits the
ledger, defers a P0 fix, and puts IMP's only critical-path milestone last.

---

## 8. True Critical Path

`IMPLEMENTATION_MASTER_PLAN.md:27` and `:221` declare it outright:

> **M0 → M1 → M2 → M3 → M6 → M7 → M8 → M11 → M12 → M13 → M15 → M18** (≈28 engineering days)

**M15 is on the critical path. M16 and M17 are not** — IMP:240 schedules them as parallel Week-6
lanes. M18's longest activity is "one week OIP dogfood overlapping"; OIP *is* M15. M15 therefore
gates the longest-duration item in M18, and the audit's ordering places it last.

| Path | Order | Why |
|---|---|---|
| **Shortest** | M15 → M16 → M17 | The stack already merges clean; ~20 lines of fixes reach `verify: ALL GATES PASSED`. Executed. |
| **Lowest-risk** | Same, with P2 split into its own commit + the mid-fallback test landing with it | Isolates the one semantically load-bearing change for review; unblocks main's hang first. |
| **Highest-confidence** | Same, integration branch pushed to CI before each squash-merge | CI on non-main branches is proven live; converts every verdict from self-reported to machine-proven. |

The three converge — which is itself the finding. They diverge only if the founder gate stalls, and
the contingency in §6 covers that case.

---

## 9. Estimated Remaining Effort

| Work | Effort | Basis |
|---|---|---|
| Code fixes (all 6, all three milestones) | **~1 h** | Executed end to end; ~20 lines total |
| 12 contract files + 2 moves + 1 compiled map | **~3–4 h** | IKB §3 rates compilation at ~30 min/module, mechanical |
| `STATE.md` reconciliation | **~1 h** | Ledger/repo contradiction — needs Fable-grade adjudication, not a rewrite |
| P2 review + mid-fallback-join test + replay equivalence | **~2–3 h** | Harness exists (`assertProjectionEquivalence`); only the case is missing |
| EDR-007 amendment draft | **~1 h** | One table row + one prose clause |
| 3 CI runs + 3 V-cards | **~1 h** wall clock | CI run ≈ 90 s |
| **Machine total** | **~1.5–2 days** | |
| Founder: G3, TDS-06, EDR-007, 3 squash merges | **latency, not effort** | Human-gated; start day 0 |

---

## 10. Final Verdict

**The Repository Truth Audit should be accepted as a records audit and rejected as an engineering plan.**

It did the repository a genuine service: it proved the milestone branches have no machine-checked
evidence, it caught M16's false lint tick with two independent proofs, it correctly identified S3, S5
and the `docs-lint` guard, and it was right that the founder brief is the long pole and should start
on day 0. Those findings stand.

But it never ran the code it rated riskiest. Had it executed one probe against `main`, it would have
found that `main` hangs on any non-initial fallback, that the change it was preparing to gate behind
a founder verdict and schedule last is the fix, and that the real exposure is an undisclosed EDR-007
contradiction it did not mention. Instead it reasoned from a diff, assigned HIGH to the wrong
mechanism, and built a 14-step programme on the assumption that an independent *delta* implies an
independently mergeable *branch* — an assumption the branch topology refutes, and the source of
almost every unnecessary step in its plan.

Repository truth, as measured: **the code is much closer to mergeable than either the records or the
audit suggest — and `main` is further from correct than either admits.**

The corrected order of business is:
1. `main` has a P0 hang. Fix it — merged with M15, or extracted as a hotfix if the gate stalls.
2. Merge in stack order. Nothing needs rebaselining.
3. Ask the founder the *right* question — EDR-007 and the status semantics, not just QG-4.
4. Pin the linter, so no verdict is ever again recorded against a gate that could not run.

*Independently derived from the git object store, the GitHub Actions API, and executed gates. Every
"CONFIRMED" above was re-run; no verdict rests on the audited report, on `STATE.md`, or on a commit
message.*
