// instanceDetail.ts — GUI_PHASE1_SCREEN_SPEC.md §2. The countdown re-derives
// client-side between polls from the last-fetched timeout_remaining_s plus
// elapsed wall time, so it visibly ticks rather than only updating once per
// 15s poll.
import { html, render, TemplateResult } from "lit-html";
import { getInstance } from "../apiClient.js";
import type { InstanceDetail } from "../types.js";
import { navigate } from "../router.js";
import {
  loadingView,
  errorView,
  notFoundView,
  isNotFound,
  errorMessage,
  statusBadge,
  freshnessView,
  startVisibilityAwarePoll,
} from "../ui.js";

const POLL_MS = 15000;
const TICK_MS = 1000;

interface State {
  loading: boolean;
  error: unknown;
  data: InstanceDetail | null;
  fetchedAt: number; // Date.now() at the moment `data` was fetched, for the countdown
}

export function mountInstanceDetail(container: HTMLElement, instanceId: string): () => void {
  const state: State = { loading: true, error: null, data: null, fetchedAt: 0 };
  let stopped = false;

  function draw(): void {
    render(view(state, actions, instanceId), container);
  }

  async function load(): Promise<void> {
    state.loading = state.data === null;
    state.error = null;
    draw();
    try {
      const data = await getInstance(instanceId);
      if (stopped) return;
      state.data = data;
      state.fetchedAt = Date.now();
    } catch (err) {
      if (stopped) return;
      state.error = err;
    } finally {
      if (!stopped) {
        state.loading = false;
        draw();
      }
    }
  }

  const actions = {
    retry(): void {
      void load();
    },
    viewEvents(): void {
      navigate(`/instances/${encodeURIComponent(instanceId)}/events`);
    },
  };

  void load();
  const stopPolling = startVisibilityAwarePoll(() => void load(), POLL_MS);
  // A faster tick just re-renders between polls without re-fetching — cheap,
  // and keeps both the countdown and the freshness indicator visibly
  // moving. Not visibility-gated itself (cheap enough to not matter, and
  // stopping it would leave a stale countdown/freshness label frozen
  // mid-count on tab return instead of resuming smoothly).
  //
  // GUI Beta FE-4b fix: this used to only fire when timeout_remaining_s was
  // non-null, which meant the "as of Xs ago" freshness indicator sat frozen
  // at "just now" for any instance that wasn't currently waiting-with-a-
  // timeout — confirmed by the Beta Verifier (10s wait on a completed
  // instance's detail page, indicator never moved). Now ticks whenever data
  // exists, regardless of countdown state; draw() itself is cheap enough
  // that ticking once a second with nothing to show for it is a non-issue.
  const tickHandle = setInterval(() => {
    if (state.data) draw();
  }, TICK_MS);

  return () => {
    stopped = true;
    stopPolling();
    clearInterval(tickHandle);
  };
}

interface Actions {
  retry(): void;
  viewEvents(): void;
}

function liveCountdown(state: State): number | null {
  if (!state.data || state.data.timeout_remaining_s === null) return null;
  const elapsedS = Math.floor((Date.now() - state.fetchedAt) / 1000);
  return Math.max(0, state.data.timeout_remaining_s - elapsedS);
}

function formatCountdown(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}m ${s}s remaining`;
}

function view(state: State, actions: Actions, instanceId: string): TemplateResult {
  return html`
    <section class="screen">
      <div class="screen-header">
        <h2>Instance ${instanceId}</h2>
        ${state.fetchedAt > 0 ? freshnessView(state.fetchedAt) : ""}
      </div>
      ${bodyView(state, actions)}
    </section>
  `;
}

// bodyView follows GUI_PHASE1_SCREEN_SPEC.md §1/§2's error-state contract: a
// transient fetch failure shows the error banner ABOVE the last-successful
// detail view (stale but visible) when one exists. A 404 is deliberately
// exempt from this — the instance genuinely does not exist at this id, so
// there is no "stale but still true" data to keep showing alongside that
// message; showing both would be actively misleading, not just noisy.
function bodyView(state: State, actions: Actions): TemplateResult {
  if (state.error && isNotFound(state.error)) {
    return notFoundView(`This instance does not exist. (${errorMessage(state.error)})`);
  }
  if (state.loading && !state.data) return loadingView("Loading instance…");
  if (state.error) {
    return html`${errorView(state.error, actions.retry)}${state.data
      ? dataView(state.data, liveCountdown(state), actions)
      : ""}`;
  }
  if (!state.data) return loadingView("Loading instance…");
  return dataView(state.data, liveCountdown(state), actions);
}

function dataView(d: InstanceDetail, countdown: number | null, actions: Actions): TemplateResult {
  return html`
    <dl class="detail-grid">
      <dt>Status</dt>
      <dd>${statusBadge(d.status)}</dd>
      <dt>Definition</dt>
      <dd>${d.definition_id} @ ${d.version}</dd>
      <dt>Current Step(s)</dt>
      <dd>${d.current_steps.join(", ") || "—"}</dd>
      <dt>Created</dt>
      <dd>${d.created_at}</dd>
      <dt>Updated</dt>
      <dd>${d.updated_at}</dd>
    </dl>

    ${d.status === "waiting"
      ? html`
          <div class="wait-info">
            <h3>Waiting</h3>
            <p>Signal: <strong>${d.signal_name ?? "—"}</strong></p>
            <p>${countdown === null ? "No timeout configured" : formatCountdown(countdown)}</p>
          </div>
        `
      : ""}

    <details>
      <summary>Inputs</summary>
      <pre class="json-block">${JSON.stringify(d.inputs ?? {}, null, 2)}</pre>
    </details>
    <details>
      <summary>Outputs</summary>
      <pre class="json-block">${JSON.stringify(d.outputs ?? {}, null, 2)}</pre>
    </details>

    <button class="primary" @click=${actions.viewEvents}>View Events</button>
  `;
}
