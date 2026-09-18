# GUI Phase 1 — Architecture

**Process note before the content:** this document's conclusions were originally drafted
as a proposal and then adversarially challenged, per this exercise's subagent protocol.
The dedicated "Frontend challenger" subagent failed mid-run — not a finding, an
infrastructure failure (the session hit its usage limit before the agent could report
back). Rather than ship the unchallenged first draft, the supervisor performed the same
challenge directly, in the same adversarial spirit (attempt to falsify, not rubber-stamp).
Three real weaknesses were found and are reflected as revisions below, not as the
original proposal. This is disclosed so the reader can weigh the critique's rigor
accordingly — it did not have a second, independent pair of eyes on it.

---

## 1. Frontend architecture

**No SPA framework** (React/Vue/Svelte/Next.js/Electron/Tauri all explicitly
out of scope for this exercise, and none is warranted for 6 read-only screens with no
mutation and no complex client state).

**Rendering: [lit-html](https://lit.dev/docs/libraries/standalone-templates/), not
fully hand-rolled DOM manipulation.** The original draft proposed hand-written
render-and-replace functions per screen. Challenged directly: for 6 screens with tables,
filters, pagination, and a 15-second poll, a naive "clear `innerHTML` and rebuild" cycle
on every poll tick loses scroll position, input focus, and any expanded/collapsed row
state — a real, user-visible defect, not a hypothetical one. lit-html is a ~5KB
templating/rendering library (not an application framework: no component lifecycle, no
required build tooling, no router or state management opinions) that does keyed DOM
patching for you — only the parts of the DOM that actually changed are touched. It is
loadable as a native ES module with zero required build step, and every screen becomes a
plain function `(state) => html\`...\`` passed to lit-html's `render()`. This keeps the
"no framework" spirit (it does one job, rendering, and nothing else) while removing a
real, specific pain point the original proposal didn't handle.

**Build step: [esbuild](https://esbuild.github.io/), not `tsc`-only.** The original
draft proposed compiling TypeScript with `tsc` alone and loading the output as native
ES modules with no bundler. Challenged directly: `tsc` does not rewrite import specifiers
to include the `.js` extension the browser requires (`import './foo'` in a `.ts` file
must be written as `import './foo.js'` in the source for the compiled output to resolve
correctly in a browser with no bundler) — a well-known, easy-to-hit footgun for a team
not already fluent in this specific pattern. esbuild is a single, dependency-free static
binary, configures in a few lines, builds in milliseconds, and handles module resolution,
bundling, and minification transparently — removing this footgun without introducing
webpack-class configuration complexity. This is still "no heavyweight bundler," just not
"zero bundler."

**Routing: hash-based (`#/instances/:id`), confirmed correct on challenge.** Zero
server-side routing/fallback configuration is needed — a hash fragment is never sent to
the server, so `cmd/awis-server` only ever needs to serve one static `index.html` (plus
assets) regardless of client-side route. Browser back/forward navigation works correctly
with hash routing via the standard `hashchange` event and normal history-entry pushes;
there is no real downside for an internal, single-operator tool where deep-link
shareability and SEO are irrelevant.

**State management.** No global store. Each screen module owns its own local state
(current filter values, current page/cursor, last-fetched data, loading/error flags).
Data flows one way: fetch → local state → render. The only cross-screen state is the
current route (owned by the router) and, trivially, the currently-selected namespace
filter if a future revision wants it to persist across navigation — not required for
Phase 1's acceptance criteria, so not built now.

**API integration.** A single `apiClient.ts` module, hand-typed to mirror the 6 real Go
response shapes exactly (see `GUI_PHASE1_API_MATRIX.md` for the authoritative field
list) — one function per route, each returning a typed Promise, each throwing a typed
error distinguishing a 404 (structured, "not found") from a network/5xx failure
(unstructured, "something went wrong"), matching the two distinct error-state
treatments `GUI_PHASE1_SCREEN_SPEC.md` specifies per screen.

**Polling.** A simple `setInterval`-driven refetch, 15 seconds, on any screen currently
showing list or detail data; paused while a request is in flight (no overlapping
requests) and resumable via a manual refresh button that resets the interval. 15 seconds
against a loopback server serving a single operator's browser tab is not a load concern
worth engineering around further.

## 2. Deployment model

**Production: a single binary.** Compiled frontend assets (HTML/CSS/esbuild output) are
embedded into `cmd/awis-server` via `//go:embed` and served at `/` (and any non-`/api/v1`
path, since hash routing means every client-side route still requests the same
`index.html`). This matches the backend's own established deployment ethos exactly —
loopback-bound, single process, minimal external dependencies — and avoids CORS
entirely, since the frontend and API are always same-origin.

**Development: a `--static-dir` flag, not a rebuild-the-Go-binary loop.** The original
draft assumed `go:embed` unconditionally, which was challenged directly: rebuilding and
restarting the Go binary on every CSS/JS edit during active frontend development is real
iteration friction the original proposal didn't address. Revision: `cmd/awis-server`
gains a `--static-dir <path>` flag (small, additive — see
`GUI_PHASE1_BACKLOG.md`'s backend-additions list) that serves static assets from disk
instead of the embedded `embed.FS` when set. A frontend engineer runs `esbuild --watch`
against a `web/` source directory, points `awis-server --static-dir web/dist` at its
output, and edits are visible on browser refresh with no Go rebuild in the loop at all.
The embedded-FS path remains the default and the only thing a production build ships —
`--static-dir` is a development convenience, not a second deployment mode to maintain.

**This backend addition is the single concrete "highest-leverage" item this document
identifies** — see `GUI_PHASE1_BACKLOG.md` and the final recommendation.

## 3. What this architecture deliberately does not include

No state management library (Redux/Zustand/etc. — nothing to manage that a few local
variables per screen don't already cover). No CSS framework/design system decision made
here — out of this document's scope; any reasonable minimal CSS approach (hand-written,
or a small utility set) is compatible with everything above and left to whoever
implements the visual design. No testing framework choice made here — the existing
`internal/api` test suite already proves the backend contract; frontend testing strategy
is Beta-or-later scope, not required to start building Phase 1's screens.

---

*Companion documents: `GUI_PHASE1_PRD.md`, `GUI_PHASE1_SCREEN_SPEC.md`,
`GUI_PHASE1_API_MATRIX.md`, `GUI_PHASE1_BACKLOG.md`.*
