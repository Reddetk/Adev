package http

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Reddetk/Adev/internal/usecase"
)

// ActivityBook reads the live state of recent stage runs.
type ActivityBook interface {
	List() []usecase.Activity
}

// ActivityHandler serves what the agents work on right now.
type ActivityHandler struct {
	book ActivityBook
}

// NewActivityHandler builds the activity handler.
func NewActivityHandler(book ActivityBook) *ActivityHandler {
	return &ActivityHandler{book: book}
}

// SetupActivityRoutes registers the activity route.
func SetupActivityRoutes(router *mux.Router, handler *ActivityHandler) {
	router.HandleFunc("/activities", handler.List).Methods(http.MethodGet)
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
