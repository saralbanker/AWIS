# AWIS — ANSWERS TO THE 28 OPEN ITEMS

Flat answer list, numbered to match the original 28. Nothing else.
2026-09-06 · `engine-hardening` @ `8a87f70` + uncommitted working tree · code beats doc.

Status: **DECIDED** · **DE-FACTO** (acted on, never ratified) · **OPEN** · **GATED** ·
**ACTION** (work item, not a decision) · **CONFIRMED** (the concern is factually correct).

---

**1 · Authoritative state given dirty tree + untracked GUI/API — ACTION**
Authoritative code = `engine-hardening@8a87f70` **plus the uncommitted working tree**; neither
builds the product alone. `main` is 62 commits stale, lacks M10–M17. Correct: there is no
reproducible source of truth today. Not a decision — it is `AD-01`: commit `internal/api/`,
`cmd/awis-server/`, `internal/buildinfo/`, `web/`, fix the false `.gitignore` comment (claims
`web/` source "is tracked" — it is not), re-verify. That will be the first verification pass
ever to cover that code.

**2 · Status of the untracked GUI/API code — DECIDED (accepted-but-uncommitted)**
Not experimental. Logged card-by-card as "Landed, integrated, uncommitted"; the gap analysis
calls it "MVP (shipped)" and treats running behaviour as overriding docs. Product-grade code
that was never `git add`-ed. [`DASHBOARD_GATE_STATUS.md`, `GUI_BETA_GAP_ANALYSIS.md`]

**3 · AWIS acronym — OPEN, formally logged**
No expansion exists anywhere. Tracked as `NCI-18: "Define AWIS acronym or retire it"`,
classified `SAFE_TO_DEFER`. Glossary defines OIP = Organizational Intelligence Platform, never
AWIS. One line to close, either way. [`AWIS_CANONICAL_SPECIFICATION_TRIBUNAL_REPORT.md:764,922`]

**4 · Runtime vs automation platform vs application platform vs OIP substrate — OPEN (tier conflict)**
The corpus contradicts itself across authority tiers. Tier-0 `AWIS_PRD.md` §3 explicitly
rejects being a no-code automation tool (Zapier/Make), a visual builder, a BPM/BPMN engine, or
a UiPath-style platform: AWIS = "workflow runtime… engine developers never want to rebuild."
Tier-2 `GUI_PRD.md:7-8` targets "an n8n-class visual workflow product" in five phases. n8n *is*
the category the PRD rejects. Authority resolves upward
(`docs/07-indices/canonical-reference-map.md`), so **as written the frozen PRD already forbids
the GUI roadmap's destination**. No CONTRA protocol has been run on it. Same decision as #8/#9/#10.

**5 · Not strictly event-sourced — CONFIRMED, and documented in code**
True. `internal/storage/rebuild.go:38-44` states that `waiting` status and
`cancellation_requested` are tick-time/flag state which "a pure event replay can never
reconstruct"; rebuild recovers them from `wait_records` and a pre-wipe snapshot. So the model is
event log **plus two side-channels** — an honest, recorded deviation, not a hidden bug. What is
OPEN is whether that is permanent. Two coherent end-states, neither chosen: **(a) strict** — add
`WorkflowWaiting`/`CancellationRequested` event types (13–14 total, frozen-surface change),
rebuild becomes pure; **(b) pragmatic** — ratify "log + projection side tables" and correct the
blueprint's "state IS an event-sourced append-only log" claim. The unstated middle is worst,
because it lets docs overclaim.

**6 · Replay: recovery only or future runtime feature — DECIDED (recovery + debug only)**
Two distinct commands, neither a runtime primitive. `awis replay <id>` = dry-run, walks the
event log, prints what *would* run, **writes nothing**. `awis rebuild-state` = projection
rebuild after corruption or manual log edits. No document anywhere proposes replay as runtime
re-execution (Temporal-style). [`docs/CLI_CONTRACT.md:978-1000, 1271-1277`]

**7 · Version migration of running instances — OPEN, possibly moot by design**
No document addresses it. But definitions are immutable and every save is a **new version, never
an in-place update**, so a running instance stays pinned to the version it started under. If that
is the intent, the migration question does not arise — it simply needs stating. Right now it is
an accident that reads as a gap.

