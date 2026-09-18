# AWIS GUI — Roadmap

Status: proposed. Basis: `GUI_MASTER_PLAN.md` §13 (milestone roadmap G0–G10), reframed
here into the five product phases requested for planning purposes: read-only dashboard
→ workflow monitoring → workflow management → visual workflow editor → full n8n-class
experience. Every G-milestone from the master plan is used exactly once below; nothing
is invented and nothing is dropped. See `GUI_DEPENDENCY_MAP.md` for the full DAG and
critical path; see `GUI_PRD.md` (organized by the same five phases) for the functional
requirements each phase delivers.

---

## How to read this

Each phase lists: required backend work, required frontend work, dependencies (founder
decisions and prior phases/milestones), and risks. Estimates are the master plan's own
per-milestone estimates, rolled up.

---

## Phase 1 — Read-only dashboard

**Delivers:** FR-1.1–1.5 (`GUI_PRD.md` §9). An operator can see every workflow
definition and every instance, running or historical, without the CLI.

**Required backend work**
- G0 — Freeze close-out: B-31 remediation (already done, 2026-08-30), the D4 decision,
  delete/rebuild the stale repo-root `awis` binary, add a `make` target so future live
  audits can't run against a stale one. *1 day.*
- G1 — Contract hardening: golden JSON snapshots on `ExecutionEvent` and CLI output
  shapes, a reflective test pinning `StoragePort` at 12 methods, a `GetWorkflow`
  not-found sentinel, and a corrected `CLI_CONTRACT.md`. *2–3 days.*
- G4, read-only subset — `cmd/awis-server` skeleton + the five GET routes
  (`/workflows`, `/workflows/{id}/{version}`, `/instances`, `/instances/{id}`,
  `/instances/{id}/events`) and `/healthz`. No SSE, no mutations yet. *~2 days of the
  full G4 estimate.*

**Required frontend work**
- SPA shell, routing, definitions list/detail, instance list/detail (paged), event
  timeline view (static, polling-refresh is acceptable here — no live requirement yet).
- Build against the binary and the G1 golden snapshots, not `CLI_CONTRACT.md` (§6.3 of
  the master plan documents at least eleven points of divergence).

**Dependencies:** none blocking — this phase can start immediately (`GUI_MASTER_PLAN.md`
§0, §14: "frontend and history views can start tomorrow"). Does not require D1, D2, or
D3 to be *decided*, but D3 (in-module) determines where the read routes live, so land
that decision before G4 work starts, not after.

**Risks:** R3 (building against the stale contract doc), R4 (freeze is comment-only —
this is exactly what G1 exists to fix), R15 (`cmd/awis` — the package the new contract
tests sit beside — currently has the lowest test coverage of any core package; a flaky
neighbor erodes trust in the new gates).

**Estimated size:** ~1 week (G0 + G1 in series, G4-read-subset overlapping the tail of
G1 once the contract is pinned).

---

## Phase 2 — Workflow monitoring

**Delivers:** FR-2.1–2.4. A running-instances view that updates live and never lies
about `waiting` state.

**Required backend work**
- G2 — Cursor + eventing: migration 0007 (`state_changes`, ADR-1 in
  `GUI_ARCHITECTURE.md`), writes added to the two existing transactions
  (`AppendEvent`, `UpsertInstance`), a `ReadChangesSince` method, an SDK wrapper, and a
  regression test proving a parked instance actually produces a `state_changes` row
  (not just that the migration runs). *2 days.*
- The `/stream` SSE endpoint (part of G4, pulled forward here since Phase 2 is the
  first consumer): tails `state_changes` from `?since=`, resumable via
  `Last-Event-ID`. *Included in G4's remaining estimate, not separately budgeted.*

**Required frontend work**
- Live-updating instance list (subscribe to `/stream`, patch in place).
- Per-instance live timeline component.
- Reconnect/backoff handling using `Last-Event-ID` — do not silently drop events on a
  network blip.
- UI copy that reflects tick-quantized latency honestly (no spinner implying
  sub-second updates); see `GUI_ARCHITECTURE.md` §5.2.

