package repository

import (
	"github.com/Reddetk/Adev/internal/usecase"

	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
)

const taskColumns = `id, pipeline_id, parent_id, owner, task_type, status, contract_ref`

type postgresTaskRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTaskRepository(pool *pgxpool.Pool) usecase.TaskRepository {
	return &postgresTaskRepository{pool: pool}
}

func (p *postgresTaskRepository) Save(task *domain.Task) error {
	return p.pool.QueryRow(context.Background(),
		`INSERT INTO tasks (pipeline_id, parent_id, owner, task_type, status, contract_ref)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		task.PipelineID, task.ParentID, task.Owner, task.TaskType, task.Status, task.ContractRef,
	).Scan(&task.ID)
}

func (p *postgresTaskRepository) FindByID(id int) (*domain.Task, error) {
	return scanTask(p.pool.QueryRow(context.Background(),
		`SELECT `+taskColumns+` FROM tasks WHERE id = $1`, id))
}

func (p *postgresTaskRepository) FindByPipelineID(pipelineID int) (*domain.Task, error) {
	return scanTask(p.pool.QueryRow(context.Background(),
		`SELECT `+taskColumns+` FROM tasks WHERE pipeline_id = $1`, pipelineID))
}

func (p *postgresTaskRepository) FindByParentID(parentID int) (*domain.Task, error) {
	return scanTask(p.pool.QueryRow(context.Background(),
		`SELECT `+taskColumns+` FROM tasks WHERE parent_id = $1`, parentID))
}

func (p *postgresTaskRepository) FindByOwner(owner string) (*domain.Task, error) {
	return scanTask(p.pool.QueryRow(context.Background(),
		`SELECT `+taskColumns+` FROM tasks WHERE owner = $1`, owner))
}

func (p *postgresTaskRepository) FindByTaskType(taskType string) (*domain.Task, error) {
	return scanTask(p.pool.QueryRow(context.Background(),
		`SELECT `+taskColumns+` FROM tasks WHERE task_type = $1`, taskType))
}

func (p *postgresTaskRepository) FindByStatus(status string) (*domain.Task, error) {
	return scanTask(p.pool.QueryRow(context.Background(),
		`SELECT `+taskColumns+` FROM tasks WHERE status = $1`, status))
}

func (p *postgresTaskRepository) FindByContractRef(contractRef string) (*domain.Task, error) {
	return scanTask(p.pool.QueryRow(context.Background(),
		`SELECT `+taskColumns+` FROM tasks WHERE contract_ref = $1`, contractRef))
}

func (p *postgresTaskRepository) Update(task *domain.Task) error {
	_, err := p.pool.Exec(context.Background(),
		`UPDATE tasks SET pipeline_id = $1, parent_id = $2, owner = $3, task_type = $4,
		 status = $5, contract_ref = $6 WHERE id = $7`,
		task.PipelineID, task.ParentID, task.Owner, task.TaskType, task.Status, task.ContractRef, task.ID)
	return err
}

func (p *postgresTaskRepository) Delete(id int) error {
	_, err := p.pool.Exec(context.Background(), `DELETE FROM tasks WHERE id = $1`, id)
	return err
}

func (p *postgresTaskRepository) FindAll() ([]domain.Task, error) {
	rows, err := p.pool.Query(context.Background(),
		`SELECT `+taskColumns+` FROM tasks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Task
	for rows.Next() {
		var entity domain.Task
		if err := rows.Scan(&entity.ID, &entity.PipelineID, &entity.ParentID, &entity.Owner,
			&entity.TaskType, &entity.Status, &entity.ContractRef); err != nil {
			return nil, err
		}
		result = append(result, entity)
	}
	return result, rows.Err()
}

func scanTask(row pgx.Row) (*domain.Task, error) {
	var entity domain.Task
	if err := row.Scan(&entity.ID, &entity.PipelineID, &entity.ParentID, &entity.Owner,
		&entity.TaskType, &entity.Status, &entity.ContractRef); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}
