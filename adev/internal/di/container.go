package di

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/handler/http"
	"github.com/Reddetk/Adev/internal/repository"
	"github.com/Reddetk/Adev/internal/usecase"
)

type Container struct {
	pool *pgxpool.Pool

	// Repositories
	targetsystemRepo    usecase.TargetSystemRepository
	pipelineRepo        usecase.PipelineRepository
	runRepo             usecase.RunRepository
	operatorrequestRepo usecase.OperatorRequestRepository
	taskRepo            usecase.TaskRepository
	modelcallRepo       usecase.ModelCallRepository
	auditeventRepo      usecase.AuditEventRepository

	// Use Cases
	targetsystemUC    usecase.TargetSystemUseCase
	pipelineUC        usecase.PipelineUseCase
	runUC             usecase.RunUseCase
	operatorrequestUC usecase.OperatorRequestUseCase
	taskUC            usecase.TaskUseCase
	modelcallUC       usecase.ModelCallUseCase
	auditeventUC      usecase.AuditEventUseCase

	// Handlers
	targetSystemHandler    *http.TargetSystemHandler
	pipelineHandler        *http.PipelineHandler
	runHandler             *http.RunHandler
	operatorRequestHandler *http.OperatorRequestHandler
	taskHandler            *http.TaskHandler
	modelCallHandler       *http.ModelCallHandler
	auditEventHandler      *http.AuditEventHandler

	// adev services
	formService     *usecase.FormService
	settingsService *usecase.SettingsService
	pipelineService *usecase.Planner
	activityService *usecase.ActivityService
	stageService    *usecase.StageService
	modelCheck      *usecase.ModelCheck
	formHandler     *http.FormHandler
	stageHandler    *http.StageHandler
	questionHandler *http.QuestionHandler
}

func NewContainer(pool *pgxpool.Pool) *Container {
	c := &Container{pool: pool}
	c.setupRepositories()
	c.setupUseCases()
	c.setupHandlers()
	c.setupAdEvServices()
	return c
}

// setupAdEvServices wires the adev services on the pgx stores.
func (c *Container) setupAdEvServices() {
	formsStore := repository.NewFormsStore(c.pool)
	settingsStore := repository.NewPostgresAgentConfigStore(c.pool)
	c.formService = usecase.NewFormService(formsStore, nil)
	c.settingsService = usecase.NewSettingsService(settingsStore)
	c.pipelineService = usecase.NewPlanner(c.settingsService)
	c.formHandler = http.NewFormHandler(c.formService)

	// Stage runs need a model client. The run recorder stays nil until the run table
	// takes the string pipeline id of the forms.
	factory := repository.NewOpenAICompatibleFactory()
	questions := usecase.NewQuestionService(c.formService, c.operatorrequestRepo)
	activities := usecase.NewActivityService()
	c.activityService = activities
	c.stageService = usecase.NewStageService(c.formService, c.settingsService, factory, nil).WithQuestions(questions).WithActivity(activities)
	c.stageHandler = http.NewStageHandler(c.stageOrchestrator())
	c.questionHandler = http.NewQuestionHandler(questions)
	c.modelCheck = usecase.NewModelCheck(c.settingsService, factory)
}

// stageOrchestrator wraps the stage service: a successful run moves the pipeline
// cursor to the next stage of the registry route.
func (c *Container) stageOrchestrator() *usecase.Orchestrator {
	return usecase.NewOrchestrator(c.stageService, c.pipelineRepo)
}

func (c *Container) setupRepositories() {
	c.targetsystemRepo = repository.NewPostgresTargetSystemRepository(c.pool)
	c.pipelineRepo = repository.NewPostgresPipelineRepository(c.pool)
	c.runRepo = repository.NewPostgresRunRepository(c.pool)
	c.operatorrequestRepo = repository.NewPostgresOperatorRequestRepository(c.pool)
	c.taskRepo = repository.NewPostgresTaskRepository(c.pool)
	c.modelcallRepo = repository.NewPostgresModelCallRepository(c.pool)
	c.auditeventRepo = repository.NewPostgresAuditEventRepository(c.pool)
}

