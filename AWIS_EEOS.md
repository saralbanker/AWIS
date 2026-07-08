# AWIS ENGINEERING EXECUTION OPERATING SYSTEM (EEOS)
## Repository-Resident Execution Protocol for M07 → Baseline V1

**Document Status:** **CANONICAL — APPROVED by founder 2026-07-08.** Tier 2 operational; companion to AWIS_ENGINEERING_ORGANIZATION.md (AEO). Frozen for Baseline V1: amendments require objective implementation evidence per §13; no further protocol optimization is authorized during Baseline V1 (founder directive, 2026-07-08).
**Version:** 2.0 — V1 accepted in principle by founder 2026-07-07; V2 is the founder-directed final optimization pass (token minimization only; no redesign). All V2 deltas are consolidated in §15 and patched into the normative sections they amend; every V2 delta is marked ⬖.
**Date:** 2026-07-08 (V1: 2026-07-07)
**Produced by:** CE (Fable) EEOS design session
**Authority:** OIP_CONSTITUTION.md → Finalization → Blueprint → PRD (Tier 0) → IMP + Verification (Tier 1) → IKB, AEO (Tier 2) → **this document**. The EEOS adds no design, no scope, no schedule, no roles. Where the EEOS and any document above it disagree, the higher document wins and the EEOS entry is a defect — except the four explicit supersessions listed in §14, which amend the AEO under its own §22 trigger (c).
**Amendment trigger satisfied (AEO §22 F7c):** two consecutive implemented milestones breached the same protocol despite compliant cards — M04 (continuation card M04-C2r) and M06 (continuation cards C1r/C2r/C2r2/V1r, per M06 TRACEABILITY §Deviations) both lost agent runs mid-card and required card reconstruction plus CE disk-audit from a session holding the only copy of the card. The failing protocol is AEO §16 L1: task cards are ephemeral prompt content. The EEOS fixes exactly that class.
**Scope of validity:** M07 entry until Gate G4 approves `v1.0.0`. Intentionally unchanged for that entire window; §13 defines the only amendment path.

---

## 0. WHAT THE EEOS IS (AND IS NOT)

The AEO (§1–§25) already answers *who executes and how context flows*: six roles, four permanent subagents, task cards, the EEP loop, escalation, review, merge, budgets. The IKB already answers *where knowledge lives*: partitioned modules, 7-file contract, indices, write triggers. **Neither answers where execution state lives between sessions.** Today it lives in the CE session's conversation — which is exactly the "conversational drift" failure mode AEO §2.4 names, and which M04/M06 paid for in continuation cards.

The EEOS is the AEO plus **three mechanisms and one discipline**:

| # | Mechanism | Replaces |
|---|---|---|
| 1 | **Persisted Execution Cards** — cards are repo files, not prompt content (§4) | AEO §16 L1 (ephemeral cards) |
| 2 | **The STATE ledger** — one fixed-location file carrying all live execution state (§3.2) | AEO §15/F3 ad-hoc "state notes" |
| 3 | **The Bootstrap file** — `/EEOS.md`, the single entry point a fresh session reads first (§5) | re-explaining the process each fresh chat |
| D | **Lazy-CTO session discipline** — Fable-grade judgment concentrated at two wake points per milestone; everything between is file-driven dispatch (§6) | resident always-on Fable orchestration |

Everything else — roles, card template fields, escalation ladder, review rungs, verification independence, merge preconditions, budgets, model matrix — is the AEO and IKB unchanged, referenced by coordinate, never restated here.

**Constraint reconciliation (recorded once, binding):** where this design brief's generic model rules conflict with frozen Tier-1 text, frozen text wins (the brief's own rule: architecture is never reinterpreted). Concretely: (a) Opus participates exactly where IMP §28 designates it (M07 signal atomicity is Opus-primary) — the §28 Opus rows *are* the pre-adjudicated standing escalation criteria; outside them, Opus runs only via the AEO §20.1 upward-substitution rule. (b) Haiku (`awis-scribe`) touches production code only in the IMP §28 Haiku rows (M13/M17 pattern-exemplar batches), where the exemplar, not Haiku, carries the judgment. (c) Sonnet performs no architecture reasoning — enforced structurally by card scope walls and E1 STOP triggers, unchanged from AEO §18.

---

## 1. REPOSITORY LAYOUT (Part 1)

The EEOS adds **one root file, one ledger file, and one subdirectory per milestone**. Nothing else. Nothing moves.

```
/                                    (repo root — canonical corpus, unchanged)
├── EEOS.md                          ★ NEW: bootstrap protocol — the ONLY file a fresh
│                                      session must read first ("Load EEOS" target)
├── AWIS_EEOS.md                     this specification (rationale + copy-ready blocks)
├── AWIS_ENGINEERING_ORGANIZATION.md AEO — roles, protocols (unchanged)
├── .claude/agents/awis-*.md         the four L0 identities (frozen, unchanged)
└── docs/
    ├── 00-foundation/README.md      session recipe + contradiction protocol (unchanged)
    ├── 05-implementation/
    │   ├── STATE.md                 ★ NEW: the execution ledger — all live state
    │   ├── V-COMMON.md              ⬖ NEW (V2): the invariant verification block — clean-tree
    │   │                              procedure + make build/test/lint/race + AWIS-E1 — written
    │   │                              once, cited by every MXX-V1 (§15 D5)
    │   └── MXX-<slug>/
    │       ├── README.md … DEPENDENCY_MAP.md   (IKB §2 seven-file contract, unchanged)
    │       └── cards/               ★ NEW: persisted Execution Cards for MXX
    │           ├── MXX-C1.md …      implementation cards (immutable once READY)
    │           ├── MXX-C2r.md …     revision cards (new file per revision, M06 naming)
    │           └── MXX-V1.md        verification card
    └── 07-indices/                  lookup surface (unchanged)
```

