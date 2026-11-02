// Target System is the overview page: the single documentation of the project,
// read from the stored forms of one pipeline. Every section is a dynamic document
// (FR, NFR, the HLD extracts) plus the component map: the future C4 view where a
// container bundles its layers, a layer its packages, and a package its modules.
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import type {
  Api,
  FormDetail,
  HLDDegradation,
  HLDPayload,
  HLDRelation,
  LLDPayload,
  LLDContainer,
  LLDModule,
  SRSPayload,
  SRSFr,
  SRSNfr,
  Text,
} from './api';

const text = (value?: Text) => (value?.text ? value.text : '');
// A stored form may carry a value the shape calls a scalar but the model wrote as
// an object. Rendering that object crashes React, so every cell falls back to
// text or a dash instead of the raw value.
const cell = (value: unknown): string => {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'object') return '—';
  return String(value);
};

// latestForm picks the newest version of one type the gates accepted; a draft
// version may exist beside it, the map renders what survived the checks.
function latestForm(list: FormDetail[]): FormDetail | null {
  const valid = [...list].sort((a, b) => b.formVersion - a.formVersion);
  return valid.find((form) => form.status === 'valid') ?? valid[0] ?? null;
}

// The labels of the HLD entities, read by the operator instead of the codes the
// registry uses. An unknown value stays as it came: a new area must not vanish.
const AREA_LABEL: Record<string, string> = {
  architecture: 'архитектура',
  communication: 'взаимодействие',
  storage: 'хранилище',
  scaling: 'масштабирование',
  consistency: 'согласованность',
  reliability: 'надёжность',
  stack: 'стек',
  ops: 'эксплуатация',
  process: 'процесс',
  metrics: 'метрики',
};
const STYLE_LABEL: Record<string, string> = {
  monolith: 'монолит',
  modular_monolith: 'модульный монолит',
  hexagonal: 'гексагональная',
  microservices: 'микросервисы',
  event_driven: 'событийная',
  cell_based: 'ячейки',
  serverless: 'serverless',
};
const OWNERSHIP_LABEL: Record<string, string> = {
  database_per_service: 'база на сервис',
  shared_database: 'общая база',
  read_model: 'read model из событий',
};
const DECISION_STATUS_LABEL: Record<string, string> = {
  proposed: 'предложено',
  accepted: 'принято',
  superseded: 'заменено',
};
// D7 mechanisms and D8 deployment patterns, named as the catalogue names them.
const PATTERN_LABEL: Record<string, string> = {
  transactional_outbox: 'транзакционный outbox',
  transaction_log_tailing: 'чтение журнала БД',
  polling_publisher: 'периодический публикатор',
  dlq: 'очередь недоставленных (DLQ)',
  parking_lot: 'парковка сообщений',
  bounded_queue_block: 'ограниченная очередь с блокировкой',
  drop_new: 'сброс новых сообщений',
  degrade: 'деградация',
  scale_out: 'масштабирование',
  saga_orchestration: 'сага (оркестрация)',
  saga_choreography: 'сага (хореография)',
  two_phase_commit: 'двухфазная фиксация',
};
const DEPLOYMENT_LABEL: Record<string, string> = {
  multiple_services_per_host: 'несколько сервисов на хосте',
  service_per_host: 'сервис на хост',
  service_per_vm: 'сервис на VM',
  service_per_container: 'сервис на контейнер',
  serverless: 'serverless',
  service_deployment_platform: 'платформа развёртывания',
};
// PACELC: P is the network partition, E the normal case, L latency, C consistency.
const PACELC_HINT: Record<string, string> = {
  'PA/EL': 'при разделении сети отвечает, в норме быстрый: устаревшие данные допустимы',
  'PC/EC': 'при разделении сети ждёт, в норме согласованный: платит ожиданием',
  'PA/EC': 'при разделении сети отвечает, в норме согласованный',
  'PC/EL': 'при разделении сети ждёт, в норме быстрый',
};

