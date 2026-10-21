// A payload is a document tree, not a text blob: every branch opens on demand and
// every field keeps its registry name. The operator reads fields, never JSON.
import type { ReactNode } from 'react';

type Field = { key: string; value: unknown };

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

// A Text value is {text, lang}: the operator reads the sentence, not the code.
function sentenceOf(value: unknown): string | null {
  if (isRecord(value) && typeof value.text === 'string' && typeof value.lang === 'string') {
    return value.text;
  }
  return null;
}

function Scalar({ value }: { value: unknown }) {
  if (value === null || value === undefined) return <span className="json-empty">—</span>;
  if (typeof value === 'boolean') return <span className="json-bool">{value ? 'да' : 'нет'}</span>;
  if (typeof value === 'number') return <span className="json-num">{value}</span>;
  return <span>{String(value)}</span>;
}

function Name({ name }: { name: string }) {
  return <span className="json-name">{name}</span>;
}

function Row({ name, children }: { name: string; children: ReactNode }) {
  return (
    <div className="json-row">
      <Name name={name} />
      <span className="json-value">{children}</span>
    </div>
  );
}

function Branch({
  name,
  count,
  open,
  children,
}: {
  name: string;
  count?: number;
  open: boolean;
  children: ReactNode;
}) {
  return (
    <details className="json-branch" open={open}>
      <summary>
        <Name name={name} />
        {count !== undefined && <span className="json-count">· {count}</span>}
      </summary>
      <div className="json-children">{children}</div>
    </details>
  );
}

// Node renders one field: a sentence, a scalar, or a branch the operator opens.
function Node({ field, depth }: { field: Field; depth: number }) {
  const { key, value } = field;
  const sentence = sentenceOf(value);
  if (sentence !== null) {
    return <Row name={key}>{sentence}</Row>;
  }
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return <Row name={key}><span className="json-empty">пусто</span></Row>;
    }
    return (
      <Branch name={key} count={value.length} open={depth === 0}>
        {value.map((item, index) => (
          <Node key={index} field={{ key: String(index + 1), value: item }} depth={depth + 1} />
        ))}
      </Branch>
    );
  }
  if (isRecord(value)) {
    const entries = Object.entries(value);
    if (entries.length === 0) {
      return <Row name={key}><span className="json-empty">пусто</span></Row>;
    }
    return (
      <Branch name={key} open={depth === 0}>
        {entries.map(([childKey, childValue]) => (
          <Node key={childKey} field={{ key: childKey, value: childValue }} depth={depth + 1} />
        ))}
      </Branch>
    );
  }
  return (
    <Row name={key}>
      <Scalar value={value} />
    </Row>
  );
}

// JsonTree is the viewer of every document that has no hand-written view.
export function JsonTree({ value }: { value: unknown }) {
  const entries: Field[] = Array.isArray(value)
    ? value.map((item, index) => ({ key: String(index + 1), value: item }))
    : isRecord(value)
      ? Object.entries(value).map(([key, child]) => ({ key, value: child }))
      : [];
  if (entries.length === 0) {
    return <p className="hint">Документ пуст.</p>;
  }
  return (
    <div className="json-tree">
      {entries.map((field) => (
        <Node key={field.key} field={field} depth={0} />
      ))}
    </div>
  );
}
