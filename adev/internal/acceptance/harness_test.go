//go:build tdd

// Package acceptance holds the TDD tests of docs/refactor-plan.md. Every test
// names the PR whose acceptance criterion it checks. The runs use the real
// domain, use case and HTTP layers over the in-memory stores, so a test breaks
// when the behavior of a PR regresses, not only when a helper changes.
//
// The build tag keeps `make test` green while the plan is open:
//
//	go test ./...                        # green today
//	go test -tags tdd ./internal/acceptance/...   # what each PR still owes
package acceptance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/repository"
	"github.com/Reddetk/Adev/internal/usecase"
)

const (
	testPipeline = "1"
	testTarget   = int64(1)
)

func ste(value string) domain.Text { return domain.Text{Value: value, Lang: domain.LangSTE100} }

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return raw
}

// scriptedLLM records every request and answers with the payload that respond picks.
type scriptedLLM struct {
	respond  func(usecase.LLMRequest) (usecase.LLMResponse, error)
	requests []usecase.LLMRequest
}

func (s *scriptedLLM) Complete(_ context.Context, request usecase.LLMRequest) (usecase.LLMResponse, error) {
	s.requests = append(s.requests, request)
	return s.respond(request)
}

// respondWith serves one payload per output form type. The stage instruction names
// the form type, so a stage with several outputs gets its own answer per output.
func respondWith(t *testing.T, byOutput map[domain.FormType]json.RawMessage) func(usecase.LLMRequest) (usecase.LLMResponse, error) {
	t.Helper()
	return func(request usecase.LLMRequest) (usecase.LLMResponse, error) {
		for formType, payload := range byOutput {
			if strings.Contains(request.User, fmt.Sprintf("the %s form", formType)) {
				return usecase.LLMResponse{Payload: payload}, nil
			}
		}
		return usecase.LLMResponse{}, fmt.Errorf("acceptance: no scripted answer in %q", request.User)
	}
}

// fakePipelines is the in-process PipelineRepository of the acceptance runs.
type fakePipelines struct{ pipeline *domain.Pipeline }

func (f *fakePipelines) Save(pipeline *domain.Pipeline) error { f.pipeline = pipeline; return nil }
func (f *fakePipelines) FindByID(int) (*domain.Pipeline, error) {
	if f.pipeline == nil {
		return nil, errors.New("pipeline not found")
	}
	return f.pipeline, nil
}
func (f *fakePipelines) FindByTargetSystemID(int) (*domain.Pipeline, error) {
	return nil, errors.New("pipeline not found")
}
func (f *fakePipelines) FindByRoute(string) (*domain.Pipeline, error) {
	return nil, errors.New("pipeline not found")
}
func (f *fakePipelines) FindByStatus(string) (*domain.Pipeline, error) {
	return nil, errors.New("pipeline not found")
}
func (f *fakePipelines) FindByCurrentStage(string) (*domain.Pipeline, error) {
	return nil, errors.New("pipeline not found")
}
func (f *fakePipelines) Update(pipeline *domain.Pipeline) error { f.pipeline = pipeline; return nil }
func (f *fakePipelines) Delete(int) error                       { return nil }
func (f *fakePipelines) FindAll() ([]domain.Pipeline, error)    { return nil, nil }

// fakeRequests is the in-process OperatorRequestRepository.
type fakeRequests struct{ saved []domain.OperatorRequest }

func (f *fakeRequests) Save(request *domain.OperatorRequest) error {
	request.ID = uint(len(f.saved) + 1)
	f.saved = append(f.saved, *request)
	return nil
}
func (f *fakeRequests) Update(request *domain.OperatorRequest) error {
	if request.ID == 0 || int(request.ID) > len(f.saved) {
		return domain.ErrFormNotFound
	}
	f.saved[request.ID-1] = *request
	return nil
}
func (f *fakeRequests) FindByPointer(pointer string) (*domain.OperatorRequest, error) {
	for index := range f.saved {
		if f.saved[index].Pointer == pointer {
			return &f.saved[index], nil
		}
	}
	return nil, domain.ErrFormNotFound
}
func (f *fakeRequests) FindByID(int) (*domain.OperatorRequest, error) {
	return nil, domain.ErrFormNotFound
}
func (f *fakeRequests) FindByPipelineID(int) (*domain.OperatorRequest, error) {
	return nil, domain.ErrFormNotFound
}
func (f *fakeRequests) FindByQuestion(string) (*domain.OperatorRequest, error) {
	return nil, domain.ErrFormNotFound
}
func (f *fakeRequests) FindByStatus(string) (*domain.OperatorRequest, error) {
	return nil, domain.ErrFormNotFound
}
func (f *fakeRequests) Delete(int) error                           { return nil }
func (f *fakeRequests) FindAll() ([]domain.OperatorRequest, error) { return f.saved, nil }

