package examples

import (
	"errors"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/mocks"
	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Compile-time assertions that the generated mocks satisfy the real interfaces.
var (
	_ usecase.PipelineRepository = (*mocks.MockPipelineRepository)(nil)
	_ usecase.PipelineUseCase    = (*mocks.MockPipelineUseCase)(nil)
)

// Example: stubbing repository methods and verifying expectations.
func TestMockPipelineRepository_Usage(t *testing.T) {
	mockRepo := mocks.NewMockPipelineRepository()

	expected := &domain.Pipeline{ID: 1}
	mockRepo.On("FindByID", 1).Return(expected, nil)
	mockRepo.On("Save", mock.AnythingOfType("*domain.Pipeline")).Return(nil)

	got, err := mockRepo.FindByID(1)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)

	err = mockRepo.Save(&domain.Pipeline{})
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Example: stubbing an error return.
func TestMockPipelineRepository_NotFound(t *testing.T) {
	mockRepo := mocks.NewMockPipelineRepository()

	expectedErr := errors.New("pipeline not found")
	mockRepo.On("FindByID", 999).Return(nil, expectedErr)

	got, err := mockRepo.FindByID(999)
	assert.Nil(t, got)
	assert.Equal(t, expectedErr, err)

	mockRepo.AssertExpectations(t)
}

// Example: stubbing use-case methods.
func TestMockPipelineUseCase_Usage(t *testing.T) {
	mockUC := mocks.NewMockPipelineUseCase()

	mockUC.On("GetPipeline", 1).Return(&domain.Pipeline{ID: 1}, nil)
	mockUC.On("DeletePipeline", 1).Return(nil)

	got, err := mockUC.GetPipeline(1)
	assert.NoError(t, err)
	assert.NotNil(t, got)

	err = mockUC.DeletePipeline(1)
	assert.NoError(t, err)

	mockUC.AssertExpectations(t)
}
