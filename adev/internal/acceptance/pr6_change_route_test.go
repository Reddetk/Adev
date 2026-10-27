//go:build tdd

package acceptance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
)

// PR6 acceptance: every route is a stage list in the registry, including both
// branches of a change request.
func TestPR6EveryRouteHasItsStages(t *testing.T) {
	for _, route := range []domain.Route{domain.RouteChange, domain.RouteMinorChange, domain.RouteMajorChange} {
		stages, err := domain.Stages(route)
		require.NoError(t, err, "route %q must be implemented", route)
		assert.NotEmpty(t, stages)
	}

	change, err := domain.Stages(domain.RouteChange)
	require.NoError(t, err)
	ids := stageIDs(change)
	assert.Contains(t, ids, "00", "a change request starts with the CR brief")
	assert.Contains(t, ids, "01", "then the impact assessment")
}

// PR6 acceptance: the change impact assessment reads the accepted documents of
// the system, so the stage input source is part of the stage contract.
func TestPR6TheChangeImpactAssessmentReadsTheSystemBaseline(t *testing.T) {
	stage, err := domain.StageByID(domain.RouteChange, "01")
	require.NoError(t, err)

	raw, err := json.Marshal(stage)
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, json.Unmarshal(raw, &document))

	assert.Equal(t, "system-baseline", document["inputSource"],
		"stage 01 of a change request reads the as-is documents of another pipeline")
}

// PR6 acceptance: stages 07, 08, 09 and 08.R carry form types with gates.
func TestPR6Stages07To09HavePayloadGates(t *testing.T) {
	greenfield, err := domain.Stages(domain.RouteGreenfield)
	require.NoError(t, err)
	ids := stageIDs(greenfield)
	for _, stage := range []string{"07", "08", "09", "08.R"} {
		assert.Contains(t, ids, stage, "the full route ends at stage 09")
	}

	for _, formType := range []domain.FormType{
		domain.TypeTestContract, domain.TypeImplementationSpec, domain.TypeVerdict, domain.TypeRefactorSpec,
	} {
		assert.Contains(t, domain.FormTypesWithGates(), formType)
		shape, err := domain.PayloadShape(formType)
		require.NoError(t, err)
		assert.NotEmpty(t, shape)
	}

	next, err := domain.NextStage(domain.RouteGreenfield, "08.R")
	require.NoError(t, err)
	assert.Equal(t, "09", next.ID, "08.R is the revise loop back into the verdict")
}

func stageIDs(stages []domain.Stage) []string {
	ids := make([]string, 0, len(stages))
	for _, stage := range stages {
		ids = append(ids, stage.ID)
	}
	return ids
}
