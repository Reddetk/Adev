package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
)

const agentConfigColumns = `target_system_id, role, provider, base_url, model, api_key, temperature, max_tokens`

// PostgresAgentConfigStore persists target system settings in PostgreSQL.
type PostgresAgentConfigStore struct {
	pool *pgxpool.Pool
}

// NewPostgresAgentConfigStore uses an existing pool.
// A nil pool makes every call fail with ErrNoDatabase.
func NewPostgresAgentConfigStore(pool *pgxpool.Pool) *PostgresAgentConfigStore {
	return &PostgresAgentConfigStore{pool: pool}
}

func (s *PostgresAgentConfigStore) ready() error {
	if s.pool == nil {
		return domain.ErrNoDatabase
	}
	return nil
}

// SetWorkspace writes the workspace directory of a target system.
func (s *PostgresAgentConfigStore) SetWorkspace(ctx context.Context, targetSystemID int64, dir string) error {
	if err := s.ready(); err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE target_systems SET workspace_dir = $2 WHERE id = $1`, targetSystemID, dir)
	if err != nil {
		return fmt.Errorf("set workspace: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: target system %d", domain.ErrSettingNotFound, targetSystemID)
	}
	return nil
}

// Workspace reads the workspace directory of a target system.
func (s *PostgresAgentConfigStore) Workspace(ctx context.Context, targetSystemID int64) (string, error) {
	if err := s.ready(); err != nil {
		return "", err
	}
	var dir string
	err := s.pool.QueryRow(ctx, `SELECT workspace_dir FROM target_systems WHERE id = $1`, targetSystemID).Scan(&dir)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("%w: target system %d", domain.ErrSettingNotFound, targetSystemID)
	}
	if err != nil {
		return "", fmt.Errorf("workspace: %w", err)
	}
	return dir, nil
}

// UpsertAgent writes one agent config.
func (s *PostgresAgentConfigStore) UpsertAgent(ctx context.Context, config domain.AgentConfig) error {
	if err := s.ready(); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO agent_configs (target_system_id, role, provider, base_url, model, api_key, temperature, max_tokens)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (target_system_id, role) DO UPDATE SET
			provider = EXCLUDED.provider,
			base_url = EXCLUDED.base_url,
			model = EXCLUDED.model,
			api_key = EXCLUDED.api_key,
			temperature = EXCLUDED.temperature,
			max_tokens = EXCLUDED.max_tokens,
			updated_at = now()`,
		config.TargetSystemID, config.Role, config.Provider, config.BaseURL, config.Model,
		config.APIKey, config.Temperature, config.MaxTokens,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("%w: target system %d", domain.ErrSettingNotFound, config.TargetSystemID)
		}
		return fmt.Errorf("upsert agent: %w", err)
	}
	return nil
}

// Agents reads the agent configs of a target system.
func (s *PostgresAgentConfigStore) Agents(ctx context.Context, targetSystemID int64) ([]domain.AgentConfig, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+agentConfigColumns+` FROM agent_configs WHERE target_system_id = $1 ORDER BY role`, targetSystemID)
	if err != nil {
		return nil, fmt.Errorf("agents: %w", err)
	}
	defer rows.Close()

	var result []domain.AgentConfig
	for rows.Next() {
		config, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, config)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Role < result[j].Role })
	return result, nil
}

// Agent reads one agent config.
func (s *PostgresAgentConfigStore) Agent(ctx context.Context, targetSystemID int64, role domain.Role) (domain.AgentConfig, error) {
	if err := s.ready(); err != nil {
		return domain.AgentConfig{}, err
	}
	row := s.pool.QueryRow(ctx,
		`SELECT `+agentConfigColumns+` FROM agent_configs WHERE target_system_id = $1 AND role = $2`, targetSystemID, role)
	config, err := scanAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.AgentConfig{}, fmt.Errorf("%w: agent %s of target system %d", domain.ErrSettingNotFound, role, targetSystemID)
	}
	return config, err
}

func scanAgent(row pgx.Row) (domain.AgentConfig, error) {
	var config domain.AgentConfig
	if err := row.Scan(&config.TargetSystemID, &config.Role, &config.Provider, &config.BaseURL,
		&config.Model, &config.APIKey, &config.Temperature, &config.MaxTokens); err != nil {
		return domain.AgentConfig{}, err
	}
	return config, nil
}
