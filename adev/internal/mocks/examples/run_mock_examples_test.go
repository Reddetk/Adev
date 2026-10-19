package examples

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/mocks"
	"github.com/Reddetk/Adev/internal/usecase"
)

// Compile-time assertions that the generated mocks satisfy the real interfaces.
var (
	_ usecase.RunRepository = (*mocks.MockRunRepository)(nil)
	_ usecase.RunUseCase    = (*mocks.MockRunUseCase)(nil)
)

func TestCreateRunSavesTheMappedFieldsOnce(t *testing.T) {
	repo := mocks.NewMockRunRepository()
	var saved *domain.Run
	repo.On("Save", mock.Anything).
		Run(func(args mock.Arguments) {
			saved = args.Get(0).(*domain.Run)
			saved.ID = 99
		}).
		Return(nil).
		Once()

	output, err := usecase.NewRunService(repo).CreateRun(usecase.CreateRunInput{
		PipelineID:   7,
		Stage:        "00",
		Status:       "running",
		CheckpointID: "cp-1",
		Attempts:     1,
	})

	require.NoError(t, err)
	require.NotNil(t, saved, "the service must call Save")
	assert.Equal(t, 7, saved.PipelineID)
	assert.Equal(t, "00", saved.Stage)
	assert.Equal(t, "running", saved.Status)
	assert.Equal(t, "cp-1", saved.CheckpointID)
	assert.Equal(t, 1, saved.Attempts)
	assert.Equal(t, uint(99), output.ID)
	assert.Equal(t, "run created successfully", output.Message)
	repo.AssertNumberOfCalls(t, "Save", 1)
	repo.AssertExpectations(t)
}

func TestCreateRunRejectsAnEmptyStageBeforeTheRepository(t *testing.T) {
	repo := mocks.NewMockRunRepository()

	_, err := usecase.NewRunService(repo).CreateRun(usecase.CreateRunInput{
		PipelineID:   7,
		Status:       "running",
		CheckpointID: "cp-1",
	})

	require.Error(t, err)
	repo.AssertNotCalled(t, "Save", mock.Anything)
}

func TestCreateRunReturnsTheRepositoryError(t *testing.T) {
	repo := mocks.NewMockRunRepository()
	repo.On("Save", mock.Anything).Return(errors.New("database is down")).Once()

	_, err := usecase.NewRunService(repo).CreateRun(usecase.CreateRunInput{
		PipelineID:   7,
		Stage:        "00",
		Status:       "running",
		CheckpointID: "cp-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database is down")
	repo.AssertExpectations(t)
}

// UpdateRun reads the stored run first and then writes it back, so the mock has
// to serve both calls in that order.
func TestUpdateRunReadsThenWrites(t *testing.T) {
	repo := mocks.NewMockRunRepository()
	repo.On("FindByID", 3).Return(&domain.Run{ID: 3, PipelineID: 7, Stage: "00", Status: "running", CheckpointID: "cp-1"}, nil).Once()

	var updated *domain.Run
	repo.On("Update", mock.Anything).
		Run(func(args mock.Arguments) { updated = args.Get(0).(*domain.Run) }).
		Return(nil).
		Once()

	status := "failed"
	require.NoError(t, usecase.NewRunService(repo).UpdateRun(3, usecase.UpdateRunInput{Status: &status}))

	require.NotNil(t, updated)
	assert.Equal(t, "failed", updated.Status, "the new status must be written")
	assert.Equal(t, "cp-1", updated.CheckpointID, "untouched fields must stay")
	repo.AssertExpectations(t)
}

func TestUpdateRunStopsWhenTheRunIsMissing(t *testing.T) {
	repo := mocks.NewMockRunRepository()
	repo.On("FindByID", 3).Return(nil, errors.New("run not found")).Once()

	status := "failed"
	err := usecase.NewRunService(repo).UpdateRun(3, usecase.UpdateRunInput{Status: &status})

	require.Error(t, err)
	repo.AssertNotCalled(t, "Update", mock.Anything)
}

func TestListRunsReportsTheRepositoryResult(t *testing.T) {
	repo := mocks.NewMockRunRepository()
	repo.On("FindAll").Return([]domain.Run{{ID: 1}, {ID: 2}, {ID: 3}}, nil).Once()

	output, err := usecase.NewRunService(repo).ListRuns()

	require.NoError(t, err)
	assert.Equal(t, 3, output.Total)
	assert.Equal(t, "runs listed successfully", output.Message)
	repo.AssertExpectations(t)
}
