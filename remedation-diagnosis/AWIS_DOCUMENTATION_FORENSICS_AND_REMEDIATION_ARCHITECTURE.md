# AWIS — Documentation Forensics and Remediation Architecture

**Produced:** 2026-09-11 · **Repository state:** `engine-hardening` @ `8a87f70`, working tree as found
**Method:** every conclusion re-derived from primary evidence in this repository. No conclusion, verdict,
or framing from any prior investigation was accepted as input. Where a prior report reached the same
conclusion, that is noted as convergence, not as citation. Where a prior report is wrong, it is corrected.
**Deliverable:** truth discovery and target architecture only. No files moved, archived, deleted, or edited.

---

## 0. Method and evidence discipline

### 0.1 What was actually examined

| Evidence class | What was done |
|---|---|
| Git object graph | 186 commits across 24 branches re-read in full; ancestry tested with `merge-base --is-ancestor`; tags, merges, and per-path history enumerated |
| Filesystem state | 380 markdown files inventoried; mtime clustering; md5 deduplication across the corpus |
| Tracking state | `git ls-files` differenced against the working tree, per path, for both documentation and source |
| Controls | `scripts/docs-lint.sh` read and **executed**; `.github/workflows/ci.yml` read in full |
| Build reality | A clean `git clone` of `HEAD` was made into a scratch directory and `make build` + `make test` run against it |
| Document content | All Tier-0 canon headers, all index files, all 5 `FINAL_VERDICT.md`, all cluster READMEs, the EEOS protocol, and the execution ledger read directly |

### 0.2 Evidence tiers used in this report

The brief's tier ordering is a statement about **document authority**. It is not usable as an ordering for
**truth**, because the question under investigation is precisely whether those documents deserve authority.
Using it as given would assume the answer. This report therefore separates the two:

- **Authority tiers (T0–T5)** — the brief's hierarchy, used when discussing what *governs*.
- **Evidence tiers (E1–E4)** — used when discussing what is *true*:
  - **E1** Executed reality: command output, a clean-clone build, a control actually run.
  - **E2** Repository facts: git ancestry, tracking state, file bytes, checksums.
  - **E3** Source code and tests.
  - **E4** Document assertions — *hypotheses, never evidence for themselves*.

Every load-bearing claim below carries an E-tier. No claim in this report rests on E4 alone.

### 0.3 One correction to the brief's own framing

The brief instructs that Tier 1 is "original foundation documents, specifically `00-foundation`
… `07-indices`". **This is not where the foundation lives** [E2]. Those seven directories contain
**one README each** — 5,255 bytes total. They are pointer stubs. The actual Tier-0 corpus
(`OIP_CONSTITUTION.md`, `AWIS_ARCHITECTURE_BLUEPRINT.md`, `AWIS_ARCHITECTURE_FINALIZATION.md`,
`AWIS_PRD.md` — 387 KB) lives at **repository root**, by explicit original design:

> `docs/01-vision/README.md`: "Canonical, immutable, **at repository root**"
> `docs/07-indices/canonical-reference-map.md`: Tier 0 and Tier 1 are root files; `docs/` is **Tier 2**, "a VIEW".

This is not a triviality. It is the first structural finding of this investigation, and §8.1 shows it is
load-bearing for the entire failure.

---

## 1. Executive verdict

**AWIS's documentation did not drift, fragment, or degrade in quality. It was orphaned.**

The documentation system was built to feed exactly one consumer: a human integration gate called
**E-MERGE** — a founder reviewing a pull request carrying the diff, the verification report, the
deviations, and the ledger diff. Milestone modules, HANDOFFs, TRACEABILITY files, validation checklists,
and execution cards had no other purpose. They were inputs to a decision.

**That gate stopped running on 2026-07-10, after M09** [E2]. It never ran again in a recognizable form.
Everything the brief describes as a documentation problem is a second-order consequence of that single
fact, and the consequences arrive in a strict, verifiable order:

1. **The gate stalls.** M10–M17 are all still marked `PHASE: E-MERGE (blocked on founder)` in the ledger.
2. **Documentation loses its consumer.** Artifacts whose only reader is a gate that never runs stop being
   produced. Module conformance degrades monotonically from M15 onward — 7/7 → 6/7 → **3/7** → 5/7 → 1/7 [E1].
3. **Integration degrades in lockstep.** Individually reviewed merges (M01–M09) → one unreviewable
   184-file bulk merge for M10–M14 six weeks late → M15/M16/M17 **never merged** → the API/GUI **never
   committed at all** [E2].
4. **The last mechanical control is switched off.** `docs-lint` was removed from CI on 2026-08-21 because
   it was correctly reporting the degradation, with a written promise to re-enable it that was never
   kept [E1].
5. **The ledger dies.** `STATE.md` — whose governing rule is *"no transition is real until written
   here"* — was last written 2026-08-21 21:58 [E2]. Thirty-plus commits of engine hardening follow it.
   It contains zero mentions of hardening, of the GUI, or of any work after that date.
6. **Re-derivation replaces the ledger.** With no maintained statement of state, every subsequent question
   is answered by re-reading the code and **writing a new document**. Because no mechanism can retire the
   old one, the authority population grows monotonically: 25 root-level audit documents, four unnumbered
   `docs/` clusters, five files named `FINAL_VERDICT.md`.

**GUI planning is not the root cause.** It is the 51st day of a failure that began on 2026-07-10, and it
is — measurably — the *best-governed* cluster in the repository (§7).

**The foundation was not sound-but-overwhelmed either.** It shipped three specific, latent structural
defects that determined the *shape* of the collapse (§8), the most consequential being that it placed
canonical authority in the root namespace, where no structural property distinguishes the constitution
from any file someone drops beside it.

**The single most damning fact in the repository** is not about documentation at all, and it proves the
diagnosis: the entire Beta deliverable — `internal/api`, `cmd/awis-server`, `web/`, `internal/buildinfo`,
**42 source files** — has **zero commits in the project's entire history, on any branch** [E2], and is
not ignored, merely never added. The documentation and the code failed *in the same way, on the same
curve, for the same reason*: the repository stopped being the system of record. A diagnosis confined to
"documentation governance" cannot explain why the code did it too. This one does.

---

## 2. Documentation evolution timeline

Dates are from git author dates (E2) and filesystem mtimes (E2). Phase boundaries are drawn where the
*governance mode* changes, not where topics change.

### Phase I — Canon formation · 2026-07-02 → 07-03 · **healthy**

387 KB of Tier-0 canon produced in ~48 hours: constitution, blueprint, finalization, PRD, plus the
pre-canon investigations that fed them. `IMPLEMENTATION_MASTER_PLAN.md` (19 milestones M00–M18),
its verification report (binding amendments F-1..F-5), and the IKB partitioning scheme follow.
`docs/` is created as a Tier-2 *view* over root canon: seven pointer stubs, an indices layer, and
`05-implementation/` with a module per milestone.

**Governance mode:** authority resolves upward to frozen root documents; conflicts go to a CONTRA protocol;
supersession is demonstrated once, correctly (§8.3).

### Phase II — Gated milestone execution · 2026-07-03 → 07-09 · **healthy, and it worked**

M00–M09 execute. EEOS v2.0 is frozen mid-stream at M07 (2026-07-08), formalizing the five-phase machine
(A-INIT / B-BUILD / C-VERIFY / D-CLOSE / E-MERGE) and introducing execution cards.

Evidence this phase genuinely functioned [E2]:
- Every milestone M01–M09 is an **individual ancestor of `main`** — nine separate reviewed merges.
- Milestone tags exist for M02–M09.
- Module conformance is **7/7 contract files for every milestone M00–M14** [E1].
- Gate G1 was exercised with real founder adjudications (ADJ-1..8) recorded in `docs/04-planning/README.md`.
- M08-V1 **failed** verification (5 test gaps), produced a revision card, and re-passed — the loop closed.

This is the part of the brief's hypothesis that survives: the foundation documents *did* produce the
engine, and the engine they produced is real (§8.5).

### Phase III — The directive, and the silent inflection · **2026-07-10** · **the hinge**

The ledger records a founder directive dated 2026-07-10: *"complete all milestones"*, with milestones to
**stack branches** and E-MERGE deferred. The cadence changes from one-milestone-one-merge to
build-everything-merge-later.

Three things begin on this exact date, and none of them were noticed [E1/E2]:

| Signal | Before 2026-07-10 | From 2026-07-10 |
|---|---|---|
| Merge to `main` | 9 individual reviewed merges | none, for six weeks |
| Module conformance | 7/7, fifteen milestones running | M15 drops to 6/7 with 2 off-contract extras; **M16 to 3/7**; M17 5/7; M18 never materialized |
| Milestone tags | M02…M09 | none until a single M14 tag |

**This is where control was lost.** Not in September. The rest of the timeline is consequence.

### Phase IV — Unmerged accumulation · 2026-07-10 → 08-20 · **degrading**

M10–M17 are built on stacked branches. All eight are recorded `E-MERGE (blocked on founder)`. On
2026-08-20 a single commit `f6aa755` titled **"M10-M14"** lands on `main`: **184 files, +18,763 / −94**
[E2]. Five milestones, six weeks of work, collapsed into one diff no human reviews at that size. This is
the E-MERGE gate's last appearance, and it appears in a form that cannot perform its function.

M15, M16, M17 are never merged and remain unmerged today (ahead of `main` by 10, 13, and 30 commits) [E2].
`main` is frozen at 2026-08-20 and has been ever since.

### Phase V — The control is disabled · **2026-08-21** · **the point of no return**

