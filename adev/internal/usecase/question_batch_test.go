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

// feasibilityDraft is a TEJ that misses three premises: the operator answers them
// through questions, the rest of the document already passes the gates.
func feasibilityDraft(t *testing.T, pipelineID string) *domain.Form {
	t.Helper()
	budget := domain.Cost{Currency: "EUR", Amount: 800}
	cost := domain.Cost{Currency: "EUR", Amount: 1000}
	payload := domain.Feasibility{
		ReportType: domain.ReportTEJ,
		Premises: domain.Premises{
			Traffic:       domain.TrafficProfile{PeakQps: 0, ReadWriteRatio: 4, PayloadBytes: 2048},
			Storage:       domain.StorageProfile{GBPerYear: 40},
			Team:          domain.TeamProfile{Size: 0, Languages: []string{"go"}, OpsMaturity: domain.OpsMedium},
			BudgetMonthly: &budget,
			DeadlineDays:  120,
			Availability:  99.9,
			RPOMinutes:    15,
			RTOMinutes:    60,
		},
		Options: []domain.Option{{
			ID: "opt-1", Summary: domain.Text{Value: "Build it.", Lang: domain.LangSTE100},
			Cost: &cost, ScheduleDays: 30, Confidence: domain.ConfidenceHigh,
		}},
		RecommendedOption: "opt-1",
		Verdict:           domain.VerdictGo,
		Reason:            domain.Text{Value: "The team can build it.", Lang: domain.LangSTE100},
		OpenGaps: []domain.Gap{
			{ID: "gap-1", Pointer: "/premises/traffic/peakQps", Question: domain.Text{Value: "What is the peak request rate?", Lang: domain.LangSTE100}},
			{ID: "gap-2", Pointer: "/premises/storage/gbPerYear", Question: domain.Text{Value: "How many gigabytes a year?", Lang: domain.LangSTE100}},
			{ID: "gap-3", Pointer: "/premises/team/size", Question: domain.Text{Value: "How many people build it?", Lang: domain.LangSTE100}},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-" + pipelineID + "-01-FEASIBILITY", FormType: domain.TypeFeasibility, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "01", Mode: domain.ModeFull,
		PipelineID: pipelineID, TargetSystemID: "1", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(raw); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	return form
}

func activeRequests(requests *fakeQuestionRequests, pipelineID int) []domain.OperatorRequest {
	var out []domain.OperatorRequest
	for _, request := range requests.saved {
		if request.PipelineID == pipelineID && request.Status == "active" {
			out = append(out, request)
		}
	}
	return out
}

// TestTheBatchIsFilledWithGateQuestions: a stage that asked one question from a
// gap must not stop there. The gate failures fill the rest of the batch, so the
// operator answers five values in one round instead of one value per run.
func TestTheBatchIsFilledWithGateQuestions(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	form := feasibilityDraft(t, "31")
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("store the draft: %v", err)
	}

	published, err := service.PublishGaps(ctx, form, 31)
	if err != nil {
		t.Fatalf("publish gaps: %v", err)
	}
	if published != 3 {
		t.Fatalf("want the three gaps, got %d", published)
	}

	// The gates still fail on the same three fields: the batch is already full, so
	// nothing new goes out and the operator is not asked twice.
	failures := []domain.ValidationError{
		{Field: "premises.traffic.peakQps", Problem: "must be greater than zero"},
		{Field: "premises.storage.gbPerYear", Problem: "must be greater than zero"},
		{Field: "premises.team.size", Problem: "must be greater than zero"},
	}
	asked, err := service.PublishGate(ctx, form, failures, 31, 2)
	if err != nil {
		t.Fatalf("publish gate: %v", err)
	}
	if asked != 0 {
		t.Fatalf("a full batch must not ask again, got %d", asked)
	}
	if got := len(activeRequests(requests, 31)); got != 3 {
		t.Fatalf("want three open questions, got %d", got)
	}
}

// TestGateQuestionsTakeTheRestOfTheBatch: one gap leaves room for the gate
// failures, and they fill it up to the cap.
func TestGateQuestionsTakeTheRestOfTheBatch(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	form := feasibilityDraft(t, "32")
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("store the draft: %v", err)
	}

	published, err := service.PublishGaps(ctx, form, 32)
	if err != nil || published != 3 {
		t.Fatalf("publish gaps: %d, %v", published, err)
	}

	// Five fields fail the gates; the batch holds five questions in total, so only
	// two more go out after the three gaps.
	failures := []domain.ValidationError{
		{Field: "premises.traffic.peakQps", Problem: "must be greater than zero"},
		{Field: "premises.storage.gbPerYear", Problem: "must be greater than zero"},
		{Field: "premises.team.size", Problem: "must be greater than zero"},
		{Field: "premises.deadlineDays", Problem: "must be greater than zero"},
		{Field: "premises.availability", Problem: "must be at least 99.0"},
	}
	asked, err := service.PublishGate(ctx, form, failures, 32, 2)
	if err != nil {
		t.Fatalf("publish gate: %v", err)
	}
	if asked != 2 {
		t.Fatalf("want the two free slots filled, got %d", asked)
	}
	if got := len(activeRequests(requests, 32)); got != 5 {
		t.Fatalf("want five open questions, got %d", got)
	}
}