// fixture wires the layers the same way the composition root does.
type fixture struct {
	store        *repository.MemoryFormsStore
	forms        *usecase.FormService
	settings     *usecase.SettingsService
	llm          *scriptedLLM
	requests     *fakeRequests
	pipelines    *fakePipelines
	stages       *usecase.StageService
	questions    *usecase.QuestionService
	orchestrator *usecase.Orchestrator
}

func newFixture(t *testing.T, respond func(usecase.LLMRequest) (usecase.LLMResponse, error)) *fixture {
	t.Helper()
	if respond == nil {
		respond = func(usecase.LLMRequest) (usecase.LLMResponse, error) {
			return usecase.LLMResponse{}, errors.New("acceptance: the model must not be called")
		}
	}
	f := &fixture{
		store:     repository.NewMemoryFormsStore(),
		settings:  usecase.NewSettingsService(repository.NewMemoryAgentConfigStore(testTarget)),
		llm:       &scriptedLLM{respond: respond},
		requests:  &fakeRequests{},
		pipelines: &fakePipelines{pipeline: &domain.Pipeline{ID: 1, TargetSystemID: int(testTarget), Route: string(domain.RouteGreenfield), Status: "active", CurrentStage: "00"}},
	}
	f.forms = usecase.NewFormService(f.store, nil)
	f.questions = usecase.NewQuestionService(f.forms, f.requests)
	f.stages = usecase.NewStageService(f.forms, f.settings, usecase.FixedLLM(f.llm), nil).WithQuestions(f.questions)
	f.orchestrator = usecase.NewOrchestrator(f.stages, f.pipelines)

	require.NoError(t, f.settings.SetWorkspace(context.Background(), testTarget, `D:\work\library`))
	for _, role := range []domain.Role{domain.RoleAnalyst, domain.RoleArchitect} {
		require.NoError(t, f.settings.SetAgent(context.Background(), domain.AgentConfig{
			TargetSystemID: testTarget, Role: role, Provider: domain.ProviderOllama,
			BaseURL: "http://localhost:11434", Model: "qwen3-coder:30b",
		}))
	}
	return f
}

func stageRequest(stage string) usecase.StageRequest {
	return usecase.StageRequest{
		PipelineID: testPipeline, TargetSystemID: testTarget,
		Route: domain.RouteGreenfield, StageID: stage,
	}
}

// run executes one stage through the stage service, without the cursor chain.
func (f *fixture) run(t *testing.T, stage string) usecase.StageOutcome {
	t.Helper()
	outcome, err := f.stages.Run(context.Background(), stageRequest(stage))
	require.NoError(t, err)
	return outcome
}

// submit stores a version-1 form that already passed its gates.
func (f *fixture) submit(t *testing.T, id string, formType domain.FormType, stage string, payload json.RawMessage) *domain.Form {
	t.Helper()
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: id, FormType: formType, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusValid, Route: domain.RouteGreenfield, Stage: stage, Mode: domain.ModeFull,
		PipelineID: testPipeline, TargetSystemID: "1", ProducedBy: domain.RoleAnalyst,
		Inputs: []domain.Ref{},
	}}
	require.NoError(t, form.SetRawPayload(payload))
	require.NoError(t, f.forms.Submit(context.Background(), form))
	return form
}

// saveDraft stores a form without running the payload gates.
func (f *fixture) saveDraft(t *testing.T, form *domain.Form) {
	t.Helper()
	require.NoError(t, f.store.Save(context.Background(), form))
}

// --- payload builders -------------------------------------------------------

func validBRD(t *testing.T) json.RawMessage {
	baseline, target := 90.0, 20.0
	return mustJSON(t, domain.BRD{
		Goals: []domain.Goal{{
			ID: "GOAL-1", Outcome: ste("The system shall speed up the lending desk."), MetricID: "M-1",
		}},
		Metrics: []domain.Metric{{
			ID: "M-1", Name: "lend_seconds", Unit: "second", Baseline: &baseline, Target: &target,
		}},
		Scope:          domain.Scope{In: []domain.Text{ste("Book registration")}},
		SuccessMetrics: []domain.Success{{MetricID: "M-1", Threshold: "20", Window: "30 day"}},
	})
}

func validFeasibility(t *testing.T) json.RawMessage {
	cost := domain.Cost{Currency: "EUR", Amount: 1000}
	return mustJSON(t, domain.Feasibility{
		ReportType: domain.ReportTEJ,
		Options: []domain.Option{{
			ID: "OPT-1", Summary: ste("Build the lending module."),
			Cost: &cost, ScheduleDays: 30, Confidence: domain.ConfidenceHigh,
		}},
		RecommendedOption: "OPT-1",
		Verdict:           domain.VerdictGo,
		Reason:            ste("The team can build it."),
	})
}

