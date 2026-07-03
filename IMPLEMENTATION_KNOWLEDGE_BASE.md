# AWIS IMPLEMENTATION KNOWLEDGE BASE
## AI-Native, Canonically Traceable, Context-Partitioned

**Generated:** 2026-07-02/03 · **Status:** GENERATED · **Tier:** 2 (living operational view)
**Immutable inputs:** OIP_CONSTITUTION.md · AWIS_ARCHITECTURE_BLUEPRINT.md · AWIS_ARCHITECTURE_FINALIZATION.md · AWIS_PRD.md (Tier 0) · IMPLEMENTATION_MASTER_PLAN.md + IMPLEMENTATION_MASTER_PLAN_VERIFICATION_REPORT.md (Tier 1, archival)

**Nature of this document:** deterministic **partitioning**, not compression. Nothing here is new design. Every statement either (a) points at a canonical coordinate or (b) transcribes one. Where this KB and a canonical document disagree, the canonical document wins and the KB entry is a defect.

---

## 1. DOCUMENTATION TREE

```
docs/
├── README.md                      root router: authority order + AI session loading protocol
├── 00-foundation/                 session recipe (<8k tokens), contradiction protocol
├── 01-vision/                     why AWIS/OIP exist (pointers into Constitution + PRD)
├── 02-architecture/               pointers into Blueprint + Finalization (blocker map)
├── 03-product/                    pointers into PRD (FR/NFR families, QG-1..5)
├── 04-planning/                   pointers into IMP + Verification (marked ARCHIVAL)
├── 05-implementation/             ★ the operational surface
│   ├── README.md                  materialization policy (see §2)
│   ├── M00-repository-bootstrap/  FULLY MATERIALIZED (7 files)
│   ├── M01-schema-format-freeze/  FULLY MATERIALIZED (7 files)
│   └── M02…M18 (17 dirs)          partitioned: README now, 6 files at entry (see §4)
├── 06-reference/                  TDS-01..07 mapping table + path resolution
└── 07-indices/                    canonical-reference-map · traceability-matrix
                                   · dependency-index · cross-reference-index
archive/
└── README.md                      ENGINEERING_ARCHITECTURE_BLUEPRINT.md → SUPERSEDED
                                   (Finalization Blocker 1); never cite
```

Root-level canonical documents stay at repo root, untouched. The tree contains **views**, never replacements.

## 2. MILESTONE PARTITION STRATEGY

One directory per milestone, M00–M18; names/scope/effort/dependencies fixed by IMP §27 as amended F-1..F-5. Each fully materialized module carries exactly seven files:

| File | Contract |
|---|---|
| `README.md` | Identity card: objective, window, deps, amendments, sources, key ACs |
| `IMPLEMENTATION_SPEC.md` | Transcribed build steps + scope walls + AC/DoD/Merge/RB |
| `AI_EXECUTION_CONTEXT.md` | Loading order, model allocation (IMP §28), hard constraints, escalation triggers |
| `VALIDATION_CHECKLIST.md` | Binary pass/fail exit list (incl. relevant global DoD items) |
| `HANDOFF.md` | Guaranteed outputs / assumptions for successors / limitations; **actuals filled at merge** |
| `TRACEABILITY.md` | Every task → canonical coordinate; requirements satisfied; contradictions touched |
| `DEPENDENCY_MAP.md` | Upstream consumed, downstream produced, critical-path position |

**Lazy materialization (binding policy, recorded in `docs/05-implementation/README.md`):**
- **M00, M01: fully materialized now.** Implementation begins there; their content depends on nothing unexecuted.
- **M02–M18: README-only now; the remaining six files are compiled at milestone entry** (~30 min, mechanical) by executing the §4 block against the immutable sources.

Rationale (verification-grade): (1) `HANDOFF.md` states truthful actuals — pre-writing week-5 handoffs fabricates them; (2) eager copies of post-G2/G3 content would freeze assumptions those gates exist to test — the exact drift this KB prevents; (3) the partitioning rules prohibit duplication not required for execution safety. A materialized module changes only via: executing its milestone, a gate outcome, or a recorded CONTRA entry.

## 3. DOCUMENTATION GENERATION SPECIFICATION

The deterministic procedure that produced M00/M01 and produces M02–M18 at entry. Given milestone MXX:

1. **Collect sources** exactly as listed in §4's row for MXX (IMP §27.MXX is always primary; add the DAG/critical-path context §9–§11, storage plan §14 if migrations, gates §23 if gated, risks §25 rows cited, model row §28).
2. **Apply amendments** F-1..F-5 where §4 marks them — amendments override the IMP text they amend; record the override inline ("per F-x").
3. **Write the six files** per the §2 contracts. Transcribe; never paraphrase normative language (field names, SQL, grammar text, gate questions are verbatim).
4. **Resolve nothing silently.** A gap or conflict in sources → CONTRA-style entry in the module's TRACEABILITY.md + escalation per `docs/00-foundation/README.md`.
5. **Verify**: every task in IMPLEMENTATION_SPEC.md has a TRACEABILITY.md row; every checklist item is binary; HANDOFF.md's guaranteed outputs match the IMP's "Out"/"Repo" fields; file set is exactly 7.

Determinism claim: two independent compilations from the same frozen sources differ only in prose style, never in tasks, ACs, sources, or constraints — because every load-bearing sentence has a coordinate.

## 4. GENERATED STRUCTURE PER MILESTONE (compilation specs)

Legend: **Src** = sources beyond the always-loaded IMP §27.MXX row · **Amd** = binding amendments · **Model** = IMP §28 allocation · **Gate** = human gate involvement.

| M | Directory | Src | Amd | Model | Gate |
|---|---|---|---|---|---|
| 00 | `M00-repository-bootstrap` | IMP §4, §5, §21; CONTRA-1 | — | Sonnet | — |
| 01 | `M01-schema-format-freeze` | Blueprint §6, §9 (verbatim); Finalization B2 (verbatim); IMP §12, §23, IR-1/IR-2 | **F-1** | **Opus + human** | **G1** |
| 02 | `M02-storage-foundation` | TDS-01/02; Blueprint §9, §20; IMP §14 | — | Sonnet + Opus review (append path) | — |
| 03 | `M03-state-projection` | TDS-01; Blueprint §9; IMP §14, §26 (`rebuild-state` deliberately early) | — | Sonnet + Opus review (rebuild fidelity) | — |
| 04 | `M04-intelligence-seam` | Blueprint §13–§17; CONTRA-3 | — | Sonnet | — |
| 05 | `M05-expression-engine` | TDS-03 + corpus (oracle); Finalization B2; IR-2 (fuzzing) | — | **Opus** | — |
| 06 | `M06-execution-engine` | Blueprint §5 L1/L2, §8; Finalization B4 (cancellation), B6 (compensation note), B7 (fan-out); Constitution Art. 32 (E1 live) | **F-2** (SCAN_TRIGGERABLE + DomainEvent ingestion + `domain_events` TTL 7d) | **Opus** | — |
| 07 | `M07-signal-subsystem` | Finalization B3 verbatim SQL; IR-3 (crash injection) | **F-4** (`audit_log` migration + append API here; SignalDelivered write site) | **Opus + human** | **G2** |
| 08 | `M08-sdk-public-surface` | PRD FR-SDK-*; F-1 alias surface from M01 | **F-2** (TriggerAPI intake) | Mixed (Opus API review) | — |
| 09 | `M09-test-infrastructure` | PRD FR-SDK-06/07/08, QG-5; Blueprint deterministic-mode text | — | Sonnet | — |
| 10 | `M10-yaml-dsl` | TDS-02 (equivalence oracle); PRD FR-WD-* | — | Sonnet | — |
| 11 | `M11-subprocess-runner` | **TDS-04 written day 1** (Blueprint §25); PRD FR-SE-02, FR-SDK-10 | — | Sonnet | — |
| 12 | `M12-plugin-system` | **TDS-05 written day 1** (Blueprint §11: JSON-RPC 2.0, lifecycle FSM, ≤3 restarts, idle kill); IMP §18 | F-4 write site (PluginRegistered) | Mixed (Opus FSM) | — |
| 13 | `M13-git-context-plugin` | TDS-05; Blueprint §11 reference-plugin text | — | Sonnet/Haiku | — |
| 14 | `M14-core-cli` | **TDS-07 written day 1 per F-3** (PRD §15–20, §26; Blueprint §28 socket vs SQLite-WAL + §9 refinement if latter; PID-file+SIGTERM); IR-5 | **F-3**, **F-5** (minimal `plugin install/list` moved here) | Sonnet | — |
| 15 | `M15-oip-on-awis` | **TDS-06 days 1–2, human sign-off before handlers** (Constitution Title II; Blueprint §10/§18); Finalization B5 (`oip.db` ownership), B6; CONTRA-3 (semantic-rank FTS pass-through); IR-4 contingency | — | Mixed + **human-only verdict** | **G3** + TDS-06 sign-off |
| 16 | `M16-anthropic-adapter` | Blueprint §14; PRD FR-IL-02/05..09; CONTRA-3 (no Embed) | — | Sonnet | — |
| 17 | `M17-full-cli-init` | TDS-07; PRD §15 full tree; IMP §14 (migration 0005 recall FTS); CONTRA-4 | **F-2** (cron trigger); F-5 note (plugin cmds already at M14) | Sonnet/Haiku batch | — |
| 18 | `M18-hardening-release` | PRD QG-1..5 + NFR benchmarks; CONTRA-2 (10ms target / 50ms gate); IMP §26 (release immutability) | F-6 advisory (week-5 float pre-start) | Mixed + human | **G4** |