Two events, hours apart, on the same day:

1. `docs-lint` — the *only* mechanical documentation control in the project — is deliberately excluded
   from CI. The reasoning is written into `.github/workflows/ci.yml` and is, in isolation, good
   engineering judgment:

   > "docs-lint: currently FAILS because the M15/M16/M17 EEOS module artifacts are incomplete. Wiring it
   > now would make CI permanently red, which trains people to ignore it. **Wire it in the same change
   > that materializes those files.**"

   The files were never materialized. The promise was never kept. **Run today, 21 days later, the control
   still fails with three true positives** [E1] — M15, M16 and M17, exactly as predicted.

2. `STATE.md` is written for the last time, at 21:58 [E2].

From this moment the project has no ledger and no documentation control. Everything after is unpoliced.

### Phase VI — Engine hardening outside the system · 2026-08-26 → 09-05 · **productive, ungoverned**

Thirty-plus commits close defect classes B-0…B-31 and RC-1…RC-5. The work is **genuinely good** — the
defects are real, the fixes carry regression tests, and the clean-clone build and full test suite pass
today [E1].

But it executes entirely outside the governance system:
- No milestone. **`M18-hardening-release` — the milestone that exists precisely for this work — was never
  materialized** [E1]. It holds one README from 2026-07-02.
- No cards, no verifier dispatch, no STATE writes, no HANDOFF, no TRACEABILITY.
- Its documentation goes to a new, unnumbered folder: `docs/08-engine-hardening/` (5 files).
- It happens on a branch that diverges from `main` and is never merged.

**This is the moment the milestone architecture was abandoned** — six days before GUI planning began.
The brief asks whether GUI planning introduced a parallel structure outside M-series governance. It did,
but it was not first: **engine hardening did it first**, for the engine itself, in the slot reserved for it.

### Phase VII — Audit explosion · 2026-09-05 → 09-09 · **collapse made visible**

With no ledger, every question is answered by re-derivation from code, and each re-derivation produces a
document. The filesystem records the mechanism precisely [E2]:

- **54 of 78 untracked documents share one mtime: `2026-09-06 00:18`.** They were not authored in place
  over time; they arrived as a single bulk materialization.
- `docs/11` + `docs/12` — 15 documents, ~150 KB of architecture decisions — were produced between
  **12:40 and 13:00 on 2026-09-06**. Twenty minutes.
- 25 root-level audit documents land in the **Tier-0 namespace**, where nothing distinguishes them from
  the constitution by position.

The chain of terminal verdicts in this window is, notably, **temporally coherent**: 09-03 REJECTED →
09-05 PASS WITH RISKS → 09-05/06 CONDITIONAL PASS → 09-09 `TRUTH_CLOSURE.md` (CANONICAL). Each
supersession is declared. §6.3 explains why a coherent chain nonetheless produced an unusable corpus.

### Phase VIII — Current state · 2026-09-11

319 AWIS documents (349,146 words), plus a 55-document generic agent toolkit that mentions AWIS zero
times (§6.5). 78 documents and 42 source files outside version control. `main` three weeks stale.
One control, disabled and still failing. No ledger.

---

## 3. Root cause analysis

Ranked by causal depth: removing #1 prevents everything below it.

### RC-1 — The integration gate stopped running, and nothing was designed to notice · **ROOT CAUSE** · E1/E2

EEOS makes E-MERGE human-only and gives it no timeout, no escalation, and no alarm. `STATE.md` can
sit at `E-MERGE (blocked on founder)` indefinitely and that is a *valid* state. Eight milestones
occupied it simultaneously.

The system had **liveness controls for work and no liveness control for approval**. Every other failure
descends from this: documentation lost its consumer (RC-3), integration lost its ratchet (RC-2), and
truth lost its home (RC-4).

*Falsification attempted:* if this were merely a scheduling inconvenience, artifact quality would be
unaffected — merges would just be late. Instead conformance degrades **monotonically from the exact date
the gate stalls** (M15 →M16 →M17 →M18: 6/7, 3/7, 5/7, 1/7), which is what an incentive change looks like,
not what a queue looks like. Survives.

### RC-2 — Integration degraded through four modes, each hiding the next · E2

`individually reviewed merges → one 184-file bulk merge → never merged → never committed`

Each step is locally defensible and each destroys a different guarantee: reviewability, then integration,
then existence. The endpoint — 42 source files with zero commits ever — is not a documentation failure
and cannot be explained by any documentation-only theory.

### RC-3 — Documentation had exactly one consumer, and no independent reason to exist · E1/E4

Module artifacts (HANDOFF, TRACEABILITY, VALIDATION_CHECKLIST, DEPENDENCY_MAP, AI_EXECUTION_CONTEXT) were
inputs to a gate. When the gate stopped consuming them, producing them became unrewarded work — and
production stopped, in the order of least-immediately-useful first. M16, the most mechanical milestone,
lost the most (3/7).

A documentation artifact that is not read by a control, a gate, or a build is a *hope*, not a system.

### RC-4 — Re-derivation became the truth-recovery method, and it is append-only · E2

Re-deriving state from code is correct as an *act* and catastrophic as a *habit*, for one structural
reason: **it produces a new document instead of correcting the old one.** With no retirement mechanism
(RC-5), authority count grows monotonically. 25 root audit documents in five days is the arithmetic of
that habit, not evidence of confusion or carelessness — each individual document is competent.

### RC-5 — Supersession was proven once and never reused · E1/E2

The foundation implemented supersession *correctly* on day one (§8.3). Of the 24 documents later declared
superseded, **2 say so in their own text** [E1]. Twenty-two read as live to any reader who opens them.
Supersession was recorded **forward-only** — asserted by the successor, never inscribed on the ancestor.

### RC-6 — Canonical authority was addressed by filesystem position, in the least protected namespace · E2

Tier 0 = repository root, by original design. The root namespace is where every tool, agent, and human
writes by default. Twenty-five audit documents therefore landed *inside the constitutional tier*, and no
structural property — path, naming convention, frontmatter, permission — distinguishes
`RELEASE_BLOCKERS.md` from `OIP_CONSTITUTION.md`. §8.1.

### RC-7 — The one control was disabled rather than satisfied, with a promise as the fallback · E1

2026-08-21. The reasoning was sound; the mechanism — a comment in a YAML file — had no enforcement,
no owner, and no expiry. Twenty-one days later it still fails with the same three true positives.

### RC-8 — Controls could only see one directory · E1

Even at full strength, `docs-lint` checks only `docs/05-implementation/` module contracts, ledger
existence, and archive citations. It cannot see root-level proliferation, tracking status, duplicate
filenames, ID collisions, missing tombstones, or the empty `docs/06-reference/`. **Not one of the failures
that actually occurred was in its field of view.** Disabling it mattered less than the fact that it was
aimed at the wrong thing.

---

## 4. Evidence chain

Every claim in this report, with its verification method. Reproduce any row from a clone plus this table.

| # | Claim | Tier | How verified |
|---|---|---|---|
| E-01 | Tier-0 canon lives at repo root; `docs/00-04,06` are 1-README pointer stubs | E2 | `ls docs/*/`; read all 7 READMEs; `canonical-reference-map.md` tier table |
| E-02 | 42 source files of the Beta deliverable have **zero commits ever, any branch** | E2 | `git ls-files` = 0 and `git log --all -- <dir>` = 0 for `internal/api`(15), `cmd/awis-server`(8), `web`(18), `internal/buildinfo`(1) |
| E-03 | They are untracked, **not ignored** | E2 | `git check-ignore -v` returns nothing for all four |
| E-04 | `.gitignore` asserts "source is tracked" — false | E2 | Read `.gitignore` comment vs E-02 |
| E-05 | Clean clone of HEAD **builds and passes all tests**; API/server/web absent from it | **E1** | `git clone --no-local` → scratch; `make build` ✓, `make test` ✓; four dirs ABSENT |
| E-06 | 78 markdown files untracked = exactly clusters 09/10/11/12 + 19 root files | E2 | `comm -13` of `git ls-files` vs `find` |
| E-07 | **54 of 78 share mtime `2026-09-06 00:18`** — bulk drop, not authorship | E2 | `stat -c %y` on all 78, clustered |
| E-08 | `docs/11`+`docs/12` (15 docs, ~150 KB) produced 12:40–13:00, 2026-09-06 | E2 | mtime clustering |
| E-09 | `STATE.md` last written 2026-08-21 21:58; zero mentions of hardening/GUI/M18 | E2 | `stat`; `grep -ciE 'hardening\|B-[0-9]+\|RC-[0-9]\|GUI'` → 0 |
| E-10 | `docs-lint` excluded from CI 2026-08-21 with a written, unkept promise | E2 | `.github/workflows/ci.yml` comment block |
| E-11 | **The control still fails today, 3 true positives** | **E1** | `bash scripts/docs-lint.sh` → exit 1, M15/M16/M17 |
| E-12 | Module conformance: M00–M14 7/7; M15 6/7+2 extras; M16 3/7; M17 5/7; M18 1/7 | E1 | Per-file existence test against the 7-file contract |
| E-13 | M01–M09 individually merged to `main`; M10–M14 in one 184-file commit; M15–M17 never | E2 | `git merge-base --is-ancestor` per commit; `git show --shortstat f6aa755` |
| E-14 | `main` frozen 2026-08-20; `engine-hardening` +62 / −6 divergent | E2 | `git log main..engine-hardening`, reverse |
| E-15 | 186 commits, **1 merge commit**, 9 tags (M02–M09, M14) | E2 | `git log --all --merges`, `git tag` |
| E-16 | `M18-hardening-release` never materialized; hardening ran in `docs/08-` instead | E1 | `ls`; `docs/08-engine-hardening/README.md` |
| E-17 | `docs/06-reference/` is empty; its README's path map contradicts IMP §12 and reality | E2 | `ls`; TDS files found at `docs/*.md`; IMP §12 lines 252–258 |
| E-18 | That README states a failure oracle that has been false since 2026-07-03 | E4→E2 | "A missing TDS here means its milestone has not executed" vs `git log --diff-filter=A` |
| E-19 | 5 × `FINAL_VERDICT.md`; root vs `docs/10` hold **opposite verdicts** | E2 | `find -name FINAL_VERDICT.md`; read both ("PASS WITH RISKS" vs "NOT release-ready") |
| E-20 | 4 byte-identical duplicate pairs between root and `docs/10` | E2 | `md5sum` collision groups |
| E-21 | 25 distinct ID prefixes; `ADR-001` collides across Tier 0 and `docs/12` | E2 | Corpus-wide regex extraction; collision inspection |
| E-22 | **2 of 24** superseded docs are tombstoned in their own text | E1 | Per-file grep for self-declared supersession |
| E-23 | Supersession *was* implemented correctly once, at the foundation | E2 | `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` header: struck status + ⚠ banner + registrar + docs-lint rule |
| E-24 | The two 2026-09-06 investigation lineages are citation-isolated | E2 | 1 citation of 15 docs one way; **0** of 44 the other |
| E-25 | GUI planning's "start now" does **not** contradict the freeze report — scope elision | E4→E2 | `GUI_START_LINE.md` §"two start lines"; `FINAL_VERDICT.md` lines 18–19 name both blockers |
| E-26 | `.agent/` toolkit (55 docs, dated 2026-05-25) mentions AWIS/EEOS **zero** times; prescribes conflicting branch naming | E2 | `grep -ci 'eeos\|awis' .agent/AGENTS.md` → 0; `project-conventions.md` |
| E-27 | `archive/` holds only a README — by design, not by neglect | E2 | `ls archive/`; `git log --all -- 'archive/*'`; registrar text |

