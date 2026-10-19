package usecase

import "github.com/Reddetk/Adev/internal/domain"

type PipelineUseCase interface {
	CreatePipeline(input CreatePipelineInput) (CreatePipelineOutput, error)
	GetPipeline(id int) (*domain.Pipeline, error)
	UpdatePipeline(id int, input UpdatePipelineInput) error
	DeletePipeline(id int) error
	ListPipelines() (ListPipelineOutput, error)
}
