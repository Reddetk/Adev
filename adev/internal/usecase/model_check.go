package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Reddetk/Adev/internal/domain"
)

// ModelCheckResult is the answer of one agent config to a one line request.
type ModelCheckResult struct {
	Role      domain.Role     `json:"role"`
	Provider  domain.Provider `json:"provider"`
	Model     string          `json:"model"`
	HasAPIKey bool            `json:"hasApiKey"`
	Answer    string          `json:"answer"`
}

// ModelCheck asks the model of one role for a short answer. A stage hides a broken
// provider, a wrong key or an unknown model name behind one generic failure; this
// call shows the operator the real message before a stage runs.
type ModelCheck struct {
	settings *SettingsService
	llm      LLMFactory
}

// NewModelCheck builds the model checker.
func NewModelCheck(settings *SettingsService, llm LLMFactory) *ModelCheck {
	return &ModelCheck{settings: settings, llm: llm}
}

// Check calls the model of one role once.
func (c *ModelCheck) Check(ctx context.Context, targetSystemID int64, role domain.Role) (ModelCheckResult, error) {
	agent, err := c.settings.Agent(ctx, targetSystemID, role)
	if err != nil {
		return ModelCheckResult{}, fmt.Errorf("%w: role %q: %v", ErrAgentNotConfigured, role, err)
	}
	client, err := c.llm.For(ctx, agent)
	if err != nil {
		return ModelCheckResult{}, err
	}

	response, err := client.Complete(ctx, LLMRequest{
		Role:        agent.Role,
		Instruction: `Answer with the payload only: {"ok":true}`,
		Shape:       `{"ok":true}`,
		Context:     json.RawMessage(`{}`),
	})
	if err != nil {
		return ModelCheckResult{}, fmt.Errorf("model of role %q: %w", role, err)
	}

	// The key stays on the server: the result only reports that one is set.
	return ModelCheckResult{
		Role:      agent.Role,
		Provider:  agent.Provider,
		Model:     agent.Model,
		HasAPIKey: agent.APIKey != "",
		Answer:    string(response.Payload),
	}, nil
}
