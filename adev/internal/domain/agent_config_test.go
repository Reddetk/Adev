package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestAgentConfigRejectsCloudProviderWithoutKey(t *testing.T) {
	config := AgentConfig{
		TargetSystemID: 1, Role: RoleAnalyst, Provider: ProviderOpenAI, Model: "gpt-4o",
	}
	if err := config.Validate(); !errors.Is(err, ErrAPIKeyRequired) {
		t.Fatalf("want ErrAPIKeyRequired, got %v", err)
	}
}

func TestAgentConfigRejectsUnknownProvider(t *testing.T) {
	config := AgentConfig{
		TargetSystemID: 1, Role: RoleAnalyst, Provider: Provider("mystery"), Model: "x", APIKey: "k",
	}
	if err := config.Validate(); !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("want ErrUnknownProvider, got %v", err)
	}
}

func TestAgentConfigRequiresBaseURLForOllama(t *testing.T) {
	config := AgentConfig{
		TargetSystemID: 1, Role: RoleAnalyst, Provider: ProviderOllama, Model: "qwen3-coder",
	}
	if err := config.Validate(); err == nil || !strings.Contains(err.Error(), "baseUrl") {
		t.Fatalf("want a baseUrl error, got %v", err)
	}
}

func TestAgentConfigRejectsTemperatureOutOfRange(t *testing.T) {
	config := AgentConfig{
		TargetSystemID: 1, Role: RoleAnalyst, Provider: ProviderOllama,
		BaseURL: "http://localhost:11434", Model: "m", Temperature: 3,
	}
	if err := config.Validate(); err == nil || !strings.Contains(err.Error(), "temperature") {
		t.Fatalf("want a temperature error, got %v", err)
	}
}

func TestAgentConfigAcceptsLocalAndCloud(t *testing.T) {
	local := AgentConfig{
		TargetSystemID: 1, Role: RoleAnalyst, Provider: ProviderOllama,
		BaseURL: "http://localhost:11434", Model: "qwen3-coder:30b", Temperature: 0.2,
	}
	if err := local.Validate(); err != nil {
		t.Fatalf("local agent rejected: %v", err)
	}
	cloud := AgentConfig{
		TargetSystemID: 1, Role: RoleArchitect, Provider: ProviderOpenRouter,
		Model: "qwen/qwen3-235b", APIKey: "secret-key",
	}
	if err := cloud.Validate(); err != nil {
		t.Fatalf("cloud agent rejected: %v", err)
	}
}

func TestAgentConfigMaskedHidesTheAPIKey(t *testing.T) {
	config := AgentConfig{Role: RoleAnalyst, Provider: ProviderOpenAI, Model: "gpt-4o", APIKey: "secret"}
	view := config.Masked()
	if view.Model != "gpt-4o" || !view.HasAPIKey {
		t.Fatalf("unexpected view: %+v", view)
	}
	if strings.Contains(view.Model, "secret") {
		t.Fatal("the view leaked the api key")
	}
}
