package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Reddetk/Adev/internal/domain"
)

// LLM is the model port of a stage agent. The adapter (the model gateway of M4)
// turns the request into a tool call or a JSON-mode completion; the service only
// looks at the payload of the response.
type LLM interface {
	Complete(ctx context.Context, request LLMRequest) (LLMResponse, error)
}

// LLMFile is one text file attached to a model call, rendered into the user
// message by the adapter. Only text attachments are portable across the
// OpenAI-compatible providers adev talks to.
type LLMFile struct {
	Name    string
	Content string
}

// LLMRequest carries the two messages of one attempt plus optional text files.
// The use case composes them; the adapter only sends them.
type LLMRequest struct {
	Role   domain.Role
	System string
	User   string
	Files  []LLMFile
}

// UserWithFiles appends the attached files to the user message exactly the way
// the adapter sends them, so the journal records what the model read. The tag
// delimits the file: the registry pages hold dozens of ``` fences, so a fence
// would end the attachment at the first one inside.
func (r LLMRequest) UserWithFiles() string {
	user := r.User
	for _, file := range r.Files {
		user += fmt.Sprintf("\n\n<file name=\"%s\">\n%s\n</file>", file.Name, file.Content)
	}
	return user
}

// LLMUsage is what one call cost: the token counts of the provider and the wall
// time the adapter measured. A provider that reports no usage leaves the counts at zero.
type LLMUsage struct {
	TokensIn  int
	TokensOut int
	LatencyMs int
}

// LLMResponse is the payload the model wants to store in the output form.
type LLMResponse struct {
	Payload json.RawMessage
	// Raw keeps the full model text when the payload did not survive, so the
	// operator can still see what the model actually said.
	Raw   string
	Usage LLMUsage
}

// LLMFactory builds the model client of one agent config. The model gateway (M4)
// will add quotas, routing and budgets here; today one adapter serves every
// OpenAI-compatible provider.
type LLMFactory interface {
	For(ctx context.Context, agent domain.AgentConfig) (LLM, error)
}

// FixedLLM returns a factory that serves the same client for every config.
// Tests use it; production code uses the real factory.
func FixedLLM(llm LLM) LLMFactory { return fixedLLM{llm: llm} }

type fixedLLM struct{ llm LLM }

func (f fixedLLM) For(context.Context, domain.AgentConfig) (LLM, error) { return f.llm, nil }
