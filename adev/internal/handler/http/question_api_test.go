package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
	apphttp "github.com/Reddetk/Adev/internal/handler/http"
)

// fakeAnswers stands for the QuestionService: the answer already landed on the
// form, the handler only decides whether the stage continues.
type fakeAnswers struct {
	form *domain.Form
}

func (f *fakeAnswers) AnswerOnPipeline(context.Context, int, string, json.RawMessage) (*domain.Form, error) {
	return f.form, nil
}

func (f *fakeAnswers) AnswerChoice(context.Context, int, string, string, json.RawMessage) (*domain.Form, error) {
	return f.form, nil
}

func newQuestionAPI(answers apphttp.AnswerService, starter apphttp.StageRunStarter) *mux.Router {
	router := mux.NewRouter()
	version := router.PathPrefix("/api/v1").Subrouter()
	apphttp.SetupQuestionRoutes(version, apphttp.NewQuestionHandler(answers, starter))
	return router
}

func answerRequest(t *testing.T, router *mux.Router) *httptest.ResponseRecorder {
	t.Helper()
	body := strings.NewReader(`{"pipelineId":9,"pointer":"/containers/0","value":"C-1"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/operatorrequests/answer", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// TestAnswerContinuesTheStage: the last answer leaves a draft, so the handler
// starts the stage again instead of waiting for the operator to press the button.
func TestAnswerContinuesTheStage(t *testing.T) {
	answers := &fakeAnswers{form: &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-9-05-LLD", FormType: domain.TypeLLD, FormVersion: 4,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "05",
		PipelineID: "9", TargetSystemID: "1",
	}}}
	starter := &fakeRunStarter{}

	recorder := answerRequest(t, newQuestionAPI(answers, starter))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "05", starter.request.StageID)
	assert.Equal(t, "9", starter.request.PipelineID)
	assert.Equal(t, int64(1), starter.request.TargetSystemID)
	assert.Equal(t, domain.RouteGreenfield, starter.request.Route)
}

// TestAnswerWaitingKeepsTheStageStopped: while a question waits, the stage must
// not start another run.
func TestAnswerWaitingKeepsTheStageStopped(t *testing.T) {
	answers := &fakeAnswers{form: &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-9-05-LLD", FormType: domain.TypeLLD, FormVersion: 4,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "05",
		PipelineID: "9", TargetSystemID: "1",
	}}}
	starter := &fakeRunStarter{}

	recorder := answerRequest(t, newQuestionAPI(answers, starter))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, starter.request.StageID)
}
