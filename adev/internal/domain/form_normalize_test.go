package domain

import "testing"

func normalizeDesign() HLD {
	return HLD{
		BoundedContexts: []BoundedContext{{ID: "bc-1", Name: "Delivery", ContainerIds: []string{"c-app"}}},
		Containers: []Container{{
			ObjectID: "c-app", BoundedContextID: "bc-1", Tech: TechGo,
			Purpose: Text{Value: "Serve the client."},
		}},
		Decisions: []Decision{{
			ID: "d-architecture", Area: AreaArchitecture, Style: StyleMicroservices,
			Title:    Text{Value: "Architecture"},
			Decision: Text{Value: "Microservices."},
			Alternatives: []Alternative{{
				Option: Text{Value: "Monolith"}, RejectedBecause: Text{Value: "The team deploys apart."},
			}},
		}},
	}
}

// TestNormalizeHLDMovesDeploymentToOps: a deployment that landed on the
// architecture decision is moved to one ops decision, and the style stays where
// it belongs.
func TestNormalizeHLDMovesDeploymentToOps(t *testing.T) {
	hld := normalizeDesign()
	hld.Decisions[0].Area = AreaOps
	hld.Decisions[0].Deployment = DeploymentServicePerContainer
	hld.Decisions = append(hld.Decisions, Decision{
		ID: "d-ops", Area: AreaOps, Deployment: DeploymentServicePerContainer,
		Title: Text{Value: "Deployment"}, Decision: Text{Value: "One container."},
	})

	NormalizeHLD(&hld)

	if hld.Decisions[0].Area != AreaArchitecture {
		t.Fatalf("the style must return to the architecture decision, got %q", hld.Decisions[0].Area)
	}
	if hld.Decisions[0].Deployment != "" {
		t.Fatalf("the architecture decision must carry no deployment, got %q", hld.Decisions[0].Deployment)
	}
	deployments := 0
	for _, decision := range hld.Decisions {
		if decision.Deployment != "" {
			deployments++
			if decision.Area != AreaOps {
				t.Fatalf("a deployment belongs to ops, got %q", decision.Area)
			}
		}
	}
	if deployments != 1 {
		t.Fatalf("want one deployment decision, got %d", deployments)
	}
}

// TestNormalizeHLDAddsDeploymentFromStyle: the classifier picks the pattern from
// the style and the system writes the decision, so no model answer is needed for
// the structural fork.
func TestNormalizeHLDAddsDeploymentFromStyle(t *testing.T) {
	hld := normalizeDesign()
	NormalizeHLD(&hld)

	var ops *Decision
	for i := range hld.Decisions {
		if hld.Decisions[i].Area == AreaOps {
			ops = &hld.Decisions[i]
		}
	}
	if ops == nil {
		t.Fatal("the normalizer must add the ops decision")
	}
	if ops.Deployment != DeploymentServicePerContainer {
		t.Fatalf("microservices must deploy per container, got %q", ops.Deployment)
	}
	if len(ops.Checklist) == 0 {
		t.Fatal("the deployment decision must carry the system checklist")
	}
	if ops.Title.Value == "" || ops.Decision.Value == "" || len(ops.Alternatives) == 0 {
		t.Fatalf("the system decision must be complete: %+v", ops)
	}
}

// TestNormalizeHLDAddsTheStoreTheDecisionNames: a storage decision that names
// PostgreSQL but no container creates the store container it implies.
func TestNormalizeHLDAddsTheStoreTheDecisionNames(t *testing.T) {
	hld := normalizeDesign()
	hld.Decisions = append(hld.Decisions, Decision{
		ID: "d-storage", Area: AreaStorage, Ownership: OwnershipPerService,
		Title: Text{Value: "Storage"}, Decision: Text{Value: "PostgreSQL."},
		Techs: []string{TechPostgres}, ObjectRefs: []string{"c-app"},
	})

	NormalizeHLD(&hld)

	store := ""
	for _, container := range hld.Containers {
		if container.Tech == TechPostgres {
			store = container.ObjectID
		}
	}
	if store == "" {
		t.Fatal("the storage decision must create its store container")
	}
	if len(hld.BoundedContexts[0].ContainerIds) != 2 || hld.BoundedContexts[0].ContainerIds[1] != store {
		t.Fatalf("the store must join its context, got %+v", hld.BoundedContexts[0].ContainerIds)
	}
	storage := Decision{}
	for _, decision := range hld.Decisions {
		if decision.ID == "d-storage" {
			storage = decision
		}
	}
	if len(storage.ObjectRefs) != 2 || storage.ObjectRefs[1] != store {
		t.Fatalf("the storage decision must name its store, got %+v", storage.ObjectRefs)
	}

	NormalizeHLD(&hld)
	if len(hld.Containers) != 2 {
		t.Fatalf("a second pass must not duplicate the store, got %+v", hld.Containers)
	}
}