---

## 5. Failure mechanism reconstruction

The complete chain, each link verified, each link necessary:

```
2026-07-02  Canon is written and frozen.  Authority = filesystem position, at repo root.   [RC-6 latent]
            Supersession demonstrated correctly, once.                                     [RC-5 latent]
            No document anywhere specifies documentation governance.                       [RC-8 latent]
                │
2026-07-03  docs/ built as a Tier-2 view. 06-reference README invents a TDS path map that
   ─07-09   contradicts IMP §12; M01 correctly follows IMP; the index is wrong from day 2
            and its failure oracle reads false forever. Nobody reads it. [E-17, E-18]
                │
            M00–M09 execute under a working gate. Nine reviewed merges. 7/7 conformance.
            THE SYSTEM WORKS. This is the control group.
                │
2026-07-10  ── FOUNDER DIRECTIVE: "complete all milestones", stack branches, defer merge ──
                │
                ├─► E-MERGE stalls. No timeout exists. 8 milestones queue in a VALID state.  [RC-1]
                ├─► Documentation loses its only consumer.                                   [RC-3]
                └─► M15 conformance breaks (6/7 + 2 off-contract files). Unnoticed.
                │
2026-07-11  M16: 3/7. The worst module in the project. No TRACEABILITY, no HANDOFF,
            no AI_EXECUTION_CONTEXT — yet D-CLOSE records "V1 PASS; APPROVED". [E-12]
                │
2026-08-20  M10–M14 bulk-merged: 184 files, +18,763 lines, ONE commit, six weeks late.
            The gate runs in a form that cannot perform its function.                        [RC-2]
                │
2026-08-21  docs-lint removed from CI — because it is correctly reporting the above.
            Remediation deferred to a comment in a YAML file. Never done.                    [RC-7]
            STATE.md written for the last time, 21:58.
                │
            ═══ NO LEDGER · NO CONTROL · NO MERGE GATE · NO TAG ═══
                │
2026-08-26  Engine hardening begins. Real work, real fixes, real tests — and entirely
   ─09-05   outside the system. M18, the slot built for it, is never materialized;
            docs/08- is invented instead. 30+ commits, zero ledger writes.                   [E-16]
            *** THE MILESTONE ARCHITECTURE IS ABANDONED HERE — 6 DAYS BEFORE GUI PLANNING ***
                │
2026-08-30  GUI planning begins. It inherits a dead governance system, invents a parallel
   ─09-01   milestone namespace (G0–G10), documents itself unusually well, and correctly
            diagnoses the problem it landed in ("reconcile STATE.md and merge … otherwise
            'the engine' is ambiguous"). Nobody acts on it.                                  [§7]
                │
2026-09-05  With no ledger, every question is re-derived from code. Each re-derivation
   ─09-09   emits a NEW document; nothing can retire the OLD one.                            [RC-4]
            54 documents materialize in one minute. 15 more in 20 minutes.
            25 land in the ROOT — i.e. inside the Tier-0 namespace.                          [RC-6]
            22 of 24 superseded documents carry no tombstone.                                [RC-5]
                │
2026-09-11  387 files. 5 × FINAL_VERDICT.md. 25 ID schemes. 42 uncommitted source files.
            No ledger. One disabled control, still failing.
```

**The authority collapse sequence**, stated plainly: *canonical → gated → stalled → unmerged →
uncommitted → unlocatable.* Documentation tracks this exactly one step behind, every time.

---

## 6. Authority model diagnosis

### 6.1 The model as designed

A clean five-tier upward-resolving hierarchy (`canonical-reference-map.md`): Constitution → Blueprint →
Finalization → PRD → IMP+Verification → KB, with conflicts escalated via CONTRA and never silently
resolved. **As a model, it is good.** Gate G1's adjudication log (ADJ-1..8) proves it was genuinely
exercised: real contradictions found, escalated, ruled on by the founder, recorded, and folded back.

### 6.2 Defect A — the authority model has no canonical home

The document defining the tier hierarchy — `canonical-reference-map.md` — sits at **Tier 2**, inside the
very layer it subordinates. Worse, `docs/07-indices/README.md` declares its own tier decorative:

> "These files contain NO new information… If an index row and its target ever disagree, **the target wins
> and the index row is a bug.**"

The tier map has no target. It *is* the only statement of the authority model, and it is filed in the one
tier the project declared non-authoritative. **The authority model is not itself authoritative** — a
circularity present from 2026-07-03 that no control could detect.

### 6.3 Defect B — supersession is forward-only

Every supersession in this corpus is asserted by the *successor*. Nothing is inscribed on the *ancestor*.
The consequence is not confusion for someone reading the whole corpus in order — the chain is temporally
coherent. The consequence is for the only reader who matters: **someone who opens one file.**

A reader who opens `FINAL_VERDICT.md` sees a document that confidently supersedes an earlier verdict. They
cannot learn from it that two later documents superseded it, that a third declared it non-canonical, or
that an identically-named file in `docs/10-release-candidate-audit/` holds the **opposite verdict** [E-19].
All five `FINAL_VERDICT.md` files present as live. Twenty-two of twenty-four superseded documents present
as live [E-22].

This is the precise mechanism by which a corpus of individually-competent documents becomes collectively
unusable — and it needs no contradiction, no drift, and no incompetence to occur.

### 6.4 Defect C — authority by position, in the default write location

§8.1. The root namespace is simultaneously the constitutional tier and the path of least resistance for
every writer. Twenty-five audit documents did not "escape" into Tier 0; they had nowhere else to land.

### 6.5 Defect D — a third instruction layer with no relationship to AWIS

`.agent/` — 55 documents, internally dated **2026-05-25**, i.e. predating AWIS's foundation — was staged
into the repository on 2026-09-06, in the same window as the audit explosion. It mentions AWIS or EEOS
**zero times** [E-26]. Its `project-conventions.md` prescribes `feature/[task-slug]` branch naming, which
contradicts the milestone-branch convention EEOS depends on (`m10-yaml-dsl`). `CLAUDE.md` names it "the
full baseline" while simultaneously overriding it for any EEOS session.

The project now has three mutually-unaware instruction layers — EEOS, `.agent/`, and `CLAUDE.md`'s
override — resolved only by a precedence note in a file agents may or may not read first. This is an
active, current conflict, not a historical one.

---

## 7. GUI planning diagnosis

The brief asks whether GUI planning is guilty. **I attempted to convict it and failed.** The evidence runs
the other way, and this section states the case against my own initial hypothesis.

### 7.1 What GUI planning actually did wrong

Two real charges, both genuine:

1. **It created a parallel milestone namespace.** G0–G10 with ~80 cards, structurally mimicking M00–M18
   but outside EEOS, outside `STATE.md`, outside `docs-lint`'s field of view. A second execution system
   with no ledger.
2. **Its output was executed entirely outside version control.** The GUI MVP was built, browser-verified,
   and declared shipped — and `web/`, `internal/api/`, `cmd/awis-server/` have never been committed [E-02].
   Its own README says so: *"No separate report document for this pass — the code and its tests are the
   record."* The record does not exist in the repository.

### 7.2 Why it cannot be the root cause

