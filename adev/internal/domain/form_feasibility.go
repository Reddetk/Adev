package domain

import (
	"fmt"
	"strings"
)

// Confidence is the strength of a cost estimate. A percentage without a method is a guess.
type Confidence string

const (
	ConfidenceLow    Confidence = "low"
	ConfidenceMedium Confidence = "medium"
	ConfidenceHigh   Confidence = "high"
)

// OpsMaturity is how much operational work the team already carries: a low
// maturity team cannot absorb a platform that needs a dedicated operator.
type OpsMaturity string

const (
	OpsLow    OpsMaturity = "low"
	OpsMedium OpsMaturity = "medium"
	OpsHigh   OpsMaturity = "high"
)

// TrafficProfile is the load the design must survive: peak requests per second,
// how many reads fall on one write, and the size of a request body.
type TrafficProfile struct {
	PeakQps        int     `json:"peakQps"`
	ReadWriteRatio float64 `json:"readWriteRatio,omitempty"`
	PayloadBytes   int     `json:"payloadBytes,omitempty"`
}

// StorageProfile is how much data the system keeps and how fast it grows.
type StorageProfile struct {
	GBPerYear     float64 `json:"gbPerYear"`
	GrowthPerYear float64 `json:"growthPerYear,omitempty"`
}

// TeamProfile is who builds and runs the system. The language is a premise, not a
// preference: the design takes what the team already runs.
type TeamProfile struct {
	Size        int         `json:"size"`
	Languages   []string    `json:"languages"`
	OpsMaturity OpsMaturity `json:"opsMaturity"`
}

// Premises are the facts the design decisions are scored against. A premise is
// never a default: a missing one becomes a question for the operator.
type Premises struct {
	Traffic       TrafficProfile `json:"traffic"`
	Storage       StorageProfile `json:"storage"`
	Team          TeamProfile    `json:"team"`
	BudgetMonthly *Cost          `json:"budgetMonthly,omitempty"`
	DeadlineDays  int            `json:"deadlineDays"`
	Availability  float64        `json:"availability"`
	RPOMinutes    int            `json:"rpoMinutes"`
	RTOMinutes    int            `json:"rtoMinutes"`
	Compliance    []string       `json:"compliance,omitempty"`
}

// Cost is a monetary amount in one currency.
type Cost struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

// Option is one solution that can satisfy the BRD goals.
type Option struct {
	ID           string     `json:"id"`
	Summary      Text       `json:"summary"`
	Reused       []string   `json:"reused,omitempty"`
	Created      []Ref      `json:"created,omitempty"`
	Cost         *Cost      `json:"cost,omitempty"`
	ScheduleDays int        `json:"scheduleDays"`
	Confidence   Confidence `json:"confidence"`
}

// Risk is an event that can stop the project.
type Risk struct {
	ID         string `json:"id"`
	Event      Text   `json:"event"`
	Likelihood int    `json:"likelihood"`
	Impact     int    `json:"impact"`
	Mitigation Text   `json:"mitigation,omitempty"`
	ObjectID   string `json:"objectId,omitempty"`
}

// ScopeDelta is one correction the study makes to the BRD scope.
type ScopeDelta struct {
	Removed string `json:"removed,omitempty"`
	Added   string `json:"added,omitempty"`
	Reason  Text   `json:"reason"`
}

// ReportType is the document the operator asked for at stage 01: a technical and
// economic justification or a risk assessment report. They are alternatives, not
// sections of one document.
type ReportType string

const (
	ReportTEJ ReportType = "tej"
	ReportRAR ReportType = "rar"
)

// Verdict is the outcome of the feasibility study.
type Verdict string

const (
	VerdictGo     Verdict = "go"
	VerdictNoGo   Verdict = "no_go"
	VerdictRevise Verdict = "revise"
)

// Feasibility is the techno-economic justification of the greenfield route: it
// compares options for the BRD goals, prices them and picks a verdict.
// shortcut: the Risk field and ReportRAR stay in the schema for a future change
// route with a running baseline; the greenfield gates accept a TEJ only.
type Feasibility struct {
	ReportType        ReportType   `json:"reportType"`
	Premises          Premises     `json:"premises"`
	Options           []Option     `json:"options"`
	Risks             []Risk       `json:"risks,omitempty"`
	RecommendedOption string       `json:"recommendedOption"`
	Verdict           Verdict      `json:"verdict"`
	Reason            Text         `json:"reason"`
	BrdRef            *Ref         `json:"brdRef,omitempty"`
	ScopeDelta        []ScopeDelta `json:"scopeDelta,omitempty"`
	OpenGaps          []Gap        `json:"openGaps,omitempty"`
}

// Texts returns every free-text value for the language checker.
func (f Feasibility) Texts() []Text {
	var out []Text
	out = append(out, f.Reason)
	for _, option := range f.Options {
		out = append(out, option.Summary)
	}
	for _, risk := range f.Risks {
		out = append(out, risk.Event, risk.Mitigation)
	}
	for _, delta := range f.ScopeDelta {
		out = append(out, delta.Reason)
	}
	return out
}

// architectureWords are the style names the TEJ must not choose: the style is a
// decision of the HLD, where the bounded contexts already exist and the container
// count can prove or break it.
var architectureWords = []string{
	"monolith", "microservice", "event-driven", "event driven",
	"serverless", "hexagonal", "cell-based", "cell based",
}

// validateFeasibilityOptions keeps the estimate an estimate: a delivery option,
// not a deployment style. The live TEJ offered "a monolith" and "microservices" as
// options before the domain had a single bounded context.
// shortcut: a keyword check on the summary; replace it with the option kind when the
// option carries one.
func validateFeasibilityOptions(options []Option) []error {
	var errs []error
	for i, option := range options {
		lower := strings.ToLower(option.Summary.Value)
		for _, word := range architectureWords {
			if strings.Contains(lower, word) {
				errs = append(errs, invalid(fmt.Sprintf("options[%d].summary", i),
					fmt.Sprintf("names the architecture style %q; the HLD chooses the style after the bounded contexts, the estimate prices a way to deliver", word)))
				break
			}
		}
	}
	return errs
}
