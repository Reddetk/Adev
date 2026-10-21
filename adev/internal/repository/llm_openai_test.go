package repository

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

func TestOpenAICompatibleFactoryBuildsAProviderURL(t *testing.T) {
	factory := NewOpenAICompatibleFactory()
	ctx := context.Background()

	client, err := factory.For(ctx, domain.AgentConfig{Provider: domain.ProviderOllama, Model: "qwen3-coder:30b"})
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "http://localhost:11434/v1", client.(*openAICompatibleLLM).baseURL)

	// The bare host of a local Ollama is a common config; /v1 must be added once.
	client, err = factory.For(ctx, domain.AgentConfig{
		Provider: domain.ProviderOllama, BaseURL: "http://127.0.0.1:11434/", Model: "m",
	})
	require.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:11434/v1", client.(*openAICompatibleLLM).baseURL)

	// An explicit path is never rewritten.
	client, err = factory.For(ctx, domain.AgentConfig{
		Provider: domain.ProviderOllama, BaseURL: "http://127.0.0.1:11434/v1", Model: "m",
	})
	require.NoError(t, err)
	assert.Equal(t, "http://127.0.0.1:11434/v1", client.(*openAICompatibleLLM).baseURL)

	// A pasted OpenRouter model page becomes the API root: that page answers 404
	// with HTML and hides the real problem.
	client, err = factory.For(ctx, domain.AgentConfig{
		Provider: domain.ProviderOpenRouter,
		BaseURL:  "https://openrouter.ai/nvidia/nemotron:free",
		Model:    "nvidia/nemotron:free",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://openrouter.ai/api/v1", client.(*openAICompatibleLLM).baseURL)

	// A path on the provider host that already names the protocol stays as given.
	client, err = factory.For(ctx, domain.AgentConfig{
		Provider: domain.ProviderOpenAI, BaseURL: "https://api.openai.com/v1/", Model: "m",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://api.openai.com/v1", client.(*openAICompatibleLLM).baseURL)

	_, err = factory.For(ctx, domain.AgentConfig{Provider: domain.Provider("mystery"), Model: "m"})
	assert.Error(t, err, "a provider without a known endpoint needs a baseUrl")

	_, err = factory.For(ctx, domain.AgentConfig{Provider: domain.ProviderOpenRouter})
	assert.Error(t, err, "a model is required")

	client, err = factory.For(ctx, domain.AgentConfig{
		Provider: domain.ProviderOpenAI, BaseURL: "http://127.0.0.1:9/", Model: "m",
	})
	require.NoError(t, err, "an explicit baseUrl works for any provider")
	require.NotNil(t, client)
}

func TestOpenAICompatibleCompleteSendsTheStageRequest(t *testing.T) {
	var received struct {
		path          string
		authorization string
		body          map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		received.path = r.URL.Path
		received.authorization = r.Header.Get("Authorization")
		_ = json.Unmarshal(raw, &received.body)
		w.Header().Set("Content-Type", "application/json")
		// A fenced answer, as small models often produce.
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Here is the form:\n` + "```json" + `\n{\"goals\":[]}\n` + "```" + `"}}]}`))
	}))
	defer server.Close()

	client, err := NewOpenAICompatibleFactory().For(context.Background(), domain.AgentConfig{
		Provider: domain.ProviderOpenAI, BaseURL: server.URL, Model: "qwen3-coder:30b",
		APIKey: "sk-secret", Temperature: 0.2,
	})
	require.NoError(t, err)

	response, err := client.Complete(context.Background(), usecase.LLMRequest{
		Role:   domain.RoleAnalyst,
		System: "system prompt of the stage",
		User:   "projected input and gate failures",
	})
	require.NoError(t, err)

	assert.Equal(t, "/chat/completions", received.path)
	assert.Equal(t, "Bearer sk-secret", received.authorization)
	assert.Equal(t, "qwen3-coder:30b", received.body["model"])

	messages, ok := received.body["messages"].([]any)
	require.True(t, ok, "the request must carry messages")
	require.Len(t, messages, 2)
	system := messages[0].(map[string]any)["content"].(string)
	user := messages[1].(map[string]any)["content"].(string)
	assert.Equal(t, "system prompt of the stage", system, "the adapter sends the system message as given")
	assert.Equal(t, "projected input and gate failures", user, "the adapter sends the user message as given")

	assert.JSONEq(t, `{"goals":[]}`, string(response.Payload), "the code fence must be stripped")
}

func TestOpenAICompatibleCompleteReportsAnHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"model is loading"}}`))
	}))
	defer server.Close()

	client, err := NewOpenAICompatibleFactory().For(context.Background(), domain.AgentConfig{
		Provider: domain.ProviderOllama, BaseURL: server.URL, Model: "m",
	})
	require.NoError(t, err)

	_, err = client.Complete(context.Background(), usecase.LLMRequest{System: "x", User: "{}"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "model is loading")
}

func TestOpenAICompatibleCompleteRejectsAnEmptyChoice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client, err := NewOpenAICompatibleFactory().For(context.Background(), domain.AgentConfig{
		Provider: domain.ProviderOllama, BaseURL: server.URL, Model: "m",
	})
	require.NoError(t, err)

	_, err = client.Complete(context.Background(), usecase.LLMRequest{System: "x", User: "{}"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choice")
}

func TestExtractJSONKeepsPlainAndFencedAnswers(t *testing.T) {
	assert.Equal(t, `{"a":1}`, string(extractJSON(`{"a":1}`)))
	assert.Equal(t, `{"a":1}`, string(extractJSON("```json\n{\"a\":1}\n```")))
	assert.Equal(t, `{"a":1}`, string(extractJSON("Sure:\n{\"a\":1}\nDone.")))
	assert.Equal(t, "no json here", string(extractJSON("no json here")), "a plain answer stays visible for the gates to reject")
	assert.True(t, strings.HasPrefix(string(extractJSON("  {\"a\":1}  ")), "{"))
}
