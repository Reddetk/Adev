//go:build tdd

package acceptance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
)

// PR0 acceptance: a new form type is one table row plus one stage row. Nothing
// in stage_service, the gates, the shape or the projection knows its name.
func TestPR0TestContractIsOneTableRow(t *testing.T) {
	assert.Contains(t, domain.FormTypesWithGates(), domain.TypeTestContract,
		"TEST_CONTRACT must be a row of the form table")

	shape, err := domain.PayloadShape(domain.TypeTestContract)
	require.NoError(t, err, "a table row generates the shape")
	assert.NotEmpty(t, shape)

	_, err = domain.ProjectionFor(domain.TypeTestContract, domain.RoleArchitect)
	assert.NoError(t, err, "a table row gives the projection")
}

// PR0 acceptance: one question rule for every stage, no branch on the stage id.
func TestPR0OneQuestionRuleReachesEveryStage(t *testing.T) {
	f := newFixture(t, respondWith(t, map[domain.FormType]json.RawMessage{
		domain.TypeBRD: validBRD(t),
		domain.TypeHLD: validHLD(t),
	}))
	f.submit(t, "FRM-1-IN", domain.TypeUserInput, "—", mustJSON(t, domain.UserInput{
		RawRequest: ste("The operator needs a library accounting system."),
		Source:     domain.SourceOperator,
		ReceivedAt: "2026-10-19T21:00:00Z",
	}))
	f.submit(t, "FRM-1-02", domain.TypeSRS, "02", validSRS(t))

	f.run(t, "00")
	f.run(t, "03")

	require.Len(t, f.llm.requests, 2)
	for _, request := range f.llm.requests {
		assert.Contains(t, request.User, "put one openGaps entry",
			"every stage teaches the same question rule")
	}
	assert.Contains(t, f.llm.requests[0].User, "stage 00")
	assert.Contains(t, f.llm.requests[1].User, "stage 03")
}

// PR0 acceptance: the last stage of the route completes the pipeline.
func TestPR0TheLastStageCompletesThePipeline(t *testing.T) {
	f := newFixture(t, respondWith(t, map[domain.FormType]json.RawMessage{
		domain.TypeTaskPlan: validTaskPlan(t),
	}))
	f.submit(t, "FRM-1-02", domain.TypeSRS, "02", validSRS(t))
	f.submit(t, "FRM-1-05", domain.TypeLLD, "05", validLLD(t))
	f.pipelines.pipeline.CurrentStage = "06"

	_, err := f.orchestrator.Run(t.Context(), stageRequest("06"))
	require.NoError(t, err)

	assert.Equal(t, "completed", f.pipelines.pipeline.Status,
		"a successful run of the last stage closes the pipeline")
}
