# AWIS Documentation Authority Forensics

**Investigation scope:** documentation evolution and authority, not software verification.  
**Repository observed:** `/mnt/data/rj/AWIS`, branch `engine-hardening`, HEAD `8a87f70` (2026-09-05).  
**Method:** documentation structure first; governance documents second; repository history, implementation state, and executed-evidence claims only as tie-breaks. Filesystem birth/mtime was treated as checkout metadata, not authorship. Git history was treated as provenance only where a path exists in history. Untracked documents were treated as evidence, not durable authority.

## Executive summary

AWIS did not begin with a failed documentation system. It began with a deliberately layered execution system:

`OIP_CONSTITUTION.md` → architecture/finalization → `AWIS_PRD.md` → `IMPLEMENTATION_MASTER_PLAN.md` and its verification amendments → the Implementation Knowledge Base → milestone modules → human gates and merges.

That system was unusually explicit. `docs/07-indices/canonical-reference-map.md:2-15` defines upward conflict resolution; `docs/00-foundation/README.md:5-20` requires a STOP/CONTRA record rather than silent interpretation; `docs/05-implementation/README.md:3-25` makes milestone modules the operational surface; and AEO/EEOS documents assign implementation, verification, status, and merge authority to separate roles. The M00–M18 structure therefore functioned as an **execution and governance system**, with planning as its input and reports as its output. It was not merely a plan repository.

The original system succeeded because it constrained context and authority: milestones were dependency-ordered; M00/M01 froze the irreversible formats before executable logic; later work was partitioned into bounded modules; execution cards had scope walls and binary acceptance; independent verification was separated from implementation; and the human retained gate/merge authority. The tracked milestone corpus contains 174 Markdown files, and its Git history shows a coherent growth sequence from the initial corpus on 2026-07-03 through M17 in July/August.

The first unmistakable documentation breakdown is not GUI planning. It is the August 20 repository-truth/audit cascade. Commit `1b98ee2` added a repository truth audit; `f348b28` then added an explicit **audit of that audit**, followed by architecture arbitration (`8c7ac6a`) and a final decision review (`8f88458`). This was the point at which an evidence record became the subject of a competing evidence record. It is distinct from the earlier, legitimate verification of the Implementation Master Plan (the verification report explicitly tests the plan itself) and from ordinary milestone verification.

The first operational authority fragmentation appears in two stages:

1. **Epistemic duplication, 2026-08-20:** competing audits and arbitration began producing parallel accounts of repository truth.
2. **Execution/governance duplication, approximately 2026-08-30 onward:** the GUI corpus introduced G0–G10 planning and card-like execution structures outside EEOS/STATE, while its files remained untracked; release/audit documents then created root-level verdict and blocker systems beside the tracked milestone and hardening records.

The deepest cause is therefore not “GUI planning” and not simply “bad foundational documents.” It is **loss of execution ownership at the control-plane boundary**: human-only E-MERGE/gate authority did not close the implemented milestone line into a single durable branch/state record, while the governance model had no reliable, enforced tombstone/supersession mechanism for successor reports. Implementation, planning, and audit activity continued in parallel worktrees and untracked files. The resulting authority vacuum was filled by increasingly confident reports, each re-deriving the same state and often inventing a new identifier system or verdict lineage.

The foundation was not flawless. It contains an authority-order inconsistency (`canonical-reference-map.md:6-15` versus `docs/README.md:6-9` and AEO:8), stale or unamended reference material, and a model in which the authority map is itself a Tier-2 view. Those are enabling weaknesses, not the earliest cause. The decisive failure was that the system’s prescribed execution closure, durable status, and supersession controls stopped being the repository’s effective control plane.

**Final determination:** AWIS documentation failed through a combination, with the deepest mechanism being **authority collapse caused by execution-ownership loss and missing durable supersession**. Authority drifted away from the foundation; later documentation introduced competing truth systems; GUI planning accelerated the multiplication but did not originate it.

## Documentation timeline

### 1. Foundation and execution system: 2026-07-02/03

The initial Git corpus appears in commit `441f017` on 2026-07-03. It introduced the canonical root documents, the foundation README, the indices, M00–M18 module stubs, fully materialized M00/M01 material, and the Implementation Knowledge Base. The initial structure already distinguished:

