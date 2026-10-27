// live_run drives one full greenfield pipeline 00→09 against the real model.
// Temporary utility for the refactor-plan verification; not part of the server.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/Reddetk/Adev/internal/di"
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/Reddetk/Adev/pkg/config"
	"github.com/Reddetk/Adev/pkg/database"
	"github.com/Reddetk/Adev/pkg/env"
	"github.com/Reddetk/Adev/pkg/logger"
)

const model = "~z-ai/glm-flash-latest"

var liveStages = []string{"00", "01", "02", "03", "04", "05", "06", "07", "08", "09"}

func main() {
	_ = godotenv.Load()
	logger.Init()
	cfg := config.Load()
	pool, err := database.Open(context.Background(), database.Config{
		DSN: cfg.GetDatabaseURL(),
	})
	if err != nil {
		fatal(fmt.Errorf("database: %w", err))
	}
	defer pool.Close()

	if len(os.Args) > 1 && os.Args[1] == "-probe" {
		// probe <targetSystem> <pipeline> <stage>: one stage run with the raw answer.
		container := di.NewContainer(pool)
		stage := os.Args[4]
		body, _ := json.Marshal(map[string]string{"pipelineId": os.Args[3]})
		path := "/api/v1/targetsystems/" + os.Args[2] + "/stages/" + stage + "/run"
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		request = mux.SetURLVars(request, map[string]string{"id": os.Args[2], "stage": stage})
		recorder := httptest.NewRecorder()
		container.StageHandler().Run(recorder, request)
		fmt.Println("HTTP:", recorder.Code)
		fmt.Println(strings.TrimSpace(recorder.Body.String()))
		dumpRequests(pool, os.Args[3])
		return
	}

	// resume <targetSystem> <pipeline>: drive the stages from the pipeline cursor
	// to 09 on an existing pipeline.
	if len(os.Args) > 1 && os.Args[1] == "-resume" {
		container := di.NewContainer(pool)
		resumeFrom(pool, container, os.Args[2], os.Args[3])
		return
	}

	fmt.Println("=== adev live run 00→09, model", model, "===")

	apiKey, err := env.ModelKey()
	if err != nil {
		fatal(err)
	}

	container := di.NewContainer(pool)
	ctx := context.Background()

	// 1. Target system.
	var targetSystem int64
	err = pool.QueryRow(ctx,
		`INSERT INTO target_systems (name, repo_url, description) VALUES ($1, $2, $3) RETURNING id`,
		"live-run "+time.Now().Format("15:04:05"), "https://example.invalid/live-run",
		"Temporary target system for the live 00→09 verification of "+time.Now().Format("2006-01-02 15:04"),
	).Scan(&targetSystem)
	if err != nil {
		fatal(fmt.Errorf("target system: %w", err))
	}
	fmt.Println("target system:", targetSystem)

	// 2. Workspace and agent configs, one per role.
	workspace := filepath.Join(os.TempDir(), "adev-live-run")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		fatal(err)
	}
	settings := container.SettingsService()
	if err := settings.SetWorkspace(ctx, targetSystem, workspace); err != nil {
		fatal(fmt.Errorf("workspace: %w", err))
	}
	for _, role := range []domain.Role{domain.RoleAnalyst, domain.RoleArchitect, domain.RoleQA, domain.RoleDev} {
		if err := settings.SetAgent(ctx, domain.AgentConfig{
			TargetSystemID: targetSystem, Role: role,
			Provider: domain.ProviderOpenRouter, Model: model, APIKey: apiKey,
		}); err != nil {
			fatal(fmt.Errorf("agent %s: %w", role, err))
		}
	}
	fmt.Println("agents configured (analyst/architect/qa/dev)")

	// A ten-second model probe so a rejected key fails before the long run.
	if _, err := container.ModelCheck().Check(ctx, int64(targetSystem), domain.RoleAnalyst); err != nil {
		fatal(fmt.Errorf("model probe: %w", err))
	}
	fmt.Println("model probe ok")

	// 3. Pipeline.
	planner := container.Planner()
	plan, err := planner.Plan(ctx, targetSystem, domain.RouteGreenfield)
	if err != nil {
		fatal(fmt.Errorf("plan: %w", err))
	}
	if !plan.Readiness.Ready {
		fatal(fmt.Errorf("the route is not ready: %v", plan.Readiness))
	}
	created, err := container.PipelineUseCase().CreatePipeline(usecase.CreatePipelineInput{
		TargetSystemID: int(targetSystem), Route: string(domain.RouteGreenfield),
		Status: "ready", CurrentStage: "00",
	})
	if err != nil {
		fatal(fmt.Errorf("create pipeline: %w", err))
	}
	pipelineID := strconv.FormatUint(uint64(created.ID), 10)
	fmt.Println("pipeline:", pipelineID)

	// 4. The USER_INPUT form the stage 00 reads.
	rawRequest := domain.Text{
		Value: "Build a web system for a small lending desk: the operator registers books, visitors reserve them and take them home.",
		Lang:  domain.LangSTE100,
	}
	input := domain.Form{Envelope: domain.Envelope{
		FormID:        fmt.Sprintf("FRM-%s-IN", pipelineID),
		FormType:      domain.TypeUserInput,
		SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusValid, Route: domain.RouteGreenfield, Stage: "00", Mode: domain.ModeFull,
		PipelineID: pipelineID, TargetSystemID: strconv.FormatInt(targetSystem, 10),
		ProducedBy: domain.RoleOperator, Inputs: []domain.Ref{},
	}}
	if err := input.SetPayload(domain.UserInput{
		RawRequest: rawRequest, Source: domain.SourceOperator,
		ReceivedAt: time.Now().Format(time.RFC3339), Locale: domain.LocaleEN,
	}); err != nil {
		fatal(err)
	}
	if err := container.FormService().Submit(ctx, &input); err != nil {
		fatal(fmt.Errorf("user input: %w", err))
	}

	// 5. Stages one by one; questions get automatic operator answers.
	runAllStages(pool, container, int(targetSystem), pipelineID)
}

