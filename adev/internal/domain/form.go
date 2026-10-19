// Form contract of adev. Every form uses the envelope below.
package domain

import (
	"encoding/json"
	"fmt"
)

// FormType is one file in the form registry.
type FormType string

const (
	TypeUserInput          FormType = "USER_INPUT"
	TypeBRD                FormType = "BRD"
	TypeFeasibility        FormType = "FEASIBILITY"
	TypeCRBrief            FormType = "CR_BRIEF"
	TypeImpactAssessment   FormType = "IMPACT_ASSESSMENT"
	TypeSRS                FormType = "SRS"
	TypeHLD                FormType = "HLD"
	TypeADR                FormType = "ADR"
	TypeLLD                FormType = "LLD"
	TypeTaskPlan           FormType = "TASK_PLAN"
	TypeTestContract       FormType = "TEST_CONTRACT"
	TypeImplementationSpec FormType = "IMPLEMENTATION_SPEC"
	TypeVerdict            FormType = "VERDICT"
	TypeRefactorSpec       FormType = "REFACTOR_SPEC"
	TypeQuestion           FormType = "QUESTION"
)

// Status is the lifecycle state of a form.
type Status string

const (
	StatusDraft           Status = "draft"
	StatusValid           Status = "valid"
	StatusWaitingOperator Status = "waiting_operator"
	StatusApproved        Status = "approved"
	StatusRejected        Status = "rejected"
	StatusSuperseded      Status = "superseded"
)

// Route is the pipeline route.
type Route string

const (
	RouteGreenfield Route = "greenfield"
	RouteChange     Route = "change"
)

// Mode marks a full form or a JSON Patch delta against a base form.
type Mode string

const (
	ModeFull  Mode = "full"
	ModeDelta Mode = "delta"
)

// Ref points at a form, optionally at one node inside it.
type Ref struct {
	FormID      string   `json:"formId"`
	FormType    FormType `json:"formType"`
	FormVersion int      `json:"formVersion"`
	Pointer     string   `json:"pointer,omitempty"`
	Hash        string   `json:"hash,omitempty"`
}

// Envelope carries the fields that every form shares.
type Envelope struct {
	FormID              string          `json:"formId"`
	FormType            FormType        `json:"formType"`
	SchemaVersion       int             `json:"schemaVersion"`
	FormVersion         int             `json:"formVersion"`
	PreviousFormVersion *Ref            `json:"previousFormVersion,omitempty"`
	Status              Status          `json:"status"`
	Route               Route           `json:"route"`
	Stage               string          `json:"stage"`
	Mode                Mode            `json:"mode"`
	Base                *Ref            `json:"base,omitempty"`
	Ops                 json.RawMessage `json:"ops,omitempty"`
	PipelineID          string          `json:"pipelineId"`
	TargetSystemID      string          `json:"targetSystemId"`
	ProducedBy          Role            `json:"producedBy"`
	Inputs              []Ref           `json:"inputs"`
	PayloadHash         string          `json:"payloadHash"`
}

// Form is the envelope plus the typed payload of one form type.
type Form struct {
	Envelope
	Payload json.RawMessage `json:"payload"`
}

// SetPayload stores a typed payload and computes its hash.
func (f *Form) SetPayload(v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	f.Payload = raw
	hash, err := HashPayload(raw)
	if err != nil {
		return err
	}
	f.PayloadHash = hash
	return nil
}

// BindPayload decodes the payload into a typed value.
func (f *Form) BindPayload(v any) error {
	if len(f.Payload) == 0 {
		return fmt.Errorf("payload is empty")
	}
	if err := json.Unmarshal(f.Payload, v); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	return nil
}

// SetRawPayload stores an already encoded payload and computes its hash.
// A form that arrives over the wire is not decoded into a typed value first.
func (f *Form) SetRawPayload(raw json.RawMessage) error {
	if len(raw) == 0 {
		return fmt.Errorf("payload is empty")
	}
	hash, err := HashPayload(raw)
	if err != nil {
		return err
	}
	f.Payload = raw
	f.PayloadHash = hash
	return nil
}

// ComputeHash returns the payload hash of the current payload.
func (f *Form) ComputeHash() (string, error) {
	return HashPayload(f.Payload)
}
