import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { FormDocument } from './FormDocument';
import { PipelineGraph, type NodeState } from './PipelineGraph';
import { QuestionsDashboard } from './QuestionsDashboard';
import { AgentsDashboard } from './AgentsDashboard';
import { TargetSystemTab } from './TargetSystemTab';
import { stageTitle, type PointerChoice } from './api';
import {
  ApiError,
  createApi,
  type AgentActivity,
  type AgentInput,
  type FormDetail,
  type LogEntry,
  type OperatorRequest,
  type PipelineRow,
  type Plan,
  type RunRow,
  type TargetSystemSummary,
} from './api';

const PROVIDERS = ['ollama', 'openrouter', 'opencode', 'openai', 'deepseek', 'qwen'];
const STAGE_INPUT = '00';
const ROUTE_GREENFIELD = 'greenfield';

// The paragraph titles of every form, taken from form-registry/*.md. A stage that
// has not run yet shows its document outline: the operator sees what to expect.
const DOCUMENT_OUTLINE: Record<string, string[]> = {
  USER_INPUT: ['Request source', 'Requested result', 'Evidence'],
  BRD: [
    'Business outcome',
    'Project boundary',
    'Actors and tasks',
    'Constraints and assumptions',
    'Success measurement',
    'Missing data',
  ],
  FEASIBILITY: ['Option set', 'Cost and schedule', 'Recommendation', 'BRD correction'],
  SRS: [
    'Functional requirement',
    'Acceptance scenario',
    'Error and boundary behavior',
    'Non-functional requirement',
    'Out of scope',
    'Traceability',
  ],
  HLD: [
    'Ubiquitous language',
    'Bounded context',
    'Container',
    'Aggregate',
    'Context relation',
    'Invariant',
    'Degradation strategy',
  ],
  ADR: ['Decision identity', 'Context', 'Decision', 'Alternatives', 'Consequences', 'Links'],
  LLD: [
    'Container layout',
    'Module dependency',
    'Contract',
    'Error catalog',
    'Happy path',
    'Negative path',
    'Event map',
    'Decision link',
  ],
  TASK_PLAN: [
    'Task identity',
    'Tree position',
    'Owner',
    'Inputs',
    'Outputs',
    'Acceptance',
    'Dependencies',
  ],
  FR: ['Requirement identity', 'Statement', 'Priority', 'Acceptance scenario', 'Traceability'],
  NFR: ['Requirement identity', 'Metric', 'Unit and threshold', 'Measurement point', 'Window'],
  TEST_CONTRACT: ['Test identity', 'Given', 'When', 'Then', 'Expected error', 'Coverage'],
  IMPLEMENTATION_SPEC: [
    'Task reference',
    'Implementation target',
    'Step',
    'Constraint',
    'Test map',
  ],
  VERDICT: ['Run identity', 'Test result', 'Violation', 'Verdict', 'Evidence'],
  REFACTOR_SPEC: ['Verdict reference', 'Violation to close', 'Allowed files', 'Step', 'Tests to keep'],
  CR_BRIEF: ['Change type', 'Driver', 'Current behaviour', 'Requested behaviour', 'Missing data'],
  IMPACT_ASSESSMENT: [
    'Blast radius',
    'Contract impact',
    'NFR baseline delta',
    'Routing rule',
    'Missing data',
  ],
};

type AgentDraft = {
  role: string;
  provider: string;
  baseUrl: string;
  model: string;
  apiKey: string;
  temperature: string;
};

// Every role of the route needs a valid agent: the plan names the owners, and a
// role without a config keeps the pipeline not ready.
function defaultAgent(role: string): AgentDraft {
  return {
    role,
    provider: 'ollama',
    baseUrl: 'http://localhost:11434',
    model: 'qwen3-coder:30b',
    apiKey: '',
    temperature: '0.2',
  };
}

const initialAgents: AgentDraft[] = ['analyst', 'architect', 'qa', 'dev'].map(defaultAgent);

// The console remembers the operator input, so a reload does not lose the setup.
// The api key is not kept in the browser: the server already holds it and an empty
// field means "keep the saved key".
const STORAGE_KEY = 'adev.console';

type SavedState = {
  targetSystemId: number | null;
  pipeline: PipelineChoice | null;
  view: View;
  name: string;
  description: string;
  workspaceDir: string;
  agents: Omit<AgentDraft, 'apiKey'>[];
  request: string;
};

// View is the tab the operator works in. The target system overview opens first.
type View = 'targetsystem' | 'pipeline' | 'settings';

// PipelineChoice is the pipeline the console watches.
type PipelineChoice = { id: number; stage: string; route: string; status: string };

function loadSaved(): Partial<SavedState> {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as Partial<SavedState>) : {};
  } catch {
    return {};
  }
}

