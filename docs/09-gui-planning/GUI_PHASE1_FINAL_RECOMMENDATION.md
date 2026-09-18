# GUI Phase 1 — Final Recommendation

Direct, implementation-ready answers. Full detail: `GUI_PHASE1_PRD.md`,
`GUI_PHASE1_ARCHITECTURE.md`, `GUI_PHASE1_SCREEN_SPEC.md`, `GUI_PHASE1_API_MATRIX.md`,
`GUI_PHASE1_BACKLOG.md`.

**Process disclosure:** this phase's architecture, screen set, and MVP scope were meant
to go through an independent "Frontend challenger" subagent per this exercise's own
protocol. That subagent failed mid-run from a session usage-limit error, not a finding.
The supervisor performed the same adversarial challenge directly rather than ship the
unchallenged first draft — three real weaknesses were found and revised (below); two
decisions held. This recommendation reflects the challenged, revised version, but it did
not get a second independent reviewer. Flagging this so the reader can weigh it
accordingly, and so a real second review remains a live option if wanted.

---

**1. What exact GUI should be built first?**

A single-page app with four screens shipping together as MVP: **Instance List**
(landing, default-sorted by most-recently-updated, with namespace/status filters),
**Instance Detail** (including the signal-name/timeout-countdown wrapper), **Event
Timeline** (per-instance, cursor-paginated, with the 12-event-type summary table from
`GUI_PHASE1_API_MATRIX.md`), and **Workflow List**. Workflow Detail is real but
deferred to Beta — it's the one screen with genuine design complexity relative to
everything else in this set. Full screen-by-screen behavior is in
`GUI_PHASE1_SCREEN_SPEC.md`.

**2. What framework should be used?**

No SPA framework. **lit-html** (~5KB, a rendering library, not an application
framework) for keyed DOM updates without hand-rolled re-render logic — this replaced the
original "fully hand-written DOM manipulation" draft after direct challenge showed it
would lose scroll/focus/expanded-row state on every 15-second poll. **esbuild** for the
build step — replaced an original `tsc`-only proposal after challenge surfaced a real
ES-module-extension footgun `tsc` alone doesn't solve. Hash-based client-side routing
(`#/instances/:id`) — challenged and confirmed correct, since it needs zero server-side
routing configuration. No state-management library; no CSS framework decision made here.
Full reasoning in `GUI_PHASE1_ARCHITECTURE.md`.

**3. What can be shipped in the fastest possible MVP?**

The four-screen MVP above, once one small backend addition lands: a `--static-dir` flag
on `cmd/awis-server` (or its `go:embed` production equivalent) so the frontend has
anywhere to be served from at all — `GET /` 404s on the binary as it exists today. That
backend change is roughly 1-2 hours of work. Everything else is pure frontend work
against the 6 API routes that already exist, tested, and verified working end-to-end
this session. No further backend changes are required to start or finish this MVP.

**4. What backend additions would provide the highest GUI leverage?**

Ranked:
1. **The `--static-dir`/`go:embed` static-file-serving addition** — blocking, not
   optional; nothing in this phase ships without it. See `GUI_PHASE1_BACKLOG.md`.
2. **A `/api/v1/stats`-style aggregate endpoint** (instance counts by status in one
   query) — real leverage, but only for a Beta-or-later "counts" dashboard variant that
   this recommendation does not include in MVP; the existing routes already support a
   perfectly good "recent activity" landing view with zero new backend work.
   Not needed to ship MVP.
3. Nothing else. The 6-route surface that shipped this session is sufficient for the
   entirety of this phase's screen set as specified.

**5. What should explicitly NOT be built yet?**

Everything this task's brief already excluded, restated as a concrete stop-list:
workflow editing, workflow creation, any mutation action (submit/signal/cancel) from the
GUI, canvas/graph rendering of any kind (even read-only — correctly deferred past this
phase, not because it's forbidden forever but because building it now risks exactly the
scope drift this phase was scoped to avoid), authentication or per-user access control,
new observability/metrics infrastructure, AI-assisted features, and live/streaming
updates via SSE (a distinct, later phase gated on a migration that hasn't landed and a
founder decision that hasn't been made). Also not yet: Workflow Detail (Beta, not MVP),
a true counts-based Dashboard screen (Beta-or-never, pending real usage feedback), and
any frontend testing-framework decision (not required to start building).

---

*This document and its companions (`GUI_PHASE1_PRD.md`, `GUI_PHASE1_ARCHITECTURE.md`,
`GUI_PHASE1_SCREEN_SPEC.md`, `GUI_PHASE1_API_MATRIX.md`, `GUI_PHASE1_BACKLOG.md`) are
sufficient for an engineer to begin building today.*
