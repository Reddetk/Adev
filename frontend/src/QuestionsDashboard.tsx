// Open questions the models asked: one card per question, blocking ones first.
// The operator reads them here instead of hunting through the journal.
import { useState } from 'react';
import type { OperatorRequest, PointerChoice } from './api';

export function QuestionsDashboard({
  questions,
  systemOf,
  loading,
  error,
  onRefresh,
  onAnswer,
  choice,
  onChoose,
}: {
  questions: OperatorRequest[];
  // systemOf names the target system a pipeline belongs to: the list shows every
  // pipeline, so a question must say whose business it asks about.
  systemOf?: (pipelineId: number) => number | undefined;
  loading: boolean;
  error: string;
  onRefresh: () => void;
  onAnswer: (pointer: string, value: unknown, pipelineId: number) => void;
  choice: { asked: string; choices: PointerChoice[] } | null;
  onChoose: (pointer: string) => void;
}) {
  const open = questions
    .filter((question) => question.status !== 'completed')
    .sort((left, right) => Number(right.blocking) - Number(left.blocking) || left.id - right.id);
  // A pipeline filter: every pipeline keeps its own questions, and the operator
  // works on one system at a time. The chips hold only the pipelines that still
  // have open questions, plus the one selected, so an emptied filter stays
  // visible with a zero instead of silently jumping to another pipeline.
  const [only, setOnly] = useState<number | 'all'>('all');
  const withOpen = [...new Set(open.map((question) => question.pipelineid))].sort((a, b) => a - b);
  const filterable = withOpen.length > 1 || (only !== 'all' && withOpen.length > 0);
  const active = filterable ? only : 'all';
  const chips =
    only === 'all' || withOpen.includes(only) ? withOpen : [...withOpen, only].sort((a, b) => a - b);
  const shown = active === 'all' ? open : open.filter((question) => question.pipelineid === active);
  const countOf = (pipelineId: number) =>
    open.filter((question) => question.pipelineid === pipelineId).length;

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
      {choice && (
        <div className="banner bad">
          <b>Какое место в форме?</b> Вопрос про {choice.asked} не называет место до конца. Выберите,
          куда относится ответ:
          <div className="question-options">
            {choice.choices.map((option) => (
              <button key={option.pointer} className="chip" onClick={() => onChoose(option.pointer)}>
                {option.label} · {option.pointer}
              </button>
            ))}
          </div>
        </div>
      )}
      {!loading && !error && open.length === 0 && <p className="hint">Открытых вопросов нет.</p>}

      {filterable && (
        <div className="question-options">
          <button
            className={`chip${active === 'all' ? ' on' : ''}`}
            onClick={() => setOnly('all')}
          >
            все пайплайны ({open.length})
          </button>
          {chips.map((pipelineId) => (
            <button
              key={pipelineId}
              className={`chip${active === pipelineId ? ' on' : ''}`}
              onClick={() => setOnly(pipelineId)}
            >
              пайплайн {pipelineId} ({countOf(pipelineId)})
            </button>
          ))}
        </div>
      )}

      {open.length > 0 && shown.length === 0 && (
        <p className="hint">В пайплайне {active} открытых вопросов нет: выберите другой или ответьте на остальные.</p>
      )}

      <div className="questions">
        {shown.map((question) => (
          <QuestionCard
            key={question.id}
            question={question}
            system={systemOf?.(question.pipelineid)}
            onAnswer={onAnswer}
          />
        ))}
      </div>
    </section>
  );
}

function QuestionCard({
  question,
  system,
  onAnswer,
}: {
  question: OperatorRequest;
  system?: number;
  onAnswer: (pointer: string, value: unknown, pipelineId: number) => void;
}) {
  const [value, setValue] = useState('');
  const [sent, setSent] = useState('');
  // The choices arrive as data, so a choice is one click. The format names the
  // input control; a number field blocks words: the draft would brake its shape.
  const options = question.options ?? [];
  const isNumber = question.format === 'number';
  const isList = question.format === 'list';
  const numeric = !Number.isNaN(Number(value.trim()));
  const answered = sent !== '';

  function submit(answer: unknown) {
    setSent(String(answer));
    setValue('');
    // The answer goes to the pipeline that asked, not to the one open in the console.
    onAnswer(question.pointer, answer, question.pipelineid);
  }

  function submitTyped() {
    const trimmed = value.trim();
    if (!trimmed) return;
    // The field wants the type its shape holds: words go as JSON strings,
    // a number field sends a number.
    let answer: unknown = trimmed;
    if (isNumber) {
      answer = Number(trimmed);
    } else {
      try {
        answer = JSON.parse(trimmed);
      } catch {
        answer = trimmed;
      }
    }
    submit(answer);
  }

  return (
    <article className={`question${question.blocking ? ' blocking' : ''}`}>
      <div className="question-meta">
        <span className={`state ${question.blocking ? 'bad' : 'ok'}`}>
          {question.blocking ? 'блокирует' : 'не блокирует'}
        </span>
        <span className="tag">пайплайн {question.pipelineid}</span>
        {system !== undefined && <span className="tag">система {system}</span>}
        <span className="mono">{question.pointer}</span>
      </div>
      <p className="question-text">{question.question}</p>
      {answered ? (
        <p className="check ok">
          отправлено: {sent} — вопрос закроется после ответа сервера
        </p>
      ) : (
        <>
          {options.length > 0 && (
            <div className="question-options">
              {options.map((option) => (
                <button key={option} className="chip" onClick={() => submit(option)}>
                  {option}
                </button>
              ))}
            </div>
          )}
          <div className="question-answer">
            {isNumber ? (
              <input
                type="number"
                step="any"
                placeholder="Число — например 5"
                value={value}
                onChange={(event) => setValue(event.target.value)}
                onKeyDown={(event) => event.key === 'Enter' && numeric && submitTyped()}
              />
            ) : isList ? (
              <textarea
                rows={2}
                placeholder="Одна запись списка — своими словами"
                value={value}
                onChange={(event) => setValue(event.target.value)}
              />
            ) : (
              <input
                placeholder={
                  options.length > 0
                    ? 'Или впишите свой вариант'
                    : 'Ответ своими словами'
                }
                value={value}
                onChange={(event) => setValue(event.target.value)}
                onKeyDown={(event) => event.key === 'Enter' && submitTyped()}
              />
            )}
            <button onClick={submitTyped} disabled={!value.trim() || (isNumber && !numeric)}>
              Ответить
            </button>
            {isNumber && value.trim() !== '' && !numeric && (
              <p className="check bad">Нужно число: в это поле формы слова не пройдут.</p>
            )}
          </div>
        </>
      )}
    </article>
  );
}
