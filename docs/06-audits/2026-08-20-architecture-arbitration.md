# AWIS — Final Architecture Arbitration

**Date:** 2026-08-20 · **Inputs:** (A) Repository Truth Audit · (B) Independent Audit of the
Repository Truth Audit · **Tree of record:** `main` @ `98350f6`
**Standing:** arbitration over two audits treated as evidence, not authority. No new repository
facts were gathered for this review; every claim traces to executed evidence, quoted text, or a
code trace already recorded in A or B.

> **Declared conflict of interest.** Audit B was produced by this same reviewer. Arbitrating it is a
> conflict. It is handled two ways: (1) every B finding is graded by *evidence class*, and B's
> second-hand and unexecuted claims are demoted accordingly; (2) this review overturns one of B's
> own load-bearing conclusions — its placement of M18 A-INIT (§5). A reviewer who cannot find an
> error in their own prior work has not reviewed it.

**Evidence classes used throughout.** `EXEC` — a command was run and its output recorded.
`TRACE` — derived by reading source, not executed. `QUOTE` — verbatim canonical text, transcribed
by an agent rather than re-read by the arbiter. `INFER` — reasoning over the above.
`INTERP` — governance interpretation, where reasonable people can differ.

---

## 1. Truth Table

Findings that materially affect M15/M16/M17/M18. Nothing else is listed.

| # | Finding | Status | Class | Note |
|---|---|---|---|---|
| T-1 | A non-initial step whose `fallback:` fires never terminates on `main` | **Proven** | EXEC ×2 | Probe on two worktrees (200 ticks, no terminal status); independently, reverting P2 on `m15` breaks `TestOIPSystemE2E` with the same `claim lost` signature |
| T-2 | M15's P2 change is the fix for T-1 | **Proven** | EXEC | Same probes pass post-P2 |
| T-3 | P2's risk is "expressions resolve to zero-values instead of failing" (A's §F-9) | **Refuted** | TRACE | FR-WD-05 is total; resolved value *and* warning string are identical pre/post |
| T-4 | P2 changes `WorkflowCompleted.outputs` for a fallback final step | **Proven** | EXEC | `{"fb":{…}}` → `{"a":{},"fb":{…}}` |
| T-5 | P2 flips `steps.X.status` from null to `'completed'` for fallback-origin steps | **Mostly Proven** | TRACE | Direct read of `buildEnv`; never executed as a probe |
| T-6 | P2 contradicts EDR-007 §2/§3 with no G2 amendment recorded | **Mostly Proven** | QUOTE | Verbatim EDR text transcribed by agent; not re-read by the arbiter. **Verify before acting.** |
| T-7 | P2 was executed under no Execution Card | **Mostly Proven** | QUOTE | `cards/M15-C3r.md` does not mention it; narrated post-hoc in TRACEABILITY/HANDOFF/G3_BRIEF |
| T-8 | M16-V1's `[x] V-COMMON all ✅` lint tick is false | **Proven** | EXEC ×2 | Both audits independently found 3 issues at `anthropic.go:259/339/342` |
| T-9 | M15-V1's lint tick is likewise false | **Refuted** | EXEC | m15 tip lints 0 issues, both modules. Unexecuted at the time, but substantively correct |
| T-10 | No milestone branch has ever been executed by CI | **Proven** | EXEC | Actions API; zero runs with `head_branch` ∈ {m15, m16, m17\*} |
| T-11 | "Exactly 3 CI runs, all on main" | **Refuted as stated** | EXEC | 5 runs; 2 on the audit's own branch. A's §F-3 already superseded A's headline |
| T-12 | `main` @ `98350f6` is green on Linux + macOS with lint genuinely executing | **Proven** | EXEC | Run `32339388733`, step 8 `success` on both legs |
| T-13 | M15+M16+M17 merge into `main` with zero conflicts | **Proven** | EXEC | 126 files, +10,272/−38 |
| T-14 | That merged tree reaches `verify: ALL GATES PASSED` after ~20 lines of fixes | **Proven** | EXEC | Full gate run, exit 0 |
| T-15 | Each *intermediate* merge point (post-M16, post-M17) is also green | **Disputed** | — | **Never verified.** M15-alone was separately proven green; the other two intermediate states were not. B over-generalised here |
| T-16 | M16's delta rebaselines cleanly onto `main` | **Proven** | EXEC ×2 | `git apply --check` exit 0 in both audits |
| T-17 | Cherry-picking M16's commits conflicts on `STATE.md` | **Proven** | EXEC | First commit `3518380` produces real conflict markers |
| T-18 | The S3 lint fix is required on *every* M16 path | **Proven** | EXEC | Confirmed on the direct-merge tree after an isolated-cache re-run resolved a contradictory result |
| T-19 | S5 (`apps/oip` require+replace) is required and belongs with M15 | **Proven** | EXEC | `GOWORK=off go build ./...` fails without it |
| T-20 | M17 depends on M15 | **Refuted** | QUOTE+EXEC | M17 README: "Depends on: M14, M12; soft on M16". Zero code references to `apps/oip` |
| T-21 | M17-C3 is "genuinely unfinished" | **Refuted** | EXEC | Goldens generate via the repo's own `-update` flag; all card acceptance criteria then pass |
| T-22 | `os.Exit` in `cmd/awis/errors.go:37` masks ~130 tests | **Proven** | EXEC | Verbose run aborts after 2 failures |
| T-23 | `docs-lint` goes red for M15/M16/M17 and guards the merge | **Proven** | EXEC | Run against every merge candidate |
| T-24 | Two `git mv`s into `cards/` + a compiled `DEPENDENCY_MAP.md` gives `docs-lint: OK` | **Proven** | EXEC | Exit 0 |
| T-25 | EEOS requires "rebaselining" onto `main` | **Refuted** | QUOTE | Term absent from all canon; `STATE.md` documents stacking as practice |
| T-26 | EEOS requires withdrawing a verdict whose gate did not execute | **Refuted** | QUOTE | NOT FOUND across EEOS/AWIS_EEOS/AEO/IKB/IMP/V-COMMON |
| T-27 | Four *divergent* `STATE.md` ledgers | **Refuted** | EXEC | Append-only chain; each is the parent's plus its own blocks |
| T-28 | `main`'s `STATE.md` is materially false about M10–M14 | **Proven** | EXEC | `f6aa755` is an ancestor of `main`; ledger says `E-MERGE (blocked)` |
| T-29 | Remapping "Fable" to "Opus 5" is policy-compliant | **Refuted** | QUOTE | `CLAUDE.md` forbids Opus and keeps Fable dormant absent human request |
| T-30 | Merging M16 first is optimal | **Disputed** | INTERP | Strategy, not fact. See §5 |
| T-31 | `steps.X.status == 'fallback'` transitions in both shipped workflows are dead | **Mostly Proven** | TRACE | Grammar defines no `fallback` status and `buildEnv` never emits one; never executed as a probe |
| T-32 | A "namespace extraction" proposal exists | **Irrelevant** | EXEC | Zero matches on any ref; absent from both audits. Nothing to arbitrate |
| T-33 | M18 A-INIT requires M15+M16+M17 to be **merged** | **Refuted** | QUOTE | EEOS's phase table fuses the next A-INIT into the current D-CLOSE. **This overturns B's own step 11.** See §5 |

