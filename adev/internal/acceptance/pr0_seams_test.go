//go:build tdd

package acceptance

import (
	"encoding/json"
	"strings"
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

	// The HLD is filled level by level: the skeleton call and one call per
	// container. Every call teaches the same question rule.
	require.Len(t, f.llm.requests, 3)
	for _, request := range f.llm.requests {
		assert.True(t, strings.Contains(strings.ToLower(request.User), "put one opengaps entry") || strings.Contains(request.User, "put one openGaps entry"),
			"every stage teaches the same question rule")
	}
	assert.Contains(t, f.llm.requests[0].User, "stage 00")
	assert.Contains(t, f.llm.requests[1].User, "stage 03")
	assert.Contains(t, f.llm.requests[2].User, "the containers entry with objectId")
}

// PR0 acceptance: the last stage of the route completes the pipeline.
func TestPR0TheLastStageCompletesThePipeline(t *testing.T) {
	f := newFixture(t, respondWith(t, map[domain.FormType]json.RawMessage{
		domain.TypeVerdict: validVerdict(t),
	}))
	f.submit(t, "FRM-1-07", domain.TypeTestContract, "07", validTestContract(t))
	f.submit(t, "FRM-1-08", domain.TypeImplementationSpec, "08", validImplementationSpec(t))

	stages, err := domain.Stages(domain.RouteGreenfield)
	require.NoError(t, err)
	last := stages[len(stages)-1].ID
	f.pipelines.pipeline.CurrentStage = last

	_, err = f.orchestrator.Run(t.Context(), stageRequest(last))
	require.NoError(t, err)

	assert.Equal(t, "completed", f.pipelines.pipeline.Status,
		"a successful run of the last stage closes the pipeline")
}