At entry to MXX: load this row + the module README + the §3 procedure; compile the six files; only then begin implementation.

## 5. CROSS-REFERENCE INDEX
Canonical location: **`docs/07-indices/cross-reference-index.md`** — topic → coordinate for the twenty-one recurring lookup targets (grammars, atomicity, cancellation, FTS ownership, plugin/subprocess protocols, CLI↔runtime model, Record format, triggers, migrations, secrets, E1). Not duplicated here (no-duplication rule).

## 6. DEPENDENCY INDEX
Canonical location: **`docs/07-indices/dependency-index.md`** — full edge list, critical path (M00→M01→M02→M03→M06→M07→M08→M11→M12→M13→M15→M18, ≈28d), float carriers, gate stops, headline artifact flows. Per-milestone detail: each module's `DEPENDENCY_MAP.md`.

## 7. TRACEABILITY MATRIX
Canonical location: **`docs/07-indices/traceability-matrix.md`** — requirement family → milestone for every FR/NFR family and QG, with F-2/F-4/F-5 relocations marked and all four CONTRA dispositions listed. Per-task granularity: each module's `TRACEABILITY.md`.

## 8. CANONICAL REFERENCE MAP
Canonical location: **`docs/07-indices/canonical-reference-map.md`** — the tier table (Tier 0 frozen corpus → Tier 1 archival IMP/Verification → Tier 2 this KB → archive), mutability rules, and the upward conflict-resolution order.

## 9. INFORMATION PRESERVATION VERIFICATION

Claim: **zero information loss** relative to the planning corpus, achieved by reference, not by copy.

| Information class | Where preserved | Mechanism |
|---|---|---|
| All Tier-0 text | root canonical files, byte-identical | never edited; KB only points |
| IMP's 30 sections | IMPLEMENTATION_MASTER_PLAN.md (archival) | 04-planning marks it ARCHIVAL; §4 rows cite every §27 milestone; §5–§8 index the rest |
| Verification's 31 sections + F-1..F-5 | verification report (archival) | amendments folded into §4 rows and every affected milestone README |
| CONTRA-1..4 | IMP §25 | reproduced in traceability-matrix + affected milestone rows (execution-safety duplication, permitted) |
| Superseded blueprint | archive/ | supersession recorded, content retained |
| Gate questions/evidence | IMP §23 | transcribed verbatim into gated milestone modules (M01 done; M07/M15/M18 at materialization) |

Checked failure modes: (a) **paraphrase drift** — normative text is transcribed verbatim or pointed at, never restated; (b) **orphaned facts** — every §4 row names its sources, every module file cites coordinates; (c) **silent resolution** — the contradiction protocol is embedded in 00-foundation, §3 step 4, and every AI_EXECUTION_CONTEXT.md. No canonical sentence was deleted, weakened, or reinterpreted in generating this KB.

## 10. CONTEXT OPTIMIZATION ANALYSIS

The corpus totals ≫100k tokens; no implementation session should carry it.

| Session type | Loads | ≈ tokens |
|---|---|---|
| Any session start | 00-foundation/README.md | <8k |
| M00 execution | + M00 module | ~6k |
| M01 execution (worst legit case) | + M01 module + Blueprint §6/§9 + Finalization B2 verbatim | ~15–25k |
| M02–M18 entry (compilation step) | + §3/§4 + milestone README + listed sources | ~10–20k |
| M02–M18 execution | + the 7 compiled module files | ~8–15k |
| Lookup during any session | one 07-indices file → one coordinate | ~1–3k |

Optimization comes from **partition boundaries that match execution boundaries** (a milestone is the unit of loading, reviewing, merging, and reverting — IMP §26), not from summarizing. The only content ever duplicated is what a milestone cannot safely execute without (gate questions, amendment text, verbatim grammar/SQL) — each instance marked with its source coordinate so drift is mechanically detectable by diff.

