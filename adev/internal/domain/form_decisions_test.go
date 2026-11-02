package domain

import (
	"math/rand"
	"strings"
	"testing"
)

// D1/D4/D6: the class of a context, the storage class and the architecture style
// are decisions of the design, not fields scattered over it. Each one is an entry
// with its area, its evidence and its rejected options.

// design builds an HLD with one context, n application containers, an optional
// store container and the decisions the caller names.
func design(style ArchitectureStyle, appContainers int, store string) HLD {
	contexts := []BoundedContext{{
		ID: "bc-1", Name: "Delivery", Purpose: Text{Value: "Move parcels."},
		PACELC: PACELCPaEl,
	}}
	containers := make([]Container, 0, appContainers+1)
	for i := 0; i < appContainers; i++ {
		containers = append(containers, Container{
			ObjectID: "c-" + string(rune('a'+i)), BoundedContextID: "bc-1",
			Purpose: Text{Value: "Serve the client."}, Tech: TechGo,
		})
	}
	if store != "" {
		containers = append(containers, Container{
			ObjectID: "c-store", BoundedContextID: "bc-1",
			Purpose: Text{Value: "Keep the records."}, Tech: store,
		})
	}
	contexts[0].ContainerIds = make([]string, 0, len(containers))
	for _, container := range containers {
		contexts[0].ContainerIds = append(contexts[0].ContainerIds, container.ObjectID)
	}

	decisions := []Decision{{
		ID: "d-consistency", Area: AreaConsistency, ContextRef: "bc-1",
		Title:    Text{Value: "Consistency"},
		Decision: Text{Value: "The context answers after the write."},
		Alternatives: []Alternative{{
			Option:          Text{Value: "Eventual"},
			RejectedBecause: Text{Value: "The client confirms the receipt."},
		}},
	}}
	if store != "" {
		decisions = append(decisions, Decision{
			ID: "d-storage", Area: AreaStorage, Ownership: OwnershipPerService, Title: Text{Value: "Storage"},
			Decision: Text{Value: "PostgreSQL."},
			Alternatives: []Alternative{{
				Option:          Text{Value: "MongoDB"},
				RejectedBecause: Text{Value: "The route needs transactions."},
			}},
			ObjectRefs: []string{"c-store"}, Techs: []string{store},
		})
	}
	decisions = append(decisions, Decision{
		ID: "d-architecture", Area: AreaArchitecture, Style: style,
		Title:    Text{Value: "Architecture"},
		Decision: Text{Value: "The style the design implements."},
		Alternatives: []Alternative{{
			Option:          Text{Value: "The other style"},
			RejectedBecause: Text{Value: "The team size and the budget decide."},
		}},
	})
	deployment := DeploymentServicePerContainer
	if style == StyleServerless {
		deployment = DeploymentServerless
	}
	decisions = append(decisions, Decision{
		ID: "d-deployment", Area: AreaOps, Deployment: deployment,
		Title:    Text{Value: "Deployment"},
		Decision: Text{Value: "The containers run as the pattern says."},
		Alternatives: []Alternative{{
			Option:          Text{Value: "The other pattern"},
			RejectedBecause: Text{Value: "The style decides."},
		}},
	})

	aggregates := make([]Aggregate, 0, appContainers)
	for i := 0; i < appContainers; i++ {
		aggregates = append(aggregates, Aggregate{
			ID: "a-" + string(rune('a'+i)), ContainerID: "c-" + string(rune('a'+i)),
			Root: "shipment", ModuleRef: "m-1",
			Invariants: []Statement{{Text: "A shipment has one route.", Modality: "shall", Priority: "must"}},
		})
	}

	hld := hldWith(contexts, containers, nil)
	hld.Decisions = decisions
	hld.Aggregates = aggregates
	return hld
}

