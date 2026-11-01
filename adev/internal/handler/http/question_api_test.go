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
	"github.com/Reddetk/Adev/internal/usecase"
)

// fakeAnswers stands for the QuestionService: the answer already landed on the
// form, the handler only decides whether the stage continues.
type fakeAnswers struct {
	form       *domain.Form
	batchCalls int
	lastBatch  []usecase.AnswerInput
}

func (f *fakeAnswers) AnswerOnPipeline(context.Context, int, string, json.RawMessage) (*domain.Form, error) {
	return f.form, nil
}

func (f *fakeAnswers) AnswerChoice(context.Context, int, string, string, json.RawMessage) (*domain.Form, error) {
	return f.form, nil
}

func (f *fakeAnswers) AnswerManyOnPipeline(_ context.Context, _ int, answers []usecase.AnswerInput) (*domain.Form, error) {
	f.batchCalls++
	f.lastBatch = answers
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

// TestAnswersApplyTheBatchOnce: the console sends five answers as one request, the
// service applies them as one action, and the stage restarts once.
func TestAnswersApplyTheBatchOnce(t *testing.T) {
	answers := &fakeAnswers{form: &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-9-01-FEASIBILITY", FormType: domain.TypeFeasibility, FormVersion: 1,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "01",
		PipelineID: "9", TargetSystemID: "1",
	}}}
	starter := &fakeRunStarter{}
	router := newQuestionAPI(answers, starter)

	body := strings.NewReader(`{"pipelineId":9,"answers":[` +
		`{"pointer":"/premises/traffic/peakQps","value":120},` +
		`{"pointer":"/premises/team/size","value":4}]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/operatorrequests/answers", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, 1, answers.batchCalls, "the handler must apply the batch in one call")
	require.Len(t, answers.lastBatch, 2)
	assert.Equal(t, "/premises/traffic/peakQps", answers.lastBatch[0].Pointer)
	assert.Equal(t, "01", starter.request.StageID, "the stage restarts once after the batch")
}

// TestAnswersRefuseAnEmptyBatch: a request without answers is a client mistake, not
// a stage restart.
func TestAnswersRefuseAnEmptyBatch(t *testing.T) {
	answers := &fakeAnswers{form: &domain.Form{}}
	starter := &fakeRunStarter{}
	router := newQuestionAPI(answers, starter)

	body := strings.NewReader(`{"pipelineId":9,"answers":[]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/operatorrequests/answers", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Equal(t, 0, answers.batchCalls)
	assert.Empty(t, starter.request.StageID)
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

// TestAnswerCompletedTheFormContinuesTheStage: an answer that makes the form valid
// still starts the run, because only the run moves the pipeline cursor to the next
// stage. Without it the pipeline stays on the stage the operator just finished.
func TestAnswerCompletedTheFormContinuesTheStage(t *testing.T) {
	answers := &fakeAnswers{form: &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-9-03", FormType: domain.TypeHLD, FormVersion: 8,
		Status: domain.StatusValid, Route: domain.RouteGreenfield, Stage: "03",
		PipelineID: "9", TargetSystemID: "1",
	}}}
	starter := &fakeRunStarter{}

	recorder := answerRequest(t, newQuestionAPI(answers, starter))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "03", starter.request.StageID)
	assert.Equal(t, "9", starter.request.PipelineID)
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