**Disputed count: 2.** Everything else in contention between the audits is now settled by execution
or by verbatim canon. That is the useful outcome: the two audits together have converged the
factual surface almost completely, and what remains is strategy (T-30) and one unverified
intermediate state (T-15).

---

## 2. Runtime Risk Assessment

**Framing correction first.** Both audits speak as if `main` were in production. It is not: AWIS is
pre-`v1.0.0`, G4 has not been called, and the only consumer is OIP dogfood — which is itself an M18
activity. So "production impact" here means *impact on the first real user, which is AWIS itself*.
That moderates severity language without changing the ranking.

| Rank | Risk | Impact if unaddressed | Class | Severity |
|---|---|---|---|---|
| **1** | **`main`'s fallback hang** (T-1) | Any workflow whose `fallback:` fires on a non-initial step never completes. Both `fallback:`-using workflows in the repo have that shape, including `examples/workflows/with-intelligence.yaml`, already on `main`. It breaks OIP `capture-decision`'s designed zero-AI manual-entry path — the exact scenario M18's dogfood week exists to exercise, and the exact claim Constitution Art. 32 rests on | EXEC | **Critical** |
| **2** | **M17 `awis init`** | `//go:embed scaffold` silently drops `.gitignore`; `runInit` then calls `fail()` → `os.Exit`. `awis init` aborts hard for *every* new user, and QG-1 (init→trace ≤5 min on a clean machine) is an M18 exit criterion. Currently zero runtime exposure — it is not on `main` | EXEC | **High if merged unfixed / None today** |
| **3** | **M15 P2 itself, as residual risk** | Not the hazard A described. Residual is two observable deltas: `steps.X.status` reporting `completed` for a step that fell back (T-5), and a `{"X":{}}` key entering `WorkflowCompleted.outputs` (T-4). Both are silent-wrong-answer classes, but bounded to fallback topologies | EXEC + TRACE | **Medium** |
| **4** | **OIP module isolation (S5)** | No runtime misbehaviour. `apps/oip` resolves the SDK only through `go.work`; it fails the moment it is consumed as the standalone module IMP §15 requires. A build-integrity and packaging defect, invisible to every gate except `GOWORK=off` | EXEC | **Low runtime / High integrity** |
| **5** | **M16 Anthropic adapter** | 3 lint findings: one `defer resp.Body.Close()` whose error is discarded (a genuine, minor resource-error swallow) and two QF1012 style items. **Neither audit executed the adapter against a live API or a fake transport.** Its *lint* status is proven; its *behaviour* is untested by either audit | EXEC (lint only) | **Lowest proven / evidence gap** |

