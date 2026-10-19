package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

func TestSettingsWorkspaceRoundTrip(t *testing.T) {
	service := NewSettingsService(newFakeAgentConfigStore())
	ctx := context.Background()

	if err := service.SetWorkspace(ctx, 7, `D:\work\demo`); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	dir, err := service.Workspace(ctx, 7)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	if dir != `D:\work\demo` {
		t.Fatalf("unexpected dir %q", dir)
	}
}

func TestSettingsRejectsEmptyWorkspace(t *testing.T) {
	service := NewSettingsService(newFakeAgentConfigStore())
	if err := service.SetWorkspace(context.Background(), 1, "   "); !errors.Is(err, domain.ErrWorkspaceRequired) {
		t.Fatalf("want ErrWorkspaceRequired, got %v", err)
	}
}

func TestSettingsRejectsUnknownRole(t *testing.T) {
	service := NewSettingsService(newFakeAgentConfigStore())
	config := domain.AgentConfig{
		TargetSystemID: 1, Role: domain.Role("mystery"), Provider: domain.ProviderOllama,
		BaseURL: "http://localhost:11434", Model: "m",
	}
	if err := service.SetAgent(context.Background(), config); !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("want ErrInvalidRole, got %v", err)
	}
}

func TestSettingsStoresAgentsOrderedByRole(t *testing.T) {
	service := NewSettingsService(newFakeAgentConfigStore())
	ctx := context.Background()

	local := domain.AgentConfig{
		TargetSystemID: 1, Role: domain.RoleAnalyst, Provider: domain.ProviderOllama,
		BaseURL: "http://localhost:11434", Model: "qwen3-coder:30b",
	}
	if err := service.SetAgent(ctx, local); err != nil {
		t.Fatalf("set local agent: %v", err)
	}
	cloud := domain.AgentConfig{
		TargetSystemID: 1, Role: domain.RoleArchitect, Provider: domain.ProviderOpenRouter,
		Model: "qwen/qwen3-235b", APIKey: "secret-key",
	}
	if err := service.SetAgent(ctx, cloud); err != nil {
		t.Fatalf("set cloud agent: %v", err)
	}

	agents, err := service.Agents(ctx, 1)
	if err != nil {
		t.Fatalf("agents: %v", err)
	}
	if len(agents) != 2 || agents[0].Role != domain.RoleAnalyst || agents[1].Role != domain.RoleArchitect {
		t.Fatalf("unexpected agents: %+v", agents)
	}
	if agents[1].APIKey != "secret-key" {
		t.Fatal("the store lost the api key")
	}
}
