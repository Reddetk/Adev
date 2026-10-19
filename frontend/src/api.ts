// The HTTP contract of the adev server. Types mirror the Go DTOs: the form
// envelope plus the pieces the operator console needs.
const BASE = '/api/v1';

export type LogEntry = { kind: 'request' | 'ok' | 'error'; text: string };

export type TargetSystem = { id: number; name: string; description: string };

export type Readiness = { ready: boolean; missing?: string[] };

export type Stage = {
  id: string;
  name: string;
  owner: string;
  inputs: string[] | null;
  outputs: string[] | null;
};

export type Plan = {
  targetSystemId: number;
  route: string;
  stages: Stage[];
  readiness: Readiness;
};

export type Pipeline = { id: number; route: string; stage: string; readiness: Readiness };

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

export type FormSummary = {
  formId: string;
  formType: string;
  formVersion: number;
  status: string;
  payloadHash: string;
  producedBy: string;
  inputs: FormRef[] | null;
};

export type FormDetail = FormSummary & { payload: unknown };

export type StageOutcome = {
  stage: string;
  formId: string;
  formVersion: number;
  attempts: number;
  blocked: boolean;
  gaps: string[] | null;
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
    setWorkspace: (id: number, dir: string) => call<{ dir: string }>('PUT', `/targetsystems/${id}/workspace`, { dir }),
    setAgent: (id: number, role: string, agent: AgentInput) =>
      call<AgentView>('PUT', `/targetsystems/${id}/agents/${role}`, agent),
    listAgents: (id: number) => call<{ agents: AgentView[] }>('GET', `/targetsystems/${id}/agents`),
    checkAgent: (id: number, role: string) =>
      call<ModelCheckResult>('POST', `/targetsystems/${id}/agents/${role}/check`),
    plan: (id: number) => call<Plan>('GET', `/targetsystems/${id}/plan`),
    createPipeline: (id: number) => call<Pipeline>('POST', `/targetsystems/${id}/pipelines`, { route: 'greenfield' }),
    submitForm: (form: unknown) => call<FormDetail>('POST', '/forms', form),
    runStage: (id: number, stage: string, pipelineId: string) =>
      call<StageOutcome>('POST', `/targetsystems/${id}/stages/${stage}/run`, { pipelineId, route: 'greenfield' }),
    formsByPipeline: (pipelineId: string, type: string) =>
      call<FormSummary[]>('GET', `/pipelines/${pipelineId}/forms?type=${type}`),
    form: (formId: string, version: number) =>
      call<FormDetail>('GET', `/forms/${formId}/versions/${version}`),
  };
}
