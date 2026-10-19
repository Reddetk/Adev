package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

// StageRunner runs one stage of a pipeline.
type StageRunner interface {
	Run(ctx context.Context, request usecase.StageRequest) (usecase.StageOutcome, error)
}

// StageHandler starts a stage.
type StageHandler struct {
	stages StageRunner
}

// NewStageHandler builds the stage handler.
func NewStageHandler(stages StageRunner) *StageHandler {
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

// Run executes one stage and returns its outcome. A blocked stage answers 200 with
// blocked true and the reasons, because the model simply could not satisfy a gate.
// @Summary Run a stage
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path int true "Target system ID"
// @Param stage path string true "Stage ID" Enums(00, 01, 02, 03, 04, 05, 06)
// @Param body body StageRunRequest true "Pipeline and route"
// @Success 200 {object} usecase.StageOutcome
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string "The stage is not ready: a form or an agent config is missing"
// @Failure 502 {object} map[string]string "The model call failed"
// @Router /targetsystems/{id}/stages/{stage}/run [post]
func (h *StageHandler) Run(w http.ResponseWriter, r *http.Request) {
	// A stage waits on the model, which takes minutes, but the server write timeout
	// is ten seconds. Extend the deadline for this request only.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(20 * time.Minute))
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
	if body.PipelineID == "" {
		writeError(w, http.StatusBadRequest, "pipelineId is required")
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

	outcome, err := h.stages.Run(r.Context(), usecase.StageRequest{
		PipelineID:     body.PipelineID,
		TargetSystemID: id,
		Route:          route,
		StageID:        mux.Vars(r)["stage"],
	})
	if err != nil {
		writeError(w, stageStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, outcome)
}

func stageStatus(err error) int {
	var validation *domain.ValidationError
	switch {
	case errors.Is(err, usecase.ErrStageInputMissing), errors.Is(err, usecase.ErrAgentNotConfigured):
		return http.StatusConflict
	case errors.Is(err, domain.ErrSettingNotFound), errors.Is(err, domain.ErrFormNotFound):
		return http.StatusConflict
	case errors.As(err, &validation):
		return http.StatusConflict
	default:
		return http.StatusBadGateway
	}
}
