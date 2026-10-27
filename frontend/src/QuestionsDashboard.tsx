// Open questions the models asked: one card per question, blocking ones first.
// The operator reads them here instead of hunting through the journal.
import { useState } from 'react';
import type { OperatorRequest, PointerChoice } from './api';

export function QuestionsDashboard({
  questions,
  loading,
  error,
  onRefresh,
  onAnswer,
  choice,
  onChoose,
}: {
  questions: OperatorRequest[];
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
