package mocks

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/stretchr/testify/mock"
)

// MockPipelineUseCase is a mock implementation of usecase.PipelineUseCase
type MockPipelineUseCase struct {
	mock.Mock
}

// CreatePipeline mocks the CreatePipeline method
func (m *MockPipelineUseCase) CreatePipeline(input usecase.CreatePipelineInput) (usecase.CreatePipelineOutput, error) {
	args := m.Called(input)
	return args.Get(0).(usecase.CreatePipelineOutput), args.Error(1)
}

// GetPipeline mocks the GetPipeline method
func (m *MockPipelineUseCase) GetPipeline(id int) (*domain.Pipeline, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Pipeline), args.Error(1)
}

// UpdatePipeline mocks the UpdatePipeline method
func (m *MockPipelineUseCase) UpdatePipeline(id int, input usecase.UpdatePipelineInput) error {
	args := m.Called(id, input)
	return args.Error(0)
}

// DeletePipeline mocks the DeletePipeline method
func (m *MockPipelineUseCase) DeletePipeline(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

// ListPipelines mocks the ListPipelines method
func (m *MockPipelineUseCase) ListPipelines() (usecase.ListPipelineOutput, error) {
	args := m.Called()
	return args.Get(0).(usecase.ListPipelineOutput), args.Error(1)
}

// NewMockPipelineUseCase creates a new mock use case
func NewMockPipelineUseCase() *MockPipelineUseCase {
	return &MockPipelineUseCase{}
}
