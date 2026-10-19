package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/gorilla/mux"
)

type ModelCallHandler struct {
	usecase usecase.ModelCallUseCase
}

func NewModelCallHandler(uc usecase.ModelCallUseCase) *ModelCallHandler {
	return &ModelCallHandler{usecase: uc}
}

// Create modelcall godoc
// @Summary Create modelcall
// @Tags modelcalls
// @Accept json
// @Produce json
// @Param body body usecase.CreateModelCallInput true "ModelCall payload"
// @Success 201 {object} usecase.CreateModelCallOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /modelcalls [post]
func (m *ModelCallHandler) CreateModelCall(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateModelCallInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	output, err := m.usecase.CreateModelCall(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}

// Get modelcall by ID godoc
// @Summary Get modelcall by ID
// @Tags modelcalls
// @Accept json
// @Produce json
// @Param id path int true "ModelCall ID"
// @Success 200 {object} domain.ModelCall
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /modelcalls/{id} [get]
func (m *ModelCallHandler) GetModelCall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid modelcall ID", http.StatusBadRequest)
		return
	}

	modelcall, err := m.usecase.GetModelCall(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modelcall)
}

// Update modelcall godoc
// @Summary Update modelcall
// @Tags modelcalls
// @Accept json
// @Produce json
// @Param id path int true "ModelCall ID"
// @Param body body usecase.UpdateModelCallInput true "ModelCall payload"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /modelcalls/{id} [put]
func (m *ModelCallHandler) UpdateModelCall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid modelcall ID", http.StatusBadRequest)
		return
	}

	var input usecase.UpdateModelCallInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := m.usecase.UpdateModelCall(id, input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete modelcall godoc
// @Summary Delete modelcall
// @Tags modelcalls
// @Accept json
// @Produce json
// @Param id path int true "ModelCall ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /modelcalls/{id} [delete]
func (m *ModelCallHandler) DeleteModelCall(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid modelcall ID", http.StatusBadRequest)
		return
	}

	if err := m.usecase.DeleteModelCall(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List modelcalls godoc
// @Summary List modelcalls
// @Tags modelcalls
// @Accept json
// @Produce json
// @Success 200 {object} usecase.ListModelCallOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /modelcalls [get]
func (m *ModelCallHandler) ListModelCalls(w http.ResponseWriter, r *http.Request) {
	output, err := m.usecase.ListModelCalls()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}
