package repository

import (
	"github.com/Reddetk/Adev/internal/usecase"

	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
)

const pipelineColumns = `id, target_system_id, route, status, current_stage`

type postgresPipelineRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPipelineRepository(pool *pgxpool.Pool) usecase.PipelineRepository {
	return &postgresPipelineRepository{pool: pool}
}

func (p *postgresPipelineRepository) Save(pipeline *domain.Pipeline) error {
	return p.pool.QueryRow(context.Background(),
		`INSERT INTO pipelines (target_system_id, route, status, current_stage)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		pipeline.TargetSystemID, pipeline.Route, pipeline.Status, pipeline.CurrentStage,
	).Scan(&pipeline.ID)
}

func (p *postgresPipelineRepository) FindByID(id int) (*domain.Pipeline, error) {
	return scanPipeline(p.pool.QueryRow(context.Background(),
		`SELECT `+pipelineColumns+` FROM pipelines WHERE id = $1`, id))
}

func (p *postgresPipelineRepository) FindByTargetSystemID(targetSystemID int) (*domain.Pipeline, error) {
	return scanPipeline(p.pool.QueryRow(context.Background(),
		`SELECT `+pipelineColumns+` FROM pipelines WHERE target_system_id = $1`, targetSystemID))
}

func (p *postgresPipelineRepository) FindByRoute(route string) (*domain.Pipeline, error) {
	return scanPipeline(p.pool.QueryRow(context.Background(),
		`SELECT `+pipelineColumns+` FROM pipelines WHERE route = $1`, route))
}

func (p *postgresPipelineRepository) FindByStatus(status string) (*domain.Pipeline, error) {
	return scanPipeline(p.pool.QueryRow(context.Background(),
		`SELECT `+pipelineColumns+` FROM pipelines WHERE status = $1`, status))
}

func (p *postgresPipelineRepository) FindByCurrentStage(currentStage string) (*domain.Pipeline, error) {
	return scanPipeline(p.pool.QueryRow(context.Background(),
		`SELECT `+pipelineColumns+` FROM pipelines WHERE current_stage = $1`, currentStage))
}

func (p *postgresPipelineRepository) Update(pipeline *domain.Pipeline) error {
	_, err := p.pool.Exec(context.Background(),
		`UPDATE pipelines SET target_system_id = $1, route = $2, status = $3, current_stage = $4 WHERE id = $5`,
		pipeline.TargetSystemID, pipeline.Route, pipeline.Status, pipeline.CurrentStage, pipeline.ID)
	return err
}

func (p *postgresPipelineRepository) Delete(id int) error {
	_, err := p.pool.Exec(context.Background(), `DELETE FROM pipelines WHERE id = $1`, id)
	return err
}

func (p *postgresPipelineRepository) FindAll() ([]domain.Pipeline, error) {
	rows, err := p.pool.Query(context.Background(),
		`SELECT `+pipelineColumns+` FROM pipelines ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Pipeline
	for rows.Next() {
		var entity domain.Pipeline
		if err := rows.Scan(&entity.ID, &entity.TargetSystemID, &entity.Route, &entity.Status, &entity.CurrentStage); err != nil {
			return nil, err
		}
		result = append(result, entity)
	}
	return result, rows.Err()
}

func scanPipeline(row pgx.Row) (*domain.Pipeline, error) {
	var entity domain.Pipeline
	if err := row.Scan(&entity.ID, &entity.TargetSystemID, &entity.Route, &entity.Status, &entity.CurrentStage); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}
