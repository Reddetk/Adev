package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

// FormHandler serves the form contract: submit a version, read versions, change status.
type FormHandler struct {
	forms *usecase.FormService
}

// NewFormHandler builds the form handler.
func NewFormHandler(forms *usecase.FormService) *FormHandler {
	return &FormHandler{forms: forms}
}

// SetupFormRoutes registers the form routes.
func SetupFormRoutes(router *mux.Router, handler *FormHandler) {
	router.HandleFunc("/forms", handler.Submit).Methods(http.MethodPost)
	router.HandleFunc("/forms/{formId}/latest", handler.Latest).Methods(http.MethodGet)
	router.HandleFunc("/forms/{formId}/versions/{version}", handler.GetVersion).Methods(http.MethodGet)
	router.HandleFunc("/forms/{formId}/versions/{version}/status", handler.SetStatus).Methods(http.MethodPut)
	router.HandleFunc("/pipelines/{pipelineId}/forms", handler.ByPipeline).Methods(http.MethodGet)
}

// FormRequest is the body of Submit: the envelope plus the payload.
type FormRequest struct {
	domain.Envelope
	Payload json.RawMessage `json:"payload"`
}

// StatusRequest is the body of SetStatus.
type StatusRequest struct {
	Status domain.Status `json:"status" example:"valid"`
}

// Submit stores a new form version after the envelope, the references, the language
// and the payload gates pass. The payload hash is computed here, not sent by the client.
// @Summary Submit a form version
// @Tags forms
// @Accept json
// @Produce json
// @Param body body FormRequest true "Form envelope and payload"
// @Success 201 {object} map[string]interface{} "Stored form"
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string "This form version already exists"
// @Failure 422 {object} map[string]string "A gate rejected the form"
// @Router /forms [post]
func (h *FormHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var body FormRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	form := &domain.Form{Envelope: body.Envelope}
	if err := form.SetRawPayload(body.Payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := h.forms.Submit(r.Context(), form); err != nil {
		writeError(w, formStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, form)
}

// Latest reads the highest version of a form.
// @Summary Read the latest form version
// @Tags forms
// @Produce json
// @Param formId path string true "Form ID"
// @Success 200 {object} map[string]interface{} "Stored form"
// @Failure 404 {object} map[string]string
// @Router /forms/{formId}/latest [get]
func (h *FormHandler) Latest(w http.ResponseWriter, r *http.Request) {
	form, err := h.forms.Latest(r.Context(), mux.Vars(r)["formId"])
	if err != nil {
		writeError(w, formStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, form)
}

// GetVersion reads one form version.
// @Summary Read one form version
// @Tags forms
// @Produce json
// @Param formId path string true "Form ID"
// @Param version path int true "Form version"
// @Success 200 {object} map[string]interface{} "Stored form"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /forms/{formId}/versions/{version} [get]
func (h *FormHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	version, ok := formVersion(w, r)
	if !ok {
		return
	}
	form, err := h.forms.Get(r.Context(), mux.Vars(r)["formId"], version)
	if err != nil {
		writeError(w, formStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, form)
}

// SetStatus changes the status of a stored version through the form lifecycle.
// @Summary Change the status of a form version
// @Tags forms
// @Accept json
// @Produce json
// @Param formId path string true "Form ID"
// @Param version path int true "Form version"
// @Param body body StatusRequest true "New status"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string "The lifecycle forbids this change"
// @Router /forms/{formId}/versions/{version}/status [put]
func (h *FormHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	version, ok := formVersion(w, r)
	if !ok {
		return
	}
	var body StatusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !body.Status.Valid() {
		writeError(w, http.StatusBadRequest, "unknown status")
		return
	}
	if err := h.forms.SetStatus(r.Context(), mux.Vars(r)["formId"], version, body.Status); err != nil {
		writeError(w, formStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ByPipeline reads the versions of one form type in one pipeline, oldest first.
// @Summary List the form versions of a pipeline
// @Tags forms
// @Produce json
// @Param pipelineId path string true "Pipeline ID"
// @Param type query string true "Form type" Enums(BRD, SRS, HLD, ADR, LLD, TASK_PLAN)
// @Success 200 {array} map[string]interface{} "Stored forms, oldest first"
// @Failure 400 {object} map[string]string
// @Router /pipelines/{pipelineId}/forms [get]
func (h *FormHandler) ByPipeline(w http.ResponseWriter, r *http.Request) {
	formType := domain.FormType(r.URL.Query().Get("type"))
	if formType == "" {
		writeError(w, http.StatusBadRequest, "the type query parameter is required")
		return
	}
	if !formType.Valid() {
		writeError(w, http.StatusBadRequest, "unknown form type")
		return
	}
	forms, err := h.forms.ByPipeline(r.Context(), mux.Vars(r)["pipelineId"], formType)
	if err != nil {
		writeError(w, formStatus(err), err.Error())
		return
	}
	if forms == nil {
		forms = []*domain.Form{}
	}
	writeJSON(w, http.StatusOK, forms)
}

func formVersion(w http.ResponseWriter, r *http.Request) (int, bool) {
	version, err := strconv.Atoi(mux.Vars(r)["version"])
	if err != nil || version < 1 {
		writeError(w, http.StatusBadRequest, "invalid form version")
		return 0, false
	}
	return version, true
}

func formStatus(err error) int {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		return http.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrNoPayloadGates):
		return http.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrMissingPrevious):
		return http.StatusUnprocessableEntity
	case errors.Is(err, domain.ErrFormNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrVersionExists), errors.Is(err, domain.ErrInvalidStatusChange):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
