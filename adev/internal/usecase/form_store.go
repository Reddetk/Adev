package usecase

import (
	"context"

	"github.com/Reddetk/Adev/internal/domain"
)

// FormStore is the persistence port of the form versions.
// The repository layer implements it; a stored version is never edited.
type FormStore interface {
	Save(ctx context.Context, form *domain.Form) error
	Get(ctx context.Context, formID string, version int) (*domain.Form, error)
	Latest(ctx context.Context, formID string) (*domain.Form, error)
	ByPipeline(ctx context.Context, pipelineID string, formType domain.FormType) ([]*domain.Form, error)
	SetStatus(ctx context.Context, formID string, version int, status domain.Status) error
}

// TextChecker validates one text value. The language checker implements it.
type TextChecker interface {
	Check(domain.Text) error
}
