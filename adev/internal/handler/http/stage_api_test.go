package http_test

import (
	"context"
	"encoding/json"
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

// fakeRunStarter stands for the StageRunService: the answer is the accepted run
// row, the chain runs in the background.
type fakeRunStarter struct {
	run     *domain.Run
	err     error
	request usecase.StageRequest
}

func (f *fakeRunStarter) Start(_ context.Context, request usecase.StageRequest) (*domain.Run, error) {
	f.request = request
	if f.err == nil && f.run == nil {
		f.run = &domain.Run{ID: 7, PipelineID: 9, Stage: request.StageID, Status: "running", CheckpointID: string(request.Route)}
	}
	return f.run, f.err
}

type stageAPI struct {
	router *mux.Router
	start  *fakeRunStarter
}

func newStageAPI(starter *fakeRunStarter) *stageAPI {
	router := mux.NewRouter()
	version := router.PathPrefix("/api/v1").Subrouter()
	apphttp.SetupStageRoutes(version, apphttp.NewStageHandler(starter))
	return &stageAPI{router: router, start: starter}
}

func (a *stageAPI) run(t *testing.T, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	a.router.ServeHTTP(recorder, request)
	return recorder
}

// TestRunStageAcceptsTheRun: the stage chain works in the background, so the
// answer is the accepted run row and never the result of attempts.
func TestRunStageAcceptsTheRun(t *testing.T) {
	starter := &fakeRunStarter{}
	fixture := newStageAPI(starter)

	response := fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"9","route":"greenfield"}`)

	require.Equal(t, http.StatusAccepted, response.Code)
	var accepted domain.Run
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &accepted))
	assert.Equal(t, uint(7), accepted.ID)
	assert.Equal(t, "running", accepted.Status)
	assert.Equal(t, "00", accepted.Stage)
}

func TestRunStageAnswersAnAcceptEvenForABlockedOutcome(t *testing.T) {
	// The blocked outcome belongs to the run row now: the POST still answers at once.
	starter := &fakeRunStarter{}
	fixture := newStageAPI(starter)

	assert.Equal(t, http.StatusAccepted, fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"9"}`).Code)
}

func TestRunStageReportsARecordingFailure(t *testing.T) {
	starter := &fakeRunStarter{err: errors.New("record run: db down")}
	fixture := newStageAPI(starter)

	response := fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"9"}`)

	assert.Equal(t, http.StatusBadGateway, response.Code)
	assert.Contains(t, response.Body.String(), "db down")
}

func TestRunStageRejectsBadRequests(t *testing.T) {
	starter := &fakeRunStarter{}
	fixture := newStageAPI(starter)

	assert.Equal(t, http.StatusBadRequest, fixture.run(t, "/api/v1/targetsystems/abc/stages/00/run", `{"pipelineId":"9"}`).Code)
	assert.Equal(t, http.StatusBadRequest, fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `not json`).Code)
	assert.Equal(t, http.StatusBadRequest, fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"route":"greenfield"}`).Code)
	assert.Equal(t, http.StatusBadRequest, fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"1","route":"change"}`).Code)
	assert.Equal(t, http.StatusBadRequest, fixture.run(t, "/api/v1/targetsystems/1/stages/00/run", `{"pipelineId":"PIPE-9"}`).Code,
		"a run the table cannot keep has no row: refuse it at once")
}

func TestRunStagePassesTheRequestThrough(t *testing.T) {
	starter := &fakeRunStarter{}
	fixture := newStageAPI(starter)

	fixture.run(t, "/api/v1/targetsystems/7/stages/00/run", `{"pipelineId":"9"}`)

	assert.Equal(t, usecase.StageRequest{
		PipelineID: "9", TargetSystemID: 7, Route: domain.RouteGreenfield, StageID: "00",
	}, starter.request)
}
