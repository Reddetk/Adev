package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apphttp "github.com/Reddetk/Adev/internal/handler/http"
	"github.com/Reddetk/Adev/internal/repository"
	"github.com/Reddetk/Adev/internal/usecase"
)

const (
	userInputFormID = "FRM-IN-1"
	pipelineID      = "PIPE-1"
)

type formAPI struct {
	router *mux.Router
}

func newFormAPI() *formAPI {
	router := mux.NewRouter()
	version := router.PathPrefix("/api/v1").Subrouter()
	apphttp.SetupFormRoutes(version, apphttp.NewFormHandler(usecase.NewFormService(repository.NewMemoryFormsStore(), nil)))
	return &formAPI{router: router}
}

func (a *formAPI) send(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(raw)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	a.router.ServeHTTP(recorder, request)
	return recorder
}

func userInputBody() map[string]any {
	return map[string]any{
		"formId": userInputFormID, "formType": "USER_INPUT", "schemaVersion": 1, "formVersion": 1,
		"status": "draft", "route": "greenfield", "stage": "—", "mode": "full",
		"pipelineId": pipelineID, "targetSystemId": "TGT-1", "producedBy": "operator",
		"inputs": []any{},
		"payload": map[string]any{
			"rawRequest": map[string]any{"text": "The operator needs a library accounting system.", "lang": "STE100"},
			"source":     "operator",
			"receivedAt": "2026-10-19T21:00:00Z",
		},
	}
}

func brdBody(inputHash string) map[string]any {
	return map[string]any{
		"formId": "FRM-BRD-1", "formType": "BRD", "schemaVersion": 1, "formVersion": 1,
		"status": "draft", "route": "greenfield", "stage": "00", "mode": "full",
		"pipelineId": pipelineID, "targetSystemId": "TGT-1", "producedBy": "analyst",
		"inputs": []any{map[string]any{
			"formId": userInputFormID, "formType": "USER_INPUT", "formVersion": 1, "hash": inputHash,
		}},
		"payload": map[string]any{
			"goals": []any{map[string]any{
				"id": "GOAL-1", "outcome": map[string]any{"text": "The system shall speed up the lending desk.", "lang": "STE100"},
				"metricId": "M-1",
			}},
			"metrics": []any{map[string]any{
				"id": "M-1", "name": "lend_seconds", "unit": "second", "baseline": 90, "target": 20,
			}},
			"scope":          map[string]any{"in": []any{}, "out": []any{}},
			"actors":         []any{},
			"successMetrics": []any{map[string]any{"metricId": "M-1", "threshold": "20", "window": "30 day"}},
		},
	}
}

func (a *formAPI) submitUserInput(t *testing.T) string {
	t.Helper()
	response := a.send(t, http.MethodPost, "/api/v1/forms", userInputBody())
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var stored struct {
		PayloadHash string `json:"payloadHash"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &stored))
	require.NotEmpty(t, stored.PayloadHash)
	return stored.PayloadHash
}

// --- submit -----------------------------------------------------------------

func TestSubmitFormHappyPath(t *testing.T) {
	response := newFormAPI().send(t, http.MethodPost, "/api/v1/forms", userInputBody())

	assert.Equal(t, http.StatusCreated, response.Code)
	var stored struct {
		FormID      string `json:"formId"`
		FormVersion int    `json:"formVersion"`
		Status      string `json:"status"`
		PayloadHash string `json:"payloadHash"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &stored))
	assert.Equal(t, userInputFormID, stored.FormID)
	assert.Equal(t, 1, stored.FormVersion)
	assert.Equal(t, "draft", stored.Status)
	assert.Contains(t, stored.PayloadHash, "sha256:", "the handler computes the hash, the client does not send it")
}

func TestSubmitFormRejectsBrokenJSON(t *testing.T) {
	response := newFormAPI().send(t, http.MethodPost, "/api/v1/forms", nil)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestSubmitFormRejectsAnUnknownFormType(t *testing.T) {
	body := userInputBody()
	body["formType"] = "MAGIC"

	response := newFormAPI().send(t, http.MethodPost, "/api/v1/forms", body)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "formType")
}

func TestSubmitFormRejectsAFailedPayloadGate(t *testing.T) {
	fixture := newFormAPI()
	hash := fixture.submitUserInput(t)

	body := brdBody(hash)
	body["payload"] = map[string]any{"goals": []any{}, "metrics": []any{}}

	response := fixture.send(t, http.MethodPost, "/api/v1/forms", body)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "goals")
}

func TestSubmitFormRejectsAnUnresolvedInput(t *testing.T) {
	response := newFormAPI().send(t, http.MethodPost, "/api/v1/forms", brdBody("sha256:x"))

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "not resolved")
}

func TestSubmitFormRejectsAMismatchedInputHash(t *testing.T) {
	fixture := newFormAPI()
	fixture.submitUserInput(t)

	response := fixture.send(t, http.MethodPost, "/api/v1/forms", brdBody("sha256:wrong"))

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "does not match")
}

func TestSubmitFormAcceptsAFormWithResolvedInputs(t *testing.T) {
	fixture := newFormAPI()
	hash := fixture.submitUserInput(t)

	response := fixture.send(t, http.MethodPost, "/api/v1/forms", brdBody(hash))

	assert.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	assert.Contains(t, response.Body.String(), `"formType":"BRD"`)
}

