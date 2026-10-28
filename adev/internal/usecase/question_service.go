package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"slices"
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

// questionsPerRun caps one batch of questions: a document that asks for 78 values
// at once buries the operator, so the rest wait for the next run.
const questionsPerRun = 5

// PublishGaps turns the open gaps of a stored draft into operator questions.
// One run asks at most questionsPerRun gaps; only a gap that names one answerable
// value goes out, because a structural hole (a whole layer, a package layout) has
// no single answer and the next pass of the stage fills it. Already-active requests
// are never asked again. The stage stays blocked while any of its questions waits
// for an answer, so the value valid arrives only after the last one.
// shortcut: one question per run is the registry rule; here a run is one small
// batch, reconcile with the registry when the ask-loop moves to model-driven
// conversation.
func (s *QuestionService) PublishGaps(ctx context.Context, form *domain.Form, pipelineID int) (published, blocked bool, err error) {
	if s.requests == nil || form == nil {
		return false, false, nil
	}
	// Questions whose document can no longer take an answer leave the dashboard.
	s.CloseStale(ctx, pipelineID)
	// A batch already waits for this document: the answers publish the next batch,
	// a re-run must not stack a second one on top.
	if s.hasActiveRequestsForForm(pipelineID, form) {
		return false, true, nil
	}
	asked := 0
	for _, position := range form.Gaps() {
		gap := position.Gap
		pointer := normalizePointer(gap.Pointer)
		if !askableGap(form.FormType, form.Payload, gap) {
			// The pointer is missing, names a list the draft does not have, or
			// needs a whole structure. An answer would die with 422 or write into
			// a junk field, so the question waits for the next pass, where the
			// model fills the structure from the answers it already has.
			log.Printf("question pipeline %d: gap %s of %s is not one answerable value (%s), not asked", pipelineID, gap.ID, form.FormID, pointer)
			continue
		}
		if alreadyAsked(ctx, s.requests, pipelineID, pointer) {
			continue
		}
		question := fmt.Sprintf("Уточнить у оператора (%s, %s): %s", form.FormType, gap.ID, gap.Question.Value)
		if err := s.requests.Save(&domain.OperatorRequest{
			PipelineID: pipelineID,
			Pointer:    pointer,
			Question:   question,
			Options:    gap.Options,
			Format:     answerFormat(form.FormType, pointer, gap.Options),
			Blocking:   gap.Blocking,
			Status:     "active",
		}); err != nil {
			return published, false, err
		}
		published = true
		asked++
		if asked >= questionsPerRun {
			break
		}
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
	pointer = normalizePointer(pointer)
	all, err := requests.FindAll()
	if err != nil {
		return false
	}
	for i := range all {
		if all[i].PipelineID == pipelineID && all[i].Status == "active" && normalizePointer(all[i].Pointer) == pointer {
			return true
		}
	}
	return false
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
	pointer = normalizePointer(pointer)
	return s.answerAt(ctx, form, pointer, pointer, value, pipelineID)
}

// answerAt writes the value to pointer. asked is the pointer the question named: the
// gap that asked it is removed even when the answer chose another entry.
func (s *QuestionService) answerAt(ctx context.Context, form *domain.Form, asked, pointer string, value json.RawMessage, pipelineID int) (*domain.Form, error) {
	asked = normalizePointer(asked)
	pointer = normalizePointer(pointer)
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

	// An answered gap leaves the payload: the decision went into its field. The
	// entries are read from the raw JSON: a payload broken elsewhere binds into no
	// gap, and the answered entry would stay forever.
	var removals []int
	for index, gapPointer := range rawGapPointers(form.Payload) {
		normalized := normalizePointer(gapPointer)
		if normalized == asked || resolvePointer(form.Payload, normalized) == pointer {
			removals = append(removals, index)
		}
	}
	// The operator answers in plain words; the field keeps the type it has.
	value = coerceAnswer(form.FormType, pointer, value)
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
		var ops []patchOp
		for _, parent := range parentsToCreate(doc, pointer) {
			ops = append(ops, patchOp{Op: "add", Path: parent, Value: json.RawMessage(`{}`)})
		}
		ops = append(ops, patchOp{Op: op, Path: pointer, Value: value})
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
	// The answer decides the waiting draft in place: the operator's words do not stack versions.
	derived := *form
	if err := derived.SetPayload(payload); err != nil {
		return nil, err
	}
	if err := s.forms.UpdatePayload(ctx, &derived); err != nil {
		return nil, err
	}

	// The answered value is decided: the request closes even when other gaps remain.
	s.closeRequests(ctx, pipelineID, pointer)
	// The gap gate wakes up only on a valid status: check the draft as a would-be valid form.
	derived.Status = domain.StatusValid
	if err := derived.ValidatePayloadWithStatus(); err != nil {
		log.Printf("question pipeline %d: answer %s still fails: %v", pipelineID, pointer, err)
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
		// A failure no question can answer (a payload that no longer decodes) must not
		// hold the version waiting: nothing would wake it up.
		remaining := answerable(gapGate(collectFailures(err)))
		if len(remaining) == 0 {
			// A question that can land on this version still waits in the dashboard:
			// keep the version waiting, so the pending answer finds it. Questions
			// for other documents do not hold this one.
			if s.hasActiveRequestsForForm(pipelineID, &derived) {
				derived.Status = domain.StatusWaitingOperator
				return &derived, nil
			}
			// Nothing answers anywhere and no question went out: falling back to
			// the stored draft lets the next stage run repair the payload. A
			// waiting version with no question would hold the pipeline forever
			// (the stalled pipeline 1 of the Email chat target).
			derived.Status = domain.StatusDraft
			if err := s.forms.SetStatus(ctx, derived.FormID, derived.FormVersion, domain.StatusDraft); err != nil {
				return nil, err
			}
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

// PublishGate turns a run's gate mismatch into one blocking operator question:
// the run path reaches the operator the same way the answer path does, so a
// mismatch the model left asks instead of silently blocking the stage.
func (s *QuestionService) PublishGate(ctx context.Context, form *domain.Form, failures []domain.ValidationError, pipelineID int) (bool, error) {
	if s.requests == nil || form == nil || len(failures) == 0 {
		return false, nil
	}
	failures = valueFailures(form.FormType, form.Payload, answerable(gapGate(failures)))
	if len(failures) == 0 {
		return false, nil
	}
	if err := s.ask(ctx, form, failures, pipelineID); err != nil {
		return false, err
	}
	return true, nil
}

// ask marks the form waiting and writes one operator request for the first failure.
func (s *QuestionService) ask(ctx context.Context, form *domain.Form, failures []domain.ValidationError, pipelineID int) error {
	failures = answerable(failures)
	failures = valueFailures(form.FormType, form.Payload, failures)
	if (len(failures) == 0 || s.requests == nil) && !s.hasActiveRequestsForForm(pipelineID, form) {
		// Nothing can be asked and nothing is open: a waiting version would hold the
		// stage with no question, so it returns to the stage for a re-run.
		previous := form.Status
		form.Status = domain.StatusDraft
		if previous != domain.StatusDraft {
			if err := s.forms.SetStatus(ctx, form.FormID, form.FormVersion, domain.StatusDraft); err != nil {
				log.Printf("question pipeline %d: set draft: %v", pipelineID, err)
			}
		}
		return nil
	}
	// The version waits even when no new question goes out: the questions already
	// in the dashboard must find it, or the next answer dies with "no form waiting".
	if form.Status != domain.StatusWaitingOperator {
		if err := s.forms.SetStatus(ctx, form.FormID, form.FormVersion, domain.StatusWaitingOperator); err != nil {
			log.Printf("question pipeline %d: set waiting_operator: %v", pipelineID, err)
		}
	}
	if len(failures) == 0 || s.requests == nil {
		return nil
	}
	first := failures[0]
	pointer := pointerOf(first.Field)
	// The dashboard already holds this question: do not ask twice.
	if all, err := s.requests.FindAll(); err == nil {
		for i := range all {
			if all[i].PipelineID == pipelineID && all[i].Pointer == pointer && all[i].Status == "active" {
				return nil
			}
		}
	}
	return s.requests.Save(&domain.OperatorRequest{
		PipelineID: pipelineID,
		Pointer:    pointer,
		Question:   fmt.Sprintf("Гейт бракует поле %s: %s. Что записать в поле формы %s?", first.Field, first.Problem, form.FormType),
		Options:    nil,
		Format:     answerFormat(form.FormType, pointer, nil),
		Blocking:   true,
		Status:     "active",
	})
}

// hasActiveRequests reports an open question for the pipeline: its answer must
// find the waiting version.
func (s *QuestionService) hasActiveRequests(pipelineID int) bool {
	if s.requests == nil {
		return false
	}
	all, err := s.requests.FindAll()
	if err != nil {
		return false
	}
	for i := range all {
		if all[i].PipelineID == pipelineID && all[i].Status == "active" {
			return true
		}
	}
	return false
}

// hasActiveRequestsForForm reports an open question whose pointer can land on the
// given form. A question for another document must not hold this one waiting:
// its answer looks for the form that carries the pointer, and this version would
// stay blocked forever otherwise.
func (s *QuestionService) hasActiveRequestsForForm(pipelineID int, form *domain.Form) bool {
	if s.requests == nil || form == nil {
		return false
	}
	all, err := s.requests.FindAll()
	if err != nil {
		return false
	}
	for i := range all {
		if all[i].PipelineID != pipelineID || all[i].Status != "active" {
			continue
		}
		if domain.Patchable(form.FormType, form.Payload, normalizePointer(all[i].Pointer)) {
			return true
		}
	}
	return false
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

// coerceAnswer shapes a plain operator answer into the type the field holds: a
// sentence becomes a Text, one sentence in a list becomes one entry, a number field
// takes a number. A value the operator already wrote as JSON is used as is.
func coerceAnswer(formType domain.FormType, pointer string, value json.RawMessage) json.RawMessage {
	var answer any
	if err := json.Unmarshal(value, &answer); err != nil {
		return value
	}
	switch answer.(type) {
	case map[string]any, []any:
		return value
	}
	shaped, ok := answerShape(fieldShape(formType, pointer), answer)
	if !ok {
		return value
	}
	encoded, err := json.Marshal(shaped)
	if err != nil {
		return value
	}
	return encoded
}

// fieldShape is the skeleton of the field a pointer names. The shape of the payload
// says what one entry of a list looks like even when the draft left the list out,
// so the answer lands in the place the model asked about.
func fieldShape(formType domain.FormType, pointer string) any {
	raw, err := domain.PayloadShape(formType)
	if err != nil {
		return nil
	}
	var skeleton any
	if err := json.Unmarshal([]byte(raw), &skeleton); err != nil {
		return nil
	}
	current := skeleton
	for _, raw := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		segment := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		switch node := current.(type) {
		case map[string]any:
			current = node[segment]
		case []any:
			if len(node) == 0 {
				return nil
			}
			// The skeleton holds one example entry: every index names that entry.
			current = node[0]
		default:
			return nil
		}
	}
	return current
}

// answerShape turns a scalar answer into the value the shape expects.
func answerShape(shape any, answer any) (any, bool) {
	switch node := shape.(type) {
	case []any:
		if len(node) == 0 {
			return nil, false
		}
		item, ok := answerShape(node[0], answer)
		if !ok {
			return nil, false
		}
		return []any{item}, true
	case map[string]any:
		if _, hasText := node["text"].(string); hasText {
			// A Text value: the words keep the shape and get the document language.
			if text, ok := answer.(string); ok {
				return map[string]any{"text": text, "lang": domain.LangSTE100}, true
			}
			return nil, false
		}
		if _, hasGiven := node["given"].(string); hasGiven {
			// A Scenario (given/when/then): the sentences of the answer fill the
			// three fields in order; the last sentence repeats when there are fewer.
			if text, ok := answer.(string); ok {
				sentences := splitSentences(text)
				return map[string]any{
					"given": sentences[0], "when": sentences[1], "then": sentences[2],
				}, true
			}
			return nil, false
		}
		// A record whose free text carries the sentence, such as the condition of
		// an error entry: the other fields stay empty for the model to fill.
		for _, key := range []string{"condition", "message", "description"} {
			if _, has := node[key].(string); has {
				if text, ok := answer.(string); ok {
					return map[string]any{key: text}, true
				}
			}
		}
	case string:
		switch typed := answer.(type) {
		case string:
			return typed, true
		case float64:
			// A text field keeps text even when the operator typed a bare number.
			return fmt.Sprintf("%v", typed), true
		}
	case float64:
		if text, ok := answer.(string); ok {
			// A number field: the number inside the sentence wins ("1 user" = 1).
			// A sentence without a number stays a string: the field refuses it and
			// the operator rewrites, instead of the payload breaking its shape.
			match := numberInSentence.FindString(strings.ReplaceAll(text, ",", "."))
			if number, err := strconv.ParseFloat(match, 64); match != "" && err == nil {
				return number, true
			}
		}
	}
	return nil, false
}

// numberInSentence finds the first number in a free-text answer.
var numberInSentence = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)

// splitSentences cuts an answer into one, two or three sentences: an answer
// with fewer sentences repeats its last one, so the three scenario fields
// always carry words.
func splitSentences(text string) [3]string {
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?' || r == ';' || r == '\n'
	})
	sentences := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			sentences = append(sentences, trimmed)
		}
	}
	if len(sentences) == 0 {
		return [3]string{text, text, text}
	}
	var out [3]string
	for i := 0; i < 3; i++ {
		index := i
		if index >= len(sentences) {
			index = len(sentences) - 1
		}
		out[i] = sentences[index]
	}
	return out
}

