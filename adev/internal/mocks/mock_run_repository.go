package mocks

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/stretchr/testify/mock"
)

// MockRunRepository is a mock implementation of usecase.RunRepository
type MockRunRepository struct {
	mock.Mock
}

// Save mocks the Save method
func (m *MockRunRepository) Save(run *domain.Run) error {
	args := m.Called(run)
	return args.Error(0)
}

// FindByID mocks the FindByID method
func (m *MockRunRepository) FindByID(id int) (*domain.Run, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Run), args.Error(1)
}

// FindByPipelineID mocks the FindByPipelineID method
func (m *MockRunRepository) FindByPipelineID(pipelineid int) (*domain.Run, error) {
	args := m.Called(pipelineid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Run), args.Error(1)
}

// FindByStage mocks the FindByStage method
func (m *MockRunRepository) FindByStage(stage string) (*domain.Run, error) {
	args := m.Called(stage)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Run), args.Error(1)
}

// FindByStatus mocks the FindByStatus method
func (m *MockRunRepository) FindByStatus(status string) (*domain.Run, error) {
	args := m.Called(status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Run), args.Error(1)
}

// FindByCheckpointID mocks the FindByCheckpointID method
func (m *MockRunRepository) FindByCheckpointID(checkpointid string) (*domain.Run, error) {
	args := m.Called(checkpointid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Run), args.Error(1)
}

// FindByAttempts mocks the FindByAttempts method
func (m *MockRunRepository) FindByAttempts(attempts int) (*domain.Run, error) {
	args := m.Called(attempts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Run), args.Error(1)
}

// FindByLastError mocks the FindByLastError method
func (m *MockRunRepository) FindByLastError(lasterror string) (*domain.Run, error) {
	args := m.Called(lasterror)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Run), args.Error(1)
}

// Update mocks the Update method
func (m *MockRunRepository) Update(run *domain.Run) error {
	args := m.Called(run)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockRunRepository) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindAll mocks the FindAll method
func (m *MockRunRepository) FindAll() ([]domain.Run, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Run), args.Error(1)
}

// NewMockRunRepository creates a new mock repository
func NewMockRunRepository() *MockRunRepository {
	return &MockRunRepository{}
}
