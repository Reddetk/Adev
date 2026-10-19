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
			return err
		}
		return ValidateUserInput(in)
	case TypeBRD:
		var brd BRD
		if err := f.BindPayload(&brd); err != nil {
			return err
		}
		return ValidateBRD(brd, f.Status)
	default:
		return fmt.Errorf("no payload gates for form type %q yet", f.FormType)
	}
}
