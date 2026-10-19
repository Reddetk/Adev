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
	_ usecase.RunRepository = (*mocks.MockRunRepository)(nil)
	_ usecase.RunUseCase    = (*mocks.MockRunUseCase)(nil)
)

// Example: stubbing repository methods and verifying expectations.
func TestMockRunRepository_Usage(t *testing.T) {
	mockRepo := mocks.NewMockRunRepository()

	expected := &domain.Run{ID: 1}
	mockRepo.On("FindByID", 1).Return(expected, nil)
	mockRepo.On("Save", mock.AnythingOfType("*domain.Run")).Return(nil)

	got, err := mockRepo.FindByID(1)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)

	err = mockRepo.Save(&domain.Run{})
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Example: stubbing an error return.
func TestMockRunRepository_NotFound(t *testing.T) {
	mockRepo := mocks.NewMockRunRepository()

	expectedErr := errors.New("run not found")
	mockRepo.On("FindByID", 999).Return(nil, expectedErr)

	got, err := mockRepo.FindByID(999)
	assert.Nil(t, got)
	assert.Equal(t, expectedErr, err)

	mockRepo.AssertExpectations(t)
}

// Example: stubbing use-case methods.
func TestMockRunUseCase_Usage(t *testing.T) {
	mockUC := mocks.NewMockRunUseCase()

	mockUC.On("GetRun", 1).Return(&domain.Run{ID: 1}, nil)
	mockUC.On("DeleteRun", 1).Return(nil)

	got, err := mockUC.GetRun(1)
	assert.NoError(t, err)
	assert.NotNil(t, got)

	err = mockUC.DeleteRun(1)
	assert.NoError(t, err)

	mockUC.AssertExpectations(t)
}
