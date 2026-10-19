package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/gorilla/mux"
)

type AuditEventHandler struct {
	usecase usecase.AuditEventUseCase
}

func NewAuditEventHandler(uc usecase.AuditEventUseCase) *AuditEventHandler {
	return &AuditEventHandler{usecase: uc}
}

// Create auditevent godoc
// @Summary Create auditevent
// @Tags auditevents
// @Accept json
// @Produce json
// @Param body body usecase.CreateAuditEventInput true "AuditEvent payload"
// @Success 201 {object} usecase.CreateAuditEventOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auditevents [post]
func (a *AuditEventHandler) CreateAuditEvent(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateAuditEventInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	output, err := a.usecase.CreateAuditEvent(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}

// Get auditevent by ID godoc
// @Summary Get auditevent by ID
// @Tags auditevents
// @Accept json
// @Produce json
// @Param id path int true "AuditEvent ID"
// @Success 200 {object} domain.AuditEvent
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auditevents/{id} [get]
func (a *AuditEventHandler) GetAuditEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid auditevent ID", http.StatusBadRequest)
		return
	}

	auditevent, err := a.usecase.GetAuditEvent(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(auditevent)
}

// Update auditevent godoc
// @Summary Update auditevent
// @Tags auditevents
// @Accept json
// @Produce json
// @Param id path int true "AuditEvent ID"
// @Param body body usecase.UpdateAuditEventInput true "AuditEvent payload"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auditevents/{id} [put]
func (a *AuditEventHandler) UpdateAuditEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid auditevent ID", http.StatusBadRequest)
		return
	}

	var input usecase.UpdateAuditEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := a.usecase.UpdateAuditEvent(id, input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete auditevent godoc
// @Summary Delete auditevent
// @Tags auditevents
// @Accept json
// @Produce json
// @Param id path int true "AuditEvent ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auditevents/{id} [delete]
func (a *AuditEventHandler) DeleteAuditEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid auditevent ID", http.StatusBadRequest)
		return
	}

	if err := a.usecase.DeleteAuditEvent(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List auditevents godoc
// @Summary List auditevents
// @Tags auditevents
// @Accept json
// @Produce json
// @Success 200 {object} usecase.ListAuditEventOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auditevents [get]
func (a *AuditEventHandler) ListAuditEvents(w http.ResponseWriter, r *http.Request) {
	output, err := a.usecase.ListAuditEvents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}