// resumeFrom drives the remaining stages of an existing pipeline.
func resumeFrom(pool *pgxpool.Pool, container *di.Container, targetSystem, pipelineID string) {
	var cursor string
	if err := pool.QueryRow(context.Background(),
		`SELECT current_stage FROM pipelines WHERE id = $1`, pipelineID).Scan(&cursor); err != nil {
		fatal(fmt.Errorf("pipeline %s: %w", pipelineID, err))
	}
	fmt.Println("resuming pipeline", pipelineID, "at stage", cursor)
	started := false
	for _, stage := range liveStages {
		if !started {
			if stage != cursor {
				continue
			}
			started = true
		}
		runAllStages(pool, container, mustInt(targetSystem), pipelineID, stage)
	}
}

func mustInt(value string) int {
	parsed, _ := strconv.Atoi(value)
	return parsed
}

// runAllStages walks one stage per call; a blocked stage gets automatic operator
// answers and a rerun, up to 8 answer rounds.
func runAllStages(pool *pgxpool.Pool, container *di.Container, targetSystem int, pipelineID string, from ...string) {
	stages := liveStages
	if len(from) > 0 {
		started := false
		stages = nil
		for _, stage := range liveStages {
			if !started && stage != from[0] {
				continue
			}
			started = true
			stages = append(stages, stage)
		}
	}
	for _, stage := range stages {
		ok := runStageOnce(container, int64(targetSystem), pipelineID, stage)
		for round := 1; !ok && round <= 8; round++ {
			answered := answerOpenRequests(container, pool, pipelineID)
			if !answered {
				break
			}
			ok = runStageOnce(container, int64(targetSystem), pipelineID, stage)
		}
		if !ok {
			fatal(fmt.Errorf("stage %s stayed blocked", stage))
		}
	}

	fmt.Println("=== pipeline cursor reached the end ===")
	dumpDocuments(pool, pipelineID)
}

func fatal(err error) {
	fmt.Println("FAIL:", err)
	os.Exit(1)
}

// runStageOnce posts one stage run through the real handler and reports whether
// the stage came back unblocked.
func runStageOnce(container *di.Container, targetSystem int64, pipelineID, stage string) bool {
	body, _ := json.Marshal(map[string]string{"pipelineId": pipelineID})
	path := "/api/v1/targetsystems/" + strconv.FormatInt(targetSystem, 10) + "/stages/" + stage + "/run"
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	request = mux.SetURLVars(request, map[string]string{
		"id": strconv.FormatInt(targetSystem, 10), "stage": stage,
	})
	recorder := httptest.NewRecorder()
	container.StageHandler().Run(recorder, request)

	if recorder.Code != http.StatusOK {
		fmt.Printf("stage %s: HTTP %d: %s\n", stage, recorder.Code, recorder.Body.String())
		return false
	}
	var outcome usecase.StageOutcome
	if err := json.Unmarshal(recorder.Body.Bytes(), &outcome); err != nil {
		fmt.Printf("stage %s: bad outcome: %v: %s\n", stage, err, recorder.Body.String())
		return false
	}
	fmt.Printf("stage run: requests %s | attempts=%d form=%s\n", outcome.Stage, outcome.Attempts, outcome.FormID)
	if outcome.Blocked {
		for _, gap := range outcome.Gaps {
			fmt.Printf("gap: %s\n", gap)
		}
	}
	return !outcome.Blocked
}

