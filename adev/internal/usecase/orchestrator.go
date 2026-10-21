package usecase

import (
	"context"
	"strconv"

	"github.com/Reddetk/Adev/internal/domain"
)

// Orchestrator runs a stage and advances the pipeline cursor. The route belongs
// to the registry: after a successful run of the current stage, the next stage
// becomes the cursor and the operator runs it the same way.
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
	if err != nil || pipeline == nil || pipeline.CurrentStage != request.StageID {
		return outcome, nil
	}
	next, err := domain.NextStage(request.Route, request.StageID)
	if err != nil || next.ID == pipeline.CurrentStage {
		return outcome, nil
	}
	pipeline.CurrentStage = next.ID
	_ = o.pipelines.Update(pipeline)
	return outcome, nil
}