## 11. AI LOADING STRATEGY

Protocol (also at `docs/README.md` and `docs/00-foundation/README.md`):

1. **Always first:** `docs/00-foundation/README.md` — authority order + contradiction protocol.
2. **Route by task:** implementing → `05-implementation/MXX/`; specification lookup → `07-indices/cross-reference-index.md`; "why" questions → 01/02/03 pointers; planning questions → 04-planning (read-only, archival).
3. **Model discipline (IMP §28):** Opus on M01/M05/M06/M07 semantics and API reviews; Sonnet on specified breadth; Haiku on M13/M17 mechanical batches; humans on G1–G4 + TDS-06. A session MUST check its milestone's `AI_EXECUTION_CONTEXT.md` model row before proceeding.
4. **Escalation:** every module lists its triggers; the universal one is any frozen-text ambiguity → stop, record, ask.
5. **Never:** load the whole IMP into an implementation session; cite the archived blueprint; resolve a contradiction inline.

## 12. REPOSITORY INTEGRATION STRATEGY

Current state: planning workspace, **not yet a git repository**. Integration at M00:

- M00's `git init` commits this KB **as-is** in the first commit: canonical docs at root, `docs/` tree, `archive/`.
- The IMP §5 repo structure references `docs/` for TDS files — TDS-01..07 land **inside this same tree** (`docs/EVENTLOG_FORMAT.md` etc.), making the KB's 06-reference pointers real files as milestones execute. No parallel doc hierarchies ever exist.
- `docs/edr/` (M00 deliverable) joins the tree beside the TDS pack.
- One milestone = one squash-merged PR (IMP §26); the milestone's module updates (checklist ticks, HANDOFF actuals) ride in that same PR — documentation and code are never out of sync on `main`.
- CI's docs job (IMP §21/§22) can lint: every `05-implementation` dir has README; materialized dirs have exactly 7 files; no file cites `archive/`.

## 13. LONG-TERM MAINTENANCE STRATEGY

- **Write triggers (exhaustive):** milestone execution (its module + HANDOFF actuals) · gate outcome (gated module + 04-planning gate log) · CONTRA entry (module TRACEABILITY + 07-indices) · milestone entry (materialize six files per §4).
- **Prohibited forever:** editing Tier-0/Tier-1 documents; "refreshing" archival planning text; deleting HANDOFF actuals; un-superseding the archive.
- **Post-v1.0.0:** 05-implementation becomes the historical execution record (HANDOFFs = what actually happened); V2 planning starts a NEW planning cycle and a new KB generation — this KB is never mutated into a V2 plan.
- **Index hygiene:** indices contain pointers only; a stale pointer is a bug fixed by re-pointing, never by editing the target.
- **Drift audit (cheap, mechanical):** diff every verbatim-marked block against its coordinate; verify 7-file contract on materialized dirs; verify no doc cites the archive. Suitable for a Haiku-grade batch check at each gate.

## 14. FINAL VALIDATION

| Requirement | Status |
|---|---|
| Deterministic partitioning, not compression | ✅ §3 procedure; verbatim-or-pointer rule throughout |
| Zero information loss | ✅ §9 (by reference; canonical files untouched) |
| Zero architectural drift | ✅ no design added anywhere; contradiction protocol embedded at every layer |
| Canonical traceability | ✅ 07-indices + per-module TRACEABILITY; every load-bearing sentence has a coordinate |
| docs/ tree 00–07 + archive | ✅ built (26 READMEs + 4 indices + 14 module files) |
| One dir per milestone, 7-file contract | ✅ M00/M01 materialized (7/7 each); M02–M18 partitioned with binding entry-compilation spec (§2 rationale, §4 rows) |
| Milestones independently executable without the whole IMP | ✅ §10 loading table; M00/M01 demonstrate the pattern |
| Cross-reference over copying | ✅ §5–§8 delegate to 07-indices; duplication only for execution safety, marked |
| Master document, 14 sections | ✅ this file |
| IMP remains immutable archival reference | ✅ Tier 1 in the reference map; 04-planning marked ARCHIVAL |

---

**IMPLEMENTATION KNOWLEDGE BASE GENERATED**

*"The Implementation Knowledge Base is AI-optimized, canonically traceable, context-partitioned, and ready for long-term AI-assisted implementation. The Implementation Master Plan remains the immutable archival reference."*
