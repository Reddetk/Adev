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

// requestLabel names the work of one call in one line, read from the user message
// of the stored prompt: a detail call starts with "Detail exactly one entry: ...",
// a skeleton call with "You fill the HLD form of stage 03". The operator sees what
// the agent asks the model about instead of the first line of a wall of text.
function requestLabel(row: ModelCall): string {
  const text = row.request ?? '';
  const marker = '=== USER MESSAGE ===';
  const at = text.indexOf(marker);
  const body = (at >= 0 ? text.slice(at + marker.length) : text).trim();
  if (!body) return 'промпт ещё не записан';
  const detail = 'Detail exactly one entry: ';
  let label = body.includes(detail) ? body.slice(body.indexOf(detail) + detail.length) : body;
  label = label.split('\n')[0].replace(/^You fill the /, '').replace(/^You interview the operator about the /, 'интервью: ');
  const stop = label.search(/\.(\s|$)/);
  if (stop > 0) label = label.slice(0, stop);
  return label.length > 120 ? `${label.slice(0, 117)}…` : label;
}

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
            Каждая карточка — один прогон стадии: агент, форма, попытки модели. {AGENT_RULE}
            История вызовов отделена по номеру прогона, у каждого запроса свой номер и тема.
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
// to "why did this run not count". The number is the order of the request inside
// the run, so the operator can say "запрос 3" and point at the same call.
function CallRow({ row, number }: { row: ModelCall; number: number }) {
  const [open, setOpen] = useState(false);
  const pending = !row.response && !row.errors;
  const rejected = Boolean(row.errors);
  return (
    <>
      <tr
        className={open ? 'calls-open' : ''}
        onClick={() => setOpen(!open)}
        style={{ cursor: 'pointer' }}
        title={`Запрос ${number} прогона (вызов #${row.id}). Открыть диалог: промпт, ответ модели, вердикты гейтов`}
      >
        <td className="calls-request">
          <span className="mono">#{number}</span> {requestLabel(row)}
        </td>
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
          <td colSpan={7} className="call-dialogue">
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
  // The calls are separated by the run row, not by the pipeline: one pipeline has
  // a chain of runs, and a merged history hides which run asked for what.
  const loadCalls = () => {
    void api
      .modelCalls()
      .then((list) =>
        setCalls(
          list.modelcalls
            .filter((row) => row.runid === activity.runId && row.profile === activity.role)
            .sort((left, right) => right.id - left.id),
        ),
      )
      .catch((failure) => setCallsError(String(failure)));
  };
  useEffect(() => {
    if (!open || activity.status !== 'running') return;
    const timer = window.setInterval(loadCalls, 5000);
    return () => window.clearInterval(timer);
  }, [api, open, activity.status, activity.runId, activity.role]);

  function toggle() {
    setOpen(!open);
    if (open) return;
    setCalls(null);
    setCallsError('');
    loadCalls();
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
        {activity.runId > 0 && <span className="tag">прогон #{activity.runId}</span>}
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
            <p className="hint">
              В прогоне #{activity.runId || '—'} ещё нет записей этого агента.
            </p>
          )}
          {calls !== null && calls.length > 0 && (
            <table>
              <thead>
                <tr>
                  <th>запрос</th>
                  <th>стадия</th>
                  <th>модель</th>
                  <th>токены вход</th>
                  <th>токены выход</th>
                  <th>ожидание, мс</th>
                  <th>статус</th>
                </tr>
              </thead>
              <tbody>
                {calls.map((row, index) => (
                  <CallRow key={row.id} row={row} number={calls.length - index} />
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </article>
  );
}
