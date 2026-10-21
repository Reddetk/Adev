//go:build tdd

package acceptance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
)

// PR4 acceptance: FR and NFR are two documents with their own types and statuses.
func TestPR4FRAndNFRAreSeparateFormTypes(t *testing.T) {
	for _, formType := range []domain.FormType{"FR", "NFR"} {
		assert.Contains(t, domain.FormTypesWithGates(), formType,
			"%s must be its own form type", formType)

		shape, err := domain.PayloadShape(formType)
		require.NoError(t, err)
		assert.NotEmpty(t, shape)

		_, err = domain.ProjectionFor(formType, domain.RoleQA)
		assert.NoError(t, err)
	}
}

// PR4 acceptance: every item of the clean BRD is closed by at least one FR or NFR.
func TestPR4AnUncoveredBRDItemFailsTheCoverageGate(t *testing.T) {
	f := newFixture(t, nil)
	srs := domain.SRS{
		FR: []domain.FunctionalRequirement{{
			ID:         "FR-1",
			Statement:  domain.Statement{ID: "FR-1", Text: "The system shall register a book.", Modality: domain.ModalityShall, Priority: domain.PriorityMust},
			Actor:      "librarian",
			Trigger:    "a book arrives",
			Response:   "the system stores the book",
			Priority:   domain.PriorityMust,
			TraceGoal:  "GOAL-1",
			Acceptance: []domain.Scenario{{Given: "a librarian", When: "a book arrives", Then: "the book is stored"}},
		}},
		TraceToGoals: []string{"GOAL-1", "GOAL-2"},
	}

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-1-02", FormType: domain.TypeSRS, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusValid, Route: domain.RouteGreenfield, Stage: "02", Mode: domain.ModeFull,
		PipelineID: testPipeline, TargetSystemID: "1", ProducedBy: domain.RoleAnalyst,
		Inputs: []domain.Ref{},
	}}
	require.NoError(t, form.SetRawPayload(mustJSON(t, srs)))

	err := f.forms.Submit(t.Context(), form)

	require.Error(t, err, "GOAL-2 is in the trace but no requirement covers it")
	assert.Contains(t, err.Error(), "GOAL-2")
}

// PR4 acceptance: the skeleton goes out, then every requirement is detailed by its
// own call, so one huge answer is not the only path.
func TestPR4TheSkeletonIsDetailedByOneCallPerRequirement(t *testing.T) {
	skeleton := mustJSON(t, domain.SRS{
		FR: []domain.FunctionalRequirement{
			skeletonFR("FR-1"), skeletonFR("FR-2"), skeletonFR("FR-3"),
		},
		TraceToGoals: []string{"GOAL-1"},
	})
	f := newFixture(t, respondWith(t, map[domain.FormType]json.RawMessage{
		domain.TypeSRS: skeleton,
	}))
	f.submit(t, "FRM-1-00", domain.TypeBRD, "00", validBRD(t))
	f.submit(t, "FRM-1-01-FEASIBILITY", domain.TypeFeasibility, "01", validFeasibility(t))

	f.run(t, "02")

	assert.GreaterOrEqual(t, len(f.llm.requests), 4,
		"one skeleton call plus one detail call per FR: %d call(s) means one giant answer", len(f.llm.requests))
}

// PR4 acceptance: the TaskPlan of stage 06 is a tree, not a flat list.
func TestPR4TheTaskPlanMustBeATree(t *testing.T) {
	flat := domain.TaskPlan{Tasks: []domain.PlanTask{
		planTask("T-1", ""), planTask("T-2", ""), planTask("T-3", ""),
	}}

	err := domain.ValidateTaskPlan(flat)

	require.Error(t, err, "a plan without a single parent is a list, not a decomposition")
}

func skeletonFR(id string) domain.FunctionalRequirement {
	return domain.FunctionalRequirement{
		ID:        id,
		Statement: domain.Statement{ID: id, Text: "The system shall register a book.", Modality: domain.ModalityShall, Priority: domain.PriorityMust},
		Priority:  domain.PriorityMust,
		Acceptance: []domain.Scenario{{
			Given: "a librarian", When: "a book arrives", Then: "the book is stored",
		}},
	}
}

func planTask(id, parent string) domain.PlanTask {
	return domain.PlanTask{
		ID: id, Title: ste("Register a book."), TaskType: domain.TaskFeature, Owner: domain.OwnerDev,
		ParentID:     parent,
		Acceptance:   []domain.Statement{{ID: "AC-" + id, Text: "A book is stored.", Modality: domain.ModalityShall, Priority: domain.PriorityMust}},
		ContractRefs: []string{"contract-1"},
	}
}