**The honest gap in this ranking.** #5's low position reflects an absence of findings, not a presence
of evidence. Both audits linted M16 and neither exercised it. If something is wrong with the adapter,
neither audit would have seen it. That is the one place where "no risk found" should not be read as
"low risk".

**The dangerous inversion, stated plainly.** Audit A ranked P2 as the repository's single HIGH risk
and scheduled M15 last. Executed evidence shows P2 is the remediation for the #1 runtime risk. Acting
on A's ordering would have held the fix behind a human gate while `main` continued to carry the
defect.

---

## 3. Governance Risk Assessment

**Consolidation first.** The brief lists five items, but three of them — the EDR-007 contradiction,
the status semantics, and the `StepFallbackActivated` projection rules — are **one governance
object**: a single frozen projection rule was changed, and the status and payload effects are that
change's consequences. Treating them as three invites three separate decisions where one is correct.

| Rank | Risk | Why it ranks here | Class | Severity |
|---|---|---|---|---|
| **1** | **The EDR-007 object** — projection rule changed, status semantics altered, payload shape altered, no G2 amendment | EDR-007 §2 tabulates `StepFallbackActivated → variables: —` and §3 says "no other event type mutates variables", amendable only by explicit G2. The change is real, correct, and undisclosed *as an amendment* — M15's own disclosure docs cite EDR-007's forward≡rebuild clause while leaving the variables clause unaddressed. Uncorrected, this is the precedent that makes every other "frozen" document advisory | QUOTE + EXEC | **Critical** |
| **2** | **P2 executed under no card** | EEOS rules 3 and 4 require a revision card and one commit per card. The engine change entered through a card (`M15-C3r`) whose text does not describe it. This is the *mechanism* by which #1 happened, and it is the more generalisable defect: the card system did not catch an unchartered frozen-surface edit | QUOTE | **High** |
| **3** | **Verification history integrity** | M16-V1 records a PASS whose lint row is provably false (T-8). AEO §12's independence guarantee produced a green that never existed. Aggravating: the cause (R3, a local toolchain mismatch) is a ~90-second fix that was never made, so the same failure mode was available to every milestone | EXEC | **High** |
| **4** | **`STATE.md` accuracy** | `main`'s ledger asserts M10–M14 are `E-MERGE (blocked on founder)` while `f6aa755` sits in `main`'s history, and `DONE-MILESTONES` stops at M09. The aggravator is recurrence: the ledger's own `LAST:` line records a 2026-07-10 reconciliation of *this same class of error*. A control that failed twice is not a lapse, it is an ineffective control | EXEC + QUOTE | **Medium-High** |
| **5** | **Verdict-withdrawal governance gap** | There is no rule for what happens when a verdict's evidence is later disproven (T-26). A's instinct to void was right in spirit and unfounded in text. The gap itself is the risk — the next false tick has no defined remedy either | QUOTE (NOT FOUND) | **Medium** |

