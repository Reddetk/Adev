package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

// MemoryFormsStore is an in-process form store for tests and for runs without a database.
type MemoryFormsStore struct {
	mu      sync.RWMutex
	forms   map[string]map[int]*domain.Form
	created map[string]bool
}

// MemoryFormsStore is the second adapter of the same port.
var _ usecase.FormStore = (*MemoryFormsStore)(nil)

// NewMemoryFormsStore returns an empty store.
func NewMemoryFormsStore() *MemoryFormsStore {
	return &MemoryFormsStore{
		forms:   map[string]map[int]*domain.Form{},
		created: map[string]bool{},
	}
}

// Save appends a new form version.
func (m *MemoryFormsStore) Save(ctx context.Context, form *domain.Form) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if form == nil {
		return fmt.Errorf("save: form is nil")
	}
	if err := form.ValidateNewVersion(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.created[formKey(form.FormID, form.FormVersion)] {
		return fmt.Errorf("%w: %s:%d", domain.ErrVersionExists, form.FormID, form.FormVersion)
	}
	if form.FormVersion > 1 && !m.created[formKey(form.FormID, form.FormVersion-1)] {
		return fmt.Errorf("%w: %s:%d", domain.ErrMissingPrevious, form.FormID, form.FormVersion-1)
	}

	versions, ok := m.forms[form.FormID]
	if !ok {
		versions = map[int]*domain.Form{}
		m.forms[form.FormID] = versions
	}
	versions[form.FormVersion] = cloneForm(form)
	m.created[formKey(form.FormID, form.FormVersion)] = true
	return nil
}

// Get reads one form version.
func (m *MemoryFormsStore) Get(ctx context.Context, formID string, version int) (*domain.Form, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	versions, ok := m.forms[formID]
	if !ok {
		return nil, fmt.Errorf("%w: %s:%d", domain.ErrFormNotFound, formID, version)
	}
	form, ok := versions[version]
	if !ok {
		return nil, fmt.Errorf("%w: %s:%d", domain.ErrFormNotFound, formID, version)
	}
	return cloneForm(form), nil
}

// Latest reads the highest form version.
func (m *MemoryFormsStore) Latest(ctx context.Context, formID string) (*domain.Form, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	versions, ok := m.forms[formID]
	if !ok || len(versions) == 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrFormNotFound, formID)
	}
	highest := 0
	for version := range versions {
		if version > highest {
			highest = version
		}
	}
	return cloneForm(versions[highest]), nil
}

// ByPipeline reads the versions of one form type in one pipeline, oldest first.
func (m *MemoryFormsStore) ByPipeline(ctx context.Context, pipelineID string, formType domain.FormType) ([]*domain.Form, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*domain.Form
	for _, versions := range m.forms {
		for _, form := range versions {
			if form.PipelineID == pipelineID && form.FormType == formType {
				result = append(result, cloneForm(form))
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].FormVersion < result[j].FormVersion })
	return result, nil
}

// SetStatus changes the status of one form version.
func (m *MemoryFormsStore) SetStatus(ctx context.Context, formID string, version int, status domain.Status) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	versions, ok := m.forms[formID]
	if !ok {
		return fmt.Errorf("%w: %s:%d", domain.ErrFormNotFound, formID, version)
	}
	form, ok := versions[version]
	if !ok {
		return fmt.Errorf("%w: %s:%d", domain.ErrFormNotFound, formID, version)
	}
	if !domain.CanChangeStatus(form.Status, status) {
		return fmt.Errorf("%w: %s to %s", domain.ErrInvalidStatusChange, form.Status, status)
	}
	form.Status = status
	return nil
}

func formKey(formID string, version int) string { return fmt.Sprintf("%s:%d", formID, version) }

// cloneForm copies a form so that a caller cannot change the stored value.
func cloneForm(form *domain.Form) *domain.Form {
	raw, err := json.Marshal(form)
	if err != nil {
		return form
	}
	var copy domain.Form
	if err := json.Unmarshal(raw, &copy); err != nil {
		return form
	}
	return &copy
}
