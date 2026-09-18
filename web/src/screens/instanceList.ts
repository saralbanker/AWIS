// instanceList.ts — the landing screen (GUI_PHASE1_SCREEN_SPEC.md §1).
// Filters: namespace (free text) + status (fixed dropdown, the API takes
// exactly one InstanceStatus value, not a list). Polls every 15s.
//
// FE-7 (GUI Beta, confirmed real bug): rows previously re-sorted in full on
// every poll tick, so any row whose updated_at changed visibly jumped
// position on a routine background refresh. Fixed two ways: (1) rows are
// only re-sorted when the underlying instance *set* actually changes
// (a row added/removed) — a routine refresh of the same set of instances
// preserves prior display order even if their data changed; (2) the table
// body uses lit-html's keyed repeat() directive, keyed by instance_id, so
// DOM nodes patch in place rather than tearing down/rebuilding on every
// render.
import { html, render, TemplateResult } from "lit-html";
import { repeat } from "lit-html/directives/repeat.js";
import { listInstances, listWorkflows } from "../apiClient.js";
import type { InstanceEntry, WorkflowEntry } from "../types.js";
import { INSTANCE_STATUSES } from "../types.js";
import { navigate } from "../router.js";
import {
  loadingView,
  emptyView,
  errorView,
  relativeTime,
  statusBadge,
  freshnessView,
  startVisibilityAwarePoll,
} from "../ui.js";
import { onboardingView } from "../onboarding.js";

const PAGE_SIZE = 50;
const POLL_MS = 15000;
// TICK_MS re-renders the "as of Xs ago" freshness indicator on its own
// clock, independent of POLL_MS — GUI Beta FE-4b fix. Without this, the
// label is computed once at fetch time and then sits frozen in the DOM
// until the next poll resets it (confirmed via the Beta Verifier: it read
// "as of just now" at every observation point across a full 15s poll
// cycle). A frozen indicator is worse than none: its whole purpose is to
// reveal staleness, including the case where polling itself has silently
// stalled — which a frozen "just now" would hide instead of expose.
const TICK_MS = 1000;

interface State {
  namespace: string;
  status: string;
  // definitionId is GUI Beta FE-2 (consumes BE-1's server-side filter).
  definitionId: string;
  offset: number;
  loading: boolean;
  error: unknown;
  data: { instances: InstanceEntry[]; total: number } | null;
  fetchedAt: number;
  // jumpId is FE-3's exact-ID-navigation input; not part of the filter set
  // above, it's a separate "just take me there" affordance.
  jumpId: string;
  // onboardingWorkflows is non-null once checked: FE-5's true-first-run
  // detection queries /workflows only when instances comes back empty AND
  // every filter is at its default (an overly narrow filter must show the
  // ordinary empty state, not onboarding) — set to [] if /workflows is also
  // empty, or the real list if not (used for the example command).
  onboardingWorkflows: WorkflowEntry[] | null;
}

