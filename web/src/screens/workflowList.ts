// workflowList.ts — GUI_PHASE1_SCREEN_SPEC.md §4. Unbounded (ListWorkflows
// has no pagination, by design — see GUI_PHASE1_SCREEN_SPEC.md's reasoning);
// no polling, since registered definitions change far less often than
// instance state.
//
// Rows are clickable again as of GUI Beta FE-1 — Workflow Detail
// (workflowDetail.ts) now exists, reversing the MVP-era decision to disable
// this to avoid a dead link.
import { html, render, TemplateResult } from "lit-html";
import { listWorkflows } from "../apiClient.js";
import type { WorkflowEntry } from "../types.js";
import { navigate } from "../router.js";
import { loadingView, emptyView, errorView } from "../ui.js";

interface State {
  namespace: string;
  loading: boolean;
  error: unknown;
  data: WorkflowEntry[] | null;
}

export function mountWorkflowList(container: HTMLElement): () => void {
  const state: State = { namespace: "", loading: true, error: null, data: null };
  let stopped = false;

  function draw(): void {
    render(view(state, actions), container);
  }

  async function load(): Promise<void> {
    state.loading = state.data === null;
    state.error = null;
    draw();
    try {
      const data = await listWorkflows(state.namespace || undefined);
      if (stopped) return;
      state.data = data;
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
    setNamespace(v: string): void {
      state.namespace = v;
      void load();
    },
    retry(): void {
      void load();
    },
    openWorkflow(id: string, version: string): void {
      navigate(`/workflows/${encodeURIComponent(id)}/${encodeURIComponent(version)}`);
    },
  };

  void load();

  return () => {
    stopped = true;
  };
}

interface Actions {
  setNamespace(v: string): void;
  retry(): void;
  openWorkflow(id: string, version: string): void;
}

function view(state: State, actions: Actions): TemplateResult {
  return html`
    <section class="screen">
      <h2>Workflows</h2>
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
      </div>
      ${bodyView(state, actions)}
    </section>
  `;
}

// bodyView follows GUI_PHASE1_SCREEN_SPEC.md §4's error-state contract
// ("same conventions as Instance List"): a transient failure shows the
// error banner ABOVE the last-successful table (stale but visible) when one
// exists.
function bodyView(state: State, actions: Actions): TemplateResult {
  if (state.loading) return loadingView("Loading workflows…");
  if (state.error) {
    return html`${errorView(state.error, actions.retry)}${state.data && state.data.length > 0
      ? dataView(state.data, actions)
      : ""}`;
  }
  if (!state.data || state.data.length === 0) {
    return emptyView("No workflows registered in this namespace.");
  }
  return dataView(state.data, actions);
}

function dataView(data: WorkflowEntry[], actions: Actions): TemplateResult {
  return html`
    <table class="data-table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Version</th>
          <th>Namespace</th>
          <th>Steps</th>
        </tr>
      </thead>
      <tbody>
        ${data.map(
          (w) => html`
            <tr class="clickable-row" @click=${() => actions.openWorkflow(w.id, w.version)}>
              <td>${w.id}</td>
              <td>${w.version}</td>
              <td>${w.namespace}</td>
              <td>${w.step_count}</td>
            </tr>
          `
        )}
      </tbody>
    </table>
  `;
}