// answerOpenRequests answers every active question of the pipeline: the first
// option when the question carries some, otherwise a plain acknowledgment.
func answerOpenRequests(container *di.Container, pool *pgxpool.Pool, pipelineID string) bool {
	var answered bool
	id, _ := strconv.Atoi(pipelineID)
	rows, err := pool.Query(context.Background(), `SELECT id, pointer, options FROM operator_requests
		WHERE pipeline_id = $1 AND status = 'active' ORDER BY id`, id)
	if err != nil {
		fatal(fmt.Errorf("open questions: %w", err))
	}
	type open struct {
		id      int
		pointer string
		options []string
	}
	var pending []open
	for rows.Next() {
		var o open
		if err := rows.Scan(&o.id, &o.pointer, &o.options); err != nil {
			fatal(err)
		}
		pending = append(pending, o)
	}
	rows.Close()

	for _, o := range pending {
		value := answerByPointer(o.pointer, o.options)
		fmt.Printf("answer #%d %s <- %s\n", o.id, o.pointer, string(value))
		body, _ := json.Marshal(map[string]any{"pipelineId": id, "pointer": o.pointer, "value": json.RawMessage(value)})
		request := httptest.NewRequest(http.MethodPost, "/api/v1/operatorrequests/answer", bytes.NewReader(body))
		recorder := httptest.NewRecorder()
		container.QuestionHandler().Answer(recorder, request)
		if recorder.Code != http.StatusOK {
			fmt.Printf("answer failed: HTTP %d: %s\n", recorder.Code, recorder.Body.String())
		}
	}
	if !answered {
		fmt.Println("no open answers: the stage stays blocked without questions")
	}
	return answered
}

// answerByPointer is the canned operator for the demo domain: a list question
// gets a real entry in the shape the field expects, an enum question picks the
// first option, anything else gets a plain acknowledgment.
func answerByPointer(pointer string, options []string) []byte {
	text := `"The operator accepted a reasonable default for now."`
	ste := func(t string) map[string]string { return map[string]string{"text": t, "lang": "STE100"} }
	switch {
	case strings.HasSuffix(pointer, "/goals"):
		value, _ := json.Marshal([]map[string]any{{
			"id": "GOAL-1", "outcome": ste("The desk registers a book in under one minute."), "metricId": "M-1",
		}})
		return value
	case strings.HasSuffix(pointer, "/metrics"):
		value, _ := json.Marshal([]map[string]any{{
			"id": "M-1", "name": "register_seconds", "unit": "second", "baseline": 90.0, "target": 20.0,
		}})
		return value
	case strings.HasSuffix(pointer, "/successMetrics"):
		value, _ := json.Marshal([]map[string]any{{
			"metricId": "M-1", "threshold": "20", "window": "30 day",
		}})
		return value
	case strings.HasSuffix(pointer, "/actors"):
		value, _ := json.Marshal([]map[string]any{{
			"id": "OP-1", "name": "desk operator", "tasks": []any{ste("Register a book.")},
			"frequency": "day", "businessObject": "book",
		}})
		return value
	case strings.HasSuffix(pointer, "/constraints"):
		value, _ := json.Marshal([]map[string]any{{
			"id": "C-1", "text": "A reservation expires after three days.", "modality": "shall", "priority": "must",
		}})
		return value
	case strings.Contains(pointer, "/actors/") && strings.HasSuffix(pointer, "/tasks"):
		value, _ := json.Marshal([]any{ste("Register a book.")})
		return value
	case strings.Contains(pointer, "/scope/"):
		value, _ := json.Marshal([]any{ste("Book registration.")})
		return value
	case strings.HasSuffix(pointer, "/assumptions"):
		value, _ := json.Marshal([]any{ste("The library holds fewer than ten thousand books.")})
		return value
	case strings.Contains(pointer, "/baseline"):
		return []byte(`90`)
	case strings.Contains(pointer, "/target"):
		return []byte(`20`)
	case strings.HasSuffix(pointer, "cost"):
		value, _ := json.Marshal(map[string]any{"currency": "EUR", "amount": 1000})
		return value
	case strings.HasSuffix(pointer, "scheduleDays"):
		return []byte(`30`)
	case strings.HasSuffix(pointer, "confidence"):
		return []byte(`"high"`)
	case strings.HasSuffix(pointer, "recommendedOption"):
		return []byte(`"OPT-1"`)
	case strings.HasSuffix(pointer, "verdict"):
		return []byte(`"go"`)
	}
	if len(options) > 0 {
		if json.Valid([]byte(options[0])) {
			return []byte(options[0])
		}
		quoted, _ := json.Marshal(options[0])
		return quoted
	}
	return []byte(text)
}

func dumpRequests(pool *pgxpool.Pool, pipelineID string) {
	id, _ := strconv.Atoi(pipelineID)
	rows, err := pool.Query(context.Background(),
		`SELECT id, pointer, question, status FROM operator_requests WHERE pipeline_id = $1 ORDER BY id`, id)
	if err != nil {
		fmt.Println("requests dump failed:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var n int
		var pointer, question, status string
		if err := rows.Scan(&n, &pointer, &question, &status); err != nil {
			fmt.Println("scan:", err)
			return
		}
		fmt.Printf("request #%d %s %q %s\n", n, pointer, question, status)
	}
}

func dumpDocuments(pool *pgxpool.Pool, pipelineID string) {
	rows, err := pool.Query(context.Background(),
		`SELECT form_id, form_type, form_version, status FROM forms
		 WHERE pipeline_id = $1 ORDER BY stage, form_version`, pipelineID)
	if err != nil {
		fmt.Println("document dump failed:", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, kind, status string
		var version int
		if err := rows.Scan(&id, &kind, &version, &status); err != nil {
			fmt.Println("scan:", err)
			return
		}
		fmt.Printf("document %s v%d %s %s\n", id, version, kind, status)
	}
}
