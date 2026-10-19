package usecase

import "github.com/Reddetk/Adev/internal/domain"

type RunUseCase interface {
	CreateRun(input CreateRunInput) (CreateRunOutput, error)
	GetRun(id int) (*domain.Run, error)
	UpdateRun(id int, input UpdateRunInput) error
	DeleteRun(id int) error
	ListRuns() (ListRunOutput, error)
}
