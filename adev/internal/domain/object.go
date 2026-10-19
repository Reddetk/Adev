package domain

// Kind is the physical kind of an object, taken from targetObjects.md.
type Kind string

const (
	KindContainer Kind = "container"
	KindLayer     Kind = "layer"
	KindPackage   Kind = "package"
	KindModule    Kind = "module"
	KindTable     Kind = "table"
	KindBroker    Kind = "broker"
	KindTest      Kind = "test"
	KindConfig    Kind = "config"
	KindPipeline  Kind = "pipeline"
	KindEnv       Kind = "env"
	KindDoc       Kind = "doc"
)

var knownKinds = map[Kind]bool{
	KindContainer: true, KindLayer: true, KindPackage: true, KindModule: true,
	KindTable: true, KindBroker: true, KindTest: true, KindConfig: true,
	KindPipeline: true, KindEnv: true, KindDoc: true,
}

// Valid reports whether the kind is known.
func (k Kind) Valid() bool { return knownKinds[k] }

// ObjectStatus is the lifecycle state of a registry object.
type ObjectStatus string

const (
	ObjectPlanned ObjectStatus = "planned"
	ObjectActive  ObjectStatus = "active"
	ObjectRemoved ObjectStatus = "removed"
)

// Valid reports whether the object status is known.
func (s ObjectStatus) Valid() bool {
	return s == ObjectPlanned || s == ObjectActive || s == ObjectRemoved
}

// Object is one entry of the target system registry.
type Object struct {
	ObjectID       string         `json:"objectId"`
	TargetSystemID int64          `json:"targetSystemId"`
	Kind           Kind           `json:"kind"`
	Name           string         `json:"name"`
	Path           string         `json:"path,omitempty"`
	ContainerID    string         `json:"containerId,omitempty"`
	Status         ObjectStatus   `json:"status"`
	Fields         map[string]any `json:"fields,omitempty"`
	IntroducedBy   Ref            `json:"introducedBy"`
	ChangedBy      []Ref          `json:"changedBy"`
}

var allowedTransitions = map[ObjectStatus]map[ObjectStatus]bool{
	ObjectPlanned: {ObjectActive: true, ObjectRemoved: true},
	ObjectActive:  {ObjectRemoved: true},
	ObjectRemoved: {},
}

// CanChange reports whether the object lifecycle allows the status change.
func CanChange(from, to ObjectStatus) bool { return allowedTransitions[from][to] }
