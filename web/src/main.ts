// main.ts — app shell and route registration. Owns the single screen
// lifecycle: before mounting a new screen, calls the previous screen's
// cleanup (clears its poll interval) so screens never leak timers into
// each other across navigation. Also owns the connection-health/version
// indicator (GUI Beta FE-6a/FE-6b), which is deliberately NOT part of any
// screen's own lifecycle — it must keep reflecting connectivity even while
// sitting on a screen with no poll loop of its own (e.g. Workflow Detail).
import { html, render } from "lit-html";
import { route, startRouter, navigate, setNotFoundRedirect } from "./router.js";
import { mountInstanceList } from "./screens/instanceList.js";
import { mountInstanceDetail } from "./screens/instanceDetail.js";
import { mountEventTimeline } from "./screens/eventTimeline.js";
import { mountWorkflowList } from "./screens/workflowList.js";
import { mountWorkflowDetail } from "./screens/workflowDetail.js";
import { getInfo } from "./apiClient.js";
import { startVisibilityAwarePoll } from "./ui.js";

const appEl = document.getElementById("app");
if (!appEl) throw new Error("main.ts: #app container not found in index.html");

let cleanupCurrentScreen: (() => void) | null = null;

function mount(fn: (container: HTMLElement) => () => void): void {
  if (cleanupCurrentScreen) {
    cleanupCurrentScreen();
    cleanupCurrentScreen = null;
  }
  // Each screen gets its own fresh child element, never a reused #app
  // directly: lit-html's render() attaches internal bookkeeping (a cached
  // "part" reference) to whatever container element it's given. Manually
  // clearing that same element's children on navigation (e.g. via
  // replaceChildren()) removes the DOM nodes lit-html's cached part still
  // points to, which throws "Cannot read properties of null (reading
  // 'insertBefore')" on the next render into it — reproduced live via
  // browser testing when navigating from Instances to Workflows. Discarding
  // the whole previous screen's container element (instead of clearing its
  // insides) means lit-html's stale bookkeeping is discarded along with it,
  // and the new screen's container starts with no cached part to conflict.
  appEl!.replaceChildren();
  const screenContainer = document.createElement("div");
  appEl!.appendChild(screenContainer);
  cleanupCurrentScreen = fn(screenContainer);
}

route("/instances", () => mount((c) => mountInstanceList(c)));
route("/instances/:id", (params) => mount((c) => mountInstanceDetail(c, params.id!)));
route("/instances/:id/events", (params) => mount((c) => mountEventTimeline(c, params.id!)));
route("/workflows", () => mount((c) => mountWorkflowList(c)));
route("/workflows/:id/:version", (params) => mount((c) => mountWorkflowDetail(c, params.id!, params.version!)));
// Instance List is the landing screen (GUI_PHASE1_PRD.md §3) — both the bare
// "/" root and any unmatched hash redirect there.
setNotFoundRedirect("/instances");

function setupNav(): void {
  const links = document.querySelectorAll<HTMLAnchorElement>("[data-nav]");
  links.forEach((a) => {
    a.addEventListener("click", (e) => {
      e.preventDefault();
      navigate(a.getAttribute("data-nav")!);
    });
  });
}

// startConnectionIndicator polls GET /api/v1/info (which requires a
// successful round-trip and returns version/uptime in one call — no reason
// to also poll /healthz separately) every 10s, independent of whatever
// screen happens to be mounted, and renders a colored dot + version string
// into the nav bar. A failed request flips the dot red immediately, giving
// a persistent, always-visible connectivity signal instead of relying on
// each screen's own per-request error banner.
const CONN_POLL_MS = 10000;

function startConnectionIndicator(): void {
  const el = document.getElementById("conn-indicator");
  if (!el) return;

  let status: "unknown" | "ok" | "down" = "unknown";
  let version = "";

  function draw(): void {
    const cls = status === "ok" ? "conn-dot conn-ok" : status === "down" ? "conn-dot conn-down" : "conn-dot";
    render(
      html`<span class="conn-indicator">
        <span class="${cls}"></span>
        ${version ? html`v${version}` : ""}
      </span>`,
      el!
    );
  }

  async function check(): Promise<void> {
    try {
      const info = await getInfo();
      status = "ok";
      version = info.version;
    } catch {
      status = "down";
    }
    draw();
  }

  void check();
  startVisibilityAwarePoll(() => void check(), CONN_POLL_MS);
}

setupNav();
startConnectionIndicator();
startRouter();
