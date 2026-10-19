package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

type PipelineHandler struct {
	usecase usecase.PipelineUseCase
}

func NewPipelineHandler(uc usecase.PipelineUseCase) *PipelineHandler {
	return &PipelineHandler{usecase: uc}
}

// Create pipeline godoc
// @Summary Create pipeline
// @Tags pipelines
// @Accept json
// @Produce json
// @Param body body usecase.CreatePipelineInput true "Pipeline payload"
// @Success 201 {object} usecase.CreatePipelineOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /pipelines [post]
func (p *PipelineHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreatePipelineInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validator.New().Struct(input); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	output, err := p.usecase.CreatePipeline(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}

// Get pipeline by ID godoc
// @Summary Get pipeline by ID
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path int true "Pipeline ID"
// @Success 200 {object} domain.Pipeline
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /pipelines/{id} [get]
func (p *PipelineHandler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid pipeline ID", http.StatusBadRequest)
		return
	}

	pipeline, err := p.usecase.GetPipeline(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pipeline)
}

// Update pipeline godoc
// @Summary Update pipeline
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path int true "Pipeline ID"
// @Param body body usecase.UpdatePipelineInput true "Pipeline payload"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /pipelines/{id} [put]
func (p *PipelineHandler) UpdatePipeline(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid pipeline ID", http.StatusBadRequest)
		return
	}

	var input usecase.UpdatePipelineInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := validator.New().Struct(input); err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if err := p.usecase.UpdatePipeline(id, input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete pipeline godoc
// @Summary Delete pipeline
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path int true "Pipeline ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /pipelines/{id} [delete]
func (p *PipelineHandler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid pipeline ID", http.StatusBadRequest)
		return
	}

	if err := p.usecase.DeletePipeline(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List pipelines godoc
// @Summary List pipelines
// @Tags pipelines
// @Accept json
// @Produce json
// @Success 200 {object} usecase.ListPipelineOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /pipelines [get]
func (p *PipelineHandler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	output, err := p.usecase.ListPipelines()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}
