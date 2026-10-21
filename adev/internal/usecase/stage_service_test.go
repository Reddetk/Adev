package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/repository"
	"github.com/Reddetk/Adev/internal/usecase"
)

// Scenarios live in docs/m2-scenarios.md; the test names match the ids there.
const (
	m2Pipeline  = "PIPE-M2"
	m2Target    = 1
	stageOutput = "FRM-" + m2Pipeline + "-00"
)

type scriptedLLM struct {
	responses []json.RawMessage
	errs      []error
	requests  []usecase.LLMRequest
}

func (f *scriptedLLM) Complete(_ context.Context, request usecase.LLMRequest) (usecase.LLMResponse, error) {
	index := len(f.requests)
	f.requests = append(f.requests, request)
	if index < len(f.errs) && f.errs[index] != nil {
		return usecase.LLMResponse{}, f.errs[index]
	}
	if index >= len(f.responses) {
		return usecase.LLMResponse{}, errors.New("scriptedLLM: the script is over")
	}
	return usecase.LLMResponse{
		Payload: f.responses[index],
		Usage:   usecase.LLMUsage{TokensIn: 10, TokensOut: 20, LatencyMs: 5},
	}, nil
}

type recordedRuns struct{ runs []usecase.StageRun }

func (r *recordedRuns) Record(_ context.Context, run usecase.StageRun) error {
	r.runs = append(r.runs, run)
	return nil
}

type recordedCalls struct{ calls []domain.ModelCall }

func (r *recordedCalls) Save(call *domain.ModelCall) error {
	r.calls = append(r.calls, *call)
	return nil
}

type m2Fixture struct {
	forms    *usecase.FormService
	store    *repository.MemoryFormsStore
	settings *usecase.SettingsService
	llm      *scriptedLLM
	runs     *recordedRuns
	calls    *recordedCalls
	stages   *usecase.StageService
}

func newM2Fixture(t *testing.T, responses ...json.RawMessage) *m2Fixture {
	t.Helper()
	fixture := &m2Fixture{
		store:    repository.NewMemoryFormsStore(),
		settings: usecase.NewSettingsService(repository.NewMemoryAgentConfigStore(m2Target)),
		llm:      &scriptedLLM{responses: responses},
		runs:     &recordedRuns{},
		calls:    &recordedCalls{},
	}
	fixture.forms = usecase.NewFormService(fixture.store, nil)
	fixture.stages = usecase.NewStageService(fixture.forms, fixture.settings, usecase.FixedLLM(fixture.llm), fixture.runs).
		WithModelCalls(fixture.calls)
	return fixture
}

// configure gives the target system a workspace and an analyst agent.
func (f *m2Fixture) configure(t *testing.T) {
	t.Helper()
	require.NoError(t, f.settings.SetWorkspace(context.Background(), m2Target, `D:\work\library`))
	require.NoError(t, f.settings.SetAgent(context.Background(), domain.AgentConfig{
		TargetSystemID: m2Target, Role: domain.RoleAnalyst, Provider: domain.ProviderOllama,
		BaseURL: "http://localhost:11434", Model: "qwen3-coder:30b",
	}))
}

// submitUserInput stores the input form of stage 00.
func (f *m2Fixture) submitUserInput(t *testing.T, version int) *domain.Form {
	t.Helper()
	envelope := domain.Envelope{
		FormID: "FRM-IN-M2", FormType: domain.TypeUserInput, SchemaVersion: 1, FormVersion: version,
		Status: domain.StatusDraft, Route: domain.RouteGreenfield, Stage: "—", Mode: domain.ModeFull,
		PipelineID: m2Pipeline, TargetSystemID: "1", ProducedBy: domain.RoleOperator,
		Inputs: []domain.Ref{},
	}
	if version > 1 {
		previous, err := f.forms.Latest(context.Background(), "FRM-IN-M2")
		require.NoError(t, err)
		envelope.PreviousFormVersion = &domain.Ref{
			FormID: previous.FormID, FormType: previous.FormType,
			FormVersion: previous.FormVersion, Hash: previous.PayloadHash,
		}
	}
	form := &domain.Form{Envelope: envelope}
	require.NoError(t, form.SetRawPayload(userInputPayload(version)))
	require.NoError(t, f.forms.Submit(context.Background(), form))
	return form
}

