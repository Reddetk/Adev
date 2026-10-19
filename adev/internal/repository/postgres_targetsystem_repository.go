package repository

import (
	"github.com/Reddetk/Adev/internal/usecase"

	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
)

const targetSystemColumns = `id, name, repo_url, description`

type postgresTargetSystemRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTargetSystemRepository(pool *pgxpool.Pool) usecase.TargetSystemRepository {
	return &postgresTargetSystemRepository{pool: pool}
}

func (p *postgresTargetSystemRepository) Save(targetsystem *domain.TargetSystem) error {
	return p.pool.QueryRow(context.Background(),
		`INSERT INTO target_systems (name, repo_url, description)
		 VALUES ($1, $2, $3) RETURNING id`,
		targetsystem.Name, targetsystem.RepoURL, targetsystem.Description,
	).Scan(&targetsystem.ID)
}

func (p *postgresTargetSystemRepository) FindByID(id int) (*domain.TargetSystem, error) {
	return scanTargetSystem(p.pool.QueryRow(context.Background(),
		`SELECT `+targetSystemColumns+` FROM target_systems WHERE id = $1`, id))
}

func (p *postgresTargetSystemRepository) FindByName(name string) (*domain.TargetSystem, error) {
	return scanTargetSystem(p.pool.QueryRow(context.Background(),
		`SELECT `+targetSystemColumns+` FROM target_systems WHERE name = $1`, name))
}

func (p *postgresTargetSystemRepository) FindByRepoURL(repoURL string) (*domain.TargetSystem, error) {
	return scanTargetSystem(p.pool.QueryRow(context.Background(),
		`SELECT `+targetSystemColumns+` FROM target_systems WHERE repo_url = $1`, repoURL))
}

func (p *postgresTargetSystemRepository) FindByDescription(description string) (*domain.TargetSystem, error) {
	return scanTargetSystem(p.pool.QueryRow(context.Background(),
		`SELECT `+targetSystemColumns+` FROM target_systems WHERE description = $1`, description))
}

func (p *postgresTargetSystemRepository) Update(targetsystem *domain.TargetSystem) error {
	_, err := p.pool.Exec(context.Background(),
		`UPDATE target_systems SET name = $1, repo_url = $2, description = $3 WHERE id = $4`,
		targetsystem.Name, targetsystem.RepoURL, targetsystem.Description, targetsystem.ID)
	return err
}

func (p *postgresTargetSystemRepository) Delete(id int) error {
	_, err := p.pool.Exec(context.Background(), `DELETE FROM target_systems WHERE id = $1`, id)
	return err
}

func (p *postgresTargetSystemRepository) FindAll() ([]domain.TargetSystem, error) {
	rows, err := p.pool.Query(context.Background(),
		`SELECT `+targetSystemColumns+` FROM target_systems ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.TargetSystem
	for rows.Next() {
		var entity domain.TargetSystem
		if err := rows.Scan(&entity.ID, &entity.Name, &entity.RepoURL, &entity.Description); err != nil {
			return nil, err
		}
		result = append(result, entity)
	}
	return result, rows.Err()
}

func scanTargetSystem(row pgx.Row) (*domain.TargetSystem, error) {
	var entity domain.TargetSystem
	if err := row.Scan(&entity.ID, &entity.Name, &entity.RepoURL, &entity.Description); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &entity, nil
}
