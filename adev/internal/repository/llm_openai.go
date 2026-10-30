package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

// OpenAICompatibleFactory builds one model client per agent config. Ollama,
// OpenRouter, OpenAI, Qwen and DeepSeek all speak the same chat completions API,
// so one adapter covers them; the model gateway (M4) will add quotas and routing.
type OpenAICompatibleFactory struct {
	client *http.Client
}

// NewOpenAICompatibleFactory builds the factory.
func NewOpenAICompatibleFactory() *OpenAICompatibleFactory {
	// A local model on a full stage can take minutes.
	return &OpenAICompatibleFactory{client: &http.Client{Timeout: 10 * time.Minute}}
}

// apiEndpoint is the API root of every known provider. They all speak the same
// chat completions protocol, so one adapter serves them.
var apiEndpoint = map[domain.Provider]string{
	domain.ProviderOllama:     "http://localhost:11434/v1",
	domain.ProviderOpenAI:     "https://api.openai.com/v1",
	domain.ProviderOpenRouter: "https://openrouter.ai/api/v1",
	domain.ProviderOpencode:   "https://opencode.ai/inference/openai/v1",
	domain.ProviderDeepSeek:   "https://api.deepseek.com/v1",
	domain.ProviderQwen:       "https://dashscope.aliyuncs.com/compatible-mode/v1",
}

// For builds the client of one agent config.
func (f *OpenAICompatibleFactory) For(_ context.Context, agent domain.AgentConfig) (usecase.LLM, error) {
	baseURL := strings.TrimRight(agent.BaseURL, "/")
	endpoint := apiEndpoint[agent.Provider]

	switch {
	case baseURL == "":
		if endpoint == "" {
			return nil, fmt.Errorf("provider %q needs a baseUrl", agent.Provider)
		}
		baseURL = endpoint
	case agent.Provider == domain.ProviderOllama && isHostOnly(baseURL):
		// Ollama serves the OpenAI-compatible API under /v1. A config with the bare
		// host would otherwise post to /chat/completions and get a 404.
		baseURL += "/v1"
	case endpoint != "" && sameHost(baseURL, endpoint) && !strings.Contains(baseURL, "/v1"):
		// A pasted model page, such as https://openrouter.ai/<model>. The provider's
		// API root is the only useful value for that host.
		baseURL = endpoint
	}
	if strings.TrimSpace(agent.Model) == "" {
		return nil, fmt.Errorf("the agent config has no model")
	}
	return &openAICompatibleLLM{client: f.client, baseURL: baseURL, agent: agent}, nil
}

// isHostOnly reports a base URL with no path, such as http://localhost:11434.
func isHostOnly(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Path == ""
}

// sameHost reports whether two URLs address the same host.
func sameHost(left, right string) bool {
	first, err := url.Parse(left)
	if err != nil {
		return false
	}
	second, err := url.Parse(right)
	if err != nil {
		return false
	}
	return first.Host != "" && first.Host == second.Host
}

type openAICompatibleLLM struct {
	client  *http.Client
	baseURL string
	agent   domain.AgentConfig
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// Reasoning carries the thinking text of reasoning models; it is read only,
	// so the request side never sends it.
	Reasoning string `json:"reasoning,omitempty"`
}

type completionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

type completionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// maxTokensOrDefault keeps a zero (unset) config from inheriting a tiny provider
// default that truncates JSON documents mid-way. Reasoning models spend a large
// share of the budget before the answer, so the reserve is generous.
func maxTokensOrDefault(value int) int {
	if value <= 0 {
		return 32768
	}
	return value
}

// A model call retries a transient failure: a firewall refuses the first
// connections of a run, or a provider answers 429/5xx while it warms up. Ten
// attempts with a short backoff ride that out; the sum stays under the 10-minute
// client timeout.
// shortcut: a retried call may cost a second completion when the provider
// processed the request before the connection dropped; raise the backoff, not the
// count, when a provider starts charging for those.
const maxModelAttempts = 10

var (
	modelRetryBackoff    = 500 * time.Millisecond
	modelRetryMaxBackoff = 5 * time.Second
)

// retryModelCall reports whether the call may be tried again, waits the backoff
// and logs the failure. It stops on the last attempt and on a cancelled context.
func retryModelCall(ctx context.Context, attempt int, backoff *time.Duration, err error) bool {
	if attempt >= maxModelAttempts || ctx.Err() != nil {
		return false
	}
	log.Printf("model call: attempt %d/%d failed: %v: retrying in %s", attempt, maxModelAttempts, err, *backoff)
	timer := time.NewTimer(*backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
	}
	*backoff *= 2
	if *backoff > modelRetryMaxBackoff {
		*backoff = modelRetryMaxBackoff
	}
	return true
}

