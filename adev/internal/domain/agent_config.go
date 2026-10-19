package domain

import (
	"fmt"
	"strings"
)

// Provider is the model provider of an agent.
type Provider string

const (
	ProviderOpenAI     Provider = "openai"
	ProviderOllama     Provider = "ollama"
	ProviderOpenRouter Provider = "openrouter"
	ProviderQwen       Provider = "qwen"
	ProviderDeepSeek   Provider = "deepseek"
	ProviderArk        Provider = "ark"
)

var knownProviders = map[Provider]bool{
	ProviderOpenAI: true, ProviderOllama: true, ProviderOpenRouter: true,
	ProviderQwen: true, ProviderDeepSeek: true, ProviderArk: true,
}

// Valid reports whether adev supports the provider.
func (p Provider) Valid() bool { return knownProviders[p] }

// NeedsAPIKey reports whether the provider needs a credential.
func (p Provider) NeedsAPIKey() bool { return p != ProviderOllama }

// AgentConfig is the model configuration of one role in one target system.
// APIKey is a secret. It never leaves the server.
type AgentConfig struct {
	TargetSystemID int64    `json:"targetSystemId"`
	Role           Role     `json:"role"`
	Provider       Provider `json:"provider"`
	BaseURL        string   `json:"baseUrl,omitempty"`
	Model          string   `json:"model"`
	APIKey         string   `json:"-"`
	Temperature    float64  `json:"temperature"`
	MaxTokens      int      `json:"maxTokens"`
}

// AgentView is the safe form of AgentConfig for an API response.
type AgentView struct {
	Role        Role     `json:"role"`
	Provider    Provider `json:"provider"`
	BaseURL     string   `json:"baseUrl,omitempty"`
	Model       string   `json:"model"`
	Temperature float64  `json:"temperature"`
	MaxTokens   int      `json:"maxTokens"`
	HasAPIKey   bool     `json:"hasApiKey"`
}

// Validate checks one agent config.
func (a AgentConfig) Validate() error {
	if !a.Provider.Valid() {
		return fmt.Errorf("%w: %q", ErrUnknownProvider, a.Provider)
	}
	if strings.TrimSpace(a.Model) == "" {
		return fmt.Errorf("model is required")
	}
	if a.Provider.NeedsAPIKey() && strings.TrimSpace(a.APIKey) == "" {
		return fmt.Errorf("%w for provider %q", ErrAPIKeyRequired, a.Provider)
	}
	if a.Provider == ProviderOllama && strings.TrimSpace(a.BaseURL) == "" {
		return fmt.Errorf("baseUrl is required for provider %q", ProviderOllama)
	}
	if a.Temperature < 0 || a.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if a.MaxTokens < 0 {
		return fmt.Errorf("maxTokens cannot be negative")
	}
	return nil
}

// Masked returns the safe form of the config.
func (a AgentConfig) Masked() AgentView {
	return AgentView{
		Role:        a.Role,
		Provider:    a.Provider,
		BaseURL:     a.BaseURL,
		Model:       a.Model,
		Temperature: a.Temperature,
		MaxTokens:   a.MaxTokens,
		HasAPIKey:   a.APIKey != "",
	}
}
