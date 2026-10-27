// What the agents work on right now: one card per stage run, running first.
// App polls the live book and passes the snapshot down, like QuestionsDashboard.
// A click on a card opens the cost book of that agent: every model attempt with
// its tokens and wait time, and an operator can stop a running agent in place.
import { useEffect, useState } from 'react';
import { NODE_LABELS } from './PipelineGraph';
import { STAGE_TITLES } from './api';
import type { AgentActivity, Api, ModelCall } from './api';

// The run status of the stage, not the state of its document: a finished run may
// have left a draft that the gates rejected. The document state comes from the
// stored forms and is shown next to it.
const STATUS_LABEL: Record<string, string> = {
  running: 'работает',
  blocked: 'ждёт ответа',
  done: 'проход завершён',
  failed: 'ошибка',
  stopped: 'остановлен оператором',
};

// The agent works in top-down batches: one level of the document tree is one
// model request, so a small request count near the end is normal, not a fault.
const AGENT_RULE = 'Принцип агента: 1 уровень вложенности документа = 1 запрос, сверху вниз.';

export function AgentsDashboard({
  api,
  activities,
  loading,
  documentStatus,
  onStop,
}: {
  api: Api;
  activities: AgentActivity[];
  loading: boolean;
  documentStatus: (activity: AgentActivity) => string | undefined;
  onStop: (activity: AgentActivity) => void;
}) {
  const running = activities.filter((activity) => activity.status === 'running');
  const recent = activities.filter((activity) => activity.status !== 'running');

  return (
    <section>
      <div className="section-head">
        <div>
          <h2>Работа агентов</h2>
          <p className="hint">
            Каждая карточка — один проход стадии: агент, форма, попытки модели. {AGENT_RULE}
          </p>
        </div>
      </div>

      {loading && running.length === 0 && <p className="hint">Загружаю…</p>}
      {!loading && running.length === 0 && <p className="hint">Сейчас никто не работает.</p>}

      {running.length > 0 && (
        <div className="questions">
          {running.map((activity) => (
            <ActivityCard key={activity.id} api={api} activity={activity} documentStatus={documentStatus} onStop={onStop} />
          ))}
        </div>
      )}

      {recent.length > 0 && (
        <>
          <h3 className="hint">Недавние прогоны</h3>
          <div className="questions">
            {recent.slice(0, 8).map((activity) => (
              <ActivityCard key={activity.id} api={api} activity={activity} documentStatus={documentStatus} onStop={onStop} />
            ))}
          </div>
        </>
      )}
    </section>
  );
}

// One attempt of the cost book. The row opens into the dialogue: what the agent
// sent, what the model answered, and which gates returned the draft — the answer
// to "why did this run not count".
function CallRow({ row }: { row: ModelCall }) {
  const [open, setOpen] = useState(false);
  const pending = !row.response && !row.errors;
  const rejected = Boolean(row.errors);
  return (
    <>
      <tr
        className={open ? 'calls-open' : ''}
        onClick={() => setOpen(!open)}
        style={{ cursor: 'pointer' }}
        title="Открыть диалог: промпт, ответ модели, вердикты гейтов"
      >
        <td>{row.stage || '—'}</td>
        <td className="mono">{row.model}</td>
        <td>{row.tokensin}</td>
        <td>{row.tokensout}</td>
        <td>{row.latencyms}</td>
          <td>
            {pending ? (
              <span className="state">ожидаем</span>
            ) : rejected ? (
              <span className="state bad">отклонён</span>
            ) : (
              <span className="state ok">принят</span>
            )}
          </td>
      </tr>
      {open && (
        <tr>
          <td colSpan={6} className="call-dialogue">
            {(row.errors ?? '')
              .split('\n')
              .filter(Boolean)
              .map((line, index) => (
                <p key={index} className="check bad mono">
                  {line}
                </p>
              ))}
            <details open>
              <summary>промпт агента ({(row.request ?? '').length} знаков)</summary>
              <pre className="mono">{row.request}</pre>
            </details>
            <details>
              <summary>ответ модели ({(row.response ?? '').length} знаков, JSON)</summary>
              <pre className="mono">{row.response}</pre>
            </details>
          </td>
        </tr>
      )}
    </>
  );
}

