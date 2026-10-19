package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Reddetk/Adev/internal/domain"
)

var (
	// ErrStageInputMissing reports a required input form that does not exist yet.
	ErrStageInputMissing = errors.New("a required input form is missing")
	// ErrAgentNotConfigured reports a stage whose owner has no valid agent config.
	ErrAgentNotConfigured = errors.New("the agent of this stage is not configured")
)

// StageRequest selects one stage of one pipeline.
type StageRequest struct {
	PipelineID     string
	TargetSystemID int64
	Route          domain.Route
	StageID        string
}

// StageOutcome is the result of one stage run.
type StageOutcome struct {
	Stage       string
	FormID      string
	FormVersion int
	Attempts    int
	Blocked     bool
	// Gaps lists why the model could not satisfy the gates. A blocked stage becomes
	// an operator question in the next slice.
	Gaps []string
}

// StageRun is what the orchestrator records about one attempt series.
type StageRun struct {
	PipelineID string
	Stage      string
	Status     string // done | blocked | failed
	Attempts   int
	LastError  string
}

// StageRunRecorder stores the result of a stage run.
type StageRunRecorder interface {
	Record(ctx context.Context, run StageRun) error
}

// StageService runs one stage: it reads the input forms, asks the model for the
// output form and stores it. The stage comes from the registry, never from the model.
type StageService struct {
	forms    *FormService
	settings *SettingsService
	llm      LLM
	runs     StageRunRecorder
	maxTries int
}

// NewStageService builds the stage runner. maxTries defaults to three.
func NewStageService(forms *FormService, settings *SettingsService, llm LLM, runs StageRunRecorder) *StageService {
	return &StageService{forms: forms, settings: settings, llm: llm, runs: runs, maxTries: 3}
}

// Run executes one stage and returns its outcome.
func (s *StageService) Run(ctx context.Context, request StageRequest) (StageOutcome, error) {
	stage, err := domain.StageByID(request.Route, request.StageID)
	if err != nil {
		return StageOutcome{}, err
	}

	agent, err := s.settings.Agent(ctx, request.TargetSystemID, stage.Owner)
	if err != nil {
		return StageOutcome{}, fmt.Errorf("%w: role %q: %v", ErrAgentNotConfigured, stage.Owner, err)
	}

	stageContext, refs, err := s.buildContext(ctx, request, stage)
	if err != nil {
		return StageOutcome{}, err
	}

	formID := stageFormID(request.PipelineID, stage.ID)
	version, previous, err := s.nextVersion(ctx, formID)
	if err != nil {
		return StageOutcome{}, err
	}

	outcome := StageOutcome{Stage: stage.ID}
	var problems []string

	for attempt := 1; attempt <= s.maxTries; attempt++ {
		outcome.Attempts = attempt

		response, err := s.llm.Complete(ctx, LLMRequest{
			Role:        agent.Role,
			Instruction: stageInstruction(stage),
			Shape:       payloadShape(stage.Outputs[0]),
			Context:     stageContext,
			Problems:    problems,
		})
		if err != nil {
			s.record(ctx, request, stage, "failed", attempt, err.Error())
			return StageOutcome{}, err
		}

		form := s.buildForm(request, stage, refs, formID, version, previous)
		if err := form.SetRawPayload(response.Payload); err != nil {
			problems = append(problems, err.Error())
			continue
		}

		if err := s.forms.Submit(ctx, form); err != nil {
			if isFormRejection(err) {
				problems = append(problems, err.Error())
				continue
			}
			s.record(ctx, request, stage, "failed", attempt, err.Error())
			return StageOutcome{}, err
		}

		outcome.FormID, outcome.FormVersion = form.FormID, form.FormVersion
		s.record(ctx, request, stage, "done", attempt, "")
		return outcome, nil
	}

	outcome.Blocked = true
	outcome.Gaps = problems
	s.record(ctx, request, stage, "blocked", s.maxTries, strings.Join(problems, "; "))
	return outcome, nil
}

