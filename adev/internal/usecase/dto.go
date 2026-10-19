package usecase

import (
	"errors"
	"strings"
	"time"

	"github.com/Reddetk/Adev/internal/domain"
)

// CreateTargetSystemInput is the DTO for creating a new targetsystem.
type CreateTargetSystemInput struct {
	Name        string `json:"name" validate:"required,min=1"`
	RepoURL     string `json:"repourl" validate:"required,min=1"`
	Description string `json:"description" validate:"required,min=1"`
}

// Validate checks the CreateTargetSystemInput DTO fields.
func (r *CreateTargetSystemInput) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(r.RepoURL) == "" {
		return errors.New("repourl is required")
	}
	if strings.TrimSpace(r.Description) == "" {
		return errors.New("description is required")
	}
	return nil
}

// CreateTargetSystemOutput DTO for the creation response
type CreateTargetSystemOutput struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	RepoURL     string `json:"repourl"`
	Description string `json:"description"`
	Message     string `json:"message"`
}

type UpdateTargetSystemInput struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1"`
	RepoURL     *string `json:"repourl,omitempty" validate:"omitempty,min=1"`
	Description *string `json:"description,omitempty" validate:"omitempty,min=1"`
}

type ListTargetSystemOutput struct {
	TargetSystems []domain.TargetSystem `json:"targetsystems"`
	Total         int                   `json:"total"`
	Message       string                `json:"message"`
}

// CreatePipelineInput is the DTO for creating a new pipeline.
type CreatePipelineInput struct {
	TargetSystemID int    `json:"targetsystemid" validate:"required,min=1"`
	Route          string `json:"route" validate:"required,min=1"`
	Status         string `json:"status" validate:"required,min=1"`
	CurrentStage   string `json:"currentstage" validate:"required,min=1"`
}

// Validate checks the CreatePipelineInput DTO fields.
func (r *CreatePipelineInput) Validate() error {
	if r.TargetSystemID < 0 {
		return errors.New("targetsystemid must be a positive number")
	}
	if strings.TrimSpace(r.Route) == "" {
		return errors.New("route is required")
	}
	if strings.TrimSpace(r.Status) == "" {
		return errors.New("status is required")
	}
	if strings.TrimSpace(r.CurrentStage) == "" {
		return errors.New("age is required")
	}
	return nil
}

// CreatePipelineOutput DTO for the creation response
type CreatePipelineOutput struct {
	ID             uint   `json:"id"`
	TargetSystemID int    `json:"targetsystemid"`
	Route          string `json:"route"`
	Status         string `json:"status"`
	CurrentStage   string `json:"currentstage"`
	Message        string `json:"message"`
}

type UpdatePipelineInput struct {
	TargetSystemID *int    `json:"targetsystemid,omitempty" validate:"omitempty,min=1"`
	Route          *string `json:"route,omitempty" validate:"omitempty,min=1"`
	Status         *string `json:"status,omitempty" validate:"omitempty,min=1"`
	CurrentStage   *string `json:"currentstage,omitempty" validate:"omitempty,min=1"`
}

type ListPipelineOutput struct {
	Pipelines []domain.Pipeline `json:"pipelines"`
	Total     int               `json:"total"`
	Message   string            `json:"message"`
}

// CreateRunInput is the DTO for creating a new run.
type CreateRunInput struct {
	PipelineID   int    `json:"pipelineid"`
	Stage        string `json:"stage"`
	Status       string `json:"status"`
	CheckpointID string `json:"checkpointid"`
	Attempts     int    `json:"attempts"`
	LastError    string `json:"lasterror"`
}

// CreateRunOutput DTO for the creation response
type CreateRunOutput struct {
	ID           uint   `json:"id"`
	PipelineID   int    `json:"pipelineid"`
	Stage        string `json:"stage"`
	Status       string `json:"status"`
	CheckpointID string `json:"checkpointid"`
	Attempts     int    `json:"attempts"`
	LastError    string `json:"lasterror"`
	Message      string `json:"message"`
}

type UpdateRunInput struct {
	PipelineID   *int    `json:"pipelineid,omitempty"`
	Stage        *string `json:"stage,omitempty"`
	Status       *string `json:"status,omitempty"`
	CheckpointID *string `json:"checkpointid,omitempty"`
	Attempts     *int    `json:"attempts,omitempty"`
	LastError    *string `json:"lasterror,omitempty"`
}

type ListRunOutput struct {
	Runs    []domain.Run `json:"runs"`
	Total   int          `json:"total"`
	Message string       `json:"message"`
}

// CreateOperatorRequestInput is the DTO for creating a new operatorrequest.
type CreateOperatorRequestInput struct {
	PipelineID int    `json:"pipelineid" validate:"required,min=1"`
	Pointer    string `json:"pointer" validate:"required,min=1"`
	Question   string `json:"question" validate:"required,min=1"`
	Blocking   bool   `json:"blocking" validate:""`
	Status     string `json:"status" validate:"required,min=1"`
}

