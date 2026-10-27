//go:build tdd

package acceptance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
)

// PR1 acceptance: a design stage that raised no question stores a valid form.
func TestPR1ADesignStageWithoutQuestionsIsValid(t *testing.T) {
	f := newFixture(t, respondWith(t, map[domain.FormType]json.RawMessage{
		domain.TypeHLD: validHLD(t),
	}))
	f.submit(t, "FRM-1-02", domain.TypeSRS, "02", validSRS(t))

	outcome := f.run(t, "03")
	require.False(t, outcome.Blocked)

	stored, err := f.forms.Latest(t.Context(), "FRM-1-03")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusValid, stored.Status,
		"no open question means the document is accepted, not a draft")
}

// PR1 acceptance: stage 04 rewrites the HLD it read instead of writing a second
// HLD under another id.
func TestPR1Stage04WritesANewVersionOfTheInputHLD(t *testing.T) {
	f := newFixture(t, respondWith(t, map[domain.FormType]json.RawMessage{
		domain.TypeHLD: validHLD(t),
		domain.TypeADR: validADR(t),
	}))
	f.submit(t, "FRM-1-02", domain.TypeSRS, "02", validSRS(t))

	f.run(t, "03")
	f.run(t, "04")

	hld, err := f.forms.Latest(t.Context(), "FRM-1-03")
	require.NoError(t, err)
	assert.Equal(t, 2, hld.FormVersion, "stage 04 writes the next version of the HLD it read")

	_, err = f.forms.Latest(t.Context(), "FRM-1-04-HLD")
	assert.Error(t, err, "the twin HLD document must not exist")

	_, err = f.forms.Latest(t.Context(), "FRM-1-04-ADR")
	assert.NoError(t, err, "the ADR is a new document of stage 04")
}

// PR1 acceptance: the HLD gate refuses a document without invariants and without
// a degradation strategy.
func TestPR1TheHLDGateRequiresInvariantsAndDegradation(t *testing.T) {
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-1-03", FormType: domain.TypeHLD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusValid, Route: domain.RouteGreenfield, Stage: "04", Mode: domain.ModeFull,
		PipelineID: testPipeline, TargetSystemID: "1", ProducedBy: domain.RoleArchitect,
	}}
	raw, err := json.Marshal(domain.HLD{
		BoundedContexts: []domain.BoundedContext{{ID: "bc-1", Name: "Lending", Purpose: ste("The desk lends books.")}},
		Containers: []domain.Container{{
			ObjectID: "obj-container-1", BoundedContextID: "bc-1",
			Purpose: ste("Serve the lending desk."), Tech: "go",
		}},
		Aggregates: []domain.Aggregate{{
			ID: "agg-1", ContainerID: "obj-container-1", Root: "Book", ModuleRef: "obj-module-1",
		}},
	})
	require.NoError(t, err)
	require.NoError(t, form.SetRawPayload(raw))

	err = form.ValidatePayload()
	require.Error(t, err, "a container without invariants and without degradation is not a design")
	assert.Contains(t, err.Error(), "invariants")
	assert.Contains(t, err.Error(), "degradationStrategy")
}

// PR1 acceptance: the cursor cannot pass an open question.
func TestPR1AnOpenQuestionHoldsTheCursor(t *testing.T) {
	f := newFixture(t, respondWith(t, map[domain.FormType]json.RawMessage{
		domain.TypeBRD: mustJSON(t, domain.BRD{
			Goals: []domain.Goal{{ID: "GOAL-1", Outcome: ste("The system shall speed up the desk."), MetricID: "M-1"}},
			Metrics: []domain.Metric{{
				ID: "M-1", Name: "lend_seconds", Unit: "second",
				Baseline: ptr(90.0), Target: ptr(20.0),
			}},
			Scope:          domain.Scope{In: []domain.Text{ste("Book registration")}},
			SuccessMetrics: []domain.Success{{MetricID: "M-1", Threshold: "20", Window: "30 day"}},
			OpenGaps: []domain.Gap{{
				ID: "GAP-1", Pointer: "/scope/in/0", Question: ste("Which branch registers first?"),
				Options: []string{"main", "second"}, Blocking: true,
			}},
		}),
	}))
	f.submit(t, "FRM-1-IN", domain.TypeUserInput, "—", mustJSON(t, domain.UserInput{
		RawRequest: ste("The operator needs a library accounting system."),
		Source:     domain.SourceOperator,
		ReceivedAt: "2026-10-19T21:00:00Z",
	}))

	_, err := f.orchestrator.Run(t.Context(), stageRequest("00"))
	require.NoError(t, err)
	require.NotEmpty(t, f.requests.saved, "the gap became a question")

	assert.Equal(t, "00", f.pipelines.pipeline.CurrentStage,
		"the pipeline waits on the operator, it does not move on")
}

func ptr(value float64) *float64 { return &value }
