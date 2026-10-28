package domain

// Dep is one allowed dependency between two modules.
type Dep struct {
	ProviderObjectID string `json:"providerObjectId"`
	Direction        string `json:"direction"`
	Type             string `json:"type"` // sync | async
}

// Module is one behavior holder inside a package.
type Module struct {
	ObjectID string `json:"objectId"`
	Path     string `json:"path"`
	Deps     []Dep  `json:"deps,omitempty"`
}

// Package is one code package inside a layer.
type Package struct {
	Path    string   `json:"path"`
	Modules []Module `json:"modules"`
}

// LayerName is one architecture layer of a container.
type LayerName string

const (
	LayerAdapter LayerName = "adapter"
	LayerDomain  LayerName = "domain"
	LayerApp     LayerName = "app"
	LayerInfra   LayerName = "infra"
)

// Layer is one architecture layer of a container.
type Layer struct {
	Name     LayerName `json:"name"`
	Packages []Package `json:"packages"`
}

// LLDContainer is the physical structure inside one container.
type LLDContainer struct {
	ContainerID string  `json:"containerId"`
	Layers      []Layer `json:"layers"`
}

// ContractError is one error that a contract can return.
type ContractError struct {
	Code       string `json:"code"`
	HTTPStatus *int   `json:"httpStatus,omitempty"`
	Condition  string `json:"condition"`
	Retryable  bool   `json:"retryable"`
}

// Contract is one interface that a module exposes.
type Contract struct {
	ID         string          `json:"id"`
	ModuleRef  Ref             `json:"moduleRef"`
	Kind       string          `json:"kind"`
	InputType  string          `json:"inputType"`
	OutputType string          `json:"outputType"`
	Errors     []ContractError `json:"errors,omitempty"`
}

// PathKind tells a successful call from a failed one.
type PathKind string

const (
	PathHappy    PathKind = "happy"
	PathNegative PathKind = "negative"
)

// Path is one call path through one contract.
type Path struct {
	ID                string   `json:"id"`
	ContractRef       string   `json:"contractRef"`
	Kind              PathKind `json:"kind"`
	Given             string   `json:"given"`
	When              string   `json:"when"`
	Then              string   `json:"then"`
	ExpectedErrorCode string   `json:"expectedErrorCode,omitempty"`
}

// EventMap is one event and its transport.
type EventMap struct {
	Event             string   `json:"event"`
	ProducerObjectID  string   `json:"producerObjectId"`
	ConsumerObjectIds []string `json:"consumerObjectIds"`
	PayloadRef        string   `json:"payloadRef,omitempty"`
	IdempotencyKey    string   `json:"idempotencyKey"`
	Ordering          string   `json:"ordering"`
}

// LLD is the low-level design of the target system.
type LLD struct {
	Containers []LLDContainer `json:"containers"`
	Contracts  []Contract     `json:"contracts"`
	Paths      []Path         `json:"paths"`
	EventMaps  []EventMap     `json:"eventMaps,omitempty"`
	AdrRefs    []Ref          `json:"adrRefs,omitempty"`
	OpenGaps   []Gap          `json:"openGaps,omitempty"`
}

// Texts returns the free-text values for the language checker.
func (l LLD) Texts() []Text { return nil }