func validSRS(t *testing.T) json.RawMessage {
	return mustJSON(t, domain.SRS{
		FR: []domain.FunctionalRequirement{{
			ID:        "FR-1",
			Statement: domain.Statement{ID: "FR-1", Text: "The system shall register a book.", Modality: domain.ModalityShall, Priority: domain.PriorityMust},
			Actor:     "librarian", Trigger: "a book arrives", Response: "the system stores the book",
			StateChange: "book registered", Priority: domain.PriorityMust,
			Acceptance: []domain.Scenario{{Given: "a librarian", When: "a book arrives", Then: "the book is stored"}},
		}},
		NFR: []domain.NonFunctionalRequirement{{
			ID: "NFR-1", Statement: domain.Statement{ID: "NFR-1", Text: "The system shall answer in 200 ms."},
			Metric: "latency", Unit: "ms", Threshold: "200", MeasurementPoint: "API", Window: "p95",
		}},
	})
}

func validHLD(t *testing.T) json.RawMessage {
	return mustJSON(t, domain.HLD{
		BoundedContexts: []domain.BoundedContext{{ID: "bc-1", Name: "Lending", Purpose: ste("The desk lends books.")}},
		Containers: []domain.Container{{
			ObjectID: "obj-container-1", BoundedContextID: "bc-1",
			Purpose: ste("Serve the lending desk."), Tech: "go",
		}},
		Aggregates: []domain.Aggregate{{
			ID: "agg-1", ContainerID: "obj-container-1", Root: "Book", ModuleRef: "obj-module-1",
			Invariants: []domain.Statement{{ID: "INV-1", Text: "A book has one holder.", Modality: domain.ModalityShall, Priority: domain.PriorityMust}},
		}},
		ContextRelations: []domain.ContextRelation{{From: "bc-1", To: "bc-1", Type: domain.RelationSync}},
		DegradationStrategy: []domain.Degradation{{
			Failure: "the database is down", Degradation: "read only mode",
			Detection: "health check", TimeoutMs: 1000,
		}},
	})
}

func validADR(t *testing.T) json.RawMessage {
	return mustJSON(t, domain.ADR{Decisions: []domain.Decision{{
		ID: "ADR-1", Title: ste("Use PostgreSQL."), Status: domain.DecisionAccepted,
		Decision: ste("Store the lending data in PostgreSQL."),
		Alternatives: []domain.Alternative{{
			Option: ste("Store the data in files."), RejectedBecause: ste("The reports need a query language."),
		}},
	}}})
}

func validLLD(t *testing.T) json.RawMessage {
	return mustJSON(t, domain.LLD{
		Containers: []domain.LLDContainer{{
			ContainerID: "obj-container-1",
			Layers: []domain.Layer{{
				Name: domain.LayerApp,
				Packages: []domain.Package{{
					Path: "internal/app",
					Modules: []domain.Module{{
						ObjectID: "obj-module-1", Path: "internal/app/book",
					}},
				}},
			}},
		}},
		Contracts: []domain.Contract{{
			ID: "contract-1", ModuleRef: domain.Ref{FormID: "FRM-1-05", FormType: domain.TypeLLD, FormVersion: 1},
			Kind: "http", InputType: "Book", OutputType: "Book",
			Errors: []domain.ContractError{{Code: "E-1", Condition: "the book is missing"}},
		}},
		Paths: []domain.Path{
			{ID: "path-1", ContractRef: "contract-1", Kind: domain.PathHappy, Given: "a book", When: "it is lent", Then: "the loan is stored"},
			{ID: "path-2", ContractRef: "contract-1", Kind: domain.PathNegative, ExpectedErrorCode: "E-1", Given: "no book", When: "it is lent", Then: "the call fails"},
		},
	})
}

func validTaskPlan(t *testing.T) json.RawMessage {
	return mustJSON(t, domain.TaskPlan{
		Tasks: []domain.PlanTask{
			{
				ID: "T-1", Title: ste("Register a book."), TaskType: domain.TaskFeature, Owner: domain.OwnerDev,
				Acceptance:     []domain.Statement{{ID: "AC-1", Text: "A book is stored.", Modality: domain.ModalityShall, Priority: domain.PriorityMust}},
				AcceptanceRefs: []string{"FR-1"},
			},
			{
				ID: "T-2", Title: ste("Add the book store."), TaskType: domain.TaskFeature, Owner: domain.OwnerDev,
				ParentID: "T-1", Acceptance: []domain.Statement{{ID: "AC-2", Text: "The store returns a book.", Modality: domain.ModalityShall, Priority: domain.PriorityMust}},
				ContractRefs: []string{"contract-1"},
			},
		},
		Roots: []string{"T-1"},
	})
}
