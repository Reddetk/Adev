package usecase

import (
	"encoding/json"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestCoerceAnswerShapesTheAnswerLikeTheField: the operator types a sentence, and
// the field keeps the type the shape gives it. A list takes one entry, a record
// takes its free text, a text field keeps text.
func TestCoerceAnswerShapesTheAnswerLikeTheField(t *testing.T) {
	cases := []struct {
		name     string
		formType domain.FormType
		pointer  string
		value    string
		want     string
	}{
		{
			name:     "a sentence becomes one entry of a list of texts",
			formType: domain.TypeBRD,
			pointer:  "/scope/in",
			value:    `"book registration"`,
			want:     `[{"lang":"STE100","text":"book registration"}]`,
		},
		{
			name:     "a sentence fills the free text of a list entry the draft left out",
			formType: domain.TypeSRS,
			pointer:  "/fr/3/errors",
			value:    `"returns 400 when the room is taken"`,
			want:     `[{"condition":"returns 400 when the room is taken"}]`,
		},
		{
			name:     "a bare number stays text in a text field",
			formType: domain.TypeSRS,
			pointer:  "/nfr/0/threshold",
			value:    `2`,
			want:     `"2"`,
		},
		{
			name:     "a spelled-out number lands as a number",
			formType: domain.TypeBRD,
			pointer:  "/metrics/0/target",
			value:    `"95"`,
			want:     `95`,
		},
		{
			name:     "the operator may write the list as JSON",
			formType: domain.TypeBRD,
			pointer:  "/scope/in",
			value:    `["first","second"]`,
			want:     `["first","second"]`,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := coerceAnswer(test.formType, test.pointer, json.RawMessage(test.value))
			var gotValue, wantValue any
			if err := json.Unmarshal(got, &gotValue); err != nil {
				t.Fatalf("got is not JSON: %v (%s)", err, got)
			}
			if err := json.Unmarshal([]byte(test.want), &wantValue); err != nil {
				t.Fatalf("want is not JSON: %v", err)
			}
			gotJSON, _ := json.Marshal(gotValue)
			wantJSON, _ := json.Marshal(wantValue)
			if string(gotJSON) != string(wantJSON) {
				t.Fatalf("want %s, got %s", wantJSON, gotJSON)
			}
		})
	}
}