// transientHTTPStatus reports a status a retry can change: a provider that is
// warming up or rate-limiting, not a request the provider rejected.
func transientHTTPStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= http.StatusInternalServerError
}

// Complete asks the model for one payload. Attached text files travel inside
// the user message: every OpenAI-compatible provider accepts multiline text,
// while binary or provider-specific file parts do not.
func (l *openAICompatibleLLM) Complete(ctx context.Context, request usecase.LLMRequest) (usecase.LLMResponse, error) {
	user := request.UserWithFiles()
	body, err := json.Marshal(completionRequest{
		Model: l.agent.Model,
		Messages: []chatMessage{
			{Role: "system", Content: request.System},
			{Role: "user", Content: user},
		},
		Temperature: l.agent.Temperature,
		MaxTokens:   maxTokensOrDefault(l.agent.MaxTokens),
	})
	if err != nil {
		return usecase.LLMResponse{}, fmt.Errorf("model call: encode request: %w", err)
	}

	started := time.Now()
	backoff := modelRetryBackoff
	var raw []byte
	for attempt := 1; ; attempt++ {
		// Every attempt needs its own request: the body reader is consumed by Do.
		httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, l.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return usecase.LLMResponse{}, fmt.Errorf("model call: %w", err)
		}
		httpRequest.Header.Set("Content-Type", "application/json")
		if l.agent.APIKey != "" {
			httpRequest.Header.Set("Authorization", "Bearer "+l.agent.APIKey)
		}

		response, err := l.client.Do(httpRequest)
		if err != nil {
			if retryModelCall(ctx, attempt, &backoff, err) {
				continue
			}
			return usecase.LLMResponse{}, fmt.Errorf("model call to %s: %w", l.baseURL, err)
		}
		raw, err = io.ReadAll(io.LimitReader(response.Body, 4<<20))
		response.Body.Close()
		if err != nil {
			if retryModelCall(ctx, attempt, &backoff, err) {
				continue
			}
			return usecase.LLMResponse{}, fmt.Errorf("model call: read response: %w", err)
		}
		if response.StatusCode != http.StatusOK {
			callErr := fmt.Errorf("model call to %s failed: %s: %s", l.baseURL, response.Status, firstLine(raw))
			if transientHTTPStatus(response.StatusCode) && retryModelCall(ctx, attempt, &backoff, callErr) {
				continue
			}
			return usecase.LLMResponse{}, callErr
		}
		break
	}

	var parsed completionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return usecase.LLMResponse{}, fmt.Errorf("model call: decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return usecase.LLMResponse{}, fmt.Errorf("model call: the response has no choice: %s", firstLine(raw))
	}
	content := parsed.Choices[0].Message.Content
	// A reasoning model that ran out of tokens answers with an empty content
	// and the whole budget spent on thinking: keep the thinking text, so the
	// operator sees where the call went instead of an empty row.
	if content == "" {
		content = parsed.Choices[0].Message.Reasoning
	}
	return usecase.LLMResponse{
		Payload: extractJSON(content),
		Raw:     content,
		Usage: usecase.LLMUsage{
			TokensIn:  parsed.Usage.PromptTokens,
			TokensOut: parsed.Usage.CompletionTokens,
			LatencyMs: int(time.Since(started).Milliseconds()),
		},
	}, nil
}

// extractJSON cuts the payload out of an answer that may be wrapped in prose or in
// a code fence. A model that answers with no JSON at all keeps its text, and the
// form gates reject it, so the failure stays visible.
func extractJSON(answer string) json.RawMessage {
	text := strings.TrimSpace(answer)
	if strings.HasPrefix(text, "```") {
		if start := strings.Index(text, "\n"); start >= 0 {
			text = text[start+1:]
		}
		if end := strings.LastIndex(text, "```"); end >= 0 {
			text = text[:end]
		}
		text = strings.TrimSpace(text)
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return json.RawMessage(text)
	}
	return json.RawMessage(text[start : end+1])
}

func firstLine(raw []byte) string {
	line := string(raw)
	if index := strings.IndexAny(line, "\r\n"); index >= 0 {
		line = line[:index]
	}
	if len(line) > 200 {
		line = line[:200]
	}
	return line
}
