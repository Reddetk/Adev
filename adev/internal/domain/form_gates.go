package domain

import "fmt"

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
	default:
		return fmt.Errorf("%w: %q", ErrNoPayloadGates, f.FormType)
	}
}

// payloadMismatch reports a payload that does not fit its form type as a validation
// failure. The agent that wrote it can repair it, and the operator sees the reason
// instead of an internal error.
func payloadMismatch(formType FormType, err error) error {
	return invalid("payload", fmt.Sprintf("does not match the %s shape: %v", formType, err))
}
