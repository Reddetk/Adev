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

	// The answer completes the field in place: the draft keeps its version.
	answer, err := service.AnswerOnPipeline(ctx, 15, "/metrics/0/unit", json.RawMessage(`"hour"`))
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	if answer.FormVersion != 1 || answer.PreviousFormVersion != nil {
		t.Fatalf("the answer must stay on the same version: %+v", answer.Envelope)
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
	if answer.FormVersion != 1 {
		t.Fatalf("want version 1, got %d", answer.FormVersion)
	}
	stored, err := store.Get(ctx, "FRM-15-00", 1)
	if err != nil {
		t.Fatalf("get v1: %v", err)
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
	if !strings.Contains(question.Question, "Choose integration method") {
		t.Fatalf("the question must carry the text: %q", question.Question)
	}
	if len(question.Options) != 2 || question.Options[0] != "Option A" || question.Options[1] != "Option B" {
		t.Fatalf("the options are a field, not text: %+v", question.Options)
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
	if answer.FormVersion != 1 {
		t.Fatalf("want version 1, got %d", answer.FormVersion)
	}
	stored, err := store.Get(ctx, "FRM-15-00", 1)
	if err != nil {
		t.Fatalf("get v1: %v", err)
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

func TestAnswerOnAFieldOfTheOnlyListEntryLands(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), &fakeQuestionRequests{})

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-20-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: "20", TargetSystemID: "23", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(json.RawMessage(`{"goals": [], "metrics": [{"id": "12", "name": "ABtests"}], "openGaps": [
		{"id": "GAP-3", "pointer": "/metrics/unit", "question": {"text": "Which unit?", "lang": "STE100"}}]}`)); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	answered, err := service.AnswerOnPipeline(ctx, 20, "/metrics/unit", json.RawMessage(`"count"`))

	if err != nil {
		t.Fatalf("a field of the only list entry must land: %v", err)
	}
	var payload struct {
		Metrics []struct {
			Unit string `json:"unit"`
		} `json:"metrics"`
		OpenGaps []json.RawMessage `json:"openGaps"`
	}
	if err := json.Unmarshal(answered.Payload, &payload); err != nil {
		t.Fatalf("decode answered payload: %v", err)
	}
	if len(payload.Metrics) != 1 || payload.Metrics[0].Unit != "count" {
		t.Fatalf("the answer must set metrics[0].unit, got %s", answered.Payload)
	}
	if len(payload.OpenGaps) != 0 {
		t.Fatalf("the answered gap must leave the draft, got %s", answered.Payload)
	}
}

func TestAnswerCreatesTheMissingObjectOfAGapPointer(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), &fakeQuestionRequests{})

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-22-01", FormType: domain.TypeFeasibility, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "01", Mode: domain.ModeFull,
		PipelineID: "22", TargetSystemID: "24", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(json.RawMessage(`{"options": [{"id": "OPT-01"}], "openGaps": [
		{"id": "GAP-02", "pointer": "/options/0/cost/amount", "question": {"text": "How much?", "lang": "STE100"}}]}`)); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	answered, err := service.AnswerOnPipeline(ctx, 22, "/options/0/cost/amount", json.RawMessage(`1000`))

	if err != nil {
		t.Fatalf("a field under a missing object must land: %v", err)
	}
	var payload struct {
		Options []struct {
			Cost struct {
				Amount float64 `json:"amount"`
			} `json:"cost"`
		} `json:"options"`
		OpenGaps []json.RawMessage `json:"openGaps"`
	}
	if err := json.Unmarshal(answered.Payload, &payload); err != nil {
		t.Fatalf("decode answered payload: %v", err)
	}
	if len(payload.Options) != 1 || payload.Options[0].Cost.Amount != 1000 {
		t.Fatalf("the answer must set options[0].cost.amount, got %s", answered.Payload)
	}
	if len(payload.OpenGaps) != 0 {
		t.Fatalf("the answered gap must leave the draft, got %s", answered.Payload)
	}
}

func TestAnAnswerThatBreaksTheDecodeReturnsTheDraftToTheStage(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), &fakeQuestionRequests{})

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-24-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: "24", TargetSystemID: "26", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(json.RawMessage(`{"goals": "a string where the list belongs", "metrics": []}`)); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	if _, err := service.AnswerOnPipeline(ctx, 24, "/goals", json.RawMessage(`"improve user experience"`)); err != nil {
		t.Fatalf("answer: %v", err)
	}
	stored, err := store.Get(ctx, "FRM-24-00", 1)
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}
	if stored.Status != domain.StatusDraft {
		t.Fatalf("no question can wake a version that does not decode: want draft so the stage re-runs, got %s", stored.Status)
	}
}

