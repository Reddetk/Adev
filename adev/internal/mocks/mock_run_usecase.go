package mocks

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/stretchr/testify/mock"
)

// MockRunUseCase is a mock implementation of usecase.RunUseCase
type MockRunUseCase struct {
	mock.Mock
}

// CreateRun mocks the CreateRun method
func (m *MockRunUseCase) CreateRun(input usecase.CreateRunInput) (usecase.CreateRunOutput, error) {
	args := m.Called(input)
	return args.Get(0).(usecase.CreateRunOutput), args.Error(1)
}

// GetRun mocks the GetRun method
func (m *MockRunUseCase) GetRun(id int) (*domain.Run, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Run), args.Error(1)
}

// UpdateRun mocks the UpdateRun method
func (m *MockRunUseCase) UpdateRun(id int, input usecase.UpdateRunInput) error {
	args := m.Called(id, input)
	return args.Error(0)
}

// DeleteRun mocks the DeleteRun method
func (m *MockRunUseCase) DeleteRun(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

// ListRuns mocks the ListRuns method
func (m *MockRunUseCase) ListRuns() (usecase.ListRunOutput, error) {
	args := m.Called()
	return args.Get(0).(usecase.ListRunOutput), args.Error(1)
}

// NewMockRunUseCase creates a new mock use case
func NewMockRunUseCase() *MockRunUseCase {
	return &MockRunUseCase{}
}
