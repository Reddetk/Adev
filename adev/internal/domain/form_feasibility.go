package domain

// Confidence is the strength of a cost estimate. A percentage without a method is a guess.
type Confidence string

const (
	ConfidenceLow    Confidence = "low"
	ConfidenceMedium Confidence = "medium"
	ConfidenceHigh   Confidence = "high"
)

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
