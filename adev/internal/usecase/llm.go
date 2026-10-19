package usecase

import (
	"context"
	"encoding/json"

	"github.com/Reddetk/Adev/internal/domain"
)

// LLM is the model port of a stage agent. The adapter (the model gateway of M4)
// turns the request into a tool call or a JSON-mode completion; the service only
// looks at the payload of the response.
type LLM interface {
	Complete(ctx context.Context, request LLMRequest) (LLMResponse, error)
}

// LLMRequest carries everything an agent may see for one attempt.
type LLMRequest struct {
	Role        domain.Role
	Instruction string
	Shape       string
	Context     json.RawMessage
	// Problems lists the validation failures of the previous attempts. The model
	// must fix them; it never gets a second chance to guess silently.
	Problems []string
}

// LLMResponse is the payload the model wants to store in the output form.
type LLMResponse struct {
	Payload json.RawMessage
}