export function mountInstanceList(container: HTMLElement): () => void {
  const state: State = {
    namespace: "",
    status: "",
    definitionId: "",
    offset: 0,
    loading: true,
    error: null,
    data: null,
    fetchedAt: 0,
    jumpId: "",
    onboardingWorkflows: null,
  };
  let stopped = false;

  function draw(): void {
    render(view(state, actions), container);
  }

  async function load(): Promise<void> {
    state.loading = state.data === null; // only show the full loading state on first load
    state.error = null;
    draw();
    try {
      const resp = await listInstances({
        namespace: state.namespace || undefined,
        status: state.status || undefined,
        definitionId: state.definitionId || undefined,
        limit: PAGE_SIZE,
        offset: state.offset,
      });
      if (stopped) return;
      // RC-3: the API's native order is (started_at, instance_id) DESCENDING
      // (internal/storage/sqlite.go's ListInstancesPaged) — newest first, so
      // page 1 always contains the most recently started instances. This
      // client-side re-sort by updated_at is now a secondary, independent
      // ordering applied within that page (e.g. a long-running instance that
      // updated most recently still floats to the top), not a correction for
      // an inverted backend order.
      //
      // Only actually re-sort when the instance SET changed (an id added or
      // removed) — a routine poll refreshing the same set of instances
      // preserves the previous display order instead, even though each
      // row's own fields (status, updated_at, ...) still update in place.
      // This is the FE-7 fix: without it, any row whose updated_at ticked
      // forward between polls would visibly jump to the top every refresh.
      const previousOrder = state.data ? state.data.instances.map((i) => i.instance_id) : [];
      const byId = new Map(resp.instances.map((i) => [i.instance_id, i]));
      const previousSet = new Set(previousOrder);
      const sameSet =
        previousOrder.length === byId.size &&
        previousOrder.every((id) => byId.has(id)) &&
        [...byId.keys()].every((id) => previousSet.has(id));

      const ordered =
        sameSet && previousOrder.length > 0
          ? previousOrder.map((id) => byId.get(id)!)
          : [...resp.instances].sort(
              (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
            );
      state.data = { instances: ordered, total: resp.total };
      state.fetchedAt = Date.now();

      // FE-5: only probe for true-first-run when there's actually nothing
      // to show and no filter could be responsible for that. Runs after the
      // main render below so the instance table (or its empty state) is
      // never blocked on this extra call.
      const noFilters = !state.namespace && !state.status && !state.definitionId;
      if (resp.total === 0 && noFilters) {
        void checkOnboarding();
      } else {
        state.onboardingWorkflows = null;
      }
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

  async function checkOnboarding(): Promise<void> {
    try {
      const workflows = await listWorkflows();
      if (stopped) return;
      state.onboardingWorkflows = workflows;
    } catch {
      // A failed workflows check just means no onboarding banner this
      // render — the ordinary empty-state remains correct, and the next
      // poll will retry it as a side effect of calling load() again.
      if (!stopped) state.onboardingWorkflows = null;
    }
    if (!stopped) draw();
  }

  const actions = {
    setNamespace(v: string): void {
      state.namespace = v;
      state.offset = 0;
      void load();
    },
    setStatus(v: string): void {
      state.status = v;
      state.offset = 0;
      void load();
    },
    setDefinitionId(v: string): void {
      state.definitionId = v;
      state.offset = 0;
      void load();
    },
    setJumpId(v: string): void {
      state.jumpId = v;
    },
    jumpToInstance(): void {
      const id = state.jumpId.trim();
      if (!id) return;
      // No local validation beyond "non-empty": Instance Detail's own 404
      // handling already renders the correct "does not exist" state for a
      // bad/unknown id — this action is pure navigation, not a second
      // error-handling path (FE-3's scope, per GUI_BETA_WORK_BREAKDOWN.md).
      navigate(`/instances/${encodeURIComponent(id)}`);
    },
    nextPage(): void {
      if (!state.data) return;
      if (state.offset + PAGE_SIZE >= state.data.total) return;
      state.offset += PAGE_SIZE;
      void load();
    },
    prevPage(): void {
      state.offset = Math.max(0, state.offset - PAGE_SIZE);
      void load();
    },
    retry(): void {
      void load();
    },
    openInstance(id: string): void {
      navigate(`/instances/${encodeURIComponent(id)}`);
    },
  };

  void load();
  const stopPolling = startVisibilityAwarePoll(() => void load(), POLL_MS);
  // Ticks the freshness indicator only — no fetch, no data change, just a
  // re-render so "as of Xs ago" actually counts up between polls.
  const tickHandle = setInterval(() => {
    if (state.fetchedAt > 0) draw();
  }, TICK_MS);

  return () => {
    stopped = true;
    stopPolling();
    clearInterval(tickHandle);
  };
}

interface Actions {
  setNamespace(v: string): void;
  setStatus(v: string): void;
  setDefinitionId(v: string): void;
  setJumpId(v: string): void;
  jumpToInstance(): void;
  nextPage(): void;
  prevPage(): void;
  retry(): void;
  openInstance(id: string): void;
}

function view(state: State, actions: Actions): TemplateResult {
  return html`
    <section class="screen">
      <div class="screen-header">
        <h2>Instances</h2>
        ${state.fetchedAt > 0 ? freshnessView(state.fetchedAt) : ""}
      </div>
      <div class="filters">
        <label
          >Namespace
          <input
            type="text"
            placeholder="All namespaces"
            .value=${state.namespace}
            @change=${(e: Event) => actions.setNamespace((e.target as HTMLInputElement).value)}
          />
        </label>
        <label
          >Status
          <select @change=${(e: Event) => actions.setStatus((e.target as HTMLSelectElement).value)}>
            <option value="" ?selected=${state.status === ""}>All statuses</option>
            ${INSTANCE_STATUSES.map(
              (s) => html`<option value=${s} ?selected=${state.status === s}>${s}</option>`
            )}
          </select>
        </label>
        <label
          >Workflow
          <input
            type="text"
            placeholder="All workflows"
            .value=${state.definitionId}
            @change=${(e: Event) => actions.setDefinitionId((e.target as HTMLInputElement).value)}
          />
        </label>
        <label
          >Jump to instance
          <form
            @submit=${(e: SubmitEvent) => {
              e.preventDefault();
              actions.jumpToInstance();
            }}
          >
            <input
              type="text"
              placeholder="Paste an instance ID"
              .value=${state.jumpId}
              @input=${(e: Event) => actions.setJumpId((e.target as HTMLInputElement).value)}
            />
          </form>
        </label>
      </div>
      ${bodyView(state, actions)}
    </section>
  `;
}

// bodyView follows GUI_PHASE1_SCREEN_SPEC.md §1's error-state contract: a
// transient fetch failure shows the error banner ABOVE the last-successful
// table (stale but visible), not instead of it — only a failure on the very
// first load (state.data still null) shows the banner alone, since there is
// nothing yet to keep showing.
function bodyView(state: State, actions: Actions): TemplateResult {
  if (state.loading) return loadingView("Loading instances…");
  if (state.error) {
    return html`${errorView(state.error, actions.retry)}${state.data
      ? dataView(state.data, state.offset, actions)
      : ""}`;
  }
  if (!state.data || state.data.instances.length === 0) {
    // FE-5: onboardingWorkflows is only ever set when the instance set is
    // genuinely empty AND every filter is at its default — see load()'s
    // noFilters check. state.data.instances.length === 0 here can also mean
    // "no instances match this specific filter," which correctly falls
    // through to the ordinary empty state instead.
    if (state.onboardingWorkflows !== null) {
      return onboardingView(state.onboardingWorkflows);
    }
    return emptyView("No instances match these filters.");
  }
  return dataView(state.data, state.offset, actions);
}

function dataView(
  data: { instances: InstanceEntry[]; total: number },
  offset: number,
  actions: Actions
): TemplateResult {
  const { instances, total } = data;
  const page = Math.floor(offset / PAGE_SIZE) + 1;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  return html`
    <table class="data-table">
      <thead>
        <tr>
          <th>Instance</th>
          <th>Definition</th>
          <th>Version</th>
          <th>Status</th>
          <th>Current Step(s)</th>
          <th>Updated</th>
        </tr>
      </thead>
      <tbody>
        ${repeat(
          instances,
          (inst) => inst.instance_id,
          (inst) => html`
            <tr class="clickable-row" @click=${() => actions.openInstance(inst.instance_id)}>
              <td class="mono" title=${inst.instance_id}>${inst.instance_id.slice(0, 8)}…</td>
              <td>${inst.definition_id}</td>
              <td>${inst.version}</td>
              <td>${statusBadge(inst.status)}</td>
              <td>${inst.current_steps.join(", ") || "—"}</td>
              <td title=${inst.updated_at}>${relativeTime(inst.updated_at)}</td>
            </tr>
          `
        )}
      </tbody>
    </table>
    <div class="pagination">
      <button ?disabled=${offset === 0} @click=${actions.prevPage}>Prev</button>
      <span>Page ${page} of ${pageCount} (${total} total)</span>
      <button ?disabled=${offset + PAGE_SIZE >= total} @click=${actions.nextPage}>Next</button>
    </div>
  `;
}
