package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/Reddetk/Adev/internal/domain"
)

// QuestionService turns a blocked stage into an operator question and applies the
// answer as a correction of the draft, so the stage never re-asks the same field.
type QuestionService struct {
	forms    *FormService
	requests OperatorRequestRepository
}

// NewQuestionService builds the question service.
func NewQuestionService(forms *FormService, requests OperatorRequestRepository) *QuestionService {
	return &QuestionService{forms: forms, requests: requests}
}

// Publish stores the draft the model produced, marks it waiting for the operator,
// and writes one operator request for the first gate failure.
func (s *QuestionService) Publish(ctx context.Context, form *domain.Form, failures []domain.ValidationError, pipelineID int) error {
	if form == nil {
		return fmt.Errorf("publish question: form is nil")
	}
	if err := s.forms.SubmitDraft(ctx, form); err != nil {
		// The draft may hold non-JSON garbage; the question must still reach the operator.
		log.Printf("question pipeline %d: keep draft of %s: %v", pipelineID, form.FormType, err)
	}
	return s.ask(ctx, form, failures, pipelineID)
}

// PublishGaps turns the open gaps of a stored draft into operator questions.
// One run asks every unanswered gap question at once (the operator answers the
// cards one by one); already-active requests are never asked again. The stage
// stays blocked while any of its questions waits for an answer, so the value
// valid arrives only after the last one.
// shortcut: the registry gate 10 says one question per run; here a run is one
// batch of questions, reconcile with the registry when the ask-loop moves to
// model-driven conversation.
func (s *QuestionService) PublishGaps(ctx context.Context, form *domain.Form, pipelineID int) (published, blocked bool, err error) {
	if s.requests == nil || form == nil {
		return false, false, nil
	}
	for _, position := range form.Gaps() {
		gap := position.Gap
		if gap.Question.Value == "" {
			continue
		}
		if gap.Pointer == "" || !strings.HasPrefix(gap.Pointer, "/") {
			// Without a valid pointer the answer has no place to land in the payload.
			log.Printf("question pipeline %d: gap %s of %s has no pointer, not asked", pipelineID, gap.ID, form.FormID)
			continue
		}
		if alreadyAsked(ctx, s.requests, pipelineID, gap.Pointer) {
			continue
		}
		question := fmt.Sprintf("Уточнить у оператора (%s, %s): %s", form.FormType, gap.ID, gap.Question.Value)
		if err := s.requests.Save(&domain.OperatorRequest{
			PipelineID: pipelineID,
			Pointer:    gap.Pointer,
			Question:   question,
			Options:    gap.Options,
			Blocking:   gap.Blocking,
			Status:     "active",
		}); err != nil {
			return published, false, err
		}
		published = true
	}
	if published && s.forms != nil {
		if err := s.forms.SetStatus(ctx, form.FormID, form.FormVersion, domain.StatusWaitingOperator); err != nil {
			log.Printf("question pipeline %d: set waiting_operator: %v", pipelineID, err)
		}
	}
	return published, published, nil
}

// alreadyAsked reports an active request for the same pointer and pipeline, so a
// re-run of the stage never asks the operator the same question twice.
func alreadyAsked(ctx context.Context, requests OperatorRequestRepository, pipelineID int, pointer string) bool {
	request, err := requests.FindByPointer(pointer)
	return err == nil && request.PipelineID == pipelineID && request.Status == "active"
}

// pointerOf turns the field path of a gate failure into a JSON Pointer, so the
// answer can patch the draft: "metrics[0].unit" becomes "/metrics/0/unit".
func pointerOf(field string) string {
	withIndices := regexp.MustCompile(`\[([^\]]+)\]`).ReplaceAllString(field, "/$1")
	if !strings.HasPrefix(withIndices, "/") {
		withIndices = "/" + withIndices
	}
	return strings.ReplaceAll(withIndices, ".", "/")
}

// patchOp is one RFC 6902 operation of the answer patch.
type patchOp struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value"`
}