**Dependencies:** **D1 must be decided before this phase starts** — it is the one
founder decision that gates a phase outright, because without it the live view cannot
represent `waiting`, timeout-resumption, or cancellation-requested at all (the single
highest-severity finding in the underlying review, R1 "Critical"). Depends on Phase 1's
G1 contract work being in place so the SSE payload shape is itself pinned.

**Risks:** R1 (the defect this phase exists to close — verify the fix against a live
daemon before calling this phase done, not just against the migration's unit test), R7
(two engines on one DB — add the PID/lock guard here since this is where a live
`awis-server` process starts competing with `awis start`).

**Estimated size:** ~2 days engine work (G2) + SSE handler and frontend live-view work,
roughly comparable in effort to a polling-based version — per the master plan, "the
cheapest path [polling] is a false economy: polling is not much less work than the SSE
handler, and it must be thrown away" (§13.1). Build the SSE version directly.

---

## Phase 3 — Workflow management

**Delivers:** FR-3.1–3.4. Submit, signal, cancel (with compensate), register new
versions, and namespace-correct filtering everywhere.

**Required backend work**
- G3 — Namespace resolution: unify the three duplicated `"default"` predicate
  implementations (`export.go`, `history.go`, `metrics.go`) into one helper, give
  `ReadEventRange` a wildcard branch, add `--all-namespaces` / `?all_namespaces=true`,
  migration notes for the behavior change. *2 days.*
- G4, remainder — the POST routes (`/instances`, `/instances/{id}/signal`,
  `/instances/{id}/cancel`, `/workflows`, `/workflows/validate`), plus exposing
  `compensate` on `Runtime.Cancel` (SDK hardcodes `false` today — a small, isolated
  fix). *Remainder of the 5–7 day G4 estimate not already spent in Phases 1–2.*

**Required frontend work**
- Submit form (workflow + version + input picker), signal delivery UI, cancel action
  with compensate toggle and "requested, not stopped" messaging, workflow
  registration/versioning UI.
- Namespace selector consistent across every view — this is the direct product
  consequence of G3 landing; do not ship a namespace picker before G3, or the picker
  itself will encode the ambiguity R2 describes.

**Dependencies:** **D2 must be decided before this phase's namespace-filtering work
starts** (it can run in parallel with G2/Phase 2 — they touch disjoint code). Depends
on Phase 1's API skeleton and Phase 2's SSE hub for immediate feedback on submitted
actions.

**Risks:** R2 (Critical — `StepStats` silently zeros on namespace mismatch; do not ship
the stats/metrics surface from Phase 5 ahead of this fix, and do not ship *this* phase's
namespace filter without it either), the namespace breakage itself is intentional and
disclosed (§6.4 of the master plan) — communicate it in release notes, don't silently
ship it.

**Estimated size:** G3 (2 days) runs in parallel with Phase 2's G2; G4's mutation
routes and frontend forms are the larger share of this phase's time.

---

## Phase 4 — Visual workflow editor

**Delivers:** FR-4.1–4.6. Build and edit a workflow visually, save it back as valid
YAML, with a palette that reflects what's actually registered.

**Required backend work**
- G6 — Serializer + round trip: `internal/dsl` gets a YAML emitter matching the
  parser's own `yaml:` struct tags (none exists today — zero `yaml.Marshal` calls
  repo-wide), plus the round-trip test that doesn't exist yet. Canvas editing and live
  validation ride on existing, already-pure `validate.Validate` and
  `expr.Parse{Template,Condition}` — no new engine work needed for diagnostics. *8–12
  days, the largest single milestone in the roadmap.*
- G7 — Palette + introspection: three small `List*` accessors on `engine.runners`,
  `NativeRunner.handlers`, and `plugin.Manager` so the node palette reflects the live
  deployment instead of a hardcoded list. *3–4 days.*

**Required frontend work**
- Canvas rendering all six relationship types (unconditional/conditional/error
  transitions, fallback, compensation, retry/timeout badges) — see
  `GUI_ARCHITECTURE.md` §8.1, including the fallback/cycle-detection asymmetry the
  canvas must mirror exactly.
