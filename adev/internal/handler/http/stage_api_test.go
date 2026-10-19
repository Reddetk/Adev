package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
	apphttp "github.com/Reddetk/Adev/internal/handler/http"
	"github.com/Reddetk/Adev/internal/usecase"
)

type fakeStageRunner struct {
	outcome usecase.StageOutcome
	err     error
	request usecase.StageRequest
	called  bool
}

func (f *fakeStageRunner) Run(_ context.Context, request usecase.StageRequest) (usecase.StageOutcome, error) {
	f.called = true
	f.request = request
	return f.outcome, f.err
}

type stageAPI struct {
	router *mux.Router
	runner *fakeStageRunner
}

func newStageAPI(outcome usecase.StageOutcome, err error) *stageAPI {
	runner := &fakeStageRunner{outcome: outcome, err: err}
	router := mux.NewRouter()
	version := router.PathPrefix("/api/v1").Subrouter()
	apphttp.SetupStageRoutes(version, apphttp.NewStageHandler(runner))
	return &stageAPI{router: router, runner: runner}
}

func (a *stageAPI) run(t *testing.T, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	a.router.ServeHTTP(recorder, request)
	return recorder
}

func TestRunStageHappyPath(t *testing.T) {
	fixture := newStageAPI(usecase.StageOutcome{Stage: "00", FormID: "FRM-1-00", FormVersion: 1, Attempts: 1}, nil)

	response := fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"1","route":"greenfield"}`)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"formId":"FRM-1-00"`)
	assert.Contains(t, response.Body.String(), `"attempts":1`)
}

func TestRunStageReportsABlockedStageWithGaps(t *testing.T) {
	fixture := newStageAPI(usecase.StageOutcome{Stage: "00", Attempts: 3, Blocked: true, Gaps: []string{"goals: must contain at least one entry"}}, nil)

	response := fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"1"}`)

	assert.Equal(t, http.StatusOK, response.Code, "a blocked stage is an outcome, not an error")
	assert.Contains(t, response.Body.String(), `"blocked":true`)
	assert.Contains(t, response.Body.String(), "goals")
}

func TestRunStageReportsAMissingInput(t *testing.T) {
	fixture := newStageAPI(usecase.StageOutcome{}, usecase.ErrStageInputMissing)

	response := fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"1"}`)

	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Contains(t, response.Body.String(), "input form is missing")
}

func TestRunStageReportsAMissingAgentConfig(t *testing.T) {
	fixture := newStageAPI(usecase.StageOutcome{}, usecase.ErrAgentNotConfigured)

	response := fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"1"}`)

	assert.Equal(t, http.StatusConflict, response.Code)
}

func TestRunStageReportsAModelFailure(t *testing.T) {
	fixture := newStageAPI(usecase.StageOutcome{}, errors.New("model call to http://localhost:11434/v1: connection refused"))

	response := fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"1"}`)

	assert.Equal(t, http.StatusBadGateway, response.Code)
	assert.Contains(t, response.Body.String(), "connection refused")
}

func TestRunStageRejectsBadRequests(t *testing.T) {
	fixture := newStageAPI(usecase.StageOutcome{}, nil)

	assert.Equal(t, http.StatusBadRequest, fixture.run(t, "/api/v1/targetsystems/abc/stages/00/run", `{"pipelineId":"1"}`).Code)
	assert.Equal(t, http.StatusBadRequest, fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `not json`).Code)
	assert.Equal(t, http.StatusBadRequest, fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"route":"greenfield"}`).Code)
	assert.Equal(t, http.StatusBadRequest, fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"1","route":"change"}`).Code)
}

func TestRunStagePassesTheRequestThrough(t *testing.T) {
	fixture := newStageAPI(usecase.StageOutcome{Stage: "00"}, nil)

	fixture.run(t, "/api/v1/targetsystems/7/stages/00/run", `{"pipelineId":"PIPE-9"}`)

	require.True(t, fixture.runner.called, "the runner must be called")
	assert.Equal(t, usecase.StageRequest{
		PipelineID: "PIPE-9", TargetSystemID: 7, Route: domain.RouteGreenfield, StageID: "00",
	}, fixture.runner.request)
}
