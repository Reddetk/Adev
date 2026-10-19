package repository

import (
	"github.com/Reddetk/Adev/internal/usecase"

	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
)

const auditEventColumns = `id, pipeline_id, run_id, event_type, created_at`

type postgresAuditEventRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAuditEventRepository(pool *pgxpool.Pool) usecase.AuditEventRepository {
	return &postgresAuditEventRepository{pool: pool}
}

func (p *postgresAuditEventRepository) Save(auditevent *domain.AuditEvent) error {
	createdAt := auditevent.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	err := p.pool.QueryRow(context.Background(),
		`INSERT INTO audit_events (pipeline_id, run_id, event_type, created_at)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		auditevent.PipelineID, auditevent.RunID, auditevent.EventType, createdAt,
	).Scan(&auditevent.ID)
	if err != nil {
		return err
	}
	auditevent.CreatedAt = createdAt
	return nil
}

func (p *postgresAuditEventRepository) FindByID(id int) (*domain.AuditEvent, error) {
	return scanAuditEvent(p.pool.QueryRow(context.Background(),
		`SELECT `+auditEventColumns+` FROM audit_events WHERE id = $1`, id))
}

func (p *postgresAuditEventRepository) FindByPipelineID(pipelineID int) (*domain.AuditEvent, error) {
	return scanAuditEvent(p.pool.QueryRow(context.Background(),
		`SELECT `+auditEventColumns+` FROM audit_events WHERE pipeline_id = $1`, pipelineID))
}

func (p *postgresAuditEventRepository) FindByRunID(runID int) (*domain.AuditEvent, error) {
	return scanAuditEvent(p.pool.QueryRow(context.Background(),
		`SELECT `+auditEventColumns+` FROM audit_events WHERE run_id = $1`, runID))
}

func (p *postgresAuditEventRepository) FindByEventType(eventType string) (*domain.AuditEvent, error) {
	return scanAuditEvent(p.pool.QueryRow(context.Background(),
		`SELECT `+auditEventColumns+` FROM audit_events WHERE event_type = $1`, eventType))
}

func (p *postgresAuditEventRepository) Update(auditevent *domain.AuditEvent) error {
	_, err := p.pool.Exec(context.Background(),
		`UPDATE audit_events SET pipeline_id = $1, run_id = $2, event_type = $3, created_at = $4 WHERE id = $5`,
		auditevent.PipelineID, auditevent.RunID, auditevent.EventType, auditevent.CreatedAt, auditevent.ID)
	return err
}

func (p *postgresAuditEventRepository) Delete(id int) error {
	_, err := p.pool.Exec(context.Background(), `DELETE FROM audit_events WHERE id = $1`, id)
	return err
}

func (p *postgresAuditEventRepository) FindAll() ([]domain.AuditEvent, error) {
	rows, err := p.pool.Query(context.Background(),
		`SELECT `+auditEventColumns+` FROM audit_events ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.AuditEvent
	for rows.Next() {
		var entity domain.AuditEvent
		if err := rows.Scan(&entity.ID, &entity.PipelineID, &entity.RunID, &entity.EventType, &entity.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, entity)
	}
	return result, rows.Err()
}

func scanAuditEvent(row pgx.Row) (*domain.AuditEvent, error) {
	var entity domain.AuditEvent
	if err := row.Scan(&entity.ID, &entity.PipelineID, &entity.RunID, &entity.EventType, &entity.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}
