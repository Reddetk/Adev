package http

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

// AnswerService applies one operator answer to the waiting form.
type AnswerService interface {
	AnswerOnPipeline(ctx context.Context, pipelineID int, pointer string, value json.RawMessage) (*domain.Form, error)
	AnswerChoice(ctx context.Context, pipelineID int, asked, chosen string, value json.RawMessage) (*domain.Form, error)
	// AnswerManyOnPipeline applies a batch as one operator action: one payload
	// write and one gate check for the whole set.
	AnswerManyOnPipeline(ctx context.Context, pipelineID int, answers []usecase.AnswerInput) (*domain.Form, error)
}

// QuestionHandler applies the operator answer to a blocked form.
type QuestionHandler struct {
	questions AnswerService
	starter   StageRunStarter
}

// NewQuestionHandler builds the question handler. A nil starter leaves the stage
// blocked after the last answer, which is what the tests without a runner need.
func NewQuestionHandler(questions AnswerService, starter StageRunStarter) *QuestionHandler {
	return &QuestionHandler{questions: questions, starter: starter}
}

// SetupQuestionRoutes registers the answer routes.
func SetupQuestionRoutes(router *mux.Router, handler *QuestionHandler) {
	router.HandleFunc("/operatorrequests/answer", handler.Answer).Methods(http.MethodPost)
	router.HandleFunc("/operatorrequests/answers", handler.Answers).Methods(http.MethodPost)
}

// AnswerRequest is the body of an operator answer. The pointer is the one the
// question asked about; the value is raw JSON in the shape of that field.
type AnswerRequest struct {
	PipelineID int             `json:"pipelineId"`
	Pointer    string          `json:"pointer"`
	Value      json.RawMessage `json:"value"`
	// Asked is the pointer of the question when Pointer is the entry picked from choices.
	Asked string `json:"asked,omitempty"`
}

// Answer applies the value to the waiting form and closes the question.
// @Summary Answer an operator question
// @Tags operatorrequests
// @Accept json
// @Produce json
// @Param body body AnswerRequest true "Pipeline, pointer and the value"
// @Success 200 {object} map[string]any "The corrected form version"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string "No form waits on this pointer"
// @Failure 422 {object} map[string]string "The corrected payload failed the gates"
// @Router /operatorrequests/answer [post]
func (h *QuestionHandler) Answer(w http.ResponseWriter, r *http.Request) {
	var body AnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.PipelineID < 1 || body.Pointer == "" || len(body.Value) == 0 {
		writeError(w, http.StatusBadRequest, "pipelineId, pointer and value are required")
		return
	}

	var form *domain.Form
	var err error
	if body.Asked != "" {
		form, err = h.questions.AnswerChoice(r.Context(), body.PipelineID, body.Asked, body.Pointer, body.Value)
	} else {
		form, err = h.questions.AnswerOnPipeline(r.Context(), body.PipelineID, body.Pointer, body.Value)
	}
	if err != nil {
		var ambiguous *usecase.AmbiguousPointerError
		if errors.As(err, &ambiguous) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": err.Error(), "choices": ambiguous.Choices})
			return
		}
		writeError(w, questionStatus(err), err.Error())
		return
	}
	// The last answer leaves a draft with no question or completes the form: the
	// stage continues with a run either way, instead of waiting for the operator
	// to press the button. A valid form is confirmed by the run's first check, and
	// the orchestrator moves the cursor on; without that run the pipeline hangs on
	// the answered stage.
	h.continueStage(r, form)
	writeJSON(w, http.StatusOK, form)
}

// AnswersRequest is the body of a batch of operator answers: the console sends
// one request per round of questions.
type AnswersRequest struct {
	PipelineID int           `json:"pipelineId"`
	Answers    []BatchAnswer `json:"answers"`
}

// BatchAnswer is one decision of a batch.
type BatchAnswer struct {
	Pointer string          `json:"pointer"`
	Value   json.RawMessage `json:"value"`
}

// Answers applies a batch of answers as one operator action and restarts the
// stage once.
// @Summary Answer a batch of operator questions
// @Tags operatorrequests
// @Accept json
// @Produce json
// @Param body body AnswersRequest true "Pipeline and the answers"
// @Success 200 {object} map[string]any "The corrected form version"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string "No form waits on the first pointer"
// @Failure 422 {object} map[string]string "A pointer outside the draft or a repeated pointer"
// @Router /operatorrequests/answers [post]
func (h *QuestionHandler) Answers(w http.ResponseWriter, r *http.Request) {
	var body AnswersRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.PipelineID < 1 || len(body.Answers) == 0 {
		writeError(w, http.StatusBadRequest, "pipelineId and at least one answer are required")
		return
	}
	answers := make([]usecase.AnswerInput, 0, len(body.Answers))
	for _, answer := range body.Answers {
		if answer.Pointer == "" || len(answer.Value) == 0 {
			writeError(w, http.StatusBadRequest, "every answer needs a pointer and a value")
			return
		}
		answers = append(answers, usecase.AnswerInput{Pointer: answer.Pointer, Value: answer.Value})
	}

	form, err := h.questions.AnswerManyOnPipeline(r.Context(), body.PipelineID, answers)
	if err != nil {
		writeError(w, questionStatus(err), err.Error())
		return
	}
	h.continueStage(r, form)
	writeJSON(w, http.StatusOK, form)
}

// continueStage starts the stage again when the answers left it runnable.
func (h *QuestionHandler) continueStage(r *http.Request, form *domain.Form) {
	if h.starter == nil || form == nil || form.Stage == "" {
		return
	}
	if form.Status != domain.StatusDraft && form.Status != domain.StatusValid {
		return
	}
	targetSystemID, _ := strconv.ParseInt(form.TargetSystemID, 10, 64)
	if _, err := h.starter.Start(r.Context(), usecase.StageRequest{
		PipelineID:     form.PipelineID,
		TargetSystemID: targetSystemID,
		Route:          domain.Route(form.Route),
		StageID:        form.Stage,
	}); err != nil {
		log.Printf("question pipeline %s: continue stage %s: %v", form.PipelineID, form.Stage, err)
	}
}

func questionStatus(err error) int {
	switch {
	case formStatus(err) != http.StatusInternalServerError:
		return formStatus(err)
	default:
		return http.StatusUnprocessableEntity
	}
}
