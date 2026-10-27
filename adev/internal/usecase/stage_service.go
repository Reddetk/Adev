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
}

// NewStageService builds the stage runner.
func NewStageService(forms *FormService, settings *SettingsService, llm LLMFactory, runs StageRunRecorder) *StageService {
	return &StageService{forms: forms, settings: settings, llm: llm, runs: runs}
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
		if finished != nil {
			outcome.FormID, outcome.FormVersion = finished.FormID, finished.FormVersion
			if finished.Status == domain.StatusWaitingOperator {
				// The questions from the earlier attempt still wait: the run pauses
				// again instead of calling the model.
				outcome.Blocked = true
				outcome.Gaps = []string{"the draft waits for operator answers"}
				s.record(ctx, request, stage, "blocked", 0, outcome.Gaps[0])
				continue
			}
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

// runForm fills one output form of a stage with one model call. Any answer is
// accepted: a gate mismatch rides on the stored draft as an indicator, and the
// rerun and the answers are the operator's decisions, so no auto-fix loop burns
// model calls.
// shortcut: the mismatch of one run does not reach the prompt of the next run;
// feed the last run row back when the operator's reruns visibly regress.
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
	outcome := StageOutcome{Stage: stage.ID, Attempts: 1}
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
	activity.Attempt()

	// The BRD is filled in two phases: the first call collects the operator
	// questions, and every later run assembles the document from the answers
	// that already stand in the stored draft. The draft belongs to the same
	// document: a fresh output of a different type starts without it.
	resume := json.RawMessage(nil)
	if previous != nil && previous.FormType == output {
		if prior, err := s.forms.Get(ctx, previous.FormID, previous.FormVersion); err == nil && len(prior.Payload) > 0 {
			resume = prior.Payload
		}
	}

	system, user := stagePrompt(stage, output, shape, stageContext, nil)
	if output == domain.TypeBRD && previous == nil {
		system, user = promptMessages(questionsInstruction(stage, output), shape, stageContext, nil)
	}
	if len(resume) > 0 {
		user += fmt.Sprintf("\nThe previousDraft holds the draft of this document and the operator answers: keep the answered values and fill only what the answers or the context cover.\n%s\n", resume)
	}
	// Record the attempt before the model answers: the dashboard shows the prompt
	// and the "running" state at once, instead of waiting for the whole call.
	complete := LLMRequest{Role: agent.Role, System: system, User: user, Files: registryFiles(output)}
	call := s.startModelCall(request, agent, complete)
	response, err := client.Complete(ctx, complete)
	if err != nil {
		if ctx.Err() != nil {
			activity.Finish("stopped", "stopped by the operator")
			s.record(ctx, request, stage, "failed", 1, ErrStageStopped.Error())
			return StageOutcome{}, ErrStageStopped
		}
		activity.Finish("failed", err.Error())
		s.record(ctx, request, stage, "failed", 1, err.Error())
		// A call that never answered is on the book, too: the operator sees
		// the model request that produced the failure.
		s.finishModelCall(call, LLMResponse{}, nil, []string{err.Error()})
		return StageOutcome{}, err
	}

	form := s.buildForm(request, stage, refs, output, formID, version, previous)
	payload := normalizeGapPointers(response.Payload)
	// A composite document goes out in batches: the skeleton first, then one
	// call per entry, so no single answer has to carry the whole list.
	if plans, ok := batchPlans[output]; ok {
		payload = s.detailBatches(ctx, request, client, agent, stage, output, shape, stageContext, plans, payload)
		// The detail calls ask their own questions: their pointers read like the
		// payload root and take the same normalization as the skeleton.
		payload = normalizeGapPointers(payload)
	}
	// The model answers the meaning and loses the wrapper: a question arrives
	// as a plain string instead of the Text record. Repair the wrapping only;
	// the gates still judge the values.
	payload = coercePayload(output, payload)
	// Unaskable gaps must leave the payload before the gates see it.
	payload = purgeUnaskableGaps(output, payload)
	if err := form.SetRawPayload(payload); err != nil {
		activity.Finish("failed", err.Error())
		s.record(ctx, request, stage, "failed", 1, err.Error())
		s.finishModelCall(call, response, payload, []string{err.Error()})
		return StageOutcome{}, err
	}

	var mismatch []string
	if err := s.forms.Submit(ctx, form); err != nil {
		if !isFormRejection(err) {
			activity.Finish("failed", err.Error())
			s.record(ctx, request, stage, "failed", 1, err.Error())
			s.finishModelCall(call, response, payload, []string{err.Error()})
			return StageOutcome{}, err
		}
		if draftErr := s.forms.SubmitDraft(ctx, form); draftErr != nil {
			activity.Finish("failed", draftErr.Error())
			s.record(ctx, request, stage, "failed", 1, draftErr.Error())
			s.finishModelCall(call, response, payload, []string{draftErr.Error()})
			return StageOutcome{}, draftErr
		}
		mismatch = []string{err.Error()}
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
	if s.questions != nil && request.PipelineID != "" {
		pipelineID, _ := strconv.Atoi(request.PipelineID)
		_, gapBlocked, err := s.questions.PublishGaps(ctx, form, pipelineID)
		if err != nil {
			log.Printf("stage %s pipeline %s: publish gap: %v", stage.ID, request.PipelineID, err)
		}
		if gapBlocked {
			blocked = true
			reasons = append(reasons, "the draft holds open questions for the operator")
		} else if blocked {
			// No gap went out: the gate mismatch itself asks the operator, the
			// same repair path the answer flow uses.
			asked, gateErr := s.questions.PublishGate(ctx, form, mismatch, pipelineID)
			if gateErr != nil {
				log.Printf("stage %s pipeline %s: publish gate: %v", stage.ID, request.PipelineID, gateErr)
			}
			if asked {
				reasons = append(reasons, "the draft holds open questions for the operator")
			}
		}
	}
	if blocked {
		outcome.Blocked = true
		outcome.Gaps = reasons
		s.record(ctx, request, stage, "blocked", 1, strings.Join(reasons, "; "))
		activity.Finish("blocked", strings.Join(reasons, "; "))
		return outcome, nil
	}
	// Nothing waits for the operator and the gates passed: the document is accepted.
	s.markValid(ctx, form)
	s.record(ctx, request, stage, "done", 1, "")
	activity.Finish("done", "")
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
// call returns the outline; every later call details one entry of one level.
var batchPlans = map[domain.FormType][]batchPlan{
	domain.TypeSRS:      {{kind: "fr"}, {kind: "nfr"}},
	domain.TypeTaskPlan: {{kind: "tasks"}},
	// The HLD is detailed by one call per container, which writes the aggregates
	// with their invariants of that container.
	domain.TypeHLD: {{kind: "containers", idField: "objectId", with: "aggregates", ref: "containerId"}},
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

// detailBatches asks the model to fill one entry per call and merges the answers
// back into the skeleton. A failed detail call keeps the skeleton entry: the gates
// still judge the document, and the operator sees what is missing.
func (s *StageService) detailBatches(ctx context.Context, request StageRequest, client LLM, agent domain.AgentConfig, stage domain.Stage, output domain.FormType, shape string, stageContext json.RawMessage, plans []batchPlan, payload json.RawMessage) json.RawMessage {
	return s.detailLevel(ctx, request, client, agent, stage, output, shape, stageContext, plans, payload, nil)
}

// detailLevel fills one level of the document: every entry of every plan names a
// model call, the answer is merged, and the children of the entry go through the
// same walk on the updated entry.
func (s *StageService) detailLevel(
	ctx context.Context, request StageRequest, client LLM, agent domain.AgentConfig,
	stage domain.Stage, output domain.FormType, shape string, stageContext json.RawMessage,
	plans []batchPlan, doc json.RawMessage, parents []entryLabel,
) json.RawMessage {
	for _, plan := range plans {
		entries, err := listOf(doc, plan.kind)
		if err != nil {
			continue
		}
		for index, entry := range entries {
			id := fieldOf(entry, plan.idFieldOf())
			if id == "" {
				continue
			}
			labels := append(append([]entryLabel{}, parents...), entryLabel{plan: plan, idValue: id})
			echo := s.detailCall(ctx, request, client, agent, stage, output, shape, stageContext, labels, plan)
			updated, ok := entryFromEcho(echo, labels)
			if !ok {
				log.Printf("stage %s: detail %s %s: the answer does not carry the entry", stage.ID, plan.kind, id)
				continue
			}
			doc = detailGaps(doc, echo)
			if plan.with != "" && len(parents) == 0 {
				// The companion list lives at the document root: the answer merges
				// beside the entry, the entry itself is set back below.
				if merged, err := replaceOwned(doc, echo, plan, id); err == nil {
					doc = merged
				} else {
					log.Printf("stage %s: merge %s of %s %s: %v", stage.ID, plan.with, plan.kind, id, err)
				}
			}
			if len(plan.children) > 0 {
				updated = s.detailLevel(ctx, request, client, agent, stage, output, shape, stageContext, plan.children, updated, labels)
			}
			entries[index] = updated
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
	labels []entryLabel, plan batchPlan,
) json.RawMessage {
	last := labels[len(labels)-1]
	instruction := fmt.Sprintf(
		"You fill the %s form of stage %s. Answer with one JSON object only. Do not add fields. "+
			"Detail exactly one entry: the %s entry with %s %q, now %s. "+
			"Put the filled entry under the key \"entry\". "+
			"If the context does not give a concrete value or you are not sure, do not invent it: "+
			"put one openGaps entry at the answer root with the question, the pointer of the field it would fill, "+
			"and the options when it is a choice.",
		output, stage.ID, plan.kind, plan.idFieldOf(), last.idValue, lastValue(labels))
	if plan.with != "" {
		instruction += fmt.Sprintf(" Put the %s entries whose %s is %q under the key %q.",
			plan.with, plan.ref, last.idValue, plan.with)
	}
	for index := len(labels) - 2; index >= 0; index-- {
		parent := labels[index]
		instruction += fmt.Sprintf(" The %s entry belongs to the %s entry with %s %q.",
			plan.kind, parent.plan.kind, parent.plan.idFieldOf(), parent.idValue)
	}
	context := focusContext(output, stageContext, labels)
	system, user := promptMessages(instruction, shape, context, nil)
	complete := LLMRequest{Role: agent.Role, System: system, User: user, Files: registryFileFor(output, plan.kind)}
	call := s.startModelCall(request, agent, complete)
	response, err := client.Complete(ctx, complete)
	if err != nil {
		s.finishModelCall(call, LLMResponse{}, nil, []string{err.Error()})
		log.Printf("stage %s: detail %s %s: %v", stage.ID, plan.kind, last.idValue, err)
		return nil
	}
	s.finishModelCall(call, response, response.Payload, nil)
	return response.Payload
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

// detailGaps keeps the questions one detail call asked: the pointer names a
// field of the document root, so they merge beside the entries.
func detailGaps(doc, echo json.RawMessage) json.RawMessage {
	var answer map[string]json.RawMessage
	if err := json.Unmarshal(echo, &answer); err != nil {
		return doc
	}
	var gaps []json.RawMessage
	if err := json.Unmarshal(answer["openGaps"], &gaps); err != nil || len(gaps) == 0 {
		return doc
	}
	existing, _ := listOf(doc, "openGaps")
	merged, err := setList(doc, "openGaps", append(existing, gaps...))
	if err != nil {
		return doc
	}
	return merged
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
