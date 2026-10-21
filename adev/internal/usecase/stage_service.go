package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/Reddetk/Adev/internal/domain"
)

var (
	// ErrStageInputMissing reports a required input form that does not exist yet.
	ErrStageInputMissing = errors.New("a required input form is missing")
	// ErrAgentNotConfigured reports a stage whose owner has no valid agent config.
	ErrAgentNotConfigured = errors.New("the agent config of this role is not set")
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
	Stage       string `json:"stage"`
	FormID      string `json:"formId"`
	FormVersion int    `json:"formVersion"`
	Attempts    int    `json:"attempts"`
	Blocked     bool   `json:"blocked"`
	// Gaps lists why the model could not satisfy the gates. A blocked stage becomes
	// an operator question in the next slice.
	Gaps []string `json:"gaps"`
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
	forms      *FormService
	settings   *SettingsService
	llm        LLMFactory
	runs       StageRunRecorder
	questions  *QuestionService
	activities *ActivityService
	maxTries   int
}

// NewStageService builds the stage runner. maxTries defaults to three.
func NewStageService(forms *FormService, settings *SettingsService, llm LLMFactory, runs StageRunRecorder) *StageService {
	return &StageService{forms: forms, settings: settings, llm: llm, runs: runs, maxTries: 3}
}

// WithQuestions attaches the question publisher. A stage without it blocks and
// returns its gaps, as before; a stage with it publishes an operator question.
func (s *StageService) WithQuestions(questions *QuestionService) *StageService {
	s.questions = questions
	return s
}

// WithActivity attaches the live book of runs, so the dashboard can show what
// the agents work on right now.
func (s *StageService) WithActivity(activities *ActivityService) *StageService {
	s.activities = activities
	return s
}

// Run executes one stage and returns its outcome. Every form type of the stage
// output is its own run: the architect writes the HLD and the ADR as two forms
// from the same context, and a blocked form stops the stage.
func (s *StageService) Run(ctx context.Context, request StageRequest) (StageOutcome, error) {
	stage, err := domain.StageByID(request.Route, request.StageID)
	if err != nil {
		return StageOutcome{}, err
	}

	agent, err := s.settings.Agent(ctx, request.TargetSystemID, stage.Owner)
	if err != nil {
		return StageOutcome{}, fmt.Errorf("%w: role %q: %v", ErrAgentNotConfigured, stage.Owner, err)
	}
	client, err := s.llm.For(ctx, agent)
	if err != nil {
		s.record(ctx, request, stage, "failed", 0, err.Error())
		return StageOutcome{}, fmt.Errorf("stage %s: model client: %w", stage.ID, err)
	}

	stageContext, refs, err := s.buildContext(ctx, request, stage)
	if err != nil {
		return StageOutcome{}, err
	}

	outcome := StageOutcome{Stage: stage.ID}
	for _, output := range stage.Outputs {
		formID := stageFormID(request.PipelineID, stage.ID, output, len(stage.Outputs) > 1)
		finished, version, previous, err := s.nextVersion(ctx, formID)
		if err != nil {
			return StageOutcome{}, err
		}
		if finished != nil {
			// The operator answer already made this version valid: the run just confirms it.
			outcome.FormID, outcome.FormVersion = finished.FormID, finished.FormVersion
			s.record(ctx, request, stage, "done", 0, "")
			continue
		}
		formOutcome, err := s.runForm(ctx, request, stage, agent, client, stageContext, refs, output, formID, version, previous)
		if err != nil || formOutcome.Blocked {
			return formOutcome, err
		}
		outcome.Attempts += formOutcome.Attempts
		outcome.FormID, outcome.FormVersion = formOutcome.FormID, formOutcome.FormVersion
	}
	return outcome, nil
}

