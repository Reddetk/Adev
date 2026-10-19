package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/Reddetk/Adev/internal/domain"
)

// MemoryAgentConfigStore is an in-process settings store for tests and for runs
// without a database. An unknown target system is a not-found, exactly as in the
// pgx store, so a test cannot pass with data the real store would reject.
type MemoryAgentConfigStore struct {
	mu         sync.RWMutex
	known      map[int64]bool
	workspaces map[int64]string
	agents     map[int64]map[domain.Role]domain.AgentConfig
}

// NewMemoryAgentConfigStore registers the target systems that exist.
func NewMemoryAgentConfigStore(targetSystemIDs ...int64) *MemoryAgentConfigStore {
	known := make(map[int64]bool, len(targetSystemIDs))
	for _, id := range targetSystemIDs {
		known[id] = true
	}
	return &MemoryAgentConfigStore{
		known:      known,
		workspaces: map[int64]string{},
		agents:     map[int64]map[domain.Role]domain.AgentConfig{},
	}
}

// SetWorkspace writes the workspace directory of a known target system.
func (m *MemoryAgentConfigStore) SetWorkspace(ctx context.Context, targetSystemID int64, dir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.known[targetSystemID] {
		return domain.ErrSettingNotFound
	}
	m.workspaces[targetSystemID] = dir
	return nil
}

// Workspace reads the workspace directory. A known target system without a
// workspace returns an empty string, exactly as the column does in the pgx store.
func (m *MemoryAgentConfigStore) Workspace(ctx context.Context, targetSystemID int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.known[targetSystemID] {
		return "", domain.ErrSettingNotFound
	}
	return m.workspaces[targetSystemID], nil
}

// UpsertAgent writes one agent config of a known target system.
func (m *MemoryAgentConfigStore) UpsertAgent(ctx context.Context, config domain.AgentConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.known[config.TargetSystemID] {
		return domain.ErrSettingNotFound
	}
	roles, ok := m.agents[config.TargetSystemID]
	if !ok {
		roles = map[domain.Role]domain.AgentConfig{}
		m.agents[config.TargetSystemID] = roles
	}
	roles[config.Role] = config
	return nil
}

// Agents reads the agent configs of a target system, ordered by role.
func (m *MemoryAgentConfigStore) Agents(ctx context.Context, targetSystemID int64) ([]domain.AgentConfig, error) {
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
func (m *MemoryAgentConfigStore) Agent(ctx context.Context, targetSystemID int64, role domain.Role) (domain.AgentConfig, error) {
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
