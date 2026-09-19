# AWIS — OPEN DECISIONS REGISTER

Companion to `AWIS_RECONSTRUCTION_ANSWERS.md`. Takes the 28 items still considered
unresolved and sorts them into **ALREADY ANSWERED IN REPO** vs **GENUINELY OPEN**.
Compiled 2026-09-06 @ `engine-hardening` / `8a87f70` + uncommitted working tree.

Tags: **DECIDED** (recorded decision) · **DE-FACTO** (acted on, never ratified) ·
**OPEN** (no decision, no owner) · **GATED** (blocked on a named gate) · **ACTION** (not a
decision — work item).

Headline: of the 28 items, **13 are already answered** in the corpus, **11 are genuinely
open**, and **4 are actions, not decisions**. The open ones collapse onto **six gates**:
`AM-1`, `D1`, `D2`, `D4`, the DAG-identity ruling, and licensing.

---

## THE FOUR BIG QUESTIONS

### BQ-1 + BQ-4 ARE THE SAME DECISION — and it is the top open item

Your Q1 ("what is AWIS becoming") and Q4 ("DAG only vs full automation platform") are not
two questions. They are one, and the corpus **contradicts itself across authority tiers**:

- **Tier-0 `AWIS_PRD.md` §3 "What AWIS Is Not"** explicitly rejects being a no-code
  automation tool (Zapier/Make), a visual builder, a BPM/BPMN engine, and a UiPath-style
  process automation platform. AWIS = "workflow runtime… engine developers never want to
  rebuild." FROZEN tier-0.
- **Tier-2 `docs/09-gui-planning/GUI_PRD.md:7-8`** sets the target as
  "**an n8n-class visual workflow product**, reached in five phases," with Phase 4 a visual
  editor and Phase 5 "full n8n-class experience."

n8n *is* a no-code visual automation product. The GUI corpus is steering at precisely the
category the frozen PRD rejects. Per `docs/07-indices/canonical-reference-map.md`, authority
resolves **upward** and tier-0 wins — so as written, **the PRD already forbids the GUI
roadmap's stated destination**. Nobody has run the CONTRA protocol on this.

The engineering cost is now precisely known, not vague:

> Loops are blocked by **step identity**, not by the validator. `step_claims` PRIMARY KEY is
> `(instance_id, step_id)` [`internal/storage/migrations/0001_core_execution.sql:43-48`], so
> a step can be claimed **exactly once per instance, forever**. Note the result cache key
> already carries attempt (`hash(instance_id + step_id + attempt)`, `:21`) — claims do not.
> Iteration requires putting an iteration/attempt ordinal into that PK: a **frozen-surface
> migration** touching claim, projection and rebuild paths.

`D-12` in `archive/VERIFIED_DEFECT_REGISTER.md:120` is explicit: **"NOT A DEFECT… deliberate, cited,
frozen architectural decision (EDR-010). No action taken or recommended without a product
decision to support loops, which would require redesigning step identity on a frozen surface."**

**Status: OPEN — highest-value decision in the project.** Everything in your items 8, 9, 10
and half of 4 is downstream of it. Until it is ruled, Phase 4/5 GUI planning is speculative
work against a spec that forbids it.

