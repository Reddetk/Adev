package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/gorilla/mux"
)

type RunHandler struct {
	usecase usecase.RunUseCase
}

func NewRunHandler(uc usecase.RunUseCase) *RunHandler {
	return &RunHandler{usecase: uc}
}

// Create run godoc
// @Summary Create run
// @Tags runs
// @Accept json
// @Produce json
// @Param body body usecase.CreateRunInput true "Run payload"
// @Success 201 {object} usecase.CreateRunOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /runs [post]
func (r *RunHandler) CreateRun(w http.ResponseWriter, rt *http.Request) {
	var input usecase.CreateRunInput

	if err := json.NewDecoder(rt.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	output, err := r.usecase.CreateRun(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}

// Get run by ID godoc
// @Summary Get run by ID
// @Tags runs
// @Accept json
// @Produce json
// @Param id path int true "Run ID"
// @Success 200 {object} domain.Run
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /runs/{id} [get]
func (r *RunHandler) GetRun(w http.ResponseWriter, rt *http.Request) {
	vars := mux.Vars(rt)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid run ID", http.StatusBadRequest)
		return
	}

	run, err := r.usecase.GetRun(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// Update run godoc
// @Summary Update run
// @Tags runs
// @Accept json
// @Produce json
// @Param id path int true "Run ID"
// @Param body body usecase.UpdateRunInput true "Run payload"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /runs/{id} [put]
func (r *RunHandler) UpdateRun(w http.ResponseWriter, rt *http.Request) {
	vars := mux.Vars(rt)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid run ID", http.StatusBadRequest)
		return
	}

	var input usecase.UpdateRunInput
	if err := json.NewDecoder(rt.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := r.usecase.UpdateRun(id, input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete run godoc
// @Summary Delete run
// @Tags runs
// @Accept json
// @Produce json
// @Param id path int true "Run ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /runs/{id} [delete]
func (r *RunHandler) DeleteRun(w http.ResponseWriter, rt *http.Request) {
	vars := mux.Vars(rt)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid run ID", http.StatusBadRequest)
		return
	}

	if err := r.usecase.DeleteRun(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List runs godoc
// @Summary List runs
// @Tags runs
// @Accept json
// @Produce json
// @Success 200 {object} usecase.ListRunOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /runs [get]
func (r *RunHandler) ListRuns(w http.ResponseWriter, rt *http.Request) {
	output, err := r.usecase.ListRuns()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}
