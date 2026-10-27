package usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/mocks"
	"github.com/Reddetk/Adev/internal/usecase"
)

func readyInput(targetSystemID int, route string) usecase.CreatePipelineInput {
	return usecase.CreatePipelineInput{
		TargetSystemID: targetSystemID, Route: route,
		Status: "ready", CurrentStage: "00",
	}
}

// TestCreatePipelineRejectsASecondGreenfieldPipeline: a target system owns one
// feature-design pipeline; the change routes walk later.
func TestCreatePipelineRejectsASecondGreenfieldPipeline(t *testing.T) {
	repo := mocks.NewMockPipelineRepository()
	repo.On("FindAll").Return([]domain.Pipeline{
		{ID: 7, TargetSystemID: 1, Route: string(domain.RouteGreenfield), Status: "completed", CurrentStage: "09"},
	}, nil).Maybe()
	service := usecase.NewPipelineService(repo)

	_, err := service.CreatePipeline(readyInput(1, string(domain.RouteGreenfield)))

	require.ErrorIs(t, err, usecase.ErrPipelineExists)
}

// TestCreatePipelineRejectsTheRouteOnlyForGreenfield: another greenfield target
// system and change routes of the same one are still allowed.
func TestCreatePipelineAllowsOtherOwnedRoutes(t *testing.T) {
	repo := mocks.NewMockPipelineRepository()
	repo.On("FindAll").Return([]domain.Pipeline{
		{ID: 7, TargetSystemID: 1, Route: string(domain.RouteGreenfield), Status: "active", CurrentStage: "03"},
	}, nil)
	repo.On("Save", mock.Anything).Return(nil).Twice()
	service := usecase.NewPipelineService(repo)

	first, err := service.CreatePipeline(readyInput(2, string(domain.RouteGreenfield)))
	require.NoError(t, err)
	// A mock store does not mint an id: the save call is all that matters.
	_ = first

	_, err = service.CreatePipeline(readyInput(1, string(domain.RouteChange)))
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

// TestCreatePipelineReportsTheStoreFailureAsIs.
func TestCreatePipelineReportsTheStoreFailure(t *testing.T) {
	repo := mocks.NewMockPipelineRepository()
	repo.On("FindAll").Return(nil, errors.New("db down"))
	service := usecase.NewPipelineService(repo)

	_, err := service.CreatePipeline(readyInput(1, string(domain.RouteGreenfield)))

	require.EqualError(t, err, "db down")
}
