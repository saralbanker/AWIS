// onboarding.ts — GUI Beta FE-5. A genuinely distinct component (not the
// generic emptyView with swapped-in text — a reused component would not
// actually satisfy "distinguishable at a glance," GUI_BETA_PRD.md F5) shown
// only on true first-run: both /workflows and /instances are empty, and no
// filter is narrowing the view (an overly narrow filter must show the
// ordinary "no results" state, not this one — GUI_BETA_PRD.md's explicit
// acceptance criterion).
import { html, TemplateResult } from "lit-html";
import type { WorkflowEntry } from "./types.js";

export function onboardingView(workflows: WorkflowEntry[]): TemplateResult {
  // The example command uses a real, currently-registered workflow id when
  // one exists (zero new API calls — workflows is already fetched by the
  // caller), falling back to a generic awis init pointer only when
  // /workflows is also genuinely empty.
  const example = workflows[0];
  return html`
    <div class="onboarding">
      <h3>Welcome to AWIS</h3>
      <p>
        Nothing has run here yet — this GUI is read-only, so workflows are still created
        and started from the command line.
      </p>
      ${example
        ? html`<pre class="json-block">awis submit ${example.id}</pre>`
        : html`<pre class="json-block">awis init
awis submit hello-world</pre>`}
      <p class="onboarding-note">This page updates automatically once an instance exists.</p>
    </div>
  `;
}
