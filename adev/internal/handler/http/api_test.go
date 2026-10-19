package http_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
	apphttp "github.com/Reddetk/Adev/internal/handler/http"
	"github.com/Reddetk/Adev/internal/mocks"
	"github.com/Reddetk/Adev/internal/repository"
	"github.com/Reddetk/Adev/internal/usecase"
)

const targetSystemID = 1

// fakeTargetSystem is the use case double for the CRUD endpoints.
type fakeTargetSystem struct {
	createdOutput usecase.CreateTargetSystemOutput
	createdErr    error
	getValue      *domain.TargetSystem
	getErr        error
	list          []domain.TargetSystem
	listErr       error
}

func (f *fakeTargetSystem) CreateTargetSystem(usecase.CreateTargetSystemInput) (usecase.CreateTargetSystemOutput, error) {
	return f.createdOutput, f.createdErr
}

func (f *fakeTargetSystem) GetTargetSystem(int) (*domain.TargetSystem, error) {
	return f.getValue, f.getErr
}

func (f *fakeTargetSystem) UpdateTargetSystem(int, usecase.UpdateTargetSystemInput) error {
	return nil
}

func (f *fakeTargetSystem) DeleteTargetSystem(int) error { return nil }

func (f *fakeTargetSystem) ListTargetSystems() (usecase.ListTargetSystemOutput, error) {
	return usecase.ListTargetSystemOutput{TargetSystems: f.list, Total: len(f.list), Message: "targetsystems listed successfully"}, f.listErr
}

type api struct {
	router  *mux.Router
	targets *fakeTargetSystem
	pipes   *mocks.MockPipelineUseCase
	runs    *mocks.MockRunUseCase
}

func newAPI() *api {
	fixture := &api{
		targets: &fakeTargetSystem{},
		pipes:   mocks.NewMockPipelineUseCase(),
		runs:    mocks.NewMockRunUseCase(),
	}

	router := mux.NewRouter()
	version := router.PathPrefix("/api/v1").Subrouter()
	apphttp.SetupTargetSystemRoutes(version, fixture.targets)
	apphttp.SetupPipelineRoutes(version, fixture.pipes)
	apphttp.SetupRunRoutes(version, fixture.runs)

	settings := usecase.NewSettingsService(repository.NewMemoryAgentConfigStore(targetSystemID))
	handler := apphttp.NewSettingsHandler(settings, usecase.NewPlanner(settings), fixture.pipes)
	apphttp.SetupSettingsRoutes(version, handler)

	fixture.router = router
	return fixture
}

func (a *api) do(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	a.router.ServeHTTP(recorder, request)
	return recorder
}

func (a *api) configure(t *testing.T) {
	t.Helper()
	require.Equal(t, http.StatusOK, a.do(t, http.MethodPut, "/api/v1/targetsystems/1/workspace", `{"dir":"D:\\work\\library"}`).Code)
	require.Equal(t, http.StatusOK, a.do(t, http.MethodPut, "/api/v1/targetsystems/1/agents/analyst",
		`{"provider":"ollama","baseUrl":"http://localhost:11434","model":"qwen3-coder:30b"}`).Code)
	require.Equal(t, http.StatusOK, a.do(t, http.MethodPut, "/api/v1/targetsystems/1/agents/architect",
		`{"provider":"openrouter","model":"qwen/qwen3-235b","apiKey":"sk-secret"}`).Code)
}

// --- target systems ---------------------------------------------------------

func TestCreateTargetSystemHappyPath(t *testing.T) {
	fixture := newAPI()
	fixture.targets.createdOutput = usecase.CreateTargetSystemOutput{ID: 1, Name: "library", Message: "targetsystem created successfully"}

	// No repourl: a target system does not ask for a GitHub repository.
	response := fixture.do(t, http.MethodPost, "/api/v1/targetsystems", `{"name":"library","description":"books"}`)

	assert.Equal(t, http.StatusCreated, response.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "library", body["name"])
	assert.NotEmpty(t, body["message"])
}