func TestAnEntryNamedByItsIDLandsOnItsIndex(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), &fakeQuestionRequests{})

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-25-01", FormType: domain.TypeFeasibility, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "01", Mode: domain.ModeFull,
		PipelineID: "25", TargetSystemID: "27", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(json.RawMessage(`{"options": [{"id": "OPT-1"}, {"id": "OPT-2"}], "openGaps": [
		{"id": "GAP-1", "pointer": "/options/OPT-2/cost/amount", "question": {"text": "How much?", "lang": "STE100"}}]}`)); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	answered, err := service.AnswerOnPipeline(ctx, 25, "/options/OPT-2/cost/amount", json.RawMessage(`1000`))
	if err != nil {
		t.Fatalf("an entry named by its id must land: %v", err)
	}
	var payload struct {
		Options []struct {
			Cost struct {
				Amount float64 `json:"amount"`
			} `json:"cost"`
		} `json:"options"`
		OpenGaps []json.RawMessage `json:"openGaps"`
	}
	if err := json.Unmarshal(answered.Payload, &payload); err != nil {
		t.Fatalf("decode answered payload: %v", err)
	}
	if payload.Options[1].Cost.Amount != 1000 || len(payload.OpenGaps) != 0 {
		t.Fatalf("the answer must set options[1].cost.amount and close the gap, got %s", answered.Payload)
	}
}

func TestAnAnswerToAnObjectPlaceOffersItsFields(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), &fakeQuestionRequests{})

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-26-01", FormType: domain.TypeFeasibility, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "01", Mode: domain.ModeFull,
		PipelineID: "26", TargetSystemID: "28", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(json.RawMessage(`{"options": [{"id": "OPT-1"}], "openGaps": [
		{"id": "GAP-1", "pointer": "/options/0/cost", "question": {"text": "Which currency and amount?", "lang": "STE100"}}]}`)); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	_, err := service.AnswerOnPipeline(ctx, 26, "/options/0/cost", json.RawMessage(`"USD"`))
	ambiguous, ok := err.(*usecase.AmbiguousPointerError)
	if !ok || len(ambiguous.Choices) != 2 || ambiguous.Choices[0].Pointer != "/options/0/cost/amount" {
		t.Fatalf("a text answer to an object must offer its fields, got %v", err)
	}

	answered, err := service.AnswerChoice(ctx, 26, "/options/0/cost", "/options/0/cost/currency", json.RawMessage(`"USD"`))
	if err != nil {
		t.Fatalf("the chosen field must take the answer: %v", err)
	}
	var payload struct {
		Options []struct {
			Cost struct {
				Currency string `json:"currency"`
			} `json:"cost"`
		} `json:"options"`
	}
	if err := json.Unmarshal(answered.Payload, &payload); err != nil {
		t.Fatalf("decode answered payload: %v", err)
	}
	if payload.Options[0].Cost.Currency != "USD" {
		t.Fatalf("the currency must land on options[0].cost.currency, got %s", answered.Payload)
	}
}

func TestAmbiguousPointerOffersTheEntriesAndTheChoiceLands(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), &fakeQuestionRequests{})

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-23-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: "23", TargetSystemID: "25", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(json.RawMessage(`{"metrics": [{"id": "M1", "name": "uptime"}, {"id": "M2", "name": "latency"}], "openGaps": [
		{"id": "GAP-3", "pointer": "/metrics/unit", "question": {"text": "Which unit?", "lang": "STE100"}}]}`)); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	_, err := service.AnswerOnPipeline(ctx, 23, "/metrics/unit", json.RawMessage(`"ms"`))
	ambiguous, ok := err.(*usecase.AmbiguousPointerError)
	if !ok || len(ambiguous.Choices) != 2 {
		t.Fatalf("want the two entries to choose from, got %v", err)
	}
	if ambiguous.Choices[1].Pointer != "/metrics/1/unit" || ambiguous.Choices[1].Label != "#1 · latency" {
		t.Fatalf("want /metrics/1/unit labelled by its name, got %+v", ambiguous.Choices[1])
	}

	answered, err := service.AnswerChoice(ctx, 23, "/metrics/unit", "/metrics/1/unit", json.RawMessage(`"ms"`))
	if err != nil {
		t.Fatalf("the chosen entry must take the answer: %v", err)
	}
	var payload struct {
		Metrics []struct {
			Unit string `json:"unit"`
		} `json:"metrics"`
		OpenGaps []json.RawMessage `json:"openGaps"`
	}
	if err := json.Unmarshal(answered.Payload, &payload); err != nil {
		t.Fatalf("decode answered payload: %v", err)
	}
	if payload.Metrics[1].Unit != "ms" || payload.Metrics[0].Unit != "" {
		t.Fatalf("the unit must land on the chosen entry only, got %s", answered.Payload)
	}
	if len(payload.OpenGaps) != 0 {
		t.Fatalf("the gap asked about /metrics/unit must leave the draft, got %s", answered.Payload)
	}
}

