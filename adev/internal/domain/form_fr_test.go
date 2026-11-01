package domain

import (
	"math/rand"
	"strings"
	"testing"
)

// A functional requirement is scored later by the consistency, communication and
// reliability decisions, so it must say how it behaves under two callers, a retry
// and a failure. Without those four answers the design is a guess.

func validFR() FunctionalRequirement {
	return FunctionalRequirement{
		ID:          "fr-1",
		Statement:   Statement{ID: "fr-1", Text: "The client creates a shipment.", Modality: ModalityShall, Priority: PriorityMust},
		Actor:       "client",
		Trigger:     "The client submits the shipment form.",
		Response:    "The system creates the shipment.",
		StateChange: "A shipment row exists.",
		Priority:    PriorityMust,
		Concurrency: ConcurrencyPerKey,
		Idempotency: IdempotencyRequired,
		Delivery:    DeliverySync,
		FailureMode: FailureRetry,
		Acceptance:  []Scenario{{Given: "a client", When: "it submits the form", Then: "the shipment exists"}},
	}
}

// TestFRNeedsTheBehaviourPremises: each of the four answers is a gate, and the
// failure names the field the operator can answer.
func TestFRNeedsTheBehaviourPremises(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(*FunctionalRequirement)
		field  string
	}{
		{"no concurrency", func(fr *FunctionalRequirement) { fr.Concurrency = "" }, "fr[0].concurrency"},
		{"unknown concurrency", func(fr *FunctionalRequirement) { fr.Concurrency = "parallel" }, "fr[0].concurrency"},
		{"no idempotency", func(fr *FunctionalRequirement) { fr.Idempotency = "" }, "fr[0].idempotency"},
		{"unknown idempotency", func(fr *FunctionalRequirement) { fr.Idempotency = "maybe" }, "fr[0].idempotency"},
		{"no delivery", func(fr *FunctionalRequirement) { fr.Delivery = "" }, "fr[0].delivery"},
		{"unknown delivery", func(fr *FunctionalRequirement) { fr.Delivery = "sometimes" }, "fr[0].delivery"},
		{"no failure mode", func(fr *FunctionalRequirement) { fr.FailureMode = "" }, "fr[0].failureMode"},
		{"unknown failure mode", func(fr *FunctionalRequirement) { fr.FailureMode = "hope" }, "fr[0].failureMode"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fr := validFR()
			tc.break_(&fr)
			err := ValidateSRS(SRS{FR: []FunctionalRequirement{fr}})
			if err == nil {
				t.Fatalf("a requirement without %s must not pass", tc.field)
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("the failure must name %s, got %v", tc.field, err)
			}
		})
	}
}

// TestFRAcceptsEveryAllowedValue: the palette is closed, and every value of it is
// a legal answer, not only the default one.
func TestFRAcceptsEveryAllowedValue(t *testing.T) {
	for _, concurrency := range []Concurrency{ConcurrencyNone, ConcurrencyPerKey, ConcurrencyAny} {
		for _, idempotency := range []Idempotency{IdempotencyNone, IdempotencyRequired} {
			for _, delivery := range []Delivery{DeliverySync, DeliveryAtMostOnce, DeliveryAtLeastOnce, DeliveryExactlyOnce} {
				for _, failure := range []FailureMode{FailureFailFast, FailureRetry, FailureCompensate} {
					fr := validFR()
					fr.Concurrency, fr.Idempotency, fr.Delivery, fr.FailureMode = concurrency, idempotency, delivery, failure
					if err := ValidateSRS(SRS{FR: []FunctionalRequirement{fr}}); err != nil {
						t.Fatalf("%s/%s/%s/%s must pass, got %v", concurrency, idempotency, delivery, failure, err)
					}
				}
			}
		}
	}
}

// TestFRPremisesReachTheShape: the model reads the fields from the generated shape.
func TestFRPremisesReachTheShape(t *testing.T) {
	shape, err := PayloadShape(TypeSRS)
	if err != nil {
		t.Fatalf("shape: %v", err)
	}
	for _, want := range []string{"concurrency", "idempotency", "delivery", "failureMode"} {
		if !strings.Contains(shape, want) {
			t.Fatalf("the shape must show %q, got %s", want, shape)
		}
	}
}

// TestFRPremisesSurviveARandomRequirement: any legal combination passes, a single
// broken field is named. The seed is fixed so a failure reproduces.
func TestFRPremisesSurviveARandomRequirement(t *testing.T) {
	random := rand.New(rand.NewSource(11))
	concurrencies := []Concurrency{ConcurrencyNone, ConcurrencyPerKey, ConcurrencyAny}
	idempotencies := []Idempotency{IdempotencyNone, IdempotencyRequired}
	deliveries := []Delivery{DeliverySync, DeliveryAtMostOnce, DeliveryAtLeastOnce, DeliveryExactlyOnce}
	failures := []FailureMode{FailureFailFast, FailureRetry, FailureCompensate}

	for i := 0; i < 200; i++ {
		fr := validFR()
		fr.ID = "fr-" + string(rune('a'+random.Intn(26)))
		fr.Concurrency = concurrencies[random.Intn(len(concurrencies))]
		fr.Idempotency = idempotencies[random.Intn(len(idempotencies))]
		fr.Delivery = deliveries[random.Intn(len(deliveries))]
		fr.FailureMode = failures[random.Intn(len(failures))]

		if err := ValidateSRS(SRS{FR: []FunctionalRequirement{fr}}); err != nil {
			t.Fatalf("iteration %d: a legal requirement must pass, got %v", i, err)
		}

		broken := fr
		switch random.Intn(4) {
		case 0:
			broken.Concurrency = ""
		case 1:
			broken.Idempotency = ""
		case 2:
			broken.Delivery = ""
		default:
			broken.FailureMode = ""
		}
		if err := ValidateSRS(SRS{FR: []FunctionalRequirement{broken}}); err == nil {
			t.Fatalf("iteration %d: an incomplete requirement must fail", i)
		}
	}
}
