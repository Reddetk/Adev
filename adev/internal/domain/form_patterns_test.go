package domain

import (
	"math/rand"
	"strings"
	"testing"
)

// D7/D8: the distributed-data tree and the deployment pattern. A branch of the
// tree fires only when the design holds its premise; a decision that names a
// pattern without that premise is a solution to a problem the system does not
// have. The deployment checklist is written by the system from the pattern.

// patternDesign builds a microservices design with three code containers in
// three contexts, a store and a message bus. Every gate of D1/D4/D6 is satisfied;
// a test adds only the relations it wants to judge.
func patternDesign() HLD {
	contexts := []BoundedContext{
		{ID: "bc-a", Name: "Enrolment", Purpose: Text{Value: "Enrol a student."}, PACELC: PACELCPaEl, ContainerIds: []string{"c-a"}},
		{ID: "bc-b", Name: "Progress", Purpose: Text{Value: "Track progress."}, PACELC: PACELCPaEl, ContainerIds: []string{"c-b", "c-bus"}},
		{ID: "bc-c", Name: "Payment", Purpose: Text{Value: "Take a payment."}, PACELC: PACELCPaEl, ContainerIds: []string{"c-c", "c-store"}},
	}
	containers := []Container{
		{ObjectID: "c-a", BoundedContextID: "bc-a", Tech: TechGo, Purpose: Text{Value: "Serve the enrolment."}},
		{ObjectID: "c-b", BoundedContextID: "bc-b", Tech: TechGo, Purpose: Text{Value: "Serve the progress."}},
		{ObjectID: "c-c", BoundedContextID: "bc-c", Tech: TechGo, Purpose: Text{Value: "Serve the payment."}},
		{ObjectID: "c-store", BoundedContextID: "bc-c", Tech: TechPostgres, Purpose: Text{Value: "Keep the records."}},
		{ObjectID: "c-bus", BoundedContextID: "bc-b", Tech: TechKafka, Purpose: Text{Value: "Deliver the events."}},
	}
	aggregates := []Aggregate{
		{ID: "a-a", ContainerID: "c-a", Root: "Enrolment", ModuleRef: "m-a", Invariants: []Statement{{Text: "A student enrols once.", Modality: "shall", Priority: "must"}}},
		{ID: "a-b", ContainerID: "c-b", Root: "Progress", ModuleRef: "m-b", Invariants: []Statement{{Text: "Progress moves forward.", Modality: "shall", Priority: "must"}}},
		{ID: "a-c", ContainerID: "c-c", Root: "Payment", ModuleRef: "m-c", Invariants: []Statement{{Text: "A payment has one amount.", Modality: "shall", Priority: "must"}}},
	}
	decisions := []Decision{
		patternDecision("d-architecture", AreaArchitecture, "", ""),
		patternDecision("d-storage", AreaStorage, "", ""),
		{ID: "d-deployment", Area: AreaOps, Deployment: DeploymentServicePerContainer, Title: Text{Value: "Deployment"}, Decision: Text{Value: "One container per service."}, Alternatives: []Alternative{{Option: Text{Value: "One host"}, RejectedBecause: Text{Value: "The services deploy apart."}}}},
	}
	decisions[0].Style = StyleMicroservices
	decisions[1].Ownership = OwnershipPerService
	decisions[1].Techs = []string{TechPostgres}
	decisions[1].ObjectRefs = []string{"c-store"}
	for _, context := range contexts {
		decisions = append(decisions, patternDecision("d-consistency-"+context.ID, AreaConsistency, context.ID, ""))
	}
	return HLD{
		BoundedContexts: contexts,
		Containers:      containers,
		Aggregates:      aggregates,
		Decisions:       decisions,
		DegradationStrategy: []Degradation{{
			Failure: "The database is down", Degradation: "The last status stays visible",
			Detection: "A health check fails", TimeoutMs: 1000,
		}},
	}
}

func patternDecision(id string, area DecisionArea, contextRef string, pattern DistributedPattern) Decision {
	return Decision{
		ID: id, Area: area, ContextRef: contextRef, Pattern: pattern,
		Title:    Text{Value: string(area)},
		Decision: Text{Value: "The design takes this option."},
		Alternatives: []Alternative{{
			Option:          Text{Value: "The other option"},
			RejectedBecause: Text{Value: "The premise decides."},
		}},
	}
}

