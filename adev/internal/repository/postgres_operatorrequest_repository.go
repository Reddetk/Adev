package repository

import (
	"github.com/Reddetk/Adev/internal/usecase"

	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
)

const operatorRequestColumns = `id, pipeline_id, pointer, question, options, format, blocking, status`

// optionsOrEmpty writes an empty array for a question without choices: the column
// is NOT NULL, and a nil slice would travel as NULL and break the insert.
func optionsOrEmpty(options []string) []string {
	if options == nil {
		return []string{}
	}
	return options
}

type postgresOperatorRequestRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresOperatorRequestRepository(pool *pgxpool.Pool) usecase.OperatorRequestRepository {
	return &postgresOperatorRequestRepository{pool: pool}
}

func (p *postgresOperatorRequestRepository) Save(request *domain.OperatorRequest) error {
	return p.pool.QueryRow(context.Background(),
		`INSERT INTO operator_requests (pipeline_id, pointer, question, options, format, blocking, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		request.PipelineID, request.Pointer, request.Question, optionsOrEmpty(request.Options), request.Format, request.Blocking, request.Status,
	).Scan(&request.ID)
}

func (p *postgresOperatorRequestRepository) FindByID(id int) (*domain.OperatorRequest, error) {
	return scanOperatorRequest(p.pool.QueryRow(context.Background(),
		`SELECT `+operatorRequestColumns+` FROM operator_requests WHERE id = $1`, id))
}

func (p *postgresOperatorRequestRepository) FindByPipelineID(pipelineID int) (*domain.OperatorRequest, error) {
	return scanOperatorRequest(p.pool.QueryRow(context.Background(),
		`SELECT `+operatorRequestColumns+` FROM operator_requests WHERE pipeline_id = $1`, pipelineID))
}

func (p *postgresOperatorRequestRepository) FindByPointer(pointer string) (*domain.OperatorRequest, error) {
	return scanOperatorRequest(p.pool.QueryRow(context.Background(),
		`SELECT `+operatorRequestColumns+` FROM operator_requests WHERE pointer = $1 AND status = 'active' LIMIT 1`, pointer))
}

func (p *postgresOperatorRequestRepository) FindByQuestion(question string) (*domain.OperatorRequest, error) {
	return scanOperatorRequest(p.pool.QueryRow(context.Background(),
		`SELECT `+operatorRequestColumns+` FROM operator_requests WHERE question = $1`, question))
}

func (p *postgresOperatorRequestRepository) FindByStatus(status string) (*domain.OperatorRequest, error) {
	return scanOperatorRequest(p.pool.QueryRow(context.Background(),
		`SELECT `+operatorRequestColumns+` FROM operator_requests WHERE status = $1`, status))
}

func (p *postgresOperatorRequestRepository) Update(request *domain.OperatorRequest) error {
	_, err := p.pool.Exec(context.Background(),
		`UPDATE operator_requests SET pipeline_id = $1, pointer = $2, question = $3,
		 options = $4, format = $5, blocking = $6, status = $7 WHERE id = $8`,
		request.PipelineID, request.Pointer, request.Question, optionsOrEmpty(request.Options), request.Format, request.Blocking, request.Status, request.ID)
	return err
}

func (p *postgresOperatorRequestRepository) Delete(id int) error {
	_, err := p.pool.Exec(context.Background(), `DELETE FROM operator_requests WHERE id = $1`, id)
	return err
}

func (p *postgresOperatorRequestRepository) FindAll() ([]domain.OperatorRequest, error) {
	rows, err := p.pool.Query(context.Background(),
		`SELECT `+operatorRequestColumns+` FROM operator_requests ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.OperatorRequest
	for rows.Next() {
		var entity domain.OperatorRequest
		if err := rows.Scan(&entity.ID, &entity.PipelineID, &entity.Pointer, &entity.Question,
			&entity.Options, &entity.Format, &entity.Blocking, &entity.Status); err != nil {
			return nil, err
		}
		result = append(result, entity)
	}
	return result, rows.Err()
}

func scanOperatorRequest(row pgx.Row) (*domain.OperatorRequest, error) {
	var entity domain.OperatorRequest
	if err := row.Scan(&entity.ID, &entity.PipelineID, &entity.Pointer, &entity.Question,
		&entity.Options, &entity.Format, &entity.Blocking, &entity.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}
