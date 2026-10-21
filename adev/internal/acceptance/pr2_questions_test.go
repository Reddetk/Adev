//go:build tdd

package acceptance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
)

// PR2 acceptance: the options of a gap are data, not text glued onto the question.
func TestPR2AGapPublishesItsOptionsAsData(t *testing.T) {
	form := gapDraft(t)

	f := newFixture(t, nil)
	f.saveDraft(t, form)

	published, blocked, err := f.questions.PublishGaps(t.Context(), form, 1)
	require.NoError(t, err)
	require.True(t, published)
	require.True(t, blocked)
	require.Len(t, f.requests.saved, 1)

	encoded, err := json.Marshal(f.requests.saved[0])
	require.NoError(t, err)
	var request map[string]any
	require.NoError(t, json.Unmarshal(encoded, &request))
	assert.Equal(t, []any{"Option A", "Option B"}, request["options"],
		"the dashboard renders options from a field, it does not parse the question text")
}

// PR2 acceptance: an operator answer never fails. Any input becomes either the
// value of the field or a new question.
func TestPR2AnyOperatorInputIsAccepted(t *testing.T) {
	tests := []struct {
		name  string
		value json.RawMessage
	}{
		{name: "plain words", value: json.RawMessage(`"abc"`)},
		{name: "bare number", value: json.RawMessage(`123`)},
		{name: "broken json", value: json.RawMessage(`not json`)},
		{name: "object for a text field", value: json.RawMessage(`{"other":true}`)},
		{name: "null", value: json.RawMessage(`null`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t, nil)
			form := gapDraft(t)
			f.saveDraft(t, form)
			_, _, err := f.questions.PublishGaps(t.Context(), form, 1)
			require.NoError(t, err)

			answered, err := f.questions.AnswerOnPipeline(t.Context(), 1, "/metrics/0/unit", test.value)

			assert.NoError(t, err, "an answer is a decision, never a 4xx or a 5xx")
			assert.NotNil(t, answered)
		})
	}
}

// PR2 acceptance: a gate question is closable by the operator, not a dead end.
func TestPR2AGateQuestionIsClosable(t *testing.T) {
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-1-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: testPipeline, TargetSystemID: "1", ProducedBy: domain.RoleAnalyst,
		Inputs: []domain.Ref{},
	}}
	require.NoError(t, form.SetRawPayload(mustJSON(t, domain.BRD{
		Goals:          []domain.Goal{{ID: "GOAL-1", Outcome: ste("The system shall speed up the desk."), MetricID: "M-1"}},
		Metrics:        []domain.Metric{{ID: "M-1", Name: "lend_seconds", Unit: "second", Baseline: ptr(90), Target: ptr(20)}},
		Scope:          domain.Scope{In: []domain.Text{ste("Book registration")}},
		SuccessMetrics: []domain.Success{{MetricID: "M-1", Threshold: "20", Window: "30 day"}},
	})))

	f := newFixture(t, nil)
	f.saveDraft(t, form)
	require.NoError(t, f.questions.Publish(t.Context(), form,
		[]domain.ValidationError{{Field: "successMetrics[0].threshold", Problem: "is empty"}}, 1))

	answered, err := f.questions.AnswerOnPipeline(t.Context(), 1, "/successMetrics/0/threshold", json.RawMessage(`"20"`))

	assert.NoError(t, err, "a failed gate is a question the operator can answer")
	assert.NotNil(t, answered)
}

func gapDraft(t *testing.T) *domain.Form {
	t.Helper()
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-1-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: testPipeline, TargetSystemID: "1", ProducedBy: domain.RoleAnalyst,
		Inputs: []domain.Ref{},
	}}
	require.NoError(t, form.SetRawPayload(mustJSON(t, domain.BRD{
		Goals:          []domain.Goal{{ID: "GOAL-1", Outcome: ste("The system shall speed up the desk."), MetricID: "M-1"}},
		Metrics:        []domain.Metric{{ID: "M-1", Name: "lend_seconds", Baseline: ptr(90), Target: ptr(20)}},
		Scope:          domain.Scope{In: []domain.Text{ste("Book registration")}},
		SuccessMetrics: []domain.Success{{MetricID: "M-1", Threshold: "20", Window: "30 day"}},
		OpenGaps: []domain.Gap{{
			ID: "GAP-1", Pointer: "/metrics/0/unit",
			Question: ste("Which unit measures the lend time?"),
			Options:  []string{"Option A", "Option B"}, Blocking: true,
		}},
	})))
	return form
}
