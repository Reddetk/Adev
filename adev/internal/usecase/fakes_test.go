package usecase

import (
	"context"
	"sort"
	"sync"

	"github.com/Reddetk/Adev/internal/domain"
)

// fakeAgentConfigStore is an in-process double of AgentConfigStore.
type fakeAgentConfigStore struct {
	mu         sync.RWMutex
	workspaces map[int64]string
	agents     map[int64]map[domain.Role]domain.AgentConfig
}

func newFakeAgentConfigStore() *fakeAgentConfigStore {
	return &fakeAgentConfigStore{
		workspaces: map[int64]string{},
		agents:     map[int64]map[domain.Role]domain.AgentConfig{},
	}
}

func (f *fakeAgentConfigStore) SetWorkspace(_ context.Context, targetSystemID int64, dir string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.workspaces[targetSystemID] = dir
	return nil
}

func (f *fakeAgentConfigStore) Workspace(_ context.Context, targetSystemID int64) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	dir, ok := f.workspaces[targetSystemID]
	if !ok {
		return "", domain.ErrSettingNotFound
	}
	return dir, nil
}

func (f *fakeAgentConfigStore) UpsertAgent(_ context.Context, config domain.AgentConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	roles, ok := f.agents[config.TargetSystemID]
	if !ok {
		roles = map[domain.Role]domain.AgentConfig{}
		f.agents[config.TargetSystemID] = roles
	}
	roles[config.Role] = config
	return nil
}

func (f *fakeAgentConfigStore) Agents(_ context.Context, targetSystemID int64) ([]domain.AgentConfig, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var result []domain.AgentConfig
	for _, config := range f.agents[targetSystemID] {
		result = append(result, config)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Role < result[j].Role })
	return result, nil
}

func (f *fakeAgentConfigStore) Agent(_ context.Context, targetSystemID int64, role domain.Role) (domain.AgentConfig, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	config, ok := f.agents[targetSystemID][role]
	if !ok {
		return domain.AgentConfig{}, domain.ErrSettingNotFound
	}
	return config, nil
}
