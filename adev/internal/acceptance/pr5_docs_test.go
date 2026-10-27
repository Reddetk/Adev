//go:build tdd

package acceptance

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
	apphttp "github.com/Reddetk/Adev/internal/handler/http"
)

// PR5 acceptance: the Docs tab reads every document of a pipeline in one request.
func TestPR5TheDocsEndpointListsEveryTypeWithoutTheTypeQuery(t *testing.T) {
	f := newFixture(t, nil)
	f.saveValid(t, "FRM-1-02", domain.TypeSRS, "02")
	f.saveValid(t, "FRM-1-03", domain.TypeHLD, "03")

	response := docsRequest(t, f, "/api/v1/pipelines/1/forms")

	assert.Equal(t, http.StatusOK, response.Code,
		"the type query must be optional for the document list")
	var forms []map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &forms))
	assert.Len(t, forms, 2)
}

// PR5 acceptance: the list filters by pipeline, stage, type and status.
func TestPR5TheDocsEndpointFiltersByTypeAndStatus(t *testing.T) {
	f := newFixture(t, nil)
	f.saveValid(t, "FRM-1-02", domain.TypeSRS, "02")
	f.saveValid(t, "FRM-1-03", domain.TypeHLD, "03")
	f.saveDraftForm(t, "FRM-1-03", 2, domain.TypeHLD, "03")

	byType := docsRequest(t, f, "/api/v1/pipelines/1/forms?type=HLD")
	assert.Equal(t, http.StatusOK, byType.Code)
	var hlds []map[string]any
	require.NoError(t, json.Unmarshal(byType.Body.Bytes(), &hlds))
	assert.Len(t, hlds, 2, "both versions of the HLD are listed")

	byStatus := docsRequest(t, f, "/api/v1/pipelines/1/forms?status=valid")
	assert.Equal(t, http.StatusOK, byStatus.Code)
	var valid []map[string]any
	require.NoError(t, json.Unmarshal(byStatus.Body.Bytes(), &valid))
	require.Len(t, valid, 2)
	for _, form := range valid {
		assert.Equal(t, "valid", form["status"])
	}
}

func docsRequest(t *testing.T, f *fixture, path string) *httptest.ResponseRecorder {
	t.Helper()
	router := mux.NewRouter()
	apphttp.SetupFormRoutes(router.PathPrefix("/api/v1").Subrouter(), apphttp.NewFormHandler(f.forms))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

// saveValid stores a version-1 form with the given status.
func (f *fixture) saveValid(t *testing.T, id string, formType domain.FormType, stage string) {
	t.Helper()
	f.saveDraftForm(t, id, 1, formType, stage)
	require.NoError(t, f.forms.SetStatus(t.Context(), id, 1, domain.StatusValid))
}

// saveDraftForm stores one version with a payload that is never read by the list.
func (f *fixture) saveDraftForm(t *testing.T, id string, version int, formType domain.FormType, stage string) {
	t.Helper()
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: id, FormType: formType, SchemaVersion: 1, FormVersion: version,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: stage, Mode: domain.ModeFull,
		PipelineID: testPipeline, TargetSystemID: "1", ProducedBy: domain.RoleAnalyst,
		Inputs: []domain.Ref{},
	}}
	if version > 1 {
		previous, err := f.forms.Latest(t.Context(), id)
		require.NoError(t, err)
		form.PreviousFormVersion = &domain.Ref{
			FormID: previous.FormID, FormType: previous.FormType,
			FormVersion: previous.FormVersion, Hash: previous.PayloadHash,
		}
	}
	require.NoError(t, form.SetRawPayload(json.RawMessage(`{}`)))
	f.saveDraft(t, form)
}
