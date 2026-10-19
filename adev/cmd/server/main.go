package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Reddetk/Adev/internal/di"
	apphttp "github.com/Reddetk/Adev/internal/handler/http"
	"github.com/Reddetk/Adev/pkg/config"
	"github.com/Reddetk/Adev/pkg/database"
	"github.com/Reddetk/Adev/pkg/logger"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/Reddetk/Adev/docs"
)

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Version   string            `json:"version"`
}

var (
	// Build information (set by build flags)
	Version   = "dev"
	BuildTime = "unknown"
	pool      *pgxpool.Pool
)

// @title adev API
// @version 1.0
// @description Control plane of adev: target systems, pipelines, forms, runs, operator requests.
// @BasePath /api/v1
func main() {
	showVersion := flag.Bool("version", false, "print the build version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("adev %s (built %s)\n", Version, BuildTime)
		return
	}

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init()

	log.Printf("Starting application v%s (built: %s)", Version, BuildTime)
	log.Printf("Environment: %s", cfg.Environment)

	// Connect to database with retry
	var err error
	pool, err = connectToDatabase(cfg)
	if err != nil {
		log.Printf("Warning: Database connection failed: %v", err)
		log.Printf("Server will start in degraded mode. Check your database configuration.")
		log.Printf("Tip: Configure database environment variables in .env file")
		log.Printf("Tip: Apply the schema with: make migrate-up")
		pool = nil // Ensure pool is nil for health checks
	} else {
		log.Printf("Database connected successfully")
		log.Printf("Apply the schema with: make migrate-up")
	}

	// Setup router
	router := mux.NewRouter()

	// Dependency injection container
	container := di.NewContainer(pool)
	_ = container

	// API v1 routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apphttp.SetupTargetSystemRoutes(apiRouter, container.TargetSystemUseCase())       // targetsystem routes
	apphttp.SetupPipelineRoutes(apiRouter, container.PipelineUseCase())               // pipeline routes
	apphttp.SetupRunRoutes(apiRouter, container.RunUseCase())                         // run routes
	apphttp.SetupOperatorRequestRoutes(apiRouter, container.OperatorRequestUseCase()) // operatorrequest routes
	apphttp.SetupTaskRoutes(apiRouter, container.TaskUseCase())                       // task routes
	apphttp.SetupModelCallRoutes(apiRouter, container.ModelCallUseCase())             // modelcall routes
	apphttp.SetupAuditEventRoutes(apiRouter, container.AuditEventUseCase())           // auditevent routes

	// adev settings and pipeline plan
	apphttp.SetupSettingsRoutes(apiRouter, apphttp.NewSettingsHandler(container.SettingsService(), container.Planner(), container.PipelineUseCase()))
	// adev form contract
	apphttp.SetupFormRoutes(apiRouter, container.FormHandler())
	// adev stage runs
	apphttp.SetupStageRoutes(apiRouter, container.StageHandler())

	// Minimal frontend: target system setup and the first stage run.
	router.PathPrefix("/app/").Handler(http.StripPrefix("/app/", http.FileServer(http.Dir("frontend"))))
	// goca:routes -- feature routes are registered above this line

	// Swagger UI at /swagger/index.html
	router.PathPrefix("/swagger").Handler(httpSwagger.WrapHandler)

	// Health check endpoint with comprehensive checks
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")
	router.HandleFunc("/health/ready", readinessHandler).Methods("GET")
	router.HandleFunc("/health/live", livenessHandler).Methods("GET")

	// AuditEvent routes
	auditeventHandler := container.AuditEventHandler()
	router.HandleFunc("/api/v1/auditevents", auditeventHandler.CreateAuditEvent).Methods("POST")
	router.HandleFunc("/api/v1/auditevents/{id}", auditeventHandler.GetAuditEvent).Methods("GET")
	router.HandleFunc("/api/v1/auditevents/{id}", auditeventHandler.UpdateAuditEvent).Methods("PUT")
	router.HandleFunc("/api/v1/auditevents/{id}", auditeventHandler.DeleteAuditEvent).Methods("DELETE")
	router.HandleFunc("/api/v1/auditevents", auditeventHandler.ListAuditEvents).Methods("GET")

	// ModelCall routes
	modelcallHandler := container.ModelCallHandler()
	router.HandleFunc("/api/v1/modelcalls", modelcallHandler.CreateModelCall).Methods("POST")
	router.HandleFunc("/api/v1/modelcalls/{id}", modelcallHandler.GetModelCall).Methods("GET")
	router.HandleFunc("/api/v1/modelcalls/{id}", modelcallHandler.UpdateModelCall).Methods("PUT")
	router.HandleFunc("/api/v1/modelcalls/{id}", modelcallHandler.DeleteModelCall).Methods("DELETE")
	router.HandleFunc("/api/v1/modelcalls", modelcallHandler.ListModelCalls).Methods("GET")

	// OperatorRequest routes
	operatorrequestHandler := container.OperatorRequestHandler()
	router.HandleFunc("/api/v1/operatorrequests", operatorrequestHandler.CreateOperatorRequest).Methods("POST")
	router.HandleFunc("/api/v1/operatorrequests/{id}", operatorrequestHandler.GetOperatorRequest).Methods("GET")
	router.HandleFunc("/api/v1/operatorrequests/{id}", operatorrequestHandler.UpdateOperatorRequest).Methods("PUT")
	router.HandleFunc("/api/v1/operatorrequests/{id}", operatorrequestHandler.DeleteOperatorRequest).Methods("DELETE")
	router.HandleFunc("/api/v1/operatorrequests", operatorrequestHandler.ListOperatorRequests).Methods("GET")

	// Pipeline routes
	pipelineHandler := container.PipelineHandler()
	router.HandleFunc("/api/v1/pipelines", pipelineHandler.CreatePipeline).Methods("POST")
	router.HandleFunc("/api/v1/pipelines/{id}", pipelineHandler.GetPipeline).Methods("GET")
	router.HandleFunc("/api/v1/pipelines/{id}", pipelineHandler.UpdatePipeline).Methods("PUT")
	router.HandleFunc("/api/v1/pipelines/{id}", pipelineHandler.DeletePipeline).Methods("DELETE")
	router.HandleFunc("/api/v1/pipelines", pipelineHandler.ListPipelines).Methods("GET")

	// Run routes
	runHandler := container.RunHandler()
	router.HandleFunc("/api/v1/runs", runHandler.CreateRun).Methods("POST")
	router.HandleFunc("/api/v1/runs/{id}", runHandler.GetRun).Methods("GET")
	router.HandleFunc("/api/v1/runs/{id}", runHandler.UpdateRun).Methods("PUT")
	router.HandleFunc("/api/v1/runs/{id}", runHandler.DeleteRun).Methods("DELETE")
	router.HandleFunc("/api/v1/runs", runHandler.ListRuns).Methods("GET")

	// TargetSystem routes
	targetsystemHandler := container.TargetSystemHandler()
	router.HandleFunc("/api/v1/targetsystems", targetsystemHandler.CreateTargetSystem).Methods("POST")
	router.HandleFunc("/api/v1/targetsystems/{id}", targetsystemHandler.GetTargetSystem).Methods("GET")
	router.HandleFunc("/api/v1/targetsystems/{id}", targetsystemHandler.UpdateTargetSystem).Methods("PUT")
	router.HandleFunc("/api/v1/targetsystems/{id}", targetsystemHandler.DeleteTargetSystem).Methods("DELETE")
	router.HandleFunc("/api/v1/targetsystems", targetsystemHandler.ListTargetSystems).Methods("GET")

	// Task routes
	taskHandler := container.TaskHandler()
	router.HandleFunc("/api/v1/tasks", taskHandler.CreateTask).Methods("POST")
	router.HandleFunc("/api/v1/tasks/{id}", taskHandler.GetTask).Methods("GET")
	router.HandleFunc("/api/v1/tasks/{id}", taskHandler.UpdateTask).Methods("PUT")
	router.HandleFunc("/api/v1/tasks/{id}", taskHandler.DeleteTask).Methods("DELETE")
	router.HandleFunc("/api/v1/tasks", taskHandler.ListTasks).Methods("GET")

	// Setup HTTP server with timeouts
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	if pool != nil {
		pool.Close()
	}

	log.Println("Server exited")
}