- frozen design authority at the root;
- archival planning (`IMPLEMENTATION_MASTER_PLAN.md`);
- binding verification amendments;
- a living, partitioned implementation view under `docs/`;
- milestone execution modules and traceability;
- indices that point to authority rather than replace it.

M00 and M01 were then executed on July 3. M01’s README makes the design intent explicit: irreversible EventLog, workflow-definition, and grammar artifacts were to be frozen before executable logic (`M01.../README.md:1-11`). The subsequent milestone records show a dependency-ordered build from M02 through M17, with EEOS v2 introduced on July 8 and M07–M17 card-based execution material added through July 11 and later M17 revisions in August.

This chronology is important: the documentation system was initially producing implementation artifacts, not only prose. The early plan verification report is documentation about a plan, but it is a normal governance control: it challenges the plan, adopts binding amendments F-1–F-5, and feeds the execution modules. It is not evidence of failure by itself.

### 2. Hardening lineage: 2026-08-28/29

The hardening family began as root-level `docs/ENGINE_HARDENING_PLAN.md` and `docs/ENGINE_FREEZE_REPORT.md` in commit `4c483dd` on August 28. An adversarial review and correction followed in `46f807f`; commit `f004f4f` on August 29 moved the lineage into `docs/08-engine-hardening/` and added the README plus pre/post review reports.

This family remained comparatively disciplined. Its plan labels findings as executed behavior or exact source reading, and its post-review report records corrections to its own evidence, including the B-4 test-shape error. It therefore provides strong historical audit evidence for the engine-hardening phase, but it does not become the implementation authority for M00–M18 or the release authority for later GUI/API work.

### 3. First audit-on-audit cascade: 2026-08-20

The earliest explicit audit-on-audit chain is recorded in Git history:

| Commit | Event | Significance |
|---|---|---|
| `1b98ee2` | Repository Truth Audit | Re-derived M15/M16/M17 repository state. |
| `f348b28` | Audit of the Repository Truth Audit | Explicitly treated the prior audit as a hypothesis and audited it. |
| `8c7ac6a` | Architecture arbitration | Added another adjudication layer over the two audits. |
| `8f88458` | Final decision review | Reviewed the preceding review documents. |

The `f348b28` step is the first unambiguous authority duplication: the prior report is no longer merely evidence; it is challenged by a new report with its own truth claim. This is also the earliest evidence that documentation began consuming the work previously reserved for implementation-state closure. Confidence is **high** for chronology and **medium-high** for the causal interpretation, because the original `docs/06-audits/` files are not present in the current checkout, although the commits and agent-retrieved descriptions are available.

### 4. GUI planning and release audit expansion: approximately 2026-08-29 through 2026-09-03

The GUI family’s earliest conceptual authority is `GUI_MASTER_PLAN.md`, whose README identifies an August 29/30 evidence base. The current 38-file GUI family is untracked and has no Git history; its near-identical filesystem times are batch materialization times, not reliable authorship dates.

The GUI corpus evolved through several internally superseding passes: master plan, comprehensive summary, execution program, optimization pass, start line, six-card preparation package, Phase 1 package, MVP record, and Beta package. Its own README says the original G0→G1→G2 start-line conclusion was superseded by later card-level tracing (`docs/09-gui-planning/README.md:101-111`). This is evidence of planning drift and local supersession, not proof that GUI planning caused the repository-wide failure.

The release-audit family is also untracked and internally dated 2026-09-03. It evaluates `e75c1f1` plus uncommitted API/GUI work and says “REJECTED FOR BETA RELEASE” (`docs/10-release-candidate-audit/README.md:3-5,21-32`). Later root-level reports on September 5 say PASS WITH RISKS or CONDITIONAL PASS. The disagreement is partly temporal and target-dependent, but it became a governance failure because the older tracked-looking verdict remained discoverable without a tombstone while newer verdicts were untracked.

### 5. Intelligence investigation and decision packages: 2026-09-06

The intelligence-architecture family contains eight untracked documents and explicitly says “Planning and investigation only. No code was written or modified” (`docs/11-intelligence-architecture/README.md:3-7`). The seven-document decision family says “No implementation, no code changes, no commits, no roadmap” and “Supersedes nothing” (`docs/12-intelligence-architecture-decision/README.md:3-6`). These documents preserve the authority boundary in their text, but their lack of Git history means their recommendations are not durable execution authority.

### 6. Closure documents: 2026-09-06 through 2026-09-17

The later root lineage is itself diagnostic:

