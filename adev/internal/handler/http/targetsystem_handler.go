package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

type TargetSystemHandler struct {
	usecase usecase.TargetSystemUseCase
}

func NewTargetSystemHandler(uc usecase.TargetSystemUseCase) *TargetSystemHandler {
	return &TargetSystemHandler{usecase: uc}
}

// Create targetsystem godoc
// @Summary Create targetsystem
// @Tags targetsystems
// @Accept json
// @Produce json
// @Param body body usecase.CreateTargetSystemInput true "TargetSystem payload"
// @Success 201 {object} usecase.CreateTargetSystemOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /targetsystems [post]
func (t *TargetSystemHandler) CreateTargetSystem(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateTargetSystemInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validator.New().Struct(input); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	output, err := t.usecase.CreateTargetSystem(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}

// Get targetsystem by ID godoc
// @Summary Get targetsystem by ID
// @Tags targetsystems
// @Accept json
// @Produce json
// @Param id path int true "TargetSystem ID"
// @Success 200 {object} domain.TargetSystem
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /targetsystems/{id} [get]
func (t *TargetSystemHandler) GetTargetSystem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid targetsystem ID", http.StatusBadRequest)
		return
	}

	targetsystem, err := t.usecase.GetTargetSystem(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(targetsystem)
}

// Update targetsystem godoc
// @Summary Update targetsystem
// @Tags targetsystems
// @Accept json
// @Produce json
// @Param id path int true "TargetSystem ID"
// @Param body body usecase.UpdateTargetSystemInput true "TargetSystem payload"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /targetsystems/{id} [put]
func (t *TargetSystemHandler) UpdateTargetSystem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid targetsystem ID", http.StatusBadRequest)
		return
	}

	var input usecase.UpdateTargetSystemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validator.New().Struct(input); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err := t.usecase.UpdateTargetSystem(id, input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete targetsystem godoc
// @Summary Delete targetsystem
// @Tags targetsystems
// @Accept json
// @Produce json
// @Param id path int true "TargetSystem ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /targetsystems/{id} [delete]
func (t *TargetSystemHandler) DeleteTargetSystem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid targetsystem ID", http.StatusBadRequest)
		return
	}

	if err := t.usecase.DeleteTargetSystem(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List targetsystems godoc
// @Summary List targetsystems
// @Tags targetsystems
// @Accept json
// @Produce json
// @Success 200 {object} usecase.ListTargetSystemOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /targetsystems [get]
func (t *TargetSystemHandler) ListTargetSystems(w http.ResponseWriter, r *http.Request) {
	output, err := t.usecase.ListTargetSystems()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}