func TestSubmitFormRefusesADuplicateVersion(t *testing.T) {
	fixture := newFormAPI()
	fixture.submitUserInput(t)

	response := fixture.send(t, http.MethodPost, "/api/v1/forms", userInputBody())

	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Contains(t, response.Body.String(), "already exists")
}

func TestSubmitFormRefusesATypeWithoutGates(t *testing.T) {
	body := userInputBody()
	body["formType"] = "QUESTION"
	body["stage"] = "\u2014"

	response := newFormAPI().send(t, http.MethodPost, "/api/v1/forms", body)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "payload gates")
}

func TestSubmitFormRejectsAnInvalidUserInputPayload(t *testing.T) {
	body := userInputBody()
	body["payload"] = map[string]any{
		"rawRequest": map[string]any{"text": "The operator needs a system.", "lang": "STE100"},
		"source":     "telepathy",
		"receivedAt": "yesterday",
	}

	response := newFormAPI().send(t, http.MethodPost, "/api/v1/forms", body)

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Contains(t, response.Body.String(), "source")
}

// --- read -------------------------------------------------------------------

func TestGetLatestFormVersion(t *testing.T) {
	fixture := newFormAPI()
	fixture.submitUserInput(t)

	response := fixture.send(t, http.MethodGet, "/api/v1/forms/"+userInputFormID+"/latest", nil)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), userInputFormID)
}

func TestGetLatestFormVersionNotFound(t *testing.T) {
	response := newFormAPI().send(t, http.MethodGet, "/api/v1/forms/FRM-none/latest", nil)

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestGetFormVersion(t *testing.T) {
	fixture := newFormAPI()
	fixture.submitUserInput(t)

	response := fixture.send(t, http.MethodGet, "/api/v1/forms/"+userInputFormID+"/versions/1", nil)

	assert.Equal(t, http.StatusOK, response.Code)
}

func TestGetFormVersionRejectsANonNumericVersion(t *testing.T) {
	response := newFormAPI().send(t, http.MethodGet, "/api/v1/forms/"+userInputFormID+"/versions/abc", nil)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestGetFormVersionNotFound(t *testing.T) {
	fixture := newFormAPI()
	fixture.submitUserInput(t)

	response := fixture.send(t, http.MethodGet, "/api/v1/forms/"+userInputFormID+"/versions/9", nil)

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestListFormsByPipeline(t *testing.T) {
	fixture := newFormAPI()
	fixture.submitUserInput(t)

	response := fixture.send(t, http.MethodGet, "/api/v1/pipelines/"+pipelineID+"/forms?type=USER_INPUT", nil)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), userInputFormID)
}

func TestListFormsByPipelineWithoutATypeListsEveryType(t *testing.T) {
	response := newFormAPI().send(t, http.MethodGet, "/api/v1/pipelines/"+pipelineID+"/forms", nil)

	assert.Equal(t, http.StatusOK, response.Code)
}

func TestListFormsByPipelineRejectsAnUnknownType(t *testing.T) {
	response := newFormAPI().send(t, http.MethodGet, "/api/v1/pipelines/"+pipelineID+"/forms?type=MAGIC", nil)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestListFormsByPipelineIsEmptyForAnUnknownPipeline(t *testing.T) {
	response := newFormAPI().send(t, http.MethodGet, "/api/v1/pipelines/PIPE-none/forms?type=BRD", nil)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "[]", strings.TrimSpace(response.Body.String()))
}

// --- status -----------------------------------------------------------------

func TestSetFormStatusFollowsTheLifecycle(t *testing.T) {
	fixture := newFormAPI()
	fixture.submitUserInput(t)

	response := fixture.send(t, http.MethodPut, "/api/v1/forms/"+userInputFormID+"/versions/1/status",
		map[string]string{"status": "valid"})
	assert.Equal(t, http.StatusNoContent, response.Code)

	latest := fixture.send(t, http.MethodGet, "/api/v1/forms/"+userInputFormID+"/latest", nil)
	assert.Contains(t, latest.Body.String(), `"status":"valid"`)
}

func TestSetFormStatusRefusesAnIllegalTransition(t *testing.T) {
	fixture := newFormAPI()
	fixture.submitUserInput(t)

	illegal := fixture.send(t, http.MethodPut, "/api/v1/forms/"+userInputFormID+"/versions/1/status",
		map[string]string{"status": "draft"})
	assert.Equal(t, http.StatusConflict, illegal.Code)

	fixture.send(t, http.MethodPut, "/api/v1/forms/"+userInputFormID+"/versions/1/status", map[string]string{"status": "valid"})
	back := fixture.send(t, http.MethodPut, "/api/v1/forms/"+userInputFormID+"/versions/1/status",
		map[string]string{"status": "draft"})
	assert.Equal(t, http.StatusConflict, back.Code, "a valid form cannot go back to draft")
}

func TestSetFormStatusRejectsAnUnknownStatus(t *testing.T) {
	fixture := newFormAPI()
	fixture.submitUserInput(t)

	response := fixture.send(t, http.MethodPut, "/api/v1/forms/"+userInputFormID+"/versions/1/status",
		map[string]string{"status": "maybe"})

	assert.Equal(t, http.StatusBadRequest, response.Code)
}

func TestSetFormStatusOnAMissingForm(t *testing.T) {
	response := newFormAPI().send(t, http.MethodPut, "/api/v1/forms/FRM-none/versions/1/status",
		map[string]string{"status": "valid"})

	assert.Equal(t, http.StatusNotFound, response.Code)
}
