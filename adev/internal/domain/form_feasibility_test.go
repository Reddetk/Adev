package domain

import (
	"math/rand"
	"strings"
	"testing"
)

// premises is the workload the design decisions are scored against: a system
// without QPS, volume, team and SLO has nothing to decide with.

// validPremises is one complete, realistic premise set for the lending desk.
func validPremises() Premises {
	return Premises{
		Traffic:       TrafficProfile{PeakQps: 120, ReadWriteRatio: 4, PayloadBytes: 2048},
		Storage:       StorageProfile{GBPerYear: 40, GrowthPerYear: 1.5},
		Team:          TeamProfile{Size: 4, Languages: []string{"go"}, OpsMaturity: OpsMedium},
		BudgetMonthly: &Cost{Currency: "EUR", Amount: 800},
		DeadlineDays:  120,
		Availability:  99.9,
		RPOMinutes:    15,
		RTOMinutes:    60,
		Compliance:    []string{"gdpr"},
	}
}

// feasibilityWith builds a TEJ that passes every gate except the one under test.
func feasibilityWith(premises Premises) Feasibility {
	cost := Cost{Currency: "EUR", Amount: 1000}
	return Feasibility{
		ReportType: ReportTEJ,
		Options: []Option{{
			ID: "opt-1", Summary: Text{Value: "Build the lending module.", Lang: LangSTE100},
			Cost: &cost, ScheduleDays: 30, Confidence: ConfidenceHigh,
		}},
		RecommendedOption: "opt-1",
		Verdict:           VerdictGo,
		Reason:            Text{Value: "The team can build it.", Lang: LangSTE100},
		Premises:          premises,
	}
}

// TestFeasibilityNeedsThePremises: every premise is a question the operator
// answers before the design can choose anything. A missing premise is a gate
// failure that names the field, so the operator answer has somewhere to land.
func TestFeasibilityNeedsThePremises(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(*Premises)
		field  string
	}{
		{"no traffic", func(p *Premises) { p.Traffic.PeakQps = 0 }, "premises.traffic.peakQps"},
		{"negative traffic", func(p *Premises) { p.Traffic.PeakQps = -1 }, "premises.traffic.peakQps"},
		{"negative read ratio", func(p *Premises) { p.Traffic.ReadWriteRatio = -1 }, "premises.traffic.readWriteRatio"},
		{"negative payload", func(p *Premises) { p.Traffic.PayloadBytes = -1 }, "premises.traffic.payloadBytes"},
		{"no volume", func(p *Premises) { p.Storage.GBPerYear = 0 }, "premises.storage.gbPerYear"},
		{"negative growth", func(p *Premises) { p.Storage.GrowthPerYear = -1 }, "premises.storage.growthPerYear"},
		{"no team", func(p *Premises) { p.Team.Size = 0 }, "premises.team.size"},
		{"no language", func(p *Premises) { p.Team.Languages = nil }, "premises.team.languages"},
		{"empty language", func(p *Premises) { p.Team.Languages = []string{" "} }, "premises.team.languages"},
		{"unknown maturity", func(p *Premises) { p.Team.OpsMaturity = "expert" }, "premises.team.opsMaturity"},
		{"no budget", func(p *Premises) { p.BudgetMonthly = nil }, "premises.budgetMonthly"},
		{"zero budget", func(p *Premises) { p.BudgetMonthly = &Cost{Currency: "EUR", Amount: 0} }, "premises.budgetMonthly"},
		{"no deadline", func(p *Premises) { p.DeadlineDays = 0 }, "premises.deadlineDays"},
		{"no availability", func(p *Premises) { p.Availability = 0 }, "premises.availability"},
		{"negative rpo", func(p *Premises) { p.RPOMinutes = -1 }, "premises.rpoMinutes"},
		{"negative rto", func(p *Premises) { p.RTOMinutes = -1 }, "premises.rtoMinutes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			premises := validPremises()
			tc.break_(&premises)
			err := ValidateFeasibility(feasibilityWith(premises))
			if err == nil {
				t.Fatalf("a TEJ without %s must not pass", tc.field)
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("the failure must name %s, got %v", tc.field, err)
			}
		})
	}
}

