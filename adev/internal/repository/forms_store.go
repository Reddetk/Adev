package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

const formColumns = `form_id, form_version, form_type, schema_version, previous_form, status,
	route, stage, mode, base, ops, pipeline_id, target_system_id, produced_by, inputs, payload, payload_hash`

// FormsStore persists forms in the append-only forms table.
type FormsStore struct {
	pool *pgxpool.Pool
}

// FormsStore and MemoryFormsStore are the adapters of the same port.
var _ usecase.FormStore = (*FormsStore)(nil)

// OpenFormsStore opens a pool and checks the connection.
func OpenFormsStore(ctx context.Context, dsn string) (*FormsStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres store: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("open postgres store: %w", err)
	}
	return &FormsStore{pool: pool}, nil
}

// NewFormsStore uses an existing pool.
func NewFormsStore(pool *pgxpool.Pool) *FormsStore {
	return &FormsStore{pool: pool}
}

// Close releases the pool.
func (s *FormsStore) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// Save appends a new form version.
func (s *FormsStore) Save(ctx context.Context, form *domain.Form) error {
	if form == nil {
		return fmt.Errorf("save: form is nil")
	}
	if err := form.ValidateNewVersion(); err != nil {
		return err
	}
	if form.FormVersion > 1 {
		exists, err := s.exists(ctx, form.FormID, form.FormVersion-1)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: %s:%d", domain.ErrMissingPrevious, form.FormID, form.FormVersion-1)
		}
	}

	previous, err := optionalJSON(form.PreviousFormVersion)
	if err != nil {
		return err
	}
	base, err := optionalJSON(form.Base)
	if err != nil {
		return err
	}
	inputs := form.Inputs
	if inputs == nil {
		inputs = []domain.Ref{}
	}
	inputsJSON, err := json.Marshal(inputs)
	if err != nil {
		return fmt.Errorf("save: encode inputs: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO forms (
			form_id, form_version, form_type, schema_version, previous_form, status,
			route, stage, mode, base, ops, pipeline_id, target_system_id, produced_by,
			inputs, payload, payload_hash
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		form.FormID, form.FormVersion, form.FormType, form.SchemaVersion, previous, form.Status,
		form.Route, form.Stage, form.Mode, base, nullableJSON(form.Ops), form.PipelineID,
		form.TargetSystemID, form.ProducedBy, inputsJSON, form.Payload, form.PayloadHash,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: %s:%d", domain.ErrVersionExists, form.FormID, form.FormVersion)
		}
		return fmt.Errorf("save form: %w", err)
	}
	return nil
}

// Get reads one form version.
func (s *FormsStore) Get(ctx context.Context, formID string, version int) (*domain.Form, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+formColumns+` FROM forms WHERE form_id = $1 AND form_version = $2`, formID, version)
	form, err := scanForm(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s:%d", domain.ErrFormNotFound, formID, version)
	}
	return form, err
}

// Latest reads the highest form version.
func (s *FormsStore) Latest(ctx context.Context, formID string) (*domain.Form, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+formColumns+` FROM forms WHERE form_id = $1 ORDER BY form_version DESC LIMIT 1`, formID)
	form, err := scanForm(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", domain.ErrFormNotFound, formID)
	}
	return form, err
}

// ByPipeline reads the versions of one form type in one pipeline, oldest first.
func (s *FormsStore) ByPipeline(ctx context.Context, pipelineID string, formType domain.FormType) ([]*domain.Form, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+formColumns+` FROM forms WHERE pipeline_id = $1 AND form_type = $2 ORDER BY form_version ASC`, pipelineID, formType)
	if err != nil {
		return nil, fmt.Errorf("by pipeline: %w", err)
	}
	defer rows.Close()

	var result []*domain.Form
	for rows.Next() {
		form, err := scanForm(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, form)
	}
	return result, rows.Err()
}

// SetStatus changes the status of one form version.
func (s *FormsStore) SetStatus(ctx context.Context, formID string, version int, status domain.Status) error {
	current, err := s.Get(ctx, formID, version)
	if err != nil {
		return err
	}
	if !domain.CanChangeStatus(current.Status, status) {
		return fmt.Errorf("%w: %s to %s", domain.ErrInvalidStatusChange, current.Status, status)
	}

	tag, err := s.pool.Exec(ctx,
		`UPDATE forms SET status = $3 WHERE form_id = $1 AND form_version = $2 AND status = $4`,
		formID, version, status, current.Status)
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s:%d", domain.ErrInvalidStatusChange, formID, version)
	}
	return nil
}

func (s *FormsStore) exists(ctx context.Context, formID string, version int) (bool, error) {
	var found bool
	err := s.pool.QueryRow(ctx, `SELECT true FROM forms WHERE form_id = $1 AND form_version = $2`, formID, version).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("exists: %w", err)
	}
	return found, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanForm(row rowScanner) (*domain.Form, error) {
	var (
		form     domain.Form
		previous []byte
		base     []byte
		ops      []byte
		inputs   []byte
		payload  []byte
	)
	err := row.Scan(
		&form.FormID, &form.FormVersion, &form.FormType, &form.SchemaVersion, &previous, &form.Status,
		&form.Route, &form.Stage, &form.Mode, &base, &ops, &form.PipelineID,
		&form.TargetSystemID, &form.ProducedBy, &inputs, &payload, &form.PayloadHash,
	)
	if err != nil {
		return nil, err
	}

	references, err := decodeOptionalRef(previous)
	if err != nil {
		return nil, fmt.Errorf("scan form: decode previousFormVersion: %w", err)
	}
	form.PreviousFormVersion = references

	baseForm, err := decodeOptionalRef(base)
	if err != nil {
		return nil, fmt.Errorf("scan form: decode base: %w", err)
	}
	form.Base = baseForm
	if len(ops) > 0 {
		form.Ops = json.RawMessage(ops)
	}
	if len(inputs) > 0 {
		if err := json.Unmarshal(inputs, &form.Inputs); err != nil {
			return nil, fmt.Errorf("scan form: decode inputs: %w", err)
		}
	}
	form.Payload = json.RawMessage(payload)
	return &form, nil
}

// optionalJSON encodes a nullable value. A typed nil pointer becomes SQL NULL, not
// the JSON literal null: otherwise a read resurrects an empty reference.
func optionalJSON(value any) ([]byte, error) {
	if isNil(value) {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode reference: %w", err)
	}
	return raw, nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

// decodeOptionalRef reads a nullable jsonb column. A SQL NULL and the JSON literal
// null both mean "no reference"; older rows stored the literal.
func decodeOptionalRef(raw []byte) (*domain.Ref, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var ref domain.Ref
	if err := json.Unmarshal(raw, &ref); err != nil {
		return nil, err
	}
	return &ref, nil
}

func nullableJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return nil
	}
	return raw
}