// Validate checks the CreateOperatorRequestInput DTO fields.
func (r *CreateOperatorRequestInput) Validate() error {
	if r.PipelineID < 0 {
		return errors.New("pipelineid must be a positive number")
	}
	if strings.TrimSpace(r.Pointer) == "" {
		return errors.New("pointer is required")
	}
	if strings.TrimSpace(r.Question) == "" {
		return errors.New("question is required")
	}
	if strings.TrimSpace(r.Status) == "" {
		return errors.New("status is required")
	}
	return nil
}

// CreateOperatorRequestOutput DTO for the creation response
type CreateOperatorRequestOutput struct {
	ID         uint   `json:"id"`
	PipelineID int    `json:"pipelineid"`
	Pointer    string `json:"pointer"`
	Question   string `json:"question"`
	Blocking   bool   `json:"blocking"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

type UpdateOperatorRequestInput struct {
	PipelineID *int    `json:"pipelineid,omitempty" validate:"omitempty,min=1"`
	Pointer    *string `json:"pointer,omitempty" validate:"omitempty,min=1"`
	Question   *string `json:"question,omitempty" validate:"omitempty,min=1"`
	Blocking   *bool   `json:"blocking,omitempty" validate:"omitempty"`
	Status     *string `json:"status,omitempty" validate:"omitempty,min=1"`
}

type ListOperatorRequestOutput struct {
	OperatorRequests []domain.OperatorRequest `json:"operatorrequests"`
	Total            int                      `json:"total"`
	Message          string                   `json:"message"`
}

// CreateTaskInput is the DTO for creating a new task.
type CreateTaskInput struct {
	PipelineID  int    `json:"pipelineid"`
	ParentID    int    `json:"parentid"`
	Owner       string `json:"owner"`
	TaskType    string `json:"tasktype"`
	Status      string `json:"status"`
	ContractRef string `json:"contractref"`
}

// CreateTaskOutput DTO for the creation response
type CreateTaskOutput struct {
	ID          uint   `json:"id"`
	PipelineID  int    `json:"pipelineid"`
	ParentID    int    `json:"parentid"`
	Owner       string `json:"owner"`
	TaskType    string `json:"tasktype"`
	Status      string `json:"status"`
	ContractRef string `json:"contractref"`
	Message     string `json:"message"`
}

type UpdateTaskInput struct {
	PipelineID  *int    `json:"pipelineid,omitempty"`
	ParentID    *int    `json:"parentid,omitempty"`
	Owner       *string `json:"owner,omitempty"`
	TaskType    *string `json:"tasktype,omitempty"`
	Status      *string `json:"status,omitempty"`
	ContractRef *string `json:"contractref,omitempty"`
}

type ListTaskOutput struct {
	Tasks   []domain.Task `json:"tasks"`
	Total   int           `json:"total"`
	Message string        `json:"message"`
}

// CreateModelCallInput is the DTO for creating a new modelcall.
type CreateModelCallInput struct {
	RunID     int     `json:"runid"`
	Profile   string  `json:"profile"`
	Model     string  `json:"model"`
	TokensIn  int     `json:"tokensin"`
	TokensOut int     `json:"tokensout"`
	Cost      float64 `json:"cost"`
	LatencyMs int     `json:"latencyms"`
}

// CreateModelCallOutput DTO for the creation response
type CreateModelCallOutput struct {
	ID        uint    `json:"id"`
	RunID     int     `json:"runid"`
	Profile   string  `json:"profile"`
	Model     string  `json:"model"`
	TokensIn  int     `json:"tokensin"`
	TokensOut int     `json:"tokensout"`
	Cost      float64 `json:"cost"`
	LatencyMs int     `json:"latencyms"`
	Message   string  `json:"message"`
}

type UpdateModelCallInput struct {
	RunID     *int     `json:"runid,omitempty"`
	Profile   *string  `json:"profile,omitempty"`
	Model     *string  `json:"model,omitempty"`
	TokensIn  *int     `json:"tokensin,omitempty"`
	TokensOut *int     `json:"tokensout,omitempty"`
	Cost      *float64 `json:"cost,omitempty"`
	LatencyMs *int     `json:"latencyms,omitempty"`
}

type ListModelCallOutput struct {
	ModelCalls []domain.ModelCall `json:"modelcalls"`
	Total      int                `json:"total"`
	Message    string             `json:"message"`
}

// CreateAuditEventInput is the DTO for creating a new auditevent.
type CreateAuditEventInput struct {
	PipelineID int       `json:"pipelineid"`
	RunID      int       `json:"runid"`
	EventType  string    `json:"eventtype"`
	CreatedAt  time.Time `json:"createdat"`
}

// CreateAuditEventOutput DTO for the creation response
type CreateAuditEventOutput struct {
	ID         uint      `json:"id"`
	PipelineID int       `json:"pipelineid"`
	RunID      int       `json:"runid"`
	EventType  string    `json:"eventtype"`
	CreatedAt  time.Time `json:"createdat"`
	Message    string    `json:"message"`
}

type UpdateAuditEventInput struct {
	PipelineID *int       `json:"pipelineid,omitempty"`
	RunID      *int       `json:"runid,omitempty"`
	EventType  *string    `json:"eventtype,omitempty"`
	CreatedAt  *time.Time `json:"createdat,omitempty"`
}

type ListAuditEventOutput struct {
	AuditEvents []domain.AuditEvent `json:"auditevents"`
	Total       int                 `json:"total"`
	Message     string              `json:"message"`
}