**Directory responsibilities:**
- **Repo root** — frozen canonical corpus plus the two process constitutions (AEO, EEOS) and the bootstrap file. Root is where a session lands; the bootstrap must be addressable without any prior knowledge, hence `/EEOS.md`.
- **`docs/05-implementation/`** — the operational surface (IKB §1). It gains the ledger because execution state *is* implementation state and must ride in milestone PRs like every other module update (IKB §12: docs and code never out of sync on `main`).
- **`MXX-<slug>/cards/`** — the milestone's delegation record. Cards are execution artifacts of exactly one milestone, so they live in its module, are reviewed in its PR, and become part of the historical execution record post-v1.0.0 (IKB §13). The 7-file contract is untouched — `cards/` is a subdirectory, and the CI docs-lint rule extends to: *materialized dirs have exactly 7 files plus an optional `cards/` dir*.
- **`.claude/agents/`** — L0 identities, frozen for the baseline (AEO §16). The EEOS requires zero identity edits: the dispatch prompt (§4.3) points agents at their card file, which their existing `Read` tool and "read only what the card names" constraint already cover.

Rejected placements: a top-level `eeos/` or `.agents/eeos/` directory (new hierarchy for three files — fails the simplicity test); state inside each milestone module (fresh session would need to know the current milestone to find the state that names the current milestone — circular); state in `.claude/` or memory (invisible to PRs and to non-Claude tooling; the repository, not the harness, is the operating system).

---

## 2. PERMANENT PROTOCOL HIERARCHY (Part 2)

Every protocol file, exhaustively. Files marked ● exist; ★ are created on approval; nothing else is ever created.

| File | Purpose | Owner (writes) | Lifecycle | Update policy |
|---|---|---|---|---|
| ● `OIP_CONSTITUTION.md`, Blueprint, Finalization, `AWIS_PRD.md` | Tier 0 frozen design | nobody | immutable | never (IKB §13) |
| ● `IMPLEMENTATION_MASTER_PLAN.md` + Verification | Tier 1 archival plan | nobody | immutable | never |
| ● `IMPLEMENTATION_KNOWLEDGE_BASE.md` | corpus partitioning + write triggers | CE | stable | own §13 triggers only |
| ● `AWIS_ENGINEERING_ORGANIZATION.md` | who executes; all inter-role protocol | human on CE proposal | stable | §22 triggers only |
| ★ `AWIS_EEOS.md` (this file) | execution-state layer: rationale + normative spec | human on CE proposal | stable | §13 below |
| ★ `EEOS.md` | bootstrap: startup sequence, phase machine, wake rules, dispatch template | human on CE proposal | **frozen after approval** | only via an approved edit to this spec — it contains zero milestone-specific content, so it should never need one |
| ★ `docs/05-implementation/STATE.md` | the execution ledger — current milestone, phase, card statuses, blockers, next action | **CE only** (scribe may draft; CE commits) | mutated at every phase/card transition | §3.3 write rule; format is normative (§3.2) |
| ⬖ `docs/05-implementation/V-COMMON.md` | invariant verification block shared by all MXX-V1 cards | CE, once | **immutable cache** — written at V2 materialization, never regenerated | edit = §13 amendment (it encodes frozen DoD/E1 items only) |
| ★ `MXX/cards/MXX-*.md` | persisted Execution Cards | CE authors | DRAFT → READY → immutable; revisions are new files | §4.2 |
| ⬖ `MXX/IMPLEMENTATION_SPEC.md`, `MXX/AI_EXECUTION_CONTEXT.md` | IKB §2 contract **satisfied by reference** from M07 on: SPEC = card index (objectives + scope-wall summary + card→TRACEABILITY rows); AI_EXEC = model row + dispatch order + milestone escalation deltas; each ≤15 lines pointing at `cards/` | CE at Phase A | per milestone | §15 D1 — 7-file count and CI lint unchanged; the full transcription layer they used to carry lives once, in the cards |
| ● `.claude/agents/awis-*.md` | L0 identities | human on CE proposal | frozen for baseline | AEO §22 only |
| ● milestone module 7-files | milestone ground truth | per IKB §13 write triggers | per milestone | IKB §13 |
| ● `docs/00-foundation/README.md`, `docs/07-indices/*` | session recipe; lookup surface | CE (re-pointing only) | stable | IKB §13 index hygiene |

Hierarchy rule (one line, total order): **Tier 0 → Tier 1 → IKB → AEO → EEOS → cards → code comments.** A card citing a coordinate never restates it; a session citing a card never re-derives it.

---

## 3. REPOSITORY STATE TRACKING — THE STATE LEDGER (Parts 2, 13)

### 3.1 Why a single ledger
Fresh-session determinism requires that *one file at one fixed path* answers: what milestone, what phase, which cards in what status, what's blocked, what's next. Today those answers live in conversation history (fails on session death — proven twice) or must be re-derived from `git log` + module diffs (~10–20k tokens of Fable-grade archaeology per fresh session). The ledger reduces bootstrap to one ~40-line read.

### 3.2 Normative format (machine-checkable, line-oriented)

```markdown
# EEOS STATE LEDGER — single source of live execution state
# Write rule: no phase or card transition is real until it is written here.

MILESTONE: M07-signal-subsystem
BRANCH: m07-signal-subsystem
PHASE: B-BUILD                      # A-INIT | B-BUILD | C-VERIFY | D-CLOSE | E-MERGE | IDLE
GATE: G2 (pending)                  # none | Gx (pending|passed YYYY-MM-DD)
CARDS:                              # status lives HERE and only here; card files are immutable
  M07-C1  DONE        (commit 3f2a1c9)
  M07-C2  DISPATCHED  awis-core-engineer
  M07-C3  READY
  M07-V1  DRAFT
BLOCKERS: none                      # or: E1 on C2 — <one line>; E2 CONTRA-n drafted
NEXT: on C2 DONE → dispatch C3; then materialize V1
LAST: 2026-07-07 CE dispatched C2
DONE-MILESTONES: M00 M01 M02 M03 M04 M05 M06   # history beyond this: git log + HANDOFFs
```