- `DOCUMENTATION_DIVERGENCE_REPORT.md` audits the document layer and records the same uncommitted GUI/API fact being rediscovered at least four times (`:7-16`), plus repeated re-litigation (`:35-39`).
- `TRUTH_CLOSURE.md` declares itself canonical for its audit corpus, re-tests 24 documents, and supersedes a long list of prior reports (`:1-9,188-217`), but is itself untracked.
- `AWIS_SYSTEM_HEALTH_BASELINE.md` explicitly calls itself “an audit act” and says its conclusions should be moved to durable state and the file tombstoned rather than joining the final-verdict pile (`:3-5`).

The last document recognizes the loop, but recognition is not closure. The control problem remains visible in the artifact’s own status.

## Authority evolution map

### Intended authority at project start

The intended chain was:

1. **Tier 0 foundational authority:** Constitution, architecture blueprint/finalization, canonical PRD.
2. **Tier 1 planning/verification authority:** Implementation Master Plan and binding verification amendments.
3. **Tier 2 operational authority:** IKB, AEO, EEOS, and milestone modules.
4. **Execution state:** `docs/05-implementation/STATE.md`, branch/commit state, milestone TRACEABILITY/HANDOFF records.
5. **Evidence/reporting:** verifier reports, gate briefs, hardening reports, and historical records.
6. **Human irreversibility:** founder gate verdicts and merges.

`docs/README.md:3-25`, the canonical map, AEO, and EEOS all support this shape. Reports were supposed to point to durable coordinates and not restate authority. `STATE.md` was the single source of live execution status; Git was the implementation state; TRACEABILITY/HANDOFF carried durable milestone outcomes.

### Authority during successful M00–M18 execution

The execution system preserved the intended chain reasonably well. M00/M01 were fully materialized; M02–M18 were partitioned and compiled at entry. Cards bound a worker to a bounded task. Verification was independent. Gates were human-owned. The successful characteristics were:

- dependency order and explicit blockers;
- irreversible schemas frozen before logic;
- narrow context loading rather than corpus browsing;
- immutable READY cards and revision-card handling;
- binary acceptance criteria and traceability;
- verifier independence;
- a written ledger and one-commit-per-card discipline;
- human-only merges.

These are mechanisms, not just documentation virtues. They explain why the engine implementation advanced through the milestones even though later release and GUI documentation became unstable.

### Authority fragmentation

The first fragmentation point depends on what “authority” means:

- **Evidence authority:** `f348b28` (2026-08-20), the audit of an audit, is the first explicit competing truth system.
- **Operational status authority:** by August 21, `STATE.md` stopped being a current account of the repository. It records M15/M16 at E-MERGE and M17 in C-VERIFY, while later commits and hardening work exist; `AWIS_SYSTEM_HEALTH_BASELINE.md:19-21` independently confirms that STATE predates hardening. This is the first clear durable-ledger drift.
- **Competing execution authority:** the GUI corpus introduced G0–G10 and roughly 80 card-like work units outside EEOS/STATE, with code produced uncommitted. This is the first clear second execution system.
- **Competing release authority:** the September 3 tracked audit and September 5 untracked root verdicts produced incompatible release headlines, with no common supersession register.

Thus the exact beginning is **August 20 for epistemic fragmentation**, becoming **August 21 for state/ledger fragmentation**, and **August 30 for a parallel execution system**. GUI planning was an accelerator and a visible branch of the failure, not the origin.

### Structural authority defects in the foundation

The foundation contains real weaknesses:

- `canonical-reference-map.md:6-15` orders Blueprint before Finalization, while `docs/README.md:6-9`, AEO:8, and `IMPLEMENTATION_MASTER_PLAN.md:8` list Finalization before Blueprint.
- The map is itself a Tier-2 index, while `docs/07-indices/README.md` says the target wins if the index is wrong; this makes the authority map useful but not self-authorizing.
- The canonical PRD remained frozen while later GUI/API work evaluated a scope it explicitly places in V2/V3 (`AWIS_PRD.md:1078-1123,2378-2410`).
- Supersession is forward-declared by successors but usually not inscribed on ancestors. `TRUTH_CLOSURE.md` and the system-health baseline both identify this missing tombstone pattern.

These defects made fragmentation easier to sustain, but the evidence does not show that they prevented the original milestones from executing.

## Failure-point analysis