| Test | Result |
|---|---|
| Did it precede the collapse? | **No.** It begins 2026-08-30. The gate stalled 2026-07-10 (**51 days earlier**); conformance broke at M15; `docs-lint` was disabled and `STATE.md` died on 08-21 (**9 days earlier**); the milestone architecture was abandoned by engine hardening on 08-26 (**4 days earlier**). |
| Did it break a working control? | **No.** Every control was already dead or disabled when it started. |
| Did it introduce the parallel-folder pattern? | **No.** `docs/08-engine-hardening/` did, six days earlier, for the engine's own work, in the slot `M18` was reserved for. |
| Did it cause the root-level explosion? | **No.** All 38 of its documents are in `docs/09-gui-planning/`. It put **zero** files at root. The 25 root documents came from the release-audit and reconstruction lineages. |
| Is it internally ungoverned? | **No — it is the best-governed cluster in the repository.** |

### 7.3 The finding that reverses the charge

`docs/09-gui-planning/README.md` is, by a wide margin, the strongest governance artifact produced after
Phase II. It:

- labels **six explicitly numbered passes**, each dated, each stating its relationship to the prior
  ("second pass — challenges and cuts the execution program above; **no new investigation**");
- pins its evidence base to a **specific commit** (`f004f4f`) and re-verification date;
- contains `REPOSITORY_TRUTH_AUDIT.md`, a **deliberate self-falsification attempt** against every other
  document in the folder, whose findings are recorded including one that damages the plan;
- publishes a **reading order** with time budgets;
- records its own corrections in the open.

And decisively — **it diagnosed the real root cause, correctly, in its own terminal verdict**, eleven days
before this investigation:

> "reconcile `STATE.md` and merge M15→M16→M17→`engine-hardening` into `main` (or explicitly declare
> `engine-hardening` the new base) — ~1 day, zero engineering, but otherwise **'the engine' is ambiguous**"

with addendum finding **N7**: *"this whole plan is evaluated against a branch `main` is 60 commits
behind."* My independent measurement today: **62 commits** [E-14].

**A cluster that identifies the governance failure it is operating inside, quantifies it to within two
commits, and is ignored, is not the cause of that failure. It is the highest-quality symptom in the
corpus.**

### 7.4 The apparent contradiction, tested and dismissed

`docs/08-engine-hardening/` (08-29) says *"The GUI is not ready to start"*; `docs/09-gui-planning/`
(09-01) says GUI work can start today. I treated this as the corpus's sharpest contradiction and tested it.

**It is not a contradiction** [E-25]. The freeze report names two blockers: no global event cursor, and no
definition→YAML serializer. GUI planning **names both** and scopes them: the *live* view needs migration
0007 / `state_changes`; the *editor save path* needs G6's serializer; the frontend shell and read-only
views need neither. The substance agrees completely. Only the headline sentences collide.

This generalizes into one of this report's most important findings: **most apparent contradictions in this
corpus are scope-elision artifacts** — a correct conclusion stated as an unqualified headline, colliding
with another correct conclusion at a different scope. The corpus is far more *internally consistent* than
its surface suggests, which is precisely why readers lose trust in it: the headlines fight while the
analyses agree, and only a full read reveals it.

**Verdict on GUI planning: contributor, not cause. It added volume and a second ungoverned namespace to a
system whose governance had already been dead for nine days — and it documented itself better than
anything else built after M09.**

---

## 8. Foundation diagnosis

The brief forbids assuming the foundation is correct. Tested on both axes — was it well-designed, and was
it faithfully executed — the answer is split, and the split is the interesting part.

### 8.1 Foundation flaw #1 — canonical authority placed in the least protected namespace · **most consequential**

Tier 0 and Tier 1 are repository-root files, by design [E-01]. Root is where every human, agent, and tool
writes by default. There is no structural property — path, prefix, frontmatter, ownership, permission —
separating constitutional text from an ad-hoc report.

This flaw is **latent, not active**. For two months it cost nothing: between 2026-07-08 and 2026-09-05,
**zero** new root documents were created [E2] — the entire engine was built without adding one file to
root, because milestone execution had a proper home in `docs/05-implementation/`. The moment a phase of
work had *no* designated home (the audit phase), 25 documents landed in the constitutional tier in five
days.

**Root-level proliferation is therefore the foundation's design flaw being exercised, not a new behavior.**

### 8.2 Foundation flaw #2 — no document in the entire corpus specifies documentation governance

A corpus-wide search for any statement of documentation placement rules, naming rules, lifecycle,
retention, supersession protocol, or where a new document may be created returns **zero matches** across
all 319 AWIS documents [E2].

The foundation specified, in rigorous detail: event schemas, expression grammars, migration sequencing,
plugin lifecycle FSMs, model allocation per card, context budgets per session, and a CONTRA protocol for
contradictions in *frozen text*. It specified nothing about its own corpus.

The one partial exception proves the rule: `docs/05-implementation/` has a 7-file module contract, and it
is **the only part of the corpus that held 7/7 for fifteen consecutive milestones** [E-12]. Where a
contract existed, conformance was perfect. Where none existed — root, and every cluster after `docs/08-` —
there was nothing to conform to.

**Governance was applied to the engine's artifacts and never to the corpus that produced them.**

### 8.3 Foundation flaw #3 — the correct supersession pattern was demonstrated once and never reused

This is where I must correct both my own working hypothesis and a prior investigation in this directory,
which records *"`archive/` used 0 times in 387 docs"* as evidence of a missing mechanism.

**That reading is wrong, and the truth is worse.** The foundation did not omit supersession. It
implemented it, correctly, on day one, using a deliberate *tombstone-in-place* pattern [E-23]:

- `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` carries a struck-through status line and a
  `⚠ SUPERSEDED — DO NOT USE FOR IMPLEMENTATION` banner in its header;
- `archive/README.md` is a **registrar** explaining the supersession and stating the file is deliberately
  "retained immutable at the repository root as historical record";
- `docs/02-architecture/README.md` repeats it: "SUPERSEDED … Never implement from it";
- `canonical-reference-map.md` gives it a tier row of `ARCHIVED`;
- `docs-lint` rule 3 **enforces** that no document may cite `archive/` except three allowlisted registrars.

`archive/` is empty **by design**. The mechanism is the tombstone plus the registrar, and it was executed
properly — four independent markers and a lint rule, for a single superseded document.

That pattern was then applied to **2 of the next 24 superseded documents** [E-22].

The significance: *"they didn't know how"* is eliminated as an explanation. The project knew exactly how,
proved it, protected it with a lint rule — and abandoned it the moment the gate that would have demanded
it stopped running. This is RC-1 leaving a fingerprint on RC-5.

### 8.4 Foundation flaw #4 — a false state oracle, live since day two

`docs/06-reference/README.md` declares that IMP §12's `docs/X.md` paths "resolve here", and states a
failure oracle:

> "Empty until the creating milestone runs. **A missing TDS here means its milestone has not executed.**"

The directory has been empty since 2026-07-03 and is empty today. The TDS files were created at the
IMP §12 paths (`docs/EVENTLOG_FORMAT.md` etc.) by M01, M11, M12 and M14 [E-17].

Note the authority analysis: **the implementation was right.** IMP §12 is Tier 1; the KB README is Tier 2;
authority resolves upward, so M01 correctly followed IMP. The *index* is the bug — exactly as
`07-indices/README.md` says it must be ("the target wins and the index row is a bug").

So a Tier-2 view unilaterally contradicted a Tier-1 plan, was correctly ignored by implementation, and
**stated a false failure condition for 70 days without anyone noticing.** Read literally today, it asserts
that M01, M11, M12 and M14 have never executed. A control that is permanently, loudly wrong and never
triggers is a control nobody reads.

### 8.5 What the foundation got right — verified, not assumed

The brief forbids assuming the engine is correct, so I tested it. A clean `git clone` of `HEAD` into a
scratch directory **builds clean and passes the full test suite** [E-05], on go1.26.5.

The tracked engine is real, self-contained, and healthy. M00–M14 held 7/7 module conformance across
fifteen consecutive milestones. Nine milestones were individually reviewed and merged. Gate G1 produced
eight real founder adjudications. M08 verification genuinely *failed* and forced a revision card.

**The foundation's documentation system worked, and produced a working engine. It was not defeated by its
own design defects — those stayed latent for two months. It was defeated when its terminal gate stopped
running.**

---

## 9. Documentation category analysis

### 9.1 Categories present

| # | Category | Count | Required? | Assessment |
|---|---|---:|---|---|
| 1 | Constitutional | 1 | **Yes** | Healthy. Never contradicted. |
| 2 | Architectural (frozen) | 3 | **Yes** | Healthy; `ADR-001` namespace since collided [E-21]. |
| 3 | Product requirements | 1 | **Yes** | Healthy. |
| 4 | Normative specs (TDS-01..07) | 7 | **Yes** | Content sound; **location wrong vs its own index** [E-17]. |
| 5 | Decision records (EDR) | 11 | **Yes** | **The single healthiest category in the project** — one page, numbered, never duplicated, never contradicted, zero collisions. The model for §11. |
| 6 | User documentation | 4 | **Yes** | Healthy (`CLI.md`, `DSL.md`, `PLUGIN_GUIDE.md`, `PROVIDERS.md`). |
| 7 | Governance / protocol | 4 | **Yes** | Sound as written; **unenforced since 08-21**. |
| 8 | Milestone modules | 174 | **Yes, while live** | 7/7 through M14, then collapse. Should become immutable record on close. |
| 9 | Execution cards | 49 | **Yes, while live** | Worked well. Ephemeral by nature; currently kept forever. |
| 10 | Pre-canon investigations | 4 | Historical only | Correctly superseded (1 of them); the rest carry no marker. |
| 11 | Roadmaps / planning passes | ~30 | **Dangerous** | §9.2. |
| 12 | Audit / verification reports | ~35 | **Dangerous** | §9.2. |
| 13 | Terminal verdicts (`FINAL_*`) | ~12 | **Most dangerous** | §9.2. |
| 14 | Registers (debt / defect / deferred) | ~6 | Merge | Four parallel registers for one concept. |
| 15 | Q&A / reconstruction answers | 3 | Delete | Conversation transcripts as documents; no stable question. |
| 16 | Generic agent toolkit (`.agent/`) | 55 | **Out of scope, actively harmful** | §6.5 — zero AWIS awareness, conflicting conventions. |

