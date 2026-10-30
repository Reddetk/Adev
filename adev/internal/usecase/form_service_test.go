package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

type fakeFormStore struct {
	forms   map[string]*domain.Form
	saved   int
	updated int
}

func newFakeFormStore() *fakeFormStore {
	return &fakeFormStore{forms: map[string]*domain.Form{}}
}

// Save mimics the database: a version is written once, and the stored row is a
// copy, not the caller's pointer.
func (f *fakeFormStore) Save(_ context.Context, form *domain.Form) error {
	key := form.FormID + ":" + itoa(form.FormVersion)
	if _, ok := f.forms[key]; ok {
		return fmt.Errorf("%w: %s", domain.ErrVersionExists, key)
	}
	stored := *form
	f.forms[key] = &stored
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

// ByPipeline returns the stored versions of one pipeline, oldest first. A form
// type that is empty returns every type.
func (f *fakeFormStore) ByPipeline(_ context.Context, pipelineID string, formType domain.FormType) ([]*domain.Form, error) {
	out := make([]*domain.Form, 0)
	for _, form := range f.forms {
		if form.PipelineID != pipelineID {
			continue
		}
		if formType != "" && form.FormType != formType {
			continue
		}
		out = append(out, form)
	}
	slices.SortFunc(out, func(a, b *domain.Form) int { return a.FormVersion - b.FormVersion })
	return out, nil
}

// SetStatus moves the stored version, as the database does.
func (f *fakeFormStore) SetStatus(_ context.Context, formID string, version int, status domain.Status) error {
	stored, ok := f.forms[formID+":"+itoa(version)]
	if !ok {
		return domain.ErrFormNotFound
	}
	stored.Status = status
	return nil
}

// UpdatePayload rewrites the stored draft: the checkpoint path of a long walk.
func (f *fakeFormStore) UpdatePayload(_ context.Context, form *domain.Form) error {
	key := form.FormID + ":" + itoa(form.FormVersion)
	stored, ok := f.forms[key]
	if !ok {
		return domain.ErrFormNotFound
	}
	stored.Payload = form.Payload
	stored.PayloadHash = form.PayloadHash
	f.updated++
	return nil
}

func (f *fakeFormStore) Delete(context.Context, string, int) error { return nil }

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

// TestSubmitDraftRewritesItsOwnVersion: the detail walk saves progress into the
// draft of the version it fills, so the next checkpoint rewrites that draft instead
// of stacking a version per model call. An accepted form still needs its own
// version: the append-only rule does not move.
func TestSubmitDraftRewritesItsOwnVersion(t *testing.T) {
	store := newFakeFormStore()
	service := NewFormService(store, nil)
	form := userInputForm(t)

	if err := service.SubmitDraft(context.Background(), form); err != nil {
		t.Fatalf("save the first checkpoint: %v", err)
	}
	partial := *form
	if err := partial.SetRawPayload(json.RawMessage(`{"rawRequest":{"text":"one more entry answered","lang":"STE100"}}`)); err != nil {
		t.Fatalf("set the checkpoint payload: %v", err)
	}
	if err := service.SubmitDraft(context.Background(), &partial); err != nil {
		t.Fatalf("the next checkpoint must rewrite the draft: %v", err)
	}
	if store.saved != 1 || store.updated != 1 {
		t.Fatalf("want one insert and one rewrite, got insert %d, rewrite %d", store.saved, store.updated)
	}
	stored, err := store.Get(context.Background(), form.FormID, form.FormVersion)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(stored.Payload) != string(partial.Payload) {
		t.Fatalf("the draft must hold the last checkpoint, got %s", stored.Payload)
	}

	if err := service.Submit(context.Background(), form); !errors.Is(err, domain.ErrVersionExists) {
		t.Fatalf("an accepted form must not reuse the draft version, got %v", err)
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

// TestSubmitDraftRefusesAPayloadThatIsNotADocument: the model returns a string where
// the shape is an object. Such a version serves nobody: the console cannot draw it
// and an operator answer has no field to land in.
func TestSubmitDraftRefusesAPayloadThatIsNotADocument(t *testing.T) {
	store := newFakeFormStore()
	service := NewFormService(store, nil)

	broken := userInputForm(t)
	if err := broken.SetRawPayload(json.RawMessage(`"a string, not a document"`)); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	if err := service.SubmitDraft(context.Background(), broken); err == nil {
		t.Fatal("a payload that is not a document must not be stored")
	}
	if store.saved != 0 {
		t.Fatalf("nothing must be stored, got %d versions", store.saved)
	}

	if err := service.SubmitDraft(context.Background(), userInputForm(t)); err != nil {
		t.Fatalf("a document must still be stored: %v", err)
	}
	if store.saved != 1 {
		t.Fatalf("want one saved draft, got %d", store.saved)
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
