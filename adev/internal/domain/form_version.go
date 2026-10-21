package domain

import (
	"errors"
	"fmt"
)

// ValidateNewVersion checks the identity and the previous-version link of a form
// that is about to be stored. Every store calls it: the append-only chain rule has
// one implementation, and an adapter adds only its own existence check.
func (f *Form) ValidateNewVersion() error {
	if f.FormID == "" || f.FormVersion < 1 {
		return errors.New("formId and a formVersion of at least 1 are required")
	}
	if f.FormVersion == 1 {
		return nil
	}
	switch {
	case f.PreviousFormVersion == nil:
		return fmt.Errorf("%w: formVersion %d needs previousFormVersion", ErrMissingPrevious, f.FormVersion)
	case f.PreviousFormVersion.FormVersion != f.FormVersion-1:
		return fmt.Errorf("%w: previousFormVersion must be %d", ErrMissingPrevious, f.FormVersion-1)
	}
	return nil
}
