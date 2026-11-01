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

// Concurrency says how many callers may reach the same key at once: none means
// one caller at a time, per_key serializes one key and lets the rest run, any
// means the operation is safe in parallel.
type Concurrency string

const (
	ConcurrencyNone   Concurrency = "none"
	ConcurrencyPerKey Concurrency = "per_key"
	ConcurrencyAny    Concurrency = "any"
)

// Idempotency says whether the operation may run twice with the same effect.
type Idempotency string

const (
	IdempotencyNone     Idempotency = "none"
	IdempotencyRequired Idempotency = "required"
)

// Delivery says how the operation reaches whoever depends on it: a sync call, or
// a message with at most once, at least once or exactly once delivery.
type Delivery string

const (
	DeliverySync        Delivery = "sync"
	DeliveryAtMostOnce  Delivery = "at_most_once"
	DeliveryAtLeastOnce Delivery = "at_least_once"
	DeliveryExactlyOnce Delivery = "exactly_once"
)

// FailureMode says what the system does when the operation fails: stop, retry, or
// compensate an already completed step.
type FailureMode string

const (
	FailureFailFast   FailureMode = "fail_fast"
	FailureRetry      FailureMode = "retry"
	FailureCompensate FailureMode = "compensate"
)

// FunctionalRequirement is one observable behavior of the target system.
type FunctionalRequirement struct {
	ID          string    `json:"id"`
	Statement   Statement `json:"statement"`
	Actor       string    `json:"actor"`
	Trigger     string    `json:"trigger"`
	Response    string    `json:"response"`
	StateChange string    `json:"stateChange"`
	// The four answers the later decisions are scored against: how many callers
	// reach one key, whether a repeat is safe, how the result travels, and what a
	// failure does.
	Concurrency Concurrency `json:"concurrency"`
	Idempotency Idempotency `json:"idempotency"`
	Delivery    Delivery    `json:"delivery"`
	FailureMode FailureMode `json:"failureMode"`
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
	TraceGoal        string    `json:"traceGoal,omitempty"`
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