function ActivityCard({
  api,
  activity,
  documentStatus,
  onStop,
}: {
  api: Api;
  activity: AgentActivity;
  documentStatus: (activity: AgentActivity) => string | undefined;
  onStop: (activity: AgentActivity) => void;
}) {
  const [open, setOpen] = useState(false);
  const [calls, setCalls] = useState<ModelCall[] | null>(null);
  const [callsError, setCallsError] = useState('');
  const started = new Date(activity.startedAt);
  const clock = started.toLocaleTimeString();
  const elapsed = Math.max(0, Math.round((Date.now() - started.getTime()) / 1000));
  const tone = activity.status === 'failed' ? 'bad' : activity.status === 'running' ? '' : 'ok';
  // The document state is worth a chip only when the run ended: a running run has
  // no version yet, and a blocked one always waits for the operator (the run status
  // already says so).
  const ended = activity.status === 'done' || activity.status === 'failed';
  const document = ended ? documentStatus(activity) : undefined;

  // The history reloads on every expansion: rows are written only after a model
  // call returns, and one call takes about a minute, so an older snapshot would
  // sit empty while the attempts grow. While the agent still runs and stays open,
  // the list re-reads itself every 5 seconds, so a finished call shows at once.
  useEffect(() => {
    if (!open || activity.status !== 'running') return;
    const load = () => {
      void api
        .modelCalls()
        .then((list) =>
          setCalls(
            list.modelcalls.filter(
              (row) => String(row.runid) === activity.pipelineId && row.profile === activity.role,
            ),
          ),
        )
        .catch((failure) => setCallsError(String(failure)));
    };
    const timer = window.setInterval(load, 5000);
    return () => window.clearInterval(timer);
  }, [api, open, activity.status, activity.pipelineId, activity.role]);

  function toggle() {
    setOpen(!open);
    if (open) return;
    setCalls(null);
    setCallsError('');
    void api
      .modelCalls()
      .then((list) =>
        setCalls(
          list.modelcalls.filter(
            (row) => String(row.runid) === activity.pipelineId && row.profile === activity.role,
          ),
        ),
      )
      .catch((failure) => setCallsError(String(failure)));
  }

  return (
    <article className={`question${activity.status === 'running' ? ' blocking' : ''}`}>
      <div className="question-meta">
        <span className={`state ${tone}`}>{STATUS_LABEL[activity.status] ?? activity.status}</span>
        {document && (
          <span className={`state ${document === 'draft' ? 'bad' : 'ok'}`}>
            документ: {NODE_LABELS[document] ?? document}
          </span>
        )}
        <span className="tag">
          агент {activity.role} · {STAGE_TITLES[activity.stage] ? `${activity.stage} ${STAGE_TITLES[activity.stage]}` : activity.stage} · {activity.output}
        </span>
        <span className="tag">пайплайн {activity.pipelineId}</span>
        <span className="hint">
          с {clock} · {elapsed} с · попыток: {activity.attempts}
        </span>
        {activity.status === 'running' && (
          <button className="ghost" onClick={() => onStop(activity)}>
            Остановить агента
          </button>
        )}
        <button className="ghost" onClick={toggle}>
          {open ? 'Скрыть вызовы' : 'История вызовов к LLM'}
        </button>
      </div>
      {activity.lastError && <p className="check bad mono">{activity.lastError}</p>}
      {open && (
        <div className="model-calls">
          {callsError && <p className="check bad">история недоступна: {callsError}</p>}
          {!callsError && calls !== null && calls.length === 0 && (
            <p className="hint">Вызовов этого агента в этом пайплайне ещё не записано.</p>
          )}
          {calls !== null && calls.length > 0 && (
            <table>
              <thead>
                <tr>
                  <th>стадия</th>
                  <th>модель</th>
                  <th>токены вход</th>
                  <th>токены выход</th>
                  <th>ожидание, мс</th>
                  <th>брак</th>
                </tr>
              </thead>
              <tbody>
                {calls.map((row) => (
                  <CallRow key={row.id} row={row} />
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </article>
  );
}