**What is *not* a governance risk, despite appearances.** M15's two "illegal extra" module files and
the 12 missing contract files are `docs-lint` failures with a proven, mechanical, contract-exact
remedy (T-24). They block the merge; they do not indicate a governance failure. Treating them as
constitutional (A's §8 proposes weighing an EEOS §13 amendment) inflates bookkeeping into precedent.

---

## 4. Founder Decision Packet

Five decisions genuinely require the founder. Everything else is delegable, and is listed as such.

### D-1 — Amend EDR-007, and rule on fallback status semantics *(one decision, not three)*

- **The question.** Does `StepFallbackActivated` write to `variables`, and does a fallback-origin
  step project as `status == 'completed'` — or does the enum gain a sixth value and the sentinel move
  to a different carrier?
- **Why engineering cannot decide.** EDR-007 states it is "superseded only by an explicit G2
  amendment." G2 is a human gate. Engineering amending the document that constrains engineering is
  precisely the failure the frozen-spec regime exists to prevent.
- **Consequence of delay.** The T-1 hang fix cannot land cleanly. Engineering can either ship it
  undisclosed (repeating the original defect) or hold it (leaving `main` broken). Both are bad; the
  decision is what removes the dilemma.
- **Consequence of choosing wrong.** Choosing `completed` is cheap now and creates a permanent
  semantic lie: workflow authors cannot distinguish "succeeded" from "fell back" in a condition.
  Choosing a sixth status value is correct and costs a grammar change to a frozen document plus a
  re-verification of every consumer — and after `v1.0.0` the P8 append-only rule makes it far more
  expensive. **This is the decision with the shortest window.**

### D-2 — QG-4 / G3 platform-boundary verdict

- **The question.** Did OIP require platform surgery, or did it land through the disclosed seams?
- **Why engineering cannot decide.** QG-4 is a falsifiable claim about the architecture's core
  premise. The party that wrote the platform changes cannot certify that they were minimal.
- **Consequence of delay.** M15 cannot merge; OIP dogfood cannot start; M18's longest activity does
  not begin. Pure latency on the critical path.
- **Consequence of choosing wrong.** A false pass ratifies platform edits made under app pressure —
  the exact drift the gate exists to catch, and it compounds silently into V2.

### D-3 — TDS-06 Record format sign-off

- **The question.** Is `apps/oip/docs/RECORD_FORMAT.md` (currently DRAFT) the canonical OIP Record
  format?
- **Why engineering cannot decide.** It is a durable data-format commitment, and M18's post-tag rule
  makes migrations append-only **forever**.
- **Consequence of delay.** Blocks M15's G3 alongside D-2; same latency.
- **Consequence of choosing wrong.** Effectively unrecoverable after `v1.0.0`. Of all five, this is
  the one where "decide later, decide carefully" is legitimate — but it must be decided *before*
  the tag, not before the merge.

### D-4 — The three squash merges

- **Why engineering cannot decide.** AEO §13.4: "The human performs every merge. No agent, including
  CE, merges to main." Rule-bound and non-delegable regardless of merit.
- **Consequence of delay.** Nothing reaches trunk. Note the sharp limit on this: per §5, merges do
  **not** block M18 A-INIT.
- **Consequence of choosing wrong.** Bounded — a merge is revertible, unlike D-1 and D-3.

### D-5 — Authorize the P2 hotfix path, or don't

- **The question.** If D-1/D-2 will take more than a few days, may engineering land the 15-line
  engine fix on `main` on its own, ahead of and separate from M15?
- **Why engineering cannot decide.** It trades EEOS process integrity for shipping speed on a
  frozen surface. That trade is the founder's by definition — it is the same authority D-1 exercises,
  applied to sequencing.
- **Consequence of delay.** `main` keeps a defect that breaks the flagship application's designed
  path, for however long the gates take.
- **Consequence of choosing wrong.** Authorizing it splits P2 from its disclosure context and sets a
  precedent for hotfixing frozen surfaces. Refusing it keeps `main` broken. **Deciding it early is
  worth more than deciding it either way** — it is the option that costs nothing to hold open only
  if the gates move fast.

### Explicitly delegable — do not send these to the founder

| Item | Delegate to | Basis |
|---|---|---|
| Annotate-vs-withdraw the M15/M16 verdicts | Engineering | No rule governs it (T-26); pick the honest option and record it |
| `STATE.md` reconciliation | **Fable**, per EEOS ("ledger contradicts repo → Fable wake"), not the founder | The escalation ladder names an agent role, not the human |
| `BOUNDARY_EVIDENCE` / `G3_BRIEF` placement | Engineering | A contract-exact remedy is proven (T-24); no §13 amendment needed |
| Pinning `golangci-lint`, gofmt, S3, S5, embed/path fixes, golden generation | Sonnet, immediately | Mechanical, gated, reversible |
| Compiling the 12 contract files | Sonnet | IKB §3 defines the procedure as deterministic |
| Merge order mechanics | Engineering | §5 derives it from evidence; only D-5's exception is founder-level |
| Which agent performs the P2 review | Engineering, within `CLAUDE.md`'s frozen policy | And note: A's Fable→Opus 5 remap is refuted (T-29) |

---

## 5. Critical Path Verification

The brief forbids assuming stack order or rebaselining. Deriving instead — and the derivation
overturns both audits.

**Step 1 — what M18 A-INIT actually requires.** EEOS's phase machine defines D-CLOSE as: *"semantic
review of full diff · HANDOFF actuals · PR + evidence · (non-gated boundary: also run A-INIT of the
next milestone, cards READY)"*. A-INIT of the next milestone is **fused into D-CLOSE of the current
one**. E-MERGE is a separate, later, human phase.

**Step 2 — where the milestones actually stand.** Per the branch ledgers: M15 `E-MERGE (blocked on
founder)`, M16 `E-MERGE (blocked on founder)`, M17 `B-BUILD` with C3 `DISPATCHED`. M16's A-INIT was
opened at M15's D-CLOSE; M17's at M16's D-CLOSE — the fusion rule has already been applied twice in
this repository's own history.

**Step 3 — the conclusion both audits missed.**

> **M18 A-INIT opens at M17's D-CLOSE. It does not require a single merge, and therefore does not
> require the founder at all.**

Audit A placed M18 A-INIT at step 13, behind three squash merges. Audit B placed it at step 11,
behind the same three. **Both were wrong, and B's error is the one I am correcting as arbiter.** The
founder gates (D-2, D-3, D-4) block M15's *merge*, OIP dogfood, and G4 — they do not block M18's
entry.

**Step 4 — what M17 needs to reach D-CLOSE.** C3 complete → C-VERIFY (M17-V1 all-✅) → D-CLOSE. From
executed evidence, C3 needs: `//go:embed all:scaffold` (1 line), the test path depth (3 lines), and
`-update` golden generation (mechanical). One complication is proven: the version goldens fail on the
m17 lineage and are fixed only on `main` by `d5ac004`, so M17 must be verified on a tree that carries
`main`'s base. Bringing the base branch forward into a milestone branch is ordinary branch hygiene,
not "rebaselining" — no canon forbids it and T-13 proves it is conflict-free.

**Step 5 — is stack order justified, or merely convenient?** Justified for *merging*, not assumed:

- M16's branch cannot be merged without M15 (EXEC: `apps/oip/handlers.go` present in the m16→main
  merge, 57 files). So M16-first requires extraction.
