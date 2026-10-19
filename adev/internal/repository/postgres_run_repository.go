package repository

import (
	"github.com/Reddetk/Adev/internal/usecase"

	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
)

const runColumns = `id, pipeline_id, stage, status, checkpoint_id, attempts, last_error`

type postgresRunRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRunRepository(pool *pgxpool.Pool) usecase.RunRepository {
	return &postgresRunRepository{pool: pool}
}

func (p *postgresRunRepository) Save(run *domain.Run) error {
	return p.pool.QueryRow(context.Background(),
		`INSERT INTO runs (pipeline_id, stage, status, checkpoint_id, attempts, last_error)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		run.PipelineID, run.Stage, run.Status, run.CheckpointID, run.Attempts, run.LastError,
	).Scan(&run.ID)
}

func (p *postgresRunRepository) FindByID(id int) (*domain.Run, error) {
	return scanRun(p.pool.QueryRow(context.Background(),
		`SELECT `+runColumns+` FROM runs WHERE id = $1`, id))
}

func (p *postgresRunRepository) FindByPipelineID(pipelineID int) (*domain.Run, error) {
	return scanRun(p.pool.QueryRow(context.Background(),
		`SELECT `+runColumns+` FROM runs WHERE pipeline_id = $1`, pipelineID))
}

func (p *postgresRunRepository) FindByStage(stage string) (*domain.Run, error) {
	return scanRun(p.pool.QueryRow(context.Background(),
		`SELECT `+runColumns+` FROM runs WHERE stage = $1`, stage))
}

func (p *postgresRunRepository) FindByStatus(status string) (*domain.Run, error) {
	return scanRun(p.pool.QueryRow(context.Background(),
		`SELECT `+runColumns+` FROM runs WHERE status = $1`, status))
}

func (p *postgresRunRepository) FindByCheckpointID(checkpointID string) (*domain.Run, error) {
	return scanRun(p.pool.QueryRow(context.Background(),
		`SELECT `+runColumns+` FROM runs WHERE checkpoint_id = $1`, checkpointID))
}

func (p *postgresRunRepository) FindByAttempts(attempts int) (*domain.Run, error) {
	return scanRun(p.pool.QueryRow(context.Background(),
		`SELECT `+runColumns+` FROM runs WHERE attempts = $1`, attempts))
}

func (p *postgresRunRepository) FindByLastError(lastError string) (*domain.Run, error) {
	return scanRun(p.pool.QueryRow(context.Background(),
		`SELECT `+runColumns+` FROM runs WHERE last_error = $1`, lastError))
}

func (p *postgresRunRepository) Update(run *domain.Run) error {
	_, err := p.pool.Exec(context.Background(),
		`UPDATE runs SET pipeline_id = $1, stage = $2, status = $3, checkpoint_id = $4,
		 attempts = $5, last_error = $6 WHERE id = $7`,
		run.PipelineID, run.Stage, run.Status, run.CheckpointID, run.Attempts, run.LastError, run.ID)
	return err
}

func (p *postgresRunRepository) Delete(id int) error {
	_, err := p.pool.Exec(context.Background(), `DELETE FROM runs WHERE id = $1`, id)
	return err
}

func (p *postgresRunRepository) FindAll() ([]domain.Run, error) {
	rows, err := p.pool.Query(context.Background(),
		`SELECT `+runColumns+` FROM runs ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Run
	for rows.Next() {
		var entity domain.Run
		if err := rows.Scan(&entity.ID, &entity.PipelineID, &entity.Stage, &entity.Status,
			&entity.CheckpointID, &entity.Attempts, &entity.LastError); err != nil {
			return nil, err
		}
		result = append(result, entity)
	}
	return result, rows.Err()
}

func scanRun(row pgx.Row) (*domain.Run, error) {
	var entity domain.Run
	if err := row.Scan(&entity.ID, &entity.PipelineID, &entity.Stage, &entity.Status,
		&entity.CheckpointID, &entity.Attempts, &entity.LastError); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}
