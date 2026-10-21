package http

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// LogSource reads the recent server log lines.
type LogSource interface {
	Lines() []string
}

// LogHandler serves the recent server log lines to the operator console.
type LogHandler struct {
	source LogSource
}

// NewLogHandler builds the log handler.
func NewLogHandler(source LogSource) *LogHandler {
	return &LogHandler{source: source}
}

// SetupLogRoutes registers the log route.
func SetupLogRoutes(router *mux.Router, handler *LogHandler) {
	router.HandleFunc("/logs", handler.List).Methods(http.MethodGet)
}

// List returns the recent log lines, oldest first.
// @Summary Recent server log lines
// @Tags logs
// @Produce json
// @Param limit query int false "How many lines to return, newest kept" default(200)
// @Success 200 {object} map[string]interface{}
// @Router /logs [get]
func (h *LogHandler) List(w http.ResponseWriter, r *http.Request) {
	lines := []string{}
	if h.source != nil {
		if kept := h.source.Lines(); kept != nil {
			lines = kept
		}
	}
	total := len(lines)

	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}

	writeJSON(w, http.StatusOK, map[string]any{"lines": lines, "total": total})
}
