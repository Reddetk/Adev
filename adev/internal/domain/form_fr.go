package domain

import (
	"errors"
	"fmt"
)

// FR is the functional requirements document: the fr list of the SRS as its own
// file, so the operator reads and accepts it apart from the NFR.
type FR struct {
	FR       []FunctionalRequirement `json:"fr"`
	OpenGaps []Gap                   `json:"openGaps,omitempty"`
}

// NFR is the non-functional requirements document.
type NFR struct {
	NFR      []NonFunctionalRequirement `json:"nfr"`
	OpenGaps []Gap                      `json:"openGaps,omitempty"`
}

// Texts returns every free text the language checker reads.
func (f FR) Texts() []Text {
	var out []Text
	for _, item := range f.FR {
		out = append(out, Text{Value: item.Statement.Text, Lang: LangACE})
	}
	return out
}

// Texts returns every free text the language checker reads.
func (n NFR) Texts() []Text {
	var out []Text
	for _, item := range n.NFR {
		out = append(out, Text{Value: item.Statement.Text, Lang: LangACE})
	}
	return out
}

// ValidateFR checks the gates of a functional requirements document.
func ValidateFR(f FR) error {
	if len(f.FR) == 0 {
		return invalid("fr", "must contain at least one entry")
	}
	var errs []error
	errs = append(errs, validateFunctionalRequirements(f.FR, "fr")...)
	return errors.Join(errs...)
}

// ValidateNFR checks the gates of a non-functional requirements document.
func ValidateNFR(n NFR) error {
	if len(n.NFR) == 0 {
		return invalid("nfr", "must contain at least one entry")
	}
	var errs []error
	errs = append(errs, validateNonFunctionalRequirements(n.NFR, "nfr")...)
	return errors.Join(errs...)
}

func validateFunctionalRequirements(list []FunctionalRequirement, prefix string) []error {
	var errs []error
	for i, fr := range list {
		field := fmt.Sprintf("%s[%d]", prefix, i)
		if fr.ID == "" {
			errs = append(errs, invalid(field+".id", "is empty"))
		}
		if fr.Statement.Text == "" {
			errs = append(errs, invalid(field+".statement", "is empty"))
		}
		if fr.Priority != PriorityMust && fr.Priority != PriorityHigh && fr.Priority != PriorityLow {
			errs = append(errs, invalid(field+".priority", fmt.Sprintf("unknown value %q", fr.Priority)))
		}
		if len(fr.Acceptance) == 0 {
			errs = append(errs, invalid(field+".acceptance", "must contain at least one scenario"))
		}
		for j, scenario := range fr.Acceptance {
			if scenario.Given == "" || scenario.When == "" || scenario.Then == "" {
				errs = append(errs, invalid(fmt.Sprintf("%s.acceptance[%d]", field, j), "given, when and then must be non-empty"))
			}
		}
	}
	return errs
}

func validateNonFunctionalRequirements(list []NonFunctionalRequirement, prefix string) []error {
	var errs []error
	for i, nfr := range list {
		field := fmt.Sprintf("%s[%d]", prefix, i)
		if nfr.ID == "" {
			errs = append(errs, invalid(field+".id", "is empty"))
		}
		if nfr.Metric == "" || nfr.Unit == "" || nfr.Threshold == "" || nfr.MeasurementPoint == "" || nfr.Window == "" {
			errs = append(errs, invalid(field, "metric, unit, threshold, measurementPoint and window must be non-empty"))
		}
	}
	return errs
}
