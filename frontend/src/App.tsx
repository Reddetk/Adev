import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { FormDocument } from './FormDocument';
import {
  ApiError,
  createApi,
  type AgentInput,
  type FormDetail,
  type FormSummary,
  type LogEntry,
  type Plan,
  type StageOutcome,
  type TargetSystemSummary,
} from './api';

const PROVIDERS = ['ollama', 'openrouter', 'openai', 'deepseek', 'qwen'];
const STAGE_BRD = '00';
const ROUTE_GREENFIELD = 'greenfield';

type AgentDraft = {
  role: string;
  provider: string;
  baseUrl: string;
  model: string;
  apiKey: string;
  temperature: string;
};

const initialAgents: AgentDraft[] = [
  { role: 'analyst', provider: 'ollama', baseUrl: 'http://localhost:11434', model: 'qwen3-coder:30b', apiKey: '', temperature: '0.2' },
  { role: 'architect', provider: 'ollama', baseUrl: 'http://localhost:11434', model: 'qwen3-coder:30b', apiKey: '', temperature: '0.2' },
];

// The console remembers the operator input, so a reload does not lose the setup.
// The api key is not kept in the browser: the server already holds it and an empty
// field means "keep the saved key".
const STORAGE_KEY = 'adev.console';

type SavedState = {
  targetSystemId: number | null;
  pipeline: { id: number; stage: string } | null;
  name: string;
  description: string;
  workspaceDir: string;
  agents: Omit<AgentDraft, 'apiKey'>[];
  request: string;
};

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
    { kind: 'request', text: 'Готово. Идите по шагам 1→7; каждая ошибка видна ниже и в журнале.' },
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
  const [pipeline, setPipeline] = useState<{ id: number; stage: string } | null>(saved.pipeline ?? null);
  const [request, setRequest] = useState(
    saved.request ?? 'The operator needs a library accounting system for two branches.',
  );
  const [inputForm, setInputForm] = useState<FormDetail | null>(null);
  const [outcome, setOutcome] = useState<StageOutcome | null>(null);
  const [runError, setRunError] = useState('');
  const [running, setRunning] = useState(false);
  const [formType, setFormType] = useState('USER_INPUT');
  const [forms, setForms] = useState<FormSummary[] | null>(null);
  const [opened, setOpened] = useState<FormDetail | null>(null);
  const [rawPayload, setRawPayload] = useState(false);

  const targetSystemId = targetSystem?.id ?? 0;
  const pipelineId = pipeline ? String(pipeline.id) : '';

  // Keep the operator input across reloads.
  useEffect(() => {
    const state: SavedState = {
      targetSystemId: targetSystem?.id ?? null,
      pipeline: pipeline ?? null,
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
  }, [targetSystem, pipeline, name, description, workspaceDir, agents, request]);

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
                (view) =>
                  `${view.role} · ${view.provider} · ${view.model} · ключ: ${view.hasApiKey ? 'задан' : 'нет'}`,
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
      .then((list) => setSystems(list.targetsystems))
      .catch(() => undefined);
  }, [api, targetSystemId]);

  // Switching the system drops everything that belongs to the old one.
  function switchTarget(id: number) {
    setTargetSystem({ id });
    setPipeline(null);
    setPlan(null);
    setForms(null);
    setOpened(null);
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
    const saved = await api.setWorkspace(targetSystemId, workspaceDir).catch(() => null);
    if (saved) setWorkspace(saved.dir);
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
        const view = await api.setAgent(targetSystemId, agent.role, input);
        lines.push(`${agent.role}: сохранён, hasApiKey=${view.hasApiKey}`);
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
    if (created) setPipeline({ id: created.id, stage: created.stage });
  }

  async function submitInput() {
    if (!targetSystemId || !pipelineId) return;
    const form = await api
      .submitForm({
        formId: `FRM-IN-${pipelineId}`,
        formType: 'USER_INPUT',
        schemaVersion: 1,
        formVersion: 1,
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
      .catch(() => null);
    if (form) {
      setInputForm(form);
      setOpened(form);
    }
  }

  async function runStage() {
    if (!targetSystemId || !pipelineId) return;
    setRunning(true);
    setRunError('');
    setOutcome(null);
    try {
      const result = await api.runStage(targetSystemId, STAGE_BRD, pipelineId);
      setOutcome(result);
      // The result of the stage is the form it stored: open it right away.
      if (!result.blocked) {
        setOpened(await api.form(result.formId, result.formVersion).catch(() => null));
      }
    } catch (error) {
      setRunError(`${String(error)}${failureHint(error)}`);
    } finally {
      setRunning(false);
    }
  }

  async function loadForms() {
    if (!pipelineId) return;
    setForms(await api.formsByPipeline(pipelineId, formType).catch(() => null));
  }

  async function openForm(form: FormSummary) {
    setOpened(await api.form(form.formId, form.formVersion).catch(() => null));
  }

  const ready = targetSystemId > 0;
  const hasPipeline = pipeline !== null;

  return (
    <>
      <header>
        <h1>adev · подготовка среды и запуск пайплайна</h1>
        <p>React + TypeScript: targetSystem → окружение → готовность → пайплайн → стадия 00 → формы</p>
      </header>

      <main>
        <div>
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

          <Section title="2. Окружение targetSystem" hint="Рабочая директория и модели агентов. У облачных провайдеров baseUrl можно оставить пустым — подставится адрес API; ключ уходит на сервер и никогда не возвращается.">
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
                        (view) => `${view.role} · ${view.provider} · ${view.model} · ключ: ${view.hasApiKey ? 'задан' : 'нет'}`,
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
                      <td>{stage.name}</td>
                      <td>{stage.owner}</td>
                      <td className="mono">{(stage.inputs ?? []).join(', ')}</td>
                      <td className="mono">{(stage.outputs ?? []).join(', ')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </Section>

          <Section title="4. Пайплайн" hint="Маршрут принадлежит реестру стадий; агент не выбирает следующую стадию.">
            <Actions>
              <button disabled={!ready} onClick={createPipeline}>
                Создать пайплайн
              </button>
            </Actions>
            <Cards>
              {pipeline && (
                <Card>
                  pipelineId=<b>{pipeline.id}</b> · route={ROUTE_GREENFIELD} · stage={pipeline.stage}{' '}
                  <Badge tone="ok">создан</Badge>
                </Card>
              )}
            </Cards>
          </Section>

          <Section title="5. Входная форма USER_INPUT" hint="Сырой запрос оператора. Сохраняется как форма, чтобы входные данные стадии были полными.">
            <textarea value={request} onChange={(event) => setRequest(event.target.value)} />
            <Actions>
              <button disabled={!hasPipeline} onClick={submitInput}>
                Сохранить USER_INPUT
              </button>
            </Actions>
            <Cards>
              {inputForm && (
                <Card>
                  {inputForm.formId} v{inputForm.formVersion} · {inputForm.payloadHash.slice(0, 26)}…
                </Card>
              )}
            </Cards>
          </Section>

          <Section title="6. Запуск стадии 00" hint="Агент-аналитик читает USER_INPUT, заполняет BRD и сдаёт форму. До трёх попыток, ошибки гейтов уходят модели.">
            <Actions>
              <button disabled={!hasPipeline || running} onClick={runStage}>
                {running ? 'Модель работает…' : 'Запустить стадию 00'}
              </button>
            </Actions>
            <Cards>
              {runError && (
                <Card>
                  <Badge tone="bad">ошибка</Badge> {runError}
                </Card>
              )}
              {outcome && (
                <Card>
                  {outcome.blocked ? (
                    <>
                      <Badge tone="bad">blocked</Badge> попыток: {outcome.attempts}
                    </>
                  ) : (
                    <>
                      <Badge tone="ok">done</Badge> {outcome.formId} v{outcome.formVersion} · попыток:{' '}
                      {outcome.attempts}
                    </>
                  )}
                </Card>
              )}
              {(outcome?.gaps ?? []).map((gap) => (
                <Card key={gap}>
                  <span className="mono">{gap}</span>
                </Card>
              ))}
            </Cards>
          </Section>

          <Section title="7. Формы пайплайна" hint="Нажмите строку — покажу содержимое формы.">
            <Actions>
              <button className="ghost" disabled={!hasPipeline} onClick={loadForms}>
                Обновить список
              </button>
              <select value={formType} onChange={(event) => setFormType(event.target.value)}>
                <option value="USER_INPUT">USER_INPUT</option>
                <option value="BRD">BRD</option>
              </select>
              {opened && (
                <label className="toggle">
                  <input type="checkbox" checked={rawPayload} onChange={(event) => setRawPayload(event.target.checked)} />
                  показать JSON
                </label>
              )}
            </Actions>
            {forms && forms.length === 0 && <p className="hint">Форм этого типа пока нет.</p>}
            {forms && forms.length > 0 && (
              <table>
                <thead>
                  <tr>
                    <th>form</th>
                    <th>v</th>
                    <th>status</th>
                    <th>hash</th>
                    <th>inputs</th>
                  </tr>
                </thead>
                <tbody>
                  {forms.map((form) => (
                    <tr key={`${form.formId}-${form.formVersion}`} onClick={() => openForm(form)}>
                      <td>{form.formId}</td>
                      <td>{form.formVersion}</td>
                      <td>{form.status}</td>
                      <td className="mono">{form.payloadHash.slice(0, 18)}…</td>
                      <td className="mono">
                        {(form.inputs ?? []).map((ref) => `${ref.formId}:${ref.formVersion}`).join(', ')}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            {opened && <FormDocument form={opened} raw={rawPayload} />}
          </Section>
        </div>

        <LogPanel entries={entries} />
      </main>
    </>
  );

  function updateAgent(index: number, patch: Partial<AgentDraft>) {
    setAgents((previous) => previous.map((agent, i) => (i === index ? { ...agent, ...patch } : agent)));
  }
}

function LogPanel({ entries }: { entries: LogEntry[] }) {
  const bottom = useRef<HTMLDivElement>(null);
  useEffect(() => {
    bottom.current?.scrollIntoView({ block: 'end' });
  }, [entries]);

  return (
    <div>
      <div id="log">
        {entries.map((entry, index) => (
          <div key={index} className={`line ${entry.kind}`}>
            {entry.text}
          </div>
        ))}
        <div ref={bottom} />
      </div>
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
