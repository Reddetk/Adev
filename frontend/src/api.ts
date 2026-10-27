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

// RunRow is a stage run the server accepted: the chain works in the background
// and the row moves running -> done | blocked | failed | stopped.
export type RunRow = {
  id: number;
  pipelineid: number;
  stage: string;
  status: string;
  checkpointid: string;
  attempts: number;
  lasterror: string;
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
  // The input control the field wants: choice, number, list or text (older rows: empty).
  format?: string;
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

// ModelCall is one row of the cost book: one attempt of one agent role.
// runid carries the pipeline id (shortcut: the run table stores no stage runs yet),
// so the console filters the history by pipeline and role. Every row also keeps
// its whole dialogue: the prompt the agent sent, the raw model answer, and the
// gate verdicts that rejected the draft.
export type ModelCall = {
  id: number;
  runid: number;
  profile: string;
  model: string;
  tokensin: number;
  tokensout: number;
  cost: number;
  latencyms: number;
  stage?: string;
  request?: string;
  response?: string;
  errors?: string;
};

// The stage names the operator reads. The registry keeps its English names; the
// display translates them once, so every tab says the same thing.
export const STAGE_TITLES: Record<string, string> = {
  '00': 'BRD — вопросы и границы системы',
  '01': 'Экономическое обоснование',
  '02': 'Требования (SRS)',
  '03': 'Дизайн верхнего уровня (HLD)',
  '04': 'Инварианты и ADR',
  '05': 'Дизайн нижнего уровня (LLD)',
  '06': 'План задач',
  '07': 'Тестовый контракт',
  '08': 'Спецификация реализации',
  '08.R': 'Спецификация рефакторинга',
  '09': 'Вердикт QA',
};

export function stageTitle(stage: Stage): string {
  return STAGE_TITLES[stage.id] ?? stage.name;
}

// The payloads the Target System tab reads straight from stored forms. The fields
// mirror domain/form_*.go; anything the gates did not fill stays undefined.
export type Text = { text: string };
export type SRSFr = {
  id: string;
  statement?: { text: string };
  actor?: string;
  trigger?: string;
  response?: string;
  stateChange?: string;
  priority?: string;
  traceGoal?: string;
  errors?: { code: string; condition: string; limit?: number }[] | null;
};
export type SRSNfr = {
  id: string;
  statement?: { text: string };
  metric?: string;
  unit?: string;
  threshold?: number;
  window?: string;
  traceGoal?: string;
};
export type SRSPayload = {
  fr?: SRSFr[] | null;
  nfr?: SRSNfr[] | null;
  outOfScope?: { text?: Text }[] | null;
  traceToGoals?: string[] | null;
};
export type HLDContext = { id: string; name: string; purpose?: Text; containerIds?: string[] | null };
export type HLDContainer = {
  objectId: string;
  purpose?: Text;
  boundedContextId?: string;
  tech?: string;
  nfrRefs?: string[] | null;
};
export type HLDRelation = { from: string; to: string; type?: string; contractId?: string };
export type HLDAggregate = {
  id: string;
  containerId: string;
  root: string;
  moduleRef?: string;
  invariants?: { text: string; modality?: string }[] | null;
};
export type HLDDegradation = { failure: string; degradation: string; detection?: string; timeoutMs?: number };
export type HLDPayload = {
  boundedContexts?: HLDContext[] | null;
  containers?: HLDContainer[] | null;
  aggregates?: HLDAggregate[] | null;
  contextRelations?: HLDRelation[] | null;
  degradationStrategy?: HLDDegradation[] | null;
};
export type LLDModule = { objectId: string; path: string; deps?: { providerObjectId: string; type?: string }[] | null };
export type LLDLayer = { name: string; packages?: { path: string; modules?: LLDModule[] | null }[] | null };
export type LLDContainer = { containerId: string; layers?: LLDLayer[] | null };
export type LLDContract = {
  id: string;
  moduleRef?: { formId?: string; formType?: string; formVersion?: number; pointer?: string };
  kind?: string;
  inputType?: string;
  outputType?: string;
  errors?: { code: string; condition: string }[] | null;
};
export type LLDPayload = {
  containers?: LLDContainer[] | null;
  contracts?: LLDContract[] | null;
};

// PointerChoice is one entry an operator can pick for a question about a list.
export type PointerChoice = { pointer: string; label: string };

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
    readonly choices: PointerChoice[] = [],
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// readError reads the reason out of a failed response. The server answers with
// {"error": "...", "choices": [...]} on its own routes and with plain text from http.Error.
async function readError(response: Response): Promise<{ message: string; choices: PointerChoice[] }> {
  const raw = await response.text();
  try {
    const parsed = JSON.parse(raw) as { error?: string; choices?: PointerChoice[] };
    if (parsed.error) return { message: parsed.error, choices: parsed.choices ?? [] };
  } catch {
    // Not JSON: fall through to the plain text.
  }
  return { message: raw.trim() || `${response.status} ${response.statusText}`, choices: [] };
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
      const { message, choices } = await readError(response);
      onLog({ kind: 'error', text: `${response.status} ${method} ${path}: ${message}` });
      throw new ApiError(response.status, message, choices);
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
    deletePipeline: (id: number) => call<void>('DELETE', `/pipelines/${id}`),
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
    deleteForm: (formId: string) => call<void>('DELETE', `/forms/${formId}`),
    deleteFormVersion: (formId: string, version: number) => call<void>('DELETE', `/forms/${formId}/versions/${version}`),
    updateFormPayload: (formId: string, version: number, payload: unknown) =>
      call<FormDetail>('PUT', `/forms/${formId}/versions/${version}/payload`, { payload }),
    // The answer is the accepted run row at once: the chain goes in the background.
    runStage: (id: number, stage: string, pipelineId: string) =>
      call<RunRow>('POST', `/targetsystems/${id}/stages/${stage}/run`, { pipelineId, route: 'greenfield' }),
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
    answerQuestion: (pipelineId: number, pointer: string, value: unknown, asked?: string) =>
      call<FormDetail>('POST', '/operatorrequests/answer', { pipelineId, pointer, value, asked }),
    activities: () =>
      call<{ activities: AgentActivity[] | null; total: number }>('GET', '/activities').then((list) => ({
        ...list,
        activities: list.activities ?? [],
      })),
    modelCalls: () =>
      call<{ modelcalls: ModelCall[] | null; total: number }>('GET', '/modelcalls').then((list) => ({
        ...list,
        modelcalls: list.modelcalls ?? [],
      })),
    // An empty role stops every agent of the pipeline. The run answers with the
    // number of stage runs the stop reached.
    stopAgent: (pipelineId: string, role: string) =>
      call<{ cancelled: number }>('POST', '/runs/stop', { pipelineId, role }),
  };
}