// TestFeasibilityAcceptsCompletePremises: the shape of the gate is a boundary, and
// the boundary values are the ones a real operator gives.
func TestFeasibilityAcceptsCompletePremises(t *testing.T) {
	if err := ValidateFeasibility(feasibilityWith(validPremises())); err != nil {
		t.Fatalf("a complete TEJ must pass, got %v", err)
	}

	// Two nines is the weakest SLO that still names one, and it must pass.
	premises := validPremises()
	premises.Availability = 99.0
	if err := ValidateFeasibility(feasibilityWith(premises)); err != nil {
		t.Fatalf("99.0 availability must pass, got %v", err)
	}

	// Optional numbers may stay at zero: they mean "the operator does not know".
	premises = validPremises()
	premises.Traffic.ReadWriteRatio = 0
	premises.Traffic.PayloadBytes = 0
	premises.Storage.GrowthPerYear = 0
	if err := ValidateFeasibility(feasibilityWith(premises)); err != nil {
		t.Fatalf("unknown optional numbers must not block, got %v", err)
	}
}

// TestFeasibilityAvailabilityIsAPercentage: 100% availability is not a promise a
// system can keep, and 98.9 is below the floor this form accepts.
func TestFeasibilityAvailabilityIsAPercentage(t *testing.T) {
	for _, value := range []float64{100, 100.5, 98.9, -1} {
		premises := validPremises()
		premises.Availability = value
		err := ValidateFeasibility(feasibilityWith(premises))
		if err == nil || !strings.Contains(err.Error(), "premises.availability") {
			t.Fatalf("availability %v must be refused with the field named, got %v", value, err)
		}
	}
}

// TestFeasibilityPremisesReachTheShape: the model reads the fields from the
// generated shape, so a premise missing there is a premise nobody answers.
func TestFeasibilityPremisesReachTheShape(t *testing.T) {
	shape, err := PayloadShape(TypeFeasibility)
	if err != nil {
		t.Fatalf("shape: %v", err)
	}
	for _, want := range []string{
		"premises", "traffic", "peakQps", "readWriteRatio",
		"gbPerYear", "team", "languages", "opsMaturity",
		"budgetMonthly", "deadlineDays", "availability", "rpoMinutes", "rtoMinutes",
	} {
		if !strings.Contains(shape, want) {
			t.Fatalf("the shape must show %q, got %s", want, shape)
		}
	}
}

// TestFeasibilityPremisesSurviveARandomOperator: a real operator gives any number
// in range. Whatever the numbers are, the gate accepts them; whatever one field is
// broken, the gate names it. The seed is fixed so a failure reproduces.
func TestFeasibilityPremisesSurviveARandomOperator(t *testing.T) {
	random := rand.New(rand.NewSource(7))
	corruptions := []struct {
		field  string
		break_ func(*Premises)
	}{
		{"premises.traffic.peakQps", func(p *Premises) { p.Traffic.PeakQps = 0 }},
		{"premises.storage.gbPerYear", func(p *Premises) { p.Storage.GBPerYear = 0 }},
		{"premises.team.size", func(p *Premises) { p.Team.Size = 0 }},
		{"premises.team.languages", func(p *Premises) { p.Team.Languages = nil }},
		{"premises.team.opsMaturity", func(p *Premises) { p.Team.OpsMaturity = "" }},
		{"premises.budgetMonthly", func(p *Premises) { p.BudgetMonthly = nil }},
		{"premises.deadlineDays", func(p *Premises) { p.DeadlineDays = 0 }},
		{"premises.availability", func(p *Premises) { p.Availability = 100 }},
	}

	for i := 0; i < 200; i++ {
		premises := Premises{
			Traffic: TrafficProfile{
				PeakQps:        1 + random.Intn(100000),
				ReadWriteRatio: float64(random.Intn(50)),
				PayloadBytes:   random.Intn(1 << 20),
			},
			Storage: StorageProfile{
				GBPerYear:     float64(1 + random.Intn(100000)),
				GrowthPerYear: 1 + random.Float64()*3,
			},
			Team: TeamProfile{
				Size:        1 + random.Intn(50),
				Languages:   []string{[]string{"go", "java", "typescript", "python"}[random.Intn(4)]},
				OpsMaturity: []OpsMaturity{OpsLow, OpsMedium, OpsHigh}[random.Intn(3)],
			},
			BudgetMonthly: &Cost{Currency: "EUR", Amount: 100 + random.Float64()*10000},
			DeadlineDays:  1 + random.Intn(720),
			Availability:  99.0 + random.Float64()*0.99,
			RPOMinutes:    random.Intn(240),
			RTOMinutes:    random.Intn(240),
		}
		if err := ValidateFeasibility(feasibilityWith(premises)); err != nil {
			t.Fatalf("iteration %d: a valid random premise set must pass, got %v", i, err)
		}

		broken := premises
		choice := corruptions[random.Intn(len(corruptions))]
		choice.break_(&broken)
		err := ValidateFeasibility(feasibilityWith(broken))
		if err == nil || !strings.Contains(err.Error(), choice.field) {
			t.Fatalf("iteration %d: a broken %s must fail and be named, got %v", i, choice.field, err)
		}
	}
}