// answerFormat names the input control for the operator: a choice with options,
// a number where the shape holds a number, a list entry, otherwise plain text.
func answerFormat(formType domain.FormType, pointer string, options []string) string {
	if len(options) > 0 {
		return "choice"
	}
	switch fieldShape(formType, pointer).(type) {
	case float64:
		return "number"
	case []any:
		return "list"
	}
	return "text"
}

// rawGapPointers returns the pointers of the openGaps entries in list order, without
// binding the payload to its form type.
func rawGapPointers(payload json.RawMessage) []string {
	var doc struct {
		OpenGaps []struct {
			Pointer string `json:"pointer"`
		} `json:"openGaps"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil
	}
	pointers := make([]string, 0, len(doc.OpenGaps))
	for _, gap := range doc.OpenGaps {
		pointers = append(pointers, gap.Pointer)
	}
	return pointers
}

// valueAt walks a JSON document along a pointer and returns the value it names.
func valueAt(doc any, pointer string) any {
	pointer = normalizePointer(pointer)
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
	var root any
	_ = json.Unmarshal(payload, &root)
	changed := false
	for _, gap := range doc.OpenGaps {
		p, _ := gap["pointer"].(string)
		if p == "" {
			// The model sometimes names the field instead of the pointer; the
			// gates and the operator read one format only.
			p, _ = gap["field"].(string)
			if p == "" {
				continue
			}
			gap["pointer"] = p
			delete(gap, "field")
			changed = true
		}
		fixed := normalizePointer(p)
		// The model writes the path from the document root (/payload/metrics/0);
		// the payload is the document itself, so the root segment only breaks it.
		fixed = strings.TrimPrefix(fixed, "/payload")
		if fixed == "" {
			fixed = "/payload"
		}
		fixed = listSlot(root, fixed)
		if fixed != p {
			gap["pointer"] = fixed
			changed = true
		}
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

// purgeUnaskableGaps removes the gaps the operator cannot answer with one value:
// the pointer has no place in the payload (the model invented it or glued a note
// into the path), or it names a whole list or record that only the next pass can
// fill. A gap nobody can answer would refuse the draft through the openGaps
// pointer gate forever; runForm purges before the submit, so the gates judge only
// answerable questions.
func purgeUnaskableGaps(formType domain.FormType, payload json.RawMessage) json.RawMessage {
	probe := &domain.Form{Envelope: domain.Envelope{FormID: "purge", FormType: formType}}
	if err := probe.SetRawPayload(payload); err != nil {
		return payload
	}
	var drop []int
	for _, position := range probe.Gaps() {
		if position.Gap.Question.Value != "" && !askableGap(formType, probe.Payload, position.Gap) {
			drop = append(drop, position.Index)
			log.Printf("stage gap %s of %s is not one answerable value (%s), purged", position.Gap.ID, formType, position.Gap.Pointer)
		}
	}
	if len(drop) == 0 {
		return payload
	}
	ops := make([]patchOp, 0, len(drop))
	for i := len(drop) - 1; i >= 0; i-- {
		ops = append(ops, patchOp{Op: "remove", Path: "/openGaps/" + strconv.Itoa(drop[i])})
	}
	encoded, err := json.Marshal(ops)
	if err != nil {
		return payload
	}
	purged, err := ApplyDelta(payload, encoded)
	if err != nil {
		return payload
	}
	return purged
}

// askableGap reports whether the operator can answer the gap with one value: the
// pointer names a field of the draft and the field takes the shape an answer
// gets. A gap that needs a whole structure (a layer with packages, a package
// layout) is not a question: the next pass of the stage fills it from the answers
// it already has.
func askableGap(formType domain.FormType, payload json.RawMessage, gap domain.Gap) bool {
	if gap.Question.Value == "" {
		return false
	}
	pointer := normalizePointer(gap.Pointer)
	if pointer == "" || !strings.HasPrefix(pointer, "/") {
		return false
	}
	return valueSlot(formType, payload, pointer)
}

// valueFailures keeps the gate failures that name one answerable value. A finding
// about a whole list or a record (no layer, no package layout, no event map
// fields) has no single answer: the next pass of the stage fills the structure,
// so the run asks the operator nothing about it.
func valueFailures(formType domain.FormType, payload json.RawMessage, failures []domain.ValidationError) []domain.ValidationError {
	out := make([]domain.ValidationError, 0, len(failures))
	for _, failure := range failures {
		if valueSlot(formType, payload, pointerOf(failure.Field)) {
			out = append(out, failure)
		}
	}
	return out
}

// valueSlot reports whether one operator value can land on the pointer: the field
// exists in the shape and takes the shape an answer gets.
func valueSlot(formType domain.FormType, payload json.RawMessage, pointer string) bool {
	if !domain.Patchable(formType, payload, pointer) || !domain.ShapePointer(formType, pointer) {
		return false
	}
	shape := fieldShape(formType, pointer)
	// A pointer that names a whole list is a structure, not one value: an answer
	// would have to invent its entries. The next pass fills it from the values the
	// operator already gave.
	if _, isList := shape.([]any); isList {
		return false
	}
	if _, ok := answerShape(shape, "value"); ok {
		return true
	}
	// A number field takes no sentence: probe the numeric answer too ("1").
	_, ok := answerShape(shape, "1")
	return ok
}

// parentsToCreate lists the missing parents of a pointer, outermost first: the
// objects the answer adds before its value. "/options/0/cost/amount" with no cost
// gives ["/options/0/cost"]; a list entry past the end is a missing parent too.
func parentsToCreate(doc any, pointer string) []string {
	pointer = normalizePointer(pointer)
	segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	var missing []string
	current := doc
	for i := 0; i < len(segments)-1; i++ {
		var child any
		if current != nil {
			switch node := current.(type) {
			case map[string]any:
				child = node[segments[i]]
			case []any:
				index, err := strconv.Atoi(segments[i])
				if err != nil || index < 0 || index > len(node) {
					return missing
				}
				if index < len(node) {
					child = node[index]
				}
			default:
				return missing
			}
		}
		if child == nil {
			missing = append(missing, "/"+strings.Join(segments[:i+1], "/"))
		}
		current = child
	}
	return missing
}

// resolvePointer applies listSlot to a pointer of a stored payload.
func resolvePointer(payload json.RawMessage, pointer string) string {
	var root any
	if err := json.Unmarshal(payload, &root); err != nil {
		return pointer
	}
	return listSlot(root, pointer)
}

// prepareSlot makes room for an answer whose list the model wrote as a value
// the schema does not allow: a top-level list the shape names as an array
// becomes empty, and the answer creates its first entry. The value the model
// put there is replaced; the answer carries the operator's words instead.
func prepareSlot(formType domain.FormType, payload json.RawMessage, pointer string) (json.RawMessage, string) {
	pointer = normalizePointer(pointer)
	shape, err := domain.PayloadShape(formType)
	if err != nil {
		return payload, resolvePointer(payload, pointer)
	}
	var skeleton map[string]any
	var root map[string]any
	if json.Unmarshal([]byte(shape), &skeleton) != nil || json.Unmarshal(payload, &root) != nil {
		return payload, pointer
	}
	segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	if len(segments) < 2 {
		return payload, pointer
	}
	list := segments[0]
	_, shapeIsList := skeleton[list].([]any)
	_, indexErr := strconv.Atoi(segments[1])
	if shapeIsList && indexErr != nil {
		if _, isArray := root[list].([]any); !isArray {
			root[list] = []any{}
			repaired, err := json.Marshal(root)
			if err != nil {
				return payload, pointer
			}
			payload = repaired
		}
	}
	return payload, resolvePointer(payload, pointer)
}

// entryIDs reads an entry named by its id ("/options/OPT-1/cost") as the index of
// that entry: the model writes entries both ways, and the id is not a guess.
func entryIDs(doc any, pointer string) string {
	pointer = normalizePointer(pointer)
	segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	current := doc
	for i, segment := range segments {
		switch node := current.(type) {
		case map[string]any:
			current = node[segment]
		case []any:
			if index, err := strconv.Atoi(segment); err == nil {
				if index < 0 || index >= len(node) {
					return pointer
				}
				current = node[index]
				continue
			}
			found := -1
			for index, entry := range node {
				if fields, ok := entry.(map[string]any); ok && fields["id"] == segment {
					found = index
					break
				}
			}
			if found < 0 {
				return pointer
			}
			segments[i] = strconv.Itoa(found)
			current = node[found]
		default:
			// A missing field ends the walk: the rest of the path names keys to create.
			if current == nil {
				return "/" + strings.Join(segments, "/")
			}
			return pointer
		}
	}
	return "/" + strings.Join(segments, "/")
}

// objectChoices offers the fields of an object place when the answer is not an
// object: the operator picks the field, the service does not split one value over
// several fields. A text field is not an object place: its answer becomes a Text.
func objectChoices(formType domain.FormType, pointer string, value json.RawMessage) []PointerChoice {
	var answer any
	if err := json.Unmarshal(coerceAnswer(formType, pointer, value), &answer); err == nil {
		if _, ok := answer.(map[string]any); ok {
			return nil
		}
	}
	fields, ok := fieldShape(formType, pointer).(map[string]any)
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	choices := make([]PointerChoice, 0, len(keys))
	for _, key := range keys {
		choices = append(choices, PointerChoice{Pointer: pointer + "/" + key, Label: key})
	}
	return choices
}

// listSlot gives a field of a list its entry: "/goals/outcome" becomes
// "/goals/0/outcome" when goals is empty (the answer creates the first entry) or
// holds exactly one entry (the only slot there is). Several entries keep the
// pointer: the index cannot be guessed there.
func listSlot(doc any, pointer string) string {
	pointer = normalizePointer(pointer)
	pointer = entryIDs(doc, pointer)
	segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	current := doc
	for i, segment := range segments {
		switch node := current.(type) {
		case map[string]any:
			current = node[segment]
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil {
				// "-" is the RFC 6902 append marker: it is a slot already, never a field.
				if len(node) > 1 || segment == "-" {
					return pointer
				}
				return "/" + strings.Join(slices.Insert(slices.Clone(segments), i, "0"), "/")
			}
			if index < 0 || index >= len(node) {
				return pointer
			}
			current = node[index]
		default:
			return pointer
		}
	}
	return pointer
}

// normalizePointer turns any pointer the model, the operator or a gate may send
// into a JSON Pointer: "scope.in[0].text", "/scope/in[0]/text" and
// "$.metrics[0].baseline" all become "/scope/in/0/text".
func normalizePointer(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if !strings.HasPrefix(p, "/") {
		return toJSONPointer(p)
	}
	// Slash form with bracket indices: /nfr[0]/unit -> /nfr/0/unit.
	return regexp.MustCompile(`\[([^\]]+)\]`).ReplaceAllString(p, "/$1")
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
			// The model names a list entry by its id: contracts[id=CTR-A] reads
			// like contracts[0], and entryIDs resolves the id to the slot.
			if id, ok := strings.CutPrefix(idx, "id="); ok {
				idx = strings.Trim(id, `"'`)
			}
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
// service finds the blocked form itself: two stages can wait at once, and only the
// one whose payload holds the pointer can take the answer.
func (s *QuestionService) AnswerOnPipeline(ctx context.Context, pipelineID int, pointer string, value json.RawMessage) (*domain.Form, error) {
	return s.answerOnPipeline(ctx, pipelineID, pointer, pointer, value)
}