- Extraction by cherry-pick is refuted — it conflicts on `STATE.md` at the first commit (T-17).
- Extraction by rebaseline works (T-16) but costs a hand-edited ledger, four contract docs, the same
  S3 fix, and a fresh V2 — and A's own instruction to drop `STATE.md` from the patch would land M16
  with no ledger entry, violating EEOS rule 1.
- Stack order costs none of that (T-13, T-14).

So stack order for merges is *derived*, not assumed. But it is a **conclusion about merges only**,
and merges are not on the path to A-INIT.

### The true shortest path to M18 A-INIT

```
integration tree = main + m17-c3-wip merge (carries M15+M16+M17, 0 conflicts — PROVEN)
  → ~20 lines of mechanical fixes (PROVEN to reach `verify: ALL GATES PASSED`)
  → M17-V1 on CI
  → M17 D-CLOSE
  → M18 A-INIT opens
```

**No founder gate on this path.** Estimated ~1 day of machine work.

**One caveat, stated rather than buried.** V-COMMON's procedure is `build · test · lint · race` plus
`e1`; `docs-lint` is not among its rows. So M17-V1 can pass with `docs-lint` red, and the 12 contract
files strictly block the **merge**, not A-INIT. Relying on that is process-lawyering, and I would not
build a plan on it — but it is the honest reading, and it means the doc work can run in parallel with
verification rather than ahead of it.

**Second caveat, against my own convenience.** T-15: the intermediate merge states were never
verified. If EEOS-compliant sequential merging is used (three branches, three PRs), the M16- and
M17-intermediate trees must each be gated independently. Only the combined end-state and the
M15-alone state are proven green today.

---

