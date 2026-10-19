package domain

// Frequency is how often an actor does a task.
type Frequency string

const (
	FrequencyDay     Frequency = "day"
	FrequencyWeek    Frequency = "week"
	FrequencyMonth   Frequency = "month"
	FrequencyQuarter Frequency = "quarter"
	FrequencyEvent   Frequency = "event"
)

// Goal is one business result with the metric that proves it.
type Goal struct {
	ID       string `json:"id"`
	Outcome  Text   `json:"outcome"`
	MetricID string `json:"metricId"`
}

// Metric measures one business result. Baseline and Target are pointers so that a real zero is not confused with absent data.
type Metric struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Unit     string   `json:"unit"`
	Baseline *float64 `json:"baseline"`
	Target   *float64 `json:"target"`
}

// Scope is the project boundary.
type Scope struct {
	In       []Text `json:"in"`
	Out      []Text `json:"out"`
	Deferred []Text `json:"deferred,omitempty"`
}

// Actor is a person or system that uses the target system.
type Actor struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Tasks          []Text    `json:"tasks"`
	Frequency      Frequency `json:"frequency"`
	BusinessObject string    `json:"businessObject"`
}

// BRD is the Business Requirements Document.
type BRD struct {
	Goals          []Goal      `json:"goals"`
	Metrics        []Metric    `json:"metrics"`
	Scope          Scope       `json:"scope"`
	Actors         []Actor     `json:"actors"`
	Constraints    []Statement `json:"constraints,omitempty"`
	Assumptions    []Text      `json:"assumptions,omitempty"`
	SuccessMetrics []Success   `json:"successMetrics"`
	OpenGaps       []Gap       `json:"openGaps,omitempty"`
}

// Success is the measurable event that proves a goal.
type Success struct {
	MetricID  string `json:"metricId"`
	Threshold string `json:"threshold"`
	Window    string `json:"window"`
}

// Texts returns every text value of the form. The language checker uses it.
func (b BRD) Texts() []Text {
	var out []Text
	for _, goal := range b.Goals {
		out = append(out, goal.Outcome)
	}
	out = append(out, b.Scope.In...)
	out = append(out, b.Scope.Out...)
	out = append(out, b.Scope.Deferred...)
	for _, actor := range b.Actors {
		out = append(out, actor.Tasks...)
	}
	for _, constraint := range b.Constraints {
		out = append(out, Text{Value: constraint.Text, Lang: LangSTE100})
	}
	out = append(out, b.Assumptions...)
	for _, gap := range b.OpenGaps {
		out = append(out, gap.Question)
	}
	return out
}