func connectToDatabase(cfg *config.Config) (*pgxpool.Pool, error) {
	dsn := cfg.GetDatabaseURL()

	log.Printf("Connecting to database at %s:%s/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)

	// Check if this is development mode without database
	if cfg.Environment == "development" && cfg.Database.Password == "" {
		log.Println("Warning: Development mode detected: No database password set")
		log.Println("To connect to database, set environment variables:")
		log.Println("   DB_HOST=localhost")
		log.Println("   DB_PORT=<port>")
		log.Println("   DB_USER=<user>")
		log.Println("   DB_PASSWORD=your_password")
		log.Println("   DB_NAME=your_database")
		log.Println("Server will continue without database connection...")
		return nil, fmt.Errorf("development mode: database not configured")
	}

	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err := database.Open(ctx, database.Config{
			DSN:         dsn,
			MaxConns:    int32(cfg.Database.MaxOpenConns),
			MaxIdleTime: cfg.Database.MaxLifetime,
			MaxLifetime: cfg.Database.MaxLifetime,
		})
		cancel()

		if err == nil {
			return pool, nil
		}

		lastErr = err
		log.Printf("Attempt %d: Database connection failed: %v", attempt, err)
		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to database after 5 attempts: %w", lastErr)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  make(map[string]string),
		Version:   Version,
	}

	// Check database
	if err := checkDatabase(); err != nil {
		status.Status = "degraded"
		status.Services["database"] = fmt.Sprintf("error: %v", err)
		// Don't fail the whole health check for database issues in development
		log.Printf("Database health check failed: %v", err)
	} else {
		status.Services["database"] = "healthy"
	}

	// Always return 200 for basic health check - let readiness handle critical dependencies
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	// Check if all dependencies are ready
	if err := checkDatabase(); err != nil {
		http.Error(w, fmt.Sprintf("Database not ready: %v", err), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	// Basic liveness check
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Alive"))
}

func checkDatabase() error {
	if pool == nil {
		return fmt.Errorf("database connection is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return pool.Ping(ctx)
}
