package usecase

import "github.com/Reddetk/Adev/internal/domain"

type OperatorRequestUseCase interface {
	CreateOperatorRequest(input CreateOperatorRequestInput) (CreateOperatorRequestOutput, error)
	GetOperatorRequest(id int) (*domain.OperatorRequest, error)
	UpdateOperatorRequest(id int, input UpdateOperatorRequestInput) error
	DeleteOperatorRequest(id int) error
	ListOperatorRequests() (ListOperatorRequestOutput, error)
}
