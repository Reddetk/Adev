package usecase

import (
	"errors"
	"fmt"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/messages"
)

// ErrPipelineExists reports a duplicate pipeline of the same route for one
// target system. The greenfield route builds the baseline once.
var ErrPipelineExists = errors.New("the pipeline already exists for this target system")

// ErrPipelineRunning refuses to delete a pipeline whose stage is still running:
// its chain would write into the rows that were just removed.
var ErrPipelineRunning = errors.New("the pipeline has a running stage: stop it first")

type pipelineService struct {
	repo PipelineRepository
}

func NewPipelineService(repo PipelineRepository) PipelineUseCase {
	return &pipelineService{repo: repo}
}

func (p *pipelineService) CreatePipeline(input CreatePipelineInput) (CreatePipelineOutput, error) {
	if err := input.Validate(); err != nil {
		return CreatePipelineOutput{}, err
	}

	// One feature-design pipeline per target system: the registry index enforces
	// the same rule in the database.
	if input.Route == string(domain.RouteGreenfield) {
		all, err := p.repo.FindAll()
		if err != nil {
			return CreatePipelineOutput{}, err
		}
		for _, existing := range all {
			if existing.TargetSystemID == input.TargetSystemID && existing.Route == string(domain.RouteGreenfield) {
				return CreatePipelineOutput{}, fmt.Errorf("%w: target system %d already owns greenfield pipeline %d",
					ErrPipelineExists, input.TargetSystemID, existing.ID)
			}
		}
	}

	pipeline := domain.Pipeline{
		TargetSystemID: input.TargetSystemID,
		Route:          input.Route,
		Status:         input.Status,
		CurrentStage:   input.CurrentStage,
	}

	if err := pipeline.Validate(); err != nil {
		return CreatePipelineOutput{}, err
	}

	if err := p.repo.Save(&pipeline); err != nil {
		return CreatePipelineOutput{}, err
	}

	return CreatePipelineOutput{
		ID:             pipeline.ID,
		TargetSystemID: pipeline.TargetSystemID,
		Route:          pipeline.Route,
		Status:         pipeline.Status,
		CurrentStage:   pipeline.CurrentStage,
		Message:        messages.PipelineCreatedSuccessfully,
	}, nil
}

func (p *pipelineService) GetPipeline(id int) (*domain.Pipeline, error) {
	return p.repo.FindByID(id)
}

func (p *pipelineService) UpdatePipeline(id int, input UpdatePipelineInput) error {
	pipeline, err := p.repo.FindByID(id)
	if err != nil {
		return err
	}

	if input.TargetSystemID != nil {
		pipeline.TargetSystemID = *input.TargetSystemID
	}
	if input.Route != nil {
		pipeline.Route = *input.Route
	}
	if input.Status != nil {
		pipeline.Status = *input.Status
	}
	if input.CurrentStage != nil {
		pipeline.CurrentStage = *input.CurrentStage
	}

	return p.repo.Update(pipeline)
}

func (p *pipelineService) DeletePipeline(id int) error {
	return p.repo.Delete(id)
}

func (p *pipelineService) ListPipelines() (ListPipelineOutput, error) {
	pipelines, err := p.repo.FindAll()
	if err != nil {
		return ListPipelineOutput{}, err
	}

	return ListPipelineOutput{
		Pipelines: pipelines,
		Total:     len(pipelines),
		Message:   messages.PipelinesListedSuccessfully,
	}, nil
}
