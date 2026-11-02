package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLastStageMarksTheEndOfTheRoute(t *testing.T) {
	assert.True(t, IsLastStage(RouteGreenfield, "05"), "stage 05 ends the route until 06-09 return")
	assert.False(t, IsLastStage(RouteGreenfield, "03"))
	assert.False(t, IsLastStage(RouteGreenfield, "99"), "an unknown stage ends nothing")
	assert.False(t, IsLastStage(RouteChange, "06"), "the change head ends nothing")
	assert.True(t, IsLastStage(RouteChange, "01"), "the impact assessment closes the head route")
}

// TestStagesAfterLLDAreNotAvailable: the task plan and the stages after it are
// kept in the registry for a later release, but the operator cannot run them and
// the plan does not offer them.
func TestStagesAfterLLDAreNotAvailable(t *testing.T) {
	stages, err := AvailableStages(RouteGreenfield)
	assert.NoError(t, err)
	assert.Equal(t, "05", stages[len(stages)-1].ID, "LLD is the last available stage")

	for _, id := range []string{"06", "07", "08", "08.R", "09"} {
		_, err := StageByID(RouteGreenfield, id)
		assert.Error(t, err, "stage %s is disabled", id)
	}
	if _, err := NextStage(RouteGreenfield, "05"); err == nil {
		t.Fatal("a disabled next stage must not be returned")
	}
}

// TestGreenfieldHasNoADRStage: a new system has no decision history to record. Its
// choices live in the HLD, and the LLD reads that design alone. A change request
// keeps the ADR, because there the record is why a running system moves.
func TestGreenfieldHasNoADRStage(t *testing.T) {
	stages, err := Stages(RouteGreenfield)
	assert.NoError(t, err)
	ids := map[string]bool{}
	for _, stage := range stages {
		ids[stage.ID] = true
	}
	assert.False(t, ids["04"], "the greenfield route has no ADR stage")
	assert.True(t, ids["03"])
	assert.True(t, ids["05"])

	lld, err := StageByID(RouteGreenfield, "05")
	assert.NoError(t, err)
	assert.Equal(t, []FormType{TypeHLD}, lld.Inputs, "the LLD reads the HLD alone")

	major, err := Stages(RouteMajorChange)
	assert.NoError(t, err)
	changeIDs := map[string]bool{}
	for _, stage := range major {
		changeIDs[stage.ID] = true
	}
	assert.True(t, changeIDs["04"], "a major change keeps the ADR stage")
}

// TestHLDKeepsTheDecisions: the ten architecture choices of a new system live in
// the design, not in a second document that restates it.
func TestHLDKeepsTheDecisions(t *testing.T) {
	hld := HLD{
		BoundedContexts: []BoundedContext{{ID: "bc-1", Name: "Delivery", Purpose: Text{Value: "Move parcels."}, PACELC: PACELCPaEl}},
		Containers: []Container{{
			ObjectID: "c-1", BoundedContextID: "bc-1", Purpose: Text{Value: "Serve the client."}, Tech: "go",
		}},
		Aggregates: []Aggregate{{
			ID: "a-1", ContainerID: "c-1", Root: "shipment",
			Invariants: []Statement{{Text: "A shipment has one route.", Modality: "shall", Priority: "must"}},
		}},
		Decisions: []Decision{
			{
				ID: "d-consistency", Area: AreaConsistency, ContextRef: "bc-1",
				Title:    Text{Value: "Consistency"},
				Decision: Text{Value: "The context answers after the write."},
				Alternatives: []Alternative{{
					Option:          Text{Value: "Eventual"},
					RejectedBecause: Text{Value: "The client confirms the receipt."},
				}},
			},
			{
				ID: "d-architecture", Area: AreaArchitecture, Style: StyleModularMonolith,
				Title:    Text{Value: "Architecture"},
				Decision: Text{Value: "One deployable unit."},
				Alternatives: []Alternative{{
					Option:          Text{Value: "Microservices"},
					RejectedBecause: Text{Value: "The team is small."},
				}},
			},
			{
				ID: "storage", Area: AreaStorage, Ownership: OwnershipPerService, Title: Text{Value: "Storage"},
				Decision:     Text{Value: "PostgreSQL."},
				Alternatives: []Alternative{{Option: Text{Value: "MongoDB"}, RejectedBecause: Text{Value: "The route needs transactions."}}},
				ObjectRefs:   []string{"c-1"},
			},
			{
				ID: "deployment", Area: AreaOps, Deployment: DeploymentServicePerContainer, Title: Text{Value: "Deployment"},
				Decision:     Text{Value: "One container."},
				Alternatives: []Alternative{{Option: Text{Value: "One host"}, RejectedBecause: Text{Value: "The image is the unit."}}},
			},
		},
		DegradationStrategy: []Degradation{{
			Failure: "The database is down", Degradation: "The client sees the last known status",
			Detection: "A health check fails", TimeoutMs: 1000,
		}},
	}
	assert.NoError(t, ValidateHLD(hld), "a design with one decision and its container passes")

	missing := hld
	missing.Decisions = nil
	assert.Error(t, ValidateHLD(missing), "a design without one decision is not a design")

	foreign := hld
	foreign.Decisions = []Decision{{
		ID: "storage", Title: Text{Value: "Storage"},
		Decision:     Text{Value: "PostgreSQL."},
		Alternatives: []Alternative{{Option: Text{Value: "MongoDB"}, RejectedBecause: Text{Value: "No transactions."}}},
		ObjectRefs:   []string{"c-9"},
	}}
	assert.Error(t, ValidateHLD(foreign), "a decision that names no container of this design fixes nothing")

	// A product the design does not carry is the drift the decision gate exists
	// for: the ADR of a live run named a cache and a search engine that no
	// container had.
	ghostProduct := hld
	ghostProduct.Decisions = []Decision{{
		ID: "storage", Title: Text{Value: "Storage"},
		Decision:     Text{Value: "Redis."},
		Alternatives: []Alternative{{Option: Text{Value: "Memcached"}, RejectedBecause: Text{Value: "No persistence."}}},
		ObjectRefs:   []string{"c-1"},
		Techs:        []string{"redis"},
	}}
	assert.Error(t, ValidateHLD(ghostProduct), "a product with no container is a component the design does not have")
}
