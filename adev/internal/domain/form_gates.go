package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// PayloadTexts returns every free-text value of a form.
// The use case layer runs the language checker on them.
func (f *Form) PayloadTexts() ([]Text, error) {
	switch f.FormType {
	case TypeUserInput:
		var in UserInput
		if err := f.BindPayload(&in); err != nil {
			return nil, err
		}
		return []Text{in.RawRequest}, nil
	case TypeBRD:
		var brd BRD
		if err := f.BindPayload(&brd); err != nil {
			return nil, err
		}
		return brd.Texts(), nil
	case TypeFeasibility:
		var feasibility Feasibility
		if err := f.BindPayload(&feasibility); err != nil {
			return nil, err
		}
		return feasibility.Texts(), nil
	case TypeSRS:
		var srs SRS
		if err := f.BindPayload(&srs); err != nil {
			return nil, err
		}
		return srs.Texts(), nil
	case TypeHLD:
		var hld HLD
		if err := f.BindPayload(&hld); err != nil {
			return nil, err
		}
		return hld.Texts(), nil
	case TypeADR:
		var adr ADR
		if err := f.BindPayload(&adr); err != nil {
			return nil, err
		}
		return adr.Texts(), nil
	case TypeLLD:
		var lld LLD
		if err := f.BindPayload(&lld); err != nil {
			return nil, err
		}
		return lld.Texts(), nil
	case TypeTaskPlan:
		var taskPlan TaskPlan
		if err := f.BindPayload(&taskPlan); err != nil {
			return nil, err
		}
		return taskPlan.Texts(), nil
	default:
		return nil, nil
	}
}

// ValidatePayload applies the gates of the payload type of the form.
func (f *Form) ValidatePayload() error {
	switch f.FormType {
	case TypeUserInput:
		var in UserInput
		if err := f.BindPayload(&in); err != nil {
			return payloadMismatch(f.FormType, err)
		}
		return ValidateUserInput(in)
	case TypeBRD:
		var brd BRD
		if err := f.BindPayload(&brd); err != nil {
			return payloadMismatch(f.FormType, err)
		}
		return ValidateBRD(brd, f.Status)
	case TypeFeasibility:
		var feasibility Feasibility
		if err := f.BindPayload(&feasibility); err != nil {
			return payloadMismatch(f.FormType, err)
		}
		return ValidateFeasibility(feasibility)
	case TypeSRS:
		var srs SRS
		if err := f.BindPayload(&srs); err != nil {
			return payloadMismatch(f.FormType, err)
		}
		return ValidateSRS(srs)
	case TypeHLD:
		var hld HLD
		if err := f.BindPayload(&hld); err != nil {
			return payloadMismatch(f.FormType, err)
		}
		return ValidateHLD(hld)
	case TypeADR:
		var adr ADR
		if err := f.BindPayload(&adr); err != nil {
			return payloadMismatch(f.FormType, err)
		}
		return ValidateADR(adr)
	case TypeLLD:
		var lld LLD
		if err := f.BindPayload(&lld); err != nil {
			return payloadMismatch(f.FormType, err)
		}
		return ValidateLLD(lld)
	case TypeTaskPlan:
		var taskPlan TaskPlan
		if err := f.BindPayload(&taskPlan); err != nil {
			return payloadMismatch(f.FormType, err)
		}
		return ValidateTaskPlan(taskPlan)
	default:
		return fmt.Errorf("%w: %q", ErrNoPayloadGates, f.FormType)
	}
}

// ValidatePayloadWithStatus is the accepted-state gate of the question phase:
// the form is accepted only when every operator question is closed. A blocking
// gap refuses it by definition; a gap with a question refuses it too, because
// the answer that fills the field also removes the gap entry.
func (f *Form) ValidatePayloadWithStatus() error {
	err := f.ValidatePayload()
	if err != nil {
		return err
	}
	var errs []error
	// A question is answered by a patch to the payload: a pointer into a section
	// the shape does not have leaves the operator nowhere to write the answer.
	for i, position := range f.Gaps() {
		if position.Gap.Question.Value == "" {
			continue
		}
		if !patchable(f.Payload, position.Gap.Pointer) {
			errs = append(errs, invalid(fmt.Sprintf("openGaps[%d].pointer", i),
				fmt.Sprintf("%s has no parent field in the payload; point the gap at a field the shape has, or fill the value yourself", position.Gap.Pointer)))
		}
	}
	if f.Status == StatusValid || f.Status == StatusApproved {
		for i, position := range f.Gaps() {
			if position.Gap.Blocking || position.Gap.Question.Value != "" {
				errs = append(errs, invalid(fmt.Sprintf("openGaps[%d]", i), "the operator question is not answered yet"))
			}
		}
	}
	return errors.Join(errs...)
}

// patchable reports whether an answer can land on the pointer. The field itself
// may be absent while its parent object or array exists: RFC 6902 add fills the
// slot, and naming the index right past an array appends.
func patchable(payload json.RawMessage, pointer string) bool {
	if pointer == "" || pointer == "/payload" {
		return true
	}
	if !strings.HasPrefix(pointer, "/") {
		return false
	}
	var doc any
	if err := json.Unmarshal(payload, &doc); err != nil {
		return false
	}
	segments := strings.Split(pointer[1:], "/")
	current := doc
	for _, raw := range segments[:len(segments)-1] {
		segment := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		switch node := current.(type) {
		case map[string]any:
			current = node[segment]
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(node) {
				return false
			}
			current = node[index]
		default:
			return false
		}
		if current == nil {
			return false
		}
	}
	switch current.(type) {
	case map[string]any, []any:
		return true
	}
	return false
}

// payloadMismatch reports a payload that does not fit its form type as a validation
// failure. The agent that wrote it can repair it, and the operator sees the reason
// instead of an internal error.
func payloadMismatch(formType FormType, err error) error {
	return invalid("payload", fmt.Sprintf("does not match the %s shape: %v", formType, err))
}