// Answer applies the operator value to the stored draft as the next version.
// When the corrected payload still fails a gate, the next failure becomes a new
// question, so the operator closes the gaps one by one.
func (s *QuestionService) Answer(ctx context.Context, form *domain.Form, pointer string, value json.RawMessage, pipelineID int) (*domain.Form, error) {
	if form == nil {
		return nil, fmt.Errorf("answer: form is nil")
	}
	if len(value) == 0 {
		return nil, fmt.Errorf("answer: value is empty")
	}
	// The operator may type anything: a value that is not JSON becomes a JSON
	// string, because an answer is a decision and never a rejected request.
	if !json.Valid(value) {
		quoted, err := json.Marshal(string(value))
		if err != nil {
			return nil, fmt.Errorf("answer: encode value: %w", err)
		}
		value = quoted
	}

	// An answered gap leaves the payload: the decision went into its field.
	// Every question-phase form carries its gaps in the same openGaps section.
	var removals []int
	for _, position := range form.Gaps() {
		if position.Gap.Pointer == pointer {
			removals = append(removals, position.Index)
		}
	}
	// The operator answers in plain words; the field keeps the type it already has.
	value = coerceAnswer(form.Payload, pointer, value)
	// A shape failure points at the whole payload; no path inside itself can
	// name it, so the answer replaces the root directly instead of patching.
	var payload json.RawMessage
	if pointer == "/payload" {
		payload = value
	} else {
		// A gap may name the slot right past the end of an array; RFC 6902
		// appends there with add, while replace only rewrites an existing value.
		op := "replace"
		var doc any
		if err := json.Unmarshal(form.Payload, &doc); err == nil {
			if valueAt(doc, pointer) == nil {
				op = "add"
			}
		}
		ops := []patchOp{{Op: op, Path: pointer, Value: value}}
		// Remove from the end, so the earlier indexes stay valid.
		for i := len(removals) - 1; i >= 0; i-- {
			ops = append(ops, patchOp{Op: "remove", Path: "/openGaps/" + strconv.Itoa(removals[i])})
		}
		encoded, err := json.Marshal(ops)
		if err != nil {
			return nil, fmt.Errorf("answer: encode patch: %w", err)
		}
		payload, err = ApplyDelta(form.Payload, encoded)
		if err != nil {
			return nil, fmt.Errorf("answer: apply value to %s: %w", pointer, err)
		}
	}
	derived := *form
	derived.FormVersion = form.FormVersion + 1
	derived.PreviousFormVersion = &domain.Ref{
		FormID: form.FormID, FormType: form.FormType, FormVersion: form.FormVersion, Hash: form.PayloadHash,
	}
	derived.Mode = domain.ModeFull
	derived.Base = nil
	derived.Ops = nil
	derived.Status = domain.StatusDraft
	if err := derived.SetPayload(payload); err != nil {
		return nil, err
	}
	if err := s.forms.SubmitDraft(ctx, &derived); err != nil {
		return nil, err
	}

	// The answered value is decided: the request closes even when other gaps remain.
	s.closeRequests(ctx, pipelineID, pointer)
	// The gap gate wakes up only on a valid status: check the draft as a would-be valid form.
	derived.Status = domain.StatusValid
	if err := derived.ValidatePayloadWithStatus(); err != nil {
		// A remaining gap is not a field to patch: its own question goes out
		// first, and only when none went out the gate failures take the turn.
		published, _, gapErr := s.PublishGaps(ctx, &derived, pipelineID)
		if gapErr != nil {
			log.Printf("question pipeline %d: publish gaps: %v", pipelineID, gapErr)
		}
		if published {
			// The run sees the stored status: the version still waits on questions.
			derived.Status = domain.StatusWaitingOperator
			return &derived, nil
		}
		remaining := gapGate(collectFailures(err))
		if len(remaining) == 0 {
			// Only the gap gate failed and its question is already in the
			// dashboard: keep the version waiting, so the pending answer finds it.
			if err := s.forms.SetStatus(ctx, derived.FormID, derived.FormVersion, domain.StatusWaitingOperator); err != nil {
				log.Printf("question pipeline %d: set waiting_operator: %v", pipelineID, err)
			}
			derived.Status = domain.StatusWaitingOperator
			return &derived, nil
		}
		// The returned form mirrors the stored status: a valid status would make
		// the UI re-run the stage while its question is still open.
		derived.Status = domain.StatusWaitingOperator
		return &derived, s.ask(ctx, &derived, remaining, pipelineID)
	}
	if err := s.forms.SetStatus(ctx, derived.FormID, derived.FormVersion, domain.StatusValid); err != nil {
		return nil, err
	}
	return &derived, nil
}

