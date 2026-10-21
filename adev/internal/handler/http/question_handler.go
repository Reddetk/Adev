package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Reddetk/Adev/internal/usecase"
)

// QuestionHandler applies the operator answer to a blocked form.
type QuestionHandler struct {
	questions *usecase.QuestionService
}

// NewQuestionHandler builds the question handler.
func NewQuestionHandler(questions *usecase.QuestionService) *QuestionHandler {
	return &QuestionHandler{questions: questions}
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

	form, err := h.questions.AnswerOnPipeline(r.Context(), body.PipelineID, body.Pointer, body.Value)
	if err != nil {
		writeError(w, questionStatus(err), err.Error())
		return
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
