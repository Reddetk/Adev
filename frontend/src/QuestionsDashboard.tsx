// Open questions the models asked: one card per question, blocking ones first.
// The operator reads them here instead of hunting through the journal.
import { useState } from 'react';
import type { OperatorRequest } from './api';

export function QuestionsDashboard({
  questions,
  loading,
  error,
  onRefresh,
  onAnswer,
}: {
  questions: OperatorRequest[];
  loading: boolean;
  error: string;
  onRefresh: () => void;
  onAnswer: (pointer: string, value: unknown) => void;
}) {
  const open = questions
    .filter((question) => question.status !== 'completed')
    .sort((left, right) => Number(right.blocking) - Number(left.blocking) || left.id - right.id);

  return (
    <section>
      <div className="section-head">
        <div>
          <h2>Открытые вопросы</h2>
          <p className="hint">
            Фазы уточнения (00–02): агент спрашивает, что оператор не сказал в запросе. Ответ
            запишется в поле формы, вопрос закроется, стадия продолжится. Блокирующий вопрос
            держит стадию до ответа.
          </p>
        </div>
        <button className="ghost" disabled={loading} onClick={onRefresh}>
          {loading ? 'Обновляю…' : 'Обновить'}
        </button>
      </div>

      {error && <p className="check bad">{error}</p>}
      {!loading && !error && open.length === 0 && <p className="hint">Открытых вопросов нет.</p>}

      <div className="questions">
        {open.map((question) => (
          <QuestionCard key={question.id} question={question} onAnswer={onAnswer} />
        ))}
      </div>
    </section>
  );
}

function QuestionCard({
  question,
  onAnswer,
}: {
  question: OperatorRequest;
  onAnswer: (pointer: string, value: unknown) => void;
}) {
  const [value, setValue] = useState('');
  // The question names the options when it is a choice: "Options: A, B."
  const options = question.question.match(/Options: (.+)\.$/)?.[1].split(', ') ?? [];

  function submit() {
    const trimmed = value.trim();
    if (!trimmed) return;
    // The field takes raw JSON: plain words become a JSON string, numbers and
    // objects go as the operator typed them.
    let answer: unknown = trimmed;
    try {
      answer = JSON.parse(trimmed);
    } catch {
      answer = trimmed;
    }
    onAnswer(question.pointer, answer);
    setValue('');
  }

  return (
    <article className={`question${question.blocking ? ' blocking' : ''}`}>
      <div className="question-meta">
        <span className={`state ${question.blocking ? 'bad' : 'ok'}`}>
          {question.blocking ? 'блокирует' : 'не блокирует'}
        </span>
        <span className="tag">пайплайн {question.pipelineid}</span>
        <span className="mono">{question.pointer}</span>
        <span className="hint">{question.status}</span>
      </div>
      <p className="question-text">{question.question}</p>
      {options.length > 0 && (
        <div className="question-options">
          {options.map((option) => (
            <button key={option} className="chip" onClick={() => setValue(option)}>
              {option}
            </button>
          ))}
        </div>
      )}
      <div className="question-answer">
        <input
          placeholder="Ответ своими словами — или число для метрики"
          value={value}
          onChange={(event) => setValue(event.target.value)}
          onKeyDown={(event) => event.key === 'Enter' && submit()}
        />
        <button onClick={submit} disabled={!value.trim()}>
          Ответить
        </button>
      </div>
    </article>
  );
}
