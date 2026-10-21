package domain

// Scenario is one testable path with the state before, the action, and the result.
type Scenario struct {
	Given string `json:"given"`
	When  string `json:"when"`
	Then  string `json:"then"`
}

// ErrorSpec is the behavior at an invalid input or a limit.
type ErrorSpec struct {
	Code      string   `json:"code"`
	Condition string   `json:"condition"`
	Limit     *float64 `json:"limit,omitempty"`
}

// FunctionalRequirement is one observable behavior of the target system.
type FunctionalRequirement struct {
	ID          string      `json:"id"`
	Statement   Statement   `json:"statement"`
	Actor       string      `json:"actor"`
	Trigger     string      `json:"trigger"`
	Response    string      `json:"response"`
	StateChange string      `json:"stateChange"`
	Priority    Priority    `json:"priority"`
	TraceGoal   string      `json:"traceGoal,omitempty"`
	Acceptance  []Scenario  `json:"acceptance,omitempty"`
	Errors      []ErrorSpec `json:"errors,omitempty"`
}

// NonFunctionalRequirement is one measurable quality of the target system.
type NonFunctionalRequirement struct {
	ID               string    `json:"id"`
	Statement        Statement `json:"statement"`
	Metric           string    `json:"metric"`
	Unit             string    `json:"unit"`
	Threshold        string    `json:"threshold"`
	MeasurementPoint string    `json:"measurementPoint"`
	Window           string    `json:"window"`
	VerifyMethod     string    `json:"verifyMethod"`
}

// OutOfScopeItem is a behavior that this SRS does not require.
type OutOfScopeItem struct {
	Text        Text   `json:"text"`
	BrdScopeRef string `json:"brdScopeRef,omitempty"`
}

// SRS is the software requirements specification. The question phase of the
// stage puts every unanswered question into openGaps; the operator answer
// removes the entry and fills the field it points to.
type SRS struct {
	FR           []FunctionalRequirement    `json:"fr"`
	NFR          []NonFunctionalRequirement `json:"nfr"`
	OutOfScope   []OutOfScopeItem           `json:"outOfScope,omitempty"`
	TraceToGoals []string                   `json:"traceToGoals,omitempty"`
	OpenGaps     []Gap                      `json:"openGaps,omitempty"`
}

// Texts returns every free-text value for the language checker.
func (s SRS) Texts() []Text {
	var out []Text
	for _, fr := range s.FR {
		out = append(out, Text{Value: fr.Statement.Text, Lang: LangACE})
	}
	for _, nfr := range s.NFR {
		out = append(out, Text{Value: nfr.Statement.Text, Lang: LangACE})
	}
	for _, item := range s.OutOfScope {
		out = append(out, item.Text)
	}
	return out
}
