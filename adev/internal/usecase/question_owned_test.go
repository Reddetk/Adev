package usecase

import (
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestSystemOwnedPointerDropsTheMoneyChoice: the model estimates the cost and
// picks the currency, so the FEASIBILITY interview must never ask for either.
func TestSystemOwnedPointerDropsTheMoneyChoice(t *testing.T) {
	for _, pointer := range []string{"/options/0/cost/currency", "/options/0/cost/amount", "/containers/0/id"} {
		if !systemOwnedPointer(domain.TypeFeasibility, pointer) {
			t.Fatalf("%s must be system owned", pointer)
		}
	}
	for _, pointer := range []string{"/goals/0/outcome", "/glossary/0", "/constraints/1"} {
		if systemOwnedPointer(domain.TypeBRD, pointer) {
			t.Fatalf("%s is a question for the operator", pointer)
		}
	}
}
