package usecase

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/messages"
)

type operatorrequestService struct {
	repo OperatorRequestRepository
}

func NewOperatorRequestService(repo OperatorRequestRepository) OperatorRequestUseCase {
	return &operatorrequestService{repo: repo}
}

func (o *operatorrequestService) CreateOperatorRequest(input CreateOperatorRequestInput) (CreateOperatorRequestOutput, error) {
	if err := input.Validate(); err != nil {
		return CreateOperatorRequestOutput{}, err
	}

	operatorrequest := domain.OperatorRequest{
		PipelineID: input.PipelineID,
		Pointer:    input.Pointer,
		Question:   input.Question,
		Blocking:   input.Blocking,
		Status:     input.Status,
	}

	if err := operatorrequest.Validate(); err != nil {
		return CreateOperatorRequestOutput{}, err
	}

	if err := o.repo.Save(&operatorrequest); err != nil {
		return CreateOperatorRequestOutput{}, err
	}

	return CreateOperatorRequestOutput{
		ID:         operatorrequest.ID,
		PipelineID: operatorrequest.PipelineID,
		Pointer:    operatorrequest.Pointer,
		Question:   operatorrequest.Question,
		Blocking:   operatorrequest.Blocking,
		Status:     operatorrequest.Status,
		Message:    messages.OperatorRequestCreatedSuccessfully,
	}, nil
}

func (o *operatorrequestService) GetOperatorRequest(id int) (*domain.OperatorRequest, error) {
	return o.repo.FindByID(id)
}

func (o *operatorrequestService) UpdateOperatorRequest(id int, input UpdateOperatorRequestInput) error {
	operatorrequest, err := o.repo.FindByID(id)
	if err != nil {
		return err
	}

	if input.PipelineID != nil {
		operatorrequest.PipelineID = *input.PipelineID
	}
	if input.Pointer != nil {
		operatorrequest.Pointer = *input.Pointer
	}
	if input.Question != nil {
		operatorrequest.Question = *input.Question
	}
	if input.Blocking != nil {
		operatorrequest.Blocking = *input.Blocking
	}
	if input.Status != nil {
		operatorrequest.Status = *input.Status
	}

	return o.repo.Update(operatorrequest)
}

func (o *operatorrequestService) DeleteOperatorRequest(id int) error {
	return o.repo.Delete(id)
}

func (o *operatorrequestService) ListOperatorRequests() (ListOperatorRequestOutput, error) {
	operatorrequests, err := o.repo.FindAll()
	if err != nil {
		return ListOperatorRequestOutput{}, err
	}

	return ListOperatorRequestOutput{
		OperatorRequests: operatorrequests,
		Total:            len(operatorrequests),
		Message:          messages.OperatorRequestsListedSuccessfully,
	}, nil
}
