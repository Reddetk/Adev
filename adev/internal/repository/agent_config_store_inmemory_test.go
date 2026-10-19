package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/Reddetk/Adev/internal/domain"
)

// inMemoryAgentConfigStore is an in-process settings store for tests.
type inMemoryAgentConfigStore struct {
	mu         sync.RWMutex
	workspaces map[int64]string
	agents     map[int64]map[domain.Role]domain.AgentConfig
}

// newInMemoryAgentConfigStore returns an empty store.
func newInMemoryAgentConfigStore() *inMemoryAgentConfigStore {
	return &inMemoryAgentConfigStore{
		workspaces: map[int64]string{},
		agents:     map[int64]map[domain.Role]domain.AgentConfig{},
	}
}

// SetWorkspace writes the workspace directory.
func (m *inMemoryAgentConfigStore) SetWorkspace(ctx context.Context, targetSystemID int64, dir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.workspaces[targetSystemID] = dir
	return nil
}

// Workspace reads the workspace directory.
func (m *inMemoryAgentConfigStore) Workspace(ctx context.Context, targetSystemID int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	dir, ok := m.workspaces[targetSystemID]
	if !ok {
		return "", domain.ErrSettingNotFound
	}
	return dir, nil
}

// UpsertAgent writes one agent config.
func (m *inMemoryAgentConfigStore) UpsertAgent(ctx context.Context, config domain.AgentConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	roles, ok := m.agents[config.TargetSystemID]
	if !ok {
		roles = map[domain.Role]domain.AgentConfig{}
		m.agents[config.TargetSystemID] = roles
	}
	roles[config.Role] = config
	return nil
}

// Agents reads the agent configs of a target system.
func (m *inMemoryAgentConfigStore) Agents(ctx context.Context, targetSystemID int64) ([]domain.AgentConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []domain.AgentConfig
	for _, config := range m.agents[targetSystemID] {
		result = append(result, config)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Role < result[j].Role })
	return result, nil
}

// Agent reads one agent config.
func (m *inMemoryAgentConfigStore) Agent(ctx context.Context, targetSystemID int64, role domain.Role) (domain.AgentConfig, error) {
	if err := ctx.Err(); err != nil {
		return domain.AgentConfig{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	config, ok := m.agents[targetSystemID][role]
	if !ok {
		return domain.AgentConfig{}, domain.ErrSettingNotFound
	}
	return config, nil
}
