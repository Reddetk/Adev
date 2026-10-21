package usecase

import (
	"context"
	"strconv"

	"github.com/Reddetk/Adev/internal/domain"
)

// Orchestrator runs a stage and advances the pipeline cursor. The route belongs
// to the registry: after a successful run of the current stage, the next stage
// starts by itself, so the request walks the whole pipeline without clicks.
// shortcut: the chain lives inside one POST; a stretch longer than the 20-minute
// write deadline needs a background worker, move it there when stages get slower.
type Orchestrator struct {
	stages    *StageService
	pipelines PipelineRepository
}

// NewOrchestrator builds the pipeline orchestrator.
func NewOrchestrator(stages *StageService, pipelines PipelineRepository) *Orchestrator {
	return &Orchestrator{stages: stages, pipelines: pipelines}
}

// Run executes the stage and, on success of the current stage, moves the pipeline
// cursor to the next stage. Advancing is best-effort: a bookkeeping failure must
// not undo a stored form.
func (o *Orchestrator) Run(ctx context.Context, request StageRequest) (StageOutcome, error) {
	outcome, err := o.stages.Run(ctx, request)
	if err != nil || outcome.Blocked || o.pipelines == nil {
		return outcome, err
	}

	pipelineID, _ := strconv.Atoi(request.PipelineID)
	pipeline, err := o.pipelines.FindByID(pipelineID)
	if err != nil || pipeline == nil {
		return outcome, nil
	}
	next, err := domain.NextStage(request.Route, request.StageID)
	if err != nil {
		return outcome, nil
	}
	switch pipeline.CurrentStage {
	case request.StageID:
		pipeline.CurrentStage = next.ID
		_ = o.pipelines.Update(pipeline)
	case next.ID:
		// The cursor already stands on the next stage: an earlier run moved it
		// before the operator finished answering. The chain continues from there.
	default:
		// A stale stage finished behind the cursor; the pipeline is already further.
		return outcome, nil
	}
	// The next stage of the route runs right away; it ends the chain itself when
	// it asks the operator questions or the readiness check stops it.
	follow := request
	follow.StageID = next.ID
	return o.Run(ctx, follow)
}