// withPatterns adds D7 decisions by pattern, one per branch under test.
func withPatterns(h HLD, patterns ...DistributedPattern) HLD {
	for _, pattern := range patterns {
		h.Decisions = append(h.Decisions, patternDecision("d-"+string(pattern), AreaReliability, "", pattern))
	}
	return h
}

// TestOutboxFollowsAnAsyncWrite: an event leaves a container that writes state, so
// the state change and the message must commit together.
func TestOutboxFollowsAnAsyncWrite(t *testing.T) {
	hld := patternDesign()
	hld.ContextRelations = []ContextRelation{{
		From: "c-a", To: "c-b", Type: RelationAsync, Guarantee: DeliveryAtMostOnce,
	}}
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "outbox") {
		t.Fatalf("an async write without an outbox decision must fail, got %v", err)
	}
	if err := ValidateHLD(withPatterns(hld, PatternTransactionalOutbox)); err != nil {
		t.Fatalf("the outbox pattern must satisfy the branch, got %v", err)
	}
}

// TestOutboxSkipsAPublisherThatWritesNoState: a bus or a gateway holds no
// aggregate, so an event it forwards does not need an outbox of its own.
func TestOutboxSkipsAPublisherThatWritesNoState(t *testing.T) {
	hld := patternDesign()
	hld.ContextRelations = []ContextRelation{{
		From: "c-bus", To: "c-b", Type: RelationAsync, Guarantee: DeliveryAtMostOnce,
	}}
	err := ValidateHLD(hld)
	if err != nil && strings.Contains(err.Error(), "outbox") {
		t.Fatalf("a producer without an aggregate needs no outbox, got %v", err)
	}
}

// TestDLQFollowsAtLeastOnce: a message that can be redelivered can also fail
// forever; the design names where it parks.
func TestDLQFollowsAtLeastOnce(t *testing.T) {
	hld := patternDesign()
	hld.ContextRelations = []ContextRelation{{
		From: "c-a", To: "c-b", Type: RelationAsync, Guarantee: DeliveryAtLeastOnce,
	}}
	if err := ValidateHLD(withPatterns(hld, PatternTransactionalOutbox)); err == nil || !strings.Contains(err.Error(), "dlq") {
		t.Fatalf("at-least-once without a DLQ decision must fail, got %v", err)
	}
	if err := ValidateHLD(withPatterns(hld, PatternTransactionalOutbox, PatternDLQ)); err != nil {
		t.Fatalf("outbox plus DLQ must satisfy the branches, got %v", err)
	}

	atMost := patternDesign()
	atMost.ContextRelations = []ContextRelation{{
		From: "c-a", To: "c-b", Type: RelationAsync, Guarantee: DeliveryAtMostOnce,
	}}
	if err := ValidateHLD(withPatterns(atMost, PatternTransactionalOutbox)); err != nil {
		t.Fatalf("at-most-once needs no DLQ, got %v", err)
	}
}

// TestBackpressureFollowsABus: a bus buffers, and a producer can outrun a
// consumer; the design names the flow-control rule.
func TestBackpressureFollowsABus(t *testing.T) {
	hld := patternDesign()
	hld.ContextRelations = []ContextRelation{{
		From: "c-a", To: "c-bus", Type: RelationAsync, Guarantee: DeliveryAtMostOnce,
	}}
	if err := ValidateHLD(withPatterns(hld, PatternTransactionalOutbox)); err == nil || !strings.Contains(err.Error(), "backpressure") {
		t.Fatalf("a bus with an async relation without backpressure must fail, got %v", err)
	}
	if err := ValidateHLD(withPatterns(hld, PatternTransactionalOutbox, PatternBoundedQueue)); err != nil {
		t.Fatalf("bounded queue must satisfy backpressure, got %v", err)
	}
}

// TestSagaFollowsACrossServiceChain: one operation walks three services; a
// failure in the third leaves the first two with no compensation.
func TestSagaFollowsACrossServiceChain(t *testing.T) {
	hld := patternDesign()
	hld.ContextRelations = []ContextRelation{
		{From: "c-a", To: "c-b", Type: RelationSync},
		{From: "c-b", To: "c-c", Type: RelationSync},
	}
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "saga") {
		t.Fatalf("a cross-context chain must ask for a saga, got %v", err)
	}
	if err := ValidateHLD(withPatterns(hld, PatternSagaOrchestration)); err != nil {
		t.Fatalf("saga orchestration must satisfy the branch, got %v", err)
	}

	same := patternDesign()
	same.ContextRelations = []ContextRelation{
		{From: "c-a", To: "c-a", Type: RelationSync},
	}
	if err := ValidateHLD(same); err != nil {
		t.Fatalf("a chain inside one container needs no saga, got %v", err)
	}
}

