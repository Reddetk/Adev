package usecase

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/messages"
)

type taskService struct {
	repo TaskRepository
}

func NewTaskService(repo TaskRepository) TaskUseCase {
	return &taskService{repo: repo}
}

func (t *taskService) CreateTask(input CreateTaskInput) (CreateTaskOutput, error) {
	task := domain.Task{
		PipelineID:  input.PipelineID,
		ParentID:    input.ParentID,
		Owner:       input.Owner,
		TaskType:    input.TaskType,
		Status:      input.Status,
		ContractRef: input.ContractRef,
	}

	if err := task.Validate(); err != nil {
		return CreateTaskOutput{}, err
	}

	if err := t.repo.Save(&task); err != nil {
		return CreateTaskOutput{}, err
	}

	return CreateTaskOutput{
		ID:          task.ID,
		PipelineID:  task.PipelineID,
		ParentID:    task.ParentID,
		Owner:       task.Owner,
		TaskType:    task.TaskType,
		Status:      task.Status,
		ContractRef: task.ContractRef,
		Message:     messages.TaskCreatedSuccessfully,
	}, nil
}

func (t *taskService) GetTask(id int) (*domain.Task, error) {
	return t.repo.FindByID(id)
}

func (t *taskService) UpdateTask(id int, input UpdateTaskInput) error {
	task, err := t.repo.FindByID(id)
	if err != nil {
		return err
	}

	if input.PipelineID != nil {
		task.PipelineID = *input.PipelineID
	}
	if input.ParentID != nil {
		task.ParentID = *input.ParentID
	}
	if input.Owner != nil {
		task.Owner = *input.Owner
	}
	if input.TaskType != nil {
		task.TaskType = *input.TaskType
	}
	if input.Status != nil {
		task.Status = *input.Status
	}
	if input.ContractRef != nil {
		task.ContractRef = *input.ContractRef
	}

	return t.repo.Update(task)
}

func (t *taskService) DeleteTask(id int) error {
	return t.repo.Delete(id)
}

func (t *taskService) ListTasks() (ListTaskOutput, error) {
	tasks, err := t.repo.FindAll()
	if err != nil {
		return ListTaskOutput{}, err
	}

	return ListTaskOutput{
		Tasks:   tasks,
		Total:   len(tasks),
		Message: messages.TasksListedSuccessfully,
	}, nil
}
