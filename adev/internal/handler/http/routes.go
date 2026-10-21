package http

import (
	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/gorilla/mux"
)

func SetupTargetSystemRoutes(router *mux.Router, uc usecase.TargetSystemUseCase) {
	handler := NewTargetSystemHandler(uc)

	targetsystemRouter := router.PathPrefix("/targetsystems").Subrouter()

	targetsystemRouter.HandleFunc("", handler.CreateTargetSystem).Methods("POST")
	targetsystemRouter.HandleFunc("/{id}", handler.GetTargetSystem).Methods("GET")
	targetsystemRouter.HandleFunc("/{id}", handler.UpdateTargetSystem).Methods("PUT")
	targetsystemRouter.HandleFunc("/{id}", handler.DeleteTargetSystem).Methods("DELETE")
	targetsystemRouter.HandleFunc("", handler.ListTargetSystems).Methods("GET")
}

func SetupPipelineRoutes(router *mux.Router, uc usecase.PipelineUseCase) {
	handler := NewPipelineHandler(uc)

	pipelineRouter := router.PathPrefix("/pipelines").Subrouter()

	pipelineRouter.HandleFunc("", handler.CreatePipeline).Methods("POST")
	pipelineRouter.HandleFunc("/{id}", handler.GetPipeline).Methods("GET")
	pipelineRouter.HandleFunc("/{id}", handler.UpdatePipeline).Methods("PUT")
	pipelineRouter.HandleFunc("/{id}", handler.DeletePipeline).Methods("DELETE")
	pipelineRouter.HandleFunc("", handler.ListPipelines).Methods("GET")
}

func SetupRunRoutes(router *mux.Router, uc usecase.RunUseCase) {
	handler := NewRunHandler(uc)

	runRouter := router.PathPrefix("/runs").Subrouter()

	runRouter.HandleFunc("", handler.CreateRun).Methods("POST")
	runRouter.HandleFunc("/{id}", handler.GetRun).Methods("GET")
	runRouter.HandleFunc("/{id}", handler.UpdateRun).Methods("PUT")
	runRouter.HandleFunc("/{id}", handler.DeleteRun).Methods("DELETE")
	runRouter.HandleFunc("", handler.ListRuns).Methods("GET")
}

func SetupOperatorRequestRoutes(router *mux.Router, uc usecase.OperatorRequestUseCase) {
	handler := NewOperatorRequestHandler(uc)

	operatorrequestRouter := router.PathPrefix("/operatorrequests").Subrouter()

	operatorrequestRouter.HandleFunc("", handler.CreateOperatorRequest).Methods("POST")
	operatorrequestRouter.HandleFunc("/{id}", handler.GetOperatorRequest).Methods("GET")
	operatorrequestRouter.HandleFunc("/{id}", handler.UpdateOperatorRequest).Methods("PUT")
	operatorrequestRouter.HandleFunc("/{id}", handler.DeleteOperatorRequest).Methods("DELETE")
	operatorrequestRouter.HandleFunc("", handler.ListOperatorRequests).Methods("GET")
}

func SetupTaskRoutes(router *mux.Router, uc usecase.TaskUseCase) {
	handler := NewTaskHandler(uc)

	taskRouter := router.PathPrefix("/tasks").Subrouter()

	taskRouter.HandleFunc("", handler.CreateTask).Methods("POST")
	taskRouter.HandleFunc("/{id}", handler.GetTask).Methods("GET")
	taskRouter.HandleFunc("/{id}", handler.UpdateTask).Methods("PUT")
	taskRouter.HandleFunc("/{id}", handler.DeleteTask).Methods("DELETE")
	taskRouter.HandleFunc("", handler.ListTasks).Methods("GET")
}

func SetupModelCallRoutes(router *mux.Router, uc usecase.ModelCallUseCase) {
	handler := NewModelCallHandler(uc)

	modelcallRouter := router.PathPrefix("/modelcalls").Subrouter()

	modelcallRouter.HandleFunc("", handler.CreateModelCall).Methods("POST")
	modelcallRouter.HandleFunc("/{id}", handler.GetModelCall).Methods("GET")
	modelcallRouter.HandleFunc("/{id}", handler.UpdateModelCall).Methods("PUT")
	modelcallRouter.HandleFunc("/{id}", handler.DeleteModelCall).Methods("DELETE")
	modelcallRouter.HandleFunc("", handler.ListModelCalls).Methods("GET")
}

func SetupAuditEventRoutes(router *mux.Router, uc usecase.AuditEventUseCase) {
	handler := NewAuditEventHandler(uc)

	auditeventRouter := router.PathPrefix("/auditevents").Subrouter()

	auditeventRouter.HandleFunc("", handler.CreateAuditEvent).Methods("POST")
	auditeventRouter.HandleFunc("/{id}", handler.GetAuditEvent).Methods("GET")
	auditeventRouter.HandleFunc("/{id}", handler.UpdateAuditEvent).Methods("PUT")
	auditeventRouter.HandleFunc("/{id}", handler.DeleteAuditEvent).Methods("DELETE")
	auditeventRouter.HandleFunc("", handler.ListAuditEvents).Methods("GET")
}