// TestPatternIsAClosedList: a mechanism outside the palette cannot be scored.
func TestPatternIsAClosedList(t *testing.T) {
	hld := patternDesign()
	hld.Decisions = append(hld.Decisions, patternDecision("d-magic", AreaReliability, "", DistributedPattern("magic")))
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "pattern") {
		t.Fatalf("an unknown pattern must fail, got %v", err)
	}
}

// TestPatternWithoutItsPremiseIsRefused: a saga in a system with no cross-service
// chain is a solution to a problem the system does not have.
func TestPatternWithoutItsPremiseIsRefused(t *testing.T) {
	hld := withPatterns(patternDesign(), PatternSagaOrchestration)
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "saga") {
		t.Fatalf("a pattern without its premise must fail, got %v", err)
	}
}

// TestD7BranchesSurviveARandomDesign: the gate agrees with an independent oracle on
// random relations. The seed is fixed so a failure reproduces.
func TestD7BranchesSurviveARandomDesign(t *testing.T) {
	random := rand.New(rand.NewSource(71))
	nodes := []string{"c-a", "c-b", "c-c", "c-bus"}
	types := []ContextRelationType{RelationSync, RelationAsync}
	guarantees := []Delivery{DeliveryAtMostOnce, DeliveryAtLeastOnce}

	for i := 0; i < 200; i++ {
		hld := patternDesign()
		hld.ContextRelations = nil
		for edge := 0; edge < 1+random.Intn(4); edge++ {
			from := nodes[random.Intn(len(nodes))]
			to := nodes[random.Intn(len(nodes))]
			if from == to {
				continue
			}
			relation := ContextRelation{From: from, To: to, Type: types[random.Intn(len(types))]}
			if relation.Type == RelationAsync {
				relation.Guarantee = guarantees[random.Intn(len(guarantees))]
			}
			hld.ContextRelations = append(hld.ContextRelations, relation)
		}

		want := map[DistributedPattern]bool{}
		for _, relation := range hld.ContextRelations {
			if relation.Type != RelationAsync {
				continue
			}
			if relation.From != "c-bus" {
				want[PatternTransactionalOutbox] = true
			}
			if relation.Guarantee == DeliveryAtLeastOnce {
				want[PatternDLQ] = true
			}
			if relation.From == "c-bus" || relation.To == "c-bus" {
				want[PatternBoundedQueue] = true
			}
		}
		if hasAnyChain(hld.ContextRelations) {
			want[PatternSagaOrchestration] = true
		}

		answered := make([]DistributedPattern, 0, len(want))
		for pattern := range want {
			answered = append(answered, pattern)
		}
		// Drop one branch in every second iteration: the gate must notice.
		if len(answered) > 0 && i%2 == 1 {
			answered = answered[:len(answered)-1]
		}
		for _, pattern := range answered {
			hld.Decisions = append(hld.Decisions, patternDecision("d-"+string(pattern), AreaReliability, "", pattern))
		}
		err := ValidateHLD(hld)
		wantRefused := len(answered) != len(want)
		if wantRefused && err == nil {
			t.Fatalf("iteration %d: branches %v, answered %v must fail; relations %+v", i, keysOf(want), answered, hld.ContextRelations)
		}
		if !wantRefused && err != nil {
			t.Fatalf("iteration %d: all branches answered but gate refused: %v; relations %+v", i, err, hld.ContextRelations)
		}
	}
}

// hasAnyChain is the oracle: three distinct nodes in a row, read from the
// relation list, with at least two bounded contexts among them.
func hasAnyChain(relations []ContextRelation) bool {
	nodes := []string{"c-a", "c-b", "c-c", "c-bus"}
	contexts := map[string]string{"c-a": "bc-a", "c-b": "bc-b", "c-c": "bc-c", "c-bus": "bc-b"}
	edge := func(from, to string) bool {
		for _, relation := range relations {
			if relation.From == from && relation.To == to {
				return true
			}
		}
		return false
	}
	for _, a := range nodes {
		for _, b := range nodes {
			for _, c := range nodes {
				if a == b || b == c || a == c {
					continue
				}
				if !edge(a, b) || !edge(b, c) {
					continue
				}
				if len(map[string]bool{contexts[a]: true, contexts[b]: true, contexts[c]: true}) > 1 {
					return true
				}
			}
		}
	}
	return false
}

