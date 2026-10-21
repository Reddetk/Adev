// What the agents work on right now: one card per stage run, running first.
// App polls the live book and passes the snapshot down, like QuestionsDashboard.
import type { AgentActivity } from './api';

const STATUS_LABEL: Record<string, string> = {
  running: 'работает',
  blocked: 'ждёт ответа',
  done: 'готово',
  failed: 'ошибка',
};

export function AgentsDashboard({
  activities,
  loading,
}: {
  activities: AgentActivity[];
  loading: boolean;
}) {
  const running = activities.filter((activity) => activity.status === 'running');
  const recent = activities.filter((activity) => activity.status !== 'running');

  return (
    <section>
      <div className="section-head">
        <div>
          <h2>Работа агентов</h2>
          <p className="hint">
            Каждая карточка — один проход стадии: агент, форма, попытки модели. Обновление раз в 5
            секунд, пока вкладка открыта.
          </p>
        </div>
      </div>

      {loading && running.length === 0 && <p className="hint">Загружаю…</p>}
      {!loading && running.length === 0 && <p className="hint">Сейчас никто не работает.</p>}

      {running.length > 0 && (
        <div className="questions">
          {running.map((activity) => (
            <ActivityCard key={activity.id} activity={activity} />
          ))}
        </div>
      )}

      {recent.length > 0 && (
        <>
          <h3 className="hint">Недавние прогоны</h3>
          <div className="questions">
            {recent.slice(0, 8).map((activity) => (
              <ActivityCard key={activity.id} activity={activity} />
            ))}
          </div>
        </>
      )}
    </section>
  );
}

function ActivityCard({ activity }: { activity: AgentActivity }) {
  const started = new Date(activity.startedAt);
  const clock = started.toLocaleTimeString();
  const elapsed = Math.max(0, Math.round((Date.now() - started.getTime()) / 1000));
  const tone = activity.status === 'failed' ? 'bad' : activity.status === 'running' ? '' : 'ok';

  return (
    <article className={`question${activity.status === 'running' ? ' blocking' : ''}`}>
      <div className="question-meta">
        <span className={`state ${tone}`}>{STATUS_LABEL[activity.status] ?? activity.status}</span>
        <span className="tag">
          агент {activity.role} · стадия {activity.stage} · {activity.output}
        </span>
        <span className="tag">пайплайн {activity.pipelineId}</span>
        <span className="hint">
          с {clock} · {elapsed} с · попыток: {activity.attempts}
        </span>
      </div>
      {activity.lastError && <p className="check bad mono">{activity.lastError}</p>}
    </article>
  );
}
