package usecase

import (
	"context"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

func localAgent(role domain.Role) domain.AgentConfig {
	return domain.AgentConfig{
		TargetSystemID: 1, Role: role, Provider: domain.ProviderOllama,
		BaseURL: "http://localhost:11434", Model: "qwen3-coder:30b",
	}
}

func TestPlanReportsReadyWhenConfigured(t *testing.T) {
	ctx := context.Background()
	settings := NewSettingsService(newFakeAgentConfigStore())
	if err := settings.SetWorkspace(ctx, 1, `D:\work\library`); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	if err := settings.SetAgent(ctx, localAgent(domain.RoleAnalyst)); err != nil {
		t.Fatalf("set analyst: %v", err)
	}
	if err := settings.SetAgent(ctx, localAgent(domain.RoleArchitect)); err != nil {
		t.Fatalf("set architect: %v", err)
	}

	plan, err := NewPlanner(settings).Plan(ctx, 1, domain.RouteGreenfield)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if !plan.Readiness.Ready {
		t.Fatalf("want a ready plan, got %+v", plan.Readiness)
	}
	if len(plan.Stages) != 7 || plan.Stages[0].ID != "00" {
		t.Fatalf("unexpected stages: %+v", plan.Stages)
	}
}

func TestPlanReportsMissingSettings(t *testing.T) {
	plan, err := NewPlanner(NewSettingsService(newFakeAgentConfigStore())).
		Plan(context.Background(), 1, domain.RouteGreenfield)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if plan.Readiness.Ready || len(plan.Readiness.Missing) != 3 {
		t.Fatalf("want a workspace gap and two agent gaps, got %+v", plan.Readiness)
	}
}

func TestPlanRejectsUnknownRoute(t *testing.T) {
	_, err := NewPlanner(NewSettingsService(newFakeAgentConfigStore())).
		Plan(context.Background(), 1, domain.RouteChange)
	if err == nil {
		t.Fatal("the change route must not be planned yet")
	}
}