func (f *m2Fixture) runStage00(t *testing.T) (usecase.StageOutcome, error) {
	t.Helper()
	return f.stages.Run(context.Background(), usecase.StageRequest{
		PipelineID: m2Pipeline, TargetSystemID: m2Target,
		Route: domain.RouteGreenfield, StageID: "00",
	})
}

func userInputPayload(version int) json.RawMessage {
	text := "The operator needs a library accounting system."
	if version > 1 {
		text = "The operator needs a library accounting system with two branches."
	}
	raw, _ := json.Marshal(map[string]any{
		"rawRequest": map[string]any{"text": text, "lang": "STE100"},
		"source":     "operator",
		"receivedAt": "2026-10-19T21:00:00Z",
	})
	return raw
}

func validBRD() json.RawMessage {
	raw, _ := json.Marshal(map[string]any{
		"goals": []any{map[string]any{
			"id": "GOAL-1", "outcome": map[string]any{"text": "The system shall speed up the lending desk.", "lang": "STE100"},
			"metricId": "M-1",
		}},
		"metrics": []any{map[string]any{
			"id": "M-1", "name": "lend_seconds", "unit": "second", "baseline": 90, "target": 20,
		}},
		"scope":          map[string]any{"in": []any{}, "out": []any{}},
		"actors":         []any{},
		"successMetrics": []any{map[string]any{"metricId": "M-1", "threshold": "20", "window": "30 day"}},
	})
	return raw
}

func brdWithoutGoals() json.RawMessage {
	raw, _ := json.Marshal(map[string]any{"goals": []any{}, "metrics": []any{}})
	return raw
}

// --- H1 ---------------------------------------------------------------------

func TestH1Stage00StoresABRDOnTheFirstAttempt(t *testing.T) {
	fixture := newM2Fixture(t, validBRD())
	fixture.configure(t)
	input := fixture.submitUserInput(t, 1)

	outcome, err := fixture.runStage00(t)

	require.NoError(t, err)
	assert.False(t, outcome.Blocked)
	assert.Equal(t, stageOutput, outcome.FormID)
	assert.Equal(t, 1, outcome.FormVersion)
	assert.Equal(t, 1, outcome.Attempts)

	stored, err := fixture.forms.Latest(context.Background(), stageOutput)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusValid, stored.Status, "the gates passed and no question waits for the operator")
	require.Len(t, stored.Inputs, 1)
	assert.Equal(t, input.PayloadHash, stored.Inputs[0].Hash, "the BRD must reference the input it was built from")

	require.Len(t, fixture.runs.runs, 1)
	assert.Equal(t, "done", fixture.runs.runs[0].Status)
	assert.Equal(t, 1, fixture.runs.runs[0].Attempts)
}

// --- H2 ---------------------------------------------------------------------

func TestH2TheModelGetsTheGateFailureBackAndFixesIt(t *testing.T) {
	fixture := newM2Fixture(t, brdWithoutGoals(), validBRD())
	fixture.configure(t)
	fixture.submitUserInput(t, 1)

	outcome, err := fixture.runStage00(t)

	require.NoError(t, err)
	assert.False(t, outcome.Blocked)
	assert.Equal(t, 2, outcome.Attempts)
	require.Len(t, fixture.llm.requests, 2)
	assert.NotContains(t, fixture.llm.requests[0].User, "The previous answer failed these gates")
	assert.Contains(t, fixture.llm.requests[1].User, "The previous answer failed these gates")
	assert.Contains(t, fixture.llm.requests[1].User, "goals")

	stored, err := fixture.forms.Latest(context.Background(), stageOutput)
	require.NoError(t, err)
	assert.Equal(t, 1, stored.FormVersion, "a failed attempt must not store a version")
}