- Layout persisted via `WorkflowDefinition.Metadata.ui.positions` (no schema change,
  works today).
- Inline diagnostics wired to `validate.Validate`'s structured `Issue` shape.
- Node palette driven by G7's new accessors, not hardcoded.
- Avoid the `Step.Inputs` naming trap — it is a template value map, not a JSON Schema,
  despite the field's name and comment (`GUI_ARCHITECTURE.md` §8.2).

**Dependencies:** Depends on Phase 3 (definitions must be manageable via API before
they're editable via canvas) and on G6 specifically — there is no partial-credit
version of "save," since a serializer either round-trips correctly or it doesn't.

**Risks:** this is the largest, least-de-risked milestone in the roadmap — no code for
the serializer exists today, only the parser side. Budget schedule float here first if
the program slips.

**Estimated size:** ~2–2.5 weeks (G6 dominates; G7 can run partially in parallel once
G6's data model for the palette is settled).

---

## Phase 5 — Full n8n-class experience

**Delivers:** FR-5.1–5.5. Cost/token dashboards, AI-assisted generation, multi-user
deployment.

**Required backend work**
- G8 — Observability: incremental step-stats aggregate fed off the `state_changes`
  cursor from Phase 2 (replacing `StepStats`'s current 10-year full rescan per call);
  cost/token dashboard queries via `json_extract` + a new index on `event_type` — the
  data is already persisted in `StepCompleted` payloads
  (`TestIntelligence_SuccessADJ8Payload` proves this end-to-end today), only the query
  layer is unbuilt. *4–5 days.*
- G9 — AI generation: a JSON Schema artifact generated from `core.WorkflowDefinition`
  (none exists today — an LLM must currently be prompted from prose or Go structs
  directly), wiring the CLI's discarded validation codes (`awis workflow validate
  --json` currently drops `Code` and `Field`, keeping only `line`+`message`) through to
  the API layer's repair loop. *5–8 days.*
- G10 — Multi-user: authn/authz and per-namespace ACLs from zero (no user/role/
  tenant/session concept exists anywhere today), subprocess environment scrubbing to
  match the plugin manager's existing isolation, remote bind support. *10–15 days, the
  largest milestone in the whole roadmap.*

**Required frontend work**
- Cost/token dashboard views.
- NL-to-workflow generation UI with inline repair-loop feedback.
- Login/session UI, per-namespace role management (once G10 backend exists).

**Dependencies:** G8 depends on Phase 2's `state_changes` cursor (G2). G9 depends on
Phase 4's editor (G6) for a place to land generated workflows, and on D4 (live-
intelligence gate) being resolved so generation features ship "verified" rather than
"declared-unverified." G10 depends on Phase 1's read surface existing (something to
protect) but is otherwise independent and is explicitly the correct place to *start*
authz work, not a task to fold into an earlier phase.

**Risks:** G10 is a from-zero security build, not a feature addition — treat it as its
own milestone with its own review gate, per `GUI_PRD.md` §4 non-goals. R13 (subprocess
env inheritance) must close before Phase 4's editor is opened to non-operator users,
which in practice means before G10, not as part of it.

**Estimated size:** the long tail — G8 and G9 can each land in under two weeks; G10 is
comparable in size to Phases 1–3 combined and should be scheduled as a distinct program
phase, not squeezed into a sprint.

---

## Cross-phase sequencing note

Phases 1–3 map cleanly onto the master plan's stated critical path: **G0 → G1 → G2 →
G4 → G5 delivers a demonstrable live GUI in roughly three weeks**, with G3 (Phase 3's
namespace fix) running in parallel with G2 (Phase 2's cursor work). Phase 4 (G6/G7) and
Phase 5 (G8/G9/G10) are correctly sequenced *after* a working live GUI exists, not
before — see `GUI_DEPENDENCY_MAP.md` for the full graph and the three alternative
sequencing paths (cheapest / fastest-to-screen / safest) the master plan lays out in
its §13.1.
