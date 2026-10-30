package usecase

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// The registry file name is derived one way for every form type, so an
// added form type reaches the model through this rule or not at all.
func TestRegistryFileNameFollowsTheFormType(t *testing.T) {
	cases := map[domain.FormType]string{
		domain.TypeBRD:              "brd.md",
		domain.TypeSRS:              "srs.md",
		domain.TypeFeasibility:      "feasibility.md",
		domain.TypeCRBrief:          "cr-brief.md",
		domain.TypeImpactAssessment: "impact-assessment.md",
		domain.TypeTestContract:     "test-contract.md",
	}
	for formType, want := range cases {
		if got := registryFileName(formType); got != want {
			t.Fatalf("registry file of %s: got %q, want %q", formType, got, want)
		}
	}
}

// TestDetailAndQuestionPromptsDoNotShipTheWholeShape: the detail call answers one
// entry and the interview answers questions. The full payload shape in the system
// prompt made the model retype the whole document on every call.
func TestDetailAndQuestionPromptsDoNotShipTheWholeShape(t *testing.T) {
	system, user := detailMessages(domain.TypeLLD, "instruction", json.RawMessage(`{}`), json.RawMessage(`{"containers":[]}`))
	if strings.Contains(system, "The payload shape is") {
		t.Fatalf("the detail call must not ship the whole shape: %s", system)
	}
	if !strings.Contains(system, `"entry"`) || !strings.Contains(system, `"openGaps"`) {
		t.Fatalf("the detail call must ask for the entry and its questions: %s", system)
	}
	if !strings.Contains(user, "The form so far (read only") {
		t.Fatalf("the detail call reads the whole form: %s", user)
	}

	questions, _ := questionsMessages("instruction", json.RawMessage(`{}`))
	if strings.Contains(questions, "The payload shape is") || !strings.Contains(questions, "openGaps") {
		t.Fatalf("the interview asks questions only: %s", questions)
	}
}

// TestBatchPlansCoverEveryStage: every stage that writes a list of entries uses
// the same skeleton-then-detail walk, so the model lists the entries first and
// answers each one alone afterwards.
func TestBatchPlansCoverEveryStage(t *testing.T) {
	for _, formType := range []domain.FormType{
		domain.TypeSRS, domain.TypeFeasibility, domain.TypeADR,
		domain.TypeHLD, domain.TypeLLD, domain.TypeTaskPlan, domain.TypeTestContract,
	} {
		plans, ok := batchPlans[formType]
		if !ok || len(plans) == 0 {
			t.Fatalf("%s must be filled entry by entry", formType)
		}
		shape, err := domain.PayloadShape(formType)
		if err != nil {
			t.Fatalf("%s: shape: %v", formType, err)
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(shape), &doc); err != nil {
			t.Fatalf("%s: decode shape: %v", formType, err)
		}
		for _, plan := range plans {
			if _, ok := doc[plan.kind].([]any); !ok {
				t.Fatalf("%s: %q is not a list of the shape", formType, plan.kind)
			}
		}
	}
}

// TestStageQuestionPolicyDeepensByStage: the same subject returns at the level the
// design has reached, and no stage asks about the schema the system writes.
func TestStageQuestionPolicyDeepensByStage(t *testing.T) {
	cases := map[domain.FormType]string{
		domain.TypeBRD: "what each business word means",
		domain.TypeSRS: "what goes wrong",
		domain.TypeHLD: "external system or device",
		domain.TypeLLD: "business error",
	}
	for formType, want := range cases {
		got := stageQuestionPolicy(formType)
		if !strings.Contains(got, want) {
			t.Fatalf("%s: the policy misses %q: %s", formType, want, got)
		}
		if strings.Contains(got, "inputType") && formType != domain.TypeLLD {
			t.Fatalf("%s: the policy leaks a schema field: %s", formType, got)
		}
	}
}

// TestSkeletonInstructionAsksForTheRequirements: the SRS outline is one fr per
// step of the user flow of every actor, never an empty pair of lists. An empty
// answer leaves the stage with nothing to ask and no document.
func TestSkeletonInstructionAsksForTheRequirements(t *testing.T) {
	got := skeletonInstruction(domain.Stage{ID: "02"}, domain.TypeSRS)
	for _, want := range []string{
		"List every entry of fr and nfr",
		"one fr entry for EVERY step of the real user flow of EVERY actor",
		"actors[].tasks[1]",
		"actors[].tasks[2]",
		"the payment the BRD assumptions name",
		"the notification the BRD assumptions name",
		"Do not return empty fr and nfr lists",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("the SRS outline instruction misses %q: %s", want, got)
		}
	}
}

// TestSkeletonInstructionNamesTheInfrastructure: a distributed system needs its
// gateway, identity provider and message bus as containers, not as a box inside
// an application.
func TestSkeletonInstructionNamesTheInfrastructure(t *testing.T) {
	got := skeletonInstruction(domain.Stage{ID: "03"}, domain.TypeHLD)
	for _, want := range []string{
		"API Gateway",
		"Keycloak",
		"Kafka",
		"load balancer",
		"scheduler",
		"the sign-in and the token",
		"from the publisher to the message bus",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("the HLD outline instruction misses %q: %s", want, got)
		}
	}
}

// TestQuestionsInstructionInterviewsTheSubject: the first BRD pass is a domain
// interview. Every role and every capability gets its own question, the model may
// not merge them, and it never asks the operator for an internal name or baseline.
func TestQuestionsInstructionInterviewsTheSubject(t *testing.T) {
	got := questionsInstruction(domain.Stage{ID: "00"}, domain.TypeBRD)
	for _, want := range []string{
		"/actors/0/name", "/actors/1/name",
		"/scope/in/0",
		"Never merge two roles",
		"Do not ask for an id, an internal name, a unit or a baseline",
		"/glossary/0",
		"/assumptions/0",
		"/assumptions/1",
		"/actors/<i>/tasks/1",
		"/actors/<i>/tasks/2",
		"personal cabinet",
		"payment terminal",
		"never contains the characters < or >",
		"never reads like the question for the next one",
		"Which name does the receptionist have in this business?",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("the interview misses %q: %s", want, got)
		}
	}
}
