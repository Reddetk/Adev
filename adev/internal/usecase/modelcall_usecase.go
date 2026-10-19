package usecase

import "github.com/Reddetk/Adev/internal/domain"

type ModelCallUseCase interface {
	CreateModelCall(input CreateModelCallInput) (CreateModelCallOutput, error)
	GetModelCall(id int) (*domain.ModelCall, error)
	UpdateModelCall(id int, input UpdateModelCallInput) error
	DeleteModelCall(id int) error
	ListModelCalls() (ListModelCallOutput, error)
}
