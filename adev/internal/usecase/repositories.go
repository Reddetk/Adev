package usecase

import "github.com/Reddetk/Adev/internal/domain"

// Persistence ports of the control-plane entities.
// The repository layer implements them; this layer only knows the contracts.

type TargetSystemRepository interface {
	Save(targetsystem *domain.TargetSystem) error
	FindByID(id int) (*domain.TargetSystem, error)
	FindByName(name string) (*domain.TargetSystem, error)
	FindByRepoURL(repourl string) (*domain.TargetSystem, error)
	FindByDescription(description string) (*domain.TargetSystem, error)
	Update(targetsystem *domain.TargetSystem) error
	Delete(id int) error
	FindAll() ([]domain.TargetSystem, error)
}

type PipelineRepository interface {
	Save(pipeline *domain.Pipeline) error
	FindByID(id int) (*domain.Pipeline, error)
	FindByTargetSystemID(targetsystemid int) (*domain.Pipeline, error)
	FindByRoute(route string) (*domain.Pipeline, error)
	FindByStatus(status string) (*domain.Pipeline, error)
	FindByCurrentStage(currentstage string) (*domain.Pipeline, error)
	Update(pipeline *domain.Pipeline) error
	Delete(id int) error
	FindAll() ([]domain.Pipeline, error)
}

type RunRepository interface {
	Save(run *domain.Run) error
	FindByID(id int) (*domain.Run, error)
	FindByPipelineID(pipelineid int) (*domain.Run, error)
	FindByStage(stage string) (*domain.Run, error)
	FindByStatus(status string) (*domain.Run, error)
	FindByCheckpointID(checkpointid string) (*domain.Run, error)
	FindByAttempts(attempts int) (*domain.Run, error)
	FindByLastError(lasterror string) (*domain.Run, error)
	Update(run *domain.Run) error
	Delete(id int) error
	FindAll() ([]domain.Run, error)
}

type OperatorRequestRepository interface {
	Save(operatorrequest *domain.OperatorRequest) error
	FindByID(id int) (*domain.OperatorRequest, error)
	FindByPipelineID(pipelineid int) (*domain.OperatorRequest, error)
	FindByPointer(pointer string) (*domain.OperatorRequest, error)
	FindByQuestion(question string) (*domain.OperatorRequest, error)
	FindByStatus(status string) (*domain.OperatorRequest, error)
	Update(operatorrequest *domain.OperatorRequest) error
	Delete(id int) error
	FindAll() ([]domain.OperatorRequest, error)
}

type TaskRepository interface {
	Save(task *domain.Task) error
	FindByID(id int) (*domain.Task, error)
	FindByPipelineID(pipelineid int) (*domain.Task, error)
	FindByParentID(parentid int) (*domain.Task, error)
	FindByOwner(owner string) (*domain.Task, error)
	FindByTaskType(tasktype string) (*domain.Task, error)
	FindByStatus(status string) (*domain.Task, error)
	FindByContractRef(contractref string) (*domain.Task, error)
	Update(task *domain.Task) error
	Delete(id int) error
	FindAll() ([]domain.Task, error)
}

type ModelCallRepository interface {
	Save(modelcall *domain.ModelCall) error
	FindByID(id int) (*domain.ModelCall, error)
	FindByRunID(runid int) (*domain.ModelCall, error)
	FindByProfile(profile string) (*domain.ModelCall, error)
	FindByModel(model string) (*domain.ModelCall, error)
	FindByTokensIn(tokensin int) (*domain.ModelCall, error)
	FindByTokensOut(tokensout int) (*domain.ModelCall, error)
	FindByLatencyMs(latencyms int) (*domain.ModelCall, error)
	Update(modelcall *domain.ModelCall) error
	Delete(id int) error
	FindAll() ([]domain.ModelCall, error)
}

type AuditEventRepository interface {
	Save(auditevent *domain.AuditEvent) error
	FindByID(id int) (*domain.AuditEvent, error)
	FindByPipelineID(pipelineid int) (*domain.AuditEvent, error)
	FindByRunID(runid int) (*domain.AuditEvent, error)
	FindByEventType(eventtype string) (*domain.AuditEvent, error)
	Update(auditevent *domain.AuditEvent) error
	Delete(id int) error
	FindAll() ([]domain.AuditEvent, error)
}
