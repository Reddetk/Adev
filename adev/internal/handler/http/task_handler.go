package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Reddetk/Adev/internal/usecase"
	"github.com/gorilla/mux"
)

type TaskHandler struct {
	usecase usecase.TaskUseCase
}

func NewTaskHandler(uc usecase.TaskUseCase) *TaskHandler {
	return &TaskHandler{usecase: uc}
}

// Create task godoc
// @Summary Create task
// @Tags tasks
// @Accept json
// @Produce json
// @Param body body usecase.CreateTaskInput true "Task payload"
// @Success 201 {object} usecase.CreateTaskOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tasks [post]
func (t *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateTaskInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	output, err := t.usecase.CreateTask(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(output)
}

// Get task by ID godoc
// @Summary Get task by ID
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Success 200 {object} domain.Task
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tasks/{id} [get]
func (t *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	task, err := t.usecase.GetTask(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// Update task godoc
// @Summary Update task
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Param body body usecase.UpdateTaskInput true "Task payload"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tasks/{id} [put]
func (t *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	var input usecase.UpdateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := t.usecase.UpdateTask(id, input); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete task godoc
// @Summary Delete task
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path int true "Task ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tasks/{id} [delete]
func (t *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	if err := t.usecase.DeleteTask(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List tasks godoc
// @Summary List tasks
// @Tags tasks
// @Accept json
// @Produce json
// @Success 200 {object} usecase.ListTaskOutput
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tasks [get]
func (t *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	output, err := t.usecase.ListTasks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}