func TestAnswerOnAStaleListPointerLandsOnTheFirstEntry(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), &fakeQuestionRequests{})

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-18-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: "18", TargetSystemID: "21", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(json.RawMessage(`{"goals": [], "metrics": [], "openGaps": [
		{"id": "GAP-1", "pointer": "/goals/outcome", "question": {"text": "What market purpose?", "lang": "STE100"}}]}`)); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	answered, err := service.AnswerOnPipeline(ctx, 18, "/goals/outcome", json.RawMessage(`"improve user experience"`))

	if err != nil {
		t.Fatalf("a stale pointer must still land: %v", err)
	}
	var payload struct {
		Goals []struct {
			Outcome domain.Text `json:"outcome"`
		} `json:"goals"`
		OpenGaps []json.RawMessage `json:"openGaps"`
	}
	if err := json.Unmarshal(answered.Payload, &payload); err != nil {
		t.Fatalf("decode answered payload: %v", err)
	}
	if len(payload.Goals) != 1 || payload.Goals[0].Outcome.Value != "improve user experience" {
		t.Fatalf("the answer must create goals[0].outcome, got %s", answered.Payload)
	}
	if len(payload.OpenGaps) != 0 {
		t.Fatalf("the answered gap must leave the draft, got %s", answered.Payload)
	}
}

func TestAnswerOnAListTheModelWroteAsAStringStillLands(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), &fakeQuestionRequests{})

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-19-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: "19", TargetSystemID: "22", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(json.RawMessage(`{"goals": "10 user after MVP run", "metrics": [], "openGaps": [
		{"id": "GAP-1", "pointer": "/goals/outcome", "question": {"text": "What market purpose?", "lang": "STE100"}}]}`)); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	answered, err := service.AnswerOnPipeline(ctx, 19, "/goals/outcome", json.RawMessage(`"improve user experience"`))

	if err != nil {
		t.Fatalf("the answer must land on a list the model wrote as a string: %v", err)
	}
	var payload struct {
		Goals []struct {
			Outcome domain.Text `json:"outcome"`
		} `json:"goals"`
	}
	if err := json.Unmarshal(answered.Payload, &payload); err != nil {
		t.Fatalf("decode answered payload: %v", err)
	}
	if len(payload.Goals) != 1 || payload.Goals[0].Outcome.Value != "improve user experience" {
		t.Fatalf("the answer must create goals[0].outcome, got %s", answered.Payload)
	}
}

func TestAnswerThatBreaksTheShapeReturnsTheDraft(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-17-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusWaitingOperator, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: "17", TargetSystemID: "20", ProducedBy: domain.RoleAnalyst,
	}}
	raw := json.RawMessage(`{
		"goals": [{"id": "G1", "outcome": {"text": "The system shall track books.", "lang": "STE100"}, "metricId": "M1"}],
		"metrics": [{"id": "M1", "name": "accuracy", "unit": "count", "baseline": 0, "target": 95}],
		"scope": {"in": [{"text": "Book registration", "lang": "STE100"}], "out": []}
	}`)
	if err := form.SetRawPayload(raw); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	if _, err := service.Answer(ctx, form, "/scope", json.RawMessage(`"every room"`), 17); err != nil {
		t.Fatalf("answer: %v", err)
	}
	stored, err := store.Get(ctx, "FRM-17-00", 1)
	if err != nil {
		t.Fatalf("get v1: %v", err)
	}
	// The draft no longer decodes and no question can be asked about it: waiting
	// would hold the stage forever, so the draft goes back for a re-run.
	if stored.Status != domain.StatusDraft {
		t.Fatalf("a version no question can wake must return to draft, got %s", stored.Status)
	}
	for _, request := range requests.saved {
		if request.Pointer == "/payload" {
			t.Fatalf("the shape failure must not become a question: %+v", request)
		}
	}
}