// AnswerChoice answers a question whose pointer named a field of a list with
// several entries: the operator picked the entry, so the answer lands on the chosen
// pointer and the question about the asked pointer closes.
func (s *QuestionService) AnswerChoice(ctx context.Context, pipelineID int, asked, chosen string, value json.RawMessage) (*domain.Form, error) {
	return s.answerOnPipeline(ctx, pipelineID, asked, chosen, value)
}

// PointerChoice is one entry an operator can pick for an ambiguous pointer.
type PointerChoice struct {
	Pointer string `json:"pointer"`
	Label   string `json:"label"`
}

// AmbiguousPointerError reports a pointer that names a field of a list with several
// entries without an index. The service does not guess the entry: the operator picks.
type AmbiguousPointerError struct {
	Pointer string
	Choices []PointerChoice
}

func (e *AmbiguousPointerError) Error() string {
	return fmt.Sprintf("pointer %s does not name one place of the draft: pick the entry or the field", e.Pointer)
}

func (s *QuestionService) answerOnPipeline(ctx context.Context, pipelineID int, asked, pointer string, value json.RawMessage) (*domain.Form, error) {
	asked = normalizePointer(asked)
	pointer = normalizePointer(pointer)
	pipeline := strconv.Itoa(pipelineID)
	// An answer to a dead question dismisses it instead of failing forever.
	s.CloseStale(ctx, pipelineID)
	waiting := false
	var choices []PointerChoice
	for _, formType := range domain.FormTypesWithGates() {
		versions, err := s.forms.ByPipeline(ctx, pipeline, formType)
		if err != nil {
			return nil, err
		}
		if len(versions) == 0 {
			continue
		}
		// A document whose newest version already passed the gates is finished:
		// its older waiting versions are stale, and an answer there would patch
		// a junk draft instead of the form the question was asked about.
		if last := versions[len(versions)-1]; last.Status == domain.StatusValid || last.Status == domain.StatusApproved {
			continue
		}
		// The waiting version may hide under a newer draft a rerun stacked over
		// it: the questions belong to the waiting one, so the answer lands there.
		waitingIdx := -1
		for i := len(versions) - 1; i >= 0; i-- {
			if versions[i].Status == domain.StatusWaitingOperator {
				waitingIdx = i
				break
			}
		}
		if waitingIdx < 0 {
			continue
		}
		waiting = true
		latest := versions[waitingIdx]
		// A question asked before the index rule still holds the pointer without
		// the index: the answer lands on the slot the rule names.
		payload, slot := prepareSlot(latest.FormType, latest.Payload, pointer)
		if !domain.Patchable(latest.FormType, payload, slot) {
			if choices == nil {
				choices = entryChoices(payload, slot)
			}
			continue
		}
		if choices := objectChoices(latest.FormType, slot, value); len(choices) > 0 {
			return nil, &AmbiguousPointerError{Pointer: asked, Choices: choices}
		}
		prepared := *latest
		prepared.Payload = payload
		answered, err := s.answerAt(ctx, &prepared, asked, slot, value, pipelineID)
		if err == nil {
			s.closeRequests(ctx, pipelineID, pointer)
			if asked != pointer {
				s.closeRequests(ctx, pipelineID, asked)
			}
		}
		return answered, err
	}
	if len(choices) > 0 {
		return nil, &AmbiguousPointerError{Pointer: asked, Choices: choices}
	}
	if waiting {
		return nil, fmt.Errorf("pointer %s has no place in the draft: check the list index, such as /metrics/1/unit", pointer)
	}
	return nil, fmt.Errorf("%w: pipeline %d has no form waiting on %s", domain.ErrFormNotFound, pipelineID, pointer)
}