### 9.2 The three dangerous categories, and precisely why

**Terminal verdicts** (`FINAL_VERDICT.md`, `FINAL_RECOMMENDATION.md`, `FINAL_RELEASE_VERDICT.md`,
`STARTLINE_FINAL_VERDICT.md`, `GUI_BETA_FINAL_RECOMMENDATION.md`, `GUI_PHASE1_FINAL_RECOMMENDATION.md`,
`TRUTH_CLOSURE.md`…). The word *final* asserts terminality that the filesystem cannot enforce. Five files
named `FINAL_VERDICT.md` exist; two hold opposite verdicts [E-19]. Each was final when written. **A
verdict is a value, not a document** — it must live at one stable address and be *replaced* there, never
appended beside.

**Audit reports.** Each re-derives truth from code — correct as an act. But an audit that emits a new
document instead of updating a register is an **append-only truth store with no compaction** (RC-4). The
correct output of an audit is a *diff to a register*, plus evidence — not a new authority.

**Planning passes.** `docs/09-gui-planning/` handled this as well as it can be handled (six labeled
passes, explicit "no new investigation" tags) and still produced 38 documents and five competing terminal
answers. This proves the category is dangerous *even under excellent discipline*: **plans must be
versioned, not accumulated.**

### 9.3 Categories that created confusion by colliding

25 distinct ID prefixes across the corpus [E-21], with confirmed collisions:

- **`ADR-001`** = "Step as the Fundamental Workflow Primitive" (Blueprint, Tier 0, **frozen**) *and*
  "AWIS Intelligence Architecture" (`docs/12`, 2026-09-06). A Sep-6 document silently overwrote a
  frozen Tier-0 identifier.
- **`B-1`** = an engine-hardening defect (`RebuildState` destroys `waiting`) *and* an intelligence
  blocker (SDK arity). Two registers, one namespace.
- **`G-1`** = a PRD platform goal *and* a GUI milestone; adjacent to `QG-1` and `NG-1`.

Compounding this, the two 2026-09-06 lineages are **citation-isolated**: of 15 intelligence documents,
1 cites the GUI/release corpus; of 44 GUI/release documents, **0** cite the intelligence corpus [E-24].
Two investigations ran over the same code on the same day and never met.

### 9.4 Categories to merge

`ARCHITECTURAL_DEBT_REGISTER` + `DEFERRED_TECHNICAL_DEBT` + `DEFERRED_WORK_REGISTER` +
`VERIFIED_DEFECT_REGISTER` + `PHASE2_BLOCKERS` + `RELEASE_BLOCKERS` → **one defect/debt register**, one
ID namespace, one address.

`ENGINE_READINESS_SCORECARD` + `OPERATIONAL_READINESS_REVIEW` + `RELEASE_READINESS_REPORT` +
`SCALABILITY_ASSESSMENT` + `REPOSITORY_HEALTH_REPORT` → **one release-readiness document**, replaced in
place per cycle.

---

## 10. Corpus classification (Question 5)

Scope: **319 AWIS documents** (excluding the 55-document `.agent/` toolkit and 6 `.claude/` files, which
are not AWIS documentation — see §13.6). 349,146 words.

| Disposition | Docs | % | Contents |
|---|---:|---:|---|
| **PRESERVE** — carries authority forward | **36** | **11%** | Tier-0 canon (4); TDS-01..07 (7); user docs (4); EDRs (11); governance: `AWIS_EEOS`, `EEOS`, `AEO`, `IKB` (4); `CLAUDE.md`, `README.md` (2); `apps/oip/docs/OIP_DB.md`, plugin/scaffold READMEs (3); `TRUTH_CLOSURE.md` (1, as the seed of the new state register) |
| **MERGE** — collapse into ~8 successors | **89** | **28%** | 25 root audit docs → 1 state register + 1 defect register; `docs/10` (6) → same; `docs/11`+`docs/12` (15) → 1 ADR + 1 program; `docs/09` (38) → 4 (PRD, architecture, roadmap, work breakdown); `docs/08` (5) → 1 milestone record |
| **ARCHIVE** — immutable history, zero authority | **180** | **56%** | `docs/05-implementation` M00–M17 modules + cards (174); `IMP` + verification report (2); pre-canon investigations (4) |
| **SUPERSEDE** — tombstone in place, keep readable | *(24, a subset of MERGE)* | — | Every document `TRUTH_CLOSURE.md` §7 lists, plus the 3 it missed (`docs/10/FINAL_VERDICT.md`, `docs/09/FINAL_VERDICT.md`, `docs/09/STARTLINE_FINAL_VERDICT.md`) |
| **DELETE** — no historical value | **14** | **4%** | 4 byte-identical duplicates in `docs/10` [E-20]; 6 pointer-stub READMEs (`docs/00`–`04`, `06`) replaced by one router; `archive/README.md` (mechanism redesigned); `GEMINI.md`, `CODEBASE.md` (vestigial); `AWIS_RECONSTRUCTION_ANSWERS.md` (transcript) |

**Answer to Q5: approximately 11% of the current corpus should remain authoritative** — about 36
documents. A further 56% becomes immutable historical record with authority explicitly set to zero.
The final governing system is **~45 authoritative documents** (§11), down from 319 — a 7:1 reduction
in *authority*, with almost nothing actually lost, because the 56% is preserved as history.

The important number is not 11%. It is this: **the target system has ~45 authoritative documents and
~10 stable addresses that are allowed to answer a question.** Corpus size was never the problem;
**number of things permitted to be authoritative** was.

---

## 11. Future-state documentation architecture

Designed for the finished AWIS vision — Engine, GUI, API, Intelligence layer, visual workflow builder,
multi-user platform, and future domains — not for the current project.

### 11.1 The five governing principles, each derived from a specific failure

| # | Principle | Failure it eliminates |
|---|---|---|
| **P1** | **One question, one address.** Every recurring question resolves to exactly one stable path. Answers are *replaced at the address*, never appended beside it. | 5 × `FINAL_VERDICT.md`; RC-4; RC-5 |
| **P2** | **Authority is declared in frontmatter, never inferred from location.** Every document carries `tier`, `status`, `owner`, `supersedes`, `superseded_by`, `verified_against` (a commit SHA). | RC-6; §8.1; the root namespace problem |
| **P3** | **Every document has a consumer, or it is not written.** Each type names the gate, control, or build that reads it. Artifacts with no consumer are deleted from the contract, not made optional. | RC-3; the M15→M18 conformance collapse |
| **P4** | **Controls run in CI, on the whole repository, and are never disabled — only satisfied.** A red docs gate blocks merge exactly as a red test does. | RC-7; RC-8 |
| **P5** | **Untracked is nonexistent.** Nothing may be cited that is not committed. Uncommitted work is not "in progress"; it is outside the system. | E-02; E-06; the entire Phase VII mechanism |

### 11.2 Target tree

```
/                                   ← ONLY: README.md, CLAUDE.md, LICENSE, build files
                                       ENFORCED: no other .md at root, ever (CI rule R1)

docs/
  canon/                            ← T0 · FROZEN · changes require founder ratification
    AUTHORITY.md                    ← ★ the tier model itself, AT TIER 0 (fixes §6.2)
    GOVERNANCE.md                   ← ★ documentation lifecycle: placement, naming,
                                       supersession, retention, ID namespaces (fixes §8.2)
    CONSTITUTION.md
    ARCHITECTURE.md                 ← blueprint + finalization, merged, ADR-* namespace owner
    PRD.md
  spec/                             ← T1 · NORMATIVE · the contract code must satisfy
    TDS-01-eventlog.md … TDS-07-cli-contract.md
    (+ per-domain specs as domains are added: TDS-08-http-api, TDS-09-gui-protocol, …)
  decisions/                        ← T1 · APPEND-ONLY, IMMUTABLE — the EDR model, scaled
    ADR-0001.md …                   ← architecture; ONE global namespace, never reused
    EDR-0001.md …                   ← implementation; one page each
  state/                            ← T2 · LIVING · ★ EXACTLY ONE FILE PER QUESTION
    STATE.md                        ← what is executing now (the ledger)
    TRUTH.md                        ← what is true now; audits UPDATE this, never fork it
    DEFECTS.md                      ← one register, one ID namespace (replaces 6)
    READINESS.md                    ← one release verdict, replaced in place (replaces 5)
    DECISIONS-OPEN.md               ← what awaits the founder, with age-in-days
  programs/                         ← T3 · LIVING while open → ARCHIVED on close
    <NN>-<domain>/                  ← engine · gui · api · intelligence · platform
      CHARTER.md  PLAN.md  WORK.md  VERIFICATION.md   ← ★ exactly 4 files, enforced
  guides/                           ← T4 · user-facing; CLI.md, DSL.md, PLUGIN_GUIDE.md, …
  history/                          ← T5 · IMMUTABLE · authority = 0 · never cited as truth
    milestones/M00…M17/             ← completed modules + cards, frozen
    superseded/                     ← tombstoned documents, readable, inert
    investigations/                 ← pre-canon research, audits-as-written
```

### 11.3 The six hierarchies the brief asks for

