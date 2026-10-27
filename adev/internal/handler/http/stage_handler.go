package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

// StageRunStarter accepts a stage run: the answer carries the run row at once and
// the stage chain goes in the background.
type StageRunStarter interface {
	Start(ctx context.Context, request usecase.StageRequest) (*domain.Run, error)
}

// StageHandler starts a stage.
type StageHandler struct {
	stages StageRunStarter
}

// NewStageHandler builds the stage handler.
func NewStageHandler(stages StageRunStarter) *StageHandler {
	return &StageHandler{stages: stages}
}

// SetupStageRoutes registers the stage routes.
func SetupStageRoutes(router *mux.Router, handler *StageHandler) {
	router.HandleFunc("/targetsystems/{id}/stages/{stage}/run", handler.Run).Methods(http.MethodPost)
}

// StageRunRequest is the body of a stage run.
type StageRunRequest struct {
	PipelineID string `json:"pipelineId" example:"1"`
	Route      string `json:"route" example:"greenfield"`
}

// Run accepts one stage run and answers at once: the run row carries the id the
// board and the runs table show while the chain works in the background.
// @Summary Run a stage
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path int true "Target system ID"
// @Param stage path string true "Stage ID" Enums(00, 01, 02, 03, 04, 05, 06)
// @Param body body StageRunRequest true "Pipeline and route"
// @Success 202 {object} domain.Run "The run was accepted; its row moves running -> done | blocked | failed | stopped"
// @Failure 400 {object} map[string]string
// @Failure 502 {object} map[string]string "The run could not be recorded"
// @Router /targetsystems/{id}/stages/{stage}/run [post]
func (h *StageHandler) Run(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid target system id")
		return
	}
	var body StageRunRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if _, err := strconv.Atoi(body.PipelineID); err != nil || body.PipelineID == "" {
		writeError(w, http.StatusBadRequest, "pipelineId must be a number: the run does not exist without its pipeline")
		return
	}
	route := domain.Route(body.Route)
	if route == "" {
		route = domain.RouteGreenfield
	}
	if route != domain.RouteGreenfield {
		writeError(w, http.StatusBadRequest, "unknown route")
		return
	}

	run, err := h.stages.Start(r.Context(), usecase.StageRequest{
		PipelineID:     body.PipelineID,
		TargetSystemID: id,
		Route:          route,
		StageID:        mux.Vars(r)["stage"],
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}
