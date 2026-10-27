package usecase

import (
	"context"

	"github.com/Reddetk/Adev/internal/domain"
)

// FormStore is the persistence port of the form versions.
// The repository layer implements it. A version is edited only while it waits for
// the operator: the operator's answer rewrites that draft in place.
type FormStore interface {
	Save(ctx context.Context, form *domain.Form) error
	Get(ctx context.Context, formID string, version int) (*domain.Form, error)
	Latest(ctx context.Context, formID string) (*domain.Form, error)
	ByPipeline(ctx context.Context, pipelineID string, formType domain.FormType) ([]*domain.Form, error)
	SetStatus(ctx context.Context, formID string, version int, status domain.Status) error
	UpdatePayload(ctx context.Context, form *domain.Form) error
	// Delete removes one version (version > 0) or every version of the document
	// (version <= 0). It is the escape hatch for a document that blocks the
	// pipeline and cannot be repaired by an operator answer.
	Delete(ctx context.Context, formID string, version int) error
}

// TextChecker validates one text value. The language checker implements it.
type TextChecker interface {
	Check(domain.Text) error
}