// TestHLDNeedsAStorageDecisionWhenItHasAStore: a store container without a storage
// decision is a product nobody chose.
func TestHLDNeedsAStorageDecisionWhenItHasAStore(t *testing.T) {
	hld := design(StyleModularMonolith, 1, TechPostgres)
	hld.Decisions = withoutArea(hld.Decisions, AreaStorage)
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "storage") {
		t.Fatalf("a design with a store must decide the storage, got %v", err)
	}

	noStore := design(StyleModularMonolith, 1, "")
	if err := ValidateHLD(noStore); err != nil {
		t.Fatalf("a design without a store needs no storage decision, got %v", err)
	}
}

// TestHLDNeedsAConsistencyDecisionPerContext: the class lives on the context and in
// a decision with its rejected options, so the choice has an audit trail.
func TestHLDNeedsAConsistencyDecisionPerContext(t *testing.T) {
	hld := design(StyleModularMonolith, 1, "")
	hld.Decisions = withoutArea(hld.Decisions, AreaConsistency)
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "bc-1") {
		t.Fatalf("a context without a consistency decision must fail, got %v", err)
	}

	foreign := design(StyleModularMonolith, 1, "")
	foreign.Decisions[0].ContextRef = "bc-9"
	if err := ValidateHLD(foreign); err == nil || !strings.Contains(err.Error(), "bc-9") {
		t.Fatalf("a decision about a context that does not exist must fail, got %v", err)
	}
}

// TestHLDArchitectureStyleMatchesTheContainers: the style is a claim about the
// deployment, and the containers prove or break it.
func TestHLDArchitectureStyleMatchesTheContainers(t *testing.T) {
	cases := []struct {
		style      ArchitectureStyle
		containers int
		wantError  bool
	}{
		{StyleModularMonolith, 1, false},
		{StyleModularMonolith, 2, true},
		{StyleMonolith, 1, false},
		{StyleMicroservices, 2, false},
		{StyleMicroservices, 3, false},
		{StyleMicroservices, 1, true},
		{StyleEventDriven, 1, true},
		{StyleServerless, 2, false},
	}
	for _, tc := range cases {
		hld := design(tc.style, tc.containers, "")
		err := ValidateHLD(hld)
		if tc.wantError && err == nil {
			t.Fatalf("%s over %d containers must fail", tc.style, tc.containers)
		}
		if !tc.wantError && err != nil {
			t.Fatalf("%s over %d containers must pass, got %v", tc.style, tc.containers, err)
		}
	}
}

// TestHLDDecisionAreaIsAClosedList: an area name outside the ten says nothing and
// cannot be scored.
func TestHLDDecisionAreaIsAClosedList(t *testing.T) {
	hld := design(StyleModularMonolith, 1, "")
	hld.Decisions[0].Area = "magic"
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "area") {
		t.Fatalf("an unknown area must fail, got %v", err)
	}
}

// TestHLDNeedsExactlyOneArchitectureDecision: two styles are no style.
func TestHLDNeedsExactlyOneArchitectureDecision(t *testing.T) {
	hld := design(StyleModularMonolith, 1, "")
	var architecture Decision
	for _, decision := range hld.Decisions {
		if decision.Area == AreaArchitecture {
			architecture = decision
			break
		}
	}
	architecture.ID = "d-architecture-2"
	hld.Decisions = append(hld.Decisions, architecture)
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "architecture") {
		t.Fatalf("two architecture decisions must fail, got %v", err)
	}
}

// TestDecisionRulesSurviveARandomDesign: the gate agrees with an oracle on any
// style and container count, and a store always brings its decision. The seed is
// fixed so a failure reproduces.
func TestDecisionRulesSurviveARandomDesign(t *testing.T) {
	random := rand.New(rand.NewSource(31))
	styles := []ArchitectureStyle{StyleModularMonolith, StyleMonolith, StyleMicroservices, StyleEventDriven, StyleServerless, StyleHexagonal, StyleCellBased}
	stores := []string{"", TechPostgres, TechMongo, TechCassandra}

	for i := 0; i < 200; i++ {
		style := styles[random.Intn(len(styles))]
		containers := 1 + random.Intn(4)
		store := stores[random.Intn(len(stores))]
		hld := design(style, containers, store)

		wantRefused := false
		switch style {
		case StyleModularMonolith, StyleMonolith, StyleHexagonal:
			wantRefused = containers > 1
		case StyleMicroservices, StyleEventDriven, StyleCellBased, StyleServerless:
			wantRefused = containers < 2
		}
		err := ValidateHLD(hld)
		if wantRefused && err == nil {
			t.Fatalf("iteration %d: %s over %d containers must fail", i, style, containers)
		}
		if !wantRefused && err != nil {
			t.Fatalf("iteration %d: %s over %d containers must pass, got %v", i, style, containers, err)
		}
	}
}

