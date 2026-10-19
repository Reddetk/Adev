package mocks

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/stretchr/testify/mock"
)

// MockPipelineRepository is a mock implementation of usecase.PipelineRepository
type MockPipelineRepository struct {
	mock.Mock
}

// Save mocks the Save method
func (m *MockPipelineRepository) Save(pipeline *domain.Pipeline) error {
	args := m.Called(pipeline)
	return args.Error(0)
}

// FindByID mocks the FindByID method
func (m *MockPipelineRepository) FindByID(id int) (*domain.Pipeline, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Pipeline), args.Error(1)
}

// FindByTargetSystemID mocks the FindByTargetSystemID method
func (m *MockPipelineRepository) FindByTargetSystemID(targetsystemid int) (*domain.Pipeline, error) {
	args := m.Called(targetsystemid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Pipeline), args.Error(1)
}

// FindByRoute mocks the FindByRoute method
func (m *MockPipelineRepository) FindByRoute(route string) (*domain.Pipeline, error) {
	args := m.Called(route)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Pipeline), args.Error(1)
}

// FindByStatus mocks the FindByStatus method
func (m *MockPipelineRepository) FindByStatus(status string) (*domain.Pipeline, error) {
	args := m.Called(status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Pipeline), args.Error(1)
}

// FindByCurrentStage mocks the FindByCurrentStage method
func (m *MockPipelineRepository) FindByCurrentStage(currentstage string) (*domain.Pipeline, error) {
	args := m.Called(currentstage)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Pipeline), args.Error(1)
}

// Update mocks the Update method
func (m *MockPipelineRepository) Update(pipeline *domain.Pipeline) error {
	args := m.Called(pipeline)
	return args.Error(0)
}

// Delete mocks the Delete method
func (m *MockPipelineRepository) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

// FindAll mocks the FindAll method
func (m *MockPipelineRepository) FindAll() ([]domain.Pipeline, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Pipeline), args.Error(1)
}

// NewMockPipelineRepository creates a new mock repository
func NewMockPipelineRepository() *MockPipelineRepository {
	return &MockPipelineRepository{}
}