**Document hierarchy** — `canon → spec → decisions → state → programs → guides → history`. Six live
tiers, one dead one. A document's tier is declared in frontmatter (P2) and *validated* against its
directory by CI; the directory is a convenience, never the source of authority.

**Authority hierarchy** — resolves upward, unchanged from the foundation's model, which was sound. The
fix is that `AUTHORITY.md` now lives at **Tier 0** rather than inside the tier it subordinates (§6.2),
and every document declares its own tier rather than inheriting it from a path.

**Execution hierarchy** — `program → milestone → card → commit`. One program per domain. A milestone is
the unit of merge. **A card is the unit of dispatch and the unit of commit** — the M07–M17 card model
worked and should be kept verbatim. Live execution state exists at exactly one address: `state/STATE.md`.

**Verification hierarchy** — four independent levels, each with a real consumer:
`card acceptance (the builder) → milestone DoD (an independent verifier on a clean tree) → program gate
(CI, all controls green) → release readiness (state/READINESS.md, replaced in place)`.
Verification **updates** `state/TRUTH.md`; it never emits a new authority. This is the direct fix for RC-4.

**Planning hierarchy** — `CHARTER (why, scope, non-goals) → PLAN (architecture + sequence) → WORK (cards)
→ VERIFICATION (what was proven)`. **Exactly four files per program, enforced by CI.** A second planning
pass *edits these four in place* under git history; it never adds a fifth file. `docs/09-gui-planning`'s
38 documents become 4, and its six-pass history is preserved where it belongs — in the commit log.

**Audit hierarchy** — audits are **acts, not artifacts**. An audit produces: (a) a diff to
`state/TRUTH.md`, (b) rows in `state/DEFECTS.md`, (c) a verdict written *at* `state/READINESS.md`, and
(d) its full working record filed in `history/investigations/` with `tier: 5, authority: none`. **No
audit may create a document at any live tier.** This single rule would have prevented all 25 root-level
documents.

### 11.4 Minimum viable system (Question 10)

Governing a full AWIS — Engine, GUI, API, Intelligence, visual builder, multi-user platform:

| Layer | Files | Note |
|---|---:|---|
| Canon | 5 | Authority, Governance, Constitution, Architecture, PRD |
| Normative specs | 7 + ~2/domain | TDS-01..07 today; one per new protocol surface |
| Decisions | grows ∞, 1 page each | ADR + EDR, append-only, immutable — the proven model |
| State | **5** | STATE, TRUTH, DEFECTS, READINESS, DECISIONS-OPEN — **hard cap** |
| Programs | **4 × active programs** | ~5 domains → 20 files |
| Guides | ~6 | one per user-facing surface |
| History | unbounded, authority 0 | costs nothing; answers nothing |

**~45 authoritative documents govern the finished platform.** The system scales by adding *programs* and
*decisions* — the two categories that proved healthy — and never by adding state files or terminal
verdicts, the two categories that proved fatal.

---

## 12. Canonical authority hierarchy

The complete, unambiguous resolution order for the target system. Ties resolve upward; conflicts are
recorded, never silently resolved.

```
T0  CONSTITUTION · AUTHORITY · GOVERNANCE          FROZEN — founder ratification only
T1  ARCHITECTURE · PRD · TDS-* · ADR-* · EDR-*     NORMATIVE — supersede via new ADR, never edit
T2  state/*                                        LIVING — single address per question; the ONLY
                                                   place a current answer may be read
T3  programs/<domain>/{CHARTER,PLAN,WORK,VERIFICATION}   LIVING while open; archived on close
T4  guides/*                                       LIVING — descriptive; never normative
T5  history/**                                     IMMUTABLE · AUTHORITY = 0 · NEVER cited as truth
```

Three rules with no exceptions:

1. **Code outranks every tier for questions of fact.** Documents state *intent* and *decisions*; only
   execution states *behavior*. When a document and the code disagree about behavior, the document is a
   defect — file it in `state/DEFECTS.md`. (This is the one durable lesson from the audit corpus, and
   `TRUTH_CLOSURE.md` states it well: *"Reports are hypotheses. Documentation is intent. Runtime is truth."*)
2. **Uncommitted artifacts have no tier.** They are not T5; they are outside the system entirely and may
   not be cited by anything (P5).
3. **Only T2 may be quoted as current.** T0/T1 say what was *decided*; T3 what is *planned*; T5 what was
   *thought*. If a question is about *now*, the answer is at exactly one T2 address or it does not exist.

---

## 13. Remediation architecture

Design only, per the brief. Ordered by dependency: each stage makes the next safe.

### 13.1 Stage 0 — Stop the bleeding (before any reorganization)

Reorganizing an unversioned corpus destroys evidence. Nothing below may begin until:

1. **Commit everything.** 78 documents and 42 source files enter version control, in clearly-labeled
   commits that preserve the bulk-materialization fact (`docs: import 09-gui-planning as authored
   2026-09-06`). This is a *forensic* act as much as a hygienic one — it is the last moment the mtime
   evidence in §2 Phase VII can be preserved in a commit message.
2. **Resolve the branch question.** Either merge M15→M16→M17→`engine-hardening`→`main`, or declare
   `engine-hardening` the base and reset `main` to it. **"The engine" is currently ambiguous** and has
   been for three weeks — exactly as `docs/09-gui-planning/FINAL_VERDICT.md` said on 09-01.
3. **Reconcile or formally retire `STATE.md`.** It has been wrong for 21 days. EEOS's own rule for this
   case is explicit: *"ledger contradicts repo → STOP — never 'fix' STATE to match a guess."*

### 13.2 Stage 1 — Install the controls *before* moving a single file

This ordering is the core of the remediation, and it is the one lesson of RC-7: **a corpus reorganized
without enforcement returns to entropy; controls installed without reorganization at least stop the
growth.** Controls first.

`scripts/docs-lint.sh` is rewritten from a module-contract checker into a corpus control, wired into
`make verify` and CI as a **blocking** gate:

| Rule | Check | Prevents |
|---|---|---|
| R1 | No `.md` at repository root except an allowlist of 3 | RC-6 · the 25-document explosion |
| R2 | Every `.md` under `docs/` has valid frontmatter: `tier`, `status`, `owner`, `verified_against` | RC-4, RC-5 · authority by assertion |
| R3 | Declared `tier` matches directory tier | Silent tier drift |
| R4 | **No untracked `.md` or source file anywhere in the tree** | E-02, E-06 · **this alone catches both September failures** |
| R5 | `status: superseded` ⇒ `superseded_by` present **and resolvable** | RC-5 · the 22 untombstoned documents |
| R6 | No two files share a basename across the corpus | E-19, E-20 · 5 × `FINAL_VERDICT.md` |
| R7 | Filenames may not contain `FINAL`, `TRUE`, `REAL`, `ACTUAL`, `DEFINITIVE` | §9.2 · terminality that cannot be enforced |
| R8 | Every ID prefix is registered in `GOVERNANCE.md`; no prefix defined twice | E-21 · `ADR-001` collision |
| R9 | `state/` contains exactly the 5 permitted filenames | RC-4 · re-derivation growth |
| R10 | Each open `programs/<d>/` has exactly its 4 files | §9.2 · the 38-document pass accumulation |
| R11 | Every `verified_against` SHA exists and is an ancestor of `HEAD` | Claims verified against vanished trees |
| R12 | Open milestone in `STATE.md` ⇒ ledger mtime within N days of `HEAD` | **RC-1 — the missing liveness control** |

**R12 is the single most important rule in this report.** It is the alarm that did not exist on
2026-07-10. It converts "blocked on founder" from a silent, indefinitely-valid state into a visible,
ageing, CI-surfaced one.

### 13.3 Stage 2 — Collapse the five dangerous categories to five addresses

Mechanical, once R1–R12 are live: build `state/TRUTH.md` (seeded from `TRUTH_CLOSURE.md`, which already
did this work well), `state/DEFECTS.md` (merging six registers into one ID namespace),
`state/READINESS.md` (one verdict, replaced in place), `state/DECISIONS-OPEN.md`, and reconcile
`state/STATE.md`. Every source document is tombstoned per R5 and moved to `history/superseded/`.

### 13.4 Stage 3 — Restore the milestone architecture (Question 7)

**Yes — it must be restored.** §8.5 is the argument: under it, fifteen consecutive milestones held 7/7
conformance and produced an engine that builds and passes its tests from a clean clone today. It is the
only part of this project's process with a *proven* track record. It failed by abandonment, not by design.

Three changes, each closing a specific verified failure:

1. **`program` becomes the container above `milestone`.** Engine, GUI, API, Intelligence and Platform each
   get `programs/<domain>/`. Milestones are numbered **within** a program (`engine/M18`, `gui/G0`), which
   ends the M-series-vs-G-series namespace competition (§7.1) *without* forcing GUI work into an
   engine-numbered sequence where it never belonged.
2. **E-MERGE gets a liveness control (R12) and an explicit stacking budget.** Stacking is legitimate — the
   2026-07-10 directive was not wrong — but it must be *bounded*: no more than N unmerged milestones,
   enforced, with age surfaced in CI. The failure was never stacking; it was **unbounded, invisible**
   stacking.
3. **`M18-hardening-release` is materialized retroactively**, and the `docs/08-engine-hardening/` work is
   filed into it as the historical record it always was. This closes the one milestone the project
   skipped [E-16] and removes the precedent that a phase may invent its own folder.

