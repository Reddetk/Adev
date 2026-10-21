package domain

// formSpec is everything the system knows about one payload type: how to bind it,
// how to check it, which texts the language checker reads, and which fields a role
// may see. A service looks the type up in the table instead of switching on it.
type formSpec struct {
	payload    func() any
	validate   func(any, Status) error
	texts      func(any) []Text
	gaps       func(any) []Gap
	projection Projection
}

type formSpecEntry struct {
	Type FormType
	formSpec
}

// formSpecs is the form registry of the domain. Adding a form type means adding
// one entry here: the gates, the shape, the language texts and the projection all
// come from it.
var formSpecs = []formSpecEntry{
	{TypeUserInput, formSpec{
		payload:    func() any { return &UserInput{} },
		validate:   func(payload any, _ Status) error { return ValidateUserInput(*payload.(*UserInput)) },
		texts:      func(payload any) []Text { return []Text{payload.(*UserInput).RawRequest} },
		projection: userInputProjection,
	}},
	{TypeBRD, formSpec{
		payload:    func() any { return &BRD{} },
		validate:   func(payload any, status Status) error { return ValidateBRD(*payload.(*BRD), status) },
		texts:      func(payload any) []Text { return payload.(*BRD).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*BRD).OpenGaps },
		projection: BRDProjection,
	}},
	{TypeFeasibility, formSpec{
		payload:    func() any { return &Feasibility{} },
		validate:   func(payload any, _ Status) error { return ValidateFeasibility(*payload.(*Feasibility)) },
		texts:      func(payload any) []Text { return payload.(*Feasibility).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*Feasibility).OpenGaps },
		projection: openDesignProjection,
	}},
	{TypeSRS, formSpec{
		payload:    func() any { return &SRS{} },
		validate:   func(payload any, _ Status) error { return ValidateSRS(*payload.(*SRS)) },
		texts:      func(payload any) []Text { return payload.(*SRS).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*SRS).OpenGaps },
		projection: openDesignProjection,
	}},
	{TypeHLD, formSpec{
		payload:    func() any { return &HLD{} },
		validate:   func(payload any, _ Status) error { return ValidateHLD(*payload.(*HLD)) },
		texts:      func(payload any) []Text { return payload.(*HLD).Texts() },
		projection: openDesignProjection,
	}},
	{TypeADR, formSpec{
		payload:    func() any { return &ADR{} },
		validate:   func(payload any, _ Status) error { return ValidateADR(*payload.(*ADR)) },
		texts:      func(payload any) []Text { return payload.(*ADR).Texts() },
		projection: openDesignProjection,
	}},
	{TypeLLD, formSpec{
		payload:    func() any { return &LLD{} },
		validate:   func(payload any, _ Status) error { return ValidateLLD(*payload.(*LLD)) },
		texts:      func(payload any) []Text { return payload.(*LLD).Texts() },
		projection: openDesignProjection,
	}},
	{TypeTaskPlan, formSpec{
		payload:    func() any { return &TaskPlan{} },
		validate:   func(payload any, _ Status) error { return ValidateTaskPlan(*payload.(*TaskPlan)) },
		texts:      func(payload any) []Text { return payload.(*TaskPlan).Texts() },
		projection: openDesignProjection,
	}},
}

// formSpecFor returns the spec of a form type. A type without an entry has no
// payload gates and cannot be produced.
func formSpecFor(formType FormType) (formSpec, bool) {
	for _, entry := range formSpecs {
		if entry.Type == formType {
			return entry.formSpec, true
		}
	}
	return formSpec{}, false
}