| Question | Earliest evidence | Assessment | Confidence |
|---|---|---|---|
| Earliest documentation breakdown | 2026-08-20 repository-truth audit lineage, followed by the audit-of-audit | Documentation shifted from recording execution to adjudicating competing records. The earlier plan verification was a legitimate control, not breakdown. | High for chronology; Medium-high for causal interpretation |
| Earliest authority duplication | `f348b28`, 2026-08-20 | Explicit report-on-report authority. | High |
| Earliest conflicting truths | By Aug. 20–21: M15–M17 repository state had to be re-derived while STATE and branch reality diverged; later explicit release conflict is `docs/10` REJECTED vs root PASS/CONDITIONAL PASS. | Conflict first appears as repository-state disagreement, then becomes headline verdict disagreement. | Medium-high |
| Earliest audit-on-audit behavior | `f348b28`, after `1b98ee2` | Unambiguous audit-of-audit sequence with arbitration and final review. | High |
| Earliest documentation producing more documentation than implementation | The `1b98ee2`→`f348b28`→`8c7ac6a`→`8f88458` chain | Four successive documentary artifacts were generated around one truth question before a durable state closure. | High |
| Earliest stale status evidence | `STATE.md` last modified 2026-08-21 while hardening and later work continued; GUI gate/backlog docs later remained “not started” after code existed. | Status authority ceased to be synchronized with execution. | High |
| First competing execution system | GUI G0–G10 planning/card corpus outside EEOS/STATE, approximately Aug. 30 | First direct rival to the milestone execution vocabulary. | Medium-high |

### Causal attribution among candidate origins

**A. Foundational documentation — contributory, not primary.** The frozen foundation had authority-order ambiguity, stale references, and an unamended product scope. It was not uniformly wrong: the PRD’s V1/V2/V3 boundary is explicit, and the initial execution protocol is detailed and coherent.

**B. Engine hardening documentation — not the origin.** Hardening reports added a separate evidence family, but they generally label evidence, correct their own findings, and remain bounded to engine behavior. They exposed that unit tests and abstract audits were insufficient; they did not create the first duplication.

**C. GUI planning documentation — accelerator, not root.** GUI planning multiplied documents and introduced a parallel execution vocabulary, but the audit-of-audit cascade and ledger drift predate it. GUI planning made the fragmentation more visible and larger.

**D. Release/audit documentation — major proximate cause.** Release reports multiplied verdicts, blocker IDs, and readiness scorecards; root-level reports were often untracked; older verdicts remained in place. This converted state drift into conflicting “current truth” claims.

**E. Combination — yes.** The failure required foundation weaknesses, execution-ledger nonclosure, untracked work, and later audit proliferation. The deepest mechanism is the combination, not any one family.

## Root-cause analysis

### Why 1: Why are there contradictory current truths?

Because multiple families report on the same repository state: tracked milestone/hardening records, GUI planning records, release audits, root verdicts, intelligence investigations, and later closure documents. They use different snapshots, branch assumptions, and identifiers.

### Why 2: Why did multiple families remain live at once?

Because supersession was mostly forward-only. A successor says it supersedes an earlier report, but the earlier file remains present without a tombstone. Same-named files (`FINAL_VERDICT.md`) coexist, and newer untracked reports have no durable relationship to tracked predecessors.

### Why 3: Why was a successor allowed to become a new authority?

Because execution ownership moved out of the prescribed state channel. `STATE.md` stopped reflecting later milestone/hardening work; GUI execution was planned outside EEOS; root audit documents accumulated outside the commit/merge path. The repository no longer had one durable, mechanically enforced control plane.

### Why 4: Why did execution ownership move out of that control plane?

The model placed irreversible closure in a human-only E-MERGE/gate decision, but the available evidence shows stacked milestone branches, an unclosed ledger, and later implementation/audit work continuing in the working tree. The human gate remained the intended authority, but its outcome was not transformed into one durable, repository-visible state transition for all subsequent work.

### Why 5: Why did the system respond with more reports rather than closure?

Because report production was cheaper and locally legible than resolving branch, scope, merge, and founder-authority questions. The corpus had no effective stop condition for repeated audits, no universal ID namespace, no durable supersession/tombstone rule, and no requirement that a report’s authority depend on being committed and linked to the execution ledger.

**Deepest defensible cause:** **execution-ownership loss at the merge/state boundary, amplified by missing supersession governance, caused authority collapse.** “Documentation drift” is the symptom. “GUI planning” is a later amplifier. The mechanism is that no single durable control-plane artifact remained authoritative for implementation state, release scope, and report succession at the same time.