**Where GUI planning should live:** `programs/gui/`, as 4 files, with the current 38 preserved in
`history/programs/gui-2026-09/`. **Where future platform planning should live:** `programs/<new-domain>/`,
same 4 files. **How a new domain enters without causing drift:** it files a `CHARTER.md`, registers its ID
prefix in `GOVERNANCE.md` (R8), and gets a `programs/` directory — and it inherits every control
automatically, because the controls are corpus-wide rather than directory-specific (RC-8).

### 13.5 Stage 4 — Close the authority-model circularity

`AUTHORITY.md` and `GOVERNANCE.md` are written **as Tier-0 documents** (§6.2, §8.2). These are the two
documents the foundation never wrote, and their absence is why every other flaw was undetectable.

### 13.6 Stage 5 — Resolve the third instruction layer

`.agent/` (55 documents, zero AWIS awareness, conflicting branch conventions [E-26]) must be either
**removed**, or **vendored under `history/` with authority 0 and excluded from agent discovery**. A
repository cannot have three mutually-unaware instruction layers and a precedence note as the tiebreak.
This is a live conflict today, not a historical one.

---

## 14. Migration strategy

**Sequencing constraint (non-negotiable):** *commit → control → collapse → restructure → restore.*
Every other order fails. Restructuring before committing destroys forensic evidence. Restructuring before
controls reproduces the entropy within weeks — that is the demonstrated lesson of RC-7.

| Stage | Action | Effort | Reversible? | Gate to proceed |
|---|---|---|---|---|
| 0 | Commit all 78 docs + 42 source files; resolve `main` vs `engine-hardening`; reconcile `STATE.md` | ~1 day | Yes | Clean clone builds **and serves the GUI** |
| 1 | Write `AUTHORITY.md` + `GOVERNANCE.md`; implement R1–R12; wire to `make verify` + CI **non-blocking** | ~2 days | Yes | Controls run; failure list is finite and enumerated |
| 2 | Add frontmatter corpus-wide (scriptable from the classification in §10) | ~1 day | Yes | R2/R3 green |
| 3 | Collapse to the 5 `state/` addresses; tombstone 24+3 documents per R5 | ~2 days | Yes (git) | R5/R6/R9 green |
| 4 | Move `history/`; create `programs/`; collapse `docs/09` 38→4 | ~2 days | Yes (git mv) | R10 green |
| 5 | Materialize `engine/M18`; restore the milestone machine; **flip CI to blocking** | ~1 day | — | Full `docs-lint` green; R12 armed |

**~9 working days, no engineering risk to the engine** — every step is documents, git operations, and a
shell script. Stage 0 is the only stage with any urgency, and it has the highest value per hour of
anything in this report: it is currently the case that `rm -rf` in one working directory destroys the
entire Beta deliverable and the complete audit corpus, permanently, with no copy anywhere.

**Do Stage 0 today, independently of whether the rest of this architecture is ever adopted.**

---

## 15. Risks

| # | Risk | L | I | Mitigation |
|---|---|---|---|---|
| R-1 | **Stage 0 never happens; a `git clean`, disk failure, or fresh clone destroys 42 source files and 78 documents that exist on exactly one machine** | Med | **Fatal** | Do it today. One commit. It is the only irreversible risk in this document. |
| R-2 | The reorganization becomes the project — a 10th documentation program producing a 40th planning document | **High** | High | Hard cap: 9 days, 5 state files, 4 files per program, enforced by R9/R10. The failure mode of this report is that it is read and then *elaborated*. |
| R-3 | Controls installed but left non-blocking "until the backlog clears" — **an exact replay of 2026-08-21** | **High** | **High** | Stage 5 flips to blocking as its definition of done. Precedent proves a promise in a comment is not a mitigation. |
| R-4 | E-MERGE stalls again; R12 fires and is ignored | Med | High | R12 must block CI, not warn. A gate with no teeth is the failure being remediated. |
| R-5 | Tombstoning is read as rewriting history / losing work | Low | Med | Nothing is deleted except 14 exact duplicates and stubs; 56% is *preserved* as immutable history. Authority is removed; text is not. |
| R-6 | Frontmatter is added and then not maintained | Med | Med | R2/R3/R11 are mechanical and blocking; unmaintained frontmatter fails CI like an unmaintained test. |
| R-7 | A future domain (multi-user, billing) needs a category this architecture lacks | Med | Low | `programs/` + `decisions/` are the extension points; both are proven and unbounded. Adding a **state file** requires amending `GOVERNANCE.md` at Tier 0 — deliberately expensive. |
| R-8 | This report becomes the 6th competing `FINAL_*` document | **Med** | Med | It declares no verdict on release readiness, claims no tier, and prescribes its own disposition: on adoption it is tombstoned into `history/investigations/` and its conclusions move to `canon/GOVERNANCE.md` and `state/TRUTH.md`. **It is an audit, and by its own §11.3 rule an audit is an act, not an authority.** |

---

## 16. Adversarial review — falsification log

The brief requires that major conclusions be attacked. Each hypothesis below was tested against evidence
with the intent of destroying it. Three of my own working hypotheses did not survive and are corrected in
the body.