func TestTheModelCallOfAnAttemptIsRecorded(t *testing.T) {
	fixture := newM2Fixture(t, validBRD())
	fixture.configure(t)
	fixture.submitUserInput(t, 1)

	_, err := fixture.runStage00(t)
	require.NoError(t, err)

	require.Len(t, fixture.calls.calls, 1)
	call := fixture.calls.calls[0]
	assert.Equal(t, string(domain.RoleAnalyst), call.Profile, "the profile is the role that owns the stage")
	assert.NotEmpty(t, call.Model)
	assert.Equal(t, 10, call.TokensIn)
	assert.Equal(t, 20, call.TokensOut)
	assert.Equal(t, 5, call.LatencyMs)
}

// --- H3 ---------------------------------------------------------------------

func TestH3ARerunOfAValidFormKeepsTheVersion(t *testing.T) {
	fixture := newM2Fixture(t, validBRD(), validBRD())
	fixture.configure(t)
	fixture.submitUserInput(t, 1)

	first, err := fixture.runStage00(t)
	require.NoError(t, err)
	require.Equal(t, 1, first.FormVersion)

	second, err := fixture.runStage00(t)
	require.NoError(t, err)

	assert.Equal(t, 1, second.FormVersion, "a valid document is accepted; a re-run confirms it")
	assert.Equal(t, 0, second.Attempts, "the model is not called for a stored valid document")
	stored, err := fixture.forms.Latest(context.Background(), stageOutput)
	require.NoError(t, err)
	assert.Equal(t, 1, stored.FormVersion)
}

// --- N1 ---------------------------------------------------------------------

func TestN1AMissingInputStopsTheStage(t *testing.T) {
	fixture := newM2Fixture(t, validBRD())
	fixture.configure(t)

	outcome, err := fixture.runStage00(t)

	require.Error(t, err)
	assert.ErrorIs(t, err, usecase.ErrStageInputMissing)
	assert.False(t, outcome.Blocked)
	assert.Empty(t, fixture.runs.runs, "nothing is recorded when the stage cannot start")
	_, err = fixture.forms.Latest(context.Background(), stageOutput)
	assert.ErrorIs(t, err, domain.ErrFormNotFound)
	assert.Empty(t, fixture.llm.requests, "the model must not be called")
}

// --- N2 ---------------------------------------------------------------------

func TestN2AStageWithoutAnAgentConfigStops(t *testing.T) {
	fixture := newM2Fixture(t, validBRD())
	// The workspace exists, the analyst agent does not.
	require.NoError(t, fixture.settings.SetWorkspace(context.Background(), m2Target, `D:\work\library`))
	fixture.submitUserInput(t, 1)

	_, err := fixture.runStage00(t)

	require.Error(t, err)
	assert.ErrorIs(t, err, usecase.ErrAgentNotConfigured)
	assert.Empty(t, fixture.runs.runs)
	assert.Empty(t, fixture.llm.requests)
}

// --- N3 ---------------------------------------------------------------------

func TestN3ThreeRejectionsBlockTheStage(t *testing.T) {
	fixture := newM2Fixture(t, brdWithoutGoals(), brdWithoutGoals(), brdWithoutGoals())
	fixture.configure(t)
	fixture.submitUserInput(t, 1)

	outcome, err := fixture.runStage00(t)

	require.NoError(t, err, "a blocked stage is an outcome, not a crash")
	assert.True(t, outcome.Blocked)
	assert.Equal(t, 3, outcome.Attempts)
	assert.Len(t, outcome.Gaps, 3)
	_, err = fixture.forms.Latest(context.Background(), stageOutput)
	assert.ErrorIs(t, err, domain.ErrFormNotFound, "nothing is stored when every attempt fails")

	require.Len(t, fixture.runs.runs, 1)
	assert.Equal(t, "blocked", fixture.runs.runs[0].Status)
	assert.Contains(t, fixture.runs.runs[0].LastError, "goals")
}

// --- N4 ---------------------------------------------------------------------

