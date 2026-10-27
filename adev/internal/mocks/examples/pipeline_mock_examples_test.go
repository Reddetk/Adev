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
	_ usecase.PipelineRepository = (*mocks.MockPipelineRepository)(nil)
	_ usecase.PipelineUseCase    = (*mocks.MockPipelineUseCase)(nil)
)

// The repository assigns the ID on Save. The mock emulates that contract so the
// service is driven the same way the pgx store drives it.
func TestCreatePipelineSavesOnceAndKeepsTheAssignedID(t *testing.T) {
	repo := mocks.NewMockPipelineRepository()
	repo.On("FindAll").Return(nil, nil).Once()
	var saved *domain.Pipeline
	repo.On("Save", mock.Anything).
		Run(func(args mock.Arguments) {
			saved = args.Get(0).(*domain.Pipeline)
			saved.ID = 42
		}).
		Return(nil).
		Once()

	output, err := usecase.NewPipelineService(repo).CreatePipeline(usecase.CreatePipelineInput{
		TargetSystemID: 7,
		Route:          "greenfield",
		Status:         "ready",
		CurrentStage:   "00",
	})

	require.NoError(t, err)
	require.NotNil(t, saved, "the service must call Save")
	assert.Equal(t, 7, saved.TargetSystemID)
	assert.Equal(t, "greenfield", saved.Route)
	assert.Equal(t, "ready", saved.Status)
	assert.Equal(t, "00", saved.CurrentStage)
	assert.Equal(t, uint(42), output.ID, "the output must carry the id assigned by the repository")
	assert.Equal(t, "pipeline created successfully", output.Message)
	repo.AssertNumberOfCalls(t, "Save", 1)
	repo.AssertExpectations(t)
}

func TestCreatePipelineRejectsAnEmptyRouteBeforeTheRepository(t *testing.T) {
	repo := mocks.NewMockPipelineRepository()

	_, err := usecase.NewPipelineService(repo).CreatePipeline(usecase.CreatePipelineInput{
		TargetSystemID: 7,
		Status:         "ready",
		CurrentStage:   "00",
	})

	require.Error(t, err)
	repo.AssertNotCalled(t, "Save", mock.Anything)
}

func TestCreatePipelineReturnsTheRepositoryError(t *testing.T) {
	repo := mocks.NewMockPipelineRepository()
	repo.On("FindAll").Return(nil, nil).Once()
	repo.On("Save", mock.Anything).Return(errors.New("database is down")).Once()

	output, err := usecase.NewPipelineService(repo).CreatePipeline(usecase.CreatePipelineInput{
		TargetSystemID: 7,
		Route:          "greenfield",
		Status:         "ready",
		CurrentStage:   "00",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "database is down")
	assert.Zero(t, output.ID, "no output on failure")
	repo.AssertExpectations(t)
}

func TestListPipelinesReportsTheRepositoryResult(t *testing.T) {
	repo := mocks.NewMockPipelineRepository()
	repo.On("FindAll").Return([]domain.Pipeline{{ID: 1}, {ID: 2}}, nil).Once()

	output, err := usecase.NewPipelineService(repo).ListPipelines()

	require.NoError(t, err)
	assert.Len(t, output.Pipelines, 2)
	assert.Equal(t, 2, output.Total)
	assert.Equal(t, "pipelines listed successfully", output.Message)
	repo.AssertExpectations(t)
}

func TestGetPipelinePassesTheRepositoryErrorThrough(t *testing.T) {
	repo := mocks.NewMockPipelineRepository()
	notFound := errors.New("pipeline not found")
	repo.On("FindByID", 5).Return(nil, notFound).Once()

	got, err := usecase.NewPipelineService(repo).GetPipeline(5)

	assert.Nil(t, got)
	assert.Equal(t, notFound, err)
	repo.AssertExpectations(t)
}