// ask marks the form waiting and writes one operator request for the first failure.
func (s *QuestionService) ask(ctx context.Context, form *domain.Form, failures []domain.ValidationError, pipelineID int) error {
	failures = answerable(failures)
	// The version waits even when no new question goes out: the questions already
	// in the dashboard must find it, or the next answer dies with "no form waiting".
	if err := s.forms.SetStatus(ctx, form.FormID, form.FormVersion, domain.StatusWaitingOperator); err != nil {
		log.Printf("question pipeline %d: set waiting_operator: %v", pipelineID, err)
	}
	if len(failures) == 0 || s.requests == nil {
		return nil
	}
	first := failures[0]
	// The dashboard already holds this question: do not ask twice.
	if all, err := s.requests.FindAll(); err == nil {
		for i := range all {
			if all[i].PipelineID == pipelineID && all[i].Pointer == pointerOf(first.Field) && all[i].Status == "active" {
				return nil
			}
		}
	}
	return s.requests.Save(&domain.OperatorRequest{
		PipelineID: pipelineID,
		Pointer:    pointerOf(first.Field),
		Question:   fmt.Sprintf("Уточнить у оператора %s: %s.", first.Field, first.Problem),
		Blocking:   true,
		Status:     "active",
	})
}

// answerable drops the failure that names the whole document: a payload which does
// not decode into the shape is the model's mistake, and no operator answer can
// supply a whole document. Asking anyway would block the stage on a question the
// operator cannot answer.
func answerable(failures []domain.ValidationError) []domain.ValidationError {
	out := make([]domain.ValidationError, 0, len(failures))
	for _, failure := range failures {
		if failure.Field == "payload" {
			continue
		}
		out = append(out, failure)
	}
	return out
}

// collectFailures walks a joined validation error and returns every gate failure.
func collectFailures(err error) []domain.ValidationError {
	var failures []domain.ValidationError
	var walk func(error)
	walk = func(current error) {
		if current == nil {
			return
		}
		if joined, ok := current.(interface{ Unwrap() []error }); ok {
			for _, item := range joined.Unwrap() {
				walk(item)
			}
			return
		}
		var failure *domain.ValidationError
		if errors.As(current, &failure) {
			failures = append(failures, *failure)
		}
	}
	walk(err)
	return failures
}

// coerceAnswer shapes a plain operator answer into the type the draft field
// already holds: a text field takes a Text value, a number field takes a
// number. A value the operator already wrote as JSON is used as is.
func coerceAnswer(payload json.RawMessage, pointer string, value json.RawMessage) json.RawMessage {
	var answer any
	if err := json.Unmarshal(value, &answer); err != nil {
		return value
	}
	if _, isObject := answer.(map[string]any); isObject {
		return value
	}
	var doc any
	if err := json.Unmarshal(payload, &doc); err != nil {
		return value
	}
	switch holder := valueAt(doc, pointer).(type) {
	case map[string]any:
		// A text field: the plain words become the Text value of the shape.
		if _, hasText := holder["text"]; hasText {
			if text, isString := answer.(string); isString {
				if encoded, err := json.Marshal(domain.Text{Value: text, Lang: domain.LangSTE100}); err == nil {
					return encoded
				}
			}
		}
	case string:
		// A string field keeps strings even when the operator typed a bare
		// number: the shape says threshold is a spelled-out value.
		if _, isNumber := answer.(float64); isNumber {
			if encoded, err := json.Marshal(fmt.Sprintf("%v", answer)); err == nil {
				return encoded
			}
		}
	case float64:
		// A number field: a spelled-out number still lands as a number.
		if text, isString := answer.(string); isString {
			if number, parseErr := strconv.ParseFloat(strings.TrimSpace(text), 64); parseErr == nil {
				if encoded, err := json.Marshal(number); err == nil {
					return encoded
				}
			}
		}
	}
	return value
}

