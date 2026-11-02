package domain

import (
	"math/rand"
	"strings"
	"testing"
)

// C4 level 1 is the system context: the people, the system and the external
// systems. The HLD now names all three, and a relation is a piece of the SRS
// flow, not an adjacency the model guessed.

func contextBRD() BRD {
	return BRD{Actors: []Actor{
		{ID: "actor-1", Name: "student", BusinessObject: "enrolment"},
		{ID: "actor-2", Name: "teacher", BusinessObject: "grade"},
	}}
}

func contextSRS() SRS {
	return SRS{FR: []FunctionalRequirement{
		{ID: "fr-1", Actor: "student", Delivery: DeliverySync, FailureMode: FailureFailFast, Concurrency: ConcurrencyPerKey},
		{ID: "fr-4", Actor: "teacher", Delivery: DeliverySync, FailureMode: FailureFailFast, Concurrency: ConcurrencyPerKey},
		{ID: "fr-9", Actor: "student", Delivery: DeliveryAtLeastOnce, FailureMode: FailureRetry, Concurrency: ConcurrencyPerKey},
	}}
}

func contextHLD() HLD {
	return HLD{
		Actors: []HLDActor{
			{ID: "a-student", Name: "student", BrdActorRef: "actor-1", Purpose: Text{Value: "Learns a course."}},
			{ID: "a-teacher", Name: "teacher", BrdActorRef: "actor-2", Purpose: Text{Value: "Grades the work."}},
		},
		ActorInteractions: []ActorInteraction{
			{ActorID: "a-student", ContainerID: "c-app", Description: Text{Value: "enrols in a course"}, CauseRef: "fr-1"},
			{ActorID: "a-teacher", ContainerID: "c-teacher", Description: Text{Value: "grades submitted work"}, CauseRef: "fr-4"},
		},
		BoundedContexts: []BoundedContext{
			{ID: "bc-1", Name: "Enrolment", PACELC: PACELCPaEl, ContainerIds: []string{"c-app", "c-store"}},
			{ID: "bc-2", Name: "Grading", PACELC: PACELCPaEl, ContainerIds: []string{"c-teacher"}},
		},
		Containers: []Container{
			{ObjectID: "c-app", BoundedContextID: "bc-1", Tech: TechGo, Purpose: Text{Value: "Serve the enrolment."}},
			{ObjectID: "c-teacher", BoundedContextID: "bc-2", Tech: TechGo, Purpose: Text{Value: "Serve the grading."}},
			{ObjectID: "c-store", BoundedContextID: "bc-1", Tech: TechPostgres, Purpose: Text{Value: "Keep the records."}},
			{ObjectID: "c-mail", Tech: TechExternal, Purpose: Text{Value: "Send the notices."}},
		},
		Aggregates: []Aggregate{{
			ID: "a-enrolment", ContainerID: "c-app", Root: "Enrolment",
			Invariants: []Statement{{Text: "A student enrols once.", Modality: "shall", Priority: "must"}},
		}},
		ContextRelations: []ContextRelation{
			{From: "c-app", To: "c-store", Type: RelationSync, Description: Text{Value: "writes the enrolment"}, CauseRef: "fr-1"},
			{From: "c-app", To: "c-mail", Type: RelationAsync, Guarantee: DeliveryAtLeastOnce, Description: Text{Value: "sends a grade notice"}, CauseRef: "fr-9"},
		},
		Decisions: []Decision{
			patternDecision("d-consistency-1", AreaConsistency, "bc-1", ""),
			patternDecision("d-consistency-2", AreaConsistency, "bc-2", ""),
			patternDecision("d-storage", AreaStorage, "", ""),
			patternDecision("d-architecture", AreaArchitecture, "", ""),
			{ID: "d-deployment", Area: AreaOps, Deployment: DeploymentServicePerContainer, Title: Text{Value: "Deployment"}, Decision: Text{Value: "One container per service."}, Alternatives: []Alternative{{Option: Text{Value: "One host"}, RejectedBecause: Text{Value: "The services deploy apart."}}}},
			patternDecision("d-outbox", AreaReliability, "", PatternTransactionalOutbox),
			patternDecision("d-dlq", AreaReliability, "", PatternDLQ),
		},
		DegradationStrategy: []Degradation{{
			Failure: "The mail system is down", Degradation: "The notice waits in the outbox",
			Detection: "A delivery report shows a failure", TimeoutMs: 1000,
		}},
	}
}