func TestCreateTargetSystemRejectsBrokenJSON(t *testing.T) {
	response := newAPI().do(t, http.MethodPost, "/api/v1/targetsystems", `{"name":`)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestCreateTargetSystemRejectsAnEmptyName(t *testing.T) {
	response := newAPI().do(t, http.MethodPost, "/api/v1/targetsystems", `{"name":"","repourl":"x","description":"y"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
}

func TestCreateTargetSystemReportsAUseCaseFailure(t *testing.T) {
	fixture := newAPI()
	fixture.targets.createdErr = errors.New("database is down")

	response := fixture.do(t, http.MethodPost, "/api/v1/targetsystems", `{"name":"library","repourl":"x","description":"y"}`)

	assert.Equal(t, http.StatusInternalServerError, response.Code)
}

func TestGetTargetSystemNotFound(t *testing.T) {
	fixture := newAPI()
	fixture.targets.getErr = errors.New("not found")

	response := fixture.do(t, http.MethodGet, "/api/v1/targetsystems/7", "")

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestGetTargetSystemRejectsANonNumericID(t *testing.T) {
	response := newAPI().do(t, http.MethodGet, "/api/v1/targetsystems/abc", "")

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestListTargetSystemsHappyPath(t *testing.T) {
	fixture := newAPI()
	fixture.targets.list = []domain.TargetSystem{{ID: 1}, {ID: 2}}

	response := fixture.do(t, http.MethodGet, "/api/v1/targetsystems", "")

	assert.Equal(t, http.StatusOK, response.Code)
	var body struct {
		Total int `json:"total"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, 2, body.Total)
}

// --- pipelines and runs -----------------------------------------------------

func TestCreatePipelineHappyPath(t *testing.T) {
	fixture := newAPI()
	fixture.pipes.On("CreatePipeline", usecase.CreatePipelineInput{
		TargetSystemID: 3, Route: "greenfield", Status: "ready", CurrentStage: "00",
	}).Return(usecase.CreatePipelineOutput{ID: 5, Message: "pipeline created successfully"}, nil).Once()

	response := fixture.do(t, http.MethodPost, "/api/v1/pipelines", `{"targetsystemid":3,"route":"greenfield","status":"ready","currentstage":"00"}`)

	assert.Equal(t, http.StatusCreated, response.Code)
	fixture.pipes.AssertExpectations(t)
}

func TestCreatePipelineRejectsAnEmptyRoute(t *testing.T) {
	fixture := newAPI()

	response := fixture.do(t, http.MethodPost, "/api/v1/pipelines", `{"targetsystemid":3,"route":"","status":"ready","currentstage":"00"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	fixture.pipes.AssertNotCalled(t, "CreatePipeline", mock.Anything)
}

func TestGetPipelineNotFound(t *testing.T) {
	fixture := newAPI()
	fixture.pipes.On("GetPipeline", 7).Return(nil, errors.New("pipeline not found")).Once()

	response := fixture.do(t, http.MethodGet, "/api/v1/pipelines/7", "")

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestCreateRunHappyPath(t *testing.T) {
	fixture := newAPI()
	fixture.runs.On("CreateRun", mock.Anything).
		Return(usecase.CreateRunOutput{ID: 4, Stage: "00", Message: "run created successfully"}, nil).Once()

	response := fixture.do(t, http.MethodPost, "/api/v1/runs", `{"pipelineid":3,"stage":"00","status":"running","checkpointid":"cp-1"}`)

	assert.Equal(t, http.StatusCreated, response.Code)
	fixture.runs.AssertExpectations(t)
}

func TestCreateRunRejectsBrokenJSON(t *testing.T) {
	response := newAPI().do(t, http.MethodPost, "/api/v1/runs", `not json`)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

// --- settings and plan ------------------------------------------------------

func TestWorkspaceRoundTrip(t *testing.T) {
	fixture := newAPI()

	put := fixture.do(t, http.MethodPut, "/api/v1/targetsystems/1/workspace", `{"dir":"D:\\work\\library"}`)
	assert.Equal(t, http.StatusOK, put.Code)

	get := fixture.do(t, http.MethodGet, "/api/v1/targetsystems/1/workspace", "")
	assert.Equal(t, http.StatusOK, get.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(get.Body.Bytes(), &body))
	assert.Equal(t, `D:\work\library`, body["dir"])
}

func TestWorkspaceRejectsAnEmptyDirectory(t *testing.T) {
	response := newAPI().do(t, http.MethodPut, "/api/v1/targetsystems/1/workspace", `{"dir":"   "}`)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
}

func TestWorkspaceUnknownTargetSystem(t *testing.T) {
	response := newAPI().do(t, http.MethodPut, "/api/v1/targetsystems/999/workspace", `{"dir":"D:\\work"}`)

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestSetAgentHappyPathHidesTheKey(t *testing.T) {
	fixture := newAPI()

	response := fixture.do(t, http.MethodPut, "/api/v1/targetsystems/1/agents/analyst",
		`{"provider":"ollama","baseUrl":"http://localhost:11434","model":"qwen3-coder:30b","apiKey":"sk-secret"}`)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.NotContains(t, response.Body.String(), "sk-secret", "the api key must never come back")
	assert.Contains(t, response.Body.String(), `"hasApiKey":true`)
}

func TestListAgentsNeverReturnsTheKey(t *testing.T) {
	fixture := newAPI()
	fixture.configure(t)

	response := fixture.do(t, http.MethodGet, "/api/v1/targetsystems/1/agents", "")

	assert.Equal(t, http.StatusOK, response.Code)
	assert.NotContains(t, response.Body.String(), "sk-secret")
	assert.Contains(t, response.Body.String(), `"role":"architect"`)
}

func TestSetAgentRejectsACloudProviderWithoutKey(t *testing.T) {
	response := newAPI().do(t, http.MethodPut, "/api/v1/targetsystems/1/agents/architect",
		`{"provider":"openrouter","model":"qwen/qwen3-235b"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
}

func TestSetAgentRejectsAnUnknownRole(t *testing.T) {
	response := newAPI().do(t, http.MethodPut, "/api/v1/targetsystems/1/agents/wizard",
		`{"provider":"ollama","baseUrl":"http://localhost:11434","model":"m"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
}

func TestSetAgentUnknownTargetSystem(t *testing.T) {
	response := newAPI().do(t, http.MethodPut, "/api/v1/targetsystems/999/agents/analyst",
		`{"provider":"ollama","baseUrl":"http://localhost:11434","model":"qwen3-coder:30b"}`)

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestPlanReportsWhatIsMissing(t *testing.T) {
	response := newAPI().do(t, http.MethodGet, "/api/v1/targetsystems/1/plan", "")

	assert.Equal(t, http.StatusOK, response.Code)
	var plan struct {
		Readiness struct {
			Ready   bool     `json:"ready"`
			Missing []string `json:"missing"`
		} `json:"readiness"`
		Stages []domain.Stage `json:"stages"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
	assert.False(t, plan.Readiness.Ready)
	assert.Len(t, plan.Readiness.Missing, 3)
	assert.Len(t, plan.Stages, 7)
}

func TestPlanIsReadyAfterConfiguration(t *testing.T) {
	fixture := newAPI()
	fixture.configure(t)

	response := fixture.do(t, http.MethodGet, "/api/v1/targetsystems/1/plan", "")

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"ready":true`)
}

func TestPlanRejectsAnUnknownRoute(t *testing.T) {
	response := newAPI().do(t, http.MethodGet, "/api/v1/targetsystems/1/plan?route=change", "")

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestPipelineIsRefusedUntilTheTargetSystemIsReady(t *testing.T) {
	fixture := newAPI()

	response := fixture.do(t, http.MethodPost, "/api/v1/targetsystems/1/pipelines", `{"route":"greenfield"}`)

	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Contains(t, response.Body.String(), `"ready":false`)
	fixture.pipes.AssertNotCalled(t, "CreatePipeline", mock.Anything)
}

func TestPipelineStartsAtTheFirstStageWhenReady(t *testing.T) {
	fixture := newAPI()
	fixture.configure(t)
	fixture.pipes.On("CreatePipeline", usecase.CreatePipelineInput{
		TargetSystemID: 1, Route: "greenfield", Status: "ready", CurrentStage: "00",
	}).Return(usecase.CreatePipelineOutput{ID: 9, Message: "pipeline created successfully"}, nil).Once()

	response := fixture.do(t, http.MethodPost, "/api/v1/targetsystems/1/pipelines", `{"route":"greenfield"}`)

	assert.Equal(t, http.StatusCreated, response.Code)
	assert.Contains(t, response.Body.String(), `"stage":"00"`)
	fixture.pipes.AssertExpectations(t)
}
