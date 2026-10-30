package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationError names the field that broke a rule.
type ValidationError struct {
	Field   string
	Problem string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Problem }

func invalid(field, problem string) error { return &ValidationError{Field: field, Problem: problem} }

// ResolvedRef is a reference together with the hash of the stored form.
// The use case layer resolves references through its store port, so the domain
// stays free of I/O.
type ResolvedRef struct {
	Ref         Ref
	PayloadHash string
}

var knownFormTypes = map[FormType]bool{
	TypeUserInput: true, TypeBRD: true, TypeFeasibility: true, TypeCRBrief: true,
	TypeImpactAssessment: true, TypeSRS: true, TypeHLD: true, TypeADR: true, TypeLLD: true,
	TypeTaskPlan: true, TypeTestContract: true, TypeImplementationSpec: true,
	TypeVerdict: true, TypeRefactorSpec: true, TypeQuestion: true,
}

var knownStages = map[string]bool{
	"00": true, "01": true, "02": true, "03": true, "04": true, "05": true,
	"06": true, "07": true, "08": true, "09": true, "08.R": true, "—": true,
}

// Valid reports whether the status is known.
func (s Status) Valid() bool { return knownStatus(s) }

// Valid reports whether the form type is known.
func (t FormType) Valid() bool { return knownFormTypes[t] }

func knownStatus(s Status) bool {
	switch s {
	case StatusDraft, StatusValid, StatusWaitingOperator, StatusApproved, StatusRejected, StatusSuperseded:
		return true
	}
	return false
}

func knownRoute(r Route) bool {
	return r == RouteGreenfield || r == RouteChange || r == RouteMinorChange || r == RouteMajorChange
}

func knownMode(m Mode) bool { return m == ModeFull || m == ModeDelta }

func knownRole(r Role) bool {
	switch r {
	case RoleOperator, RoleAnalyst, RoleArchitect, RoleQA, RoleDev, RoleDevOps:
		return true
	}
	return false
}

// ValidateEnvelope checks the shared fields, the payload hash and the resolved inputs.
func (f *Form) ValidateEnvelope(resolved []ResolvedRef) error {
	var errs []error

	if f.FormID == "" {
		errs = append(errs, invalid("formId", "is empty"))
	}
	if !knownFormTypes[f.FormType] {
		errs = append(errs, invalid("formType", fmt.Sprintf("unknown value %q", f.FormType)))
	}
	if f.SchemaVersion < 1 {
		errs = append(errs, invalid("schemaVersion", "must be at least 1"))
	}
	if f.FormVersion < 1 {
		errs = append(errs, invalid("formVersion", "must be at least 1"))
	}
	if f.FormVersion > 1 && f.PreviousFormVersion == nil {
		errs = append(errs, invalid("previousFormVersion", "is required when formVersion is above 1"))
	}
	if !knownStatus(f.Status) {
		errs = append(errs, invalid("status", fmt.Sprintf("unknown value %q", f.Status)))
	}
	if !knownRoute(f.Route) {
		errs = append(errs, invalid("route", fmt.Sprintf("unknown value %q", f.Route)))
	}
	if !knownStages[f.Stage] {
		errs = append(errs, invalid("stage", fmt.Sprintf("unknown value %q", f.Stage)))
	}
	if !knownMode(f.Mode) {
		errs = append(errs, invalid("mode", fmt.Sprintf("unknown value %q", f.Mode)))
	}
	switch f.Mode {
	case ModeDelta:
		if f.Base == nil {
			errs = append(errs, invalid("base", "is required when mode is delta"))
		}
		if len(f.Ops) == 0 {
			errs = append(errs, invalid("ops", "is required when mode is delta"))
		}
	case ModeFull:
		if f.Base != nil {
			errs = append(errs, invalid("base", "must be empty when mode is full"))
		}
	}
	if f.PipelineID == "" {
		errs = append(errs, invalid("pipelineId", "is empty"))
	}
	if f.TargetSystemID == "" {
		errs = append(errs, invalid("targetSystemId", "is empty"))
	}
	if !knownRole(f.ProducedBy) {
		errs = append(errs, invalid("producedBy", fmt.Sprintf("unknown value %q", f.ProducedBy)))
	}
	if f.PayloadHash == "" {
		errs = append(errs, invalid("payloadHash", "is empty"))
	} else if hash, err := f.ComputeHash(); err != nil {
		errs = append(errs, fmt.Errorf("payloadHash: %w", err))
	} else if hash != f.PayloadHash {
		errs = append(errs, invalid("payloadHash", "does not match the payload"))
	}

	hashes := map[string]string{}
	for _, item := range resolved {
		hashes[referenceKey(item.Ref.FormID, item.Ref.FormVersion)] = item.PayloadHash
	}

	for i, ref := range f.Inputs {
		if ref.FormID == "" || ref.FormVersion < 1 {
			errs = append(errs, invalid(fmt.Sprintf("inputs[%d]", i), "formId and formVersion are required"))
			continue
		}
		hash, ok := hashes[referenceKey(ref.FormID, ref.FormVersion)]
		if !ok {
			errs = append(errs, invalid(fmt.Sprintf("inputs[%d]", i), fmt.Sprintf("reference %s:%d is not resolved", ref.FormID, ref.FormVersion)))
			continue
		}
		if ref.Hash != "" && ref.Hash != hash {
			errs = append(errs, invalid(fmt.Sprintf("inputs[%d].hash", i), "does not match the referenced form"))
		}
	}

	if f.Base != nil {
		hash, ok := hashes[referenceKey(f.Base.FormID, f.Base.FormVersion)]
		switch {
		case !ok:
			errs = append(errs, invalid("base", fmt.Sprintf("reference %s:%d is not resolved", f.Base.FormID, f.Base.FormVersion)))
		case f.Base.Hash != "" && f.Base.Hash != hash:
			errs = append(errs, invalid("base.hash", "does not match the referenced form"))
		}
	}

	return errors.Join(errs...)
}

