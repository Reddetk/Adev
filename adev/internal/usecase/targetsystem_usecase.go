package usecase

import "github.com/Reddetk/Adev/internal/domain"

type TargetSystemUseCase interface {
	CreateTargetSystem(input CreateTargetSystemInput) (CreateTargetSystemOutput, error)
	GetTargetSystem(id int) (*domain.TargetSystem, error)
	UpdateTargetSystem(id int, input UpdateTargetSystemInput) error
	DeleteTargetSystem(id int) error
	ListTargetSystems() (ListTargetSystemOutput, error)
}
