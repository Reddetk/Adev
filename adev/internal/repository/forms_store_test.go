package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

func buildForm(t *testing.T, formID string, version int, pipelineID string, previous *domain.Ref) *domain.Form {
	t.Helper()
	form := &domain.Form{Envelope: domain.Envelope{
		FormID: formID, FormType: domain.TypeBRD, SchemaVersion: 1, FormVersion: version,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: pipelineID, TargetSystemID: "TGT-1", ProducedBy: domain.RoleAnalyst,
		PreviousFormVersion: previous,
	}}
	payload := domain.BRD{
		Goals: []domain.Goal{{
			ID: "GOAL-1", Outcome: domain.Text{Value: "The system shall reduce the cycle time.", Lang: domain.LangSTE100},
			MetricID: "M-1",
		}},
		Metrics: []domain.Metric{{
			ID: "M-1", Name: "cycle_time_hours", Unit: "hour",
			Baseline: floatPtr(40), Target: floatPtr(24),
		}},
		SuccessMetrics: []domain.Success{{MetricID: "M-1", Threshold: "24", Window: "30 day"}},
	}
	if err := form.SetPayload(payload); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	return form
}

func floatPtr(v float64) *float64 { return &v }

func TestInMemoryFormsStore(t *testing.T) {
	runStoreSuite(t, NewMemoryFormsStore())
}

// runStoreSuite checks the shared contract of every forms store implementation.
func runStoreSuite(t *testing.T, store usecase.FormStore) {
	ctx := context.Background()

	t.Run("save and get", func(t *testing.T) {
		form := buildForm(t, "FRM-SAVE", 1, "PIPE-SAVE", nil)
		if err := store.Save(ctx, form); err != nil {
			t.Fatalf("save: %v", err)
		}
		got, err := store.Get(ctx, "FRM-SAVE", 1)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.FormID != "FRM-SAVE" || got.PayloadHash != form.PayloadHash {
			t.Fatalf("unexpected form: %+v", got)
		}
	})

	t.Run("duplicate version is rejected", func(t *testing.T) {
		form := buildForm(t, "FRM-DUP", 1, "PIPE-DUP", nil)
		if err := store.Save(ctx, form); err != nil {
			t.Fatalf("first save: %v", err)
		}
		if err := store.Save(ctx, form); !errors.Is(err, domain.ErrVersionExists) {
			t.Fatalf("want ErrVersionExists, got %v", err)
		}
	})

	t.Run("version above one needs its predecessor", func(t *testing.T) {
		form := buildForm(t, "FRM-ORPHAN", 2, "PIPE-ORPHAN", nil)
		if err := store.Save(ctx, form); !errors.Is(err, domain.ErrMissingPrevious) {
			t.Fatalf("want ErrMissingPrevious, got %v", err)
		}
	})

	t.Run("version chain", func(t *testing.T) {
		first := buildForm(t, "FRM-CHAIN", 1, "PIPE-CHAIN", nil)
		if err := store.Save(ctx, first); err != nil {
			t.Fatalf("save v1: %v", err)
		}
		previous := &domain.Ref{FormID: "FRM-CHAIN", FormType: domain.TypeBRD, FormVersion: 1, Hash: first.PayloadHash}
		second := buildForm(t, "FRM-CHAIN", 2, "PIPE-CHAIN", previous)
		if err := store.Save(ctx, second); err != nil {
			t.Fatalf("save v2: %v", err)
		}

		latest, err := store.Latest(ctx, "FRM-CHAIN")
		if err != nil {
			t.Fatalf("latest: %v", err)
		}
		if latest.FormVersion != 2 {
			t.Fatalf("want version 2, got %d", latest.FormVersion)
		}
	})

	t.Run("latest of an unknown form", func(t *testing.T) {
		if _, err := store.Latest(ctx, "FRM-MISSING"); !errors.Is(err, domain.ErrFormNotFound) {
			t.Fatalf("want ErrFormNotFound, got %v", err)
		}
	})

	t.Run("get of an unknown version", func(t *testing.T) {
		if _, err := store.Get(ctx, "FRM-SAVE", 9); !errors.Is(err, domain.ErrFormNotFound) {
			t.Fatalf("want ErrFormNotFound, got %v", err)
		}
	})

	t.Run("by pipeline keeps only one pipeline and type", func(t *testing.T) {
		if err := store.Save(ctx, buildForm(t, "FRM-P1", 1, "PIPE-LIST", nil)); err != nil {
			t.Fatalf("save: %v", err)
		}
		second := buildForm(t, "FRM-P1", 2, "PIPE-LIST", &domain.Ref{FormID: "FRM-P1", FormType: domain.TypeBRD, FormVersion: 1})
		if err := store.Save(ctx, second); err != nil {
			t.Fatalf("save: %v", err)
		}
		if err := store.Save(ctx, buildForm(t, "FRM-P2", 1, "PIPE-OTHER", nil)); err != nil {
			t.Fatalf("save: %v", err)
		}

		forms, err := store.ByPipeline(ctx, "PIPE-LIST", domain.TypeBRD)
		if err != nil {
			t.Fatalf("by pipeline: %v", err)
		}
		if len(forms) != 2 || forms[0].FormVersion != 1 || forms[1].FormVersion != 2 {
			t.Fatalf("unexpected versions: %+v", forms)
		}
	})

	t.Run("status lifecycle", func(t *testing.T) {
		if err := store.Save(ctx, buildForm(t, "FRM-STATUS", 1, "PIPE-STATUS", nil)); err != nil {
			t.Fatalf("save: %v", err)
		}
		if err := store.SetStatus(ctx, "FRM-STATUS", 1, domain.StatusValid); err != nil {
			t.Fatalf("valid: %v", err)
		}
		if err := store.SetStatus(ctx, "FRM-STATUS", 1, domain.StatusApproved); err != nil {
			t.Fatalf("approved: %v", err)
		}
		if err := store.SetStatus(ctx, "FRM-STATUS", 1, domain.StatusSuperseded); err != nil {
			t.Fatalf("superseded: %v", err)
		}
		if err := store.SetStatus(ctx, "FRM-STATUS", 1, domain.StatusValid); !errors.Is(err, domain.ErrInvalidStatusChange) {
			t.Fatalf("want ErrInvalidStatusChange from superseded, got %v", err)
		}
	})

	t.Run("status cannot go back to draft", func(t *testing.T) {
		if err := store.Save(ctx, buildForm(t, "FRM-BACK", 1, "PIPE-BACK", nil)); err != nil {
			t.Fatalf("save: %v", err)
		}
		if err := store.SetStatus(ctx, "FRM-BACK", 1, domain.StatusValid); err != nil {
			t.Fatalf("valid: %v", err)
		}
		if err := store.SetStatus(ctx, "FRM-BACK", 1, domain.StatusDraft); !errors.Is(err, domain.ErrInvalidStatusChange) {
			t.Fatalf("want ErrInvalidStatusChange, got %v", err)
		}
	})

	t.Run("a stored form is never aliased", func(t *testing.T) {
		form := buildForm(t, "FRM-COPY", 1, "PIPE-COPY", nil)
		if err := store.Save(ctx, form); err != nil {
			t.Fatalf("save: %v", err)
		}
		form.PayloadHash = "sha256:tampered"

		got, err := store.Get(ctx, "FRM-COPY", 1)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.PayloadHash == "sha256:tampered" {
			t.Fatal("the store kept a pointer to the caller form")
		}
		if !strings.HasPrefix(got.PayloadHash, "sha256:") {
			t.Fatalf("unexpected hash: %s", got.PayloadHash)
		}
	})
}
