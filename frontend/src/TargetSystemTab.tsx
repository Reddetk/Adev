// Target System is the overview page: the single documentation of the project,
// read from the stored forms of one pipeline. Every section is a dynamic document
// (FR, NFR, the HLD extracts) plus the component map: the future C4 view where a
// container bundles its layers, a layer its packages, and a package its modules.
import { useEffect, useMemo, useState } from 'react';
import type {
  Api,
  FormDetail,
  HLDDegradation,
  HLDPayload,
  LLDPayload,
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

  return (
    <>
      <section>
        <h2>Документация проекта</h2>
        <p className="hint">
          Источник правды — сохранённые формы выбранного пайплайна. Разделы появляются по мере прохождения
          стадий: требования из SRS, карта контекстов и инварианты из HLD, физика — из LLD.
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

      {(hld.boundedContexts?.length ?? 0) > 0 && (
        <section>
          <h3>Context Map</h3>
          <div className="questions">
            {(hld.boundedContexts ?? []).map((context) => (
              <article className="question" key={context.id}>
                <b>{context.name}</b> <span className="tag">{context.id}</span>
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
                </tr>
              </thead>
              <tbody>
                {relations.map((relation, index) => (
                  <tr key={`${relation.from}-${relation.to}-${index}`}>
                     <td>{cell(relation.from)}</td>
                     <td>{cell(relation.to)}</td>
                     <td>{cell(relation.type)}</td>
                     <td className="mono">{cell(relation.contractId)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      )}

      {(hld.aggregates?.length ?? 0) > 0 && (
        <section>
          <h3>Агрегаты и инварианты</h3>
          {(hld.aggregates ?? []).map((aggregate) => (
            <div className="card" key={aggregate.id}>
              <b>{aggregate.root}</b> <span className="tag">{aggregate.id}</span>{' '}
              <span className="tag">контейнер {aggregate.containerId}</span>
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

// ComponentMap renders the C4 tree: container boxes carry their layers, layers
// their packages, packages their modules. Hover shows the purpose, right-click
// opens the contract menu of the object.
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
  // Merge the two sources of physical structure: the HLD names the container and
  // its purpose, the LLD fills the inside. An LLD container without an HLD twin
  // still shows.
  const hldById = new Map((hld.containers ?? []).map((container) => [container.objectId, container]));
  const containerIds: string[] = [];
  const seen = new Set<string>();
  for (const container of [...(hld.containers ?? []), ...(lld.containers ?? [])]) {
    const id = 'objectId' in container ? container.objectId : container.containerId;
    if (!seen.has(id)) {
      seen.add(id);
      containerIds.push(id);
    }
  }

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
        Метка каждой сущности: <span className="tag">to-be</span> создаётся этим пайплайном,{' '}
        <span className="tag">as-is</span> — существующая система. Наведение — назначение, правый клик — контракт.
      </p>
      <div className="c4">
        {containerIds.map((id) => {
          const hldContainer = hldById.get(id);
          const lldContainer = (lld.containers ?? []).find((item) => item.containerId === id);
          const related = (hld.contextRelations ?? [])
            .filter((relation) => relation.from === id || relation.to === id)
            .map((relation) => (relation.from === id ? `→ ${relation.to}` : `← ${relation.from}`));
          return (
            <div
              className="c4-container"
              key={id}
              title={`С4 Container · con-id ${id}${hldContainer ? ' · ' + text(hldContainer.purpose) : ''}`}
              onContextMenu={(event) => openMenu(event, `Контейнер ${id}`, contractLines(`/containers/${id}`))}
            >
              <div className="c4-head">
                <span className="tag">{id}</span>
                <span className="tag">C4 Container</span>
                <span className="tag">to-be</span>
                {hldContainer?.tech && <span className="tag">scope: {hldContainer.tech}</span>}
                {related.length > 0 && <span className="hint">связи: {related.join('; ')}</span>}
              </div>
              {hldContainer && <p className="c4-purpose">{text(hldContainer.purpose)}</p>}
              {(lldContainer?.layers ?? []).length === 0 && <p className="hint">слои появятся на стадии 05 (LLD)</p>}
              {(lldContainer?.layers ?? []).map((layer, layerIndex) => (
                <div
                  className="c4-layer"
                  key={layer.name}
                  title={`С4 Container Layer · ${layer.name}`}
                  onContextMenu={(event) =>
                    openMenu(event, `Слой ${layer.name} контейнера ${id}`, contractLines(`/containers/${id}/layers/${layer.name}`))
                  }
                >
                  <div className="c4-head">
                    <span className="tag">{layer.name}</span>
                    <span className="tag">C4 Container Layer</span>
                    <span className="tag">to-be</span>
                    <span className="hint">пакетов: {(layer.packages ?? []).length}</span>
                  </div>
                  {(layer.packages ?? []).map((pack, packageIndex) => (
                    <div
                      className="c4-package"
                      key={pack.path}
                      title={`С4 Component · ${pack.path}`}
                      onContextMenu={(event) =>
                        openMenu(event, `Пакет ${pack.path}`, contractLines(`/containers/${id}/layers/${layer.name}/packages/${pack.path}`))
                      }
                    >
                      <div className="c4-head">
                        <span className="tag">{pack.path}</span>
                        <span className="tag">C4 Component</span>
                        <span className="tag">to-be</span>
                      </div>
                      {(pack.modules ?? []).map((module) => (
                        <ModuleRow
                          key={module.objectId}
                          module={module}
                          pointer={`/containers/${id}/layers/${layerIndex}/packages/${packageIndex}/modules`}
                          contracts={contracts}
                          onMenu={(event, lines) => openMenu(event, `Модуль ${module.path}`, lines)}
                        />
                      ))}
                    </div>
                  ))}
                </div>
              ))}
            </div>
          );
        })}
      </div>
    </section>
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
