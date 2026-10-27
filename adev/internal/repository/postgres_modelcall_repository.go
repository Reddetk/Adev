package repository

import (
	"github.com/Reddetk/Adev/internal/usecase"

	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
)

const modelCallColumns = `id, run_id, profile, model, tokens_in, tokens_out, cost, latency_ms,
 stage, request, response, errors`

type postgresModelCallRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresModelCallRepository(pool *pgxpool.Pool) usecase.ModelCallRepository {
	return &postgresModelCallRepository{pool: pool}
}

func (p *postgresModelCallRepository) Save(modelcall *domain.ModelCall) error {
	return p.pool.QueryRow(context.Background(),
		`INSERT INTO model_calls (run_id, profile, model, tokens_in, tokens_out, cost, latency_ms,
		 stage, request, response, errors)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`,
		modelcall.RunID, modelcall.Profile, modelcall.Model, modelcall.TokensIn,
		modelcall.TokensOut, modelcall.Cost, modelcall.LatencyMs,
		modelcall.Stage, modelcall.Request, modelcall.Response, modelcall.Errors,
	).Scan(&modelcall.ID)
}

func (p *postgresModelCallRepository) FindByID(id int) (*domain.ModelCall, error) {
	return scanModelCall(p.pool.QueryRow(context.Background(),
		`SELECT `+modelCallColumns+` FROM model_calls WHERE id = $1`, id))
}

func (p *postgresModelCallRepository) FindByRunID(runID int) (*domain.ModelCall, error) {
	return scanModelCall(p.pool.QueryRow(context.Background(),
		`SELECT `+modelCallColumns+` FROM model_calls WHERE run_id = $1`, runID))
}

func (p *postgresModelCallRepository) FindByProfile(profile string) (*domain.ModelCall, error) {
	return scanModelCall(p.pool.QueryRow(context.Background(),
		`SELECT `+modelCallColumns+` FROM model_calls WHERE profile = $1`, profile))
}

func (p *postgresModelCallRepository) FindByModel(model string) (*domain.ModelCall, error) {
	return scanModelCall(p.pool.QueryRow(context.Background(),
		`SELECT `+modelCallColumns+` FROM model_calls WHERE model = $1`, model))
}

func (p *postgresModelCallRepository) FindByTokensIn(tokensIn int) (*domain.ModelCall, error) {
	return scanModelCall(p.pool.QueryRow(context.Background(),
		`SELECT `+modelCallColumns+` FROM model_calls WHERE tokens_in = $1`, tokensIn))
}

func (p *postgresModelCallRepository) FindByTokensOut(tokensOut int) (*domain.ModelCall, error) {
	return scanModelCall(p.pool.QueryRow(context.Background(),
		`SELECT `+modelCallColumns+` FROM model_calls WHERE tokens_out = $1`, tokensOut))
}

func (p *postgresModelCallRepository) FindByLatencyMs(latencyMs int) (*domain.ModelCall, error) {
	return scanModelCall(p.pool.QueryRow(context.Background(),
		`SELECT `+modelCallColumns+` FROM model_calls WHERE latency_ms = $1`, latencyMs))
}

func (p *postgresModelCallRepository) Update(modelcall *domain.ModelCall) error {
	_, err := p.pool.Exec(context.Background(),
		`UPDATE model_calls SET run_id = $1, profile = $2, model = $3, tokens_in = $4,
		 tokens_out = $5, cost = $6, latency_ms = $7, stage = $8, request = $9,
		 response = $10, errors = $11 WHERE id = $12`,
		modelcall.RunID, modelcall.Profile, modelcall.Model, modelcall.TokensIn,
		modelcall.TokensOut, modelcall.Cost, modelcall.LatencyMs, modelcall.Stage,
		modelcall.Request, modelcall.Response, modelcall.Errors, modelcall.ID)
	return err
}

func (p *postgresModelCallRepository) Delete(id int) error {
	_, err := p.pool.Exec(context.Background(), `DELETE FROM model_calls WHERE id = $1`, id)
	return err
}

func (p *postgresModelCallRepository) FindAll() ([]domain.ModelCall, error) {
	rows, err := p.pool.Query(context.Background(),
		`SELECT `+modelCallColumns+` FROM model_calls ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ModelCall
	for rows.Next() {
		var entity domain.ModelCall
		if err := rows.Scan(&entity.ID, &entity.RunID, &entity.Profile, &entity.Model,
			&entity.TokensIn, &entity.TokensOut, &entity.Cost, &entity.LatencyMs,
			&entity.Stage, &entity.Request, &entity.Response, &entity.Errors); err != nil {
			return nil, err
		}
		result = append(result, entity)
	}
	return result, rows.Err()
}

func scanModelCall(row pgx.Row) (*domain.ModelCall, error) {
	var entity domain.ModelCall
	if err := row.Scan(&entity.ID, &entity.RunID, &entity.Profile, &entity.Model,
		&entity.TokensIn, &entity.TokensOut, &entity.Cost, &entity.LatencyMs,
		&entity.Stage, &entity.Request, &entity.Response, &entity.Errors); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}
