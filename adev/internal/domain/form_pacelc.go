package domain

import (
	"fmt"
	"strings"
)

// PACELC is the consistency class of one bounded context: what it does inside a
// network partition (P) and in the normal case (E), choosing between latency (L)
// and consistency (C). The class is decided before the store and the relations,
// because it rules both of them out.
type PACELC string

const (
	// PACELCPaEl answers during a partition and is fast when nothing fails.
	PACELCPaEl PACELC = "PA/EL"
	// PACELCPcEc waits during a partition and pays latency for consistency.
	PACELCPcEc PACELC = "PC/EC"
	// PACELCPaEc answers during a partition and pays latency when nothing fails.
	PACELCPaEc PACELC = "PA/EC"
	// PACELCPcEl waits during a partition and is fast when nothing fails.
	PACELCPcEl PACELC = "PC/EL"
)

// Valid reports whether the class is one of the four the theory allows.
func (p PACELC) Valid() bool {
	switch p {
	case PACELCPaEl, PACELCPcEc, PACELCPaEc, PACELCPcEl:
		return true
	}
	return false
}

// Consistent reports a class that must not answer with stale data.
func (p PACELC) Consistent() bool {
	return p == PACELCPcEc || p == PACELCPcEl
}

// Tech is one runtime or product the design may choose. The list is closed: a
// product outside it has no path layout, no consistency capability and no gate.
const (
	TechGo        = "go"
	TechJava      = "java"
	TechNode      = "node"
	TechPython    = "python"
	TechReact     = "react"
	TechAngular   = "angular"
	TechVue       = "vue"
	TechPostgres  = "postgresql"
	TechMySQL     = "mysql"
	TechMongo     = "mongodb"
	TechCassandra = "cassandra"
	TechRedis     = "redis"
	TechKafka     = "kafka"
	TechRabbit    = "rabbitmq"
	TechNats      = "nats"
	TechNginx     = "nginx"
	TechEnvoy     = "envoy"
	TechKong      = "kong"
	TechKeycloak  = "keycloak"
	TechElastic   = "elasticsearch"
	TechMinio     = "minio"
	TechK8s       = "kubernetes"
	TechExternal  = "external"
)

// knownTechs is the closed list of the tech field.
var knownTechs = map[string]bool{
	TechGo: true, TechJava: true, TechNode: true, TechPython: true,
	TechReact: true, TechAngular: true, TechVue: true,
	TechPostgres: true, TechMySQL: true, TechMongo: true, TechCassandra: true, TechRedis: true,
	TechKafka: true, TechRabbit: true, TechNats: true,
	TechNginx: true, TechEnvoy: true, TechKong: true,
	TechKeycloak: true, TechElastic: true, TechMinio: true,
	TechK8s: true, TechExternal: true,
}

// KnownTechs lists the closed set for the prompt and the UI.
func KnownTechs() []string {
	return []string{
		TechGo, TechJava, TechNode, TechPython, TechReact, TechAngular, TechVue,
		TechPostgres, TechMySQL, TechMongo, TechCassandra, TechRedis,
		TechKafka, TechRabbit, TechNats, TechNginx, TechEnvoy, TechKong,
		TechKeycloak, TechElastic, TechMinio, TechK8s, TechExternal,
	}
}

// storeTechs are the products that hold the system of record. A cache or a bus is
// not a store: it can lose its content.
var storeTechs = map[string]bool{
	TechPostgres: true, TechMySQL: true, TechMongo: true, TechCassandra: true, TechRedis: true,
}

// techConsistency is what one store can promise inside a partition: strong waits,
// eventual answers and reconciles, either depends on the write concern.
var techConsistency = map[string]string{
	TechPostgres:  "strong",
	TechMySQL:     "strong",
	TechMongo:     "either",
	TechCassandra: "eventual",
	TechRedis:     "eventual",
}

// validatePACELC checks the class of every context and the two things that follow
// from it: the store of the context can keep the promise, and a consistent context
// is not wired only by asynchronous edges.
func validatePACELC(h HLD) []error {
	var errs []error
	containersOf := map[string][]Container{}
	for _, container := range h.Containers {
		containersOf[container.BoundedContextID] = append(containersOf[container.BoundedContextID], container)
	}
	classOf := map[string]PACELC{}
	for i, context := range h.BoundedContexts {
		field := fmt.Sprintf("boundedContexts[%d]", i)
		if !context.PACELC.Valid() {
			errs = append(errs, invalid(field+".pacelc", fmt.Sprintf("unknown value %q; name one of PA/EL, PC/EC, PA/EC, PC/EL", context.PACELC)))
			continue
		}
		classOf[context.ID] = context.PACELC
		if !context.PACELC.Consistent() {
			continue
		}
		// The store must be able to keep the promise. A context with no store
		// container stores nothing of its own and passes.
		stores := 0
		keeps := false
		for _, container := range containersOf[context.ID] {
			if !storeTechs[container.Tech] {
				continue
			}
			stores++
			if capability := techConsistency[container.Tech]; capability == "strong" || capability == "either" {
				keeps = true
			}
		}
		if stores > 0 && !keeps {
			errs = append(errs, invalid(field+".pacelc",
				fmt.Sprintf("context %q promises %s but every store it has is available-only; add a consistent store or change the class", context.ID, strings.ToLower(string(context.PACELC)))))
		}
	}
	// A consistent context cannot be wired only by async edges: nothing then waits
	// for its answer, and its promise reaches no caller.
	// shortcut: a consistent context may legitimately be fed only by events; revisit
	// when a relation carries an idempotency key and the check can be exact.
	syncEdges := map[string]bool{}
	edges := map[string]bool{}
	for _, relation := range h.ContextRelations {
		for _, contextID := range []string{relation.From, relation.To} {
			edges[contextID] = true
			if relation.Type == RelationSync {
				syncEdges[contextID] = true
			}
		}
	}
	for i, context := range h.BoundedContexts {
		if !context.PACELC.Consistent() || !edges[context.ID] || syncEdges[context.ID] {
			continue
		}
		errs = append(errs, invalid(fmt.Sprintf("boundedContexts[%d].pacelc", i),
			fmt.Sprintf("context %q promises %s and is wired only by async edges; a consistent context answers a caller synchronously", context.ID, strings.ToLower(string(context.PACELC)))))
	}
	return errs
}
