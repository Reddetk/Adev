package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Reddetk/Adev/internal/domain"
)

// FormService is the single write path of the form contract.
// It resolves references, checks the envelope, the language and the payload gates.
type FormService struct {
	store   FormStore
	checker TextChecker // nil disables the language check
}

// NewFormService builds the form service.
func NewFormService(store FormStore, checker TextChecker) *FormService {
	return &FormService{store: store, checker: checker}
}

// Submit validates a form against its references and stores it.
func (s *FormService) Submit(ctx context.Context, form *domain.Form) error {
	if form == nil {
		return fmt.Errorf("submit: form is nil")
	}

	resolved, err := s.resolveRefs(ctx, form)
	if err != nil {
		return err
	}
	if err := form.ValidateEnvelope(resolved); err != nil {
		return err
	}
	if s.checker != nil {
		texts, err := form.PayloadTexts()
		if err != nil {
			return err
		}
		if err := s.checkTexts(texts); err != nil {
			return err
		}
	}
	if err := form.ValidatePayload(); err != nil {
		return err
	}
	return s.store.Save(ctx, form)
}

// Materialize applies a JSON Patch to a base form and returns the next full version.
func (s *FormService) Materialize(base *domain.Form, ops json.RawMessage) (*domain.Form, error) {
	if base == nil {
		return nil, fmt.Errorf("materialize: base form is nil")
	}
	payload, err := ApplyDelta(base.Payload, ops)
	if err != nil {
		return nil, err
	}

	derived := *base
	derived.FormVersion = base.FormVersion + 1
	derived.PreviousFormVersion = &domain.Ref{
		FormID: base.FormID, FormType: base.FormType, FormVersion: base.FormVersion, Hash: base.PayloadHash,
	}
	derived.Mode = domain.ModeFull
	derived.Ops = nil
	derived.Base = nil
	if err := derived.SetPayload(json.RawMessage(payload)); err != nil {
		return nil, err
	}
	return &derived, nil
}

// SetStatus changes the status of a stored form version.
func (s *FormService) SetStatus(ctx context.Context, formID string, version int, status domain.Status) error {
	return s.store.SetStatus(ctx, formID, version, status)
}

func (s *FormService) resolveRefs(ctx context.Context, form *domain.Form) ([]domain.ResolvedRef, error) {
	refs := make([]domain.Ref, 0, len(form.Inputs)+1)
	refs = append(refs, form.Inputs...)
	if form.Base != nil {
		refs = append(refs, *form.Base)
	}

	resolved := make([]domain.ResolvedRef, 0, len(refs))
	for _, ref := range refs {
		if ref.FormID == "" || ref.FormVersion < 1 {
			resolved = append(resolved, domain.ResolvedRef{Ref: ref})
			continue
		}
		stored, err := s.store.Get(ctx, ref.FormID, ref.FormVersion)
		if err != nil {
			if errors.Is(err, domain.ErrFormNotFound) {
				continue
			}
			return nil, fmt.Errorf("resolve %s:%d: %w", ref.FormID, ref.FormVersion, err)
		}
		resolved = append(resolved, domain.ResolvedRef{Ref: ref, PayloadHash: stored.PayloadHash})
	}
	return resolved, nil
}

func (s *FormService) checkTexts(texts []domain.Text) error {
	var errs []error
	for i, text := range texts {
		if text.Value == "" {
			continue
		}
		if text.Lang != domain.LangSTE100 && text.Lang != domain.LangACE {
			errs = append(errs, fmt.Errorf("texts[%d]: unknown lang %q", i, text.Lang))
			continue
		}
		if err := s.checker.Check(text); err != nil {
			errs = append(errs, fmt.Errorf("texts[%d]: %w", i, err))
		}
	}
	return errors.Join(errs...)
}
