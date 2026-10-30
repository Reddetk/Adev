package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/Reddetk/Adev/internal/domain"
)

var (
	// ErrStageInputMissing reports a required input form that does not exist yet.
	ErrStageInputMissing = errors.New("a required input form is missing")
	// ErrAgentNotConfigured reports a stage whose owner has no valid agent config.
	ErrAgentNotConfigured = errors.New("the agent config of this role is not set")
	// ErrStageStopped reports a run the operator cancelled mid-flight.
	ErrStageStopped = errors.New("the run was stopped by the operator")
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

// ModelCallRecorder stores what one model call cost. A stage treats it as
// bookkeeping: a failure to record never fails the stage.
type ModelCallRecorder interface {
	Save(call *domain.ModelCall) error
	Update(call *domain.ModelCall) error
}

// StageService runs one stage: it reads the input forms, asks the model for the
// output form and stores it. The stage comes from the registry, never from the model.
type StageService struct {
	forms      *FormService
	settings   *SettingsService
	llm        LLMFactory
	runs       StageRunRecorder
	modelCalls ModelCallRecorder
	questions  *QuestionService
	activities *ActivityService
	registry   *RegistryService
	// detailSem bounds the model calls of the detail walks: a slot (a channel
	// token) stands for one call in flight, shared by every level and pipeline.
	detailSem chan struct{}
}

// stageDetailWorkers caps the detail calls in flight. Four calls share one API
// key politely: the provider rate-limits per key, not per pipeline.
const stageDetailWorkers = 4

// NewStageService builds the stage runner.
func NewStageService(forms *FormService, settings *SettingsService, llm LLMFactory, runs StageRunRecorder) *StageService {
	return &StageService{forms: forms, settings: settings, llm: llm, runs: runs,
		detailSem: make(chan struct{}, stageDetailWorkers)}
}

// acquireDetailSlot blocks until a slot frees or the context ends.
func (s *StageService) acquireDetailSlot(ctx context.Context) bool {
	if s == nil || s.detailSem == nil {
		return ctx.Err() == nil
	}
	select {
	case s.detailSem <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *StageService) releaseDetailSlot() {
	if s != nil && s.detailSem != nil {
		<-s.detailSem
	}
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

// WithModelCalls attaches the cost book, so the operator sees the tokens and the
// time of every attempt.
func (s *StageService) WithModelCalls(calls ModelCallRecorder) *StageService {
	s.modelCalls = calls
	return s
}

// WithRegistry attaches the target system registry: a stored HLD proposes its
// containers, a stored LLD proposes its layers, packages and modules.
func (s *StageService) WithRegistry(registry *RegistryService) *StageService {
	s.registry = registry
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

	// The operator can stop any agent: the run context carries the cancel, and the
	// attempt loops leave at the next check.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if s.activities != nil {
		registration := s.activities.RegisterRun(request.PipelineID, string(stage.Owner), cancel)
		defer s.activities.DropRun(registration)
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
		// A stage that writes a type it also reads rewrites that document: the HLD
		// of stage 04 is the next version of the HLD of stage 03, not a twin.
		if input, ok := refOfType(refs, output); ok {
			formID, finished, version, previous = input.FormID, nil, input.FormVersion+1, &input
		}
		if finished != nil && finished.Status == domain.StatusWaitingOperator {
			// A run on a draft that still waits for the operator must not answer for
			// him: the model invented every glossary term and actor of a hotel BRD
			// that no operator had described. The questions stay and the stage
			// reports blocked until the operator answers them.
			pipelineID, _ := strconv.Atoi(request.PipelineID)
			if s.questions != nil && s.questions.HasPending(ctx, pipelineID, finished) {
				outcome.FormID, outcome.FormVersion = finished.FormID, finished.FormVersion
				outcome.Blocked = true
				outcome.Gaps = []string{"the draft holds open questions for the operator"}
				s.record(ctx, request, stage, "blocked", 0, "the draft holds open questions for the operator")
				return outcome, nil
			}
			// The operator asked for a new pass: the questions about this draft are
			// obsolete, and the model regenerates from the answers already given.
			if s.questions != nil {
				s.questions.CloseForForm(ctx, pipelineID, finished)
			}
			previous = &domain.Ref{FormID: finished.FormID, FormType: finished.FormType, FormVersion: finished.FormVersion}
			version = finished.FormVersion + 1
			finished = nil
		}
		if finished != nil {
			outcome.FormID, outcome.FormVersion = finished.FormID, finished.FormVersion
			// The operator answer already made this version valid: the run just confirms it.
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

// runForm fills one output form of a stage. Any answer is accepted: a gate
// mismatch rides on the stored draft as an indicator. The LLD retries a
// structural hole itself, because only the model can add a layer, a package or
// a path; a question for the operator stops every form and waits for the answer.
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

	var problems []string
	// A stage may rewrite a document that already passed the gates: stage 04
	// rewrites the HLD of stage 03. The newest accepted version is the floor of
	// the rewrite, so a model that returns an accepted field empty cannot lose it.
	var accepted json.RawMessage
	if versions, err := s.forms.ByPipeline(ctx, request.PipelineID, output); err == nil {
		for i := len(versions) - 1; i >= 0; i-- {
			if versions[i].Status == domain.StatusValid || versions[i].Status == domain.StatusApproved {
				accepted = versions[i].Payload
				break
			}
		}
	}
	for attempt := 1; ; attempt++ {
		outcome.Attempts = attempt
		activity.Attempt()

		// The BRD is filled in two phases: the first call collects the operator
		// questions, and every later run assembles the document from the answers
		// that already stand in the stored draft. The draft belongs to the same
		// document: a fresh output of a different type starts without it.
		resume := json.RawMessage(nil)
		if previous != nil && previous.FormType == output {
			if prior, err := s.forms.Get(ctx, previous.FormID, previous.FormVersion); err == nil && len(prior.Payload) > 0 {
				resume = prior.Payload
				if output == domain.TypeBRD {
					// The previous draft may still hold the metric questions of an
					// older run: a new system has no measured value, so they are gone.
					resume = dropBRDMetricGaps(resume)
				}
			}
		}

		system, user := stagePrompt(stage, output, shape, stageContext, problems)
		if output == domain.TypeBRD && previous == nil {
			system, user = questionsMessages(questionsInstruction(stage, output), stageContext)
		}
		if len(resume) > 0 {
			note := "The previousDraft holds the draft of this document and the operator answers: keep the answered values and fill only what the answers or the context cover."
			if len(problems) > 0 {
				note = "The previousDraft holds the last draft: keep every value that passed the gates and fix the gate failures."
			}
			user += "\n" + note + "\n" + string(resume) + "\n"
		}
		// Record the attempt before the model answers: the dashboard shows the prompt
		// and the "running" state at once, instead of waiting for the whole call.
		complete := LLMRequest{Role: agent.Role, System: system, User: user, Files: registryFiles(output)}
		call := s.startModelCall(request, agent, complete)
		response, err := client.Complete(ctx, complete)
		if err != nil {
			if ctx.Err() != nil {
				activity.Finish("stopped", "stopped by the operator")
				s.record(ctx, request, stage, "failed", attempt, ErrStageStopped.Error())
				return StageOutcome{}, ErrStageStopped
			}
			activity.Finish("failed", err.Error())
			s.record(ctx, request, stage, "failed", attempt, err.Error())
			// A call that never answered is on the book, too: the operator sees
			// the model request that produced the failure.
			s.finishModelCall(call, LLMResponse{}, nil, []string{err.Error()})
			return StageOutcome{}, err
		}

		form := s.buildForm(request, stage, refs, output, formID, version, previous)
		payload := normalizeGapPointers(response.Payload)
		if output == domain.TypeBRD {
			payload = dropBRDMetricGaps(payload)
		}
		payload = dropFilledGaps(payload)
		if output == domain.TypeFeasibility {
			payload = dropEmptyScopeDelta(payload)
		}
		// A composite document goes out in batches: the skeleton first, then one
		// call per entry, so no single answer has to carry the whole list.
		if plans, ok := batchPlans[output]; ok {
			payload = s.detailBatches(ctx, request, client, agent, stage, output, shape, stageContext, plans, payload, s.progressSaver(ctx, form))
			// The detail calls ask their own questions: their pointers read like the
			// payload root and take the same normalization as the skeleton.
			payload = normalizeGapPointers(payload)
		}
		// The model answers the meaning and loses the wrapper: a question arrives
		// as a plain string instead of the Text record. Repair the wrapping only;
		// the gates still judge the values.
		payload = coercePayload(output, payload)
		payload = normalizeLLDLayout(output, payload)
		payload = inheritAccepted(accepted, payload)
		// Unaskable gaps must leave the payload before the gates see it.
		payload = purgeUnaskableGaps(output, payload)
		if err := form.SetRawPayload(payload); err != nil {
			activity.Finish("failed", err.Error())
			s.record(ctx, request, stage, "failed", attempt, err.Error())
			s.finishModelCall(call, response, payload, []string{err.Error()})
			return StageOutcome{}, err
		}

		var mismatch []string
		var failures []domain.ValidationError
		if err := s.forms.Submit(ctx, form); err != nil {
			if errors.Is(err, domain.ErrVersionExists) {
				// The detail walk saved this draft after every finished entry: the
				// accepted payload rewrites that version instead of a new one.
				if updateErr := s.forms.UpdatePayload(ctx, form); updateErr != nil {
					activity.Finish("failed", updateErr.Error())
					s.record(ctx, request, stage, "failed", attempt, updateErr.Error())
					s.finishModelCall(call, response, payload, []string{updateErr.Error()})
					return StageOutcome{}, updateErr
				}
			} else if !isFormRejection(err) {
				activity.Finish("failed", err.Error())
				s.record(ctx, request, stage, "failed", attempt, err.Error())
				s.finishModelCall(call, response, payload, []string{err.Error()})
				return StageOutcome{}, err
			}
			if draftErr := s.forms.SubmitDraft(ctx, form); draftErr != nil {
				activity.Finish("failed", draftErr.Error())
				s.record(ctx, request, stage, "failed", attempt, draftErr.Error())
				s.finishModelCall(call, response, payload, []string{draftErr.Error()})
				return StageOutcome{}, draftErr
			}
			mismatch = []string{err.Error()}
			failures = collectFailures(err)
			log.Printf("stage %s: the draft holds a gate mismatch: %v", stage.ID, err)
		}

		outcome.FormID, outcome.FormVersion = form.FormID, form.FormVersion
		s.finishModelCall(call, response, payload, mismatch)
		s.registerObjects(ctx, request, form)

		// The model may have left open gaps: the questions hold the stage until the
		// operator answers every one of them. They come from the draft even when
		// its gates failed, so the operator can answer instead of rerunning.
		reasons := mismatch
		blocked := len(mismatch) > 0
		// A structural hole (a layer without a package, a contract without
		// paths) is the model's job: the next pass sees the gate failures and
		// the draft it just wrote, before any question goes out.
		if blocked && structuralRetry(output, attempt) {
			if s.questions != nil && request.PipelineID != "" {
				pipelineID, _ := strconv.Atoi(request.PipelineID)
				s.questions.CloseForForm(ctx, pipelineID, form)
			}
			problems = failureProblems(failures)
			if len(problems) == 0 {
				problems = mismatch
			}
			previous = &domain.Ref{FormID: form.FormID, FormType: form.FormType, FormVersion: form.FormVersion}
			version = form.FormVersion + 1
			continue
		}
		asked := false
		gapBlocked := false
		if s.questions != nil && request.PipelineID != "" {
			pipelineID, _ := strconv.Atoi(request.PipelineID)
			_, gapBlocked, err = s.questions.PublishGaps(ctx, form, pipelineID)
			if err != nil {
				log.Printf("stage %s pipeline %s: publish gap: %v", stage.ID, request.PipelineID, err)
			}
			if gapBlocked {
				blocked = true
				reasons = append(reasons, "the draft holds open questions for the operator")
			} else if blocked {
				// No gap went out: the gate mismatch itself asks the operator, the
				// same repair path the answer flow uses.
				asked, err = s.questions.PublishGate(ctx, form, failures, pipelineID)
				if err != nil {
					log.Printf("stage %s pipeline %s: publish gate: %v", stage.ID, request.PipelineID, err)
				}
				if asked {
					reasons = append(reasons, "the draft holds open questions for the operator")
				}
			}
		}
		if blocked {
			outcome.Blocked = true
			outcome.Gaps = reasons
			s.record(ctx, request, stage, "blocked", attempt, strings.Join(reasons, "; "))
			activity.Finish("blocked", strings.Join(reasons, "; "))
			return outcome, nil
		}
		// Nothing waits for the operator and the gates passed: the document is accepted.
		s.markValid(ctx, form)
		// A retry may have filled the field a gate question asked about: the
		// accepted version makes that question moot, and the dashboard would keep
		// it blocking otherwise.
		if s.questions != nil && request.PipelineID != "" {
			pipelineID, _ := strconv.Atoi(request.PipelineID)
			s.questions.CloseStale(ctx, pipelineID)
		}
		s.record(ctx, request, stage, "done", attempt, "")
		activity.Finish("done", "")
		return outcome, nil
	}
}

// maxLLDFormAttempts caps the structural retries of a design document. A model
// that cannot fill the layout in three passes leaves the draft to the operator.
// shortcut: three passes fit the live runs; raise when a larger system needs more.
const maxLLDFormAttempts = 3

// structuralRetry reports whether the next model pass can fill a structural hole.
// A layout the operator cannot answer (an empty container list, a missing
// degradation rule) is the model's job: the retry runs before a question is
// published, and the attempt cap stops a model that cannot fill it.
func structuralRetry(output domain.FormType, attempt int) bool {
	return (output == domain.TypeLLD || output == domain.TypeHLD) && attempt < maxLLDFormAttempts
}

// failureProblems names each gate failure for the retry prompt.
func failureProblems(failures []domain.ValidationError) []string {
	out := make([]string, 0, len(failures))
	for _, failure := range failures {
		out = append(out, failure.Field+": "+failure.Problem)
	}
	return out
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
		if latest.Status == domain.StatusWaitingOperator {
			// The stage already waits on the operator: another model call would
			// only stack a draft over the waiting version and hide its questions.
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

// batchPlan names one list of a document. Children name the lists nested inside
// each entry: every level of nesting is its own model call, so no single answer
// carries the whole document.
type batchPlan struct {
	kind    string // the JSON name of the list
	idField string // the JSON name of the field that identifies one entry (default "id")
	with    string // the JSON name of a companion list at the document root
	ref     string // the field of a companion entry that names its owner
	// children lists live inside each entry of this list.
	children []batchPlan
}

// idFieldOf returns the id field of a plan; the fr, nfr and tasks lists use id.
func (p batchPlan) idFieldOf() string {
	if p.idField == "" {
		return "id"
	}
	return p.idField
}

// batchPlans marks the documents that are filled entry by entry. The skeleton
// call returns the outline; every later call details one entry of one level. A
// document that names its entries in the first answer gets the same treatment at
// every stage: the model lists the entries, then answers each one alone.
var batchPlans = map[domain.FormType][]batchPlan{
	domain.TypeSRS:          {{kind: "fr"}, {kind: "nfr"}},
	domain.TypeFeasibility:  {{kind: "options"}},
	domain.TypeADR:          {{kind: "decisions"}},
	domain.TypeTaskPlan:     {{kind: "tasks"}},
	domain.TypeTestContract: {{kind: "tests"}},
	// The HLD is detailed by one call per container, which writes the aggregates
	// with their invariants of that container, and then by one call per decision.
	domain.TypeHLD: {
		{kind: "containers", idField: "objectId", with: "aggregates", ref: "containerId"},
		{kind: "decisions"},
	},
	domain.TypeLLD: {
		{kind: "containers", idField: "containerId", children: []batchPlan{
			{kind: "layers", idField: "name", children: []batchPlan{
				{kind: "packages", idField: "path", children: []batchPlan{
					{kind: "modules", idField: "objectId"},
				}},
			}},
		}},
		{kind: "contracts", with: "paths", ref: "contractRef"},
	},
}

// alias names the walk from the payload root to one list, level by level.
type entryLabel struct {
	plan    batchPlan
	idValue string
}

// detailResult is one finished unit of a parallel detail walk. The worker sends
// it over the results channel; the receiver merges it into the document.
type detailResult struct {
	index   int
	labels  []entryLabel
	echo    json.RawMessage
	updated json.RawMessage
	gaps    []json.RawMessage
	ok      bool
}

// detailCheckpoint saves the document built so far. A detail call can take a
// minute, so a stage that runs for an hour must not lose the entries that already
// answered when the process stops or the operator stops it.
type detailCheckpoint func(document json.RawMessage) error

// detailBatches asks the model to fill one entry per call and merges the answers
// back into the skeleton. A failed detail call keeps the skeleton entry: the gates
// still judge the document, and the operator sees what is missing.
func (s *StageService) detailBatches(ctx context.Context, request StageRequest, client LLM, agent domain.AgentConfig, stage domain.Stage, output domain.FormType, shape string, stageContext json.RawMessage, plans []batchPlan, payload json.RawMessage, checkpoint detailCheckpoint) json.RawMessage {
	var gaps []json.RawMessage
	filled := s.detailLevel(ctx, request, client, agent, stage, output, shape, stageContext, plans, payload, payload, "", nil, &gaps, checkpoint)
	return mergeDetailGaps(filled, gaps)
}

// progressSaver writes the document built so far into the draft of this form. The
// draft keeps its version: every finished entry rewrites it, so a run of an hour
// leaves the work on disk and the next run reads it back as the previous draft.
// shortcut: a resumed run still calls the model per entry and keeps the answers,
// not the calls; skip an entry whose subtree is already filled when the repeat
// becomes the bottleneck.
func (s *StageService) progressSaver(ctx context.Context, form *domain.Form) detailCheckpoint {
	return func(document json.RawMessage) error {
		partial := *form
		if err := partial.SetRawPayload(document); err != nil {
			return err
		}
		return s.forms.SubmitDraft(ctx, &partial)
	}
}

// detailEntry runs one entry of one plan: the model call, the echo walk, the
// entry gaps, and the child levels of the entry. It is a pure unit of work: all
// merging into the document happens in the receiver, entries never share state.
func (s *StageService) detailEntry(
	ctx context.Context, request StageRequest, client LLM, agent domain.AgentConfig,
	stage domain.Stage, output domain.FormType, shape string, stageContext json.RawMessage,
	plan batchPlan, labels []entryLabel, entry json.RawMessage, entryPath string, document json.RawMessage,
) detailResult {
	echo := s.detailCall(ctx, request, client, agent, stage, output, shape, stageContext, labels, entry, plan, document)
	// The system owns the identifiers and the companion reference: the model
	// answers the meaning only, and the merge needs the keys it did not send.
	echo = withEntryKeys(echo, plan, labels)
	updated, ok := entryFromEcho(echo, labels)
	if !ok {
		id := lastValue(labels)
		log.Printf("stage %s: detail %s %s: the answer does not carry the entry", stage.ID, plan.kind, id)
		return detailResult{labels: labels, echo: echo}
	}
	// The model writes the meaning, the system writes the document: a field the
	// answer leaves out keeps the value the skeleton already holds.
	updated = mergeEntryFields(entry, updated)
	var gaps []json.RawMessage
	gaps = append(gaps, detailGaps(echo, entryPath)...)
	if len(plan.children) > 0 {
		// The children walk runs inside this unit: its calls take the same slots,
		// so the pool bound holds for the whole subtree. It saves no progress: a
		// child holds a fragment of the document, not the whole one.
		updated = s.detailLevel(ctx, request, client, agent, stage, output, shape, stageContext, plan.children, updated, document, entryPath, labels, &gaps, nil)
	}
	return detailResult{labels: labels, echo: echo, updated: updated, gaps: gaps, ok: true}
}

// detailLevel fills one level of the document in parallel, following the
// concurrent fork-join pattern: fork — one goroutine per entry, join — the
// results channel is closed once every fork has joined. The receiver owns the
// document and merges one result at a time, so no merge ever races another.
// Children of an entry wait for its answer; siblings run at once. The in-flight
// bound is the detailSem channel the calls take, not the goroutine count.
func (s *StageService) detailLevel(
	ctx context.Context, request StageRequest, client LLM, agent domain.AgentConfig,
	stage domain.Stage, output domain.FormType, shape string, stageContext json.RawMessage,
	plans []batchPlan, doc json.RawMessage, document json.RawMessage, basePath string, parents []entryLabel, gaps *[]json.RawMessage,
	checkpoint detailCheckpoint,
) json.RawMessage {
	for _, plan := range plans {
		entries, err := listOf(doc, plan.kind)
		if err != nil {
			continue
		}
		// The model sometimes leaves the identifier out of the outline. The id is
		// structural: without it the detail walk skips the entry, and the gates ask
		// the operator for an internal name the operator cannot know.
		entries = assignIDs(entries, plan)
		type detailJob struct {
			index     int
			id        string
			labels    []entryLabel
			entry     json.RawMessage
			entryPath string
		}
		var jobs []detailJob
		for index, entry := range entries {
			id := fieldOf(entry, plan.idFieldOf())
			if id == "" {
				continue
			}
			entryPath := basePath + "/" + plan.kind + "/" + strconv.Itoa(index)
			labels := append(append([]entryLabel{}, parents...), entryLabel{plan: plan, idValue: id})
			jobs = append(jobs, detailJob{index: index, id: id, labels: labels, entry: entry, entryPath: entryPath})
		}
		if len(jobs) == 0 {
			continue
		}

		// Fork: one goroutine per entry. The buffer is exactly one result per
		// fork, so a worker never blocks on send and a join closes at once.
		results := make(chan detailResult, len(jobs))
		var joins sync.WaitGroup
		for _, job := range jobs {
			joins.Add(1)
			go func(job detailJob) {
				defer joins.Done()
				result := s.detailEntry(ctx, request, client, agent, stage, output, shape, stageContext,
					plan, job.labels, job.entry, job.entryPath, document)
				result.index = job.index
				results <- result
			}(job)
		}
		// Join: the courier closes the channel only after every fork returned.
		go func() {
			joins.Wait()
			close(results)
		}()

		// The receiver collects while the forks still run.
		for result := range results {
			if !result.ok {
				continue
			}
			id := lastValue(result.labels)
			*gaps = append(*gaps, result.gaps...)
			if plan.with != "" && len(parents) == 0 {
				// The companion list lives at the document root: the answer merges
				// beside the entry, the entry itself is set back below.
				if merged, err := replaceOwned(doc, result.echo, plan, id); err == nil {
					doc = merged
				} else {
					log.Printf("stage %s: merge %s of %s %s: %v", stage.ID, plan.with, plan.kind, id, err)
				}
			}
			entries[result.index] = result.updated
			// One finished entry is worth keeping: the next one may take a minute,
			// and a stop, a crash or a provider timeout would lose every answer the
			// model has already given.
			if checkpoint != nil {
				if snapshot, snapErr := setList(doc, plan.kind, entries); snapErr == nil {
					if saveErr := checkpoint(snapshot); saveErr != nil {
						log.Printf("stage %s: save progress of %s: %v", stage.ID, plan.kind, saveErr)
					}
				}
			}
		}
		doc, err = setList(doc, plan.kind, entries)
		if err != nil {
			log.Printf("stage %s: store %s: %v", stage.ID, plan.kind, err)
		}
	}
	return doc
}

// detailCall runs one model call for one entry of one plan. The labels read like
// a path from the payload root: "the layers entry named app of the containers
// entry with containerId c-1".
func (s *StageService) detailCall(
	ctx context.Context, request StageRequest, client LLM, agent domain.AgentConfig,
	stage domain.Stage, output domain.FormType, shape string, stageContext json.RawMessage,
	labels []entryLabel, entry json.RawMessage, plan batchPlan, document json.RawMessage,
) json.RawMessage {
	instruction := detailInstruction(output, stage, plan, labels, entry)
	context := focusContext(output, stageContext, labels, entry)
	document, keys := focusDocument(output, document, labels)
	if len(keys) > 0 {
		instruction += "\nThe names the document already uses (do not reuse one for a new entry; a package key is its container and its path, so the same path in another container is not a duplicate): " + string(keys)
	}
	system, user := detailMessages(output, instruction, context, document)
	complete := LLMRequest{Role: agent.Role, System: system, User: user, Files: registryFileFor(output, plan.kind)}
	call := s.startModelCall(request, agent, complete)
	// The slot is taken around the call only, so a chat-completion running long
	// does not hold the pool while the answer is parsed. A cancelled context
	// frees the slot path too: the call never runs on a dead context.
	if !s.acquireDetailSlot(ctx) {
		s.finishModelCall(call, LLMResponse{}, nil, []string{"stopped: context cancelled while waiting for a call slot"})
		return nil
	}
	response, err := client.Complete(ctx, complete)
	s.releaseDetailSlot()
	if err != nil {
		s.finishModelCall(call, LLMResponse{}, nil, []string{err.Error()})
		log.Printf("stage %s: detail %s: %v", stage.ID, plan.kind, err)
		return nil
	}
	s.finishModelCall(call, response, response.Payload, nil)
	return response.Payload
}

// stageQuestionPolicy names the questions a stage may ask. The questions follow
// the subject, not the schema, and they go deeper at every stage: the same guest,
// card or booking returns at the level the design has reached. The system owns the
// ids, the refs and the layout, so a question about them is always wrong.
func stageQuestionPolicy(output domain.FormType) string {
	switch output {
	case domain.TypeBRD:
		return "Ask about the business in the real world: who the people are, what each business word means, which documents and devices they use, and what may go wrong. Do not ask for a field name, an id or a schema value."
	case domain.TypeFeasibility:
		return "Ask about the money and the time in the real world: the budget the operator has, the deadline, and the option the business prefers. Do not ask for a cost you can estimate, a currency, a confidence or a verdict."
	case domain.TypeSRS:
		return "Ask about the real-world behavior of this requirement: who starts it, what the actor holds or presents, which business rule decides the result, and what goes wrong and what the business does then. Ask about the card, document or device the request names at the level of this requirement. For a non-functional requirement ask only where the business can see the measurement and over which window: never ask for a device or an actor in a quality requirement. Do not ask for a type name, an id or a scenario format."
	case domain.TypeHLD:
		return "Ask about the outside world of this container: which external system or device it talks to, what data it reads and writes, whether it may read that data, and what the business does when the device or the service fails. Do not ask for a protocol you can choose, a type name or an id."
	case domain.TypeADR:
		return "Do not ask the operator why an option was rejected or why the other option is wrong: the architect cites the SRS requirement, the SRS NFR or the constraint the HLD names, and never a BRD rule, because the ADR does not read the BRD. Ask only for a business limit the input forms do not give and that would change the choice, such as a budget, a deadline or a law. Do not ask for an id or a document structure."
	case domain.TypeLLD:
		return "Ask about the business meaning of this contract: which business error the operation returns, what state must exist before the call, and what the client does after the call. Do not ask for inputType, outputType, an HTTP status, an id or a path."
	case domain.TypeTaskPlan, domain.TypeTestContract:
		return "Ask which behavior the business wants first and which result the acceptance must prove. Do not ask for an id, a file path or a task order."
	}
	return ""
}

// detailInstruction asks for one entry and for the layout the gates require
// below it: a container needs a layer, a layer needs a package. Modules stay
// optional. A contract also needs one happy and one negative path.
func detailInstruction(output domain.FormType, stage domain.Stage, plan batchPlan, labels []entryLabel, entry json.RawMessage) string {
	last := labels[len(labels)-1]
	instruction := fmt.Sprintf(
		"You fill the %s form of stage %s. Answer with one JSON object only. Do not add fields. "+
			"Detail exactly one entry: the %s entry with %s %q, now %s. "+
			"The entry now holds: %s. "+
			"Put an object with the fields of this entry under the key \"entry\". A paragraph is not an entry. "+
			"Do not repeat the identifiers or references: the system keeps %s, containerId, formRef, moduleRef and contractRef from the document. "+
			"If the context does not give a concrete value for one field and you are not sure, do not invent it: "+
			"put one openGaps entry at the answer root with the question, the field of this entry it would fill "+
			"(relative to the entry, for example modules[0].path), and the options when it is a choice. "+
			"Do not ask about a type name, an id or a reference: the analyst names those. "+
			"Do not ask about the layout: choose every layer, package path and module path from the HLD container tech, aggregates and relations. "+
			"A layer holds only name and packages. A package holds only path and modules. "+
			"Never put a container or a layer inside a layer or a package.",
		output, stage.ID, plan.kind, plan.idFieldOf(), last.idValue, lastValue(labels), entry, plan.idFieldOf())
	if plan.with != "" {
		instruction += fmt.Sprintf(" Put the %s entries that belong to this entry under the key %q: give only their meaning (kind, given, when, then, expectedErrorCode). Do not repeat %s or their ids: the system sets them to %q.",
			plan.with, plan.with, plan.ref, last.idValue)
	}
	for index := len(labels) - 2; index >= 0; index-- {
		parent := labels[index]
		instruction += fmt.Sprintf(" The %s entry belongs to the %s entry with %s %q.",
			plan.kind, parent.plan.kind, parent.plan.idFieldOf(), parent.idValue)
	}
	for _, child := range plan.children {
		if child.kind == "modules" {
			continue // a package may hold no module
		}
		instruction += fmt.Sprintf(" Fill the %s list of this entry: at least one entry is required.", child.kind)
	}
	if plan.with == "paths" {
		instruction += " Put one happy path and one negative path for this contract; the negative path carries its expectedErrorCode. Do not leave the paths list empty."
	}
	switch plan.kind {
	case "containers":
		instruction += " Fill tech from the HLD context (Go, Spring, React, or the stack the context names); do not leave it empty. Every aggregate entry carries root (the entity name of the aggregate), moduleRef (the module that owns it) and at least one invariant as a Statement {text, modality, priority}; an aggregate without an invariant is not a domain rule. " +
			"An infrastructure container (reverse proxy, API gateway, identity provider, message bus, load balancer, scheduler, external system) holds no aggregate and no invariant: leave its aggregates list empty, keep purpose one sentence on what it does for the other containers, and give it its own invariant-free degradation rule in the HLD."
	case "layers":
		instruction += " Give every package in this layer a path. A Go container has the goca layers domain, usecase, repository, handler, di and middleware; a UI container has the atomic layers token, atom, molecule, organism, template and page. Put each HLD aggregate moduleRef in the domain layer of its container."
	case "packages":
		instruction += " Fill the path of this package and the path of every module it holds; a module path extends the package path. The tech of the container decides the root: a Go container uses internal/<layer>/<module>, a Spring container uses src/main/java/<module>/<layer>, a UI container uses src/<layer>/<module>."
	case "contracts":
		instruction += " Fill moduleRef with the reference of the HLD that owns this contract. Name inputType and outputType from the contract and its fields; do not ask the operator for a type name. Set an HTTP status that matches the condition; do not ask for it."
	case "options":
		instruction += " Fill the summary, the cost {currency, amount}, the scheduleDays and the confidence (low, medium or high). Estimate the cost and the schedule yourself from the scope; do not ask the operator for them."
	case "decisions":
		instruction += " Fill the title, the context, the decision, at least two alternatives with their option and rejectedBecause, and the consequences. The context names the SRS requirement, the SRS NFR or the constraint the HLD names that forces this decision; the rejectedBecause names the same evidence, never a BRD rule. Do not repeat the id."
	case "tests":
		instruction += " Fill the targetTest {kind, path}, the scenario given, when and then (each a list of Text), and expectedErrorCode on a negative path. Do not repeat the test id."
	case "fr":
		instruction += " Fill the statement, the actor, the trigger, the response, the stateChange, the priority and at least one acceptance scenario {given, when, then}. The statement names the actor as the subject and the business object as the object: \"The receptionist checks the guest in\". Name the actor and the business objects with the words of the BRD glossary. Add one errors entry for one invalid input. Do not ask the operator for a scenario: write it from the trigger and the response."
	case "nfr":
		instruction += " Fill the statement, the metric, the unit, the threshold, the measurementPoint, the window and the verifyMethod. Do not ask the operator for a measurement: choose it from the NFR and the context."
	}
	if policy := stageQuestionPolicy(output); policy != "" {
		instruction += " " + policy
	}
	return instruction
}

// lastValue is the full value of the entry id: the root entry uses its id, a
// nested entry prints the path through its parents.
func lastValue(labels []entryLabel) string {
	parts := make([]string, 0, len(labels))
	for _, label := range labels {
		parts = append(parts, label.idValue)
	}
	return strings.Join(parts, " / ")
}

// withEntryKeys fills the identifiers and references the system owns. The model
// answers the meaning of one entry; the key, the companion reference and the
// entry ids of that companion come from the walk, so the model never retypes them
// and the merge finds the entry afterwards.
func withEntryKeys(echo json.RawMessage, plan batchPlan, labels []entryLabel) json.RawMessage {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(echo, &doc); err != nil {
		return echo
	}
	key := labels[len(labels)-1].idValue
	if raw, ok := doc["entry"]; ok {
		doc["entry"] = withField(raw, plan.idFieldOf(), key)
	}
	if plan.with != "" {
		if raw, ok := doc[plan.with]; ok {
			var entries []map[string]any
			if err := json.Unmarshal(raw, &entries); err == nil {
				for i := range entries {
					entries[i][plan.ref] = key
					if id, _ := entries[i]["id"].(string); id == "" {
						entries[i]["id"] = fmt.Sprintf("%s-%s-%d", key, plan.with, i+1)
					}
				}
				if encoded, err := json.Marshal(entries); err == nil {
					doc[plan.with] = encoded
				}
			}
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return echo
	}
	return out
}

// mergeEntryFields applies the fields the model answered to the entry the
// document already holds. A field the answer leaves out keeps its value, so the
// keys, the references and the layout the system filled survive a prose or a
// partial answer. A non-object answer changes nothing: the gates judge the
// document, not a paragraph.
func mergeEntryFields(existing, updated json.RawMessage) json.RawMessage {
	var base, fresh map[string]json.RawMessage
	if err := json.Unmarshal(existing, &base); err != nil {
		return updated
	}
	if err := json.Unmarshal(updated, &fresh); err != nil {
		return existing
	}
	for key, value := range fresh {
		base[key] = value
	}
	out, err := json.Marshal(base)
	if err != nil {
		return existing
	}
	return out
}

// entryFromEcho takes the entry of the walk out of the answer. The answer echoes
// the payload after the detail, so the walk follows the same labels from the
// payload root: the root list, then the lists inside each parent entry.
func entryFromEcho(echo json.RawMessage, labels []entryLabel) (json.RawMessage, bool) {
	if len(echo) == 0 {
		return nil, false
	}
	// A detail call answers the entry alone under the key "entry"; the walk below
	// stays for an answer that echoes the whole payload.
	var answer map[string]json.RawMessage
	if err := json.Unmarshal(echo, &answer); err == nil {
		if entry, ok := answer["entry"]; ok {
			return entry, true
		}
	}
	doc := echo
	for index, label := range labels {
		entries, err := listOf(doc, label.plan.kind)
		if err != nil {
			return nil, false
		}
		entry, ok := entryByID(entries, label.plan.idFieldOf(), label.idValue)
		if !ok {
			return nil, false
		}
		if index == len(labels)-1 {
			return entry, true
		}
		doc = entry
	}
	return nil, false
}

func listOf(payload json.RawMessage, kind string) ([]json.RawMessage, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, err
	}
	raw, ok := document[kind]
	if !ok {
		return nil, fmt.Errorf("no %s list", kind)
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func fieldOf(entry json.RawMessage, field string) string {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(entry, &object); err != nil {
		return ""
	}
	var value string
	_ = json.Unmarshal(object[field], &value)
	return value
}

// mergeEntry replaces one entry of a list with the detailed answer of the same id,
// and the companion entries of that id with the ones the detail call wrote.
func mergeEntry(payload, detail json.RawMessage, plan batchPlan, id string) (json.RawMessage, error) {
	merged, err := replaceEntry(payload, detail, plan.kind, id)
	if err != nil || plan.with == "" {
		return merged, err
	}
	return replaceOwned(merged, detail, plan, id)
}

func replaceEntry(payload, detail json.RawMessage, kind, id string) (json.RawMessage, error) {
	entries, err := listOf(payload, kind)
	if err != nil {
		return nil, err
	}
	details, err := listOf(detail, kind)
	if err != nil {
		return nil, err
	}
	detailed, ok := entryByID(details, "id", id)
	if !ok {
		return payload, nil
	}
	replaced := false
	for index, entry := range entries {
		if fieldOf(entry, "id") == id {
			entries[index] = detailed
			replaced = true
			break
		}
	}
	if !replaced {
		return payload, nil
	}
	return setList(payload, kind, entries)
}

// replaceOwned keeps the companion entries of the other owners and takes the ones
// the detail call wrote for this id. A detail without them leaves the owner without
// them; the gates name that gap.
func replaceOwned(payload, detail json.RawMessage, plan batchPlan, id string) (json.RawMessage, error) {
	entries, err := listOf(payload, plan.with)
	if err != nil {
		return nil, err
	}
	fresh, _ := listOf(detail, plan.with)
	var kept []json.RawMessage
	for _, entry := range entries {
		if fieldOf(entry, plan.ref) != id {
			kept = append(kept, entry)
		}
	}
	for _, entry := range fresh {
		if fieldOf(entry, plan.ref) == id {
			kept = append(kept, entry)
		}
	}
	return setList(payload, plan.with, kept)
}

// detailGaps returns the questions one detail call asked, each with the absolute
// pointer of the entry it belongs to. The model names a field of its own entry;
// only the walk knows where that entry sits in the payload.
func detailGaps(echo json.RawMessage, entryPath string) []json.RawMessage {
	var answer map[string]json.RawMessage
	if err := json.Unmarshal(echo, &answer); err != nil {
		return nil
	}
	var gaps []map[string]any
	if err := json.Unmarshal(answer["openGaps"], &gaps); err != nil || len(gaps) == 0 {
		return nil
	}
	kept := make([]json.RawMessage, 0, len(gaps))
	for _, gap := range gaps {
		field, _ := gap["field"].(string)
		if field == "" {
			field, _ = gap["pointer"].(string)
		}
		pointer := gapPointer(entryPath, field)
		if pointer == "" {
			log.Printf("detail gap of %s without a field, dropped", entryPath)
			continue
		}
		gap["pointer"] = pointer
		delete(gap, "field")
		encoded, err := json.Marshal(gap)
		if err != nil {
			continue
		}
		kept = append(kept, encoded)
	}
	return kept
}

// mergeDetailGaps puts the questions of every detail call beside the skeleton
// gaps: one openGaps list at the document root is what the gates and the
// operator read.
func mergeDetailGaps(doc json.RawMessage, gaps []json.RawMessage) json.RawMessage {
	if len(gaps) == 0 {
		return doc
	}
	existing, _ := listOf(doc, "openGaps")
	merged, err := setList(doc, "openGaps", append(existing, gaps...))
	if err != nil {
		return doc
	}
	return merged
}

// gapPointer turns the field a detail answer names into an absolute JSON Pointer.
// A field relative to the entry gets the entry path; a path the model wrote from
// the payload root keeps it. An empty field names no slot and is dropped.
func gapPointer(entryPath, field string) string {
	field = strings.TrimSpace(field)
	for _, prefix := range []string{"entry.", "/entry/", "$.", "payload.", "/payload/"} {
		field = strings.TrimPrefix(field, prefix)
	}
	pointer := normalizePointer(field)
	if !strings.HasPrefix(pointer, "/") {
		return ""
	}
	if entryPath == "" || isPayloadRoot(pointer) {
		return pointer
	}
	return entryPath + pointer
}

// isPayloadRoot reports whether the pointer already starts at a root list of the
// payload, so the entry path must not be glued in front of it.
func isPayloadRoot(pointer string) bool {
	root, _, _ := strings.Cut(strings.TrimPrefix(pointer, "/"), "/")
	switch root {
	case "containers", "contracts", "paths", "eventMaps", "adrRefs", "openGaps":
		return true
	}
	return false
}

func setList(payload json.RawMessage, kind string, entries []json.RawMessage) (json.RawMessage, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(entries)
	if err != nil {
		return nil, err
	}
	document[kind] = encoded
	return json.Marshal(document)
}

// withField sets one string field of an entry and keeps every other field as it is.
func withField(entry json.RawMessage, field, value string) json.RawMessage {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(entry, &object); err != nil {
		return entry
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return entry
	}
	object[field] = encoded
	out, err := json.Marshal(object)
	if err != nil {
		return entry
	}
	return out
}

// assignIDs fills the entry identifiers the model left out of the outline. An id
// is structural, not an operator value: the detail call addresses its entry by the
// id, so a missing one leaves the entry unfilled and the gates ask the operator
// for an internal name. Lists keyed by a business name or path keep the model's
// own keys and are left untouched.
func assignIDs(entries []json.RawMessage, plan batchPlan) []json.RawMessage {
	if plan.idFieldOf() != "id" {
		return entries
	}
	taken := map[string]bool{}
	for _, entry := range entries {
		taken[fieldOf(entry, "id")] = true
	}
	for index := range entries {
		if fieldOf(entries[index], "id") != "" {
			continue
		}
		base := strings.ToUpper(plan.kind) + "-" + strconv.Itoa(index+1)
		id := base
		for suffix := 2; taken[id]; suffix++ {
			id = base + "-" + strconv.Itoa(suffix)
		}
		taken[id] = true
		entries[index] = withField(entries[index], "id", id)
	}
	return entries
}

func entryByID(entries []json.RawMessage, idField, id string) (json.RawMessage, bool) {
	for _, entry := range entries {
		if fieldOf(entry, idField) == id {
			return entry, true
		}
	}
	return nil, false
}

// registerObjects proposes the objects a stored document names: an HLD names its
// containers, an LLD names the layers, packages and modules of every container.
// A bookkeeping failure never fails the stage.
func (s *StageService) registerObjects(ctx context.Context, request StageRequest, form *domain.Form) {
	if s.registry == nil {
		return
	}
	targetSystemID, err := strconv.ParseInt(form.TargetSystemID, 10, 64)
	if err != nil {
		log.Printf("stage %s: registry: bad target system id %q: %v", stageFormID(request.PipelineID, form.Stage, "", false), form.TargetSystemID, err)
		return
	}
	by := domain.Ref{FormID: form.FormID, FormType: form.FormType, FormVersion: form.FormVersion}
	known, proposalErr := s.proposalsOf(form)
	if proposalErr != nil {
		log.Printf("stage %s pipeline %s: registry: %v", form.Stage, request.PipelineID, proposalErr)
		return
	}
	if len(known) == 0 {
		return
	}
	existing, err := s.registry.List(ctx, targetSystemID)
	if err != nil {
		log.Printf("stage %s pipeline %s: registry list: %v", form.Stage, request.PipelineID, err)
		return
	}
	seen := map[string]bool{}
	for _, object := range existing {
		seen[registryKey(object.Kind, object.ContainerID, object.Name)] = true
	}
	for _, proposal := range known {
		key := registryKey(proposal.Kind, proposal.ContainerID, proposal.Name)
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, err := s.registry.Propose(ctx, targetSystemID, proposal.Kind, proposal.Name, proposal.Path, proposal.ContainerID, by, proposal.Fields); err != nil {
			// A duplicate id race stays in the log: the registry entry comes back
			// with the next version of the document.
			log.Printf("stage %s pipeline %s: propose %s %s: %v", form.Stage, request.PipelineID, proposal.Kind, proposal.Name, err)
		}
	}
}

// registryKey groups the objects of one container under one key, so a layer
// named the same does not collapse across containers.
func registryKey(kind domain.Kind, containerID, name string) string {
	return string(kind) + "|" + containerID + "|" + name
}

// proposalsOf extracts the registry objects a document names. The model-chosen
// objectId stays the name of the object: the documents answer with it, and the
// map of the target system matches it against the registry entries.
func (s *StageService) proposalsOf(form *domain.Form) ([]domain.Object, error) {
	switch form.FormType {
	case domain.TypeHLD:
		var payload domain.HLD
		if err := json.Unmarshal(form.Payload, &payload); err != nil {
			return nil, fmt.Errorf("read hld: %w", err)
		}
		out := make([]domain.Object, 0, len(payload.Containers))
		for _, container := range payload.Containers {
			if container.ObjectID == "" {
				continue
			}
			fields := map[string]any{"tech": container.Tech, "boundedContextId": container.BoundedContextID}
			out = append(out, domain.Object{Kind: domain.KindContainer, Name: container.ObjectID, Fields: fields})
		}
		return out, nil
	case domain.TypeLLD:
		var payload domain.LLD
		if err := json.Unmarshal(form.Payload, &payload); err != nil {
			return nil, fmt.Errorf("read lld: %w", err)
		}
		out := make([]domain.Object, 0)
		for _, container := range payload.Containers {
			for _, layer := range container.Layers {
				out = append(out, domain.Object{
					Kind: domain.KindLayer, Name: string(layer.Name), ContainerID: container.ContainerID,
				})
				for _, pkg := range layer.Packages {
					out = append(out, domain.Object{
						Kind: domain.KindPackage, Name: pkg.Path, Path: pkg.Path, ContainerID: container.ContainerID,
					})
					for _, module := range pkg.Modules {
						if module.ObjectID == "" && module.Path == "" {
							continue
						}
						out = append(out, domain.Object{
							Kind: domain.KindModule, Name: module.ObjectID, Path: module.Path, ContainerID: container.ContainerID,
						})
					}
				}
			}
		}
		return out, nil
	default:
		return nil, nil
	}
}

// markValid accepts a stored form that raised no question. A failure keeps the
// draft: the operator still sees the document, and a re-run tries again.
func (s *StageService) markValid(ctx context.Context, form *domain.Form) {
	if err := s.forms.SetStatus(ctx, form.FormID, form.FormVersion, domain.StatusValid); err != nil {
		log.Printf("form %s v%d: set valid: %v", form.FormID, form.FormVersion, err)
		return
	}
	form.Status = domain.StatusValid
}

// refOfType returns the reference of the input document of a form type.
func refOfType(refs []domain.Ref, formType domain.FormType) (domain.Ref, bool) {
	for _, ref := range refs {
		if ref.FormType == formType {
			return ref, true
		}
	}
	return domain.Ref{}, false
}

// recordModelCall stores one attempt as its whole dialogue: tokens and time,
// which stage and role, the full prompt the agent sent, the raw answer, and the
// gate verdicts that rejected the draft. The book must never fail a stage: a
// lost row costs the operator a number, a failed save would cost the form.
// startModelCall stores a placeholder row before the model answers, so the
// console shows the prompt and the running state at once. It returns the row id,
// or 0 when there is no recorder or the save failed.
func (s *StageService) startModelCall(request StageRequest, agent domain.AgentConfig, req LLMRequest) *domain.ModelCall {
	if s.modelCalls == nil {
		return nil
	}
	pipelineID, _ := strconv.Atoi(request.PipelineID)
	call := &domain.ModelCall{
		// shortcut: run_id carries the pipeline id: the run table stores no stage
		// runs yet. Point it at the run row when the recorder is wired.
		RunID:   pipelineID,
		Profile: string(agent.Role),
		Model:   agent.Model,
		Stage:   request.StageID,
		// The prompt is stored the way the model read it: system rules first,
		// then the user message with the context, the previous verdicts and the
		// attached registry files.
		Request:  req.System + "\n\n=== USER MESSAGE ===\n\n" + req.UserWithFiles(),
		Response: "",
	}
	if err := s.modelCalls.Save(call); err != nil {
		log.Printf("stage %s pipeline %s: save model call: %v", request.StageID, request.PipelineID, err)
		return nil
	}
	return call
}

// finishModelCall fills in the result of an attempt that startModelCall recorded.
// A zero id means there was no placeholder, so nothing is updated.
func (s *StageService) finishModelCall(call *domain.ModelCall, response LLMResponse, payload json.RawMessage, problems []string) {
	if call == nil {
		return
	}
	usage := response.Usage
	call.TokensIn = usage.TokensIn
	call.TokensOut = usage.TokensOut
	call.LatencyMs = usage.LatencyMs
	call.Response = string(payload)
	if len(call.Response) == 0 && len(response.Raw) > 0 {
		// The model answered, but the answer did not survive as JSON: keep its
		// text, so the operator sees what went wrong instead of an empty row.
		call.Response = response.Raw
	}
	call.Errors = strings.Join(problems, "\n")
	if err := s.modelCalls.Update(call); err != nil {
		log.Printf("model call %d: update: %v", call.ID, err)
	} else {
		log.Printf("model call %d: updated, response=%d chars", call.ID, len(call.Response))
	}
}

// stageFormID is stable per pipeline, stage and output type, so a re-run creates
// a new version. A stage with one output keeps the short id.
func stageFormID(pipelineID, stageID string, output domain.FormType, multi bool) string {
	if multi {
		return "FRM-" + pipelineID + "-" + stageID + "-" + string(output)
	}
	return "FRM-" + pipelineID + "-" + stageID
}
