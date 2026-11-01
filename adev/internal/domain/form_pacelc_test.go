package domain

import (
	"math/rand"
	"strings"
	"testing"
)

// D1: the consistency class of a bounded context is decided before the storage and
// the communication, because it rules both of them out. The class is a closed set,
// the tech is a closed set, and the gates check the pair.

func hldWith(contexts []BoundedContext, containers []Container, relations []ContextRelation) HLD {
	decisions := make([]Decision, 0, len(contexts)+2)
	for _, context := range contexts {
		decisions = append(decisions, Decision{
			ID: "d-consistency-" + context.ID, Area: AreaConsistency, ContextRef: context.ID,
			Title:    Text{Value: "Consistency", Lang: LangSTE100},
			Decision: Text{Value: "The context answers as its class says.", Lang: LangSTE100},
			Alternatives: []Alternative{{
				Option:          Text{Value: "The other class", Lang: LangSTE100},
				RejectedBecause: Text{Value: "The business needs this promise.", Lang: LangSTE100},
			}},
		})
	}
	code, store := 0, ""
	for _, container := range containers {
		if _, ok := PathRoot(container.Tech); ok {
			code++
		}
		if storeTechs[strings.ToLower(container.Tech)] && store == "" {
			store = container.Tech
		}
	}
	style := StyleModularMonolith
	if code > 1 {
		style = StyleMicroservices
	}
	decisions = append(decisions, Decision{
		ID: "d-architecture", Area: AreaArchitecture, Style: style,
		Title:    Text{Value: "Architecture", Lang: LangSTE100},
		Decision: Text{Value: "The style the containers carry.", Lang: LangSTE100},
		Alternatives: []Alternative{{
			Option:          Text{Value: "The other style", Lang: LangSTE100},
			RejectedBecause: Text{Value: "The container count decides.", Lang: LangSTE100},
		}},
	})
	if store != "" {
		decisions = append(decisions, Decision{
			ID: "d-storage", Area: AreaStorage,
			Title:    Text{Value: "Storage", Lang: LangSTE100},
			Decision: Text{Value: "The store the design carries.", Lang: LangSTE100},
			Alternatives: []Alternative{{
				Option:          Text{Value: "The other store", Lang: LangSTE100},
				RejectedBecause: Text{Value: "The class decides.", Lang: LangSTE100},
			}},
			Techs: []string{store},
		})
	}
	return HLD{
		BoundedContexts: contexts,
		Containers:      containers,
		Aggregates: []Aggregate{{
			ID: "a-1", ContainerID: containers[0].ObjectID, Root: "shipment", ModuleRef: "m-1",
			Invariants: []Statement{{Text: "A shipment has one route.", Modality: "shall", Priority: "must"}},
		}},
		ContextRelations:    relations,
		Decisions:           decisions,
		DegradationStrategy: []Degradation{{Failure: "The database is down", Degradation: "Last status", Detection: "Health check", TimeoutMs: 1000}},
	}
}

func shipmentContexts(pacelc PACELC) []BoundedContext {
	return []BoundedContext{{
		ID: "bc-1", Name: "Delivery", Purpose: Text{Value: "Move parcels.", Lang: LangSTE100},
		ContainerIds: []string{"c-1"}, PACELC: pacelc,
	}}
}

func shipmentContainers(tech string) []Container {
	return []Container{{
		ObjectID: "c-1", BoundedContextID: "bc-1",
		Purpose: Text{Value: "Serve the client.", Lang: LangSTE100}, Tech: tech,
	}}
}

// TestPathRootOfEveryCodeTech: the layout is a table, not an interpretation of a
// sentence. A product with no code has no root.
func TestPathRootOfEveryCodeTech(t *testing.T) {
	cases := map[string]string{
		TechGo: "internal", TechJava: "src/main/java", TechNode: "src", TechPython: "src",
		TechReact: "src", TechAngular: "src", TechVue: "src",
	}
	for tech, want := range cases {
		got, ok := PathRoot(tech)
		if !ok || got != want {
			t.Fatalf("root of %q: got %q %v, want %q", tech, got, ok, want)
		}
	}
	for _, tech := range []string{TechPostgres, TechKafka, TechKeycloak, TechNginx, TechExternal, ""} {
		if _, ok := PathRoot(tech); ok {
			t.Fatalf("tech %q holds no code and must have no root", tech)
		}
	}
}

// TestPathTemplateFollowsTheTech: the template puts the module and the layer where
// the technology expects them.
func TestPathTemplateFollowsTheTech(t *testing.T) {
	cases := map[string]string{
		TechGo: "internal/<layer>/<module>", TechJava: "src/main/java/<module>/<layer>",
		TechReact: "src/<layer>/<module>",
	}
	for tech, want := range cases {
		got, ok := PathTemplate(tech)
		if !ok || got != want {
			t.Fatalf("template of %q: got %q %v, want %q", tech, got, ok, want)
		}
	}
}

// TestHLDNeedsAPACELCClass: every bounded context answers the class before the
// design chooses a store or a relation.
func TestHLDNeedsAPACELCClass(t *testing.T) {
	for _, class := range []PACELC{"", "AP", "pc/ec", "strong"} {
		hld := hldWith(shipmentContexts(class), shipmentContainers(TechGo), nil)
		err := ValidateHLD(hld)
		if err == nil || !strings.Contains(err.Error(), "boundedContexts[0].pacelc") {
			t.Fatalf("class %q must be refused with the field named, got %v", class, err)
		}
	}
	for _, class := range []PACELC{PACELCPaEl, PACELCPcEc, PACELCPaEc, PACELCPcEl} {
		hld := hldWith(shipmentContexts(class), shipmentContainers(TechGo), nil)
		if err := ValidateHLD(hld); err != nil {
			t.Fatalf("class %q must pass, got %v", class, err)
		}
	}
}