### 3.3 Write rule (the determinism invariant)
**No phase transition, card dispatch, card completion, STOP, or blocker is real until written to STATE.md** — the transition and the write are one act. Consequences: (a) session death at any instant loses at most the in-flight agent invocation, never the plan; (b) any fresh session resumes from S1–S3 (§5) with zero reconstruction; (c) `STATE.md` diffs in the milestone PR give the human a free execution audit trail. Status lives *only* in STATE (cards stay immutable; durable outcomes go to `TRACEABILITY.md` per IKB §13 — three files, three non-overlapping jobs: contract / live status / durable record).

---

## 4. EXECUTION CARD SPECIFICATION (Parts 4, 6)

### 4.1 The card is a file
The Execution Card is AEO §19's task card, **unchanged in fields**, persisted at `docs/05-implementation/MXX-<slug>/cards/MXX-Cn.md`. The template, field semantics, size cap (≤1,200 tokens incl. excerpts), verbatim-excerpt rule, scope walls, and binary AC discipline are AEO §19 by reference — not restated here (no-duplication rule). The EEOS adds exactly two header lines:

```markdown
# EXECUTION CARD M07-C2
STATUS-AUTHORITY: docs/05-implementation/STATE.md      # this file never carries status
DISPATCH: awis-core-engineer                            # AEO §6 assignment, pre-resolved
<then the AEO §19 body verbatim: AGENT/BRANCH/OBJECTIVE/INPUTS/OUTPUTS/
 SCOPE WALLS/ACCEPTANCE/CONSTRAINTS DELTA/CONTEXT BUDGET/REPORT>
```

Verification cards (`MXX-V1.md`) are the AEO §12 verification card persisted the same way: checklist transcribed from `VALIDATION_CHECKLIST.md`, §20 checkpoint procedure, exact commands. Same immutability, same revision naming (`V1r` — already M06 practice).

### 4.2 Card lifecycle
`DRAFT` (CE writing, may edit freely) → `READY` (complete per template; **file frozen**) → `DISPATCHED` → `DONE | STOPPED | WITHDRAWN`. A defective or STOPped card is never edited — CE cuts a **revision card** (`MXX-Cnr.md`, `Cnr2.md`, …), which states in one line what changed and why, and cites the original. This preserves the M04/M06 continuation-card pattern but makes it cheap: the revision is a delta against a file on disk, not a reconstruction from a dying session's memory. The AEO §22 F1 failure ladder (two rounds → withdraw → re-cut) operates on these files unchanged.

### 4.3 Dispatch (the mechanism that makes Fable dormancy real)
Dispatch is one frozen prompt, identical for every card, every agent, every milestone — stored in `EEOS.md` and requiring zero judgment to issue:

```
Execute the Execution Card at <card path>.
Read that file first. Then read ONLY the files it names, within its stated
context budget. Honor every constraint in your agent identity. Work only on
the branch the card names. Return your report per your identity's contract.
Commit exactly once, when every card output is complete and its tests are
green; commit message = the card id + one line.
```

⬖ **Card-primary context rule (V2, normative):** the files a card names never include the milestone module — not the spec, not the checklist, not the module README. A worker's context is *exactly one card* plus the code/TDS/EDR files that card names. (M06 practice — "context per card: this module's 7 files + …", ~7.5k tokens of module tax per card — is retired; everything a worker needed from those files now arrives inside the card, which is what AEO §19's exhaustive-inputs property was for.)

⬖ **Salvage preamble (V2, frozen text, appended only when re-dispatching a card STATE already shows `DISPATCHED`):**

```
A prior run of this card was interrupted. Uncommitted work may exist on the
branch (git status / git diff). Before writing anything: audit what you find
against this card's OUTPUTS and ACCEPTANCE. Keep files that pass, complete or
rewrite the rest. List salvaged-vs-rewritten in your report's DEVIATIONS.
```

