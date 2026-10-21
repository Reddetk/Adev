package usecase_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/repository"
	"github.com/Reddetk/Adev/internal/usecase"
)

// fakeQuestionRequests stores the operator requests a stage published.
type fakeQuestionRequests struct{ saved []domain.OperatorRequest }

func (f *fakeQuestionRequests) Save(request *domain.OperatorRequest) error {
	request.ID = uint(len(f.saved) + 1)
	f.saved = append(f.saved, *request)
	return nil
}

func (f *fakeQuestionRequests) Update(request *domain.OperatorRequest) error {
	if request.ID == 0 || int(request.ID) > len(f.saved) {
		return domain.ErrFormNotFound
	}
	f.saved[request.ID-1] = *request
	return nil
}

func (f *fakeQuestionRequests) FindByPointer(pointer string) (*domain.OperatorRequest, error) {
	for index := range f.saved {
		if f.saved[index].Pointer == pointer {
			return &f.saved[index], nil
		}
	}
	return nil, domain.ErrFormNotFound
}

func (f *fakeQuestionRequests) FindByID(int) (*domain.OperatorRequest, error) {
	return nil, domain.ErrFormNotFound
}

func (f *fakeQuestionRequests) FindByPipelineID(int) (*domain.OperatorRequest, error) {
	return nil, domain.ErrFormNotFound
}

func (f *fakeQuestionRequests) FindByQuestion(string) (*domain.OperatorRequest, error) {
	return nil, domain.ErrFormNotFound
}

func (f *fakeQuestionRequests) FindByStatus(string) (*domain.OperatorRequest, error) {
	return nil, domain.ErrFormNotFound
}

func (f *fakeQuestionRequests) Delete(int) error { return nil }

func (f *fakeQuestionRequests) FindAll() ([]domain.OperatorRequest, error) { return f.saved, nil }

// TestQuestionRoundTrip: a blocked stage stores its draft, asks the operator about
// the failed field, and the answer completes the form as a valid next version.
func TestQuestionRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)
	baseline := 0.0

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-15-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: "15", TargetSystemID: "20", ProducedBy: domain.RoleAnalyst,
	}}
	payload := domain.BRD{
		Goals: []domain.Goal{{
			ID: "G1", Outcome: domain.Text{Value: "The system shall track books.", Lang: domain.LangSTE100}, MetricID: "M1",
		}},
		Metrics: []domain.Metric{{ID: "M1", Name: "accuracy", Unit: "", Baseline: &baseline}},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := form.SetRawPayload(raw); err != nil {
		t.Fatalf("set payload: %v", err)
	}

	// The gate failure of the last attempt reaches the operator as one question.
	failures := []domain.ValidationError{{Field: "metrics[0].unit", Problem: "is empty"}}
	if err := service.Publish(ctx, form, failures, 15); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(requests.saved) != 1 {
		t.Fatalf("want one operator request, got %d", len(requests.saved))
	}
	question := requests.saved[0]
	if question.PipelineID != 15 || question.Pointer != "/metrics/0/unit" || !question.Blocking {
		t.Fatalf("unexpected request: %+v", question)
	}

	// The answer completes the field: the next version passes the gates and is valid.
	answer, err := service.AnswerOnPipeline(ctx, 15, "/metrics/0/unit", json.RawMessage(`"hour"`))
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	if answer.FormVersion != 2 || answer.PreviousFormVersion == nil {
		t.Fatalf("the answer must be a new version: %+v", answer.Envelope)
	}
	// The metric still has no target: the next failure becomes the new question.
	if len(requests.saved) != 2 || requests.saved[1].Pointer != "/metrics/0/target" {
		t.Fatalf("want a follow-up question about the target, got %+v", requests.saved)
	}
	if requests.saved[0].Status != "completed" {
		t.Fatalf("the answered request must be completed, got %s", requests.saved[0].Status)
	}

	// The second answer closes the last gap and the form becomes valid.
	answer, err = service.AnswerOnPipeline(ctx, 15, "/metrics/0/target", json.RawMessage(`95`))
	if err != nil {
		t.Fatalf("second answer: %v", err)
	}
	if answer.FormVersion != 3 {
		t.Fatalf("want version 3, got %d", answer.FormVersion)
	}
	stored, err := store.Get(ctx, "FRM-15-00", 3)
	if err != nil {
		t.Fatalf("get v2: %v", err)
	}
	if stored.Status != domain.StatusValid {
		t.Fatalf("want status valid, got %s", stored.Status)
	}
	var corrected domain.BRD
	if err := stored.BindPayload(&corrected); err != nil {
		t.Fatalf("bind payload: %v", err)
	}
	if corrected.Metrics[0].Unit != "hour" || corrected.Metrics[0].Target == nil || *corrected.Metrics[0].Target != 95 {
		t.Fatalf("want the answers in the payload, got %+v", corrected.Metrics[0])
	}
	if requests.saved[1].Status != "completed" {
		t.Fatalf("want the request completed, got %s", requests.saved[1].Status)
	}
}