// ValidateUserInput checks the USER_INPUT gates.
func ValidateUserInput(in UserInput) error {
	var errs []error
	if in.RawRequest.Value == "" {
		errs = append(errs, invalid("rawRequest", "is empty"))
	}
	switch in.Source {
	case SourceOperator, SourceAPI, SourceIssue:
	default:
		errs = append(errs, invalid("source", fmt.Sprintf("unknown value %q", in.Source)))
	}
	if _, err := in.ParseReceivedAt(); err != nil {
		errs = append(errs, invalid("receivedAt", "must be an RFC3339 value"))
	}
	if in.Locale != "" && in.Locale != LocaleEN && in.Locale != LocaleRU {
		errs = append(errs, invalid("locale", fmt.Sprintf("unknown value %q", in.Locale)))
	}
	return errors.Join(errs...)
}

var brdBannedTerms = []string{"container", "module", "package", "struct", "database", "table"}

func ValidateBRD(brd BRD, status Status) error {
	var errs []error

	// A draft that waits for operator answers is judged on shape only: the
	// answers fill the lists, and the completeness gates run when the document
	// is assembled.
	waiting := status == StatusDraft && len(brd.OpenGaps) > 0
	if !waiting && len(brd.Goals) == 0 {
		errs = append(errs, invalid("goals", "must contain at least one entry"))
	}
	// Metrics are optional: a greenfield system has no measured value yet, so a
	// goal may name no metric. A metric that is given still needs its ID, and every
	// reference to a metric must resolve; an unknown ID is a mistake, not a gap.
	metricIDs := map[string]bool{}
	for i, metric := range brd.Metrics {
		if metric.ID == "" {
			errs = append(errs, invalid(fmt.Sprintf("metrics[%d].id", i), "is empty"))
			continue
		}
		metricIDs[metric.ID] = true
	}

	for i, goal := range brd.Goals {
		if goal.Outcome.Value == "" {
			errs = append(errs, invalid(fmt.Sprintf("goals[%d].outcome", i), "is empty"))
		}
		if goal.MetricID != "" && !metricIDs[goal.MetricID] {
			errs = append(errs, invalid(fmt.Sprintf("goals[%d].metricId", i), "does not exist in metrics"))
		}
	}

	for i, success := range brd.SuccessMetrics {
		if success.MetricID != "" && !metricIDs[success.MetricID] {
			errs = append(errs, invalid(fmt.Sprintf("successMetrics[%d].metricId", i), "does not exist in metrics"))
		}
	}

	// The blocking-gap gate is shared by every question-phase form and lives in
	// ValidatePayloadWithStatus, so it does not duplicate here.
	for i, text := range brd.Texts() {
		for _, term := range brdBannedTerms {
			if containsWord(text.Value, term) {
				errs = append(errs, invalid(fmt.Sprintf("texts[%d]", i), fmt.Sprintf("contains the design term %q", term)))
			}
		}
	}

	return errors.Join(errs...)
}

func referenceKey(formID string, version int) string { return fmt.Sprintf("%s:%d", formID, version) }

func containsWord(value, word string) bool {
	for _, token := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return r < 'a' || r > 'z'
	}) {
		if token == word {
			return true
		}
	}
	return false
}
