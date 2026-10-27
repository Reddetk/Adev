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

// Check calls the saved model of one role once.
func (c *ModelCheck) Check(ctx context.Context, targetSystemID int64, role domain.Role) (ModelCheckResult, error) {
	agent, err := c.settings.Agent(ctx, targetSystemID, role)
	if err != nil {
		return ModelCheckResult{}, fmt.Errorf("%w: role %q: %v", ErrAgentNotConfigured, role, err)
	}
	return c.call(ctx, agent)
}

// CheckDraft calls a config the operator typed but has not saved. A stage uses the
// saved config, and the browser shows the saved key only as hasApiKey, so a draft
// must be provable without overwriting the stored one.
func (c *ModelCheck) CheckDraft(ctx context.Context, draft domain.AgentConfig) (ModelCheckResult, error) {
	if !draft.Role.Valid() {
		return ModelCheckResult{}, domain.ErrInvalidRole
	}
	draft = c.settings.withStoredKey(ctx, draft)
	if err := draft.Validate(); err != nil {
		return ModelCheckResult{}, err
	}
	return c.call(ctx, draft)
}

func (c *ModelCheck) call(ctx context.Context, agent domain.AgentConfig) (ModelCheckResult, error) {
	client, err := c.llm.For(ctx, agent)
	if err != nil {
		return ModelCheckResult{}, err
	}

	system, user := promptMessages(
		`Answer with the payload only: {"ok":true}`,
		`{"ok":true}`,
		json.RawMessage(`{}`),
		nil,
	)
	response, err := client.Complete(ctx, LLMRequest{
		Role:   agent.Role,
		System: system,
		User:   user,
	})
	if err != nil {
		return ModelCheckResult{}, fmt.Errorf("model of role %q: %w", agent.Role, err)
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
