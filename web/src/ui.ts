// ui.ts — small shared lit-html templates and formatters used by every
// screen: loading/empty/error states (GUI_PHASE1_SCREEN_SPEC.md's per-screen
// state conventions), relative-time, and a status badge.
import { html, TemplateResult } from "lit-html";
import { ApiError, ApiNotFoundError } from "./apiClient.js";
import { ACTIVE_STATUSES } from "./types.js";

export function loadingView(label = "Loading…"): TemplateResult {
  return html`<div class="state state-loading">${label}</div>`;
}

export function emptyView(message: string): TemplateResult {
  return html`<div class="state state-empty">${message}</div>`;
}

export function notFoundView(message: string): TemplateResult {
  return html`<div class="state state-notfound">${message}</div>`;
}

export function errorView(err: unknown, onRetry: () => void): TemplateResult {
  const message = err instanceof Error ? err.message : String(err);
  return html`
    <div class="state state-error">
      <span>Could not load data: ${message}</span>
      <button @click=${onRetry}>Retry</button>
    </div>
  `;
}

// isNotFound lets a screen render notFoundView specifically for a 404
// (a real, structured signal — storage.ErrWorkflowNotFound /
// storage.ErrInstanceNotFound) rather than the generic errorView, per
// GUI_PHASE1_SCREEN_SPEC.md's Instance Detail state requirements.
export function isNotFound(err: unknown): boolean {
  return err instanceof ApiNotFoundError;
}

export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) return err.message;
  return err instanceof Error ? err.message : String(err);
}

// relativeTimeFromMs is the shared formatter behind both relativeTime (ISO
// timestamps from the API, e.g. updated_at) and freshnessLabel (a raw
// Date.now()-style ms timestamp tracked client-side for "as of Xs ago").
function relativeTimeFromMs(then: number): string {
  if (Number.isNaN(then)) return "unknown";
  const diffMs = Date.now() - then;
  const diffS = Math.round(diffMs / 1000);
  if (diffS < 5) return "just now";
  if (diffS < 60) return `${diffS}s ago`;
  const diffM = Math.round(diffS / 60);
  if (diffM < 60) return `${diffM}m ago`;
  const diffH = Math.round(diffM / 60);
  if (diffH < 24) return `${diffH}h ago`;
  const diffD = Math.round(diffH / 24);
  return `${diffD}d ago`;
}

export function relativeTime(iso: string): string {
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) return iso;
  return relativeTimeFromMs(then);
}

// freshnessLabel is GUI Beta FE-4b: "as of Xs ago" for a screen's last
// successful fetch, so staleness is visible instead of implicit — a
// confirmed-necessary requirement, not decoration (GUI_BETA_PRD.md F4).
export function freshnessView(fetchedAtMs: number): TemplateResult {
  return html`<span class="freshness" title="Last updated at ${new Date(fetchedAtMs).toLocaleTimeString()}"
    >as of ${relativeTimeFromMs(fetchedAtMs)}</span
  >`;
}

export function statusBadge(status: string): TemplateResult {
  const cls = ACTIVE_STATUSES.has(status) ? "badge badge-active" : "badge badge-terminal";
  return html`<span class="${cls}">${status}</span>`;
}

// startVisibilityAwarePoll runs fn on a recurring interval, but pauses while
// the browser tab is hidden (document.hidden) and refetches immediately on
// becoming visible again rather than waiting for the next scheduled tick —
// GUI Beta FE-4a. Confirmed necessary: no visibilitychange handling existed
// anywhere in this frontend before this card, so a backgrounded tab polled
// forever with no visible consequence. Does not call fn immediately itself
// — callers already do an initial load() before starting the poll.
export function startVisibilityAwarePoll(fn: () => void, intervalMs: number): () => void {
  let handle: ReturnType<typeof setInterval> | null = null;

  function start(): void {
    if (handle !== null) return;
    handle = setInterval(fn, intervalMs);
  }
  function stop(): void {
    if (handle !== null) {
      clearInterval(handle);
      handle = null;
    }
  }
  function onVisibilityChange(): void {
    if (document.hidden) {
      stop();
    } else {
      fn();
      start();
    }
  }

  start();
  document.addEventListener("visibilitychange", onVisibilityChange);

  return () => {
    stop();
    document.removeEventListener("visibilitychange", onVisibilityChange);
  };
}