## Document-family classification matrix

Counts are from the current snapshot, excluding `.agent/`, `.claude/`, and `web/node_modules/` vendor/agent material: **322 project Markdown files**. The broader glob sees 390 Markdown paths, which explains why an approximate repository total near 387 can differ by inclusion rules. Counts below are inventory counts, not correctness scores. The categories are the dominant intended/current role of each disjoint family.

| Family | Count | Classification | Examples and evidence |
|---|---:|---|---|
| Root frozen design corpus | 7 | Foundational authority | `OIP_CONSTITUTION.md`, architecture blueprint/finalization, `AWIS_PRD.md`, IMP, IMP verification, tribunal record. |
| Root process/knowledge corpus | 10 | Governance authority | `IMPLEMENTATION_KNOWLEDGE_BASE.md`, AEO, EEOS, `EEOS.md`, router/readme and research/process documents. These define execution/context flow but do not outrank Tier 0/1 design. |
| `docs/00-foundation`, `docs/01`–`04`, `docs/06-reference`, `docs/07-indices` | 11 | Governance authority | Session recipe, planning pointers, TDS/reference routing, canonical map, traceability/dependency indices. The map is a governance index, not independent design authority. |
| `docs/05-implementation` | 174 | Execution authority | M00–M18 modules, cards, STATE, TRACEABILITY, HANDOFF, validation and dependency files. This is the original operational surface. |
| Top-level normative `docs/*.md` | 11 | Foundational authority | `WORKFLOW_SCHEMA.md`, `EVENTLOG_FORMAT.md`, `CLI_CONTRACT.md`, plugin/subprocess protocols, provider/DSL references. |
| `docs/edr` | 11 | Foundational authority | EDR-001…011; design decisions and semantic boundaries referenced by implementation. |
| `docs/08-engine-hardening` | 5 | Audit evidence | Hardening plan, freeze report, pre-review, post-review, README. Strong evidence for the dated hardening scope; not a replacement for milestone authority. |
| `docs/09-gui-planning` | 38 | Temporary investigation | GUI plans, roadmaps, cards, status, and recommendations. Explicitly evolving and untracked; not durable execution authority. |
| `docs/10-release-candidate-audit` | 6 | Audit evidence | Candidate audit, scorecard, blockers, drift report, verdict. Dated challenge evidence, but its rejection headline is stale relative to later lineages. |
| `docs/11-intelligence-architecture` | 8 | Temporary investigation | Explicitly planning/investigation-only and no-code. |
| `docs/12-intelligence-architecture-decision` | 7 | Temporary investigation | Proposed Option B package; explicitly supersedes nothing and awaits founder amendment. |
| Root committed release/verification reports | 6 | Audit evidence | `FINAL_VERDICT.md`, `IMPLEMENTATION_REPORT.md`, `REGRESSION_REPORT.md`, `REPOSITORY_HEALTH_REPORT.md`, `VERIFIED_DEFECT_REGISTER.md`, `VERIFIED_GEMINI_FINDINGS.md`. Durable historical evidence at HEAD, but later closure documents dispute some conclusions. |
| Root untracked release/audit/Q&A corpus | 21 | Audit evidence | `TRUTH_CLOSURE.md`, release audits, drift/divergence reports, debt registers, reconstruction answers. High-value evidence in places, but not durable authority because untracked. |
| `ENGINEERING_ARCHITECTURE_BLUEPRINT.md` and `archive/` | 2 | Superseded material | Explicitly historical/superseded standalone OIP architecture; archive convention is clear here but not applied consistently elsewhere. |
| Other nested documentation | 5 | Unknown | Application/plugin/scaffold documentation not central to the authority chain; role varies and was not assigned by a common governance map. |

**Classification totals:** Foundational authority 29; Governance authority 21; Execution authority 174; Audit evidence 38; Temporary investigation 53; Superseded material 2; Unknown 5. Total: 322. These totals classify dominant role, not truth value.

## Current documentation health assessment

### Authority health: poor

The repository still contains a legible canonical chain, but it is not the chain a contributor encounters by ordinary browsing. The tracked implementation family and root canon are durable; the GUI/release/intelligence families are predominantly untracked. The authority map itself has ordering ambiguity, and the current document set contains multiple `FINAL_VERDICT.md` files and incompatible release conclusions.