## 6. Execution Plan — the next 10 actions

Only actions that survive adversarial scrutiny. Dropped from both prior plans: rebaselining M16
(T-25, unnecessary), the `BOUNDARY_EVIDENCE` fold (T-24, a cheaper contract-exact fix exists),
withdrawal of the M15 verdict (T-9, it is not false), and any step whose only justification was
process symmetry.

| # | Action | Owner | Effort | Blocks | Unblocks | Risk |
|---|---|---|---|---|---|---|
| 1 | Tag `preserve/m17-c3-wip` → `153772c`. Never force-push that branch | Sonnet | 5 min | — | Every M17 action | **None** — pure insurance on the sole copy of C3 |
| 2 | Pin `golangci-lint@v2.12.2` in the Makefile to match `ci.yml` | Sonnet | 15 min | — | Local verdicts become meaningful | **None** — this is the control that prevents the T-8 class recurring |
| 3 | Founder brief for D-1 + D-2 + D-3 + D-5, sent as one packet | Human-prep | 1 h | — | All four founder gates start their latency now | **Low** — the single highest-leverage act on the whole board |
| 4 | Cut `awis-integration` from `main`; merge `origin/m17-c3-wip`; apply the 6 mechanical fixes; land P2 as its own commit citing EDR-007 | Sonnet | 1 h | 5, 6, 7 | Green tree; T-1 remediated in a reviewable form | **Low** — end-state proven (T-14); P2 isolated for review |
| 5 | Add the missing engine test: mid-workflow fallback + convergent join, routed through the existing `assertProjectionEquivalence` harness | Opus/Fable-grade | 2 h | 7 | Closes the coverage hole that hid T-1 for the entire M06–M17 span | **Low** — harness exists; only the case is missing |
| 6 | Compile the 12 contract files; two `git mv`s into `cards/`; compile M15's `DEPENDENCY_MAP.md` | Sonnet | 3–4 h | Merges | `docs-lint` green (T-24 proven) | **Low** — mechanical per IKB §3; runs parallel to 5 |
| 7 | Push the integration branch → first real CI covering all three milestones | Verifier | 15 min wall | 8 | Converts every verdict from self-reported to machine-proven | **Low** — CI on non-main branches proven live |
| 8 | M17-V1 card on the verified tree → **M17 D-CLOSE → M18 A-INIT opens** | Verifier → Fable | 2 h | — | **M18 entry, with zero founder dependency** (§5) | **Low** |
| 9 | Annotate M15-V1 and M16-V1 in `TRACEABILITY.md`: M16 `SUPERSEDED — step 3 row false`; M15 `SUPERSEDED — step 3 unexecuted, re-verified clean` | Sonnet | 30 min | — | Honest evidence chain without inventing a rule (T-26) | **None** — records reality; note the two verdicts differ (T-9) |
| 10 | `STATE.md` reconciliation on `main` — escalate as a Fable-wake ledger/repo contradiction, do not silently repair | Fable | 1 h | — | Ledger stops lying; recurrence gets adjudicated rather than patched | **Medium** — the honest repair may surface further ledger/repo divergence |

**Sequencing note.** Actions 1, 2 and 3 start immediately and in parallel; 3 is human-latency and
must not wait on any of the others. Actions 4→7→8 are the A-INIT critical chain. Actions 5, 6, 9, 10
run alongside. **Nothing on this list waits for the founder except the merges themselves.**

**What was considered and cut.** A P2-only hotfix branch to `main` is not action-listed because it is
gated on D-5. If D-5 comes back "yes", it displaces nothing — action 4 already isolates P2 as its own
commit, so the hotfix is a cherry-pick of that single commit.

---

## 7. Audit Verdict

### Repository Truth Audit — **Mixed** · confidence **85%**

Its records work is genuinely strong and I would keep it: no CI on milestone branches, M16's false
lint tick with two independent proofs, S3, S5, the `docs-lint` guard, `main`'s untruthful ledger, the
`m17-c3-wip` preservation instruction, and the call to start the founder brief on day 0. Roughly a
dozen findings survived unaltered.

Its engineering plan does not survive. It rated the repository's #1 runtime *remediation* as its #1
*risk*, described that change's mechanism incorrectly, and built a fourteen-step programme on the
assumption that an independent delta implies an independently mergeable branch — which the branch
topology refutes and which generated nearly every unnecessary step it proposed. It also asserted an
EEOS obligation (verdict withdrawal) that does not exist in canon, and proposed a model remap that
the frozen policy forbids.

