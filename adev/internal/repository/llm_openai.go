package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// For builds the client of one agent config.
func (f *OpenAICompatibleFactory) For(_ context.Context, agent domain.AgentConfig) (usecase.LLM, error) {
	baseURL := strings.TrimRight(agent.BaseURL, "/")
	if baseURL == "" {
		switch agent.Provider {
		case domain.ProviderOllama:
			baseURL = "http://localhost:11434/v1"
		case domain.ProviderOpenAI:
			baseURL = "https://api.openai.com/v1"
		case domain.ProviderOpenRouter:
			baseURL = "https://openrouter.ai/api/v1"
		case domain.ProviderDeepSeek:
			baseURL = "https://api.deepseek.com/v1"
		case domain.ProviderQwen:
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		default:
			return nil, fmt.Errorf("provider %q needs a baseUrl", agent.Provider)
		}
	} else if agent.Provider == domain.ProviderOllama && isHostOnly(baseURL) {
		// Ollama serves the OpenAI-compatible API under /v1. A config with the bare
		// host would otherwise post to /chat/completions and get a 404.
		baseURL += "/v1"
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

type openAICompatibleLLM struct {
	client  *http.Client
	baseURL string
	agent   domain.AgentConfig
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
}

// Complete asks the model for one payload.
func (l *openAICompatibleLLM) Complete(ctx context.Context, request usecase.LLMRequest) (usecase.LLMResponse, error) {
	body, err := json.Marshal(completionRequest{
		Model: l.agent.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt(request)},
			{Role: "user", Content: userPrompt(request)},
		},
		Temperature: l.agent.Temperature,
		MaxTokens:   l.agent.MaxTokens,
	})
	if err != nil {
		return usecase.LLMResponse{}, fmt.Errorf("model call: encode request: %w", err)
	}

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
		return usecase.LLMResponse{}, fmt.Errorf("model call to %s: %w", l.baseURL, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return usecase.LLMResponse{}, fmt.Errorf("model call: read response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return usecase.LLMResponse{}, fmt.Errorf("model call to %s failed: %s: %s", l.baseURL, response.Status, firstLine(raw))
	}

	var parsed completionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return usecase.LLMResponse{}, fmt.Errorf("model call: decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return usecase.LLMResponse{}, fmt.Errorf("model call: the response has no choice")
	}
	return usecase.LLMResponse{Payload: extractJSON(parsed.Choices[0].Message.Content)}, nil
}

func systemPrompt(request usecase.LLMRequest) string {
	return "You fill one form of a software project. Answer with JSON only. " +
		"Do not add fields. Do not write code. Use short sentences. " +
		"The payload shape is: " + request.Shape
}

func userPrompt(request usecase.LLMRequest) string {
	var builder strings.Builder
	builder.WriteString(request.Instruction)
	builder.WriteString("\nInput forms:\n")
	builder.Write(request.Context)
	if len(request.Problems) > 0 {
		builder.WriteString("\nThe previous answer failed these gates. Fix them:\n")
		for _, problem := range request.Problems {
			builder.WriteString("- ")
			builder.WriteString(problem)
			builder.WriteString("\n")
		}
	}
	return builder.String()
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