// buildContext collects the projected input forms and the references of the stage.
func (s *StageService) buildContext(ctx context.Context, request StageRequest, stage domain.Stage) (json.RawMessage, []domain.Ref, error) {
	collected := map[string]any{}
	var refs []domain.Ref

	for _, inputType := range stage.Inputs {
		versions, err := s.forms.ByPipeline(ctx, request.PipelineID, inputType)
		if err != nil {
			return nil, nil, err
		}
		if len(versions) == 0 {
			return nil, nil, fmt.Errorf("%w: %s", ErrStageInputMissing, inputType)
		}
		latest := versions[len(versions)-1]

		fields, err := domain.ProjectionFor(inputType, stage.Owner)
		if err != nil {
			return nil, nil, err
		}
		projected, err := domain.ProjectFields(fields, latest)
		if err != nil {
			return nil, nil, err
		}

		collected[string(inputType)] = projected
		refs = append(refs, domain.Ref{
			FormID: latest.FormID, FormType: latest.FormType,
			FormVersion: latest.FormVersion, Hash: latest.PayloadHash,
		})
	}

	raw, err := json.Marshal(collected)
	if err != nil {
		return nil, nil, fmt.Errorf("stage context: %w", err)
	}
	return raw, refs, nil
}

// nextVersion reports the version a re-run of the stage must produce.
func (s *StageService) nextVersion(ctx context.Context, formID string) (int, *domain.Ref, error) {
	latest, err := s.forms.Latest(ctx, formID)
	switch {
	case err == nil:
		return latest.FormVersion + 1, &domain.Ref{
			FormID: latest.FormID, FormType: latest.FormType,
			FormVersion: latest.FormVersion, Hash: latest.PayloadHash,
		}, nil
	case errors.Is(err, domain.ErrFormNotFound):
		return 1, nil, nil
	default:
		return 0, nil, err
	}
}

func (s *StageService) buildForm(request StageRequest, stage domain.Stage, refs []domain.Ref, formID string, version int, previous *domain.Ref) *domain.Form {
	return &domain.Form{Envelope: domain.Envelope{
		FormID:              formID,
		FormType:            stage.Outputs[0],
		SchemaVersion:       1,
		FormVersion:         version,
		PreviousFormVersion: previous,
		Status:              domain.StatusDraft,
		Route:               request.Route,
		Stage:               stage.ID,
		Mode:                domain.ModeFull,
		PipelineID:          request.PipelineID,
		TargetSystemID:      strconv.FormatInt(request.TargetSystemID, 10),
		ProducedBy:          stage.Owner,
		Inputs:              refs,
	}}
}

// isFormRejection reports whether the form failed a gate, which the model can fix.
func isFormRejection(err error) bool {
	var validation *domain.ValidationError
	return errors.As(err, &validation) || errors.Is(err, domain.ErrNoPayloadGates)
}

func (s *StageService) record(ctx context.Context, request StageRequest, stage domain.Stage, status string, attempts int, lastError string) {
	if s.runs == nil {
		return
	}
	// A bookkeeping failure must not undo a stored form.
	_ = s.runs.Record(ctx, StageRun{
		PipelineID: request.PipelineID, Stage: stage.ID,
		Status: status, Attempts: attempts, LastError: lastError,
	})
}

// stageFormID is stable per pipeline and stage, so a re-run creates a new version.
func stageFormID(pipelineID, stageID string) string {
	return "FRM-" + pipelineID + "-" + stageID
}

func stageInstruction(stage domain.Stage) string {
	return fmt.Sprintf("You fill the %s form of stage %s. Answer with the payload only. Do not add fields.",
		stage.Outputs[0], stage.ID)
}

// payloadShape names the top level fields a form type expects.
func payloadShape(formType domain.FormType) string {
	switch formType {
	case domain.TypeBRD:
		return "goals[], metrics[], scope{in,out}, actors[], constraints[], assumptions[], successMetrics[], openGaps[]"
	case domain.TypeUserInput:
		return "rawRequest{text,lang}, attachments[], source, receivedAt, locale"
	default:
		return "unknown shape"
	}
}