// entryChoices lists the entries a pointer may mean when its first list without an
// index holds several entries; nil when no such list stands on the path.
func entryChoices(payload json.RawMessage, pointer string) []PointerChoice {
	var doc any
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil
	}
	segments := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	current := doc
	for i, segment := range segments {
		switch node := current.(type) {
		case map[string]any:
			current = node[segment]
		case []any:
			if index, err := strconv.Atoi(segment); err == nil {
				if index < 0 || index >= len(node) {
					return nil
				}
				current = node[index]
				continue
			}
			if segment == "-" || len(node) < 2 {
				return nil
			}
			choices := make([]PointerChoice, 0, len(node))
			for index, entry := range node {
				path := append(slices.Clone(segments[:i]), strconv.Itoa(index))
				path = append(path, segments[i:]...)
				choices = append(choices, PointerChoice{
					Pointer: "/" + strings.Join(path, "/"),
					Label:   choiceLabel(index, entry),
				})
			}
			return choices
		default:
			return nil
		}
	}
	return nil
}

// choiceLabel names an entry for the operator: its name or id when it has one.
func choiceLabel(index int, entry any) string {
	if node, ok := entry.(map[string]any); ok {
		for _, key := range []string{"name", "id"} {
			if text, ok := node[key].(string); ok && text != "" {
				return fmt.Sprintf("#%d · %s", index, text)
			}
		}
	}
	return fmt.Sprintf("#%d", index)
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
		if normalizePointer(request.Pointer) == pointer && request.PipelineID == pipelineID && request.Status == "active" {
			request.Status = "completed"
			_ = s.requests.Update(request)
		}
	}
}