### Execution-state health: poor

`STATE.md` is intended to be the single source of live execution status, yet its latest recorded state predates the hardening and later working-tree activity. M15/M16/M17 entries remain in states that do not describe the full present repository. This is a direct failure of the execution system’s status contract, not merely a stale explanatory paragraph.

### Provenance health: poor

The 09–12 families have no Git history in this checkout. Their filesystem times are batch materialization times and cannot establish authorship or exact creation order. Root audit reports likewise include a large untracked population. A report can be technically excellent and still fail to be durable evidence if it is not attached to a reproducible repository state.

### Supersession health: poor

One explicit archive/tombstone pattern exists for the standalone architecture blueprint. Most later supersession is successor-declared only. The predecessor remains discoverable as if current. This is why an old “REJECTED” verdict and newer “PASS WITH RISKS”/“CONDITIONAL PASS” documents can coexist without a reader being able to determine the active lineage from the older file itself.

### Signal distribution

**Highest signal:** the Tier-0/1 canon for design and scope; `docs/05-implementation` for intended execution; `docs/08-engine-hardening` for dated engine evidence; and later reports that expose their methods, scope, and limitations (`TRUTH_CLOSURE.md`, `AWIS_SYSTEM_HEALTH_BASELINE.md`)—with the important qualification that the latter are untracked.

**Highest confusion risk:** root-level release/audit/verdict documents; duplicate `FINAL_VERDICT.md` files; the 09 GUI status/backlog documents that contradict later code-state descriptions; `STATE.md` when read as current after 2026-08-21; and any report using a local ID scheme without cross-reference.

### Code/document tie-break findings

The code tie-break changes behavior claims but does not erase documentation authority boundaries:

- Current API source is GET-only and route-complete on disk; it contradicts stale GUI “not started” status documents, but does not prove V1 authorization.
- Current model constants support the later correction that the earlier “fictional model IDs” claim was wrong; they do not make the untracked GUI/API release-reproducible.
- Loopback is the default server address, but an override can bind without authentication; this narrows, rather than eliminates, the security claim.
- The canonical PRD still places HTTP/web work in V2/V3. Code existence establishes implementation reality, not product-scope authorization.

## Recovery implications (not a remediation plan)

This section records what the forensic evidence says must be treated as durable signal, what is low-value for present truth, and where uncertainty remains. It does not prescribe corrective work.

### Preserve at all costs

- The frozen Tier-0 design and product-scope corpus, including the explicit V1/V2/V3 boundaries.
- The milestone execution corpus, especially cards, `STATE.md`, TRACEABILITY/HANDOFF records, validation checklists, and the dependency/authority indices as historical evidence of the original system.
- Hardening reports with their evidence labels, re-review corrections, reproduction methods, and dated commit references.
- Later closure documents’ forensic observations about tracking, supersession, repeated rediscovery, and conflicting verdicts, while retaining their untracked status as a limitation rather than silently promoting them.

### Safely ignore for current authority questions

- A report’s recency, filesystem mtime, filename “FINAL,” or numerical severity alone.
- Uncommitted planning recommendations as execution instructions.
- Duplicate audit summaries that merely restate an older finding without new evidence or a durable state transition.
- The old standalone architecture blueprint for implementation decisions; it is explicitly superseded.

### Requires further investigation before any truth claim is closed

- The exact founder/gate/merge decisions for M15–M18 and the intended release branch.
- Whether the GUI/API scope was ever formally authorized against the frozen PRD, as distinct from whether code exists.
- Which later audit findings remain valid after the latest engine-hardening state; later reports themselves disagree and are not all independently verified.
- The complete mapping among SEC-, B-, D-, RA-, AD-, UF-, and other defect identifiers.
- The historical contents of the removed/unavailable `docs/06-audits` files, beyond the Git commit evidence.

### Highest confusion risk

1. Root-level release/verdict/audit reports and their same-day variants.
2. Duplicate tracked/untracked `FINAL_VERDICT.md` paths.
3. `docs/09-gui-planning` status and backlog files whose statuses are contradicted by later source state.
4. `STATE.md` read without its last-modified boundary.
5. `TRUTH_CLOSURE.md` and similar untracked documents being mistaken for committed authority.
6. Cross-family ID schemes that appear to describe the same defect but have no mapping.

### Highest signal

