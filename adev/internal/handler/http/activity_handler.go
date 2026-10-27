package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Reddetk/Adev/internal/usecase"
)

// ActivityBook reads the live state of recent stage runs and stops them.
type ActivityBook interface {
	List() []usecase.Activity
	// Cancel stops the running stage runs of one agent role, or of the whole
	// pipeline when the role is empty, and returns how many runs were cancelled.
	Cancel(pipelineID, role string) int
}

// ActivityHandler serves what the agents work on right now.
type ActivityHandler struct {
	book ActivityBook
}

// NewActivityHandler builds the activity handler.
func NewActivityHandler(book ActivityBook) *ActivityHandler {
	return &ActivityHandler{book: book}
}

// SetupActivityRoutes registers the activity routes.
func SetupActivityRoutes(router *mux.Router, handler *ActivityHandler) {
	router.HandleFunc("/activities", handler.List).Methods(http.MethodGet)
	router.HandleFunc("/runs/stop", handler.Stop).Methods(http.MethodPost)
}

// StopRunRequest names the runs to stop. An empty role stops every agent of the pipeline.
type StopRunRequest struct {
	PipelineID string `json:"pipelineId" example:"3"`
	Role       string `json:"role,omitempty" example:"architect"`
}

// Stop cancels the level of the pipeline: the current attempt of the model still
// returns, but the gate after it stops the run instead of a new try.
// @Summary Stop running agents
// @Tags activities
// @Accept json
// @Produce json
// @Param body body StopRunRequest true "Pipeline and optional role"
// @Success 200 {object} map[string]int
// @Failure 400 {object} map[string]string
// @Router /runs/stop [post]
func (h *ActivityHandler) Stop(w http.ResponseWriter, r *http.Request) {
	var body StopRunRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.PipelineID == "" {
		http.Error(w, "pipelineId is required", http.StatusBadRequest)
		return
	}
	var cancelled int
	if h.book != nil {
		cancelled = h.book.Cancel(body.PipelineID, body.Role)
	}
	writeJSON(w, http.StatusOK, map[string]int{"cancelled": cancelled})
}

// List returns the recent stage runs, running first.
// @Summary Live agent activities
// @Tags activities
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /activities [get]
func (h *ActivityHandler) List(w http.ResponseWriter, r *http.Request) {
	activities := []usecase.Activity{}
	if h.book != nil {
		if listed := h.book.List(); listed != nil {
			activities = listed
		}
	}
	total := len(activities)
	writeJSON(w, http.StatusOK, map[string]any{"activities": activities, "total": total})
}
