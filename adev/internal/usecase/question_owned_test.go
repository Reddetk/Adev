package usecase

import (
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestSystemOwnedPointerDropsTheMoneyChoice: the model estimates the cost and
// picks the currency, so the FEASIBILITY interview must never ask for either. The
// budget of the operator is a premise, not an estimate, and stays a question.
func TestSystemOwnedPointerDropsTheMoneyChoice(t *testing.T) {
	for _, pointer := range []string{"/options/0/cost/currency", "/options/0/cost/amount", "/containers/0/id"} {
		if !systemOwnedPointer(domain.TypeFeasibility, pointer) {
			t.Fatalf("%s must be system owned", pointer)
		}
	}
	for _, pointer := range []string{"/goals/0/outcome", "/glossary/0", "/constraints/1",
		"/premises/budgetMonthly/amount", "/premises/budgetMonthly/currency"} {
		if systemOwnedPointer(domain.TypeFeasibility, pointer) {
			t.Fatalf("%s is a question for the operator", pointer)
		}
	}
}

// TestSRSStructuralFieldsAreNotQuestions: actor, trigger, response, stateChange and
// the scenarios are what the requirement says, and the model writes them from the
// BRD. The four behaviour premises stay questions, because the business decides
// them.
func TestSRSStructuralFieldsAreNotQuestions(t *testing.T) {
	for _, pointer := range []string{"/fr/1/actor", "/fr/1/trigger", "/fr/1/response", "/fr/1/stateChange", "/fr/1/priority"} {
		if !systemOwnedPointer(domain.TypeSRS, pointer) {
			t.Fatalf("%s is the model job, not a question", pointer)
		}
	}
	for _, pointer := range []string{"/fr/1/concurrency", "/fr/1/idempotency", "/fr/1/delivery", "/fr/1/failureMode"} {
		if systemOwnedPointer(domain.TypeSRS, pointer) {
			t.Fatalf("%s is a business decision and must stay a question", pointer)
		}
	}
}