// withoutArea drops every decision of one area.
func withoutArea(decisions []Decision, area DecisionArea) []Decision {
	kept := make([]Decision, 0, len(decisions))
	for _, decision := range decisions {
		if decision.Area != area {
			kept = append(kept, decision)
		}
	}
	return kept
}

// D4.4 and D5: the storage decision names who owns the data, and every relation
// names the guarantee it buys. Both are decisions the gate can check.

func designWithStore(style ArchitectureStyle) HLD {
	return design(style, 1, TechPostgres)
}

// TestStorageDecisionNamesOwnership: a store without an owner is a database
// somebody will share by accident.
func TestStorageDecisionNamesOwnership(t *testing.T) {
	hld := designWithStore(StyleModularMonolith)
	for i := range hld.Decisions {
		if hld.Decisions[i].Area == AreaStorage {
			hld.Decisions[i].Ownership = ""
		}
	}
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "ownership") {
		t.Fatalf("a storage decision must name the ownership, got %v", err)
	}
	hld = designWithStore(StyleModularMonolith)
	for i := range hld.Decisions {
		if hld.Decisions[i].Area == AreaStorage {
			hld.Decisions[i].Ownership = OwnershipShared
		}
	}
	if err := ValidateHLD(hld); err != nil {
		t.Fatalf("a monolith with a shared database must pass, got %v", err)
	}
}

// TestOwnershipAgainstTheStyle: services that deploy apart cannot share one
// database without coupling every schema change to every team.
func TestOwnershipAgainstTheStyle(t *testing.T) {
	hld := design(StyleMicroservices, 2, TechPostgres)
	for i := range hld.Decisions {
		if hld.Decisions[i].Area == AreaStorage {
			hld.Decisions[i].Ownership = OwnershipShared
		}
	}
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "shared_database") {
		t.Fatalf("microservices over one shared database must fail, got %v", err)
	}

	hld = design(StyleMicroservices, 2, TechPostgres)
	for i := range hld.Decisions {
		if hld.Decisions[i].Area == AreaStorage {
			hld.Decisions[i].Ownership = OwnershipPerService
		}
	}
	if err := ValidateHLD(hld); err != nil {
		t.Fatalf("a database per service must pass, got %v", err)
	}
}

// TestAsyncRelationNamesAGuarantee: a message without a delivery guarantee is a
// promise nobody can test.
func TestAsyncRelationNamesAGuarantee(t *testing.T) {
	hld := designWithStore(StyleModularMonolith)
	hld.Decisions = append(hld.Decisions, Decision{
		ID: "d-communication", Area: AreaCommunication, ContextRef: "bc-1",
		Title:    Text{Value: "Communication"},
		Decision: Text{Value: "Events between the contexts."},
		Alternatives: []Alternative{{
			Option:          Text{Value: "Sync calls"},
			RejectedBecause: Text{Value: "The caller may continue."},
		}},
	})
	hld.ContextRelations = []ContextRelation{{From: "bc-1", To: "bc-1", Type: RelationAsync}}
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "contextRelations[0].guarantee") {
		t.Fatalf("an async relation must name its guarantee, got %v", err)
	}

	hld.ContextRelations[0].Guarantee = DeliveryAtLeastOnce
	hld.Decisions = append(hld.Decisions,
		patternDecision("d-outbox", AreaReliability, "", PatternTransactionalOutbox),
		patternDecision("d-dlq", AreaReliability, "", PatternDLQ))
	if err := ValidateHLD(hld); err != nil {
		t.Fatalf("at-least-once must pass, got %v", err)
	}
}