// TestShapeFailureIsNotAnOperatorQuestion: a payload that does not decode into the
// document shape is the model's mistake. The operator cannot type a whole document,
// so the run stores the draft without blocking on a question at /payload.
func TestShapeFailureIsNotAnOperatorQuestion(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-16-00", FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: "16", TargetSystemID: "20", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(json.RawMessage(`{"scope": "not an object"}`)); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	failures := []domain.ValidationError{{Field: "payload", Problem: "does not match the BRD shape: decode payload"}}

	if err := service.Publish(ctx, form, failures, 16); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(requests.saved) != 0 {
		t.Fatalf("a shape failure must not ask the operator, got %+v", requests.saved)
	}
}

// TestAnswerRemovesItsGapEvenWhenTheRestIsBroken: a field broke earlier, so the
// payload binds into no gap. The answer still fills its field and removes its own
// openGaps entry, otherwise the version waits forever with no question.
func TestAnswerRemovesItsGapEvenWhenTheRestIsBroken(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	form := waitingForm(t, domain.TypeSRS, "FRM-16-02", "02", `{
		"fr": [{"id": "FR-1", "errors": "payment failure, internal error"}],
		"openGaps": [
			{"id": "OG-2", "pointer": "/fr/0/response", "blocking": false, "question": {"lang": "STE100", "text": "What is the response?"}},
			{"id": "OG-3", "pointer": "/fr/0/errors", "blocking": false, "question": {"lang": "STE100", "text": "Which errors?"}}
		]
	}`)
	if err := service.Publish(ctx, form, []domain.ValidationError{{Field: "fr[0].response", Problem: "is empty"}}, 16); err != nil {
		t.Fatalf("publish: %v", err)
	}

	answer, err := service.AnswerOnPipeline(ctx, 16, "/fr/0/response", json.RawMessage(`"The system shows the booking reference."`))
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	var gaps struct {
		OpenGaps []struct {
			Pointer string `json:"pointer"`
		} `json:"openGaps"`
	}
	if err := json.Unmarshal(answer.Payload, &gaps); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if len(gaps.OpenGaps) != 1 || gaps.OpenGaps[0].Pointer != "/fr/0/errors" {
		t.Fatalf("want only the unanswered gap left, got %+v", gaps.OpenGaps)
	}
}

// waitingForm builds a form of a type with gates in the draft state of its first
// version, so a test can put several stages of one pipeline on hold at once.
func waitingForm(t *testing.T, formType domain.FormType, formID, stage, payload string) *domain.Form {
	t.Helper()
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: formID, FormType: formType, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: stage, Mode: domain.ModeFull,
		PipelineID: "16", TargetSystemID: "21", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(json.RawMessage(payload)); err != nil {
		t.Fatalf("set payload of %s: %v", formType, err)
	}
	return form
}

// TestAnswerLandsOnTheFormThatHoldsThePointer: two stages wait at once, and the
// operator request names only a pointer. The answer reaches the form whose payload
// has that slot, not the first waiting form of the pipeline.
func TestAnswerLandsOnTheFormThatHoldsThePointer(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	failure := []domain.ValidationError{{Field: "paths/-", Problem: "no negative path carries the code"}}
	srs := waitingForm(t, domain.TypeSRS, "FRM-16-02", "02", `{"fr":[],"nfr":[]}`)
	lld := waitingForm(t, domain.TypeLLD, "FRM-16-05", "05", `{"contracts":[{"id":"contract-1","errors":[]}],"paths":[]}`)
	for _, form := range []*domain.Form{srs, lld} {
		if err := service.Publish(ctx, form, failure, 16); err != nil {
			t.Fatalf("publish %s: %v", form.FormType, err)
		}
	}

	answer, err := service.AnswerOnPipeline(ctx, 16, "/contracts/0/errors",
		json.RawMessage(`[{"code": "E-1", "condition": "the gateway times out"}]`))
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	if answer.FormID != lld.FormID || answer.FormVersion != 1 {
		t.Fatalf("the answer must land on the LLD in place as version 1, got %s v%d", answer.FormID, answer.FormVersion)
	}
	var corrected domain.LLD
	if err := answer.BindPayload(&corrected); err != nil {
		t.Fatalf("bind payload: %v", err)
	}
	if len(corrected.Contracts[0].Errors) != 1 || corrected.Contracts[0].Errors[0].Condition != "the gateway times out" {
		t.Fatalf("want the answer in the contract of the LLD, got %+v", corrected.Contracts[0].Errors)
	}
}

