package domain

import (
	"fmt"
	"strings"
)

// availabilityFloor is the weakest service level the form accepts: below two
// nines the number says nothing a design can be scored against.
const availabilityFloor = 99.0

// validatePremises checks the facts the later decisions are scored against. Each
// failure names one field, because the operator answer lands on that pointer.
func validatePremises(p Premises) []error {
	var errs []error
	add := func(field, problem string) { errs = append(errs, invalid(field, problem)) }

	if p.Traffic.PeakQps <= 0 {
		add("premises.traffic.peakQps", "must be greater than zero")
	}
	if p.Traffic.ReadWriteRatio < 0 {
		add("premises.traffic.readWriteRatio", "must not be negative")
	}
	if p.Traffic.PayloadBytes < 0 {
		add("premises.traffic.payloadBytes", "must not be negative")
	}
	if p.Storage.GBPerYear <= 0 {
		add("premises.storage.gbPerYear", "must be greater than zero")
	}
	if p.Storage.GrowthPerYear < 0 {
		add("premises.storage.growthPerYear", "must not be negative")
	}
	if p.Team.Size <= 0 {
		add("premises.team.size", "must be greater than zero")
	}
	if len(p.Team.Languages) == 0 {
		add("premises.team.languages", "must name the language the team runs")
	}
	for _, language := range p.Team.Languages {
		if strings.TrimSpace(language) == "" {
			add("premises.team.languages", "must not hold an empty language")
			break
		}
	}
	switch p.Team.OpsMaturity {
	case OpsLow, OpsMedium, OpsHigh:
	default:
		add("premises.team.opsMaturity", fmt.Sprintf("unknown value %q", p.Team.OpsMaturity))
	}
	if p.BudgetMonthly == nil {
		add("premises.budgetMonthly", "must state the monthly budget")
	} else {
		if p.BudgetMonthly.Amount <= 0 {
			add("premises.budgetMonthly", "must be greater than zero")
		}
		if strings.TrimSpace(p.BudgetMonthly.Currency) == "" {
			add("premises.budgetMonthly", "must name the currency")
		}
	}
	if p.DeadlineDays <= 0 {
		add("premises.deadlineDays", "must be greater than zero")
	}
	if p.Availability < availabilityFloor || p.Availability >= 100 {
		add("premises.availability", fmt.Sprintf("must be at least %.1f and below 100, got %v", availabilityFloor, p.Availability))
	}
	if p.RPOMinutes < 0 {
		add("premises.rpoMinutes", "must not be negative")
	}
	if p.RTOMinutes < 0 {
		add("premises.rtoMinutes", "must not be negative")
	}
	return errs
}
