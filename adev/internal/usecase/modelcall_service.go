package usecase

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/messages"
)

type modelcallService struct {
	repo ModelCallRepository
}

func NewModelCallService(repo ModelCallRepository) ModelCallUseCase {
	return &modelcallService{repo: repo}
}

func (m *modelcallService) CreateModelCall(input CreateModelCallInput) (CreateModelCallOutput, error) {
	modelcall := domain.ModelCall{
		RunID:     input.RunID,
		Profile:   input.Profile,
		Model:     input.Model,
		TokensIn:  input.TokensIn,
		TokensOut: input.TokensOut,
		Cost:      input.Cost,
		LatencyMs: input.LatencyMs,
	}

	if err := modelcall.Validate(); err != nil {
		return CreateModelCallOutput{}, err
	}

	if err := m.repo.Save(&modelcall); err != nil {
		return CreateModelCallOutput{}, err
	}

	return CreateModelCallOutput{
		ID:        modelcall.ID,
		RunID:     modelcall.RunID,
		Profile:   modelcall.Profile,
		Model:     modelcall.Model,
		TokensIn:  modelcall.TokensIn,
		TokensOut: modelcall.TokensOut,
		Cost:      modelcall.Cost,
		LatencyMs: modelcall.LatencyMs,
		Message:   messages.ModelCallCreatedSuccessfully,
	}, nil
}

func (m *modelcallService) GetModelCall(id int) (*domain.ModelCall, error) {
	return m.repo.FindByID(id)
}

func (m *modelcallService) UpdateModelCall(id int, input UpdateModelCallInput) error {
	modelcall, err := m.repo.FindByID(id)
	if err != nil {
		return err
	}

	if input.RunID != nil {
		modelcall.RunID = *input.RunID
	}
	if input.Profile != nil {
		modelcall.Profile = *input.Profile
	}
	if input.Model != nil {
		modelcall.Model = *input.Model
	}
	if input.TokensIn != nil {
		modelcall.TokensIn = *input.TokensIn
	}
	if input.TokensOut != nil {
		modelcall.TokensOut = *input.TokensOut
	}
	if input.Cost != nil {
		modelcall.Cost = *input.Cost
	}
	if input.LatencyMs != nil {
		modelcall.LatencyMs = *input.LatencyMs
	}

	return m.repo.Update(modelcall)
}

func (m *modelcallService) DeleteModelCall(id int) error {
	return m.repo.Delete(id)
}

func (m *modelcallService) ListModelCalls() (ListModelCallOutput, error) {
	modelcalls, err := m.repo.FindAll()
	if err != nil {
		return ListModelCallOutput{}, err
	}

	return ListModelCallOutput{
		ModelCalls: modelcalls,
		Total:      len(modelcalls),
		Message:    messages.ModelCallsListedSuccessfully,
	}, nil
}
