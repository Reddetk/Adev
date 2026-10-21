// The pipeline is a chain of stages: the registry owns the order and the agent
// never picks the next one. The graph shows that chain and the state of each node,
// so the operator sees where the pipeline stands before reading any document.
import type { Stage } from './api';

export type NodeState = 'planned' | 'current' | 'draft' | 'valid' | 'waiting_operator' | 'approved';

const LABELS: Record<NodeState, string> = {
  planned: 'не запускалась',
  current: 'текущая',
  draft: 'черновик',
  valid: 'валидна',
  waiting_operator: 'ждёт ответа',
  approved: 'утверждена',
};

export function PipelineGraph({
  stages,
  states,
  selected,
  onSelect,
}: {
  stages: Stage[];
  states: Record<string, NodeState>;
  selected: string | null;
  onSelect: (stageID: string) => void;
}) {
  return (
    <div className="graph" role="list">
      {stages.map((stage, index) => {
        const state = states[stage.id] ?? 'planned';
        return (
          <div className="graph-step" role="listitem" key={stage.id}>
            <button
              type="button"
              className={`node ${state}${selected === stage.id ? ' selected' : ''}`}
              aria-pressed={selected === stage.id}
              onClick={() => onSelect(stage.id)}
            >
              <span className="node-id">{stage.id}</span>
              <span className="node-name">{stage.name}</span>
              <span className="node-owner">{stage.owner}</span>
              {stage.conditional && (
                <span className="tag" title="запускается только после проваленного вердикта">
                  условная
                </span>
              )}
              <span className="node-state">{LABELS[state]}</span>
            </button>
            {index < stages.length - 1 && <span className="edge" aria-hidden="true" />}
          </div>
        );
      })}
    </div>
  );
}