// TestAnAnswerAppendsThePathPastTheEndOfTheList: the LLD gate asks for the path a
// code is missing, so the answer names the slot past the end of the list and the
// document keeps the paths it already has.
func TestAnAnswerAppendsThePathPastTheEndOfTheList(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	form := waitingForm(t, domain.TypeLLD, "FRM-16-05", "05", `{
		"contracts": [{"id": "contract-1", "errors": [{"code": "E-1", "condition": "the book is missing"}]}],
		"paths": [{"id": "path-1", "contractRef": "contract-1", "kind": "happy",
			"given": "a book", "when": "it is lent", "then": "the loan is stored"}]
	}`)
	failure := []domain.ValidationError{{Field: "paths/-", Problem: `no negative path carries the code "E-1" of contract "contract-1"`}}
	if err := service.Publish(ctx, form, failure, 16); err != nil {
		t.Fatalf("publish: %v", err)
	}

	answer, err := service.AnswerOnPipeline(ctx, 16, "/paths/-", json.RawMessage(`{
		"id": "path-2", "contractRef": "contract-1", "kind": "negative",
		"given": "no book", "when": "it is lent", "then": "the call fails", "expectedErrorCode": "E-1"
	}`))
	if err != nil {
		t.Fatalf("answer: %v", err)
	}
	var corrected domain.LLD
	if err := answer.BindPayload(&corrected); err != nil {
		t.Fatalf("bind payload: %v", err)
	}
	if len(corrected.Paths) != 2 {
		t.Fatalf("want the path appended to the one already there, got %d", len(corrected.Paths))
	}
	if corrected.Paths[1].ExpectedErrorCode != "E-1" || corrected.Paths[1].Kind != domain.PathNegative {
		t.Fatalf("want the new path to carry the code, got %+v", corrected.Paths[1])
	}
}

// TestOneRunAsksABatchOfValueGaps: an LLD with seven answerable gaps asks five,
// and a re-run while they wait adds none. The rest go out as the answers close
// the batch, so the operator never receives the whole document as cards.
func TestOneRunAsksABatchOfValueGaps(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	form := waitingForm(t, domain.TypeLLD, "FRM-16-05", "05", `{
		"containers": [{"containerId": "C-1", "layers": [{"name": "app"}]}],
		"contracts": [{"id": "CTR-1", "kind": "http", "inputType": "In", "outputType": "Out"}],
		"paths": [{"id": "P-1", "contractRef": "CTR-1", "kind": "happy", "given": "a", "when": "b", "then": "c"}],
		"openGaps": [
			{"id": "G1", "pointer": "/containers/0/layers/0/name", "question": {"text": "Which layer?", "lang": "STE100"}},
			{"id": "G2", "pointer": "/contracts/0/kind", "question": {"text": "Which kind?", "lang": "STE100"}},
			{"id": "G3", "pointer": "/contracts/0/inputType", "question": {"text": "Which input?", "lang": "STE100"}},
			{"id": "G4", "pointer": "/contracts/0/outputType", "question": {"text": "Which output?", "lang": "STE100"}},
			{"id": "G5", "pointer": "/paths/0/given", "question": {"text": "Which state?", "lang": "STE100"}},
			{"id": "G6", "pointer": "/paths/0/when", "question": {"text": "Which call?", "lang": "STE100"}},
			{"id": "G7", "pointer": "/paths/0/then", "question": {"text": "Which result?", "lang": "STE100"}}
		]
	}`)
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save draft: %v", err)
	}

	published, blocked, err := service.PublishGaps(ctx, form, 16)
	if err != nil {
		t.Fatalf("publish gaps: %v", err)
	}
	if !published || !blocked {
		t.Fatalf("want the batch published and blocking, got published=%v blocked=%v", published, blocked)
	}
	if len(requests.saved) != 5 {
		t.Fatalf("want one batch of five questions, got %d", len(requests.saved))
	}

	// A re-run while the batch waits must not stack a second batch.
	if _, _, err := service.PublishGaps(ctx, form, 16); err != nil {
		t.Fatalf("republish: %v", err)
	}
	if len(requests.saved) != 5 {
		t.Fatalf("the waiting batch must hold the next one, got %d questions", len(requests.saved))
	}
}