**8 · Loops vs visual-builder ambitions — DECIDED as frozen, OPEN as product question**
`D-12` is explicit: "**NOT A DEFECT… deliberate, cited, frozen architectural decision
(EDR-010). No action taken or recommended without a product decision to support loops, which
would require redesigning step identity on a frozen surface.**" The real blocker is narrower than
the validator: `step_claims` PRIMARY KEY is `(instance_id, step_id)`
[`internal/storage/migrations/0001_core_execution.sql:43-48`], so a step can be claimed **exactly
once per instance, forever**. The result-cache key already carries attempt
(`hash(instance_id + step_id + attempt)`, `:21`); claims do not. That asymmetry *is* the
migration. Cost is now known and citable — the product ruling (#4) is not.
*Note: `D-12` is an ID collision — `archive/VERIFIED_DEFECT_REGISTER.md` D-12 = DAG/loops;
`docs/11-intelligence-architecture/00-ARCHITECTURE_REVIEW.md:166` D-12 = inert `intelligence:`
config key. Rename one.*

**9 · map/reduce, foreach, dynamic fan-out, dynamic graph generation — OPEN, all downstream of #8**
No strategy for any of them. Static fan-out transitions exist and are declared sufficient
(`FR-WD-12`: no `parallel` StepType in V1; fan-out transitions are the parallel mechanism);
`edr-011` defines the join gate for fan-out convergence. Everything *dynamic* — iteration count
known only at runtime, per-item fan-out, generated graphs — requires the same step-identity
change as #8. One decision covers all four.

**10 · Do graphs stay static forever — OPEN, this is the decision in #4/#8 restated**
Nothing commits either way. Static-DAG is frozen *today* (EDR-010 + the PK above); the GUI
roadmap's Phase 4/5 assumes it changes. Unresolved until #4 is ruled.

**11 · Long-term intelligence architecture — GATED on `AM-1`**
Nothing is decided. The ADR carries `Status: Proposed — awaiting founder decision`.
Recommendation = **Option B** (Registry + Driver + Instance: `openai-chat`, `anthropic-messages`,
`google-genai`). Option A (Registry only) rejected as expensive; Option C (cost/quality policy
routing) judged "correct end state, wrong sequence." Single hard gate: **`AM-1`, an `sdk.Config`
freeze amendment — "requires a founder decision. No workaround exists."** Everything downstream
(AM-1..AM-6, P-1..P-8) waits on it.
**Also: the routing layer cannot execute in the shipped binary.** `sdk.Config.Intelligence` is a
**single** port [`sdk/runtime.go:81`], wired as a one-element registration tagged `LocalityLocal`
[`:85-88`]; `Route()` returns `Eligible()[0]`; `model_hint` values `fast`/`quality` filter on
`LocalityCloud` and **match nothing**. 32 tests cover a subsystem that never fires, while
Blueprint §17 documents the decision tree as live.

**12 · How tools/functions fit into IntelligencePort — NONE**
No plan exists. `IntelligencePort` has no tool/function parameter in any current or proposed
revision (methods: Draft, Embed, Synthesize, Classify, IsAvailable, Capabilities, ProviderName —
`internal/core/ports.go:44-63`). Tool-calling appears **only as a risk**: a reason not to
standardise on an OpenAI-compatible shape, and a warning that Groq/Together divergence would make
a driver a "conditional swamp." Agentic execution: also NONE — no agent loops, sub-agents, or
autonomous tool use anywhere. Multimodal: NONE.

**13 · MCP-style integrations — NONE**
Repo-wide `grep -ri mcp` returns **zero hits**. Not planned, not rejected, not mentioned.

**14 · Plugin permission model — DECIDED as an intentional V1 blank**
Correct that authorization does not exist; the docs never claim it does. The V1 threat model
explicitly excludes authn/authz/network security/access control (Blueprint §21). What exists is
*isolation*: separate OS process, `Setpgid` process-group kill, minimal env (`manifest env` +
`PATH` only, `NFR-S-02`), no extra FD inheritance. Filesystem restriction is "by convention," not
enforced. Isolation ≠ authorization — the V2 authorization design is the genuinely open part, and
it is unstarted.

**15 · Plugin end-state — DECIDED**
Subprocess/JSON-RPC over stdio now; **WASM at V3** for untrusted third-party steps; marketplace
V4+. Shared-library (breaks crash isolation), gRPC (protocol complexity without proportionate
benefit at this scale) and remote services are **explicitly rejected**.
[`AWIS_ARCHITECTURE_BLUEPRINT.md` §29, :2013-2022, :2127]

**16 · GUI blocked on D1–D4 — CONFIRMED, all four still open**
Verbatim: "**Independently re-confirmed 2026-09-01: none of D1-D4 has been decided since the
corpus was written. All four decisions remain open.**"
· **D1** `state_changes` table vs `global_seq` column (global event cursor). Rec: `state_changes`.
Blocks G2 → all of Phase 2+. Today `sequence_num` is per-instance, `emitted_at` is neither unique
nor monotonic, `event_id` is an unordered UUID — no column a UI can tail. SQLite rowid rejected:
VACUUM renumbers it.
· **D2** retire the `"default"` namespace overload. Rec: retire. Blocks G3; StepStats/metrics
silently wrong; ambiguity gets baked into saved filters and URLs if deferred past G5.
· **D3** GUI backend in-module vs separate service. Rec: in-module. **DE-FACTO decided** —
`cmd/awis-server` exists and is in-module.
· **D4** gate live-intelligence checks behind a secret-gated CI job. Rec: gate. Blocks G0
close-out and G9. Precedent: the stale `claude-sonnet-4-5` model-ID bug was caught *only* by a
live check; the fake-server suite structurally cannot catch that class.
No D5+. `ENGINE_FREEZE_REPORT.md` §6's "two small blockers" are D1 and G6 — same items, different
vocabulary. [`ENGINE_GUI_DECISION_RECORD.md:9-21`]

**17 · Serializer does not exist — CONFIRMED; planned, scoped, estimated**
True today: `internal/dsl` only parses; zero `yaml.Marshal` calls in the repo. Planned as **G6,
8–12 days, "the largest single milestone in the roadmap"** — an emitter against the same `yaml:`
struct tags the parser uses, plus the round-trip test that does not exist. **Round-trip is a hard
requirement** ("a serializer either round-trips correctly or it doesn't" — no partial-credit
save). Text-first is **already committed**, not merely planned: editor layout persists into the
existing `WorkflowDefinition.Metadata.ui.positions` map — zero schema change, works today. Open
sub-risk: map-key-order-safe emission for `Step.Inputs/Outputs/Trigger.Config` must be decided
*before* G6 is built. [`GUI_ROADMAP.md` Ph.4, `GUI_ARCHITECTURE.md` §8.2-8.3]

**18 · Mutation API — DESIGNED, not built; one real gap**
Architecture exists: POST `/workflows` → `Runtime.RegisterWorkflow`, **new version only, no
update path**; POST `/workflows/validate` → full structured Issues; POST `/instances` → Submit;
`/instances/{id}/signal`; `/instances/{id}/cancel` → must expose `compensate`, which the SDK
currently hardcodes `false` (a real fix, not yet done). Typed error sentinels map to 404/409/501.
**Genuine gap: no optimistic concurrency / ETag / version-check is designed anywhere** —
new-version-on-save sidesteps the question rather than answering it. `D-11` records the whole
area as "out of V1 scope by design," not a defect. [`GUI_ARCHITECTURE.md` §6]

**19 · SSE vs alternatives — DIRECTION DECIDED (ADR-4), gated on D1; Beta ships polling on purpose**
Chosen: **SSE, not WebSocket.** Reasoning: traffic is one-directional (every mutation is an
ordinary POST), Last-Event-ID resumption and reconnection come free, works through ordinary HTTP
proxies, and needs no dependency beyond stdlib `net/http` — a WebSocket library would be the
repo's third direct dependency and the first with protocol complexity. Polling-as-final-
architecture explicitly rejected ("a false economy"). **But SSE cannot exist until D1 gives it a
cursor**, so Beta ships pure 15s interval polling by documented decision — `POLL_MS = 15000` in
`web/src/screens/{instanceList,instanceDetail}.ts`, zero `EventSource`/`text/event-stream` hits.
Known accepted gaps: row order shuffles on poll; no tab-visibility pause.

**20 · Security blanks — CONFIRMED, and understated**
Deferral is recorded and defensible for a single-operator local tool, but two things make it
worse than "not yet implemented": **`AD-07` — there is no middleware seam on the HTTP surface**,
so there is nowhere to *add* auth without restructuring (timing: **Now**); and the API has **no
authentication, CORS policy, rate limiting, body limits, or server timeouts** (verified by
zero-match grep and live probing). Loopback default limits real exposure, but `--addr` accepts any
interface with no auth, no warning, and no deployment guide saying otherwise. This is the second
reason `archive/FINAL_RELEASE_VERDICT.md` is CONDITIONAL PASS rather than PASS.

**21 · Disaster recovery — OPEN, and blocked by an undocumented durability guarantee**
Nothing beyond `rebuild-state` + event replay. No backup procedure, no restore drill, no RPO/RTO.
Compounding: **`AD-05`** — the event log runs at `synchronous=NORMAL`, meaning a power loss can
lose recent commits, and that guarantee is **undocumented**. DR cannot be specified until the
durability level is written down and ratified.

**22 · Availability / SLA — OPEN, absent**
No availability target anywhere. For a single-operator local binary this is arguably N/A — but no
document *says* N/A, so it reads as an omission rather than a decision. PRD NFRs cover performance
targets; the scalability ceiling is measured (throughput plateaus ~3.4k rps, textbook
single-server queue), but that is a measurement, not an SLA.

**23 · Deployment architecture — DECIDED for V1, thin beyond it**
Blueprint §28 defines the deployment models, and local-first is a **product commitment**, not a
stage (PRD §25), with a V2 cloud upgrade path. Postgres/cloud mode is specified only at the level
of guarantees ("at-most-once step execution via claim/version mechanism") and is **not
implemented** — SQLite is the only working backend. So: V1 deployment is decided and real;
anything past it is a sketch.

**24 · Licensing — OPEN, and worse than undecided: there is NO `LICENSE` file**
No license file, no SPDX header, no mention in `go.mod` or `README.md`. Under default copyright
that means **all rights reserved — no third party may legally use, fork, or contribute**, and the
V4+ plugin-marketplace ambition is legally unreachable until it is fixed. Cheapest
high-consequence item on this list.

**25 · Monetization — OPEN**
The only monetization surface named anywhere is the V4+ plugin marketplace. No pricing, tiering,
or open-core boundary. Reasonable to defer, but it is **coupled to #24**: open-core vs
source-available constrains the licence choice, and the licence is now urgent.

**26 · EDR-002 Cobra vs stdlib — NOT a contradiction; a stale doc**
`IMPLEMENTATION_MASTER_PLAN.md:47` always permitted "CLI framework (`spf13/cobra` **or** stdlib
`flag` — implementer's choice, decided in M0 and recorded)." M14 chose stdlib on dependency-policy
grounds and recorded it three times (`M14-core-cli/IMPLEMENTATION_SPEC.md:29,85`,
`AI_EXECUTION_CONTEXT.md:20`, `TRACEABILITY.md:22`). Cobra is absent from `go.mod` and imported
nowhere; `cmd/awis/main.go` uses a stdlib `flag` mux. **EDR-002 was simply never amended.** One
amendment note closes the drift.

**27 · Stale status docs — CONFIRMED, with a live example**
`docs/09-gui-planning/DEFERRED_WORK_REGISTER.md` still lists `G-G10-3` (subprocess env
secret-leak) as pending idle-slot work, while the same defect is `D-04` in
`archive/VERIFIED_DEFECT_REGISTER.md` and is marked **FIXED** as of 2026-09-05. Also `STATE.md` records
M10–M16 at `E-MERGE (blocked on founder)` and M17 at `C-VERIFY`, none of which is visible from
`main`.

**28 · Superseded reports still present — CONFIRMED process gap**
An archive convention exists and is enforced for exactly one document
(`archive/ENGINEERING_ARCHITECTURE_BLUEPRINT.md` — "SUPERSEDED… never cite"), but superseded root
reports were never moved. `archive/FINAL_VERDICT.md` is superseded by `archive/FINAL_RELEASE_VERDICT.md`.
`archive/ENGINE_READINESS_SCORECARD.md` (overall **3.9/10, NOT READY**) predates the `engine-hardening`
branch that closed 11 CRITICAL + 12 IMPORTANT defects — it is simultaneously the most alarming
document in the repo root and the most out of date. Both still read as current.
