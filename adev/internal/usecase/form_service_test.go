package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

type fakeFormStore struct {
	forms map[string]*domain.Form
	saved int
}

func newFakeFormStore() *fakeFormStore {
	return &fakeFormStore{forms: map[string]*domain.Form{}}
}

func (f *fakeFormStore) Save(_ context.Context, form *domain.Form) error {
	f.forms[form.FormID+":"+itoa(form.FormVersion)] = form
	f.saved++
	return nil
}

func (f *fakeFormStore) Get(_ context.Context, formID string, version int) (*domain.Form, error) {
	form, ok := f.forms[formID+":"+itoa(version)]
	if !ok {
		return nil, domain.ErrFormNotFound
	}
	return form, nil
}

func (f *fakeFormStore) Latest(context.Context, string) (*domain.Form, error) {
	return nil, domain.ErrFormNotFound
}

func (f *fakeFormStore) ByPipeline(context.Context, string, domain.FormType) ([]*domain.Form, error) {
	return nil, nil
}

func (f *fakeFormStore) SetStatus(context.Context, string, int, domain.Status) error { return nil }

type failingChecker struct{}

func (failingChecker) Check(domain.Text) error { return errors.New("forbidden word") }

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	digits := ""
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}
	return digits
}

func userInputForm(t *testing.T) *domain.Form {
	t.Helper()
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-IN", FormType: domain.TypeUserInput, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "—", Mode: domain.ModeFull,
		PipelineID: "PIPE-1", TargetSystemID: "TGT-1", ProducedBy: domain.RoleOperator,
	}}
	payload := domain.UserInput{
		RawRequest: domain.Text{Value: "The operator sends a request.", Lang: domain.LangSTE100},
		Source:     domain.SourceOperator,
		ReceivedAt: "2026-10-19T21:00:00Z",
	}
	if err := form.SetPayload(payload); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	return form
}

func TestSubmitStoresAValidForm(t *testing.T) {
	store := newFakeFormStore()
	service := NewFormService(store, nil)

	if err := service.Submit(context.Background(), userInputForm(t)); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if store.saved != 1 {
		t.Fatalf("want one saved form, got %d", store.saved)
	}
}

func TestSubmitRejectsUnresolvedInput(t *testing.T) {
	store := newFakeFormStore()
	service := NewFormService(store, nil)

	form := userInputForm(t)
	form.FormType = domain.TypeBRD
	form.Inputs = []domain.Ref{{FormID: "FRM-MISSING", FormType: domain.TypeUserInput, FormVersion: 1}}
	if err := form.SetPayload(map[string]any{}); err != nil {
		t.Fatalf("set payload: %v", err)
	}

	err := service.Submit(context.Background(), form)
	if err == nil || !contains(err.Error(), "not resolved") {
		t.Fatalf("want an unresolved input error, got %v", err)
	}
	if store.saved != 0 {
		t.Fatal("an invalid form must not be stored")
	}
}

func TestSubmitRunsTheLanguageChecker(t *testing.T) {
	store := newFakeFormStore()
	service := NewFormService(store, failingChecker{})

	err := service.Submit(context.Background(), userInputForm(t))
	if err == nil || !contains(err.Error(), "forbidden word") {
		t.Fatalf("want a language error, got %v", err)
	}
}

func TestMaterializeAppliesDelta(t *testing.T) {
	base := userInputForm(t)
	base.Payload = json.RawMessage(`{"rawRequest":{"text":"a","lang":"STE100"},"source":"operator","receivedAt":"2026-10-19T21:00:00Z"}`)
	if _, err := base.ComputeHash(); err != nil {
		t.Fatalf("hash: %v", err)
	}
	base.PayloadHash, _ = base.ComputeHash()

	derived, err := NewFormService(newFakeFormStore(), nil).Materialize(base, json.RawMessage(`[{"op":"replace","path":"/source","value":"api"}]`))
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if derived.FormVersion != 2 || derived.PreviousFormVersion == nil {
		t.Fatalf("unexpected derived form: %+v", derived.Envelope)
	}
	if derived.Mode != domain.ModeFull || derived.Ops != nil || derived.Base != nil {
		t.Fatal("the derived form must be a full version")
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