// C4 draws one box shape for every container, so the color has to say what it
// is. The classes mirror the closed tech list of the domain.
type TechClass = 'code' | 'ui' | 'store' | 'bus' | 'edge' | 'platform' | 'external' | 'unknown';
const TECH_CLASS: Record<string, TechClass> = {
  go: 'code',
  java: 'code',
  node: 'code',
  python: 'code',
  react: 'ui',
  angular: 'ui',
  vue: 'ui',
  postgresql: 'store',
  mysql: 'store',
  mongodb: 'store',
  cassandra: 'store',
  redis: 'store',
  kafka: 'bus',
  rabbitmq: 'bus',
  nats: 'bus',
  nginx: 'edge',
  envoy: 'edge',
  kong: 'edge',
  keycloak: 'platform',
  elasticsearch: 'platform',
  minio: 'platform',
  kubernetes: 'platform',
  external: 'external',
};
const TECH_CLASS_LABEL: Record<TechClass, string> = {
  code: 'код',
  ui: 'интерфейс',
  store: 'хранилище',
  bus: 'шина',
  edge: 'шлюз',
  platform: 'платформа',
  external: 'внешняя система',
  unknown: 'без технологии',
};
const techClass = (tech?: string): TechClass => TECH_CLASS[(tech ?? '').toLowerCase()] ?? 'unknown';

const relationLabel = (relation: HLDRelation): string => {
  const kind =
    relation.type === 'async' ? 'событие' : relation.type === 'sync' ? 'вызов' : relation.type || 'связь';
  return relation.guarantee ? `${kind} · ${relation.guarantee}` : kind;
};
const relationHint = (relation: HLDRelation): string =>
  [
    relation.type === 'async' ? 'асинхронное событие' : relation.type === 'sync' ? 'синхронный вызов' : 'связь',
    relation.contractId ? `контракт ${relation.contractId}` : '',
    relation.guarantee ? `гарантия доставки ${relation.guarantee}` : '',
    relation.idempotencyKey ? `ключ идемпотентности ${relation.idempotencyKey}` : '',
    relation.retry ? 'повтор при сбое' : '',
    relation.dlq ? `DLQ ${relation.dlq}` : '',
  ]
    .filter(Boolean)
    .join(' · ');

