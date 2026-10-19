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

// SettingsHandler configures a target system and plans its pipeline.
type SettingsHandler struct {
	settingsService *usecase.SettingsService
	pipelineService *usecase.Planner
	pipelines       usecase.PipelineUseCase
	models          ModelChecker
}

// ModelChecker calls the model of one role once. CheckDraft tests the values the
// operator typed without saving them.
type ModelChecker interface {
	Check(ctx context.Context, targetSystemID int64, role domain.Role) (usecase.ModelCheckResult, error)
	CheckDraft(ctx context.Context, draft domain.AgentConfig) (usecase.ModelCheckResult, error)
}

// NewSettingsHandler builds the settings handler.
func NewSettingsHandler(settingsService *usecase.SettingsService, pipelineService *usecase.Planner, pipelines usecase.PipelineUseCase, models ModelChecker) *SettingsHandler {
	return &SettingsHandler{settingsService: settingsService, pipelineService: pipelineService, pipelines: pipelines, models: models}
}

// SetupSettingsRoutes registers the settings and plan routes.
func SetupSettingsRoutes(router *mux.Router, handler *SettingsHandler) {
	router.HandleFunc("/targetsystems/{id}/workspace", handler.SetWorkspace).Methods(http.MethodPut)
	router.HandleFunc("/targetsystems/{id}/workspace", handler.GetWorkspace).Methods(http.MethodGet)
	router.HandleFunc("/targetsystems/{id}/agents", handler.ListAgents).Methods(http.MethodGet)
	router.HandleFunc("/targetsystems/{id}/agents/{role}", handler.SetAgent).Methods(http.MethodPut)
	router.HandleFunc("/targetsystems/{id}/agents/{role}/check", handler.CheckAgent).Methods(http.MethodPost)
	router.HandleFunc("/targetsystems/{id}/plan", handler.Plan).Methods(http.MethodGet)
	router.HandleFunc("/targetsystems/{id}/pipelines", handler.CreatePipeline).Methods(http.MethodPost)
}

// WorkspaceRequest is the body of SetWorkspace.
type WorkspaceRequest struct {
	Dir string `json:"dir" example:"D:\\work\\demo"`
}

// AgentRequest is the body of SetAgent. The api key is accepted and never returned.
type AgentRequest struct {
	Provider    string  `json:"provider" example:"ollama"`
	BaseURL     string  `json:"baseUrl,omitempty" example:"http://localhost:11434"`
	Model       string  `json:"model" example:"qwen3-coder:30b"`
	APIKey      string  `json:"apiKey,omitempty"`
	Temperature float64 `json:"temperature,omitempty" example:"0.2"`
	MaxTokens   int     `json:"maxTokens,omitempty" example:"8192"`
}

// WorkspaceResponse is the body of GetWorkspace.
type WorkspaceResponse struct {
	Dir string `json:"dir"`
}

// AgentListResponse is the body of ListAgents.
type AgentListResponse struct {
	Agents []domain.AgentView `json:"agents"`
}

// PipelineRequest is the body of CreatePipeline.
type PipelineRequest struct {
	Route string `json:"route" example:"greenfield"`
}

// PipelineCreatedResponse is the body of a successful CreatePipeline.
type PipelineCreatedResponse struct {
	ID        uint             `json:"id"`
	Route     domain.Route     `json:"route"`
	Stage     string           `json:"stage"`
	Readiness domain.Readiness `json:"readiness"`
}

