package usecase

import (
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// The registry file name is derived one way for every form type, so an
// added form type reaches the model through this rule or not at all.
func TestRegistryFileNameFollowsTheFormType(t *testing.T) {
	cases := map[domain.FormType]string{
		domain.TypeBRD:              "brd.md",
		domain.TypeSRS:              "srs.md",
		domain.TypeFeasibility:      "feasibility.md",
		domain.TypeCRBrief:          "cr-brief.md",
		domain.TypeImpactAssessment: "impact-assessment.md",
		domain.TypeTestContract:     "test-contract.md",
	}
	for formType, want := range cases {
		if got := registryFileName(formType); got != want {
			t.Fatalf("registry file of %s: got %q, want %q", formType, got, want)
		}
	}
}