// TestAsyncIntoAConsistentContextNeedsAKey: an at-least-once event into a context
// that promises consistency needs the key that makes the duplicate safe.
func TestAsyncIntoAConsistentContextNeedsAKey(t *testing.T) {
	hld := hldWith(
		[]BoundedContext{{
			ID: "bc-1", Name: "Delivery", Purpose: Text{Value: "Move parcels."},
			PACELC: PACELCPcEc, ContainerIds: []string{"c-1"},
		}},
		shipmentContainers(TechGo), nil,
	)
	hld.Decisions = append(hld.Decisions, Decision{
		ID: "d-communication", Area: AreaCommunication, ContextRef: "bc-1",
		Title:    Text{Value: "Communication"},
		Decision: Text{Value: "Events between the contexts."},
		Alternatives: []Alternative{{
			Option:          Text{Value: "Sync calls"},
			RejectedBecause: Text{Value: "The caller may continue."},
		}},
	})
	hld.ContextRelations = []ContextRelation{
		{From: "bc-1", To: "bc-1", Type: RelationSync},
		{From: "bc-1", To: "bc-1", Type: RelationAsync, Guarantee: DeliveryAtLeastOnce},
	}
	hld.Decisions = append(hld.Decisions,
		patternDecision("d-outbox", AreaReliability, "", PatternTransactionalOutbox),
		patternDecision("d-dlq", AreaReliability, "", PatternDLQ))
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "idempotencyKey") {
		t.Fatalf("a consistent context needs the idempotency key, got %v", err)
	}
	hld.ContextRelations[1].IdempotencyKey = "shipmentId"
	if err := ValidateHLD(hld); err != nil {
		t.Fatalf("the key must satisfy the rule, got %v", err)
	}
}

// TestOwnershipAndGuaranteeSurviveARandomDesign: the two rules agree with an oracle
// on any legal combination. The seed is fixed so a failure reproduces.
func TestOwnershipAndGuaranteeSurviveARandomDesign(t *testing.T) {
	random := rand.New(rand.NewSource(47))
	styles := []ArchitectureStyle{StyleModularMonolith, StyleMicroservices, StyleMonolith}
	ownerships := []Ownership{OwnershipPerService, OwnershipShared, OwnershipReadModel}
	guarantees := []Delivery{"", DeliveryAtLeastOnce, DeliveryAtMostOnce, DeliveryExactlyOnce}

	for i := 0; i < 200; i++ {
		style := styles[random.Intn(len(styles))]
		ownership := ownerships[random.Intn(len(ownerships))]
		guarantee := guarantees[random.Intn(len(guarantees))]

		containers := 1
		if style.DeployedApart() {
			containers = 2
		}
		hld := design(style, containers, TechPostgres)
		for j := range hld.Decisions {
			if hld.Decisions[j].Area == AreaStorage {
				hld.Decisions[j].Ownership = ownership
			}
		}
		hld.ContextRelations = []ContextRelation{{
			From: "bc-1", To: "bc-1", Type: RelationAsync, Guarantee: guarantee,
		}}
		hld.Decisions = append(hld.Decisions,
			patternDecision("d-outbox", AreaReliability, "", PatternTransactionalOutbox))
		if guarantee == DeliveryAtLeastOnce || guarantee == DeliveryExactlyOnce {
			hld.Decisions = append(hld.Decisions,
				patternDecision("d-dlq", AreaReliability, "", PatternDLQ))
		}

		err := ValidateHLD(hld)
		wantRefused := guarantee == ""
		if style.DeployedApart() && ownership == OwnershipShared {
			wantRefused = true
		}
		if wantRefused && err == nil {
			t.Fatalf("iteration %d: style %s, ownership %s, guarantee %q must fail", i, style, ownership, guarantee)
		}
		if !wantRefused && err != nil {
			t.Fatalf("iteration %d: style %s, ownership %s, guarantee %q must pass, got %v", i, style, ownership, guarantee, err)
		}
	}
}