The 15% of residual uncertainty sits mostly in T-30: its M16-first ordering is a strategy call, and
if founder latency turns out to dominate everything else, its instinct to front-load founder-free
work was defensible even though its mechanism for doing so was not.

### Independent Audit of the Repository Truth Audit — **Mostly Correct** · confidence **80%**

Its central claims are execution-backed and independently corroborated: the hang (T-1, two
independent reproductions), the clean stack merge and green gate run (T-13, T-14), the M15/M16
verdict asymmetry (T-9), M17's independence from M15 (T-20), and the cheaper `docs-lint` remedy
(T-24). Its refutations of A rest on traces and executed commands rather than assertion.

Three deductions, applied against my own work:

1. **It got M18 A-INIT wrong** (T-33). Placing it behind three merges contradicts EEOS's own phase
   table, and it is the single most consequential sequencing fact on the board — it means the founder
   does not gate M18 entry at all. That is a material error in the deliverable's most-read section.
2. **It over-generalised T-14 to T-15.** "The stack goes green" was proven for the combined
   end-state; the intermediate merge points were not tested, and an EEOS-compliant three-PR sequence
   needs them.
3. **Two load-bearing claims are not execution-grade.** The EDR-007 contradiction (T-6) is a
   transcribed quote the arbiter has not re-read, and the dead `status == 'fallback'` transitions
   (T-31) are a code trace, never probed. Both are very likely right; neither should be acted on as
   though it were measured.

The 20% of residual uncertainty is concentrated in exactly those three places, plus the conflict of
interest declared at the top of this document.

**Net assessment of the pair.** They are complementary rather than competing: A audited the records
and mis-modelled the code; B audited the code and under-read the process. Neither alone would have
produced a correct plan. Together they leave only two genuinely disputed items (T-15, T-30) and one
newly corrected one (T-33).

---

## WHAT I WOULD DO IF I OWNED AWIS

**Today, before anything else, I would send one email.** Not a status update — a four-question
decision packet: amend EDR-007 and pick the fallback status semantics; rule QG-4; sign or defer
TDS-06; and say yes or no to a standalone P2 hotfix. Every one of these is pure latency, none of them
gets cheaper by waiting, and D-1 gets dramatically more expensive after `v1.0.0` because of the
append-only rule. Everything else on the board is mine to run.

**Then I would stop treating the founder gates as the critical path, because they are not.** The
single most valuable thing this arbitration produced is that M18 A-INIT opens at M17's D-CLOSE, with
no merge and no human. That is about a day of work — the fixes are proven, the tree is proven green,
and the doc compilation is mechanical. I would have M18 open by tomorrow and let the merges land
whenever the founder gets to them.

**I would fix the hang first and loudly.** Not because it is dramatic, but because of what it says:
the flagship application's designed zero-AI path — the one Constitution Art. 32 rests on — has never
worked, and no gate caught it across twelve milestones. The fix is fifteen lines that already exist.
The test that would have caught it is one case in a harness that already exists. I would land both
this week and I would write down why the gate missed it, because that is the finding that outlives
this release.

**I would stop spending governance capital on bookkeeping.** Twelve missing contract files and two
misplaced documents are four hours of mechanical work with a proven remedy. One audit proposed
weighing a constitutional amendment over them. That instinct — treating documentation drift as
precedent-setting while a workflow-hang sat unexamined on trunk — is the actual process failure here,
and it is worth naming because it will recur.

**I would make one structural change immediately: pin the linter.** Every falsified verdict in this
repository traces to a gate that could not run locally and never ran in CI. It is a fifteen-minute
change. It is worth more than the entire re-verification debate it would have prevented.

**And I would change what "verified" means.** Both audits found that self-reported branch verdicts
were worthless, and both were right. But the fix is not re-running old verdicts on branch tips that
will never merge — it is verifying the tree that actually lands, in CI, once. That is one run
instead of three, it is stronger evidence, and it is now demonstrated to work.

**The thing I would resist.** Both audits produced elaborate multi-step programmes, and mine was one
of them. The measured remainder is about twenty lines of code, twelve documents, one test, and four
founder decisions. When the plan is longer than the work, the plan is the problem.

---

*Arbitration performed over two audits treated as evidence. Conflict of interest declared. Evidence
graded by class throughout; T-6 and T-31 are flagged for verification before they are acted upon.*