// runForm fills one output form of the stage: up to maxTries model calls, each
// gate failure goes back to the model, and the last one becomes an operator question.
func (s *StageService) runForm(
	ctx context.Context,
	request StageRequest,
	stage domain.Stage,
	agent domain.AgentConfig,
	client LLM,
	stageContext json.RawMessage,
	refs []domain.Ref,
	output domain.FormType,
	formID string,
	version int,
	previous *domain.Ref,
) (StageOutcome, error) {
	outcome := StageOutcome{Stage: stage.ID}
	shape, err := domain.PayloadShape(output)
	if err != nil {
		s.record(ctx, request, stage, "failed", 0, err.Error())
		return outcome, err
	}

	// The dashboard reads the book to see what the agents work on right now.
	activity := s.activities.Start(request.PipelineID, stage.ID, string(stage.Owner), string(output))
	defer func() {
		if !activity.Finished() {
			activity.Finish("failed", "the run stopped before the stage completed")
		}
	}()

	// problems feeds every earlier failure back to the model, so it does not
	// regress; the operator sees only the failures of the last attempt.
	var problems, lastProblems []string
	var gateFailures, lastGateFailures []domain.ValidationError
	var form *domain.Form

	for attempt := 1; attempt <= s.maxTries; attempt++ {
		outcome.Attempts = attempt
		activity.Attempt()
		lastProblems, lastGateFailures = nil, nil

		system, user := stagePrompt(stage, output, shape, stageContext, problems)
		response, err := client.Complete(ctx, LLMRequest{
			Role:   agent.Role,
			System: system,
			User:   user,
		})
		if err != nil {
			activity.Finish("failed", err.Error())
			s.record(ctx, request, stage, "failed", attempt, err.Error())
			return StageOutcome{}, err
		}

		form = s.buildForm(request, stage, refs, output, formID, version, previous)
		// Gap pointers arrive in whatever syntax the model chose; questions
		// and answers need JSON Pointers, so they are fixed before storing.
		if err := form.SetRawPayload(normalizeGapPointers(response.Payload)); err != nil {
			problems = append(problems, err.Error())
			lastProblems = append(lastProblems, err.Error())
			continue
		}

		if err := s.forms.Submit(ctx, form); err != nil {
			if isFormRejection(err) {
				problems = append(problems, err.Error())
				lastProblems = append(lastProblems, err.Error())
				if failure := validationFailure(err); failure != nil {
					gateFailures = append(gateFailures, *failure)
					lastGateFailures = append(lastGateFailures, *failure)
				}
				continue
			}
			activity.Finish("failed", err.Error())
			s.record(ctx, request, stage, "failed", attempt, err.Error())
			return StageOutcome{}, err
		}

		outcome.FormID, outcome.FormVersion = form.FormID, form.FormVersion
		s.record(ctx, request, stage, "done", attempt, "")

		// The model filled the gates but may have left open gaps: the questions
		// hold the stage until the operator answers every one of them.
		blocked := false
		if s.questions != nil && request.PipelineID != "" {
			pipelineID, _ := strconv.Atoi(request.PipelineID)
			_, blocked, err = s.questions.PublishGaps(ctx, form, pipelineID)
			if err != nil {
				log.Printf("stage %s pipeline %s: publish gap: %v", stage.ID, request.PipelineID, err)
			}
		}
		if blocked {
			outcome.Blocked = true
			outcome.Gaps = []string{"the draft holds open questions for the operator"}
			s.record(ctx, request, stage, "blocked", attempt, "open questions")
			activity.Finish("blocked", "waiting for the operator answers")
		} else {
			activity.Finish("done", "")
		}
		return outcome, nil
	}

	outcome.Blocked = true
	outcome.Gaps = problems
	s.publishQuestion(ctx, request, stage, form, lastGateFailures)
	s.record(ctx, request, stage, "blocked", s.maxTries, strings.Join(lastProblems, "; "))
	activity.Finish("blocked", strings.Join(problems, "; "))
	return outcome, nil
}

// latestVersion reads the highest stored version of a form, or zero when absent.
func (s *StageService) latestVersion(ctx context.Context, formID string) int {
	latest, err := s.forms.Latest(ctx, formID)
	if err != nil {
		return 0
	}
	return latest.FormVersion
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

		// The key is the document, not its type: a stage may read two versions of
		// one type, and a change pipeline reads the as-is and the delta of the
		// same document.
		collected[latest.FormID] = projected
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

// nextVersion reports the version a re-run of the stage must produce. A stored
// version that already passed the gates is the finished result of the stage: the
// model is not called again, an operator answer closes the run without it.
func (s *StageService) nextVersion(ctx context.Context, formID string) (*domain.Form, int, *domain.Ref, error) {
	latest, err := s.forms.Latest(ctx, formID)
	switch {
	case err == nil:
		if latest.Status == domain.StatusValid {
			return latest, 0, nil, nil
		}
		return nil, latest.FormVersion + 1, &domain.Ref{
			FormID: latest.FormID, FormType: latest.FormType,
			FormVersion: latest.FormVersion, Hash: latest.PayloadHash,
		}, nil
	case errors.Is(err, domain.ErrFormNotFound):
		return nil, 1, nil, nil
	default:
		return nil, 0, nil, err
	}
}

func (s *StageService) buildForm(request StageRequest, stage domain.Stage, refs []domain.Ref, output domain.FormType, formID string, version int, previous *domain.Ref) *domain.Form {
	return &domain.Form{Envelope: domain.Envelope{
		FormID:              formID,
		FormType:            output,
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

// validationFailure returns the gate failure when the error carries one.
func validationFailure(err error) *domain.ValidationError {
	var failure *domain.ValidationError
	if errors.As(err, &failure) {
		return failure
	}
	return nil
}

// publishQuestion saves the failed draft and asks the operator about the first
// gate failure, so the next version of the form can be completed by an answer.
func (s *StageService) publishQuestion(ctx context.Context, request StageRequest, stage domain.Stage, form *domain.Form, failures []domain.ValidationError) {
	if s.questions == nil || form == nil {
		return
	}
	pipelineID, _ := strconv.Atoi(request.PipelineID)
	if err := s.questions.Publish(ctx, form, failures, pipelineID); err != nil {
		log.Printf("stage %s pipeline %s: publish question: %v", stage.ID, request.PipelineID, err)
	}
}

func (s *StageService) record(ctx context.Context, request StageRequest, stage domain.Stage, status string, attempts int, lastError string) {
	// The operator console reads these lines when a stage fails: it is the only place
	// that shows why the model or a gate refused an attempt.
	if lastError == "" {
		log.Printf("stage %s pipeline %s: %s after %d attempt(s)", stage.ID, request.PipelineID, status, attempts)
	} else {
		log.Printf("stage %s pipeline %s: %s after %d attempt(s): %s", stage.ID, request.PipelineID, status, attempts, lastError)
	}

	if s.runs == nil {
		return
	}
	// A bookkeeping failure must not undo a stored form.
	_ = s.runs.Record(ctx, StageRun{
		PipelineID: request.PipelineID, Stage: stage.ID,
		Status: status, Attempts: attempts, LastError: lastError,
	})
}

// stageFormID is stable per pipeline, stage and output type, so a re-run creates
// a new version. A stage with one output keeps the short id.
func stageFormID(pipelineID, stageID string, output domain.FormType, multi bool) string {
	if multi {
		return "FRM-" + pipelineID + "-" + stageID + "-" + string(output)
	}
	return "FRM-" + pipelineID + "-" + stageID
}