export function TargetSystemTab({ api, pipelineId }: { api: Api; pipelineId: string }) {
  const [rows, setRows] = useState<FormDetail[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [menu, setMenu] = useState<{ x: number; y: number; lines: string[]; title: string } | null>(null);

  useEffect(() => {
    // Right-click anywhere else closes the contract menu.
    const close = () => setMenu(null);
    window.addEventListener('click', close);
    return () => window.removeEventListener('click', close);
  }, []);

  useEffect(() => {
    if (!pipelineId) {
      setRows([]);
      return;
    }
    let stopped = false;
    setLoading(true);
    void api
      .formsByPipeline(pipelineId)
      .then((list) => {
        if (!stopped) {
          setRows(list);
          setError('');
        }
      })
      .catch((failure) => {
        if (!stopped) setError(`не удалось прочитать документы: ${String(failure)}`);
      })
      .finally(() => {
        if (!stopped) setLoading(false);
      });
    return () => {
      stopped = true;
    };
  }, [api, pipelineId]);

  const docs = useMemo(() => {
    const byType = new Map<string, FormDetail[]>();
    for (const row of rows) {
      const list = byType.get(row.formType) ?? [];
      list.push(row);
      byType.set(row.formType, list);
    }
    const hld = latestForm(byType.get('HLD') ?? []);
    const srs = latestForm(byType.get('SRS') ?? []);
    const lld = latestForm(byType.get('LLD') ?? []);
    // FR and NFR go out as their own forms in later stages; until then the SRS
    // payload holds the lists, so the tab shows requirements either way.
    const frForms = (byType.get('FR') ?? []).map((form) => form.payload as SRSFr);
    const nfrForms = (byType.get('NFR') ?? []).map((form) => form.payload as SRSNfr);
    const srsPayload = (srs?.payload ?? {}) as SRSPayload;
    return {
      hld: (hld?.payload ?? {}) as HLDPayload,
      lld: (lld?.payload ?? {}) as LLDPayload,
      fr: frForms.length > 0 ? frForms : (srsPayload.fr ?? []),
      nfr: nfrForms.length > 0 ? nfrForms : (srsPayload.nfr ?? []),
      outOfScope: srsPayload.outOfScope ?? [],
      hasHld: Boolean(hld),
      hasLld: Boolean(lld),
      hasSrs: Boolean(srs),
    };
  }, [rows]);

  if (!pipelineId) {
    return <p className="hint">Выберите пайплайн на вкладке «Пайплайн» — документация появится здесь.</p>;
  }

  const hld = docs.hld;
  const lld = docs.lld;
  const relations = hld.contextRelations ?? [];
  // The term-to-context link is optional in the form: a column of dashes is noise.
  const termsWithContext = (hld.ubiquitousLanguage ?? []).some((term) => term.boundedContextId);
  // A relation names either a bounded context or a container of the design: the
  // table shows the name the operator knows, not only the id.
  const contextName = (id: string) =>
    (hld.boundedContexts ?? []).find((context) => context.id === id)?.name;
  const endpoint = (id: string) => {
    const context = contextName(id);
    if (context) return `${id} · ${context}`;
    const container = (hld.containers ?? []).find((item) => item.objectId === id);
    const owner = container?.boundedContextId ? contextName(container.boundedContextId) : undefined;
    return owner ? `${id} · ${owner}` : cell(id);
  };

  return (
    <>
      <section>
        <h2>Документация проекта</h2>
        <p className="hint">
          Источник правды — сохранённые формы выбранного пайплайна. Разделы появляются по мере прохождения
          стадий: требования из SRS, единый язык, решения, границы контекстов и инварианты из HLD, физика — из
          LLD.
        </p>
        {loading && <p className="hint">Загружаю…</p>}
        {error && <p className="check bad">{error}</p>}
        {!loading && rows.length === 0 && !error && (
          <p className="hint">Для этого пайплайна документов пока нет — запустите стадию во вкладке «Пайплайн».</p>
        )}
      </section>

      {(docs.fr.length > 0 || docs.nfr.length > 0) && (
        <section>
          <h3>Требования</h3>
          <h4 className="hint">Функциональные (FR)</h4>
          {docs.fr.length === 0 && <p className="hint">FR пока нет.</p>}
          <ul className="requirement-list">
            {docs.fr.map((fr) => (
              <li key={fr.id}>
                <b>{fr.id}</b> · {text(fr.statement)}
                {(fr.actor || fr.trigger) && (
                  <span className="hint"> — актёр: {fr.actor ?? '—'} · триггер: {fr.trigger ?? '—'}</span>
                )}
                <details>
                  <summary>поля требования</summary>
                  <dl className="mock">
                    <dt>response</dt>
                    <dd>{fr.response ?? '—'}</dd>
                    <dt>stateChange</dt>
                    <dd>{fr.stateChange ?? '—'}</dd>
                    {(fr.errors ?? []).map((item) => (
                      <span key={item.code}>
                        <dt>{item.code}</dt>
                        <dd>{item.condition}</dd>
                      </span>
                    ))}
                  </dl>
                </details>
                {fr.traceGoal && <span className="tag">к цели {fr.traceGoal}</span>}
              </li>
            ))}
          </ul>
          <h4 className="hint">Нефункциональные (NFR)</h4>
          {docs.nfr.length === 0 && <p className="hint">NFR пока нет.</p>}
          <table>
            <thead>
              <tr>
                <th>id</th>
                <th>метрика</th>
                <th>порог</th>
                <th>окно</th>
                <th>к цели</th>
              </tr>
            </thead>
            <tbody>
              {docs.nfr.map((nfr, index) => (
                <tr key={nfr.id ?? index}>
                  <td>{cell(nfr.id)}</td>
                  <td>{text(nfr.statement) || cell(nfr.metric)}</td>
                  <td className="mono">{nfr.threshold !== undefined ? `${cell(nfr.threshold)} ${cell(nfr.unit)}`.trim() : '\u2014'}</td>
                  <td>{cell(nfr.window)}</td>
                  <td>{cell(nfr.traceGoal)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      {(hld.ubiquitousLanguage?.length ?? 0) > 0 && (
        <section>
          <h3>Единый язык</h3>
          <p className="hint">
            Слова, у которых в этом проекте своё значение. Термин живёт в границах своего контекста: одно
            слово в двух контекстах — это два разных понятия.
          </p>
          <table>
            <thead>
              <tr>
                <th>термин</th>
                <th>определение</th>
                {termsWithContext && <th>контекст</th>}
                <th>источник</th>
              </tr>
            </thead>
            <tbody>
              {(hld.ubiquitousLanguage ?? []).map((term, index) => (
                <tr key={`${term.term}-${index}`}>
                  <td>
                    <b>{term.term}</b>
                  </td>
                  <td>{text(term.definition)}</td>
                  {termsWithContext && <td>{cell(term.boundedContextId)}</td>}
                  <td className="mono">{cell(term.sourceRef)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      {(hld.boundedContexts?.length ?? 0) > 0 && (
        <section>
          <h3>Context Map</h3>
          <div className="questions">
            {(hld.boundedContexts ?? []).map((context) => (
              <article className="question" key={context.id}>
                <b>{context.name}</b> <span className="tag">{context.id}</span>{' '}
                {context.pacelc && (
                  <span className="state ok" title={PACELC_HINT[context.pacelc] ?? 'класс PACELC'}>
                    {context.pacelc}
                  </span>
                )}
                <p>{text(context.purpose)}</p>
                {(context.containerIds ?? []).length > 0 && (
                  <p className="hint">контейнеры: {(context.containerIds ?? []).join(', ')}</p>
                )}
              </article>
            ))}
          </div>
          {relations.length > 0 && (
            <table>
              <thead>
                <tr>
                  <th>источник</th>
                  <th>приёмник</th>
                  <th>тип</th>
                  <th>контракт</th>
                  <th>гарантия доставки</th>
                  <th>ключ идемпотентности</th>
                  <th>надёжность</th>
                </tr>
              </thead>
              <tbody>
                {relations.map((relation, index) => {
                  const reliability = [
                    relation.retry ? 'повтор при сбое' : '',
                    relation.dlq ? `DLQ: ${relation.dlq}` : '',
                  ]
                    .filter(Boolean)
                    .join(' · ');
                  return (
                    <tr key={`${relation.from}-${relation.to}-${index}`}>
                      <td>{endpoint(relation.from)}</td>
                      <td>{endpoint(relation.to)}</td>
                      <td>
                        <span className="tag">{cell(relation.type)}</span>
                        {relation.type === 'sync' && <span className="hint"> ждёт ответа</span>}
                        {relation.type === 'async' && <span className="hint"> событие после ответа</span>}
                      </td>
                      <td className="mono">{cell(relation.contractId) || '—'}</td>
                      <td className="mono">{cell(relation.guarantee) || '—'}</td>
                      <td className="mono">{cell(relation.idempotencyKey) || '—'}</td>
                      <td>{reliability || '—'}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}
        </section>
      )}

      {(hld.decisions?.length ?? 0) > 0 && (
        <section>
          <h3>Решения (ADR)</h3>
          <p className="hint">
            Выборы, из которых собран HLD: что взяли, что отвергли и какое требование это решило.
          </p>
          <div className="questions">
            {(hld.decisions ?? []).map((decision) => (
              <article className="card" key={decision.id}>
                <div className="question-meta">
                  <span className="tag">{decision.id}</span>
                  {decision.area && <span className="tag">{AREA_LABEL[decision.area] ?? decision.area}</span>}
                  {decision.status && (
                    <span className={`state ${decision.status === 'accepted' ? 'ok' : ''}`}>
                      {DECISION_STATUS_LABEL[decision.status] ?? decision.status}
                    </span>
                  )}
                  {decision.contextRef && <span className="tag">контекст {decision.contextRef}</span>}
                  {decision.style && <span className="tag">стиль: {STYLE_LABEL[decision.style] ?? decision.style}</span>}
                  {decision.ownership && (
                    <span className="tag">данные: {OWNERSHIP_LABEL[decision.ownership] ?? decision.ownership}</span>
                  )}
                  {decision.pattern && (
                    <span className="tag" title="механизм распределённых данных (D7)">
                      механизм: {PATTERN_LABEL[decision.pattern] ?? decision.pattern}
                    </span>
                  )}
                  {decision.deployment && (
                    <span className="tag" title="паттерн развёртывания (D8)">
                      развёртывание: {DEPLOYMENT_LABEL[decision.deployment] ?? decision.deployment}
                    </span>
                  )}
                </div>
                <b>{text(decision.title)}</b>
                <p>{text(decision.decision)}</p>
                {(decision.techs ?? []).length > 0 && (
                  <p className="hint">продукты: {(decision.techs ?? []).join(', ')}</p>
                )}
                {(decision.objectRefs ?? []).length > 0 && (
                  <p className="hint">контейнеры решения: {(decision.objectRefs ?? []).join(', ')}</p>
                )}
                <details>
                  <summary>контекст, чек-лист, отвергнутые варианты и следствия</summary>
                  {text(decision.context) && <p>{text(decision.context)}</p>}
                  {(decision.checklist ?? []).length > 0 && (
                    <>
                      <p className="hint">чек-лист развёртывания:</p>
                      <ul>
                        {(decision.checklist ?? []).map((item, index) => (
                          <li key={index}>{item}</li>
                        ))}
                      </ul>
                    </>
                  )}
                  {(decision.alternatives ?? []).length > 0 && (
                    <>
                      <p className="hint">отвергнуто:</p>
                      <ul>
                        {(decision.alternatives ?? []).map((alternative, index) => (
                          <li key={index}>
                            <b>{text(alternative.option)}</b> — {text(alternative.rejectedBecause)}
                          </li>
                        ))}
                      </ul>
                    </>
                  )}
                  {(decision.consequences ?? []).length > 0 && (
                    <>
                      <p className="hint">следствия:</p>
                      <ul>
                        {(decision.consequences ?? []).map((consequence, index) => (
                          <li key={index}>{text(consequence)}</li>
                        ))}
                      </ul>
                    </>
                  )}
                </details>
              </article>
            ))}
          </div>
        </section>
      )}

      {(hld.aggregates?.length ?? 0) > 0 && (
        <section>
          <h3>Агрегаты и инварианты</h3>
          {(hld.aggregates ?? []).map((aggregate) => (
            <div className="card" key={aggregate.id}>
              <b>{aggregate.root}</b> <span className="tag">{aggregate.id}</span>{' '}
              <span className="tag">контейнер {aggregate.containerId}</span>{' '}
              {aggregate.moduleRef && (
                <span className="tag" title="модуль, который реализует корень агрегата">
                  модуль {aggregate.moduleRef}
                </span>
              )}
              {(aggregate.entityRefs ?? []).length > 0 && (
                <p className="hint">сущности: {(aggregate.entityRefs ?? []).join(', ')}</p>
              )}
              {(aggregate.eventMaps ?? []).length > 0 && (
                <p className="hint">события: {(aggregate.eventMaps ?? []).join(', ')}</p>
              )}
              {(aggregate.invariants ?? []).length === 0 ? (
                <p className="hint">инвариантов нет</p>
              ) : (
                <ul>
                  {(aggregate.invariants ?? []).map((invariant, index) => (
                    <li key={index}>
                      {invariant.modality ? <span className="tag">{invariant.modality}</span> : null} {invariant.text}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          ))}
        </section>
      )}

      {(hld.degradationStrategy?.length ?? 0) > 0 && (
        <section>
          <h3>Стратегия деградации</h3>
          <table>
            <thead>
              <tr>
                <th>отказ</th>
                <th>деградация</th>
                <th>обнаружение</th>
                <th>таймаут, мс</th>
              </tr>
            </thead>
            <tbody>
              {(hld.degradationStrategy ?? []).map((degradation: HLDDegradation, index) => (
                <tr key={index}>
                   <td>{cell(degradation.failure)}</td>
                   <td>{cell(degradation.degradation)}</td>
                   <td>{cell(degradation.detection)}</td>
                   <td>{cell(degradation.timeoutMs)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      {(hld.containers?.length ?? 0) + (lld.containers?.length ?? 0) > 0 ? (
        <ComponentMap hld={hld} lld={lld} contracts={lld.contracts ?? []} onContract={setMenu} />
      ) : (
        <section>
          <h3>Карта компонентов (C4)</h3>
          <p className="hint">
            Появится после стадии 03 (HLD) и 05 (LLD): контейнер = C4 Container, слой = Container Layer,
            пакет = Component, модуль = Code. Наведение показывает назначение, правый клик — контракт.
          </p>
        </section>
      )}

      {menu && (
        <div className="contract-menu" style={{ left: menu.x, top: menu.y }}>
          <b>{menu.title}</b>
          {menu.lines.length === 0 && <p className="hint">контрактов нет</p>}
          <ul>
            {menu.lines.map((line, index) => (
              <li key={index} className="mono">
                {line}
              </li>
            ))}
          </ul>
        </div>
      )}
    </>
  );
}

// ComponentMap is C4 level 2: bounded contexts as boundaries, containers as
// typed boxes, context relations as labelled orthogonal arrows. A click opens
// levels 3-4 of that container: layers, packages, modules.
type C4Rect = { x: number; y: number; w: number; h: number };
type C4Arrow = {
  key: string;
  d: string;
  label: string;
  x: number;
  y: number;
  width: number;
  vertical: boolean;
  async: boolean;
  hint: string;
};
type C4Node = {
  id: string;
  purpose: string;
  tech: string;
  contextId: string;
  nfrRefs: string[];
};
type C4Group = { id: string; name: string; pacelc?: string; nodes: C4Node[] };

// routeArrow keeps the connector orthogonal and exits the side the target lies
// on: horizontal when the boxes are side by side, vertical when they are stacked.
// shortcut: an arrow may cross a boundary it does not belong to and parallel
// edges overlap; a layered layout is worth it only past ~9 boxes and ~12 arrows.
function routeArrow(a: C4Rect, b: C4Rect): { d: string; x: number; y: number; vertical: boolean } {
  const ax = a.x + a.w / 2;
  const ay = a.y + a.h / 2;
  const bx = b.x + b.w / 2;
  const by = b.y + b.h / 2;
  if (Math.abs(bx - ax) >= Math.abs(by - ay)) {
    const right = bx >= ax;
    const x1 = right ? a.x + a.w : a.x;
    const x2 = right ? b.x : b.x + b.w;
    const mid = (x1 + x2) / 2;
    return { d: `M ${x1} ${ay} H ${mid} V ${by} H ${x2}`, x: mid, y: (ay + by) / 2, vertical: false };
  }
  const down = by >= ay;
  const y1 = down ? a.y + a.h : a.y;
  const y2 = down ? b.y : b.y + b.h;
  const mid = (y1 + y2) / 2;
  return { d: `M ${ax} ${y1} V ${mid} H ${bx} V ${y2}`, x: (ax + bx) / 2, y: mid, vertical: true };
}

function ComponentMap({
  hld,
  lld,
  contracts,
  onContract,
}: {
  hld: HLDPayload;
  lld: LLDPayload;
  contracts: NonNullable<LLDPayload['contracts']>;
  onContract: (menu: { x: number; y: number; lines: string[]; title: string }) => void;
}) {
  const relations = useMemo(() => hld.contextRelations ?? [], [hld]);
  const [openId, setOpenId] = useState<string | null>(null);
  const canvasRef = useRef<HTMLDivElement>(null);
  const [canvas, setCanvas] = useState<{ w: number; h: number; arrows: C4Arrow[] }>({
    w: 0,
    h: 0,
    arrows: [],
  });

  // Merge the two sources of physical structure: the HLD names the container and
  // its purpose, the LLD fills the inside. An LLD container without an HLD twin
  // still shows.
  const { groups, nodes } = useMemo(() => {
    const hldById = new Map((hld.containers ?? []).map((container) => [container.objectId, container]));
    const ids: string[] = [];
    const seen = new Set<string>();
    for (const container of [...(hld.containers ?? []), ...(lld.containers ?? [])]) {
      const id = 'objectId' in container ? container.objectId : container.containerId;
      if (!seen.has(id)) {
        seen.add(id);
        ids.push(id);
      }
    }
    const contexts = hld.boundedContexts ?? [];
    const all: C4Node[] = ids.map((id) => {
      const container = hldById.get(id);
      return {
        id,
        purpose: text(container?.purpose),
        tech: container?.tech ?? '',
        contextId: container?.boundedContextId ?? '',
        nfrRefs: container?.nfrRefs ?? [],
      };
    });
    const result: C4Group[] = contexts.map((context) => {
      const order = context.containerIds ?? [];
      const own = all.filter((node) => node.contextId === context.id);
      const ordered = [
        ...order
          .map((id) => own.find((node) => node.id === id))
          .filter((node): node is C4Node => Boolean(node)),
        ...own.filter((node) => !order.includes(node.id)),
      ];
      return { id: context.id, name: context.name, pacelc: context.pacelc, nodes: ordered };
    });
    const grouped = new Set(result.flatMap((group) => group.nodes.map((node) => node.id)));
    const loose = all.filter((node) => !grouped.has(node.id));
    if (loose.length > 0) {
      result.push({ id: '__loose', name: 'Вне границ контекста', nodes: loose });
    }
    return { groups: result, nodes: all };
  }, [hld, lld]);

  // Arrows depend on where the browser laid the boxes out: measure after layout
  // and again whenever the canvas resizes, because the boxes reflow.
  useLayoutEffect(() => {
    const root = canvasRef.current;
    if (!root) return;
    const measure = () => {
      const base = root.getBoundingClientRect();
      const rects = new Map<string, C4Rect>();
      root.querySelectorAll<HTMLElement>('[data-c4-id]').forEach((element) => {
        const id = element.dataset.c4Id;
        if (!id) return;
        const rect = element.getBoundingClientRect();
        rects.set(id, {
          x: rect.left - base.left,
          y: rect.top - base.top,
          w: rect.width,
          h: rect.height,
        });
      });
      const arrows: C4Arrow[] = [];
      for (const relation of relations) {
        const from = rects.get(relation.from);
        const to = rects.get(relation.to);
        if (!from || !to || relation.from === relation.to) continue;
        const route = routeArrow(from, to);
        const label = relationLabel(relation);
        arrows.push({
          key: `${relation.from}->${relation.to}-${arrows.length}`,
          ...route,
          label,
          width: label.length * 6.4 + 12,
          async: relation.type === 'async',
          hint: relationHint(relation),
        });
      }
      setCanvas({ w: base.width, h: base.height, arrows });
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(root);
    return () => observer.disconnect();
  }, [relations, openId]);

  const openNode = nodes.find((node) => node.id === openId) ?? null;
  const openLld = (lld.containers ?? []).find((container) => container.containerId === openNode?.id);
  const presentClasses = [...new Set(nodes.map((node) => techClass(node.tech)))];

  const openMenu = (event: React.MouseEvent, title: string, lines: string[]) => {
    event.preventDefault();
    event.stopPropagation();
    onContract({ x: event.clientX + 6, y: event.clientY + 6, lines, title });
  };

  const contractLines = (pointer: string): string[] => {
    const own = contracts.filter((contract) => contract.moduleRef?.pointer === pointer);
    if (own.length > 0) return own.map(contractLine);
    // The contract may name the container instead of one module: the right-click
    // on the container level falls back to everything of that branch.
    return contracts
      .filter((contract) => contract.moduleRef?.pointer?.startsWith(pointer))
      .map(contractLine);
  };

  const contractLine = (contract: (typeof contracts)[number]): string => {
    const kinds = [contract.kind, contract.inputType, contract.outputType].filter(Boolean).join(' · ');
    const errors = (contract.errors ?? []).map((error) => error.code).join(', ');
    return `${contract.id}: ${kinds}${errors ? ` — ошибки: ${errors}` : ''}`;
  };

  return (
    <section>
      <h3>Карта компонентов (C4)</h3>
      <p className="hint">
        C4, уровень 2 — контейнеры и связи. Граница — bounded context, цвет — класс технологии.
        Наведение — назначение, клик — уровни 3–4 этого контейнера, правый клик — контракт.
      </p>
      <div className="c4-canvas" ref={canvasRef}>
        <svg className="c4-arrows" width={canvas.w} height={canvas.h} aria-hidden="true">
          <defs>
            <marker
              id="c4-arrowhead"
              viewBox="0 0 8 8"
              refX="7"
              refY="4"
              markerWidth="7"
              markerHeight="7"
              orient="auto"
            >
              <path d="M 0 0 L 8 4 L 0 8 z" />
            </marker>
          </defs>
          {canvas.arrows.map((arrow) => {
            // A vertical edge is short: the label sits beside it, not on top of it.
            const labelX = arrow.vertical ? arrow.x + arrow.width / 2 + 10 : arrow.x;
            return (
              <g key={arrow.key} className={arrow.async ? 'c4-edge async' : 'c4-edge'}>
                <path d={arrow.d} markerEnd="url(#c4-arrowhead)">
                  <title>{arrow.hint}</title>
                </path>
                <rect
                  className="c4-edge-label-bg"
                  x={labelX - arrow.width / 2}
                  y={arrow.y - 9}
                  width={arrow.width}
                  height={18}
                  rx={4}
                />
                <text className="c4-edge-label" x={labelX} y={arrow.y}>
                  {arrow.label}
                </text>
              </g>
            );
          })}
        </svg>
        <div className="c4-boundaries">
          {groups.map((group) => (
            <div className="c4-boundary" data-c4-id={group.id} key={group.id}>
              <div className="c4-boundary-head">
                <span className="c4-boundary-name">{group.name}</span>
                <span className="tag">{group.id}</span>
                {group.pacelc && (
                  <span className="state ok" title={PACELC_HINT[group.pacelc] ?? 'класс PACELC'}>
                    {group.pacelc}
                  </span>
                )}
              </div>
              <div className="c4-nodes">
                {group.nodes.length === 0 && <p className="hint">контейнеров нет</p>}
                {group.nodes.map((node) => (
                  <button
                    type="button"
                    key={node.id}
                    className={`c4-node c4-tech-${techClass(node.tech)}${openId === node.id ? ' open' : ''}`}
                    data-c4-id={node.id}
                    title={`C4 Container · ${node.id}${node.tech ? ' · ' + node.tech : ''}${
                      node.purpose ? ' · ' + node.purpose : ''
                    }`}
                    onClick={() => setOpenId(openId === node.id ? null : node.id)}
                    onContextMenu={(event) =>
                      openMenu(event, `Контейнер ${node.id}`, contractLines(`/containers/${node.id}`))
                    }
                  >
                    <span className="c4-node-type">[Контейнер: {node.tech || 'без технологии'}]</span>
                    <span className="c4-node-name">{node.id}</span>
                    {node.purpose && <span className="c4-node-desc">{node.purpose}</span>}
                    {node.nfrRefs.length > 0 && (
                      <span className="c4-node-meta">NFR: {node.nfrRefs.join(', ')}</span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
      <div className="c4-legend">
        {presentClasses.map((cls) => (
          <span className="c4-legend-item" key={cls}>
            <span className={`c4-swatch c4-tech-${cls}`} />
            {TECH_CLASS_LABEL[cls]}
          </span>
        ))}
        <span className="c4-legend-item">
          <span className="c4-line" />
          синхронный вызов
        </span>
        <span className="c4-legend-item">
          <span className="c4-line async" />
          асинхронное событие
        </span>
      </div>
      {openNode && (
        <ContainerDetail
          node={openNode}
          container={openLld}
          contracts={contracts}
          contractLines={contractLines}
          onMenu={openMenu}
          onClose={() => setOpenId(null)}
        />
      )}
    </section>
  );
}

// ContainerDetail is C4 levels 3-4 inside one container: its layers, the packages
// of a layer and the modules of a package.
function ContainerDetail({
  node,
  container,
  contracts,
  contractLines,
  onMenu,
  onClose,
}: {
  node: C4Node;
  container?: LLDContainer;
  contracts: NonNullable<LLDPayload['contracts']>;
  contractLines: (pointer: string) => string[];
  onMenu: (event: React.MouseEvent, title: string, lines: string[]) => void;
  onClose: () => void;
}) {
  return (
    <div className="c4-detail">
      <div className="c4-detail-head">
        <b>{node.id}</b>
        <span className="tag">C4 Container</span>
        {node.tech && <span className="tag">tech: {node.tech}</span>}
        {node.nfrRefs.length > 0 && <span className="tag">NFR: {node.nfrRefs.join(', ')}</span>}
        <span className="hint">уровни 3–4</span>
        <button type="button" className="c4-close" onClick={onClose} title="закрыть">
          закрыть
        </button>
      </div>
      {node.purpose && <p className="c4-purpose">{node.purpose}</p>}
      {(container?.layers ?? []).length === 0 && (
        <p className="hint">слои появятся на стадии 05 (LLD)</p>
      )}
      {(container?.layers ?? []).map((layer, layerIndex) => (
        <div
          className="c4-layer"
          key={layer.name}
          title={`C4 Container Layer · ${layer.name}`}
          onContextMenu={(event) =>
            onMenu(
              event,
              `Слой ${layer.name} контейнера ${node.id}`,
              contractLines(`/containers/${node.id}/layers/${layer.name}`),
            )
          }
        >
          <div className="c4-head">
            <span className="tag">{layer.name}</span>
            <span className="tag">C4 Container Layer</span>
            <span className="hint">пакетов: {(layer.packages ?? []).length}</span>
          </div>
          {(layer.packages ?? []).map((pack, packageIndex) => (
            <div
              className="c4-package"
              key={pack.path}
              title={`C4 Component · ${pack.path}`}
              onContextMenu={(event) =>
                onMenu(
                  event,
                  `Пакет ${pack.path}`,
                  contractLines(`/containers/${node.id}/layers/${layer.name}/packages/${pack.path}`),
                )
              }
            >
              <div className="c4-head">
                <span className="tag">{pack.path}</span>
                <span className="tag">C4 Component</span>
              </div>
              {(pack.modules ?? []).map((module) => (
                <ModuleRow
                  key={module.objectId}
                  module={module}
                  pointer={`/containers/${node.id}/layers/${layerIndex}/packages/${packageIndex}/modules`}
                  contracts={contracts}
                  onMenu={(event, lines) => onMenu(event, `Модуль ${module.path}`, lines)}
                />
              ))}
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}

function ModuleRow({
  module,
  pointer,
  contracts,
  onMenu,
}: {
  module: LLDModule;
  pointer: string;
  contracts: NonNullable<LLDPayload['contracts']>;
  onMenu: (event: React.MouseEvent, lines: string[]) => void;
}) {
  const own = contracts.filter(
    (contract) => contract.moduleRef?.pointer === `${pointer}/${module.objectId}`,
  );
  const connected = (module.deps ?? []).map((dep) => dep.providerObjectId).join(', ');
  return (
    <div
      className="c4-module"
      title={`С4 Code · module-id ${module.objectId} · ${module.path}`}
      onContextMenu={(event) => {
        event.preventDefault();
        event.stopPropagation();
        onMenu(
          event,
          (own.length > 0
            ? own.map((contract) => `${contract.id} · ${[contract.kind, contract.inputType, contract.outputType].filter(Boolean).join(' · ')}`)
            : contracts
                .filter((contract) => contract.moduleRef?.pointer?.startsWith(`${pointer}/`))
                .map((contract) => `${contract.id} · ${[contract.kind, contract.inputType, contract.outputType].filter(Boolean).join(' · ')}`)
          ),
        );
      }}
    >
      <span className="tag">{module.path}</span>
      <span className="tag">C4 Code</span>
      <span className="tag">to-be</span>
      {connected && <span className="hint">модули: {connected}</span>}
    </div>
  );
}
