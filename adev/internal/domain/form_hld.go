package domain

// Term is one word with a special meaning in this system.
type Term struct {
	Term       string `json:"term"`
	Definition Text   `json:"definition"`
	SourceRef  string `json:"sourceRef,omitempty"`
}

// BoundedContext is one boundary of meaning.
type BoundedContext struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Purpose      Text     `json:"purpose"`
	ContainerIds []string `json:"containerIds,omitempty"`
}

// Container is one deployable unit.
type Container struct {
	ObjectID         string   `json:"objectId"`
	Purpose          Text     `json:"purpose"`
	BoundedContextID string   `json:"boundedContextId"`
	Tech             string   `json:"tech"`
	NfrRefs          []string `json:"nfrRefs,omitempty"`
}

// Aggregate is one consistency boundary inside a container.
type Aggregate struct {
	ID          string      `json:"id"`
	ContainerID string      `json:"containerId"`
	Root        string      `json:"root"`
	EntityRefs  []string    `json:"entityRefs,omitempty"`
	EventMaps   []string    `json:"eventMaps,omitempty"`
	ModuleRef   string      `json:"moduleRef"`
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
	UbiquitousLanguage  []Term            `json:"ubiquitousLanguage,omitempty"`
	BoundedContexts     []BoundedContext  `json:"boundedContexts"`
	Containers          []Container       `json:"containers"`
	Aggregates          []Aggregate       `json:"aggregates"`
	ContextRelations    []ContextRelation `json:"contextRelations,omitempty"`
	DegradationStrategy []Degradation     `json:"degradationStrategy,omitempty"`
	OpenGaps            []Gap             `json:"openGaps,omitempty"`
}

// Texts returns every free-text value for the language checker.
func (h HLD) Texts() []Text {
	var out []Text
	for _, term := range h.UbiquitousLanguage {
		out = append(out, term.Definition)
	}
	for _, bc := range h.BoundedContexts {
		out = append(out, bc.Purpose)
	}
	for _, c := range h.Containers {
		out = append(out, c.Purpose)
	}
	for _, a := range h.Aggregates {
		for _, invariant := range a.Invariants {
			out = append(out, Text{Value: invariant.Text, Lang: LangACE})
		}
	}
	for _, d := range h.DegradationStrategy {
		out = append(out, Text{Value: d.Failure, Lang: LangSTE100}, Text{Value: d.Degradation, Lang: LangSTE100})
	}
	return out
}
