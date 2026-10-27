package domain

import (
	"errors"
	"testing"
)

func TestFormValidateNewVersion(t *testing.T) {
	tests := []struct {
		name    string
		form    Form
		wantErr error
	}{
		{
			name: "the first version needs only an identity",
			form: Form{Envelope: Envelope{FormID: "FRM-1", FormVersion: 1}},
		},
		{
			name: "a later version needs the previous one",
			form: Form{Envelope: Envelope{
				FormID: "FRM-1", FormVersion: 2,
				PreviousFormVersion: &Ref{FormID: "FRM-1", FormVersion: 1},
			}},
		},
		{
			name:    "an empty form id is rejected",
			form:    Form{Envelope: Envelope{FormVersion: 1}},
			wantErr: errors.New("formId and a formVersion of at least 1 are required"),
		},
		{
			name:    "a version below one is rejected",
			form:    Form{Envelope: Envelope{FormID: "FRM-1"}},
			wantErr: errors.New("formId and a formVersion of at least 1 are required"),
		},
		{
			name:    "an orphan version is rejected",
			form:    Form{Envelope: Envelope{FormID: "FRM-1", FormVersion: 2}},
			wantErr: ErrMissingPrevious,
		},
		{
			name: "a gap in the chain is rejected",
			form: Form{Envelope: Envelope{
				FormID: "FRM-1", FormVersion: 3,
				PreviousFormVersion: &Ref{FormID: "FRM-1", FormVersion: 1},
			}},
			wantErr: ErrMissingPrevious,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.form.ValidateNewVersion()
			if test.wantErr == nil {
				if err != nil {
					t.Fatalf("want no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("want an error, got nil")
			}
			// The sentinel errors of the chain carry context, the plain one does not.
			if errors.Is(test.wantErr, ErrMissingPrevious) {
				if !errors.Is(err, ErrMissingPrevious) {
					t.Fatalf("want ErrMissingPrevious, got %v", err)
				}
				return
			}
			if err.Error() != test.wantErr.Error() {
				t.Fatalf("want %q, got %q", test.wantErr, err)
			}
		})
	}
}