*(Caution: `D-12` is an ID collision. `archive/VERIFIED_DEFECT_REGISTER.md` D-12 = DAG/loops.
`docs/11-intelligence-architecture/00-ARCHITECTURE_REVIEW.md:166` D-12 = "`intelligence:`
config key recognised but ignored." Different registers, same ID. Rename one.)*

### BQ-2 — Authoritative repository state

**ACTION, not a decision.** Authoritative code = `engine-hardening@8a87f70` **plus the
uncommitted working tree**; neither alone builds the product. `main` is 62 commits stale and
lacks M10–M17. Fix is `AD-01`: commit `internal/api/`, `cmd/awis-server/`, `internal/buildinfo/`,
`web/`, correct the false `.gitignore` comment (it claims "source is tracked" for `web/` — it
is not), then re-run verification, because that will be the **first pass that has ever covered
that code**. Nothing else on this list can be trusted until this is done.

### BQ-3 — Is event sourcing a hard invariant?

**Documents treat it as hard; the code violates it in exactly two places, and says so.**
`internal/storage/rebuild.go:38-44` states plainly that `waiting` status and the
`cancellation_requested` flag are tick-time/flag state that **"a pure event replay can never
reconstruct"**; rebuild recovers them from the `wait_records` table and a pre-wipe snapshot.

So AWIS is event-sourced **plus two side-channels**. That is an honest, documented deviation —
not a hidden bug. But it means the log is *not* a sufficient source of truth, and **no document
rules on whether that is permanent**. Two coherent end-states, neither chosen:
- **(a) Strict:** add `WorkflowWaiting` / `CancellationRequested` event types (13–14 total),
  rebuild becomes pure. Cost: event-type set is a frozen surface.
- **(b) Pragmatic:** ratify "event log + projection side tables" as the real model and correct
  the blueprint's "state IS an event-sourced append-only log" claim.

**Status: OPEN.** Recommend (a) if replay/audit is ever to be sold as a guarantee, (b) if not —
but the current unstated middle is the worst of both, because it lets docs claim (a).

---

## PART 1 — ALREADY ANSWERED IN THE REPO (13)

| # | Your item | Answer | Where |
|---|---|---|---|
| 2 | Status of untracked GUI/API code | **Accepted-but-uncommitted, product-grade — NOT experimental.** Logged card-by-card as "Landed, integrated, uncommitted"; gap analysis calls it "MVP (shipped)" and treats running behaviour as overriding docs. | `DASHBOARD_GATE_STATUS.md`, `GUI_BETA_GAP_ANALYSIS.md` |
| 6 | Replay: recovery only or runtime feature? | **Debug + recovery only, and they are two different commands.** `awis replay <id>` = dry-run, walks the log, prints what *would* run, **writes nothing**. `awis rebuild-state` = projection rebuild after corruption. No doc anywhere proposes replay as a runtime re-execution primitive. | `docs/CLI_CONTRACT.md:978-1000, 1271-1277` |
| 15 | Plugin end-state | **DECIDED.** Subprocess/JSON-RPC now; **WASM at V3** for untrusted third-party steps; marketplace V4+. Shared-library, gRPC and remote-service models are **explicitly rejected** (crash isolation / protocol complexity). | `AWIS_ARCHITECTURE_BLUEPRINT.md` §29, §2013-2022, §2127 |
| 14 | Plugin permission model | **Intentional V1 blank, correctly recorded.** V1 threat model explicitly excludes authn/authz/network security/access control. Isolation ≠ authorization, and the docs never claim otherwise. Design of the V2 model is the open part. | Blueprint §21 |
| 17 | Serializer | **Planned, scoped, estimated: `G6`, 8–12 days, "largest single milestone."** Round-trip is a **hard requirement** ("a serializer either round-trips correctly or it doesn't"). Text-first is **already committed** — editor layout persists into the existing `Metadata.ui.positions` map, zero schema change. | `GUI_ROADMAP.md` Ph.4, `GUI_ARCHITECTURE.md` §8.2-8.3 |
| 18 | Mutation API architecture | **Designed (not built).** POST `/workflows` → RegisterWorkflow, **new version only, no update path**; `/workflows/validate`; POST `/instances`, `/signal`, `/cancel` (must expose `compensate`, which the SDK currently hardcodes `false`). | `GUI_ARCHITECTURE.md` §6 |
| 19 | Realtime model | **SSE, not WebSocket — ADR-4.** Reasoning: traffic is one-directional, Last-Event-ID resumption is free, works through proxies, needs no third dependency. **Beta ships polling on purpose** (15s `setInterval`), documented as "genuinely out of scope," not an oversight. | `GUI_ARCHITECTURE.md` §5.1; `web/src/screens/instanceList.ts:32` |
| 21 | GUI Beta vs Creation/Edit APIs | **Five phases:** 1 read-only dashboard · 2 live monitoring (SSE, needs D1) · 3 management (submit/signal/cancel/register, needs D2) · 4 visual editor (G6+G7) · 5 n8n-class. "**GUI Beta**" is a deliberately narrower cut of Phases 1–2 only — an *observability release*, mutation layer and all of G3 dropped. | `GUI_ROADMAP.md`, `GUI_BETA_PRD.md:5` |
| 23 | Deployment architecture | Blueprint §28 defines the deployment models; local-first single-binary is a **product commitment**, not just a stage (PRD §25), with a V2 cloud upgrade path. | Blueprint §28, PRD §25 |
| 26 | EDR-002 Cobra vs stdlib | **Not a contradiction — a legitimate reversal recorded in the wrong place.** `IMPLEMENTATION_MASTER_PLAN.md:47` always permitted "cobra **or** stdlib `flag` — implementer's choice." M14 chose stdlib on dependency-policy grounds and recorded it three times. **EDR-002 is simply stale and needs an amendment note.** | `M14-core-cli/{IMPLEMENTATION_SPEC,AI_EXECUTION_CONTEXT,TRACEABILITY}.md` |
| 27 | Stale status docs | **Confirmed, with a live example:** `DEFERRED_WORK_REGISTER.md` still lists `G-G10-3` (subprocess env secret-leak) as pending idle-slot work, but the same defect is `D-04` in `archive/VERIFIED_DEFECT_REGISTER.md` and is marked **FIXED** as of 2026-09-05. | both registers |
| 28 | Superseded reports still present | **Confirmed process gap.** An archive convention exists and is enforced for one doc (`archive/ENGINEERING_ARCHITECTURE_BLUEPRINT.md` — "never cite"), but superseded root reports were never moved. `archive/FINAL_VERDICT.md` is superseded by `archive/FINAL_RELEASE_VERDICT.md`; `archive/ENGINE_READINESS_SCORECARD.md` (3.9/10) predates the hardening branch that closed most of what it counts. Both still sit in root looking current. | `canonical-reference-map.md`, root listing |
| 16 | GUI blocked on D1–D4 | Confirmed — enumerated in Part 2. Note `ENGINE_FREEZE_REPORT.md` §6's "two small blockers" **are** D1 (global event cursor) and G6 (serializer). Same items, two vocabularies. | `ENGINE_GUI_DECISION_RECORD.md:9-21` |

---

## PART 2 — GENUINELY OPEN (11)

### The four GUI founder decisions — *"none of D1–D4 has been decided"*, re-confirmed 2026-09-01

| ID | Question | Recommended | Blast radius if deferred |
|---|---|---|---|
| **D1** | `state_changes` table **vs** `global_seq` column, to give the UI a global event cursor | `state_changes` | **G2 blocked → all of Phase 2+ blocked.** Today `sequence_num` is per-instance, `emitted_at` is neither unique nor monotonic, `event_id` is an unordered UUID — there is no column a UI can tail. SQLite rowid rejected: VACUUM renumbers it. |
| **D2** | Retire the `"default"` namespace overload | retire | G3 blocked; StepStats/metrics **silently wrong**; ambiguity gets baked into saved filters and URLs if deferred past G5 |
| **D3** | GUI backend in-module vs separate service | in-module | No hard blocker — **DE-FACTO decided**, `cmd/awis-server` exists and is in-module |
| **D4** | Gate live-intelligence checks behind a secret-gated CI job | gate it | G0 cannot formally close; G9 ships unverified against the real API. Precedent: the stale `claude-sonnet-4-5` model-ID bug was caught **only** by a live check — the fake-server suite structurally cannot catch that class |

### Intelligence layer — one hard gate, and a subsystem that does not actually run

- **`AM-1` — `sdk.Config` freeze amendment. GATED, no workaround.** The whole intelligence
  ADR is `Status: Proposed — awaiting founder decision`. Recommendation is **Option B**
  (Registry + Driver + Instance); Option C (cost/quality policy routing) judged "correct end
  state, wrong sequence." Nothing downstream (AM-1..AM-6, P-1..P-8) moves until AM-1 is signed.
- **Your items 12 and 13 have a blunt answer: NONE.** Repo-wide grep finds **zero** mentions of
  MCP. No plan exists anywhere for tool/function-calling, agentic execution, or multimodal.
  Tool-calling appears **only** as a *risk* — a reason not to standardise on an
  OpenAI-compatible shape, and a warning that Groq/Together divergence could make a driver a
  "conditional swamp." The `IntelligencePort` has no tool parameter in any current or proposed
  revision.
- **The port is not frozen forever.** A precedent-based amendment process exists (ADJ-7/ADJ-8),
  and a five-stage migration is drafted, ending in `Invoke`/`Describe` replacing the four
  capability methods behind a back-compat shim. That path is drafted, not approved.
- **NEW FINDING — routing is built, tested, and inert in production.** I verified this directly
  rather than taking the audit's word: `sdk.Config.Intelligence` is a **single** `IntelligencePort`
  [`sdk/runtime.go:81`], wired as a **one-element** registration list tagged `LocalityLocal`
  [`sdk/runtime.go:85-88`]. `Route()` returns `Eligible()[0]`. The `model_hint` values `fast`
  and `quality` filter on `LocalityCloud` — **they can never match anything.** The DSL accepts
  the hint, 32 tests cover the router, Blueprint §17 documents a routing decision tree as if
  live, and in the shipped binary none of it can fire. This is a documentation-integrity issue
  as much as an engineering one.

### Security — blanks with no seam to fill them (your item 20: correct, and understated)

Deferral is recorded and defensible for a single-operator local tool. Two things make it worse
than "not yet implemented":
- **`AD-07`: there is no middleware seam on the HTTP surface** — no place to *add* auth without
  restructuring. Timing "**Now**."
- The HTTP API has **no authentication, CORS policy, rate limiting, body limits, or server
  timeouts** (verified by zero-match grep and live probing). Loopback default limits exposure,
  but `--addr` accepts any interface with no auth, no warning, and no deployment guide saying
  otherwise. This is the second reason `archive/FINAL_RELEASE_VERDICT.md` is CONDITIONAL PASS.

### Operations, governance, identity

- **7 · Version migration of running instances — OPEN, but possibly moot by design.** No document
  addresses it. Note the design may already answer it: definitions are immutable and every save
  is a *new version, never an in-place update*, so a running instance stays pinned to the version
  it started under. If that is the intent, **say so** — right now it is an accident that looks
  like a gap.
- **21 · Disaster recovery — OPEN.** Nothing beyond `rebuild-state` + event replay. No backup
  procedure, no restore drill, no RPO/RTO. Compounding: `AD-05` — durability runs at
  `synchronous=NORMAL`, which is an **undocumented** durability guarantee (a power loss can lose
  recent commits). DR cannot be specified until that is written down.
- **22 · Availability/SLA — OPEN/absent.** No availability target exists. For a single-operator
  local binary that is arguably N/A — but no document says "N/A," so it reads as an omission.
- **3 · The acronym — OPEN and formally logged.** `NCI-18: "Define AWIS acronym or retire it"`,
  classified `SAFE_TO_DEFER`. The glossary defines OIP = Organizational Intelligence Platform
  but never AWIS. Decide or retire; it costs one line either way.
- **24 · Licensing — OPEN, and worse than "undecided": there is NO `LICENSE` file in the repo.**
  No license, no `SPDX` header, no mention in `go.mod` or `README.md`. Under default copyright
  that means **all rights reserved — no third party may legally use, fork, or contribute**, and
  the V4+ plugin-marketplace ambition is legally unreachable until this is fixed. This is the
  cheapest high-consequence item on the list.
- **25 · Monetization — OPEN.** The only monetization surface named anywhere is the V4+ plugin
  marketplace. No pricing, tiering, or open-core boundary. Reasonable to defer — but note it is
  **coupled to licensing**: open-core vs source-available constrains #24, and #24 is now urgent.

---

## PART 3 — CRITICAL PATH

```
AD-01 commit the deliverable ─┬─> re-verify (first real pass over API/GUI code)
                              └─> everything below becomes trustworthy

LICENSE decision ──────────────> unblocks any external contribution / marketplace

PRD-vs-GUI ruling (BQ-1/BQ-4) ─┬─> DAG-only? -> GUI Phases 4-5 are out of scope, replan
                               └─> n8n-class? -> step-identity migration, PRD §3 amended
                                     └─> then, and only then, G6 serializer is worth 8-12d

D1 ──> G2 ──> Phase 2 (live monitoring, SSE)
D2 ──> G3 ──> Phase 3 (correct namespace filtering, mutation UI)
AM-1 ─> AM-2..6 / P-1..8 ──> multi-provider intelligence; also fixes the inert router
D4 ──> G0 close-out ──> G9
```

**Sequencing note:** rule BQ-1/BQ-4 **before** funding G6. If the DAG-only decision stands,
the 8–12 day serializer serves an editor the PRD forbids — the most expensive way to discover
a contradiction that is currently free to resolve.

---

## PART 4 — NEW FINDINGS FROM THIS PASS

1. **No `LICENSE` file exists.** Not "undecided" — legally absent.
2. **Intelligence routing cannot execute** in the shipped binary (single adapter, all
   `LocalityLocal`; `fast`/`quality` hints match nothing). Blueprint §17 documents it as live.
3. **The loop blocker is `step_claims` PK `(instance_id, step_id)`**, not the validator. The
   result-cache key already carries `attempt`; claims do not. That asymmetry is the migration.
4. **`D-12` is an ID collision** across two registers (DAG-only vs inert `intelligence:` config key).
5. **EDR-002 is stale, not contradicted** — the stdlib-`flag` reversal was permitted by IMP §47
   and recorded three times at M14. Add an amendment note to EDR-002 and the drift closes.
6. **The `.gitignore` comment is false**: it states `web/` source "is tracked." It is not.
7. **`archive/ENGINE_READINESS_SCORECARD.md` (3.9/10, NOT READY) predates the hardening branch** and
   should be archived — it is the most alarming document in the root and the most out of date.