// TestAnswerManyAppliesTheBatch: five answers are one operator action. The draft
// takes every value, every question closes, and the form becomes valid.
func TestAnswerManyAppliesTheBatch(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	form := feasibilityDraft(t, "33")
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("store the draft: %v", err)
	}
	if _, err := service.PublishGaps(ctx, form, 33); err != nil {
		t.Fatalf("publish gaps: %v", err)
	}

	answered, err := service.AnswerManyOnPipeline(ctx, 33, []usecase.AnswerInput{
		{Pointer: "/premises/traffic/peakQps", Value: json.RawMessage(`120`)},
		{Pointer: "/premises/storage/gbPerYear", Value: json.RawMessage(`40`)},
		{Pointer: "/premises/team/size", Value: json.RawMessage(`4`)},
	})
	if err != nil {
		t.Fatalf("answer the batch: %v", err)
	}
	if answered.Status != domain.StatusValid {
		t.Fatalf("the filled TEJ must pass, got %s", answered.Status)
	}
	stored, err := store.Latest(ctx, form.FormID)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	for _, want := range []string{`"peakQps":120`, `"gbPerYear":40`, `"size":4`} {
		if !strings.Contains(string(stored.Payload), want) {
			t.Fatalf("the payload must hold %s, got %s", want, stored.Payload)
		}
	}
	if got := len(activeRequests(requests, 33)); got != 0 {
		t.Fatalf("every answered question must close, %d still open", got)
	}
	if len(stored.Payload) == 0 || stored.FormVersion != 1 {
		t.Fatalf("the answer rewrites the draft in place, got version %d", stored.FormVersion)
	}
}

// TestAnswerManyRefusesAPointerThatIsNotAField: the batch is atomic. One wrong
// pointer refuses the whole batch, so no half-applied decision reaches the draft.
func TestAnswerManyRefusesAPointerThatIsNotAField(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), &fakeQuestionRequests{})

	form := feasibilityDraft(t, "34")
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("store the draft: %v", err)
	}

	_, err := service.AnswerManyOnPipeline(ctx, 34, []usecase.AnswerInput{
		{Pointer: "/premises/traffic/peakQps", Value: json.RawMessage(`120`)},
		{Pointer: "/premises/traffic/unknown", Value: json.RawMessage(`1`)},
	})
	if err == nil {
		t.Fatal("a pointer outside the shape must refuse the batch")
	}
	stored, readErr := store.Latest(ctx, form.FormID)
	if readErr != nil {
		t.Fatalf("read back: %v", readErr)
	}
	if strings.Contains(string(stored.Payload), `"peakQps":120`) {
		t.Fatal("a refused batch must not apply its valid half")
	}
}

// TestAnswerManyRefusesTheSamePointerTwice: two values for one field is an
// operator mistake, not a last-wins race.
func TestAnswerManyRefusesTheSamePointerTwice(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), &fakeQuestionRequests{})

	form := feasibilityDraft(t, "35")
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("store the draft: %v", err)
	}

	_, err := service.AnswerManyOnPipeline(ctx, 35, []usecase.AnswerInput{
		{Pointer: "/premises/traffic/peakQps", Value: json.RawMessage(`120`)},
		{Pointer: "/premises/traffic/peakQps", Value: json.RawMessage(`500`)},
	})
	if err == nil {
		t.Fatal("a duplicated pointer must refuse the batch")
	}
}

// TestAnAnsweredPointerIsNotAskedAgain: the model repeats a gap that the operator
// already answered. The completed question is the record of the decision, so the
// same pointer does not become a question twice.
func TestAnAnsweredPointerIsNotAskedAgain(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryFormsStore()
	requests := &fakeQuestionRequests{saved: []domain.OperatorRequest{{
		ID: 1, PipelineID: 41, Pointer: "/nfr/0/metric", Status: "completed",
	}}}
	service := usecase.NewQuestionService(usecase.NewFormService(store, nil), requests)

	payload := json.RawMessage(`{"fr":[],"nfr":[{"id":"nfr-1","metric":"availability","unit":"","threshold":"","measurementPoint":"","window":""}],"openGaps":[
		{"id":"g1","pointer":"/nfr/0/metric","question":{"lang":"STE100","text":"Which metric?"}},
		{"id":"g2","pointer":"/nfr/0/unit","question":{"lang":"STE100","text":"Which unit?"}}
	]}`)
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-41-02", FormType: domain.TypeSRS, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "02", Mode: domain.ModeFull,
		PipelineID: "41", TargetSystemID: "2", ProducedBy: domain.RoleAnalyst,
	}}
	if err := form.SetRawPayload(payload); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("store: %v", err)
	}

	published, err := service.PublishGaps(ctx, form, 41)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if published != 1 {
		t.Fatalf("want only the unanswered unit, got %d", published)
	}
	if got := activeRequests(requests, 41); len(got) != 1 || got[0].Pointer != "/nfr/0/unit" {
		t.Fatalf("want the unit question only, got %+v", got)
	}
}