export default function App() {
  const [entries, setEntries] = useState<LogEntry[]>([
    { kind: 'request', text: 'Готово. Выберите стадию на схеме — покажу её документ.' },
  ]);
  const api = useMemo(() => createApi((entry) => setEntries((previous) => [...previous, entry])), []);

  const saved = useMemo(loadSaved, []);
  const [targetSystem, setTargetSystem] = useState<{ id: number } | null>(
    saved.targetSystemId ? { id: saved.targetSystemId } : null,
  );
  const [name, setName] = useState(saved.name ?? 'library');
  const [description, setDescription] = useState(saved.description ?? 'Library accounting system');
  const [workspaceDir, setWorkspaceDir] = useState(saved.workspaceDir ?? 'D:\\work\\library');
  const [workspace, setWorkspace] = useState('');
  const [agents, setAgents] = useState<AgentDraft[]>(() =>
    saved.agents?.length ? saved.agents.map((agent) => ({ ...agent, apiKey: '' })) : initialAgents,
  );
  const [checks, setChecks] = useState<Record<string, string>>({});
  const [systems, setSystems] = useState<TargetSystemSummary[]>([]);
  const [checking, setChecking] = useState('');
  const [agentsSaved, setAgentsSaved] = useState<string[]>([]);
  const [plan, setPlan] = useState<Plan | null>(null);
  const [pipeline, setPipeline] = useState<PipelineChoice | null>(
    saved.pipeline
      ? {
          id: saved.pipeline.id,
          stage: saved.pipeline.stage ?? '00',
          route: saved.pipeline.route ?? 'greenfield',
          status: saved.pipeline.status ?? 'ready',
        }
      : null,
  );
  const [pipelines, setPipelines] = useState<PipelineRow[]>([]);
  // An older console stored different tab names: anything unknown opens the dashboard.
  const [view, setView] = useState<View>(() => {
    const restored: string = saved.view ?? '';
    return restored === 'targetsystem' || restored === 'pipeline' || restored === 'settings'
      ? restored
      : 'targetsystem';
  });
  const [request, setRequest] = useState(
    saved.request ?? 'The operator needs a library accounting system for two branches.',
  );
  const [inputForm, setInputForm] = useState<FormDetail | null>(null);
  const [selectedStage, setSelectedStage] = useState<string | null>(null);
  const [formsByType, setFormsByType] = useState<Record<string, FormDetail[]>>({});
  const [opened, setOpened] = useState<FormDetail | null>(null);
  const [rawPayload, setRawPayload] = useState(false);
  const [accepted, setAccepted] = useState<RunRow | null>(null);
  const [runError, setRunError] = useState('');
  const [running, setRunning] = useState(false);
  const [questions, setQuestions] = useState<OperatorRequest[]>([]);
  const [activities, setActivities] = useState<AgentActivity[]>([]);
  const [activitiesLoading, setActivitiesLoading] = useState(true);
  const [questionsLoading, setQuestionsLoading] = useState(false);
  const [questionsError, setQuestionsError] = useState('');
  // A question whose pointer names a list with several entries waits for the operator's pick.
  const [choice, setChoice] = useState<{
    asked: string;
    value: unknown;
    pipelineId: number;
    choices: PointerChoice[];
  } | null>(null);
  const [panel, setPanel] = useState<'journal' | 'server'>('journal');
  const [serverLines, setServerLines] = useState<string[]>([]);

  const targetSystemId = targetSystem?.id ?? 0;
  const pipelineId = pipeline ? String(pipeline.id) : '';

  // Keep the operator input across reloads.
  useEffect(() => {
    const state: SavedState = {
      targetSystemId: targetSystem?.id ?? null,
      pipeline: pipeline ?? null,
      view,
      name,
      description,
      workspaceDir,
      agents: agents.map(({ role, provider, baseUrl, model, temperature }) => ({
        role,
        provider,
        baseUrl,
        model,
        temperature,
      })),
      request,
    };
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch {
      // A browser in private mode has no storage. The console still works.
    }
  }, [targetSystem, pipeline, view, name, description, workspaceDir, agents, request]);
  // A restored target system has agent configs on the server: show which keys are set.
  useEffect(() => {
    if (!targetSystemId) return;
    void api
      .listAgents(targetSystemId)
      .then((list) =>
        setAgentsSaved(
          list.agents.length === 0
            ? ['агенты не заданы: заполните поля ниже и нажмите «Сохранить агентов»']
            : list.agents.map(
                (agent) =>
                  `${agent.role} · ${agent.provider} · ${agent.model} · ключ: ${agent.hasApiKey ? 'задан' : 'нет'}`,
              ),
        ),
      )
      .catch((error) => {
        setAgentsSaved(
          error instanceof ApiError && error.status === 409
            ? ['агенты не заданы: заполните поля ниже и нажмите «Сохранить агентов»']
            : [`не удалось прочитать агентов: ${String(error)}`],
        );
      });
  }, [api, targetSystemId]);

  // The list lets the operator switch to an already configured target system.
  useEffect(() => {
    void api
      .listTargetSystems()
      .then((list) => {
        setSystems(list.targetsystems);
        // A saved system that no longer exists would send every request to nothing.
        if (targetSystemId && !list.targetsystems.some((system) => system.id === targetSystemId)) {
          const first = list.targetsystems[0];
          setTargetSystem(first ? { id: first.id } : null);
        }
      })
      .catch(() => undefined);
  }, [api, targetSystemId]);

  // The dashboard and the document list watch every pipeline, not only the ones of
  // the selected system; the plan and its readiness belong to the system.
  useEffect(() => {
    if (view === 'settings') return;
    void refreshPipelines().catch(() => setPipelines([]));
    if (!targetSystemId) {
      setPlan(null);
      return;
    }
    void api.plan(targetSystemId).then(setPlan).catch(() => setPlan(null));
  }, [api, view, targetSystemId]);

  // A route names the roles it needs: a role without a form is appended, so the
  // operator configures it and the readiness check can turn ready.
  useEffect(() => {
    const owners = new Set((plan?.stages ?? []).map((stage) => stage.owner));
    if (owners.size === 0) return;
    setAgents((previous) => [
      ...previous,
      ...[...owners]
        .filter((role) => !previous.some((agent) => agent.role === role))
        .map(defaultAgent),
    ]);
  }, [plan]);

  // The node state comes from the stored forms of the watched pipeline; both the
  // pipeline view and the target system overview read it.
  useEffect(() => {
    if (view === 'settings' || !pipelineId) {
      setFormsByType({});
      return;
    }
    void loadPipelineForms(pipelineId);
  }, [api, view, pipelineId, plan]);

  // The operator questions and the agent board live in the pipeline tab: a model
  // asks and the stage stops until the operator answers.
  useEffect(() => {
    if (view !== 'pipeline') return;
    void loadQuestions();
    const timer = window.setInterval(() => void loadQuestions(), 10000);
    return () => window.clearInterval(timer);
  }, [api, view]);

  // The live board of stage runs feeds the pipeline tab while it is open.
  useEffect(() => {
    if (view !== 'pipeline') return;
    void loadActivities();
    const timer = window.setInterval(() => void loadActivities(), 5000);
    return () => window.clearInterval(timer);
  }, [api, view]);

  // A run ends out of band now: the chain works in the background, so when the
  // board loses a running id, the forms, the questions and the cursor may all
  // have moved. One finish can chain several stages; one reload covers them all.
  const lastRunning = useRef('');
  useEffect(() => {
    const ids = activities
      .filter((item) => item?.status === 'running')
      .map((item) => item.id)
      .join(',');
    const lost = lastRunning.current
      ? lastRunning.current
          .split(',')
          .filter(Boolean)
          .filter((id) => !ids.split(',').includes(id))
      : [];
    lastRunning.current = ids;
    if (lost.length === 0) return;
    void refreshPipelines();
    if (pipelineId) {
      void loadPipelineForms(pipelineId);
      void loadQuestions();
    }
  }, [activities]); // eslint-disable-line react-hooks/exhaustive-deps

  // The server log panel polls while it is open: the server writes lines that never
  // reach this browser any other way.
  useEffect(() => {
    if (panel !== 'server') {
      return;
    }
    let stopped = false;
    const load = () => {
      void api
        .logs(300)
        .then((result) => {
          if (!stopped) setServerLines(result.lines);
        })
        .catch(() => undefined);
    };
    load();
    const timer = window.setInterval(load, 5000);
    return () => {
      stopped = true;
      window.clearInterval(timer);
    };
  }, [api, panel]);

  // Switching the system drops everything that belongs to the old one.
  function switchTarget(id: number) {
    setTargetSystem({ id });
    setPipeline(null);
    setPlan(null);
    setFormsByType({});
    setOpened(null);
    setSelectedStage(null);
    setChecks({});
    setAgentsSaved([]);
  }

  function failureHint(error: unknown): string {
    if (error instanceof ApiError && error.status === 409) {
      return ' — сохраните агентов в разделе 2';
    }
    return '';
  }

  async function checkAgent(role: string) {
    const agent = agents.find((item) => item.role === role);
    if (!agent) return;
    setChecking(role);
    try {
      // Send the fields as typed: the saved key comes back only as hasApiKey, so a
      // draft is the only way to try a model before saving it.
      const result = await api.checkAgent(targetSystemId, role, {
        provider: agent.provider,
        baseUrl: agent.baseUrl,
        model: agent.model,
        apiKey: agent.apiKey,
        temperature: Number.parseFloat(agent.temperature) || 0,
      });
      setChecks((previous) => ({
        ...previous,
        [role]: `ответ модели · ${result.model} · ${result.answer.slice(0, 200)}`,
      }));
    } catch (error) {
      setChecks((previous) => ({ ...previous, [role]: `ошибка: ${String(error)}${failureHint(error)}` }));
    } finally {
      setChecking('');
    }
  }

  async function createTargetSystem() {
    try {
      setTargetSystem(await api.createTargetSystem({ name, description }));
    } catch {
      setTargetSystem(null);
    }
  }

  async function saveWorkspace() {
    if (!targetSystemId) return;
    const savedDir = await api.setWorkspace(targetSystemId, workspaceDir).catch(() => null);
    if (savedDir) setWorkspace(savedDir.dir);
  }

  async function saveAgents() {
    if (!targetSystemId) return;
    const lines: string[] = [];
    for (const agent of agents) {
      const input: AgentInput = {
        provider: agent.provider,
        baseUrl: agent.baseUrl,
        model: agent.model,
        apiKey: agent.apiKey,
        temperature: Number.parseFloat(agent.temperature) || 0,
      };
      try {
        const agentView = await api.setAgent(targetSystemId, agent.role, input);
        lines.push(`${agent.role}: сохранён, hasApiKey=${agentView.hasApiKey}`);
      } catch (error) {
        lines.push(`${agent.role}: ошибка — ${String(error)}`);
      }
    }
    setAgentsSaved(lines);
  }

  async function loadPlan() {
    if (!targetSystemId) return;
    setPlan(await api.plan(targetSystemId).catch(() => null));
  }

  async function createPipeline() {
    if (!targetSystemId) return;
    const created = await api.createPipeline(targetSystemId).catch(() => null);
    if (created) {
      setPipeline({ id: created.id, stage: created.stage, route: created.route, status: 'ready' });
      setPipelines((previous) => [
        ...previous,
        { id: created.id, targetsystemid: targetSystemId, route: created.route, status: 'ready', currentstage: created.stage },
      ]);
    }
  }

  // deletePipeline removes the run for good with its forms, questions and runs. A
  // running stage refuses it on the server: the operator stops it first.
  async function deletePipeline(row: PipelineRow) {
    if (!window.confirm(`Удалить пайплайн #${row.id} вместе с формами и вопросами? Это необратимо.`)) return;
    try {
      await api.deletePipeline(row.id);
    } catch (error) {
      window.alert(error instanceof Error ? error.message : 'Пайплайн не удалён');
      return;
    }
    if (pipeline?.id === row.id) setPipeline(null);
    await refreshPipelines().catch(() => setPipelines([]));
  }

  // choosePipeline switches the console to one run and drops the view of the old one.
  function choosePipeline(row: PipelineRow) {
    setPipeline({ id: row.id, stage: row.currentstage, route: row.route, status: row.status });
    setOpened(null);
    setSelectedStage(null);
    setAccepted(null);
    setInputForm(null);
  }

  // One request reads every document of the pipeline; the graph and the stage card
  // read the latest version of each type from the grouped list.
  async function loadPipelineForms(forPipeline: string) {
    const list = await api.formsByPipeline(forPipeline).catch(() => [] as FormDetail[]);
    const grouped: Record<string, FormDetail[]> = {};
    for (const form of list) {
      if (!grouped[form.formType]) grouped[form.formType] = [];
      grouped[form.formType].push(form);
    }
    setFormsByType(grouped);
  }

  async function loadQuestions() {
    setQuestionsLoading(true);
    try {
      const result = await api.operatorRequests();
      setQuestions(result.operatorrequests ?? []);
      setQuestionsError('');
    } catch (error) {
      setQuestionsError(`не удалось прочитать вопросы: ${String(error)}`);
    } finally {
      setQuestionsLoading(false);
    }
  }

  async function loadActivities() {
    setActivitiesLoading(true);
    try {
      const result = await api.activities();
      setActivities(result.activities);
    } catch {
      // the live board keeps the last snapshot; the poll retries in 5 seconds
    } finally {
      setActivitiesLoading(false);
    }
  }

  function selectStage(stageID: string) {
    setSelectedStage(stageID);
    setAccepted(null);
    setRunError('');
    const stage = plan?.stages.find((item) => item.id === stageID);
    const type = stage?.outputs?.[0];
    setOpened(type ? latestForm(type) : null);
  }

  // The list already carries the payload: opening a version needs no request.
  function latestForm(type: string): FormDetail | null {
    const list = formsByType[type] ?? [];
    return list.length > 0 ? list[list.length - 1] : null;
  }

  // The state of the document an activity produced. Only the forms of the selected
  // pipeline are loaded: another pipeline shows its runs without a document state.
  function documentStatus(activity: AgentActivity): string | undefined {
    if (activity.pipelineId !== pipelineId) return undefined;
    return latestForm(activity.output)?.status;
  }

  async function submitInput() {
    if (!targetSystemId || !pipelineId) return;
    const form = await api
      .submitForm({
        formId: `FRM-IN-${pipelineId}`,
        formType: 'USER_INPUT',
        schemaVersion: 1,
        formVersion: (formsByType.USER_INPUT?.length ?? 0) + 1,
        previousFormVersion: (formsByType.USER_INPUT?.length ?? 0) > 0
          ? { formId: `FRM-IN-${pipelineId}`, formType: 'USER_INPUT', formVersion: formsByType.USER_INPUT!.length }
          : undefined,
        status: 'draft',
        route: ROUTE_GREENFIELD,
        // The stage of an operator form is the em dash, as the registry defines.
        stage: '\u2014',
        mode: 'full',
        pipelineId,
        targetSystemId: String(targetSystemId),
        producedBy: 'operator',
        inputs: [],
        payload: {
          rawRequest: { text: request, lang: 'STE100' },
          source: 'operator',
          receivedAt: new Date().toISOString(),
        },
      })
      .catch((error) => {
        setRunError(`USER_INPUT не сохранён: ${String(error)}`);
        return null;
      });
    if (form) {
      setInputForm(form);
      setOpened(form);
      await loadPipelineForms(pipelineId);
    }
  }

  // refreshPipelines re-reads the cursor: a chain moves it in the background, and
  // the shared logic also drops a pipeline the database lost.
  async function refreshPipelines() {
    const list = await api.listPipelines().catch(() => null);
    if (!list) return;
    const rows = list.pipelines ?? [];
    setPipelines(rows);
    setPipeline((previous) => {
      if (!previous) return previous;
      const row = rows.find((item) => item.id === previous.id);
      return row ? { ...previous, stage: row.currentstage, status: row.status } : null;
    });
  }

  async function runStage(stageID: string) {
    if (!targetSystemId || !pipelineId) return;
    setRunning(true);
    setRunError('');
    setAccepted(null);
    try {
      const run = await api.runStage(targetSystemId, stageID, pipelineId);
      // The run is accepted: the chain works in the background now. The board
      // card, the questions and the forms update themselves from the polls; the
      // cursor and the documents move on their own when the chain ends.
      setAccepted(run);
      void loadQuestions();
      await loadActivities();
    } catch (error) {
      setRunError(`${String(error)}${failureHint(error)}`);
    } finally {
      setRunning(false);
    }
  }

  const ready = targetSystemId > 0;
  const hasPipeline = pipeline !== null;
  const configured = plan?.readiness.ready ?? false;
  const currentSystem = systems.find((system) => system.id === targetSystemId);
  const openQuestions = questions.filter((question) => question.status !== 'completed').length;

  // Node states: the stored form of the stage output wins over the pipeline cursor,
  // and a running agent wins over both: the stage is visible as working even
  // before its first form exists.
  const runningStages = new Set(
    activities.filter((activity) => activity?.status === 'running').map((activity) => activity.stage),
  );
  const nodeStates: Record<string, NodeState> = {};
  for (const stage of plan?.stages ?? []) {
    if (runningStages.has(stage.id)) {
      nodeStates[stage.id] = 'running';
      continue;
    }
    const type = stage.outputs?.[0];
    const versions = type ? formsByType[type] ?? [] : [];
    if (versions.length > 0) {
      const status = versions[versions.length - 1].status as NodeState;
      nodeStates[stage.id] = status;
    } else {
      nodeStates[stage.id] = pipeline?.stage === stage.id ? 'current' : 'planned';
    }
  }

  return (
    <>
      <header>
        <h1>adev · консоль оператора</h1>
        <div className="tabs">
          <button className={view === 'targetsystem' ? '' : 'ghost'} onClick={() => setView('targetsystem')}>
            Target System
          </button>
          <button className={view === 'pipeline' ? '' : 'ghost'} onClick={() => setView('pipeline')}>
            Пайплайн{openQuestions > 0 ? ` · ${openQuestions}` : ''}
          </button>
          <button className={view === 'settings' ? '' : 'ghost'} onClick={() => setView('settings')}>
            Настройки
          </button>
          <span className="hint">
            {targetSystemId
              ? `target system ${targetSystemId}${currentSystem ? ' · ' + currentSystem.name : ''}`
              : 'target system не выбран'}
          </span>
        </div>
      </header>

      <main>
        <div>
          {view === 'settings'
            ? settingsView()
            : view === 'targetsystem'
              ? <TargetSystemTab api={api} pipelineId={pipelineId} />
              : pipelineView()}
        </div>

        <details className="console">
          <summary>Журнал и логи</summary>
          <ConsolePanel entries={entries} lines={serverLines} panel={panel} onPanel={setPanel} />
        </details>
      </main>
    </>
  );

  function settingsView() {
    return (
      <>
        <Section title="1. Target system" hint="Мнимая система, которую разрабатывает adev. GitHub-репозиторий не нужен.">
          <Field label="Уже созданные">
            <select
              value={targetSystemId || ''}
              onChange={(event) => event.target.value && switchTarget(Number(event.target.value))}
            >
              <option value="">— не выбран —</option>
              {systems.map((system) => (
                <option key={system.id} value={system.id}>
                  {system.id} · {system.name}
                </option>
              ))}
            </select>
          </Field>
          <div className="row">
            <Field label="Имя">
              <input value={name} onChange={(event) => setName(event.target.value)} />
            </Field>
            <Field label="Описание">
              <input value={description} onChange={(event) => setDescription(event.target.value)} />
            </Field>
          </div>
          <Actions>
            <button onClick={createTargetSystem}>Создать</button>
          </Actions>
          <Cards>
            {targetSystem && (
              <Card>
                работаем с id=<b>{targetSystem.id}</b> <Badge tone="ok">выбран</Badge>
              </Card>
            )}
          </Cards>
        </Section>

        <Section
          title="2. Окружение targetSystem"
          hint="Рабочая директория и модели агентов. У облачных провайдеров baseUrl можно оставить пустым — подставится адрес API; ключ уходит на сервер и никогда не возвращается."
        >
          <Field label="Workspace directory">
            <input value={workspaceDir} onChange={(event) => setWorkspaceDir(event.target.value)} />
          </Field>
          <Actions>
            <button disabled={!ready} onClick={saveWorkspace}>
              Сохранить директорию
            </button>
          </Actions>
          <Cards>{workspace && <Card>workspace: {workspace}</Card>}</Cards>

          {agents.map((agent, index) => (
            <div key={agent.role}>
              <Field label={`${agent.role} · provider`}>
                <div className="row three">
                  <select
                    value={agent.provider}
                    onChange={(event) => updateAgent(index, { provider: event.target.value })}
                  >
                    {PROVIDERS.map((provider) => (
                      <option key={provider} value={provider}>
                        {provider}
                      </option>
                    ))}
                  </select>
                  <input
                    placeholder="baseUrl (облачным не нужен)"
                    value={agent.baseUrl}
                    onChange={(event) => updateAgent(index, { baseUrl: event.target.value })}
                  />
                  <input
                    placeholder="model"
                    value={agent.model}
                    onChange={(event) => updateAgent(index, { model: event.target.value })}
                  />
                </div>
              </Field>
              <div className="row">
                <input
                  placeholder="apiKey (пусто = оставить сохранённый)"
                  value={agent.apiKey}
                  onChange={(event) => updateAgent(index, { apiKey: event.target.value })}
                />
                <input
                  placeholder="temperature"
                  value={agent.temperature}
                  onChange={(event) => updateAgent(index, { temperature: event.target.value })}
                />
              </div>
              <div className="agent-foot">
                <button
                  className="ghost"
                  disabled={!ready || checking === agent.role}
                  onClick={() => checkAgent(agent.role)}
                >
                  {checking === agent.role ? 'Проверяю…' : 'Проверить модель'}
                </button>
                {checks[agent.role] && (
                  <span className={checks[agent.role].startsWith('ошибка') ? 'check bad' : 'check ok'}>
                    {checks[agent.role]}
                  </span>
                )}
              </div>
            </div>
          ))}

          <Actions>
            <button disabled={!ready} onClick={saveAgents}>
              Сохранить агентов
            </button>
            <button
              className="ghost"
              disabled={!ready}
              onClick={async () => {
                const list = await api.listAgents(targetSystemId).catch(() => null);
                if (list) {
                  setAgentsSaved(
                    list.agents.map(
                      (agentView) =>
                        `${agentView.role} · ${agentView.provider} · ${agentView.model} · ключ: ${agentView.hasApiKey ? 'задан' : 'нет'}`,
                    ),
                  );
                }
              }}
            >
              Показать агентов
            </button>
          </Actions>
          <Cards>
            {agentsSaved.map((line) => (
              <Card key={line}>{line}</Card>
            ))}
          </Cards>
        </Section>

        <Section title="3. Готовность и план" hint="Пайплайн не стартует, пока не заданы workspace и валидный агент на каждую роль.">
          <Actions>
            <button disabled={!ready} onClick={loadPlan}>
              Проверить готовность
            </button>
          </Actions>
          <Cards>
            {plan && (
              <>
                <Card>
                  готовность:{' '}
                  <Badge tone={plan.readiness.ready ? 'ok' : 'bad'}>
                    {plan.readiness.ready ? 'ready' : 'not ready'}
                  </Badge>
                </Card>
                {(plan.readiness.missing ?? []).map((item) => (
                  <Card key={item}>{item}</Card>
                ))}
              </>
            )}
          </Cards>
          {plan && (
            <table>
              <thead>
                <tr>
                  <th>stage</th>
                  <th>имя</th>
                  <th>владелец</th>
                  <th>вход</th>
                  <th>выход</th>
                </tr>
              </thead>
              <tbody>
                {plan.stages.map((stage) => (
                  <tr key={stage.id}>
                    <td>{stage.id}</td>
                    <td>{stageTitle(stage)}</td>
                    <td>{stage.owner}</td>
                    <td className="mono">{(stage.inputs ?? []).join(', ')}</td>
                    <td className="mono">{(stage.outputs ?? []).join(', ')}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Section>
      </>
    );
  }

  // asked is set when the operator picked an entry for a question about a list.
  async function answerQuestion(pointer: string, value: unknown, questionPipeline: number, asked?: string) {
    const form = await api.answerQuestion(questionPipeline, pointer, value, asked).catch((error) => {
      if (!asked && error instanceof ApiError && error.choices.length > 0) {
        setChoice({ asked: pointer, value, pipelineId: questionPipeline, choices: error.choices });
      } else {
        setQuestionsError(`ответ не принят: ${String(error)}`);
      }
      return null;
    });
    if (form) setChoice(null);
    await loadQuestions();
    if (pipelineId) await loadPipelineForms(pipelineId);
    // The answer decided the field: the corrected form already passed the gates,
    // so re-running the stage is instant and moves the cursor to the next stage.
    if (form && form.status === 'valid' && targetSystemId) {
      await runStage(form.stage).catch(() => undefined);
    }
  }

  async function deleteOpenedForm() {
    if (!opened) return;
    if (!window.confirm(`Удалить документ ${opened.formId} v${opened.formVersion}?`)) return;
    try {
      await api.deleteForm(opened.formId);
      setOpened(null);
      if (pipelineId) await loadPipelineForms(pipelineId);
    } catch (error) {
      setRunError(`не удалось удалить документ: ${String(error)}`);
    }
  }

  async function editOpenedPayload() {
    if (!opened) return;
    const raw = window.prompt('Новый payload (JSON):', JSON.stringify(opened.payload));
    if (raw === null) return;
    let parsed: unknown;
    try {
      parsed = JSON.parse(raw);
    } catch {
      setRunError('невалидный JSON в payload');
      return;
    }
    try {
      const updated = await api.updateFormPayload(opened.formId, opened.formVersion, parsed);
      setOpened(updated);
      if (pipelineId) await loadPipelineForms(pipelineId);
    } catch (error) {
      setRunError(`не удалось изменить payload: ${String(error)}`);
    }
  }

  function pipelineView() {
    if (!ready) {
      return (
        <Section title="Работа" hint="Нужен target system.">
          <Cards>
            <Card>target system не выбран — настройте его на вкладке «Настройки».</Card>
          </Cards>
          <Actions>
            <button onClick={() => setView('settings')}>Перейти в настройки</button>
          </Actions>
        </Section>
      );
    }

    return (
      <>
        <div className="toolbar">
          <div className="toolbar-field">
            <Actions>
              <button disabled={!configured} onClick={createPipeline}>
                Создать пайплайн
              </button>
            </Actions>
          </div>
          <span className="hint">{`готовность: ${configured ? 'ready' : 'не готова'} · пайплайн featureDesign: последовательность агентов, маршрут принадлежит реестру`}</span>
        </div>

        <div className="versions">
          {pipelines.length === 0 && <span className="hint">Пайплайнов пока нет — создайте первый.</span>}
          {pipelines.map((row) => (
            <button
              key={row.id}
              className={`chip${pipeline?.id === row.id ? ' on' : ''}`}
              onClick={() => choosePipeline(row)}
            >
              #{row.id} · система {row.targetsystemid} · {row.status} · стадия {row.currentstage}
            </button>
          ))}
          {pipelines
            .filter((row) => row.id === pipeline?.id)
            .map((row) => (
              <button key={`delete-${row.id}`} className="ghost" onClick={() => void deletePipeline(row)}>
                Удалить пайплайн
              </button>
            ))}
        </div>

        {plan && !configured && (
          <div className="banner bad">
            <b>Среда не готова:</b> {(plan.readiness.missing ?? []).join('; ')}
            <button className="ghost" onClick={() => setView('settings')}>
              Исправить в настройках
            </button>
          </div>
        )}

        {plan && (
          <PipelineGraph
            stages={plan.stages}
            states={nodeStates}
            selected={selectedStage}
            onSelect={(stageID) => void selectStage(stageID)}
          />
        )}

        <AgentsDashboard
          api={api}
          activities={activities}
          loading={activitiesLoading}
          documentStatus={documentStatus}
          onStop={(activity) => {
            void api
              .stopAgent(activity.pipelineId, activity.role)
              .then(() => void loadActivities())
              .catch((error) => setQuestionsError(`стоп не принят: ${String(error)}`));
          }}
        />

        <QuestionsDashboard
          questions={questions}
          loading={questionsLoading}
          error={questionsError}
          onRefresh={() => void loadQuestions()}
          onAnswer={(pointer, value, questionPipeline) => void answerQuestion(pointer, value, questionPipeline)}
          choice={choice}
          onChoose={(chosen) => choice && void answerQuestion(chosen, choice.value, choice.pipelineId, choice.asked)}
        />

        {hasPipeline && selectedStage ? (
          stagePanel()
        ) : (
          <p className="hint">
            {hasPipeline ? 'Нажмите стадию на схеме — покажу документ, который она создаёт.' : 'Создайте или выберите пайплайн.'}
          </p>
        )}
      </>
    );
  }

  function stagePanel() {
    const stage = plan?.stages.find((item) => item.id === selectedStage);
    if (!stage) return null;
    const types = stage.outputs ?? [];
    const isInputStage = stage.id === STAGE_INPUT;

    return (
      <section className="stage-panel">
        <div className="section-head">
          <div>
            <h2>
              Стадия {stage.id} · {stageTitle(stage)}
            </h2>
            <p className="hint">
              владелец: {stage.owner} · вход: {(stage.inputs ?? []).join(', ') || '—'} · выход:{' '}
              {types.join(', ') || '—'}
            </p>
          </div>
          {opened && (
            <label className="toggle">
              <input
                type="checkbox"
                checked={rawPayload}
                onChange={(event) => setRawPayload(event.target.checked)}
              />
              показать JSON
            </label>
          )}
        </div>

        {isInputStage && (
          <div className="stage-actions">
            <Field label="Запрос оператора (USER_INPUT)">
              <textarea value={request} onChange={(event) => setRequest(event.target.value)} />
            </Field>
            <Actions>
              <button className="ghost" disabled={!hasPipeline} onClick={submitInput}>
                Сохранить USER_INPUT
              </button>
            </Actions>
            {inputForm && (
              <p className="hint">
                вход: {inputForm.formId} v{inputForm.formVersion} · {inputForm.payloadHash.slice(0, 26)}…
              </p>
            )}
          </div>
        )}

        <Actions>
          <button disabled={!hasPipeline || running} onClick={() => void runStage(stage.id)}>
            {running ? 'Модель работает…' : `Запустить стадию ${stage.id}`}
          </button>
          <button className="ghost" onClick={() => void loadPipelineForms(pipelineId)}>
            Обновить
          </button>
        </Actions>

        {runError && (
          <p className="check bad">
            <Badge tone="bad">ошибка</Badge> {runError}
          </p>
        )}
        {accepted && (
          <div className="cards">
            <Card>
              <Badge tone="ok">запуск принят</Badge> run #{accepted.id} · стадия {accepted.stage}
              <p className="hint">
                Цепочка работает в фоне: обновление страницы и перезапуск сервера её не отменяют.
                Ход и вызовы к модели — в карточке агента; формы, вопросы и курсор обновятся сами,
                когда серия попыток закончится.
              </p>
            </Card>
          </div>
        )}

        {types.map((type) => {
          const versions = formsByType[type] ?? [];
          return (
            <div className="doc-group" key={type}>
              <h3>
                {type} <span className="hint">· версий: {versions.length}</span>
              </h3>
              {versions.length === 0 ? (
                <DocumentOutline type={type} />
              ) : (
                <div className="versions">
                  {versions.map((form) => (
                    <button
                      key={`${form.formId}-${form.formVersion}`}
                      className={`chip${
                        opened?.formId === form.formId && opened.formVersion === form.formVersion ? ' on' : ''
                      }`}
                      onClick={() => setOpened(form)}
                    >
                      v{form.formVersion} · {form.status} · стадия {form.stage}
                    </button>
                  ))}
                </div>
              )}
            </div>
          );
        })}

        {opened && (
          <>
            <div className="actions">
              <button className="ghost" onClick={() => void deleteOpenedForm()}>
                Удалить документ
              </button>
              <button className="ghost" onClick={() => void editOpenedPayload()}>
                Изменить payload вручную
              </button>
            </div>
            <FormDocument form={opened} raw={rawPayload} />
          </>
        )}
      </section>
    );
  }

  function updateAgent(index: number, patch: Partial<AgentDraft>) {
    setAgents((previous) => previous.map((agent, i) => (i === index ? { ...agent, ...patch } : agent)));
  }
}

// DocumentOutline shows the paragraphs the stage will fill before it runs. The list
// mirrors form-registry/*.md, so the operator knows what the document will contain.
function DocumentOutline({ type }: { type?: string }) {
  const paragraphs = type ? DOCUMENT_OUTLINE[type] : undefined;
  return (
    <div className="document outline">
      <div className="doc-head">
        <b>{type ?? 'документ'}</b> · ещё не создан
      </div>
      {paragraphs ? (
        <ol className="outline-list">
          {paragraphs.map((title) => (
            <li key={title}>{title}</li>
          ))}
        </ol>
      ) : (
        <p className="hint">Описание абзацев для этого типа формы пока не задано.</p>
      )}
    </div>
  );
}

// Follow the newest line only while the reader is already at the bottom: the
// panel polls every few seconds, and a reader who scrolled up is reading.
function useStickToBottom(dependency: unknown) {
  const box = useRef<HTMLDivElement>(null);
  const atBottom = useRef(true);

  useEffect(() => {
    const node = box.current;
    if (node && atBottom.current) node.scrollTop = node.scrollHeight;
  }, [dependency]);

  return {
    ref: box,
    onScroll: () => {
      const node = box.current;
      if (node) atBottom.current = node.scrollHeight - node.scrollTop - node.clientHeight < 24;
    },
  };
}

function LogPanel({ entries }: { entries: LogEntry[] }) {
  const stick = useStickToBottom(entries);
  return (
    <div id="log" ref={stick.ref} onScroll={stick.onScroll}>
      {entries.map((entry, index) => (
        <div key={index} className={`line ${entry.kind}`}>
          {entry.text}
        </div>
      ))}
    </div>
  );
}

// ServerLog shows the lines the server wrote. A server started as a background
// process has no console, so this panel is the only view of its log.
function ServerLog({ lines }: { lines: string[] }) {
  const stick = useStickToBottom(lines);
  return (
    <div id="log" ref={stick.ref} onScroll={stick.onScroll}>
      {lines.map((line, index) => (
        <div key={index} className="line">
          {line}
        </div>
      ))}
    </div>
  );
}

// ConsolePanel switches between the request journal of this browser and the log of
// the server behind it.
function ConsolePanel({
  entries,
  lines,
  panel,
  onPanel,
}: {
  entries: LogEntry[];
  lines: string[];
  panel: 'journal' | 'server';
  onPanel: (panel: 'journal' | 'server') => void;
}) {
  return (
    <div>
      <div className="tabs">
        <button className={panel === 'journal' ? '' : 'ghost'} onClick={() => onPanel('journal')}>
          Журнал запросов
        </button>
        <button className={panel === 'server' ? '' : 'ghost'} onClick={() => onPanel('server')}>
          Логи сервера
        </button>
        <span className="hint">{panel === 'journal' ? `${entries.length} строк` : `${lines.length} строк`}</span>
      </div>
      {panel === 'journal' ? <LogPanel entries={entries} /> : <ServerLog lines={lines} />}
    </div>
  );
}

function Section({ title, hint, children }: { title: string; hint: string; children: ReactNode }) {
  return (
    <section>
      <h2>{title}</h2>
      <p className="hint">{hint}</p>
      {children}
    </section>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <label>{label}</label>
      {children}
    </>
  );
}

function Actions({ children }: { children: ReactNode }) {
  return <div className="actions">{children}</div>;
}

function Cards({ children }: { children: ReactNode }) {
  return <div className="cards">{children}</div>;
}

function Card({ children }: { children: ReactNode }) {
  return <div className="card">{children}</div>;
}

function Badge({ tone, children }: { tone: 'ok' | 'bad'; children: ReactNode }) {
  return <span className={`state ${tone}`}>{children}</span>;
}
