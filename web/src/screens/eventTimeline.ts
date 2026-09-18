// eventTimeline.ts — GUI_PHASE1_SCREEN_SPEC.md §3, scaled up for GUI Beta
// FE-8. Cursor-paginated (NOT offset-based, unlike Instance List) — "Load
// more" appends rather than replacing, matching the API's next_cursor
// contract exactly.
//
// FE-8 (GUI Beta, confirmed real gap): PAGE_SIZE raised from 50 toward the
// backend's own cap (storage.MaxEventsPageSize = 1000) to cut "Load more"
// click count for real event volumes — an instance with several hundred
// events is now readable in ~1-2 clicks instead of a dozen. Also: raw
// payload JSON is only computed (JSON.stringify) for a row once its
// <details> is actually expanded, tracked via expandedIds — previously this
// ran unconditionally on every re-render for every row regardless of
// whether anyone had opened it.
import { html, render, TemplateResult } from "lit-html";
import { listEvents } from "../apiClient.js";
import type { ExecutionEvent } from "../types.js";
import { navigate } from "../router.js";
import { loadingView, emptyView, errorView, notFoundView, isNotFound, errorMessage } from "../ui.js";
import { summarizeEvent } from "../eventFormatters.js";

const PAGE_SIZE = 200;

interface State {
  loading: boolean;
  loadingMore: boolean;
  error: unknown;
  events: ExecutionEvent[];
  nextCursor: number | null;
  expandedIds: Set<string>;
}

export function mountEventTimeline(container: HTMLElement, instanceId: string): () => void {
  const state: State = {
    loading: true,
    loadingMore: false,
    error: null,
    events: [],
    nextCursor: null,
    expandedIds: new Set(),
  };
  let stopped = false;

  function draw(): void {
    render(view(state, actions, instanceId), container);
  }

  async function loadFirstPage(): Promise<void> {
    state.loading = true;
    state.error = null;
    state.events = [];
    state.nextCursor = null;
    draw();
    try {
      const resp = await listEvents(instanceId, { limit: PAGE_SIZE });
      if (stopped) return;
      state.events = resp.events ?? [];
      state.nextCursor = resp.next_cursor ?? null;
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

  async function loadMore(): Promise<void> {
    if (state.nextCursor === null || state.loadingMore) return;
    state.loadingMore = true;
    draw();
    try {
      const resp = await listEvents(instanceId, { from: state.nextCursor, limit: PAGE_SIZE });
      if (stopped) return;
      state.events = state.events.concat(resp.events ?? []);
      state.nextCursor = resp.next_cursor ?? null;
    } catch (err) {
      if (stopped) return;
      state.error = err;
    } finally {
      if (!stopped) {
        state.loadingMore = false;
        draw();
      }
    }
  }

  const actions = {
    retry(): void {
      void loadFirstPage();
    },
    loadMore(): void {
      void loadMore();
    },
    backToInstance(): void {
      navigate(`/instances/${encodeURIComponent(instanceId)}`);
    },
    toggleExpanded(eventId: string): void {
      if (state.expandedIds.has(eventId)) {
        state.expandedIds.delete(eventId);
      } else {
        state.expandedIds.add(eventId);
      }
      draw();
    },
  };

  void loadFirstPage();

  return () => {
    stopped = true;
  };
}

interface Actions {
  retry(): void;
  loadMore(): void;
  backToInstance(): void;
  toggleExpanded(eventId: string): void;
}

function view(state: State, actions: Actions, instanceId: string): TemplateResult {
  return html`
    <section class="screen">
      <button class="link-button" @click=${actions.backToInstance}>&larr; Back to instance</button>
      <h2>Events for ${instanceId}</h2>
      ${bodyView(state, actions)}
    </section>
  `;
}

// bodyView follows GUI_PHASE1_SCREEN_SPEC.md §3's error-state contract
// (same conventions as §1): a transient failure — most likely a failed
// "Load more" click, since loadFirstPage's own failure leaves state.events
// empty by construction — shows the error banner ABOVE the already-loaded
// events (stale but visible), not instead of them. A 404 stays exempt, same
// reasoning as Instance Detail: the instance doesn't exist, so there is
// nothing genuinely "stale but true" to keep showing.
function bodyView(state: State, actions: Actions): TemplateResult {
  if (state.error && isNotFound(state.error)) {
    return notFoundView(`This instance does not exist. (${errorMessage(state.error)})`);
  }
  if (state.loading) return loadingView("Loading events…");
  if (state.error) {
    return html`${errorView(state.error, actions.retry)}${state.events.length > 0
      ? eventsView(state, actions)
      : ""}`;
  }
  if (state.events.length === 0) return emptyView("No events yet.");
  return eventsView(state, actions);
}

function eventsView(state: State, actions: Actions): TemplateResult {
  return html`
    <ol class="event-list">
      ${state.events.map((ev) => eventRow(ev, state.expandedIds.has(ev.event_id), actions))}
    </ol>
    ${state.nextCursor !== null
      ? html`<button ?disabled=${state.loadingMore} @click=${actions.loadMore}>
          ${state.loadingMore ? "Loading…" : "Load more"}
        </button>`
      : ""}
  `;
}

// eventRow only computes JSON.stringify(payload) when this row is actually
// expanded (tracked in State.expandedIds, synced via <details>'s native
// `toggle` event) — the FE-8 fix for the stringify-on-every-render cost the
// prior version paid for every row regardless of whether anyone opened it.
function eventRow(ev: ExecutionEvent, expanded: boolean, actions: Actions): TemplateResult {
  return html`
    <li class="event-row">
      <span class="event-time" title=${ev.emitted_at}>${ev.emitted_at}</span>
      <span class="event-type">${ev.event_type}</span>
      <span class="event-summary">${summarizeEvent(ev.event_type, ev.step_id, ev.payload)}</span>
      <details ?open=${expanded} @toggle=${() => actions.toggleExpanded(ev.event_id)}>
        <summary>raw</summary>
        ${expanded ? html`<pre class="json-block">${JSON.stringify(ev.payload, null, 2)}</pre>` : ""}
      </details>
    </li>
  `;
}
