package domain

// Alternative is an option that the architect rejected and the fact that made it wrong.
type Alternative struct {
	Option          Text `json:"option"`
	RejectedBecause Text `json:"rejectedBecause"`
}

// DecisionStatus is the lifecycle state of one decision.
type DecisionStatus string

const (
	DecisionProposed   DecisionStatus = "proposed"
	DecisionAccepted   DecisionStatus = "accepted"
	DecisionSuperseded DecisionStatus = "superseded"
)

// Decision is one architecture decision record.
type Decision struct {
	ID           string         `json:"id"`
	Title        Text           `json:"title"`
	Status       DecisionStatus `json:"status"`
	Supersedes   string         `json:"supersedes,omitempty"`
	Context      Text           `json:"context,omitempty"`
	ObjectRefs   []string       `json:"objectRefs,omitempty"`
	Decision     Text           `json:"decision"`
	Alternatives []Alternative  `json:"alternatives"`
	Consequences []Text         `json:"consequences,omitempty"`
	Links        []Ref          `json:"links,omitempty"`
}

// ADR is the set of architecture decisions of the design.
type ADR struct {
	Decisions []Decision `json:"decisions"`
	OpenGaps  []Gap      `json:"openGaps,omitempty"`
}

// Texts returns every free-text value for the language checker.
func (a ADR) Texts() []Text {
	var out []Text
	for _, d := range a.Decisions {
		out = append(out, d.Title, d.Context, d.Decision)
		out = append(out, d.Consequences...)
		for _, alt := range d.Alternatives {
			out = append(out, alt.Option, alt.RejectedBecause)
		}
	}
	return out
}
