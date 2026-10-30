package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLastStageMarksTheEndOfTheRoute(t *testing.T) {
	assert.True(t, IsLastStage(RouteGreenfield, "09"), "stage 09 ends the greenfield route")
	assert.False(t, IsLastStage(RouteGreenfield, "08"))
	assert.False(t, IsLastStage(RouteGreenfield, "99"), "an unknown stage ends nothing")
	assert.False(t, IsLastStage(RouteChange, "06"), "the change head ends nothing")
	assert.True(t, IsLastStage(RouteChange, "01"), "the impact assessment closes the head route")
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
		BoundedContexts: []BoundedContext{{ID: "bc-1", Name: "Delivery", Purpose: Text{Value: "Move parcels.", Lang: LangSTE100}}},
		Containers: []Container{{
			ObjectID: "c-1", BoundedContextID: "bc-1", Purpose: Text{Value: "Serve the client.", Lang: LangSTE100}, Tech: "go",
		}},
		Aggregates: []Aggregate{{
			ID: "a-1", ContainerID: "c-1", Root: "shipment", ModuleRef: "m-1",
			Invariants: []Statement{{Text: "A shipment has one route.", Modality: "shall", Priority: "must"}},
		}},
		Decisions: []Decision{{
			ID: "storage", Title: Text{Value: "Storage", Lang: LangSTE100},
			Decision:     Text{Value: "PostgreSQL.", Lang: LangSTE100},
			Alternatives: []Alternative{{Option: Text{Value: "MongoDB", Lang: LangSTE100}, RejectedBecause: Text{Value: "The route needs transactions.", Lang: LangSTE100}}},
			ObjectRefs:   []string{"c-1"},
		}},
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
		ID: "storage", Title: Text{Value: "Storage", Lang: LangSTE100},
		Decision:     Text{Value: "PostgreSQL.", Lang: LangSTE100},
		Alternatives: []Alternative{{Option: Text{Value: "MongoDB", Lang: LangSTE100}, RejectedBecause: Text{Value: "No transactions.", Lang: LangSTE100}}},
		ObjectRefs:   []string{"c-9"},
	}}
	assert.Error(t, ValidateHLD(foreign), "a decision that names no container of this design fixes nothing")
}
