package domain

// Lang is the controlled language of a text value.
type Lang string

const (
	LangSTE100 Lang = "STE100"
	LangACE    Lang = "ACE"
)

// Text is the only carrier of free text in a form.
type Text struct {
	Value string `json:"text"`
	Lang  Lang   `json:"lang"`
}

// Modality is the strength of a normative statement.
type Modality string

const (
	ModalityShall  Modality = "shall"
	ModalityShould Modality = "should"
	ModalityMay    Modality = "may"
)

// Priority is the requirement priority.
type Priority string

const (
	PriorityMust Priority = "must"
	PriorityHigh Priority = "high"
	PriorityLow  Priority = "low"
)

// Statement is a normative sentence with a modality and a priority.
type Statement struct {
	ID       string   `json:"id"`
	Text     string   `json:"text"`
	Modality Modality `json:"modality"`
	Priority Priority `json:"priority"`
}

// Gap is missing data. A blocking gap stops the pipeline until the operator answers.
type Gap struct {
	ID       string   `json:"id"`
	FormRef  Ref      `json:"formRef"`
	Pointer  string   `json:"pointer"`
	Question Text     `json:"question"`
	Options  []string `json:"options"`
	Blocking bool     `json:"blocking"`
}