// waitingForm returns the version an answer to pointer would land on, or nil
// when no version can take it: the answer lookup skips a document type whose
// newest version already passed the gates, and a request for such a document
// can never be answered again.
func (s *QuestionService) waitingForm(ctx context.Context, pipelineID int, pointer string) *domain.Form {
	pipeline := strconv.Itoa(pipelineID)
	for _, formType := range domain.FormTypesWithGates() {
		versions, err := s.forms.ByPipeline(ctx, pipeline, formType)
		if err != nil || len(versions) == 0 {
			continue
		}
		if last := versions[len(versions)-1]; last.Status == domain.StatusValid || last.Status == domain.StatusApproved {
			continue
		}
		for i := len(versions) - 1; i >= 0; i-- {
			if versions[i].Status == domain.StatusWaitingOperator {
				payload, slot := prepareSlot(versions[i].FormType, versions[i].Payload, pointer)
				if domain.Patchable(versions[i].FormType, payload, slot) {
					return versions[i]
				}
				break
			}
		}
	}
	return nil
}

// CloseStale closes active requests no answer can ever reach, so the dashboard
// holds only the questions that still wait for the operator. It runs before the
// gap publication, where a stage run or an answer touches the service anyway.
func (s *QuestionService) CloseStale(ctx context.Context, pipelineID int) {
	if s.requests == nil || s.forms == nil {
		return
	}
	all, err := s.requests.FindAll()
	if err != nil {
		return
	}
	for i := range all {
		request := all[i]
		if request.PipelineID != pipelineID || request.Status != "active" {
			continue
		}
		if s.waitingForm(ctx, pipelineID, normalizePointer(request.Pointer)) == nil {
			log.Printf("question pipeline %d: request %d points nowhere answerable (%s), closed", pipelineID, request.ID, request.Pointer)
			s.closeRequests(ctx, pipelineID, request.Pointer)
		}
	}
}
