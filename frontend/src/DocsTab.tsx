// Every document of a pipeline in one place: one request reads all types, the
// filters narrow the list, and a version opens as the document it is.
import { useEffect, useMemo, useState } from 'react';
import { FormDocument } from './FormDocument';
import type { Api, FormDetail, PipelineRow } from './api';

export function DocsTab({ api, pipelines }: { api: Api; pipelines: PipelineRow[] }) {
  const [pipelineId, setPipelineId] = useState('');
  const [type, setType] = useState('');
  const [status, setStatus] = useState('');
  const [stage, setStage] = useState('');
  const [rows, setRows] = useState<FormDetail[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [opened, setOpened] = useState<FormDetail | null>(null);
  const [raw, setRaw] = useState(false);

  // The first pipeline is the one the operator most likely wants to read.
  useEffect(() => {
    if (!pipelineId && pipelines.length > 0) setPipelineId(String(pipelines[0].id));
  }, [pipelines, pipelineId]);

  // The filters belong to the server query: type, stage and status.
  useEffect(() => {
    if (!pipelineId) {
      setRows([]);
      return;
    }
    let stopped = false;
    setLoading(true);
    void api
      .formsByPipeline(pipelineId, { type, stage, status })
      .then((list) => {
        if (stopped) return;
        setRows(list);
        setError('');
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
  }, [api, pipelineId, type, status, stage]);

  // The list arrives by stage and version; the viewer groups it by form type.
  const byType = useMemo(() => {
    const grouped = new Map<string, FormDetail[]>();
    for (const row of rows) {
      const list = grouped.get(row.formType) ?? [];
      list.push(row);
      grouped.set(row.formType, list);
    }
    return [...grouped.entries()].map(([formType, versions]) => ({
      formType,
      versions: [...versions].sort((left, right) => left.formVersion - right.formVersion),
    }));
  }, [rows]);

  const types = useMemo(() => [...new Set(rows.map((row) => row.formType))].sort(), [rows]);
  const stages = useMemo(() => [...new Set(rows.map((row) => row.stage))].sort(), [rows]);

  return (
    <section>
      <div className="section-head">
        <div>
          <h2>Документы</h2>
          <p className="hint">
            Все версии каждого документа пайплайна: BRD, SRS, HLD, ADR, LLD, план задач и дальше.
            Фильтры читают список с сервера, версия открывается как документ.
          </p>
        </div>
        {opened && (
          <label className="toggle">
            <input type="checkbox" checked={raw} onChange={(event) => setRaw(event.target.checked)} />
            показать JSON
          </label>
        )}
      </div>

      <div className="row three">
        <div>
          <label>Пайплайн</label>
          <select value={pipelineId} onChange={(event) => setPipelineId(event.target.value)}>
            <option value="">— не выбран —</option>
            {pipelines.map((row) => (
              <option key={row.id} value={row.id}>
                {row.id} · система {row.targetsystemid} · {row.status} · стадия {row.currentstage}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label>Тип</label>
          <select value={type} onChange={(event) => setType(event.target.value)}>
            <option value="">— все —</option>
            {types.map((formType) => (
              <option key={formType} value={formType}>
                {formType}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label>Статус</label>
          <select value={status} onChange={(event) => setStatus(event.target.value)}>
            <option value="">— все —</option>
            {['draft', 'valid', 'approved', 'waiting_operator'].map((state) => (
              <option key={state} value={state}>
                {state}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="actions">
        <select value={stage} onChange={(event) => setStage(event.target.value)}>
          <option value="">стадия: все</option>
          {stages.map((id) => (
            <option key={id} value={id}>
              стадия {id}
            </option>
          ))}
        </select>
        <button className="ghost" disabled={loading} onClick={() => { setType(''); setStatus(''); setStage(''); }}>
          Сбросить фильтры
        </button>
      </div>

      {error && <p className="check bad">{error}</p>}
      {loading && <p className="hint">Читаю документы…</p>}
      {!loading && !error && rows.length === 0 && (
        <p className="hint">Документов нет: запустите стадию на вкладке «Пайплайны».</p>
      )}

      {byType.map(({ formType, versions }) => (
        <div className="doc-group" key={formType}>
          <h3>
            {formType} <span className="hint">· версий: {versions.length}</span>
          </h3>
          <div className="versions">
            {versions.map((form) => (
              <button
                key={`${form.formId}-${form.formVersion}`}
                className={`chip${
                  opened && opened.formId === form.formId && opened.formVersion === form.formVersion ? ' on' : ''
                }`}
                onClick={() => setOpened(form)}
              >
                v{form.formVersion} · {form.status} · стадия {form.stage}
              </button>
            ))}
          </div>
        </div>
      ))}

      {opened && (
        <details className="console" open>
          <summary>
            документ {opened.formType} {opened.formId} v{opened.formVersion}
          </summary>
          <FormDocument form={opened} raw={raw} />
        </details>
      )}
    </section>
  );
}
