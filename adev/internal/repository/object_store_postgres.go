package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

// postgresObjectStore keeps the objects of the target system registry in the
// object_entries table that migration 005 rebuilt.
type postgresObjectStore struct {
	pool *pgxpool.Pool
}

func NewPostgresObjectStore(pool *pgxpool.Pool) usecase.ObjectStore {
	return &postgresObjectStore{pool: pool}
}

func (p *postgresObjectStore) Save(ctx context.Context, object domain.Object) error {
	fields, changedBy, err := encodeObjectJSON(object)
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx,
		`INSERT INTO object_entries (object_id, target_system_id, kind, name, path, container_id, status, fields, introduced_by, changed_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (object_id) DO UPDATE SET
		     target_system_id = EXCLUDED.target_system_id, kind = EXCLUDED.kind, name = EXCLUDED.name,
		     path = EXCLUDED.path, container_id = EXCLUDED.container_id, status = EXCLUDED.status,
		     fields = EXCLUDED.fields, changed_by = EXCLUDED.changed_by`,
		object.ObjectID, object.TargetSystemID, string(object.Kind), object.Name, object.Path,
		object.ContainerID, string(object.Status), fields, object.IntroducedBy, changedBy,
	)
	return err
}

func (p *postgresObjectStore) Get(ctx context.Context, objectID string) (domain.Object, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT object_id, target_system_id, kind, name, path, container_id, status, fields, introduced_by, changed_by
		 FROM object_entries WHERE object_id = $1`, objectID)
	return scanObject(row)
}

func (p *postgresObjectStore) FindByTargetSystem(ctx context.Context, targetSystemID int64) ([]domain.Object, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT object_id, target_system_id, kind, name, path, container_id, status, fields, introduced_by, changed_by
		 FROM object_entries WHERE target_system_id = $1 AND status <> 'removed' ORDER BY object_id`,
		targetSystemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Object
	for rows.Next() {
		object, err := scanObject(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, object)
	}
	return result, rows.Err()
}

// Allocate returns the next free sequence of one kind. Object ids follow
// OBJ-<kind>-<seq>.
// shortcut: allocate is one read then one write; two concurrent stages could draw
// the same id and the second Save fails on the primary key. Retry it when
// parallel stage runs appear.
// Allocate returns the next free sequence of one kind. The digits live in the
// id suffix "OBJ-<kind>-<seq>"; the class [^0-9] avoids an escape whose Go raw
// string would double and match the wrong thing.
func (p *postgresObjectStore) Allocate(ctx context.Context, kind domain.Kind) (int, error) {
	var next int
	err := p.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(sub.seq), 0) + 1 FROM (
		     SELECT CAST(regexp_replace(object_id, '[^0-9]', '', 'g') AS integer) AS seq
		     FROM object_entries WHERE kind = $1 AND object_id LIKE 'OBJ-' || $1 || '-%'
		 ) sub`, string(kind)).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("allocate %s: %w", kind, err)
	}
	return next, nil
}

type pgxRow interface {
	Scan(dest ...any) error
}

func scanObject(row pgxRow) (domain.Object, error) {
	var object domain.Object
	var fields, introducedBy, changedBy []byte
	if err := row.Scan(&object.ObjectID, &object.TargetSystemID, &object.Kind, &object.Name,
		&object.Path, &object.ContainerID, &object.Status, &fields, &introducedBy, &changedBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Object{}, ErrNotFound
		}
		return domain.Object{}, err
	}
	if err := json.Unmarshal(fields, &object.Fields); err != nil {
		return domain.Object{}, fmt.Errorf("object %s fields: %w", object.ObjectID, err)
	}
	if err := json.Unmarshal(introducedBy, &object.IntroducedBy); err != nil {
		return domain.Object{}, fmt.Errorf("object %s introducedBy: %w", object.ObjectID, err)
	}
	if err := json.Unmarshal(changedBy, &object.ChangedBy); err != nil {
		return domain.Object{}, fmt.Errorf("object %s changedBy: %w", object.ObjectID, err)
	}
	return object, nil
}

func encodeObjectJSON(object domain.Object) (fields, changedBy json.RawMessage, err error) {
	if object.Fields == nil {
		object.Fields = map[string]any{}
	}
	fields, err = json.Marshal(object.Fields)
	if err != nil {
		return nil, nil, err
	}
	changedBy, err = json.Marshal(object.ChangedBy)
	if err != nil {
		return nil, nil, err
	}
	return fields, changedBy, nil
}
