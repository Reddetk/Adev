package domain

import (
	"errors"
	"fmt"
)

// ChangeType is the class of a change request.
type ChangeType string

const (
	ChangeBusiness  ChangeType = "business"
	ChangeDevSecOps ChangeType = "devsecops"
	ChangeTechDebt  ChangeType = "techdebt"
	ChangeAudit     ChangeType = "audit"
)

// Behavior is the observable behavior of the system, named by registry objects.
type Behavior struct {
	ObjectID string `json:"objectId,omitempty"`
	Input    string `json:"input"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
}

// RequestedBehavior is the behavior that must exist after the change.
type RequestedBehavior struct {
	Input        string `json:"input"`
	Output       string `json:"output"`
	RemovedError string `json:"removedError,omitempty"`
	NewError     string `json:"newError,omitempty"`
}

// CRBrief is the payload of a change request brief.
type CRBrief struct {
	ChangeType        ChangeType        `json:"changeType"`
	Driver            Text              `json:"driver"`
	CurrentBehavior   Behavior          `json:"currentBehavior"`
	RequestedBehavior RequestedBehavior `json:"requestedBehavior"`
	OpenGaps          []Gap             `json:"openGaps,omitempty"`
}

// Texts returns every free text the language checker reads.
func (c CRBrief) Texts() []Text { return []Text{c.Driver} }

// ValidateCRBrief checks the gates of the change request brief.
func ValidateCRBrief(c CRBrief) error {
	var errs []error
	switch c.ChangeType {
	case ChangeBusiness, ChangeDevSecOps, ChangeTechDebt, ChangeAudit:
	default:
		errs = append(errs, invalid("changeType", fmt.Sprintf("unknown value %q", c.ChangeType)))
	}
	if c.Driver.Value == "" {
		errs = append(errs, invalid("driver", "is empty"))
	}
	if c.CurrentBehavior.Input == "" || c.CurrentBehavior.Output == "" {
		errs = append(errs, invalid("currentBehavior", "input and output must be non-empty"))
	}
	if c.RequestedBehavior.Input == "" || c.RequestedBehavior.Output == "" {
		errs = append(errs, invalid("requestedBehavior", "input and output must be non-empty"))
	}
	return errors.Join(errs...)
}

// BlastRadius is one object the change touches.
type BlastRadius struct {
	ObjectID   string `json:"objectId"`
	ChangeKind string `json:"changeKind"` // create | change | delete
	Depth      string `json:"depth"`
	Reason     Text   `json:"reason"`
}

// ContractImpact is one interface, event or consumer that the change breaks.
type ContractImpact struct {
	ContractID    string   `json:"contractId"`
	EventName     string   `json:"eventName,omitempty"`
	ChangeType    string   `json:"changeType"`
	Consumers     []string `json:"consumers"`
	Compatibility string   `json:"compatibility"`
}

// NFRBaselineDelta is the measured value before and after the change.
type NFRBaselineDelta struct {
	NFRID        string `json:"nfrId"`
	Current      string `json:"current"`
	Required     string `json:"required"`
	Unit         string `json:"unit"`
	VerifyMethod string `json:"verifyMethod"`
}

// ChangeRouting is the branch that the change must take.
type ChangeRouting struct {
	Route  string `json:"route"` // A | B
	RuleID string `json:"ruleId"`
	Reason Text   `json:"reason"`
}

// ImpactAssessment is the payload of the change impact assessment.
type ImpactAssessment struct {
	BlastRadius      []BlastRadius      `json:"blastRadius"`
	ContractImpact   []ContractImpact   `json:"contractImpact"`
	NFRBaselineDelta []NFRBaselineDelta `json:"nfrBaselineDelta"`
	Routing          ChangeRouting      `json:"routing"`
	OpenGaps         []Gap              `json:"openGaps,omitempty"`
}

// Texts returns every free text the language checker reads.
func (i ImpactAssessment) Texts() []Text {
	out := []Text{i.Routing.Reason}
	for _, entry := range i.BlastRadius {
		out = append(out, entry.Reason)
	}
	return out
}

// ValidateImpactAssessment checks the gates of the impact assessment.
func ValidateImpactAssessment(i ImpactAssessment) error {
	var errs []error
	if len(i.BlastRadius) == 0 {
		errs = append(errs, invalid("blastRadius", "must contain at least one entry"))
	}
	for index, entry := range i.BlastRadius {
		if entry.ObjectID == "" {
			errs = append(errs, invalid(fmt.Sprintf("blastRadius[%d].objectId", index), "is empty"))
		}
		switch entry.ChangeKind {
		case "create", "change", "delete":
		default:
			errs = append(errs, invalid(fmt.Sprintf("blastRadius[%d].changeKind", index), fmt.Sprintf("unknown value %q", entry.ChangeKind)))
		}
	}
	switch i.Routing.Route {
	case "A", "B":
	default:
		errs = append(errs, invalid("routing.route", fmt.Sprintf("unknown value %q", i.Routing.Route)))
	}
	if i.Routing.RuleID == "" {
		errs = append(errs, invalid("routing.ruleId", "is empty"))
	}
	return errors.Join(errs...)
}

// ImplementationTarget is one object the implementation touches.
type ImplementationTarget struct {
	Target    TargetRef `json:"target"`
	Action    string    `json:"action"`    // create | change | delete
	Generator string    `json:"generator"` // codegen | configgen | manualException
}

// ImplementationException is the operator approval for model-written code.
type ImplementationException struct {
	Reason      Text `json:"reason"`
	ApprovalRef Ref  `json:"approvalRef"`
}

// ImplementationStep is one ordered action for one target.
type ImplementationStep struct {
	Order          int    `json:"order"`
	TargetObjectID string `json:"targetObjectId"`
	Action         string `json:"action"`
	Text           Text   `json:"text"`
}

// TestMapEntry links one test of the contract to one step.
type TestMapEntry struct {
	TestID         string `json:"testId"`
	TargetObjectID string `json:"targetObjectId"`
	StepOrder      int    `json:"stepOrder"`
}

// ImplementationSpec is the payload of stage 08.
type ImplementationSpec struct {
	TaskRef         string                   `json:"taskRef"`
	TestContractRef Ref                      `json:"testContractRef"`
	Targets         []ImplementationTarget   `json:"targets"`
	Exception       *ImplementationException `json:"exception,omitempty"`
	Steps           []ImplementationStep     `json:"steps"`
	Constraints     []Statement              `json:"constraints,omitempty"`
	TestMap         []TestMapEntry           `json:"testMap"`
	OpenGaps        []Gap                    `json:"openGaps,omitempty"`
}

// Texts returns every free text the language checker reads.
func (s ImplementationSpec) Texts() []Text {
	out := make([]Text, 0, len(s.Constraints)+len(s.Steps))
	for _, constraint := range s.Constraints {
		out = append(out, Text{Value: constraint.Text, Lang: LangACE})
	}
	for _, step := range s.Steps {
		out = append(out, step.Text)
	}
	return out
}

var knownActions = map[string]bool{"create": true, "change": true, "delete": true}

// ValidateImplementationSpec checks the gates of the implementation spec that one
// document can check; the test coverage gate needs the TEST_CONTRACT document.
func ValidateImplementationSpec(s ImplementationSpec) error {
	var errs []error
	if len(s.Targets) == 0 {
		errs = append(errs, invalid("targets", "must contain at least one entry"))
	}
	targetIDs := map[string]bool{}
	for index, target := range s.Targets {
		field := fmt.Sprintf("targets[%d]", index)
		if target.Target.ObjectID == "" && !target.Target.New {
			errs = append(errs, invalid(field+".target", "needs objectId or new"))
		}
		if target.Target.ObjectID != "" {
			targetIDs[target.Target.ObjectID] = true
		}
		if !knownActions[target.Action] {
			errs = append(errs, invalid(field+".action", fmt.Sprintf("unknown value %q", target.Action)))
		}
		switch target.Generator {
		case "codegen", "configgen":
		case "manualException":
			if target.Target.Kind == "test" {
				errs = append(errs, invalid(field+".generator", "cannot write a test by hand"))
			}
			if s.Exception == nil || s.Exception.ApprovalRef.FormID == "" {
				errs = append(errs, invalid(field+".generator", "manualException needs an approved exception"))
			}
		default:
			errs = append(errs, invalid(field+".generator", fmt.Sprintf("unknown value %q", target.Generator)))
		}
	}
	if len(s.Steps) == 0 {
		errs = append(errs, invalid("steps", "must contain at least one entry"))
	}
	orders := map[int]bool{}
	for index, step := range s.Steps {
		field := fmt.Sprintf("steps[%d]", index)
		if step.Order < 1 {
			errs = append(errs, invalid(field+".order", "must be positive"))
		}
		orders[step.Order] = true
		if step.TargetObjectID == "" {
			errs = append(errs, invalid(field+".targetObjectId", "is empty"))
		} else if !targetIDs[step.TargetObjectID] {
			errs = append(errs, invalid(field+".targetObjectId", "does not exist in targets"))
		}
		if !knownActions[step.Action] {
			errs = append(errs, invalid(field+".action", fmt.Sprintf("unknown value %q", step.Action)))
		}
	}
	for index, entry := range s.TestMap {
		field := fmt.Sprintf("testMap[%d]", index)
		if entry.TestID == "" {
			errs = append(errs, invalid(field+".testId", "is empty"))
		}
		if !orders[entry.StepOrder] {
			errs = append(errs, invalid(field+".stepOrder", "does not exist in steps"))
		}
	}
	return errors.Join(errs...)
}

// TestStatus is the result of one test run.
type TestStatus string

const (
	TestPass    TestStatus = "pass"
	TestFail    TestStatus = "fail"
	TestSkipped TestStatus = "skipped"
)

// TestResult is the result of one test.
type TestResult struct {
	TestID     string     `json:"testId"`
	Status     TestStatus `json:"status"`
	DurationMs int        `json:"durationMs"`
	Evidence   Ref        `json:"evidence"`
}

// Violation is one rule that the implementation breaks.
type Violation struct {
	TestID   string `json:"testId,omitempty"`
	Aspect   string `json:"aspect"`
	Severity string `json:"severity"` // blocker | major | minor
	Evidence Ref    `json:"evidence"`
}

// VerdictReport is the payload of stage 09.
type VerdictReport struct {
	TaskRef           string       `json:"taskRef"`
	ImplementationRef Ref          `json:"implementationRef"`
	RunID             string       `json:"runId"`
	StartedAt         string       `json:"startedAt"`
	FinishedAt        string       `json:"finishedAt"`
	Results           []TestResult `json:"results"`
	Violations        []Violation  `json:"violations"`
	Verdict           string       `json:"verdict"` // pass | fail
	Reason            Text         `json:"reason"`
	Evidence          []Ref        `json:"evidence"`
	OpenGaps          []Gap        `json:"openGaps,omitempty"`
}

// Texts returns every free text the language checker reads.
func (v VerdictReport) Texts() []Text { return []Text{v.Reason} }

var knownSeverities = map[string]bool{"blocker": true, "major": true, "minor": true}

// ValidateVerdictReport checks the gates of the verdict that one document can
// check; the test set gate needs the TEST_CONTRACT document.
func ValidateVerdictReport(v VerdictReport) error {
	var errs []error
	if v.TaskRef == "" || v.RunID == "" {
		errs = append(errs, invalid("taskRef", "taskRef and runId must be non-empty"))
	}
	if len(v.Results) == 0 {
		errs = append(errs, invalid("results", "must contain at least one entry"))
	}
	for index, result := range v.Results {
		field := fmt.Sprintf("results[%d]", index)
		switch result.Status {
		case TestPass:
		case TestFail, TestSkipped:
			if v.Verdict == "pass" {
				errs = append(errs, invalid(field+".status", "a pass verdict needs every test to pass"))
			}
		default:
			errs = append(errs, invalid(field+".status", fmt.Sprintf("unknown value %q", result.Status)))
		}
		if result.Evidence.FormID == "" && result.Evidence.Hash == "" {
			errs = append(errs, invalid(field+".evidence", "is empty"))
		}
	}
	for index, violation := range v.Violations {
		if !knownSeverities[violation.Severity] {
			errs = append(errs, invalid(fmt.Sprintf("violations[%d].severity", index), fmt.Sprintf("unknown value %q", violation.Severity)))
		}
		if v.Verdict == "pass" {
			errs = append(errs, invalid("violations", "must be empty for a pass verdict"))
		}
	}
	switch v.Verdict {
	case "pass":
		if len(v.Violations) > 0 {
			errs = append(errs, invalid("verdict", "a pass verdict has no violation"))
		}
	case "fail":
		if len(v.Violations) == 0 {
			errs = append(errs, invalid("violations", "a fail verdict names at least one violation"))
		}
	default:
		errs = append(errs, invalid("verdict", fmt.Sprintf("unknown value %q", v.Verdict)))
	}
	return errors.Join(errs...)
}

// RefactorViolation is one violation that the refactor must close.
type RefactorViolation struct {
	ViolationID string `json:"violationId"`
	TestID      string `json:"testId"`
	Severity    string `json:"severity"`
}

// RefactorStep is one ordered action that closes a violation.
type RefactorStep struct {
	Order             int    `json:"order"`
	TargetObjectID    string `json:"targetObjectId"`
	Action            string `json:"action"`
	Text              Text   `json:"text"`
	ClosesViolationID string `json:"closesViolationId"`
}

// RefactorSpec is the payload of stage 08.R.
type RefactorSpec struct {
	VerdictRef        Ref                 `json:"verdictRef"`
	TaskRef           string              `json:"taskRef"`
	ImplementationRef Ref                 `json:"implementationRef"`
	Violations        []RefactorViolation `json:"violations"`
	AllowedFiles      []TargetRef         `json:"allowedFiles"`
	Steps             []RefactorStep      `json:"steps"`
	PreserveTests     bool                `json:"preserveTests"`
	MustPass          []string            `json:"mustPass"`
	OpenGaps          []Gap               `json:"openGaps,omitempty"`
}

// Texts returns every free text the language checker reads.
func (r RefactorSpec) Texts() []Text {
	out := make([]Text, 0, len(r.Steps))
	for _, step := range r.Steps {
		out = append(out, step.Text)
	}
	return out
}

// ValidateRefactorSpec checks the gates of the refactor spec that one document can
// check; the closing-step gate needs the violated verdict.
func ValidateRefactorSpec(r RefactorSpec) error {
	var errs []error
	if !r.PreserveTests {
		errs = append(errs, invalid("preserveTests", "must be true: a refactor does not rewrite the tests"))
	}
	if len(r.AllowedFiles) == 0 {
		errs = append(errs, invalid("allowedFiles", "must contain at least one entry"))
	}
	for index, file := range r.AllowedFiles {
		if file.Kind == "test" {
			errs = append(errs, invalid(fmt.Sprintf("allowedFiles[%d]", index), "a refactor may not change a test"))
		}
		if file.ObjectID == "" && !file.New {
			errs = append(errs, invalid(fmt.Sprintf("allowedFiles[%d]", index), "needs objectId or new"))
		}
	}
	if len(r.Violations) == 0 {
		errs = append(errs, invalid("violations", "must contain at least one entry"))
	}
	closed := map[string]bool{}
	for index, step := range r.Steps {
		field := fmt.Sprintf("steps[%d]", index)
		if step.Order < 1 {
			errs = append(errs, invalid(field+".order", "must be positive"))
		}
		if step.TargetObjectID == "" {
			errs = append(errs, invalid(field+".targetObjectId", "is empty"))
		}
		if !knownActions[step.Action] {
			errs = append(errs, invalid(field+".action", fmt.Sprintf("unknown value %q", step.Action)))
		}
		if step.ClosesViolationID != "" {
			closed[step.ClosesViolationID] = true
		}
	}
	for index, violation := range r.Violations {
		if !closed[violation.ViolationID] {
			errs = append(errs, invalid(fmt.Sprintf("violations[%d]", index), "no step closes this violation"))
		}
	}
	if len(r.MustPass) == 0 {
		errs = append(errs, invalid("mustPass", "must contain at least one test id"))
	}
	return errors.Join(errs...)
}