| # | Hypothesis attacked | Attack | Outcome |
|---|---|---|---|
| A-1 | **GUI planning is the root cause** | Date it against every control failure | **REFUTED.** Postdates the gate stall by 51 days, the ledger death by 9, the abandonment of the milestone architecture by 4. Contributed zero root-level files. §7.2 |
| A-2 | **GUI planning is at least the worst-governed cluster** | Compare cluster governance artifacts | **REFUTED — reversed.** It is the best-governed cluster produced after M09, and it correctly diagnosed the actual root cause on 09-01. §7.3 |
| A-3 | **The `08-` vs `09-` GUI-readiness conflict is a real contradiction** | Read both at full scope | **REFUTED.** Both name the same two blockers; GUI planning scopes them by phase. Scope elision in headlines, not disagreement. §7.4 — *this was my own initial finding, and it was wrong* |
| A-4 | **The root-level reports "frequently disagree"** (brief's premise) | Trace the full supersession chain | **LARGELY REFUTED.** 09-03 → 09-05 → 09-05/06 → 09-09 is temporally coherent with declared supersessions. `RELEASE_AUDIT_REPORT` + `FINAL_RELEASE_VERDICT` are a *pair*, not a fork — **I initially misread this as a fork and the evidence corrected me.** One genuine contradiction survives: the two same-named `FINAL_VERDICT.md` [E-19]. The corpus's problem is untombstoned ancestors, not disagreement. §6.3 |
| A-5 | **Supersession was never implemented** (and a prior report's *"`archive/` used 0 times"*) | Read the ENGINEERING_ARCHITECTURE_BLUEPRINT header and the registrar | **REFUTED, and the prior report is wrong.** Supersession was implemented *correctly*, with four markers and a lint rule; `archive/` is empty **by design**. The real finding is worse: the pattern was proven and abandoned. §8.3 |
| A-6 | **The foundation was sound** (brief's Tier-1 presumption) | Test design and execution separately | **SPLIT.** Execution: sound — clean clone builds and tests green [E-05], 7/7 conformance for 15 milestones. Design: three latent structural defects (§8.1–8.4), dormant for two months. Neither "flawed" nor "correct". |
| A-7 | **Root proliferation is the root cause** | Test whether root growth precedes or follows collapse | **REFUTED as cause, CONFIRMED as amplifier.** Zero root documents added 07-08 → 09-05, across the entire engine build. Growth is exactly coincident with the audit phase, i.e. downstream. But §8.1 shows the *namespace design* is a genuine foundation-era cause. §17 Q9 |
| A-8 | **The engine implementation is correct** (brief forbids assuming) | Clean clone, build, full test suite | **CONFIRMED** for the tracked engine [E-05]; **the Beta deliverable does not exist in the repository at all** [E-02]. |
| A-9 | **`docs-lint`'s disablement was negligence** | Read the reasoning | **REFUTED as negligence, CONFIRMED as cause.** The reasoning is sound engineering. The defect is structural: remediation was entrusted to an unenforced comment. RC-7 |
| A-10 | **Disabling `docs-lint` is why the failures went undetected** | Test whether the control could see them | **REFUTED.** Its scope is `docs/05-implementation/` only. **None** of the failures that occurred were in its field of view. Scope, not status, was the defect. RC-8 |
| A-11 | **This is a documentation-governance failure** | Test whether code failed the same way | **REFUTED as framing.** 42 source files have zero commits ever. Documentation and code failed on the same curve, same dates, same mechanism. Any documentation-only theory is under-determined; the diagnosis must be the **integration gate**. §1, RC-1 |
| A-12 | **The 219 deleted tracked files are part of the collapse** | Enumerate them | **REFUTED — excluded as noise.** All 219 are a `.agents/` → `.agent/` toolkit swap, unrelated to AWIS documentation. |

**Conclusions that survived every attack:** RC-1 (gate stall as root cause), RC-2 (four-mode integration
decay), RC-5 (forward-only supersession), RC-6 (root namespace), RC-8 (control scope), and the finding
that documentation and code failed by one shared mechanism.

---

## 17. Direct answers to the ten questions

**Q1 — Was AWIS carrying documentation design flaws before GUI planning?**
**Yes — four, all latent, none of which had cost anything by 2026-08-30.**
*Structural:* canonical authority addressed by filesystem position, in the root namespace (§8.1).
*Governance:* **no document anywhere specifies documentation governance** — zero matches across 319 files
(§8.2). *Authority:* the tier model is filed at Tier 2, inside the layer it subordinates, and that layer
declares itself non-authoritative (§6.2). *Verification:* `docs/06-reference/` states a failure oracle
that has read false since 2026-07-03 and was never noticed (§8.4).
*Milestone/execution flaws:* **none in design.** The milestone system is the healthiest thing in this
project — 7/7 conformance for fifteen consecutive milestones, a clean-clone green build today.
Its flaws were all *operational*, and all downstream of Q4.

**Q2 — Did GUI planning introduce new failure modes?**
**Yes, two — both real, neither novel.** (1) A parallel milestone namespace (G0–G10, ~80 cards) outside
EEOS, `STATE.md`, and every control. (2) Execution outside version control — the MVP declared shipped
against code that has never been committed [E-02]. *What changed:* a second execution system.
*When:* 2026-08-30 → 09-01. *Which documents:* `GUI_ROADMAP.md` and `ENGINE_GUI_WORK_BREAKDOWN.md`
establish G0–G10; `README.md`'s fifth pass declares the uncommitted MVP shipped.
*Why controls failed to stop it:* **every control was already dead.** `docs-lint` had been out of CI for
9 days, `STATE.md` unwritten for 9 days, and no control had ever been able to see outside
`docs/05-implementation/` (RC-8). And critically: `docs/08-engine-hardening/` had set the precedent
4 days earlier by doing the same thing for the engine itself.

**Q3 — Is GUI planning the primary root cause, or the first place pre-existing weakness became visible?**
**Neither, precisely — it is the *second* place, and not even the most severe.** It is a contributor and a
symptom. The first place pre-existing weakness became visible was **engine hardening**
(2026-08-26), which abandoned the milestone architecture for the engine's own work, six days earlier,
by never materializing M18. And the weakness itself became *load-bearing* on 2026-07-10, 51 days before
GUI planning began. §7.2.

**Q4 — Why did documentation cease functioning as an execution system?**
**Because its only consumer stopped consuming it, and nothing was built to notice.**
*Exact failure chain:* §5. *Authority collapse:* canonical → gated → stalled (07-10) → unmerged (08-20)
→ uncommitted (08-30+) → unlocatable (09-06).
*Governance collapse:* ledger law → unwritten ledger (08-21) → control disabled (08-21) → no control at
all → three competing instruction layers (09-06).
*Proliferation:* one home per milestone → no home for hardening (`docs/08-`) → no home for GUI (`docs/09-`)
→ no home at all (25 documents at root).
*Contradiction:* forward-only supersession → 22 of 24 ancestors untombstoned → 5 × `FINAL_VERDICT.md`,
2 with opposite verdicts, all presenting as live.

**Q5 — What percentage should remain authoritative?**
**~11% — about 36 of 319 documents.** 28% merges into ~8 successors; **56% becomes immutable history
with authority explicitly zero**; 4% is deleted (exact duplicates and stubs). The target system has
**~45 authoritative documents and ~10 addresses permitted to answer a question.** §10.

**Q6 — The final documentation architecture.** §11, with all six required hierarchies in §11.3.
Five principles, each derived from a specific verified failure; a seven-directory tree; ~45 authoritative
documents governing Engine + GUI + API + Intelligence + builder + multi-user platform.

**Q7 — Must milestone architecture be restored?**
**Yes — it is the only process in this project with a proven record** (§8.5), and it failed by
abandonment, not design. Restored with three changes: `programs/` as the container above milestones
(ending M-vs-G namespace competition); **a liveness control on E-MERGE (R12)** plus a bounded stacking
budget; and retroactive materialization of `engine/M18`. GUI planning lives at `programs/gui/` as 4 files;
future domains enter by filing a CHARTER and registering an ID prefix, inheriting all controls
automatically. §13.4.

**Q8 — Every documentation category.** §9 — 16 categories enumerated. *Required:* constitutional,
architectural, product, normative spec, decision record, guide, governance, milestone, card.
*Dangerous:* terminal verdicts (most), audit reports, planning passes. *Created confusion:* 25 ID schemes
with confirmed `ADR-001`/`B-1`/`G-1` collisions, plus two citation-isolated lineages [E-24].
*Should be merged:* six debt/defect registers → one; five readiness documents → one; three Q&A
transcripts → deleted.

**Q9 — Is root-level proliferation a symptom, a root cause, or both?**
**Both — but in two different senses that must not be conflated, and the evidence separates them cleanly.**
*Symptom (the 25 files):* **zero** root documents were added between 2026-07-08 and 2026-09-05 — the
entire engine build — and 25 appeared in five days once the audit phase had no designated home. The
*growth* is purely downstream. [E2]
*Root cause (the namespace):* the foundation made root the **constitutional tier** [E-01], so those 25
documents landed *inside Tier 0*, where no structural property distinguishes them from the constitution.
Had canon lived at `docs/canon/`, the same 25 documents would have been merely untidy instead of
authority-destroying.
**The files are a symptom. The namespace design is a cause. Both statements are true and they are not the
same statement.**

**Q10 — Minimum documentation system for the full future AWIS.** §11.4 — ~45 authoritative documents:
5 canon, 7+ specs, unbounded 1-page decisions, **exactly 5 state files (hard cap)**, 4 files per active
program, ~6 guides, unbounded zero-authority history. It scales by adding programs and decisions — the two
categories that demonstrably stayed healthy — and never by adding state files or terminal verdicts, the
two that proved fatal.

---

## 18. Final verdict

AWIS's documentation system was well-designed for the job it was given, executed with real rigor for
fifteen consecutive milestones, and produced a working engine that builds clean and passes its full test
suite from a fresh clone today. It then collapsed — not from drift, not from GUI planning, and not from
carelessness — because **the single human gate that consumed its output stopped running on 2026-07-10,
and the system contained no mechanism capable of noticing.**

Every subsequent pathology is that absence compounding: unmerged milestones, degrading modules, a control
switched off because it was correctly reporting the decay, a ledger that stopped being written, a
hardening program that skipped the milestone built for it, a GUI program that inherited a dead system and
documented itself better than anything around it, and finally an audit phase that — having no maintained
statement of truth to update — re-derived truth from code twenty-five times and published twenty-five
authorities, none of which could retire the others.

The documents in this repository are not bad documents. Read individually, most are careful, evidenced,
and honest; several are excellent. They became collectively unusable because the system lost the ability
to say **which one is current** — and it lost that ability the moment the gate that decided currency
stopped deciding.

The proof that this is the correct diagnosis, rather than a documentation story, is that the code failed
in exactly the same way on exactly the same curve: **42 source files comprising the entire Beta
deliverable have never been committed, on any branch, in the project's history.** No theory confined to
documentation governance can explain that. This one does: the repository stopped being the system of
record, for everything.

---

## Final answer

**Was GUI planning the root cause?**
**No.** It began 51 days after the root cause, 9 days after the ledger and the last control died, and
4 days after the milestone architecture had already been abandoned by the engine's own hardening program.
It added volume and a second ungoverned namespace, and it shipped its deliverable outside version control
— both real faults. But it is measurably the best-governed cluster produced after M09, it put zero files
at root, and its own terminal verdict correctly identified the actual root cause eleven days before this
investigation, quantifying `main`'s staleness to within two commits of today's measurement. **Contributor
and high-quality symptom; not cause.**

**Was the foundation flawed?**
**Yes — but latently, and not in the way the collapse suggests.** Four structural defects: canonical
authority addressed by filesystem position in the root namespace; **no specification of documentation
governance anywhere in 319 documents**; an authority model filed inside the tier it subordinates; and a
verification oracle that read false from day two and was never noticed. All four were dormant and costless
for two months. The foundation also got the hard parts right: the milestone module contract held 7/7 for
fifteen milestones, the CONTRA/adjudication protocol genuinely worked, and the supersession pattern was
implemented *correctly* — once — with four markers and a lint rule, then never reused. **The foundation
did not cause the collapse. It determined its shape.**

**What actually broke AWIS documentation?**
**The E-MERGE gate stopped running on 2026-07-10 and nothing was built to notice.**
Documentation existed to be consumed by that gate. When consumption stopped, production of it stopped —
in strict order of least-immediate-usefulness — while the work itself continued. Integration decayed
through four modes (individually reviewed merges → one 184-file bulk merge → never merged → never
committed), the last mechanical control was switched off because it was correctly reporting the decay,
and the ledger went unwritten. From that point every question had to be answered by re-deriving truth
from code — an act that is correct once and catastrophic as a habit, because it emits a new document
instead of correcting the old one, and nothing in the system could retire the old one. Twenty-five
authorities in five days is the arithmetic of that habit, not evidence of confusion.

**What documentation architecture should govern AWIS from this point forward?**
A **~45-document system** with **one question, one address**; **authority declared in frontmatter, never
inferred from location**; **every document naming the control or gate that consumes it**; **corpus-wide
controls that run in CI and are satisfied rather than disabled**; and **untracked treated as
nonexistent**. Seven directories — `canon / spec / decisions / state / programs / guides / history` —
with `state/` hard-capped at five files and each active program hard-capped at four. Audits become acts
that *update* state, never artifacts that create authority. The milestone machine is restored under
`programs/`, and **E-MERGE gets the liveness control it never had** — the one alarm whose absence on
2026-07-10 cost this project everything that followed.

Nine days of work, no engineering risk. **One day of it — Stage 0 — is genuinely urgent**, because the
entire Beta deliverable and the complete audit corpus currently exist in one working directory on one
machine, with no copy anywhere, one `git clean` from permanent loss.

---

*This report is an audit. By its own §11.3 rule, an audit is an act and not an authority. On adoption it
should be tombstoned into `history/investigations/`, with its governing conclusions moved to
`canon/GOVERNANCE.md` and its factual findings merged into `state/TRUTH.md`. It must not become the sixth
document named FINAL.*
