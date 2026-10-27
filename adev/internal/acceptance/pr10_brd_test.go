//go:build tdd

package acceptance

// pr10_brd tests the two-phase BRD: the first call collects the operator
// questions, the operator answers map into the draft, and a later run assembles
// the document from the answered draft instead of inventing the values again.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/usecase"
)

func TestBRDStageAsksFirstAndAssemblesLater(t *testing.T) {
	questions := json.RawMessage(`{
		"scope": {"in": [], "out": [], "deferred": []},
		"metrics": [],
		"openGaps": [
			{"id": "GAP-1", "pointer": "/scope/in/-",
			 "question": {"text": "Which business process is in scope?", "lang": "STE100"},
			 "options": ["Book lending", "Warehouse"], "blocking": true},
			{"id": "GAP-2", "pointer": "/scope/out/-",
			 "question": {"text": "Which feature is out of scope?", "lang": "STE100"},
			 "options": ["Warehouse reports"], "blocking": true}
		]
	}`)
	respond := func(request usecase.LLMRequest) (usecase.LLMResponse, error) {
		if strings.Contains(request.User, "You collect the operator questions") {
			return usecase.LLMResponse{Payload: questions}, nil
		}
		return usecase.LLMResponse{Payload: validBRD(t)}, nil
	}
	fixture := newFixture(t, respond)

	// Phase A: the model collects the questions and the stage blocks.
	fixture.submit(t, "FRM-1-IN", domain.TypeUserInput, "—", mustJSON(t, domain.UserInput{
		RawRequest: ste("The operator needs a library accounting system."),
		Source:     domain.SourceOperator,
		ReceivedAt: "2026-10-19T21:00:00Z",
	}))
	fixture.run(t, "00")

	require.NotEmpty(t, fixture.llm.requests)
	assert.Contains(t, fixture.llm.requests[0].User, "You collect the operator questions")
	assert.Contains(t, fixture.llm.requests[0].User, "Do not invent any value")

	// The dashboard holds both questions and the stage waits for the answers.
	require.GreaterOrEqual(t, len(fixture.requests.saved), 2)
	savedPointers := []string{}
	for _, request := range fixture.requests.saved {
		savedPointers = append(savedPointers, request.Pointer)
	}
	assert.Contains(t, savedPointers, "/scope/in/-")
	assert.Contains(t, savedPointers, "/scope/out/-")

	// The operator answers one of the two questions in the dashboard.
	_, err := fixture.questions.AnswerOnPipeline(t.Context(), 1, "/scope/in/-", json.RawMessage(`"Book lending by mail"`))
	require.NoError(t, err)

	// Phase B: the re-run assembles the document from the answered draft.
	fixture.run(t, "00")

	require.GreaterOrEqual(t, len(fixture.llm.requests), 2)
	assembly := fixture.llm.requests[len(fixture.llm.requests)-1].User
	assert.Contains(t, assembly, "previousDraft", "the assembly reads the stored draft")
	assert.Contains(t, assembly, "Book lending by mail", "the operator answer stands in the draft")
}