func contextDesign() HLD {
	h := contextHLD()
	for i := range h.Decisions {
		switch h.Decisions[i].ID {
		case "d-storage":
			h.Decisions[i].Ownership = OwnershipPerService
			h.Decisions[i].Techs = []string{TechPostgres}
			h.Decisions[i].ObjectRefs = []string{"c-store"}
		case "d-architecture":
			h.Decisions[i].Style = StyleMicroservices
		}
	}
	return h
}

func TestContextDesignPassesTheFormAndInputGates(t *testing.T) {
	hld := contextDesign()
	if err := ValidateHLD(hld); err != nil {
		t.Fatalf("the valid context design must pass the form gates, got %v", err)
	}
	if errs := ValidateHLDAgainstInputs(hld, contextBRD(), contextSRS()); len(errs) > 0 {
		t.Fatalf("the valid context design must pass the input gates, got %v", errs)
	}
}

// TestEveryBRDActorIsAnHLDActor: a role the business names cannot vanish from the
// context diagram.
func TestEveryBRDActorIsAnHLDActor(t *testing.T) {
	hld := contextDesign()
	hld.Actors = hld.Actors[:1]
	hld.ActorInteractions = hld.ActorInteractions[:1]
	errs := ValidateHLDAgainstInputs(hld, contextBRD(), contextSRS())
	if !containsProblem(errs, "teacher") {
		t.Fatalf("a missing BRD actor must fail, got %v", errs)
	}
}

// TestActorInteractionNamesTheRightRequirement: the person on the arrow is the
// actor of the requirement, and the requirement exists.
func TestActorInteractionNamesTheRightRequirement(t *testing.T) {
	hld := contextDesign()
	hld.ActorInteractions[0].CauseRef = "fr-99"
	if errs := ValidateHLDAgainstInputs(hld, contextBRD(), contextSRS()); !containsProblem(errs, "fr-99") {
		t.Fatalf("an unknown cause must fail, got %v", errs)
	}

	wrongActor := contextDesign()
	wrongActor.ActorInteractions[0].CauseRef = "fr-4"
	if errs := ValidateHLDAgainstInputs(wrongActor, contextBRD(), contextSRS()); !containsProblem(errs, "does not match") {
		t.Fatalf("a student arrow to a teacher requirement must fail, got %v", errs)
	}
}

// TestRelationCarriesItsRequirement: an edge without a description and a cause is
// a guessed adjacency.
func TestRelationCarriesItsRequirement(t *testing.T) {
	hld := contextDesign()
	hld.ContextRelations[0].Description = Text{}
	if errs := ValidateHLDAgainstInputs(hld, contextBRD(), contextSRS()); !containsProblem(errs, "description") {
		t.Fatalf("an edge without a business action must fail, got %v", errs)
	}

	noCause := contextDesign()
	noCause.ContextRelations[0].CauseRef = ""
	if errs := ValidateHLDAgainstInputs(noCause, contextBRD(), contextSRS()); !containsProblem(errs, "causeRef") {
		t.Fatalf("an edge without a requirement must fail, got %v", errs)
	}

	unknown := contextDesign()
	unknown.ContextRelations[0].CauseRef = "fr-99"
	if errs := ValidateHLDAgainstInputs(unknown, contextBRD(), contextSRS()); !containsProblem(errs, "fr-99") {
		t.Fatalf("an edge that cites a missing requirement must fail, got %v", errs)
	}
}