This makes interruption recovery Fable-free and authoring-free: the working tree *is* the partial-implementation state, the card *is* the continuation card, and the audit is card-scoped (Sonnet/Opus-grade per the card's own DISPATCH row, never architecture reasoning). The per-card single-commit rule (above) makes `DONE ⇔ commit sha` exact, so salvage scope is always precisely "the uncommitted diff". Authored continuation cards (M04-C2r, M06-C1r/C2r/C2r2/V1r pattern) are retired for interruptions; revision files remain only for *defective* cards (AEO §22 F1).

Because the card carries 100% of task-specific content (AEO §19's exhaustive-inputs property) and the identity carries 100% of permanent constraints, the dispatching session contributes nothing but the pointer. Therefore *any* session — Fable, or a cheaper model the founder chooses for Phase B/C sitting — dispatches identically. This is the load-bearing property: it converts orchestration from reasoning into clerical work.

### 4.4 Minimality audit of fields
Every AEO §19 field already pays rent (exhaustive inputs kill corpus-browsing; pasted excerpts kill paraphrase drift; binary AC kills "done-ish"; scope walls kill drift; stated budget makes overage diagnosable). The EEOS adds only `DISPATCH` (pre-resolving AEO §6 routing so dispatch needs no matrix lookup) and the status-authority pointer (prevents dual-source status). Fields considered and rejected: priority (order is `NEXT:` in STATE), estimates (IMP §29 owns effort), inter-card dependency graph (2–5 cards per milestone — `NEXT:` in STATE is sufficient; a DAG field is speculative infrastructure).

---

## 5. FRESH CHAT PROTOCOL (Parts 3, 5, 12)

Target interaction, verbatim: **Human: "Load EEOS and execute M07."** Anything more required is an EEOS defect.

### 5.1 Bootstrap sequence (content of `/EEOS.md`, normative)

```
S1  Read docs/05-implementation/STATE.md            (~0.5k tokens)
S2  Read docs/00-foundation/README.md               (<8k — recipe + contradiction protocol)
S3  Act per the phase table below. Load NOTHING else until the row says so.
```

### 5.2 Phase-resume table (deterministic: STATE row → files → action)

| STATE says | Additionally load | Action |
|---|---|---|
| `IDLE` / requested MXX not entered | MXX `README.md` + IKB §3/§4 row | Run Phase A-INIT (§6) — requires a Fable session; if this session is not Fable-grade, say so and stop |
| `A-INIT` incomplete | partial module files | Resume materialization at the first missing IKB §3 step; then cut remaining cards |
| `B-BUILD` | the one card `NEXT:` names | Dispatch it (§4.3 prompt). On report: write STATE, then next card |
| `B-BUILD` + card `DISPATCHED` | — | The invocation died with the session. Re-dispatch the same card file **with the §4.3 salvage preamble** ⬖ — no Fable, no authored continuation card (one clean re-dispatch; a second death → treat as STOPPED, CE audits disk per AEO §22 F1) |
| `B-BUILD` + `BLOCKERS: E1…` | the STOP report line + cited coordinates | Fable wake: adjudicate per AEO §10 (07-indices lookup or revision card); E2 → CONTRA draft → human |
| `C-VERIFY` | `MXX-V1*.md` | Dispatch verifier on clean tree; ❌ → revision cards → back to B |
| `D-CLOSE` | full milestone diff + module | Fable wake: semantic review (AEO §11 rung 3), evidence assembly, PR, gate brief if gated |
| `E-MERGE` | — | Blocked on human. Report what awaits the founder; do nothing |
| Ledger contradicts the repo (e.g. card DONE but tests red at HEAD) | — | STOP. That is repository inconsistency — Fable wake trigger; never "fix" STATE to match a guess |

### 5.3 Session-death recovery (Part 12)
There is no separate recovery protocol — **recovery IS the bootstrap**, because of the §3.3 write rule. The M06 failure replayed under EEOS: agent dies mid-C2 → fresh session, S1 reads `C2 DISPATCHED` → re-dispatch `cards/M07-C2.md` verbatim → zero reconstruction, zero CE archaeology, zero Fable tokens. (Under the old protocol this cost a Fable disk-audit plus a from-memory continuation card, four times in M06 alone.)

### 5.4 Worked example: "Load EEOS and execute M07" today
S1 → STATE shows `M06 … E-MERGE, BLOCKERS: awaiting founder merge verdict`. The session reports: "M06 awaits your merge verdict; M07 is on the critical path behind it (dependency-index). Approve and merge M06, then re-issue this command — I will run M07 Phase A-INIT." One prompt in, one decision-shaped sentence out. That is the whole point.

---

## 6. IMPLEMENTATION LIFECYCLE (Parts 6, 7, 9, 10, 11, 14)

The AEO §17 EEP's nine steps, unchanged in content, bound into **five phases with explicit session boundaries and wake points**. Phases, not steps, are the unit of session planning and STATE tracking.

```
PHASE A — INIT        [FABLE WAKE #1 — the expensive 30 minutes]        (EEP-1, EEP-2)
  branch mXX-<slug> · IKB §3 materialization (scribe transcription assist)
  · cut ALL cards for the milestone incl. MXX-V1 · write cards/ · STATE → B-BUILD
  Fable-grade because: the one legitimate heavy-source load (~10–20k) and the one
  judgment-dense act (card cutting) happen here, once, and persist to disk.

PHASE B — BUILD       [FABLE DORMANT — file-driven dispatch]            (EEP-3)
  per NEXT: dispatch card (§4.3) → receive capped report → STATE write → repeat.
  Same-package cards serialized, independent cards parallel (AEO §9.4).
  Per-card acceptance = report well-formed + card's tests green (mechanical).
  E1 STOP → the only mid-phase Fable wake (adjudicate; revision card; sleep).

PHASE C — VERIFY      [FABLE DORMANT]                                   (EEP-4)
  dispatch MXX-V1 to awis-verifier on clean tree (independence per AEO §12 —
  its context is the card + repo, never reports). All-✅ → STATE → D.
  Any ❌ → revision card(s) → back to B. Gated milestones: scribe drift audit here.

PHASE D — CLOSE       [FABLE WAKE #2]                                   (EEP-5..7, EEP-9)
  CE semantic review of the FULL milestone diff, once (see consolidation note
  below) · findings → revision cards → B (bounded, AEO §22 F1) · scribe drafts
  HANDOFF actuals + checklist ticks, CE approves (final pre-merge commit) ·
  assemble PR: diff + §20 checkpoint transcript + Verification Report +
  DEVIATIONS + STATE diff · gated milestones: Gate Brief per IMP §23.

PHASE E — MERGE       [HUMAN]                                           (EEP-8)
  founder reviews PR (≤~1,500 lines by construction) · gate verdict where due ·
  squash-merge · CE confirms CI green on main · STATE → next milestone / IDLE.
```

⬖ **Boundary wake fusion (V2):** at non-gated milestone boundaries, Phase D of M(n) and Phase A of M(n+1) execute in **one Fable sitting** — everything Phase A needs exists by the end of Phase D (frozen sources; predecessor HANDOFF actuals, which are drafted pre-merge per EEP-9; the IKB §4 row). M(n+1)'s cards are written READY and its STATE block staged; Phase E's merge then flips STATE and cuts the branch. Gated boundaries (M07→M08 behind G2, M15→M16 behind G3) keep Phase A as a separate post-verdict wake, because those cards would otherwise encode assumptions the gate exists to test (IKB §2 rationale — same reason full pre-generation is prohibited, §15 Q3). TDS-day-1 milestones (M11/M12/M14/M15) need no special case: C1 *is* the TDS-authoring card, and later cards name the TDS file as an input path — it exists by their dispatch time, and naming a file is not fabricating an excerpt. Net: Fable sittings drop from 2 per milestone to ~1 per boundary (≈8 of 11 boundaries fusible).

**Review consolidation (supersession, §14.4):** AEO §11 rung 3 executes **once per milestone at Phase D over the full diff** by default, instead of per-card. Every line is still Fable-read before merge; what is eliminated is N−1 warm-up re-reads of interleaving context across N cards. CE retains discretion to review per-card inside Opus-row milestones (M07's atomicity code warrants it) — discretion to add rigor, never to skip it. Rungs 1, 2, 4 are unchanged, and nothing reaches rung 4 with known lower-rung failures.

**Transition optimizations (every arrow in the brief's flow):** Human→Fable: one sentence, resolved via STATE (§5). Fable→Card: persisted once, reused for revisions. Card→agent: one frozen prompt (§4.3). Haiku scout: **eliminated as a standing step** — cards carry exhaustive inputs, so routine scouting is unnecessary; when CE genuinely needs a lookup it uses `07-indices` (1–3k) or the built-in Explore agent (AEO §3), which pays no permanent-role rent. Sonnet→report: capped schemas (AEO §8), unchanged. Report→Fable approval: deferred and batched to Phase D. Fable→merge: Gate Brief ≤1 page, unchanged. Failure recovery (Part 11): AEO §22 F1–F7 unchanged, now operating over files — plus the F-series gains F8: *ledger/repo contradiction → Fable wake, never silent repair* (§5.2 last row).

**Human approval gates (Part 14):** exactly the frozen set — G1..G4, TDS-06 sign-off, every merge (AEO §13.4). The EEOS adds zero human touchpoints and removes zero. **Responsibility boundaries (Part 15):** AEO §5's uniqueness invariant unchanged; the EEOS introduces no new responsibility except STATE writes (CE, exclusively — a second writer would destroy the single-source property).

---

## 7. AGENT DEFINITIONS (Part 3)

The brief's hypothesis (Fable/Sonnet/Haiku/Opus as four agents) resolves to the **existing AEO organization**: the models are not the agents; the agents are role-shaped bindings of models to boundaries. Challenged and settled as follows — remove-first, per the brief:

- **Remove `awis-verifier`?** No. Independence is a boundary, not a persona — one identity cannot enforce implementer/verifier separation, and its edit-incapable tool surface is mechanical (AEO §25). Removal re-opens self-review blindness to save one 43-line file.
- **Remove `awis-scribe`?** No. IMP §28's Haiku rows are frozen; without the identity, that work runs at Sonnet cost (~10× on M13/M17 + every drift audit + every HANDOFF draft).
- **Remove `awis-core-engineer`?** No. IMP §28 mandates Opus on the correctness cliffs; frontmatter model-pinning is the only *mechanical* enforcement of §28 that exists.
- **Remove `awis-builder`?** No. ~60% of milestone-days at Sonnet cost under pinned tools; folding into the main session re-attaches full-session context to every implementation hour.
- **Add a Haiku "repository scout"?** Rejected — fails the rent test (built-in Explore + 07-indices already cover it; AEO §25 rejected researcher/explorer on identical grounds).
- **Add anything else?** No. AEO §25's rejection list stands unamended.

Compact definitions in the brief's exact format (full normative identities: AEO §18, frozen; these summaries add nothing to them):

| | Mission | Responsibilities | Authority | Inputs | Outputs | Stop conditions |
|---|---|---|---|---|---|---|
| **Founder** (human) | own irreversibility | gate verdicts; every merge; E2/E3 answers; EEOS/AEO amendments | absolute at gates/merges | Gate Briefs, PRs | verdicts, merges | — |
| **CE** (Fable, main session, dormant by default) | convert frozen plan to merged milestones at minimum judgment cost | Phase A materialization + card cutting; E1/E2 adjudication; Phase D review + evidence; STATE writes | delegation, routing, upward-only substitution; **never** production code, merges, or corpus browsing | STATE, module files, diffs, STOP reports | cards, STATE, reviews, PRs, Gate Briefs, CONTRA drafts | wake list §8 exhausted → sleep |
| **`awis-builder`** (Sonnet) | faithful implementation of well-specified cards | Sonnet-row milestones: code + tests | E0 choices inside card scope | its card + files the card names | code, tests, capped report | AEO §18.1 triggers (ambiguity, conflict, dependency, frozen-interface) |
| **`awis-core-engineer`** (Opus) | correctness cliffs (IMP §28 Opus rows — the standing escalation criteria) | M07 atomicity next; FSMs; append/rebuild paths; adversarial passes | E0 within card; findings-only in review cards | its card + named files | code, exhaustive state-space tests, report | §18.2 triggers incl. E3-grade frozen-format suspicion |
| **`awis-verifier`** (Sonnet) | independent truth on a clean tree | execute MXX-V1 exactly; binary table | none — evidence only, non-negotiable | verification card + repo at HEAD (never reports) | VERDICT + table + failures | ambiguous item; unreachable state; out-of-scope defect |
| **`awis-scribe`** (Haiku) | mechanical breadth at minimum cost | transcription assist; HANDOFF drafts; ticks; drift audits; §28 Haiku batches | none beyond exemplar substitution | batch card + exemplar | items + per-item status | item needs judgment; >20% flagged |

---

## 8. LAZY CTO PROTOCOL — FABLE WAKE RULES (Part 6, brief §lazy_cto_policy)

Fable wakes for **exactly** (each maps to a brief-mandated trigger): **W1** milestone INIT incl. card generation (Phase A) · **W2** E1 a card cannot absorb / architecture contradiction / repository inconsistency incl. ledger-repo conflict · **W3** implementation deadlock (STOP storm → AEO §22 F2 re-materialization check) · **W4** milestone CLOSE: semantic review, evidence, PR (Phase D) · **W5** human approval gates: Gate Brief authoring; merge recommendation · **W6** frozen-document conflict (CONTRA drafting, E2/E3).

Fable never: writes production code (AEO §4.1); re-inspects repository files outside W1/W4 material; re-performs rung-1/rung-2 checks; reloads corpus context (STATE + module files reconstruct any wake in <8k, per IKB §10); performs repository search (07-indices lookup or built-in Explore instead); generates documentation no write-trigger demands. **Every Fable turn outside W1–W6 is logged as an EEOS defect** (one line in STATE `LAST:`; recurring defects are §13 amendment evidence).

Mechanical note (honest, since the harness pins the main-session model per session, not per turn): dormancy is realized two ways, both EEOS-conformant — (a) a Fable main session that simply *does nothing Fable-grade* during B/C (dispatch is clerical; §4.3), or (b) the founder runs B/C sittings in a cheaper-model session — safe because dispatch is file-driven and model-agnostic, and any W-trigger makes that session stop and name the needed wake. The EEOS makes (b) *possible* and (a) *cheap*; it mandates neither, because session choice is the founder's lever.

---

## 9. CONTEXT ARCHITECTURE (Parts 3, 8, 16, 17)

**Loading (Part 3):** unchanged from AEO §14/IKB §11, with the bootstrap prepended: every session = `EEOS.md` → `STATE.md` → 00-foundation → phase-row files. Ceiling: worker ≤ card budget (AEO §15); Fable wake ≤ ~15k (STATE + module + diff); Phase A ≤ ~25k (the one sanctioned heavy load). **Inheritance (Part 8):** sessions inherit nothing from sessions — they inherit from files. The inheritance chain is exactly: STATE (live) → cards (contract) → module 7-files (milestone truth) → TRACEABILITY/HANDOFF (durable outcomes) → indices (frozen design). A fact outside that chain does not exist (AEO §2.3, now with no conversational escape hatch). **Compaction (Part 16):** compaction is *writing, not summarizing* — a session nearing limits writes STATE + any pending TRACEABILITY lines and dies; the successor bootstraps in <8k. Summarizing normative content remains forbidden (paraphrase drift, IKB §9). **Prompt elimination (Part 17):** the only recurring human prompt left is `Load EEOS and execute <MXX>` plus gate/merge verdicts — everything formerly prompted (process explanation, state recap, card re-issuance, "what's next") is now a file read. Each remaining prompt maps 1:1 to a human-only authority (initiation, irreversibility), which is the theoretical floor.

---

## 10. VERIFICATION & MERGE GOVERNANCE (Parts 9, 10)

Unchanged in substance — restated only as bindings: verification is Phase C, driven by the persisted `MXX-V1` card, executed by `awis-verifier` in an implementation-reasoning-free context (AEO §12); AWIS-E1 on every card from M06 forward, and a V-card omitting it is itself a ❌. Merge is Phase E, human-only, preconditions AEO §13.3 verbatim, with one addition: **the PR must contain the STATE diff and the `cards/` files** — the founder sees not just what changed but the exact contracts under which it changed. Post-merge defect: revert-first (IMP §26), STATE re-opened to the reverted milestone, re-entry at Phase C.

---

## 11. TOKEN OPTIMIZATION REPORT (Part 7)

Baseline = M04–M06 observed practice (resident Fable orchestration; prompt-embedded cards; state in conversation). All figures are estimates from IKB §10 measured session costs and M06's recorded incident count; marked ~.

| Waste class (observed) | Mechanism | Est. saving |
|---|---|---|
| Session-death reconstruction: 4 continuation cards in M06 alone, each = Fable disk audit + card rewrite (~10–15k Fable tokens each) | §3.3 write rule + persisted cards → re-dispatch is a file pointer | ~40–60k **Fable** tokens per affected milestone; M04/M06 base rate says most milestones are affected |
| Fresh-session state archaeology (`git log` + module diffing, ~10–20k, Fable-grade, every new sitting) | S1 bootstrap: ~8k, mostly cached, any model | ~5–12k per sitting × ~2–4 sittings per milestone |
| Process re-explanation in prompts (human re-states workflow, model re-derives it) | `/EEOS.md` read once per session | ~2–5k per session + founder attention |
| Per-card semantic-review warm-up: context re-established N times per milestone | Phase D consolidation (§6) | ~(N−1)×5–10k Fable per milestone, N=2–5 cards |
| Orchestration turns at Fable pricing during BUILD/VERIFY | dispatch made clerical (§4.3) → dormancy option (a)/(b) | not token count but **token price**: the milestone's largest-turn-count phase becomes eligible for the cheapest model |
| Card re-issuance after withdrawal (full re-write from memory) | revision-card delta against file | ~3–8k per F1 event |
| ⬖ Worker module tax: M06 loaded the full 7-file module (~7.5k measured) into every card context | §4.3 card-primary context rule | ~7.5k × N cards ≈ 15–35k **worker** tokens per milestone |
| ⬖ ag-kit rules chain: CLAUDE.md mandates ~21KB (~5k tokens) of generic rule files + routing/announcement overhead into **every** session, none of it load-bearing for frozen-plan execution | §15 D4 CLAUDE.md override: EEOS sessions load `/EEOS.md` as their P0 and skip the ag-kit chain | ~5k+ × every session ≈ 25–40k per milestone; ≈0.3–0.5M lifetime — the largest single V2 lever |
| ⬖ Double transcription: IMP §27 row → full IMPLEMENTATION_SPEC.md → cards restating the same cuts | §15 D1: Phase A compiles cards directly from IKB §3/§4 sources; SPEC/AI_EXEC become ≤15-line indices | ~3–6k authored + re-read per milestone |
| ⬖ Two Fable sittings per milestone (close, then next init, each paying bootstrap + uncached context) | §6 boundary wake fusion (non-gated boundaries) | ~8–12k Fable per fused boundary × ~8 boundaries |
| ⬖ Interruption recovery even under V1 still cost a Fable adjudication turn | §4.3 salvage preamble: recovery is a re-dispatch, zero Fable turns, zero authored text | ~3–5k Fable per incident (on top of row 1) |
| ⬖ V-card boilerplate re-transcribed per milestone (and omissible — the verifier must police E1 presence) | `V-COMMON.md` immutable cache; V-cards carry only milestone-specific checkpoints | ~0.5k × 12 + removes a defect class |

Structural savings already banked by AEO/IKB (no corpus in workers; capped reports; Haiku routing; index lookups) are unchanged and not re-counted. **Net estimate — V1 mechanisms: ~60–100k per milestone (dominated by the two M06-proven classes, rows 1–2, eliminated structurally). V2 pass adds ~45–80k per milestone (dominated by the ag-kit bypass, module-tax removal, and wake fusion). Combined: ≈1.1–1.9M tokens across M07–M18, with Fable-priced tokens falling ~50–70% per milestone against M04–M06 practice.** Cost of the EEOS itself: ~3–5k once (this spec + bootstrap + V-COMMON), ~0.5k per milestone (STATE writes + stub indices), zero new agents, zero new reviews.

---

## 12. FINAL RECOMMENDATION (Part 8)

Adopt the EEOS as specified: **the AEO organization, unchanged, plus persisted cards, one ledger, one bootstrap file, and wake discipline.**

- **Why each permanent agent exists / each removal was rejected:** §7 — every retention is a mechanical boundary (model pin, tool pin, independence) or frozen Tier-1 allocation; every rejected addition fails the AEO §25 rent test, including the brief's own Haiku-scout hypothesis.
- **Why each protocol file exists:** §2 — each answers exactly one question no other file answers (entry point / live state / task contract), and each eliminates a prompt class or a proven failure class. Nothing in the EEOS is speculative: every mechanism traces to a recorded M04/M06 incident or an IKB-measured cost.
- **Why it should remain unchanged until Baseline V1:** it contains zero milestone-specific content (all variability rides in cards and STATE, which are *data* under the protocol, not protocol); its three files are load-bearing for every future session, so churn would invalidate cached bootstraps and retrain nothing; and its amendment path (§13) already covers the only legitimate change drivers. A protocol that must be edited per milestone would be a milestone artifact, not an operating system.

**Approval & materialization (one-time, on founder verdict):** (1) commit this spec; (2) create `/EEOS.md` from §5.1/§5.2/§4.3 (the normative blocks incl. both frozen prompts, verbatim — target ≤120 lines); (3) create `STATE.md` seeded per §3.2 with `MILESTONE: M06 … PHASE: E-MERGE`; (4) extend the CI docs-lint per §1; ⬖ (5) create `V-COMMON.md` (§15 D5); ⬖ (6) append the CLAUDE.md EEOS-override paragraph (§15 D4, copy-ready); ⬖ (7) record the IKB §2 satisfaction-by-reference ruling in `docs/05-implementation/README.md` (§15 D1); (8) M07 Phase A-INIT writes the first `cards/` directory. Steps 2–7 are scribe-grade.

## 13. AMENDMENT POLICY
Human approval on CE proposal, only for: (a) a gate outcome demanding a process change; (b) a harness capability change breaking a load-bearing mechanism (file-driven dispatch, agent model-pinning); (c) two consecutive milestones breaching the same EEOS protocol despite compliant cards and ledger. Identical triggers to AEO §22 F7 — the two documents amend by the same law. **Founder directive (2026-07-08, binding):** every trigger must be backed by objective implementation evidence; speculative or optimization-motivated amendments are not authorized during Baseline V1.

## 14. EXPLICIT AEO SUPERSESSIONS (exhaustive)
1. **§16 L1** — cards are repo files (`cards/`), immutable once READY; revisions are new files. Everything else in §16 stands.
2. **§15 enforcement / §22 F3** — "state note into the milestone module" → STATE ledger writes per §3.3. F3 recovery = the §5 bootstrap.
3. **§17 EEP presentation** — nine steps bound into five phases with session boundaries and wake points (§6). Step content unchanged.
4. **§11 rung 3 cadence** — consolidated to once per milestone at Phase D by default; CE discretion to add per-card depth on Opus rows; rigor floor unchanged.

Everything else in the AEO — roles, matrices, escalation, verification, merge, budgets, report schemas, identities — stands verbatim.

---

## 15. EEOS V2 — OPTIMIZATION PASS (founder-directed, 2026-07-08)

Charter: minimize implementation tokens further; no redesign. Six questions investigated against measured repository facts (M06 module = 30,253 bytes ≈ 7.5k tokens; ag-kit rules chain = 21,364 bytes ≈ 5k tokens; M06 executed 3 implementation cards + 1 V-card with 4 continuation reconstructions). Verdicts:

**Q1 — Cards as the primary implementation artifact? YES (delta D1).** For workers this was always the design intent; M06 practice violated it (its `AI_EXECUTION_CONTEXT.md` instructed "context per card: this module's 7 files + …"). V2 makes it normative: **implementation proceeds by loading exactly one card** plus the code/TDS/EDR files that card names — never the milestone module (§4.3 card-primary rule). Consequently the module's SPEC and AI_EXEC files, whose transcription content now lives once in the cards, satisfy their IKB §2 contract *by reference* (≤15-line indices; §2 table). The 7-file contract, CI lint, and the durable-record files (README, VALIDATION_CHECKLIST, HANDOFF, TRACEABILITY, DEPENDENCY_MAP) are unchanged — cards are execution contracts, not history; HANDOFF/TRACEABILITY remain the successor-facing record. Requires one founder-approved interpretation ruling recorded in `docs/05-implementation/README.md`: *"an IKB §2 file may satisfy its contract by reference to `cards/`."*

**Q2 — Engineering state in STATE.md? NO — git already is the engineering state (delta D2).** Duplicating file-level progress into STATE would create a second source of truth that dies exactly when it's needed (the interrupted session couldn't have written it). The minimum deterministic resume state, exhaustively, for all four cases (model limit / context reset / interrupted session / partial implementation): **(a)** STATE.md protocol lines (milestone, phase, card statuses, NEXT), **(b)** the branch name, **(c)** DONE-card commit shas (guaranteed by the §4.3 one-commit-per-card rule: `DONE ⇔ sha`), **(d)** the uncommitted working tree (= the partial implementation, exactly). Recovery for all four cases is the identical procedure: bootstrap §5.1 → re-dispatch the DISPATCHED card with the salvage preamble. Nothing is authored, nothing is reconstructed, no Fable turn occurs. STATE stays protocol-only.

**Q3 — Fable gone entirely during execution? Between boundary wakes, YES; pre-generating all M07–M18 cards, NO.** Full pre-generation fabricates inputs: M08+ cards depend on the G2 verdict, M16+ on G3, and M11/M12/M14/M15 cards must cite TDS-04/05/07/06 — documents written *inside* those milestones (IMP §12 "written day 1"); predecessor HANDOFF actuals feed successor cards (AEO OR-5). Pre-writing them freezes assumptions the gates exist to test — the exact drift IKB §2's binding rationale prohibits. It would also save ~zero tokens (the same cards get authored either way; only *when* changes). What V2 does instead: (a) **boundary wake fusion** (§6) — one Fable sitting per non-gated boundary covers close(n) + init(n+1), cutting Fable sittings ~2/milestone → ~1/boundary; (b) **Fable-free recovery** (Q2) removes the residual mid-phase wakes. The founder's target flow — STATE → Card → agent → Verification → Merge — is now literally the complete flow between boundary wakes, with Fable appearing only at the boundary sitting (which ends in the merge-approval package) and at E1/E2 escalations, which are the frozen escalation ladder and not removable by protocol design.

**Q4 — Remaining waste, hunted (delta D4).** Found and eliminated: the **ag-kit rules chain** — CLAUDE.md compels ~5k tokens of generic rule files plus routing/announcement overhead into every session; none of it is load-bearing for executing a frozen plan under EEOS (it predates the AWIS corpus and partially contradicts it, e.g. routing to non-AEO agents). Fix, copy-ready for CLAUDE.md: *"**AWIS EEOS override:** in any session whose task references EEOS, a milestone (MXX), or Execution Cards, `/EEOS.md` is the P0 protocol; skip the `.agents/` loading protocol, request routing, and skill announcements entirely."* Also eliminated: worker module tax (Q1); double transcription (D1); duplicate Fable bootstraps (Q3). Checked and already clean: repeated repository scanning (dispatch prompt forbids it; Explore/07-indices for the rare lookup), repeated architecture loading (excerpts ride in cards; the only duplication is the IKB-sanctioned execution-safety kind), repeated implementation planning (cards persist; plans are never re-derived). `docs/00-foundation/README.md` measured 1.2KB — too small to be worth phase-scoping; it stays in the universal bootstrap.

**Q5 — Immutable caches? YES, two (delta D5), plus a stability rule.** (a) `docs/05-implementation/V-COMMON.md`: the invariant verification block (clean-tree procedure, `make build test lint race`, suite layers per IMP §19, AWIS-E1) — written once, cited by every `MXX-V1`, never regenerated; V-cards shrink to milestone-specific checkpoint items, and E1 omission (a defect class the verifier currently polices per AEO §18.3.4) becomes structurally impossible. (b) The two frozen prompts (dispatch, salvage) in `/EEOS.md`. Cache-stability rule: `/EEOS.md`, the four identities, `V-COMMON.md`, and READY cards are **byte-frozen during a milestone** and the bootstrap reads them in fixed order — maximizing provider prompt-cache hits across the hundreds of invocations they front. Everything else examined either already never regenerates (07-indices, canonical corpus, HANDOFFs) or is legitimately live (STATE).

**Q6 — Protocol removals (delta D6).** Removed in V2, each with the token test applied:

| Removed | Was costing | Replaced by |
|---|---|---|
| Authored continuation cards (interruption path) | ~10–15k Fable per incident, 5 incidents in M04–M06 | salvage preamble (frozen text, 0 authored tokens) |
| Full `IMPLEMENTATION_SPEC.md` + `AI_EXECUTION_CONTEXT.md` transcriptions | ~3–6k authored + re-read per milestone | ≤15-line indices over `cards/` |
| Second Fable sitting per non-gated boundary | ~8–12k per boundary | wake fusion |
| ag-kit loading protocol in EEOS sessions | ~5k+ per session | one CLAUDE.md override paragraph |
| Per-milestone V-card boilerplate | ~0.5k × 12 + defect surface | `V-COMMON.md` |

Audited and **kept** — each saves more than it costs: STATE ledger (~0.5k/milestone vs ~10–20k fresh-session archaeology, M06-proven); per-card acceptance check in Phase B (one report read vs a defective card compounding into Phase C failures); verifier independence (AEO §25 — not a token protocol; removal re-opens false-PASS); scribe drift audit (gates only, Haiku-priced, guards the verbatim chain everything else relies on); Gate Briefs (human-mandated, IMP §23); HANDOFF/TRACEABILITY (successor inputs and the durable record — deleting them re-creates the archaeology STATE was built to kill); the two-round clarification cap (it *is* a token limit). Nothing else remained to remove: every surviving protocol is either frozen above the EEOS or carries a measured rent entry.

**V2 delta index:** D1 card-primary context + satisfaction-by-reference (§2, §4.3) · D2 git-as-engineering-state + salvage preamble + one-commit-per-card (§4.3, §5.2) · D3 boundary wake fusion (§6) · D4 CLAUDE.md ag-kit override (§12.6) · D5 V-COMMON + cache-stability rule (§1, §2, §12.5) · D6 removals ledger (above). No agent, role, gate, review rung, or AEO matrix was touched.

---

**AWIS EEOS v2.0 — APPROVED AND FROZEN (founder, 2026-07-08). Materialized per §12. M07 executes under it.**