func TestN4AModelFailureIsReported(t *testing.T) {
	fixture := newM2Fixture(t, validBRD())
	fixture.configure(t)
	fixture.submitUserInput(t, 1)
	fixture.llm.errs = []error{errors.New("connection refused")}

	_, err := fixture.runStage00(t)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
	require.Len(t, fixture.runs.runs, 1)
	assert.Equal(t, "failed", fixture.runs.runs[0].Status)
}

// --- N5 ---------------------------------------------------------------------

func TestN5AnUnknownStageStops(t *testing.T) {
	fixture := newM2Fixture(t, validBRD())
	fixture.configure(t)

	_, err := fixture.stages.Run(context.Background(), usecase.StageRequest{
		PipelineID: m2Pipeline, TargetSystemID: m2Target,
		Route: domain.RouteGreenfield, StageID: "99",
	})

	require.Error(t, err)
	assert.Empty(t, fixture.runs.runs)
	assert.Empty(t, fixture.llm.requests)
}

// --- N6 ---------------------------------------------------------------------

func TestN6ANonJSONPayloadCountsAsARejection(t *testing.T) {
	notJSON := json.RawMessage(`this is not json`)
	fixture := newM2Fixture(t, notJSON, notJSON, notJSON)
	fixture.configure(t)
	fixture.submitUserInput(t, 1)

	outcome, err := fixture.runStage00(t)

	require.NoError(t, err)
	assert.True(t, outcome.Blocked)
	assert.Len(t, outcome.Gaps, 3)
	assert.True(t, strings.Contains(strings.ToLower(outcome.Gaps[0]), "json"), "the reason must name the encoding problem")
}

// --- E2 ---------------------------------------------------------------------

func TestE2AValidPayloadOnTheLastAttemptSucceeds(t *testing.T) {
	fixture := newM2Fixture(t, brdWithoutGoals(), brdWithoutGoals(), validBRD())
	fixture.configure(t)
	fixture.submitUserInput(t, 1)

	outcome, err := fixture.runStage00(t)

	require.NoError(t, err)
	assert.False(t, outcome.Blocked, "the last allowed attempt must not be given up on")
	assert.Equal(t, 3, outcome.Attempts)
	assert.Equal(t, 1, outcome.FormVersion)

	require.Len(t, fixture.llm.requests, 3)
	assert.Equal(t, 2, strings.Count(fixture.llm.requests[2].User, "goals: must contain at least one entry"),
		"the last attempt must see both earlier failures")

	stored, err := fixture.forms.Latest(context.Background(), stageOutput)
	require.NoError(t, err)
	assert.Equal(t, 1, stored.FormVersion)

	require.Len(t, fixture.runs.runs, 1)
	assert.Equal(t, "done", fixture.runs.runs[0].Status)
	assert.Equal(t, 3, fixture.runs.runs[0].Attempts)
}

// --- E3 ---------------------------------------------------------------------

func TestE3TheStageReadsTheLatestVersionOfItsInput(t *testing.T) {
	fixture := newM2Fixture(t, validBRD())
	fixture.configure(t)
	fixture.submitUserInput(t, 1)
	latest := fixture.submitUserInput(t, 2)

	outcome, err := fixture.runStage00(t)

	require.NoError(t, err)
	assert.False(t, outcome.Blocked)

	stored, err := fixture.forms.Latest(context.Background(), stageOutput)
	require.NoError(t, err)
	require.Len(t, stored.Inputs, 1)
	assert.Equal(t, 2, stored.Inputs[0].FormVersion, "the latest input version must be the one referenced")
	assert.Equal(t, latest.PayloadHash, stored.Inputs[0].Hash, "the reference must carry the hash of that version")

	require.Len(t, fixture.llm.requests, 1)
	assert.Contains(t, fixture.llm.requests[0].User, "two branches", "the model must see the text of the latest version")
	assert.Contains(t, fixture.llm.requests[0].User, `"`+latest.FormID+`"`, "the context is keyed by the document id")
}