1. Frozen root canon for design, scope, and invariants.
2. M00–M18 module/card/traceability records for the intended execution mechanism.
3. `STATE.md` and Git history for historical execution state, with explicit date limits.
4. Hardening reports for directly executed engine evidence.
5. Later closure reports only where their claims expose method, coordinates, scope, and limitations—and never merely because they are newer.

## Confidence ratings for major conclusions

| Conclusion | Confidence | Basis and limitation |
|---|---|---|
| The original M00–M18 documentation system was an execution/governance system, not merely a plan | **High** | Explicit module contract, cards, STATE, verifier roles, gates, and commit-linked milestone history. |
| M00–M18 succeeded because of bounded milestones, frozen interfaces, independent verification, and human gates | **High** | Repeatedly encoded in 00/04/05 docs, AEO/EEOS, and milestone growth history. This explains process success, not complete software correctness. |
| The first explicit audit-on-audit event was `f348b28` on 2026-08-20 | **High** | Commit sequence and explicit audit-of-audit path; source files are not present in current checkout. |
| The first durable ledger drift was present by 2026-08-21 | **High** | STATE’s last-modified boundary and later commits/health baseline. |
| GUI planning did not originate the breakdown | **Medium-high** | Audit-on-audit and ledger drift precede the untracked GUI family; exact causal attribution is historical inference. |
| GUI planning accelerated authority fragmentation | **High** | It created a large, evolving G0–G10 execution vocabulary outside EEOS/STATE and remained untracked. |
| Release/audit documents became a competing truth system | **High** | Explicit tracked/untracked verdict conflict, repeated rediscovery, duplicate IDs, and successor-only supersession. |
| Foundational documentation was imperfect but not the primary cause | **Medium-high** | Direct authority-order inconsistency and stale scope references exist; original execution controls nevertheless functioned. |
| The deepest cause was execution-ownership loss plus missing supersession governance | **Medium-high** | Strong convergence of STATE drift, unmerged/untracked work, human-only merge boundary, and report proliferation; exact founder intent is not fully recorded. |
| Current code can resolve behavior disputes but cannot authorize scope or establish reproducible release authority | **High** | Code shows local reality; PRD and Git tracking determine scope/reproducibility. |

## Unresolved questions

1. What exact founder verdicts, if any, closed G3/G4 or the M15–M18 E-MERGE states?
2. Was the GUI/API ever formally admitted into V1, or did implementation proceed on an unratified expansion of scope?
3. Which branch and commit were intended to be the release baseline when hardening, GUI work, and root audits diverged?
4. Were the August 20 audit-chain conclusions ever incorporated into `STATE.md`, a gate record, or a durable canonical document?
5. Why was `docs-lint` removed or left failing while the implementation corpus remained governed by a seven-file contract?
6. Which defect identifiers across the audit families are equivalent, and which are genuinely distinct?
7. Does the founder consider later untracked closure reports historical evidence, proposed authority, or neither?
8. Are the unresolved intelligence architecture recommendations still active, or did a later founder decision occur outside the repository?
9. Which claims in the later closure reports are independently reproduced versus inherited from prior audits?
10. Is the apparent current repository state a deliberate staging area, a failed merge boundary, or both?

## Final verdict

AWIS documentation did **not** fail primarily because the foundational documentation was wholly flawed. The foundation had real defects—an inconsistent authority-order presentation, incomplete supersession mechanics, and a frozen PRD that later work did not formally amend—but it successfully operated as an execution system for M00–M18.

It failed because **authority drifted away from that execution system and later documentation introduced competing truth systems, against a deeper background of lost execution ownership at the human merge/state boundary**. The first clear trigger was the August 20 audit-of-audit cascade; the first durable operational failure was the stale execution ledger; GUI planning later amplified the problem by creating a second execution vocabulary and a large untracked planning corpus; release audits then multiplied verdicts and left them coexisting without durable succession.

The most defensible single formulation is:

> **AWIS documentation failed through authority collapse: execution ownership was not converted into one durable repository state, and missing enforced supersession rules allowed every subsequent audit, plan, and verdict to remain locally authoritative. GUI planning was an accelerator and later competing truth system, not the root cause.**

This verdict is **medium-high confidence**. The chronology, document roles, tracking state, and explicit contradictions are strongly evidenced. The precise human decision or non-decision that left E-MERGE unresolved is not fully recoverable from the current repository, so that part remains an inference rather than a proven fact.
