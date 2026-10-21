// The HTTP contract of the adev server. Types mirror the Go DTOs: the form
// envelope plus the pieces the operator console needs.
const BASE = '/api/v1';

export type LogEntry = { kind: 'request' | 'ok' | 'error'; text: string };

export type TargetSystem = { id: number; name: string; description: string };

export type TargetSystemSummary = { id: number; name: string; description: string };

export type Readiness = { ready: boolean; missing?: string[] };

export type Stage = {
  id: string;
  name: string;
  owner: string;
  inputs: string[] | null;
  outputs: string[] | null;
  // A stage that reads another pipeline (the system baseline of a change request)
  // names its source; a conditional stage is a detour the straight run skips.
  inputSource?: string;
  conditional?: boolean;
};

export type Plan = {
  targetSystemId: number;
  route: string;
  stages: Stage[];
  readiness: Readiness;
};

export type Pipeline = { id: number; route: string; stage: string; readiness: Readiness };

export type PipelineRow = {
  id: number;
  targetsystemid: number;
  route: string;
  status: string;
  currentstage: string;
};

export type AgentView = {
  role: string;
  provider: string;
  baseUrl?: string;
  model: string;
  temperature: number;
  maxTokens: number;
  hasApiKey: boolean;
};

export type AgentInput = {
  provider: string;
  baseUrl: string;
  model: string;
  apiKey: string;
  temperature: number;
};

export type FormRef = { formId: string; formVersion: number };

// AgentActivity mirrors the live book of stage runs: what an agent works on now.
export type AgentActivity = {
  id: string;
  pipelineId: string;
  stage: string;
  role: string;
  output: string;
  attempts: number;
  status: string;
  startedAt: string;
  endedAt: string | null;
  lastError: string;
};

export type FormSummary = {
  formId: string;
  formType: string;
  formVersion: number;
  status: string;
  stage: string;
  payloadHash: string;
  producedBy: string;
  inputs: FormRef[] | null;
};

// The document list carries the payload of every version, so the viewer opens a
// document without a second request.
export type FormDetail = FormSummary & { payload: unknown };

export type StageOutcome = {
  stage: string;
  formId: string;
  formVersion: number;
  attempts: number;
  blocked: boolean;
  gaps: string[] | null;
};

// OperatorRequest is a question the model asked instead of guessing a form value.
// The goca control-plane table holds it; status "completed" means the operator answered.
export type OperatorRequest = {
  id: number;
  pipelineid: number;
  pointer: string;
  question: string;
  // The choices the model published as data; empty means a free answer.
  options?: string[] | null;
  blocking: boolean;
  status: string;
};

export type ModelCheckResult = {
  role: string;
  provider: string;
  model: string;
  hasApiKey: boolean;
  answer: string;
};

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// errorMessage reads the reason out of a failed response. The server answers with
// {"error": "..."} on its own routes and with plain text from http.Error.
async function errorMessage(response: Response): Promise<string> {
  const raw = await response.text();
  try {
    const parsed = JSON.parse(raw) as { error?: string };
    if (parsed.error) return parsed.error;
  } catch {
    // Not JSON: fall through to the plain text.
  }
  return raw.trim() || `${response.status} ${response.statusText}`;
}

export type Api = ReturnType<typeof createApi>;

export function createApi(onLog: (entry: LogEntry) => void) {
  async function call<T>(method: string, path: string, body?: unknown): Promise<T> {
    onLog({ kind: 'request', text: `${method} ${BASE}${path}${body === undefined ? '' : ' ' + JSON.stringify(body)}` });

    let response: Response;
    try {
      response = await fetch(BASE + path, {
        method,
        headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
        body: body === undefined ? undefined : JSON.stringify(body),
      });
    } catch (error) {
      const message = `сеть: ${String(error)}`;
      onLog({ kind: 'error', text: message });
      throw new ApiError(0, message);
    }

    if (!response.ok) {
      const message = await errorMessage(response);
      onLog({ kind: 'error', text: `${response.status} ${method} ${path}: ${message}` });
      throw new ApiError(response.status, message);
    }

    const raw = await response.text();
    const parsed = raw ? (JSON.parse(raw) as T) : (undefined as T);
    onLog({ kind: 'ok', text: `${response.status} ${method} ${path} ${raw.slice(0, 400)}` });
    return parsed;
  }

  return {
    createTargetSystem: (input: { name: string; description: string }) =>
      call<TargetSystem>('POST', '/targetsystems', input),
    // goca handlers marshal empty Go slices as null: every list is normalized to
    // an array here, the single HTTP layer, so no component sees null.
    listTargetSystems: () =>
      call<{ targetsystems: TargetSystemSummary[] | null }>('GET', '/targetsystems').then((list) => ({
        ...list,
        targetsystems: list.targetsystems ?? [],
      })),
    setWorkspace: (id: number, dir: string) => call<{ dir: string }>('PUT', `/targetsystems/${id}/workspace`, { dir }),
    setAgent: (id: number, role: string, agent: AgentInput) =>
      call<AgentView>('PUT', `/targetsystems/${id}/agents/${role}`, agent),
    listAgents: (id: number) => call<{ agents: AgentView[] }>('GET', `/targetsystems/${id}/agents`),
    checkAgent: (id: number, role: string, draft: AgentInput) =>
      call<ModelCheckResult>('POST', `/targetsystems/${id}/agents/${role}/check`, draft),
    plan: (id: number) => call<Plan>('GET', `/targetsystems/${id}/plan`),
    createPipeline: (id: number) => call<Pipeline>('POST', `/targetsystems/${id}/pipelines`, { route: 'greenfield' }),
    listPipelines: () =>
      call<{ pipelines: PipelineRow[] | null }>('GET', '/pipelines').then((list) => ({
        ...list,
        pipelines: list.pipelines ?? [],
      })),
    logs: (limit: number) =>
      call<{ lines: string[] | null; total: number }>('GET', `/logs?limit=${limit}`).then((result) => ({
        ...result,
        lines: result.lines ?? [],
      })),
    submitForm: (form: unknown) => call<FormDetail>('POST', '/forms', form),
    runStage: (id: number, stage: string, pipelineId: string) =>
      call<StageOutcome>('POST', `/targetsystems/${id}/stages/${stage}/run`, { pipelineId, route: 'greenfield' }),
    formsByPipeline: (pipelineId: string, filter?: { type?: string; stage?: string; status?: string }) => {
      const query = new URLSearchParams();
      for (const [name, value] of Object.entries(filter ?? {})) {
        if (value) query.set(name, value);
      }
      const suffix = query.toString() === '' ? '' : `?${query.toString()}`;
      return call<FormDetail[] | null>('GET', `/pipelines/${pipelineId}/forms${suffix}`).then((rows) => rows ?? []);
    },
    form: (formId: string, version: number) =>
      call<FormDetail>('GET', `/forms/${formId}/versions/${version}`),
    operatorRequests: () =>
      call<{ operatorrequests: OperatorRequest[]; total: number }>('GET', '/operatorrequests'),
    answerQuestion: (pipelineId: number, pointer: string, value: unknown) =>
      call<FormDetail>('POST', '/operatorrequests/answer', { pipelineId, pointer, value }),
    activities: () =>
      call<{ activities: AgentActivity[] | null; total: number }>('GET', '/activities').then((list) => ({
        ...list,
        activities: list.activities ?? [],
      })),
  };
}