// valueAt walks a JSON document along a pointer and returns the value it names.
func valueAt(doc any, pointer string) any {
	current := doc
	for _, raw := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		segment := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		switch node := current.(type) {
		case map[string]any:
			current = node[segment]
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index >= len(node) {
				return nil
			}
			current = node[index]
		default:
			return nil
		}
		if current == nil {
			return nil
		}
	}
	return current
}

// Models often write gap pointers in dot syntax (scope.in[0]); questions and
// answers work only with JSON Pointers. Normalizing at the submit border keeps
// the stored form, the published requests and the answer patches in one format.
func normalizeGapPointers(payload json.RawMessage) json.RawMessage {
	var doc struct {
		OpenGaps []map[string]any `json:"openGaps"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil || len(doc.OpenGaps) == 0 {
		return payload
	}
	changed := false
	for _, gap := range doc.OpenGaps {
		p, _ := gap["pointer"].(string)
		if p == "" || strings.HasPrefix(p, "/") {
			continue
		}
		gap["pointer"] = toJSONPointer(p)
		changed = true
	}
	if !changed {
		return payload
	}
	// Re-encode only the gaps section, so the rest of the payload bytes stay as sent.
	var sections map[string]json.RawMessage
	if err := json.Unmarshal(payload, &sections); err != nil {
		return payload
	}
	gaps, err := json.Marshal(doc.OpenGaps)
	if err != nil {
		return payload
	}
	sections["openGaps"] = gaps
	out, err := json.Marshal(sections)
	if err != nil {
		return payload
	}
	return out
}

// toJSONPointer turns "scope.in[0].text" into "/scope/in/0/text".
func toJSONPointer(p string) string {
	// The same path also arrives in JSONPath style ($.metrics[0].baseline).
	p = strings.TrimPrefix(strings.TrimPrefix(p, "$"), ".")
	var b strings.Builder
	for _, seg := range strings.Split(p, ".") {
		name, idx := seg, ""
		if i := strings.Index(seg, "["); i >= 0 {
			name, idx = seg[:i], strings.Trim(seg[i:], "[]")
		}
		if name != "" {
			b.WriteString("/" + name)
		}
		if idx != "" {
			b.WriteString("/" + idx)
		}
	}
	return b.String()
}

// gapGate drops the shared blocking-gap gate: the operator does not patch the
// gap list itself, the gap question already went out as a request.
func gapGate(failures []domain.ValidationError) []domain.ValidationError {
	var out []domain.ValidationError
	for _, failure := range failures {
		if strings.HasPrefix(failure.Field, "openGaps[") {
			continue
		}
		out = append(out, failure)
	}
	return out
}

// AnswerOnPipeline applies the answer to the form that waits for the operator in
// this pipeline. The operator request names the pointer, not the form, so the
// service finds the blocked form itself.
func (s *QuestionService) AnswerOnPipeline(ctx context.Context, pipelineID int, pointer string, value json.RawMessage) (*domain.Form, error) {
	pipeline := strconv.Itoa(pipelineID)
	for _, formType := range domain.FormTypesWithGates() {
		versions, err := s.forms.ByPipeline(ctx, pipeline, formType)
		if err != nil {
			return nil, err
		}
		if len(versions) == 0 || versions[len(versions)-1].Status != domain.StatusWaitingOperator {
			continue
		}
		return s.Answer(ctx, versions[len(versions)-1], pointer, value, pipelineID)
	}
	return nil, fmt.Errorf("%w: pipeline %d has no form waiting on %s", domain.ErrFormNotFound, pipelineID, pointer)
}

// closeRequests completes every active question that waits on this pointer:
// repeats of the same failure had stacked without this loop.
func (s *QuestionService) closeRequests(ctx context.Context, pipelineID int, pointer string) {
	if s.requests == nil {
		return
	}
	all, err := s.requests.FindAll()
	if err != nil {
		return
	}
	for i := range all {
		request := &all[i]
		if request.Pointer == pointer && request.PipelineID == pipelineID && request.Status == "active" {
			request.Status = "completed"
			_ = s.requests.Update(request)
		}
	}
}