// TestOpenGapBecomesAnOperatorQuestion: the model fills the gates but leaves a
// blocking open gap. The run stops with one question about the gap, and the
// answer writes the decision into the gap field and removes the gap entry.
func TestOpenGapBecomesAnOperatorQuestion(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-15-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: "15", TargetSystemID: "20", ProducedBy: domain.RoleAnalyst,
	}}
	raw := json.RawMessage(`{
		"goals": [{"id": "G1", "outcome": {"text": "The system shall track books.", "lang": "STE100"}, "metricId": "M1"}],
		"metrics": [{"id": "M1", "name": "accuracy", "unit": "count", "baseline": 0, "target": 95}],
		"scope": {"in": [{"text": "Book registration", "lang": "STE100"}], "out": []},
		"openGaps": [{
			"id": "GAP-001", "formRef": {"formId": "FRM-15-00", "formType": "BRD", "formVersion": 1},
			"pointer": "/scope/in/0", "question": {"text": "Choose integration method", "lang": "STE100"},
			"options": ["Option A", "Option B"], "blocking": true
		}]
	}`)
	if err := form.SetRawPayload(raw); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	// The run publishes the open gap as one question and marks the form waiting.
	published, blocked, err := service.PublishGaps(ctx, form, 15)
	if err != nil {
		t.Fatalf("publish gaps: %v", err)
	}
	if !published || !blocked {
		t.Fatalf("a blocking gap must be published and stop the stage, got published=%v blocked=%v", published, blocked)
	}
	if len(requests.saved) != 1 {
		t.Fatalf("want one operator request, got %d", len(requests.saved))
	}
	question := requests.saved[0]
	if question.PipelineID != 15 || question.Pointer != "/scope/in/0" || !question.Blocking {
		t.Fatalf("unexpected request: %+v", question)
	}
	if !strings.Contains(question.Question, "Choose integration method") || !strings.Contains(question.Question, "Option A") {
		t.Fatalf("the question must carry the text and the options: %q", question.Question)
	}
	waiting, err := store.Get(ctx, "FRM-15-00", 1)
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}
	if waiting.Status != domain.StatusWaitingOperator {
		t.Fatalf("want the draft waiting for the operator, got %s", waiting.Status)
	}

	// A re-run does not ask the same question again while the request is active.
	if _, _, err := service.PublishGaps(ctx, form, 15); err != nil {
		t.Fatalf("republish: %v", err)
	}
	if len(requests.saved) != 1 {
		t.Fatalf("the same gap must not be asked twice, got %d", len(requests.saved))
	}

	// The answer lands in the gap field, removes the gap and validates the form.
	// The operator answers in plain words: the field keeps its Text shape.
	answer, err := service.AnswerOnPipeline(ctx, 15, "/scope/in/0", json.RawMessage(`"manual registration"`))
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	if answer.FormVersion != 2 {
		t.Fatalf("want version 2, got %d", answer.FormVersion)
	}
	stored, err := store.Get(ctx, "FRM-15-00", 2)
	if err != nil {
		t.Fatalf("get v2: %v", err)
	}
	var corrected domain.BRD
	if err := stored.BindPayload(&corrected); err != nil {
		t.Fatalf("bind payload: %v", err)
	}
	if len(corrected.OpenGaps) != 0 {
		t.Fatalf("the answered gap must leave the payload, got %+v", corrected.OpenGaps)
	}
	if corrected.Scope.In[0].Value != "manual registration" {
		t.Fatalf("want the decision in the field, got %+v", corrected.Scope.In[0])
	}
	if stored.Status != domain.StatusValid {
		t.Fatalf("want status valid, got %s", stored.Status)
	}
	if requests.saved[0].Status != "completed" {
		t.Fatalf("want the request completed, got %s", requests.saved[0].Status)
	}
}