func keysOf(patterns map[DistributedPattern]bool) []DistributedPattern {
	out := make([]DistributedPattern, 0, len(patterns))
	for pattern := range patterns {
		out = append(out, pattern)
	}
	return out
}

// TestDeploymentPatternIsRequired: a design names where its containers run, and
// the pattern is one of the catalogue's deployment patterns.
func TestDeploymentPatternIsRequired(t *testing.T) {
	hld := patternDesign()
	hld.Decisions = withoutArea(hld.Decisions, AreaOps)
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "deployment") {
		t.Fatalf("a design without a deployment pattern must fail, got %v", err)
	}
	if err := ValidateHLD(patternDesign()); err != nil {
		t.Fatalf("service per container must pass, got %v", err)
	}
}

// TestDeploymentPatternIsAClosedList: an invented pattern is not a deployment.
func TestDeploymentPatternIsAClosedList(t *testing.T) {
	hld := patternDesign()
	for i := range hld.Decisions {
		if hld.Decisions[i].Deployment != "" {
			hld.Decisions[i].Deployment = DeploymentPattern("magic")
		}
	}
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "deployment") {
		t.Fatalf("an unknown deployment pattern must fail, got %v", err)
	}
}

// TestServerlessDeploymentMatchesTheStyle: a serverless style that runs on VMs,
// or a microservice style that claims serverless, is a contradiction.
func TestServerlessDeploymentMatchesTheStyle(t *testing.T) {
	hld := patternDesign()
	for i := range hld.Decisions {
		if hld.Decisions[i].Deployment != "" {
			hld.Decisions[i].Deployment = DeploymentServerless
		}
	}
	if err := ValidateHLD(hld); err == nil || !strings.Contains(err.Error(), "serverless") {
		t.Fatalf("serverless deployment under microservices must fail, got %v", err)
	}

	serverless := patternDesign()
	for i := range serverless.Decisions {
		if serverless.Decisions[i].Area == AreaArchitecture {
			serverless.Decisions[i].Style = StyleServerless
		}
		if serverless.Decisions[i].Deployment != "" {
			serverless.Decisions[i].Deployment = DeploymentServerless
		}
	}
	if err := ValidateHLD(serverless); err != nil {
		t.Fatalf("serverless style over serverless deployment must pass, got %v", err)
	}
}

// TestDeploymentChecklist: every pattern carries the steps it makes mandatory, so
// a later stage reads one source and not a prose reminder.
func TestDeploymentChecklist(t *testing.T) {
	cases := []struct {
		pattern DeploymentPattern
		want    string
	}{
		{DeploymentMultipleServicesPerHost, "port"},
		{DeploymentServicePerHost, "host"},
		{DeploymentServicePerVM, "systemd"},
		{DeploymentServicePerContainer, "probe"},
		{DeploymentServerless, "cold start"},
		{DeploymentServicePlatform, "disruption budget"},
	}
	for _, tc := range cases {
		checklist := DeploymentChecklist(tc.pattern)
		if len(checklist) < 3 {
			t.Fatalf("%s must carry a checklist, got %v", tc.pattern, checklist)
		}
		joined := strings.ToLower(strings.Join(checklist, " | "))
		if !strings.Contains(joined, tc.want) {
			t.Fatalf("%s checklist must mention %q, got %v", tc.pattern, tc.want, checklist)
		}
	}
}

// TestPalettesAreClosed: the prompt, the gate and the classifier read one list.
func TestPalettesAreClosed(t *testing.T) {
	for _, pattern := range DistributedPatterns() {
		if !pattern.Valid() {
			t.Fatalf("%q is listed but not valid", pattern)
		}
		if _, ok := patternBranches[pattern]; !ok {
			t.Fatalf("%q has no branch", pattern)
		}
	}
	if len(DistributedPatterns()) != len(patternBranches) {
		t.Fatalf("the palette lists %d patterns, %d have a branch", len(DistributedPatterns()), len(patternBranches))
	}
	for _, pattern := range DeploymentPatterns() {
		if !pattern.Valid() {
			t.Fatalf("%q is listed but not valid", pattern)
		}
		if len(DeploymentChecklist(pattern)) == 0 {
			t.Fatalf("%q has no checklist", pattern)
		}
	}
}