// TestHLDTechIsAClosedList: a product outside the list has no layout and no
// capability, so the design must not name one.
func TestHLDTechIsAClosedList(t *testing.T) {
	for _, tech := range []string{"api-gateway", "ruby", "", "GO "} {
		hld := hldWith(shipmentContexts(PACELCPaEl), shipmentContainers(tech), nil)
		if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "containers[0].tech") {
			t.Fatalf("tech %q must be refused, got %v", tech, err)
		}
	}
	for _, tech := range []string{TechGo, TechJava, TechNode, TechPython} {
		hld := hldWith(shipmentContexts(PACELCPaEl), shipmentContainers(tech), nil)
		if err := ValidateHLD(hld); err != nil {
			t.Fatalf("tech %q must pass, got %v", tech, err)
		}
	}
}

// TestPACELCClassAgainstTheStore: a consistent context cannot rest on an
// AP-only store, and an available context may.
func TestPACELCClassAgainstTheStore(t *testing.T) {
	cassandra := hldWith(shipmentContexts(PACELCPcEc), shipmentContainers(TechCassandra), nil)
	if err := ValidateHLD(cassandra); err == nil || !strings.Contains(err.Error(), "pc/ec") {
		t.Fatalf("PC/EC over Cassandra must be refused, got %v", err)
	}

	postgres := hldWith(shipmentContexts(PACELCPcEc), shipmentContainers(TechPostgres), nil)
	if err := ValidateHLD(postgres); err != nil {
		t.Fatalf("PC/EC over PostgreSQL must pass, got %v", err)
	}

	available := hldWith(shipmentContexts(PACELCPaEl), shipmentContainers(TechCassandra), nil)
	if err := ValidateHLD(available); err != nil {
		t.Fatalf("PA/EL over Cassandra must pass, got %v", err)
	}
}

// TestPACELCClassAgainstTheRelations: a context that promises consistency cannot
// be wired only by asynchronous edges, because nothing then waits for its answer.
func TestPACELCClassAgainstTheRelations(t *testing.T) {
	contexts := append(shipmentContexts(PACELCPcEc), BoundedContext{
		ID: "bc-2", Name: "Billing", Purpose: Text{Value: "Bill the client.", Lang: LangSTE100},
		ContainerIds: []string{"c-2"}, PACELC: PACELCPcEc,
	})
	containers := append(shipmentContainers(TechGo), Container{
		ObjectID: "c-2", BoundedContextID: "bc-2",
		Purpose: Text{Value: "Bill.", Lang: LangSTE100}, Tech: TechGo,
	})
	asyncOnly := hldWith(contexts, containers, []ContextRelation{
		{From: "bc-1", To: "bc-2", Type: RelationAsync},
	})
	asyncOnly.Aggregates = append(asyncOnly.Aggregates, Aggregate{
		ID: "a-2", ContainerID: "c-2", Root: "invoice", ModuleRef: "m-2",
		Invariants: []Statement{{Text: "One invoice per shipment.", Modality: "shall", Priority: "must"}},
	})
	if err := ValidateHLD(asyncOnly); err == nil || !strings.Contains(err.Error(), "sync") {
		t.Fatalf("PC/EC wired only by async edges must be refused, got %v", err)
	}

	withSync := hldWith(contexts, containers, []ContextRelation{
		{From: "bc-1", To: "bc-2", Type: RelationAsync},
		{From: "bc-2", To: "bc-1", Type: RelationSync},
	})
	withSync.Aggregates = append(withSync.Aggregates, Aggregate{
		ID: "a-2", ContainerID: "c-2", Root: "invoice", ModuleRef: "m-2",
		Invariants: []Statement{{Text: "One invoice per shipment.", Modality: "shall", Priority: "must"}},
	})
	if err := ValidateHLD(withSync); err != nil {
		t.Fatalf("one sync edge must satisfy the rule, got %v", err)
	}
}

// TestPACELCRulesSurviveARandomDesign: the gate agrees with an independent oracle
// on any legal design, and a single broken field is named. The seed is fixed.
func TestPACELCRulesSurviveARandomDesign(t *testing.T) {
	random := rand.New(rand.NewSource(23))
	classes := []PACELC{PACELCPaEl, PACELCPcEc, PACELCPaEc, PACELCPcEl}
	techs := []string{TechGo, TechJava, TechNode, TechPostgres, TechMongo, TechCassandra, TechRedis}

	for i := 0; i < 200; i++ {
		tech := techs[random.Intn(len(techs))]
		class := classes[random.Intn(len(classes))]
		hld := hldWith(shipmentContexts(class), shipmentContainers(tech), nil)

		err := ValidateHLD(hld)
		storeIsAPOnly := tech == TechCassandra || tech == TechRedis
		wantRefused := class.Consistent() && storeIsAPOnly
		if wantRefused && err == nil {
			t.Fatalf("iteration %d: %s over %s must be refused", i, class, tech)
		}
		if !wantRefused && err != nil {
			t.Fatalf("iteration %d: %s over %s must pass, got %v", i, class, tech, err)
		}
	}
}
