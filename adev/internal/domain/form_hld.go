package domain

import "strings"

// Term is one word with a special meaning in this system.
type Term struct {
	Term       string `json:"term"`
	Definition Text   `json:"definition"`
	SourceRef  string `json:"sourceRef,omitempty"`
	// BoundedContextID scopes the meaning: the same word in two contexts is two
	// terms, not one definition stretched over both.
	BoundedContextID string `json:"boundedContextId,omitempty"`
}

// HLDActor is one C4 person: a role that uses the target system. An actor is not
// a container: it has no tech, no bounded context and no aggregate.
type HLDActor struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// BrdActorRef is the BRD actors[].id this person comes from. The design does
	// not invent a role the business did not name.
	BrdActorRef string `json:"brdActorRef,omitempty"`
	Purpose     Text   `json:"purpose,omitempty"`
}

// ActorInteraction is one use of the system by an actor: the C4 context arrow
// from a person to the container that accepts the action.
type ActorInteraction struct {
	ActorID     string `json:"actorId"`
	ContainerID string `json:"containerId"`
	// Description is the business action, read on the arrow: "enrols in a
	// course", "grades submitted work". A technical verb is not a description.
	Description Text `json:"description,omitempty"`
	// CauseRef names the SRS requirement that needs this interaction.
	CauseRef string `json:"causeRef,omitempty"`
}

// BoundedContext is one boundary of meaning.
type BoundedContext struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Purpose      Text     `json:"purpose"`
	ContainerIds []string `json:"containerIds,omitempty"`
	// PACELC is the consistency class this boundary promises: decided before the
	// store and the relations, because it rules both of them out.
	PACELC PACELC `json:"pacelc"`
}

// Container is one deployable unit. An external system is the exception: it is a
// C4 external system, not a container of this system, so it carries tech
// "external", no bounded context and no aggregate.
type Container struct {
	ObjectID         string   `json:"objectId"`
	Purpose          Text     `json:"purpose"`
	BoundedContextID string   `json:"boundedContextId"`
	Tech             string   `json:"tech"`
	NfrRefs          []string `json:"nfrRefs,omitempty"`
}

// External reports a system this design does not deploy.
func (c Container) External() bool {
	return strings.EqualFold(c.Tech, TechExternal)
}

// Aggregate is one consistency boundary inside a container.
type Aggregate struct {
	ID          string      `json:"id"`
	ContainerID string      `json:"containerId"`
	Root        string      `json:"root"`
	EntityRefs  []string    `json:"entityRefs,omitempty"`
	EventMaps   []string    `json:"eventMaps,omitempty"`
	Invariants  []Statement `json:"invariants,omitempty"`
}

// ContextRelationType is how one context calls another.
type ContextRelationType string

const (
	RelationSync  ContextRelationType = "sync"
	RelationAsync ContextRelationType = "async"
)

// ContextRelation is one dependency between two contexts.
type ContextRelation struct {
	From       string              `json:"from"`
	To         string              `json:"to"`
	Type       ContextRelationType `json:"type"`
	ContractID string              `json:"contractId,omitempty"`
	// Guarantee is what an async edge promises its consumer: at most once, at
	// least once, or exactly once. A sync edge answers with its return value.
	Guarantee Delivery `json:"guarantee,omitempty"`
	// IdempotencyKey names the field that makes a repeated delivery safe; a
	// consistent consumer cannot accept an at-least-once event without it.
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
	Retry          bool   `json:"retry,omitempty"`
	DLQ            string `json:"dlq,omitempty"`
	// Description is the business action the edge carries, read on the C4 arrow:
	// "pays for a course", "publishes a grade". A technical verb is not enough.
	Description Text `json:"description,omitempty"`
	// CauseRef names the SRS requirement that needs this edge, so the relation is
	// a piece of the flow and not a guessed adjacency.
	CauseRef string `json:"causeRef,omitempty"`
}

// Degradation is one failure and the reduced behavior of the system.
type Degradation struct {
	Failure     string `json:"failure"`
	Degradation string `json:"degradation"`
	Detection   string `json:"detection"`
	TimeoutMs   int    `json:"timeoutMs"`
}

// HLD is the high-level design of the target system.
type HLD struct {
	UbiquitousLanguage []Term `json:"ubiquitousLanguage,omitempty"`
	// Actors are the people of C4 level 1. They stand outside the system and the
	// containers they reach are the only entry points of the design.
	Actors            []HLDActor         `json:"actors,omitempty"`
	ActorInteractions []ActorInteraction `json:"actorInteractions,omitempty"`
	BoundedContexts   []BoundedContext   `json:"boundedContexts"`
	Containers        []Container        `json:"containers"`
	Aggregates        []Aggregate        `json:"aggregates"`
	ContextRelations  []ContextRelation  `json:"contextRelations,omitempty"`
	// Decisions are the architecture choices this design makes: which product,
	// which consistency, which failure handling. A greenfield system has no
	// decision history to record, so the choices live in the design they shape.
	Decisions           []Decision    `json:"decisions,omitempty"`
	DegradationStrategy []Degradation `json:"degradationStrategy,omitempty"`
	OpenGaps            []Gap         `json:"openGaps,omitempty"`
}

// Texts returns every free-text value for the language checker.
func (h HLD) Texts() []Text {
	var out []Text
	for _, term := range h.UbiquitousLanguage {
		out = append(out, term.Definition)
	}
	for _, actor := range h.Actors {
		out = append(out, actor.Purpose)
	}
	for _, interaction := range h.ActorInteractions {
		out = append(out, interaction.Description)
	}
	for _, relation := range h.ContextRelations {
		out = append(out, relation.Description)
	}
	for _, bc := range h.BoundedContexts {
		out = append(out, bc.Purpose)
	}
	for _, c := range h.Containers {
		out = append(out, c.Purpose)
	}
	for _, a := range h.Aggregates {
		for _, invariant := range a.Invariants {
			out = append(out, Text{Value: invariant.Text})
		}
	}
	for _, d := range h.DegradationStrategy {
		out = append(out, Text{Value: d.Failure}, Text{Value: d.Degradation})
	}
	for _, d := range h.Decisions {
		out = append(out, d.Title, d.Context, d.Decision)
		out = append(out, d.Consequences...)
		for _, alt := range d.Alternatives {
			out = append(out, alt.Option, alt.RejectedBecause)
		}
	}
	return out
}
