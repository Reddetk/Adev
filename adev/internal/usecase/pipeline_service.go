package usecase

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/messages"
)

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