// TestRelationTypeFollowsTheRequirement: the SRS delivery premise decides sync or
// async; the model does not pick the style from the shape of the call.
func TestRelationTypeFollowsTheRequirement(t *testing.T) {
	syncAsAsync := contextDesign()
	syncAsAsync.ContextRelations[0].Type = RelationAsync
	syncAsAsync.ContextRelations[0].Guarantee = DeliveryAtLeastOnce
	if errs := ValidateHLDAgainstInputs(syncAsAsync, contextBRD(), contextSRS()); !containsProblem(errs, "sync") {
		t.Fatalf("a sync requirement with an async edge must fail, got %v", errs)
	}

	asyncAsSync := contextDesign()
	asyncAsSync.ContextRelations[1].Type = RelationSync
	asyncAsSync.ContextRelations[1].Guarantee = ""
	if errs := ValidateHLDAgainstInputs(asyncAsSync, contextBRD(), contextSRS()); !containsProblem(errs, "async") {
		t.Fatalf("an async requirement with a sync edge must fail, got %v", errs)
	}

	wrongGuarantee := contextDesign()
	wrongGuarantee.ContextRelations[1].Guarantee = DeliveryAtMostOnce
	if errs := ValidateHLDAgainstInputs(wrongGuarantee, contextBRD(), contextSRS()); !containsProblem(errs, "guarantee") {
		t.Fatalf("the guarantee must match the requirement, got %v", errs)
	}
}

// TestEveryRequirementIsCarried: an FR with no interaction and no edge is a
// requirement the design forgot.
func TestEveryRequirementIsCarried(t *testing.T) {
	srs := contextSRS()
	srs.FR = append(srs.FR, FunctionalRequirement{ID: "fr-7", Actor: "student", Delivery: DeliverySync})
	if errs := ValidateHLDAgainstInputs(contextDesign(), contextBRD(), srs); !containsProblem(errs, "fr-7") {
		t.Fatalf("an uncovered requirement must fail, got %v", errs)
	}
}

// TestEveryContainerIsUsed: a box with no arrow is a component the system does
// not use.
func TestEveryContainerIsUsed(t *testing.T) {
	hld := contextDesign()
	hld.Containers = append(hld.Containers, Container{
		ObjectID: "c-extra", BoundedContextID: "bc-1", Tech: TechGo, Purpose: Text{Value: "Nobody calls it."},
	})
	if errs := ValidateHLDAgainstInputs(hld, contextBRD(), contextSRS()); !containsProblem(errs, "c-extra") {
		t.Fatalf("an unused container must fail, got %v", errs)
	}
}

// TestExternalSystemStandsOutsideTheContext: an external system is not a
// container of this system, so it has no bounded context and no aggregate.
func TestExternalSystemStandsOutsideTheContext(t *testing.T) {
	hld := contextDesign()
	for i := range hld.Containers {
		if hld.Containers[i].ObjectID == "c-mail" {
			hld.Containers[i].BoundedContextID = "bc-1"
		}
	}
	if err := ValidateHLD(hld); err == nil || !containsProblem([]error{err}, "external") {
		t.Fatalf("an external system with a context must fail, got %v", err)
	}

	inContext := contextDesign()
	inContext.BoundedContexts[0].ContainerIds = append(inContext.BoundedContexts[0].ContainerIds, "c-mail")
	if err := ValidateHLD(inContext); err == nil || !containsProblem([]error{err}, "external") {
		t.Fatalf("a context that lists an external system must fail, got %v", err)
	}

	aggregate := contextDesign()
	aggregate.Aggregates = append(aggregate.Aggregates, Aggregate{
		ID: "a-mail", ContainerID: "c-mail", Root: "Message",
		Invariants: []Statement{{Text: "A message has one recipient.", Modality: "shall", Priority: "must"}},
	})
	if err := ValidateHLD(aggregate); err == nil || !containsProblem([]error{err}, "external") {
		t.Fatalf("an aggregate in an external system must fail, got %v", err)
	}
}

// TestInternalContainerNeedsAContext: only an external system may sit outside the
// system boundary.
func TestInternalContainerNeedsAContext(t *testing.T) {
	hld := contextDesign()
	hld.Containers[0].BoundedContextID = ""
	if err := ValidateHLD(hld); err == nil || !containsProblem([]error{err}, "boundedContextId") {
		t.Fatalf("an internal container without a context must fail, got %v", err)
	}
}

