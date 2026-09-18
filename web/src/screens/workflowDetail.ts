// workflowDetail.ts — GUI Beta FE-1. Corrected version of the design in
// GUI_PHASE1_SCREEN_SPEC.md §5: steps/transitions/triggers as structured
// tables, explicitly not a graph/canvas. The correction (found during Beta
// planning, confirmed against real workflow data): a step's Details cell is
// type-specific — "intelligence" and "signal" steps carry no `handler`
// field at all and need their own field set rendered, not a blank Handler
// column.
import { html, render, TemplateResult } from "lit-html";
import { getWorkflow } from "../apiClient.js";
import type { Step, Transition, WorkflowDefinition } from "../types.js";
import { loadingView, errorView, notFoundView, isNotFound, errorMessage } from "../ui.js";

interface State {
  loading: boolean;
  error: unknown;
  data: WorkflowDefinition | null;
}

export function mountWorkflowDetail(container: HTMLElement, id: string, version: string): () => void {
  const state: State = { loading: true, error: null, data: null };
  let stopped = false;

  function draw(): void {
    render(view(state, actions, id, version), container);
  }

  async function load(): Promise<void> {
    state.loading = true;
    state.error = null;
    draw();
    try {
      const data = await getWorkflow(id, version);
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
    retry(): void {
      void load();
    },
  };

  void load();

  return () => {
    stopped = true;
  };
}

interface Actions {
  retry(): void;
}

function view(state: State, actions: Actions, id: string, version: string): TemplateResult {
  return html`
    <section class="screen">
      <h2>Workflow ${id} @ ${version}</h2>
      ${bodyView(state, actions)}
    </section>
  `;
}

function bodyView(state: State, actions: Actions): TemplateResult {
  if (state.error) {
    return isNotFound(state.error)
      ? notFoundView(`This workflow does not exist. (${errorMessage(state.error)})`)
      : errorView(state.error, actions.retry);
  }
  if (state.loading || !state.data) return loadingView("Loading workflow…");
  return dataView(state.data);
}

function dataView(def: WorkflowDefinition): TemplateResult {
  return html`
    <dl class="detail-grid">
      <dt>Name</dt>
      <dd>${def.name}</dd>
      <dt>Namespace</dt>
      <dd>${def.namespace}</dd>
      ${def.description
        ? html`<dt>Description</dt>
            <dd>${def.description}</dd>`
        : ""}
    </dl>

    <h3>Steps</h3>
    <table class="data-table">
      <thead>
        <tr>
          <th>Step ID</th>
          <th>Name</th>
          <th>Type</th>
          <th>Details</th>
        </tr>
      </thead>
      <tbody>
        ${def.steps.map((s) => stepRow(s, def))}
      </tbody>
    </table>

    <h3>Transitions</h3>
    ${def.transitions.length === 0
      ? html`<p class="state state-empty">No transitions (single-step workflow).</p>`
      : html`
          <table class="data-table">
            <thead>
              <tr>
                <th>From</th>
                <th>To</th>
                <th>Condition</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              ${def.transitions.map((t) => transitionRow(t))}
            </tbody>
          </table>
        `}

    <h3>Triggers</h3>
    <ul class="trigger-list">
      ${def.triggers.map(
        (t) => html`
          <li>
            <strong>${t.type}</strong>
            ${Object.keys(t.config).length > 0
              ? html`<pre class="json-block">${JSON.stringify(t.config, null, 2)}</pre>`
              : ""}
          </li>
        `
      )}
    </ul>
  `;
}

function stepRow(s: Step, def: WorkflowDefinition): TemplateResult {
  const badges: TemplateResult[] = [];
  if (s.id === def.initial_step) badges.push(html`<span class="badge badge-active">initial</span>`);
  if (def.final_steps.includes(s.id)) badges.push(html`<span class="badge badge-terminal">final</span>`);

  return html`
    <tr>
      <td class="mono">${s.id} ${badges}</td>
      <td>${s.name}</td>
      <td><span class="badge">${s.type}</span></td>
      <td>${stepDetails(s)}</td>
    </tr>
  `;
}

// stepDetails is the corrected, type-specific rendering this card exists
// for: "intelligence" and "signal" steps have no `handler` field at all, so
// a design that only ever shows Handler/Retry/Timeout/Fallback would render
// them as blank — confirmed against real data (with-intelligence.yaml,
// apps/oip/workflows/capture-decision.yaml).
//
// Fallback is rendered outside the per-type branch: internal/core/step.go's
// own doc comment on Step.Fallback ("...or its capability is unavailable")
// confirms it is meaningful on intelligence steps specifically, not just
// native/subprocess/plugin ones — it is a top-level Step field valid on any
// type, so it must not be silently dropped for the two type-specific
// branches.
function stepDetails(s: Step): TemplateResult {
  return html`
    <div class="step-details">
      ${typeSpecificDetails(s)}
      ${s.fallback ? html`<div>fallback: <span class="mono">${s.fallback}</span></div>` : ""}
    </div>
  `;
}

function typeSpecificDetails(s: Step): TemplateResult {
  if (s.type === "intelligence" && s.intelligence) {
    const iq = s.intelligence;
    return html`
      <div>capability: <strong>${iq.capability}</strong></div>
      ${iq.model_hint ? html`<div>model_hint: ${iq.model_hint}</div>` : ""}
      ${iq.context_budget ? html`<div>context_budget: ${iq.context_budget}</div>` : ""}
      ${iq.required ? html`<div>required: true</div>` : ""}
    `;
  }
  if (s.type === "signal" && s.wait_signal) {
    const ws = s.wait_signal;
    return html`
      <div>signal: <strong>${ws.signal_name}</strong></div>
      ${ws.timeout ? html`<div>timeout: ${ws.timeout}</div>` : ""}
      <div>timeout_action: ${ws.timeout_action}</div>
    `;
  }
  return html`
    ${s.handler ? html`<div>handler: <span class="mono">${s.handler}</span></div>` : ""}
    ${s.retry ? html`<div>retry: ${s.retry.attempts}× (${s.retry.backoff})</div>` : ""}
    ${s.timeout ? html`<div>timeout: ${s.timeout}</div>` : ""}
  `;
}

function transitionRow(t: Transition): TemplateResult {
  return html`
    <tr>
      <td class="mono">${t.from}</td>
      <td class="mono">${t.to}</td>
      <td>${t.condition || "—"}</td>
      <td>${t.on_error ? html`<span class="badge badge-terminal">on error</span>` : ""}</td>
    </tr>
  `;
}
