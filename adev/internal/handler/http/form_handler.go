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
	router.HandleFunc("/forms/{formId}", handler.DeleteForm).Methods(http.MethodDelete)
	router.HandleFunc("/forms/{formId}/latest", handler.Latest).Methods(http.MethodGet)
	router.HandleFunc("/forms/{formId}/versions/{version}", handler.GetVersion).Methods(http.MethodGet)
	router.HandleFunc("/forms/{formId}/versions/{version}", handler.DeleteVersion).Methods(http.MethodDelete)
	router.HandleFunc("/forms/{formId}/versions/{version}/status", handler.SetStatus).Methods(http.MethodPut)
	router.HandleFunc("/forms/{formId}/versions/{version}/payload", handler.UpdatePayload).Methods(http.MethodPut)
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

// DeleteForm removes every version of a document. It is the escape hatch for a
// document that blocks the pipeline and cannot be repaired by an operator answer.
// @Summary Delete a form document
// @Tags forms
// @Param formId path string true "Form ID"
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /forms/{formId} [delete]
func (h *FormHandler) DeleteForm(w http.ResponseWriter, r *http.Request) {
	if err := h.forms.Delete(r.Context(), mux.Vars(r)["formId"], 0); err != nil {
		writeError(w, formStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteVersion removes one version of a form.
// @Summary Delete one form version
// @Tags forms
// @Param formId path string true "Form ID"
// @Param version path int true "Form version"
// @Success 204
// @Failure 404 {object} map[string]string
// @Router /forms/{formId}/versions/{version} [delete]
func (h *FormHandler) DeleteVersion(w http.ResponseWriter, r *http.Request) {
	version, ok := formVersion(w, r)
	if !ok {
		return
	}
	if err := h.forms.Delete(r.Context(), mux.Vars(r)["formId"], version); err != nil {
		writeError(w, formStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UpdatePayload rewrites the payload of a stored version in place. Only a version
// that waits for the operator (or that the operator first moved to that state)
// may change its payload, matching the append-only rule of the forms table.
// @Summary Rewrite the payload of a form version
// @Tags forms
// @Accept json
// @Produce json
// @Param formId path string true "Form ID"
// @Param version path int true "Form version"
// @Param body body map[string]any true "New payload"
// @Success 200 {object} map[string]any "Stored form"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /forms/{formId}/versions/{version}/payload [put]
func (h *FormHandler) UpdatePayload(w http.ResponseWriter, r *http.Request) {
	version, ok := formVersion(w, r)
	if !ok {
		return
	}
	formID := mux.Vars(r)["formId"]
	var body struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.Payload) == 0 {
		writeError(w, http.StatusBadRequest, "payload is required")
		return
	}
	form, err := h.forms.Get(r.Context(), formID, version)
	if err != nil {
		writeError(w, formStatus(err), err.Error())
		return
	}
	if err := form.SetRawPayload(body.Payload); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := h.forms.UpdatePayload(r.Context(), form); err != nil {
		writeError(w, formStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, form)
}

// ByPipeline reads the forms of one pipeline. Every filter is optional: the
// document list reads all types, then narrows by status or stage.
// @Summary List the forms of a pipeline
// @Tags forms
// @Produce json
// @Param pipelineId path string true "Pipeline ID"
// @Param type query string false "Form type" Enums(BRD, SRS, FR, NFR, HLD, ADR, LLD, TASK_PLAN, FEASIBILITY)
// @Param status query string false "Form status" Enums(draft, valid, approved, superseded)
// @Param stage query string false "Pipeline stage"
// @Success 200 {array} map[string]interface{} "Stored forms, oldest first"
// @Failure 400 {object} map[string]string
// @Router /pipelines/{pipelineId}/forms [get]
func (h *FormHandler) ByPipeline(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	formType := domain.FormType(query.Get("type"))
	if formType != "" && !formType.Valid() {
		writeError(w, http.StatusBadRequest, "unknown form type")
		return
	}
	forms, err := h.forms.ByPipeline(r.Context(), mux.Vars(r)["pipelineId"], formType)
	if err != nil {
		writeError(w, formStatus(err), err.Error())
		return
	}
	status, stage := domain.Status(query.Get("status")), query.Get("stage")
	matched := make([]*domain.Form, 0, len(forms))
	for _, form := range forms {
		if status != "" && form.Status != status {
			continue
		}
		if stage != "" && form.Stage != stage {
			continue
		}
		matched = append(matched, form)
	}
	writeJSON(w, http.StatusOK, matched)
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