// TestActorReachesTheSystemNotAnExternalSystem: people talk to our containers,
// not to the payment provider.
func TestActorReachesTheSystemNotAnExternalSystem(t *testing.T) {
	hld := contextDesign()
	hld.ActorInteractions[0].ContainerID = "c-mail"
	if err := ValidateHLD(hld); err == nil || !containsProblem([]error{err}, "external") {
		t.Fatalf("an actor arrow to an external system must fail, got %v", err)
	}
}

// TestActorNeedsAnInteraction: a person on the diagram with no arrow explains
// nothing.
func TestActorNeedsAnInteraction(t *testing.T) {
	hld := contextDesign()
	hld.Actors = append(hld.Actors, HLDActor{ID: "a-admin", Name: "administrator", BrdActorRef: "actor-3"})
	if err := ValidateHLD(hld); err == nil || !containsProblem([]error{err}, "interaction") {
		t.Fatalf("an actor without an interaction must fail, got %v", err)
	}
}

// TestTermContextMustExist: a term scoped to a context that the design does not
// have is a definition with no boundary.
func TestTermContextMustExist(t *testing.T) {
	hld := contextDesign()
	hld.UbiquitousLanguage = []Term{{Term: "progress", Definition: Text{Value: "A step done."}, BoundedContextID: "bc-9"}}
	if err := ValidateHLD(hld); err == nil || !containsProblem([]error{err}, "bc-9") {
		t.Fatalf("a term scoped to a missing context must fail, got %v", err)
	}
}

// TestRelationEndpointsExist: a relation to a name the design never declares is a
// typo, not an integration.
func TestRelationEndpointsExist(t *testing.T) {
	hld := contextDesign()
	hld.ContextRelations[0].To = "c-ghost"
	if err := ValidateHLD(hld); err == nil || !containsProblem([]error{err}, "c-ghost") {
		t.Fatalf("a relation to a missing endpoint must fail, got %v", err)
	}
}

// TestContextGatesSurviveARandomDesign: the delivery premise decides the edge type
// on any legal combination. The seed is fixed so a failure reproduces.
func TestContextGatesSurviveARandomDesign(t *testing.T) {
	random := rand.New(rand.NewSource(97))
	deliveries := []Delivery{DeliverySync, DeliveryAtMostOnce, DeliveryAtLeastOnce, DeliveryExactlyOnce}
	types := []ContextRelationType{RelationSync, RelationAsync}

	for i := 0; i < 200; i++ {
		hld := contextDesign()
		srs := contextSRS()
		srs.FR[2].FailureMode = FailureFailFast
		delivery := deliveries[random.Intn(len(deliveries))]
		srs.FR[2].Delivery = delivery
		relation := &hld.ContextRelations[1]
		relation.Type = types[random.Intn(len(types))]
		relation.Guarantee = ""
		if relation.Type == RelationAsync {
			relation.Guarantee = delivery
			if delivery == DeliverySync {
				relation.Guarantee = DeliveryAtLeastOnce
			}
		}

		want := true
		switch delivery {
		case DeliverySync:
			want = relation.Type == RelationSync && relation.Guarantee == ""
		case DeliveryAtMostOnce, DeliveryAtLeastOnce, DeliveryExactlyOnce:
			want = relation.Type == RelationAsync && relation.Guarantee == delivery
		}
		errs := ValidateHLDAgainstInputs(hld, contextBRD(), srs)
		if want && len(errs) > 0 {
			t.Fatalf("iteration %d: delivery %s, type %s, guarantee %s must pass, got %v",
				i, delivery, relation.Type, relation.Guarantee, errs)
		}
		if !want && len(errs) == 0 {
			t.Fatalf("iteration %d: delivery %s, type %s, guarantee %s must fail",
				i, delivery, relation.Type, relation.Guarantee)
		}
	}
}

func containsProblem(errs []error, needle string) bool {
	for _, err := range errs {
		if err != nil && strings.Contains(err.Error(), needle) {
			return true
		}
	}
	return false
}
