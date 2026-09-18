// eventFormatters.ts — one-line summaries for the 12 known event types, per
// docs/09-gui-planning/GUI_PHASE1_API_MATRIX.md's payload table (itself
// copied verbatim from internal/core/event.go's EventType doc comments).
// An unrecognized event_type falls back to the raw label — never throws —
// per GUI_PHASE1_SCREEN_SPEC.md §3's forward-compatibility requirement.

type Payload = Record<string, unknown>;
type Formatter = (payload: Payload, stepId: string) => string;

function str(v: unknown, fallback = "?"): string {
  return typeof v === "string" || typeof v === "number" ? String(v) : fallback;
}

const FORMATTERS: Record<string, Formatter> = {
  WorkflowStarted: () => "Workflow started",
  StepStarted: (p, stepId) => `Step ${stepId} started (attempt ${str(p.attempt)})`,
  StepCompleted: (p, stepId) => `Step ${stepId} completed in ${str(p.duration_ms)}ms`,
  StepFailed: (p, stepId) =>
    `Step ${stepId} failed: ${str(p.error, "unknown error")}` + (p.retrying ? " (retrying)" : ""),
  StepFallbackActivated: (p, stepId) =>
    `Step ${stepId} fell back to ${str(p.fallback_step_id)}: ${str(p.reason, "")}`,
  SignalReceived: (p) => `Signal '${str(p.signal_name)}' received`,
  WorkflowCompleted: (p) => `Workflow completed in ${str(p.duration_ms)}ms`,
  WorkflowFailed: (p, stepId) => `Workflow failed at ${stepId || str(p.step_id)}: ${str(p.error, "")}`,
  WorkflowCancelled: (p) => `Workflow cancelled: ${str(p.reason, "")}`,
  WorkflowCompensating: (p) => `Compensation started from ${str(p.from_step)}`,
  WorkflowCompensated: () => "Compensation completed",
  WorkflowCompensationFailed: (p, stepId) =>
    `Compensation failed at ${stepId || str(p.step_id)}: ${str(p.error, "")}`,
};

export function summarizeEvent(eventType: string, stepId: string, payload: unknown): string {
  const fn = FORMATTERS[eventType];
  if (!fn) return eventType;
  try {
    return fn((payload as Payload) ?? {}, stepId);
  } catch {
    return eventType;
  }
}