// SetWorkspace stores the directory that agents may read and write.
// @Summary Set the workspace directory
// @Tags settings
// @Accept json
// @Produce json
// @Param id path int true "Target system ID"
// @Param body body WorkspaceRequest true "Workspace directory"
// @Success 200 {object} WorkspaceResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /targetsystems/{id}/workspace [put]
func (h *SettingsHandler) SetWorkspace(w http.ResponseWriter, r *http.Request) {
	id, ok := targetSystemID(w, r)
	if !ok {
		return
	}
	var body WorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.settingsService.SetWorkspace(r.Context(), id, body.Dir); err != nil {
		writeError(w, settingsStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, WorkspaceResponse{Dir: body.Dir})
}

// GetWorkspace reads the workspace directory.
// @Summary Get the workspace directory
// @Tags settings
// @Produce json
// @Param id path int true "Target system ID"
// @Success 200 {object} WorkspaceResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /targetsystems/{id}/workspace [get]
func (h *SettingsHandler) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	id, ok := targetSystemID(w, r)
	if !ok {
		return
	}
	dir, err := h.settingsService.Workspace(r.Context(), id)
	if err != nil {
		writeError(w, settingsStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, WorkspaceResponse{Dir: dir})
}

// CheckAgent calls the model of one role once and reports the answer or the real
// error of the provider. A stage hides both behind one failure.
// @Summary Check the model of one agent role
// @Tags settings
// @Produce json
// @Param id path int true "Target system ID"
// @Param role path string true "Agent role" Enums(analyst, architect, qa, dev, devops, operator)
// @Success 200 {object} usecase.ModelCheckResult
// @Failure 409 {object} map[string]string "No agent config for this role"
// @Failure 502 {object} map[string]string "The provider refused the call"
// @Router /targetsystems/{id}/agents/{role}/check [post]
func (h *SettingsHandler) CheckAgent(w http.ResponseWriter, r *http.Request) {
	id, ok := targetSystemID(w, r)
	if !ok {
		return
	}
	role := domain.Role(mux.Vars(r)["role"])
	if !role.Valid() {
		writeError(w, http.StatusUnprocessableEntity, "unknown role")
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Minute))

	// A body carries the fields as typed, so a model can be tried without saving it.
	var body AgentRequest
	_ = json.NewDecoder(r.Body).Decode(&body)

	var (
		result usecase.ModelCheckResult
		err    error
	)
	if body.Provider == "" {
		result, err = h.models.Check(r.Context(), id, role)
	} else {
		result, err = h.models.CheckDraft(r.Context(), domain.AgentConfig{
			TargetSystemID: id,
			Role:           role,
			Provider:       domain.Provider(body.Provider),
			BaseURL:        body.BaseURL,
			Model:          body.Model,
			APIKey:         body.APIKey,
			Temperature:    body.Temperature,
			MaxTokens:      body.MaxTokens,
		})
	}
	if err != nil {
		writeError(w, modelCheckStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func modelCheckStatus(err error) int {
	switch {
	case errors.Is(err, usecase.ErrAgentNotConfigured), errors.Is(err, domain.ErrSettingNotFound):
		return http.StatusConflict
	default:
		return http.StatusBadGateway
	}
}

// SetAgent stores one agent config. The api key is accepted and never returned.
// @Summary Set an agent model config
// @Tags settings
// @Accept json
// @Produce json
// @Param id path int true "Target system ID"
// @Param role path string true "Agent role" Enums(analyst, architect, qa, dev, devops, operator)
// @Param body body AgentRequest true "Agent model config"
// @Success 200 {object} domain.AgentView
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /targetsystems/{id}/agents/{role} [put]
func (h *SettingsHandler) SetAgent(w http.ResponseWriter, r *http.Request) {
	id, ok := targetSystemID(w, r)
	if !ok {
		return
	}
	role := domain.Role(mux.Vars(r)["role"])
	if !role.Valid() {
		writeError(w, http.StatusUnprocessableEntity, "unknown role")
		return
	}
	var body AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	config := domain.AgentConfig{
		TargetSystemID: id,
		Role:           role,
		Provider:       domain.Provider(body.Provider),
		BaseURL:        body.BaseURL,
		Model:          body.Model,
		APIKey:         body.APIKey,
		Temperature:    body.Temperature,
		MaxTokens:      body.MaxTokens,
	}
	if err := h.settingsService.SetAgent(r.Context(), config); err != nil {
		writeError(w, settingsStatus(err), err.Error())
		return
	}
	// Answer with the stored config: an empty key in the request kept the saved one,
	// so the request alone would report hasApiKey false.
	stored, err := h.settingsService.Agent(r.Context(), id, role)
	if err != nil {
		writeError(w, settingsStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stored.Masked())
}

// ListAgents returns the agent configs without secrets.
// @Summary List the agent model configs
// @Tags settings
// @Produce json
// @Param id path int true "Target system ID"
// @Success 200 {object} AgentListResponse
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /targetsystems/{id}/agents [get]
func (h *SettingsHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	id, ok := targetSystemID(w, r)
	if !ok {
		return
	}
	agents, err := h.settingsService.Agents(r.Context(), id)
	if err != nil {
		writeError(w, settingsStatus(err), err.Error())
		return
	}
	views := make([]domain.AgentView, 0, len(agents))
	for _, config := range agents {
		views = append(views, config.Masked())
	}
	writeJSON(w, http.StatusOK, AgentListResponse{Agents: views})
}

// Plan returns the stage sequence and the settings that the route still needs.
// @Summary Plan a pipeline
// @Tags pipelines
// @Produce json
// @Param id path int true "Target system ID"
// @Param route query string false "Pipeline route" Enums(greenfield, change)
// @Success 200 {object} usecase.Plan
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /targetsystems/{id}/plan [get]
func (h *SettingsHandler) Plan(w http.ResponseWriter, r *http.Request) {
	id, ok := targetSystemID(w, r)
	if !ok {
		return
	}
	route := domain.Route(r.URL.Query().Get("route"))
	if route == "" {
		route = domain.RouteGreenfield
	}
	plan, err := h.pipelineService.Plan(r.Context(), id, route)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// CreatePipeline creates the pipeline row when the route is ready.
// @Summary Create a pipeline
// @Tags pipelines
// @Accept json
// @Produce json
// @Param id path int true "Target system ID"
// @Param body body PipelineRequest true "Pipeline route"
// @Success 201 {object} PipelineCreatedResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} usecase.Plan "The route is not ready for this target system"
// @Failure 500 {object} map[string]string
// @Router /targetsystems/{id}/pipelines [post]
func (h *SettingsHandler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	id, ok := targetSystemID(w, r)
	if !ok {
		return
	}
	var body PipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	route := domain.Route(body.Route)
	if route == "" {
		route = domain.RouteGreenfield
	}

	plan, err := h.pipelineService.Plan(r.Context(), id, route)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !plan.Readiness.Ready {
		writeJSON(w, http.StatusConflict, plan)
		return
	}

	output, err := h.pipelines.CreatePipeline(usecase.CreatePipelineInput{
		TargetSystemID: int(id),
		Route:          string(route),
		Status:         "ready",
		CurrentStage:   plan.Stages[0].ID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, PipelineCreatedResponse{
		ID:        output.ID,
		Route:     route,
		Stage:     plan.Stages[0].ID,
		Readiness: plan.Readiness,
	})
}

func targetSystemID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid target system id")
		return 0, false
	}
	return id, true
}

func settingsStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrSettingNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrNoDatabase):
		return http.StatusServiceUnavailable
	case errors.Is(err, domain.ErrWorkspaceRequired),
		errors.Is(err, domain.ErrAPIKeyRequired),
		errors.Is(err, domain.ErrUnknownProvider),
		errors.Is(err, domain.ErrInvalidRole):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