func (c *Container) setupUseCases() {
	c.targetsystemUC = usecase.NewTargetSystemService(c.targetsystemRepo)
	c.pipelineUC = usecase.NewPipelineService(c.pipelineRepo)
	c.runUC = usecase.NewRunService(c.runRepo)
	c.operatorrequestUC = usecase.NewOperatorRequestService(c.operatorrequestRepo)
	c.taskUC = usecase.NewTaskService(c.taskRepo)
	c.modelcallUC = usecase.NewModelCallService(c.modelcallRepo)
	c.auditeventUC = usecase.NewAuditEventService(c.auditeventRepo)
}

func (c *Container) setupHandlers() {
	c.targetSystemHandler = http.NewTargetSystemHandler(c.targetsystemUC)
	c.pipelineHandler = http.NewPipelineHandler(c.pipelineUC)
	c.runHandler = http.NewRunHandler(c.runUC)
	c.operatorRequestHandler = http.NewOperatorRequestHandler(c.operatorrequestUC)
	c.taskHandler = http.NewTaskHandler(c.taskUC)
	c.modelCallHandler = http.NewModelCallHandler(c.modelcallUC)
	c.auditEventHandler = http.NewAuditEventHandler(c.auditeventUC)
}

// Getters
func (c *Container) TargetSystemHandler() *http.TargetSystemHandler {
	return c.targetSystemHandler
}

func (c *Container) TargetSystemUseCase() usecase.TargetSystemUseCase {
	return c.targetsystemUC
}

func (c *Container) TargetSystemRepository() usecase.TargetSystemRepository {
	return c.targetsystemRepo
}
func (c *Container) PipelineHandler() *http.PipelineHandler {
	return c.pipelineHandler
}

func (c *Container) PipelineUseCase() usecase.PipelineUseCase {
	return c.pipelineUC
}

func (c *Container) PipelineRepository() usecase.PipelineRepository {
	return c.pipelineRepo
}

func (c *Container) RunHandler() *http.RunHandler {
	return c.runHandler
}

func (c *Container) RunUseCase() usecase.RunUseCase {
	return c.runUC
}

func (c *Container) RunRepository() usecase.RunRepository {
	return c.runRepo
}

func (c *Container) OperatorRequestHandler() *http.OperatorRequestHandler {
	return c.operatorRequestHandler
}

func (c *Container) OperatorRequestUseCase() usecase.OperatorRequestUseCase {
	return c.operatorrequestUC
}

func (c *Container) OperatorRequestRepository() usecase.OperatorRequestRepository {
	return c.operatorrequestRepo
}

func (c *Container) TaskHandler() *http.TaskHandler {
	return c.taskHandler
}

func (c *Container) TaskUseCase() usecase.TaskUseCase {
	return c.taskUC
}

func (c *Container) TaskRepository() usecase.TaskRepository {
	return c.taskRepo
}

func (c *Container) ModelCallHandler() *http.ModelCallHandler {
	return c.modelCallHandler
}

func (c *Container) ModelCallUseCase() usecase.ModelCallUseCase {
	return c.modelcallUC
}

func (c *Container) ModelCallRepository() usecase.ModelCallRepository {
	return c.modelcallRepo
}

func (c *Container) AuditEventHandler() *http.AuditEventHandler {
	return c.auditEventHandler
}

func (c *Container) AuditEventUseCase() usecase.AuditEventUseCase {
	return c.auditeventUC
}

func (c *Container) AuditEventRepository() usecase.AuditEventRepository {
	return c.auditeventRepo
}

// FormService returns the form service.
func (c *Container) FormService() *usecase.FormService { return c.formService }

// FormHandler returns the form handler.
func (c *Container) FormHandler() *http.FormHandler { return c.formHandler }

// StageHandler returns the stage handler.
func (c *Container) StageHandler() *http.StageHandler { return c.stageHandler }

// QuestionHandler returns the question handler.
func (c *Container) QuestionHandler() *http.QuestionHandler { return c.questionHandler }

// ActivityService is the live book of stage runs for the operator dashboard.
func (c *Container) ActivityService() *usecase.ActivityService { return c.activityService }

// ModelCheck returns the model checker.
func (c *Container) ModelCheck() *usecase.ModelCheck { return c.modelCheck }

// SettingsService returns the target system settings service.
func (c *Container) SettingsService() *usecase.SettingsService { return c.settingsService }

// Planner returns the pipeline planning service.
func (c *Container) Planner() *usecase.Planner { return c.pipelineService }
