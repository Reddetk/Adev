package domain

import (
	"errors"
	"fmt"
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
	if f.Status != StatusValid && f.Status != StatusApproved {
		return nil
	}
	var errs []error
	for i, position := range f.Gaps() {
		if position.Gap.Blocking || position.Gap.Question.Value != "" {
			errs = append(errs, invalid(fmt.Sprintf("openGaps[%d]", i), "the operator question is not answered yet"))
		}
	}
	return errors.Join(errs...)
}

// payloadMismatch reports a payload that does not fit its form type as a validation
// failure. The agent that wrote it can repair it, and the operator sees the reason
// instead of an internal error.
func payloadMismatch(formType FormType, err error) error {
	return invalid("payload", fmt.Sprintf("does not match the %s shape: %v", formType, err))
}
