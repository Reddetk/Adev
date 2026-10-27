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
	{TypeFR, formSpec{
		payload:    func() any { return &FR{} },
		validate:   func(payload any, _ Status) error { return ValidateFR(*payload.(*FR)) },
		texts:      func(payload any) []Text { return payload.(*FR).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*FR).OpenGaps },
		projection: openDesignProjection,
	}},
	{TypeNFR, formSpec{
		payload:    func() any { return &NFR{} },
		validate:   func(payload any, _ Status) error { return ValidateNFR(*payload.(*NFR)) },
		texts:      func(payload any) []Text { return payload.(*NFR).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*NFR).OpenGaps },
		projection: openDesignProjection,
	}},
	{TypeHLD, formSpec{
		payload:    func() any { return &HLD{} },
		validate:   func(payload any, _ Status) error { return ValidateHLD(*payload.(*HLD)) },
		texts:      func(payload any) []Text { return payload.(*HLD).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*HLD).OpenGaps },
		projection: openDesignProjection,
	}},
	{TypeADR, formSpec{
		payload:    func() any { return &ADR{} },
		validate:   func(payload any, _ Status) error { return ValidateADR(*payload.(*ADR)) },
		texts:      func(payload any) []Text { return payload.(*ADR).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*ADR).OpenGaps },
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
	{TypeTestContract, formSpec{
		payload:    func() any { return &TestContract{} },
		validate:   func(payload any, _ Status) error { return ValidateTestContract(*payload.(*TestContract)) },
		texts:      func(payload any) []Text { return payload.(*TestContract).Texts() },
		projection: openDesignProjection,
	}},
	{TypeCRBrief, formSpec{
		payload:    func() any { return &CRBrief{} },
		validate:   func(payload any, _ Status) error { return ValidateCRBrief(*payload.(*CRBrief)) },
		texts:      func(payload any) []Text { return payload.(*CRBrief).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*CRBrief).OpenGaps },
		projection: openDesignProjection,
	}},
	{TypeImpactAssessment, formSpec{
		payload:    func() any { return &ImpactAssessment{} },
		validate:   func(payload any, _ Status) error { return ValidateImpactAssessment(*payload.(*ImpactAssessment)) },
		texts:      func(payload any) []Text { return payload.(*ImpactAssessment).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*ImpactAssessment).OpenGaps },
		projection: openDesignProjection,
	}},
	{TypeImplementationSpec, formSpec{
		payload:    func() any { return &ImplementationSpec{} },
		validate:   func(payload any, _ Status) error { return ValidateImplementationSpec(*payload.(*ImplementationSpec)) },
		texts:      func(payload any) []Text { return payload.(*ImplementationSpec).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*ImplementationSpec).OpenGaps },
		projection: openDesignProjection,
	}},
	{TypeVerdict, formSpec{
		payload:    func() any { return &VerdictReport{} },
		validate:   func(payload any, _ Status) error { return ValidateVerdictReport(*payload.(*VerdictReport)) },
		texts:      func(payload any) []Text { return payload.(*VerdictReport).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*VerdictReport).OpenGaps },
		projection: openDesignProjection,
	}},
	{TypeRefactorSpec, formSpec{
		payload:    func() any { return &RefactorSpec{} },
		validate:   func(payload any, _ Status) error { return ValidateRefactorSpec(*payload.(*RefactorSpec)) },
		texts:      func(payload any) []Text { return payload.(*RefactorSpec).Texts() },
		gaps:       func(payload any) []Gap { return payload.(*RefactorSpec).OpenGaps },
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
