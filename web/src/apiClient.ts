// apiClient.ts — typed fetch wrappers for the 6 internal/api routes.
// One function per route; every error is typed so screens can distinguish a
// structured 404 (ApiNotFoundError) from a network/5xx failure (ApiError),
// per GUI_PHASE1_SCREEN_SPEC.md's per-screen error-state conventions.
import type {
  WorkflowEntry,
  WorkflowDefinition,
  InstanceListResponse,
  InstanceDetail,
  EventsPageResponse,
  InfoResponse,
} from "./types.js";

const BASE = "/api/v1";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "ApiError";
  }
}

export class ApiNotFoundError extends ApiError {
  constructor(message: string) {
    super(404, message);
    this.name = "ApiNotFoundError";
  }
}

async function getJSON<T>(path: string): Promise<T> {
  let res: Response;
  try {
    res = await fetch(BASE + path);
  } catch (err) {
    throw new ApiError(0, `network error: ${(err as Error).message}`);
  }
  if (!res.ok) {
    let message = `HTTP ${res.status}`;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch {
      // body wasn't JSON; keep the generic message
    }
    if (res.status === 404) throw new ApiNotFoundError(message);
    throw new ApiError(res.status, message);
  }
  return (await res.json()) as T;
}

// getInfo backs the connection-health/version indicator (GUI Beta, FE-6a/
// FE-6b). Deliberately calls /healthz's sibling /info, not /healthz itself
// — a pure liveness check should stay pure.
export function getInfo(): Promise<InfoResponse> {
  return getJSON<InfoResponse>("/info");
}

export function listWorkflows(namespace = ""): Promise<WorkflowEntry[]> {
  const q = namespace ? `?namespace=${encodeURIComponent(namespace)}` : "";
  return getJSON<WorkflowEntry[]>(`/workflows${q}`);
}

export function getWorkflow(id: string, version: string): Promise<WorkflowDefinition> {
  return getJSON<WorkflowDefinition>(
    `/workflows/${encodeURIComponent(id)}/${encodeURIComponent(version)}`
  );
}

export interface ListInstancesParams {
  namespace?: string;
  status?: string;
  // definitionId filters to instances of one workflow (GUI Beta, BE-1).
  definitionId?: string;
  limit?: number;
  offset?: number;
}

export function listInstances(params: ListInstancesParams = {}): Promise<InstanceListResponse> {
  const sp = new URLSearchParams();
  if (params.namespace) sp.set("namespace", params.namespace);
  if (params.status) sp.set("status", params.status);
  if (params.definitionId) sp.set("definition_id", params.definitionId);
  if (params.limit) sp.set("limit", String(params.limit));
  if (params.offset) sp.set("offset", String(params.offset));
  const qs = sp.toString();
  return getJSON<InstanceListResponse>(`/instances${qs ? "?" + qs : ""}`);
}

export function getInstance(id: string): Promise<InstanceDetail> {
  return getJSON<InstanceDetail>(`/instances/${encodeURIComponent(id)}`);
}

export interface ListEventsParams {
  from?: number;
  limit?: number;
}

export function listEvents(
  id: string,
  params: ListEventsParams = {}
): Promise<EventsPageResponse> {
  const sp = new URLSearchParams();
  if (params.from) sp.set("from", String(params.from));
  if (params.limit) sp.set("limit", String(params.limit));
  const qs = sp.toString();
  return getJSON<EventsPageResponse>(
    `/instances/${encodeURIComponent(id)}/events${qs ? "?" + qs : ""}`
  );
}
