// types.ts — hand-mirrored TypeScript interfaces for the 6 internal/api
// response shapes. Field names/optionality here MUST match
// docs/09-gui-planning/GUI_PHASE1_API_MATRIX.md and the real Go structs in
// internal/api/{workflows,instances,events}.go and internal/core/{workflow,
// step,event}.go exactly — this file is the client-side half of the
// contract those files define server-side.

export interface WorkflowEntry {
  id: string;
  version: string;
  namespace: string;
  step_count: number;
}

export interface Trigger {
  type: string;
  config: Record<string, unknown>;
}

export interface WaitConfig {
  signal_name: string;
  timeout?: string;
  timeout_action: string;
}

// IntelReq — internal/core/step.go:87-98. Present only on type="intelligence"
// steps, which carry NO handler field at all (GUI Beta finding: the original
// Phase 1 screen spec's Handler-only column design didn't account for this —
// confirmed against real workflows, examples/workflows/with-intelligence.yaml
// and apps/oip/workflows/capture-decision.yaml both use this step type).
export interface IntelReq {
  capability: string;
  model_hint?: string;
  context_budget?: number;
  required?: boolean;
}

// RetryPolicy — internal/core/step.go:58-70.
export interface RetryPolicy {
  attempts: number;
  backoff: string;
  initial_delay?: string;
  max_delay?: string;
  retryable_errors?: string[];
}

export interface Step {
  id: string;
  name: string;
  type: string;
  // handler has no json omitempty tag server-side (internal/core/step.go),
  // so it is always present in the JSON — but it's a meaningless empty
  // string "" on type="intelligence" and type="signal" steps (confirmed
  // live against GET /workflows/with-intelligence/1.0.0). Typed optional
  // here and treated as falsy in rendering code so both "" and a genuinely
  // absent value are handled the same way — do not assume every step has a
  // real handler.
  handler?: string;
  retry?: RetryPolicy;
  timeout?: string;
  fallback?: string;
  compensation?: unknown;
  wait_signal?: WaitConfig | null;
  intelligence?: IntelReq | null;
}

export interface Transition {
  from: string;
  to: string;
  condition?: string;
  on_error?: boolean;
}

export interface WorkflowDefinition {
  schema_version: number;
  id: string;
  version: string;
  namespace: string;
  name: string;
  description?: string;
  triggers: Trigger[];
  steps: Step[];
  transitions: Transition[];
  initial_step: string;
  final_steps: string[];
  compensation?: unknown;
  timeout?: string;
  metadata: Record<string, unknown>;
}

export interface InstanceEntry {
  instance_id: string;
  definition_id: string;
  version: string;
  status: string;
  current_steps: string[];
  created_at: string;
  updated_at: string;
}

export interface InstanceListResponse {
  instances: InstanceEntry[];
  total: number;
  limit: number;
  offset: number;
}

export interface InstanceDetail extends InstanceEntry {
  inputs?: Record<string, unknown>;
  outputs?: Record<string, unknown>;
  signal_name: string | null;
  timeout_remaining_s: number | null;
}

export interface ExecutionEvent {
  event_id: string;
  instance_id: string;
  namespace: string;
  event_type: string;
  step_id: string;
  payload: unknown;
  emitted_at: string;
  sequence_num: number;
  schema_version: number;
}

export interface EventsPageResponse {
  events: ExecutionEvent[] | null;
  next_cursor?: number;
}

// InfoResponse — GET /api/v1/info (GUI Beta, BE-2b).
export interface InfoResponse {
  version: string;
  go_version: string;
  uptime_s: number;
}

// The 9 InstanceStatus values (internal/core/instance.go) — a fixed set the
// status filter dropdown enumerates, not free text (the API does no fuzzy
// matching, per GUI_PHASE1_API_MATRIX.md's Instance List row).
export const INSTANCE_STATUSES = [
  "pending",
  "running",
  "waiting",
  "completed",
  "failed",
  "cancelled",
  "compensating",
  "compensated",
  "compensation_failed",
] as const;

export type InstanceStatus = (typeof INSTANCE_STATUSES)[number];

// Active vs. terminal — used to badge status visually (Instance List column,
// GUI_PHASE1_SCREEN_SPEC.md §1).
export const ACTIVE_STATUSES: ReadonlySet<string> = new Set([
  "pending",
  "running",
  "waiting",
  "compensating",
]);
