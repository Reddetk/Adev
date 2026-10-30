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

// SetupQuestionRoutes registers the answer route.
func SetupQuestionRoutes(router *mux.Router, handler *QuestionHandler) {
	router.HandleFunc("/operatorrequests/answer", handler.Answer).Methods(http.MethodPost)
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
	if h.starter != nil && (form.Status == domain.StatusDraft || form.Status == domain.StatusValid) && form.Stage != "" {
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
	writeJSON(w, http.StatusOK, form)
}

func questionStatus(err error) int {
	switch {
	case formStatus(err) != http.StatusInternalServerError:
		return formStatus(err)
	default:
		return http.StatusUnprocessableEntity
	}
}
