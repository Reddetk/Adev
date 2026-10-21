// The form payload is a document, not a data blob: the registry defines the fields
// and the agent fills them. This module turns a payload into the document the
// operator reads. A type without a hand-written view gets the field tree.
import type { ReactNode } from 'react';
import { JsonTree } from './JsonTree';
import type { FormDetail } from './api';

type Text = { text?: string; lang?: string };

type Brd = {
  goals?: { id?: string; outcome?: Text; metricId?: string }[];
  metrics?: { id?: string; name?: string; unit?: string; baseline?: number | null; target?: number | null }[];
  scope?: { in?: Text[]; out?: Text[]; deferred?: Text[] };
  actors?: { id?: string; name?: string; tasks?: Text[]; frequency?: string; businessObject?: string }[];
  constraints?: { id?: string; text?: string; modality?: string; priority?: string }[];
  assumptions?: Text[];
  successMetrics?: { metricId?: string; threshold?: string; window?: string }[];
  openGaps?: { id?: string; question?: Text; blocking?: boolean }[];
};

type UserInput = {
  rawRequest?: Text;
  source?: string;
  receivedAt?: string;
  locale?: string;
};

const text = (value?: Text) => value?.text ?? '';

function List({ items }: { items: string[] }) {
  if (items.length === 0) {
    return <p className="hint">Нет данных.</p>;
  }
  return (
    <ul>
      {items.map((item, index) => (
        <li key={index}>{item}</li>
      ))}
    </ul>
  );
}

function Block({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="doc-block">
      <h3>{title}</h3>
      {children}
    </div>
  );
}

function BrdDocument({ payload }: { payload: Brd }) {
  const goals = payload.goals ?? [];
  const metrics = payload.metrics ?? [];
  const actors = payload.actors ?? [];
  const constraints = payload.constraints ?? [];
  const successMetrics = payload.successMetrics ?? [];
  const gaps = payload.openGaps ?? [];

  return (
    <>
      <Block title="Цели">
        {goals.length === 0 ? (
          <p className="hint">Нет данных.</p>
        ) : (
          <ol>
            {goals.map((goal) => (
              <li key={goal.id}>
                {text(goal.outcome)} <span className="tag">{goal.metricId}</span>
              </li>
            ))}
          </ol>
        )}
      </Block>

      <Block title="Метрики">
        {metrics.length === 0 ? (
          <p className="hint">Нет данных.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>id</th>
                <th>название</th>
                <th>единица</th>
                <th>сейчас</th>
                <th>цель</th>
              </tr>
            </thead>
            <tbody>
              {metrics.map((metric) => (
                <tr key={metric.id}>
                  <td>{metric.id}</td>
                  <td>{metric.name}</td>
                  <td>{metric.unit}</td>
                  <td>{metric.baseline ?? '—'}</td>
                  <td>{metric.target ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Block>

      <Block title="Границы">
        <h4>Входит</h4>
        <List items={(payload.scope?.in ?? []).map(text)} />
        <h4>Не входит</h4>
        <List items={(payload.scope?.out ?? []).map(text)} />
        {(payload.scope?.deferred ?? []).length > 0 && (
          <>
            <h4>Отложено</h4>
            <List items={(payload.scope?.deferred ?? []).map(text)} />
          </>
        )}
      </Block>

      <Block title="Участники">
        {actors.length === 0 ? (
          <p className="hint">Нет данных.</p>
        ) : (
          <ul>
            {actors.map((actor) => (
              <li key={actor.id}>
                <b>{actor.name}</b> · {actor.frequency} · {actor.businessObject}
                <List items={(actor.tasks ?? []).map(text)} />
              </li>
            ))}
          </ul>
        )}
      </Block>

      <Block title="Ограничения">
        {constraints.length === 0 ? (
          <p className="hint">Нет данных.</p>
        ) : (
          <ul>
            {constraints.map((constraint) => (
              <li key={constraint.id}>
                {constraint.text} <span className="tag">{constraint.modality}</span>{' '}
                <span className="tag">{constraint.priority}</span>
              </li>
            ))}
          </ul>
        )}
      </Block>

      <Block title="Допущения">
        <List items={(payload.assumptions ?? []).map(text)} />
      </Block>

      <Block title="Признаки успеха">
        {successMetrics.length === 0 ? (
          <p className="hint">Нет данных.</p>
        ) : (
          <ul>
            {successMetrics.map((success, index) => (
              <li key={index}>
                {success.metricId} ≥ {success.threshold} за {success.window}
              </li>
            ))}
          </ul>
        )}
      </Block>

      <Block title="Открытые вопросы">
        {gaps.length === 0 ? (
          <p className="hint">Нет данных.</p>
        ) : (
          <ul>
            {gaps.map((gap) => (
              <li key={gap.id}>
                {text(gap.question)} {gap.blocking && <span className="tag warn">блокирует</span>}
              </li>
            ))}
          </ul>
        )}
      </Block>
    </>
  );
}

function UserInputDocument({ payload }: { payload: UserInput }) {
  return (
    <>
      <Block title="Запрос оператора">
        <p>{text(payload.rawRequest)}</p>
      </Block>
      <Block title="Источник">
        <p>
          {payload.source} · {payload.receivedAt} · {payload.locale ?? 'без локали'}
        </p>
      </Block>
    </>
  );
}

// FormDocument renders the stored form. The head shows the envelope facts that
// matter to the operator: what the form is, which version, and what it was made from.
export function FormDocument({ form, raw }: { form: FormDetail; raw: boolean }) {
  const inputs = (form.inputs ?? []).map((ref) => `${ref.formId}:${ref.formVersion}`).join(', ') || '—';

  return (
    <div className="document">
      <div className="doc-head">
        <div>
          <b>{form.formType}</b> {form.formId} · версия {form.formVersion} · {form.status}
        </div>
        <div className="hint">
          сделал {form.producedBy} · из {inputs} · {form.payloadHash}
        </div>
      </div>

      {raw ? (
        <pre className="payload">{JSON.stringify(form, null, 2)}</pre>
      ) : form.formType === 'BRD' ? (
        <BrdDocument payload={form.payload as Brd} />
      ) : form.formType === 'USER_INPUT' ? (
        <UserInputDocument payload={form.payload as UserInput} />
      ) : (
        <JsonTree value={form.payload} />
      )}
    </div>
  );
}
