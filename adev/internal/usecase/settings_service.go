package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/Reddetk/Adev/internal/domain"
)

// AgentConfigStore is the persistence port of the target system settings.
type AgentConfigStore interface {
	SetWorkspace(ctx context.Context, targetSystemID int64, dir string) error
	Workspace(ctx context.Context, targetSystemID int64) (string, error)
	UpsertAgent(ctx context.Context, config domain.AgentConfig) error
	Agents(ctx context.Context, targetSystemID int64) ([]domain.AgentConfig, error)
	Agent(ctx context.Context, targetSystemID int64, role domain.Role) (domain.AgentConfig, error)
}

// SettingsService validates settings before the store sees them.
type SettingsService struct {
	store AgentConfigStore
}

// NewSettingsService builds a settings service.
func NewSettingsService(store AgentConfigStore) *SettingsService {
	return &SettingsService{store: store}
}

// SetWorkspace stores the directory that agents may read and write.
func (s *SettingsService) SetWorkspace(ctx context.Context, targetSystemID int64, dir string) error {
	if targetSystemID < 1 {
		return fmt.Errorf("targetSystemId must be positive")
	}
	if strings.TrimSpace(dir) == "" {
		return domain.ErrWorkspaceRequired
	}
	return s.store.SetWorkspace(ctx, targetSystemID, strings.TrimSpace(dir))
}

// Workspace reads the workspace directory.
func (s *SettingsService) Workspace(ctx context.Context, targetSystemID int64) (string, error) {
	return s.store.Workspace(ctx, targetSystemID)
}

// SetAgent stores one agent config after validation.
func (s *SettingsService) SetAgent(ctx context.Context, config domain.AgentConfig) error {
	if config.TargetSystemID < 1 {
		return fmt.Errorf("targetSystemId must be positive")
	}
	if !config.Role.Valid() {
		return domain.ErrInvalidRole
	}
	// The API never sends a stored key back, so an empty key in a request means
	// "keep the saved one". Otherwise every save without retyping the key failed.
	config = s.withStoredKey(ctx, config)
	if err := config.Validate(); err != nil {
		return err
	}
	return s.store.UpsertAgent(ctx, config)
}

// withStoredKey fills an empty api key from the saved config of the same role and
// provider. A browser never receives the key, so an empty field means "keep it".
func (s *SettingsService) withStoredKey(ctx context.Context, config domain.AgentConfig) domain.AgentConfig {
	if strings.TrimSpace(config.APIKey) != "" {
		return config
	}
	if existing, err := s.store.Agent(ctx, config.TargetSystemID, config.Role); err == nil &&
		existing.Provider == config.Provider {
		config.APIKey = existing.APIKey
	}
	return config
}

// Agents reads the agent configs of a target system.
func (s *SettingsService) Agents(ctx context.Context, targetSystemID int64) ([]domain.AgentConfig, error) {
	return s.store.Agents(ctx, targetSystemID)
}

// Agent reads one agent config.
func (s *SettingsService) Agent(ctx context.Context, targetSystemID int64, role domain.Role) (domain.AgentConfig, error) {
	return s.store.Agent(ctx, targetSystemID, role)
}