// TestClassifyDeployment: the style decides the deployment pattern.
func TestClassifyDeployment(t *testing.T) {
	cases := map[ArchitectureStyle]DeploymentPattern{
		StyleServerless:      DeploymentServerless,
		StyleMicroservices:   DeploymentServicePerContainer,
		StyleEventDriven:     DeploymentServicePerContainer,
		StyleCellBased:       DeploymentServicePerContainer,
		StyleMonolith:        DeploymentServicePerHost,
		StyleModularMonolith: DeploymentServicePerHost,
		StyleHexagonal:       DeploymentServicePerHost,
	}
	for style, want := range cases {
		got, ok := ClassifyDeployment(style)
		if !ok || got != want {
			t.Fatalf("%s must classify as %s, got %q (%v)", style, want, got, ok)
		}
	}
}

// TestAddExternalSystemsFromTheBRDAssumptions: the BRD names a card payment and
// email/push notifications; the design gets the two C4 external systems and the
// relations from the containers that use them.
func TestAddExternalSystemsFromTheBRDAssumptions(t *testing.T) {
	hld := HLD{
		BoundedContexts: []BoundedContext{{ID: "bc-1", ContainerIds: []string{"c-pay", "c-progress"}}},
		Containers: []Container{
			{ObjectID: "c-pay", BoundedContextID: "bc-1", Tech: TechGo, Purpose: Text{Value: "Process student payments."}},
			{ObjectID: "c-progress", BoundedContextID: "bc-1", Tech: TechGo, Purpose: Text{Value: "Track progress."}},
		},
		ActorInteractions: []ActorInteraction{{
			ActorID: "a-student", ContainerID: "c-progress", CauseRef: "fr-9",
			Description: Text{Value: "learns about a grade"},
		}},
	}
	brd := BRD{Assumptions: []Text{
		{Value: "The student pays for a course by card; the platform keeps a commission; the teacher gets a payout once a month."},
		{Value: "The student learns about a new grade by email and push; the teacher learns about a submitted task by email."},
	}}
	srs := SRS{FR: []FunctionalRequirement{
		{ID: "fr-2", Actor: "student", Delivery: DeliverySync, Trigger: "sign-in", Statement: Statement{Text: "The student signs in with an email and a password."}},
		{ID: "fr-8", Actor: "student", Delivery: DeliverySync, Trigger: "payment", Statement: Statement{Text: "The student pays for a course by card."}},
		{ID: "fr-9", Actor: "student", Delivery: DeliverySync, Trigger: "notification", Statement: Statement{Text: "The student learns about a grade by email and push."}},
	}}

	AddExternalSystems(&hld, brd, srs)

	externals := map[string]ContextRelation{}
	for _, container := range hld.Containers {
		if container.External() {
			if container.BoundedContextID != "" {
				t.Fatalf("an external system stands outside every context, got %q", container.BoundedContextID)
			}
			externals[container.ObjectID] = ContextRelation{}
		}
	}
	if len(externals) != 2 {
		t.Fatalf("want a payment gateway and a notification provider, got %+v", hld.Containers)
	}
	for _, relation := range hld.ContextRelations {
		if _, ok := externals[relation.To]; ok {
			externals[relation.To] = relation
		}
	}
	if externals["c-payment-gateway"].CauseRef != "fr-8" {
		t.Fatalf("the payment gateway must carry fr-8, got %+v", externals["c-payment-gateway"])
	}
	if externals["c-notification-provider"].CauseRef != "fr-9" {
		t.Fatalf("the notification provider must carry fr-9, got %+v", externals["c-notification-provider"])
	}
	if externals["c-notification-provider"].From != "c-progress" {
		t.Fatalf("the notification provider must be used by the progress container, got %+v", externals["c-notification-provider"])
	}

	AddExternalSystems(&hld, brd, srs)
	if len(hld.Containers) != 4 {
		t.Fatalf("a second pass must not duplicate the external systems, got %+v", hld.Containers)
	}
}

// TestExternalSystemPremiseAndPresence: the gate asks for an external system only
// when the BRD names one, and sees the boxes the normalizer adds.
func TestExternalSystemPremiseAndPresence(t *testing.T) {
	plain := BRD{Assumptions: []Text{{Value: "The system keeps the records."}}}
	if ExternalSystemPremise(plain) {
		t.Fatal("a BRD with no external system must not demand one")
	}
	named := BRD{Assumptions: []Text{{Value: "The student pays by card and receives email."}}}
	if !ExternalSystemPremise(named) {
		t.Fatal("a BRD that names a card and email must demand an external system")
	}
	if HasExternalSystem(HLD{}) {
		t.Fatal("an empty design has no external system")
	}
}
