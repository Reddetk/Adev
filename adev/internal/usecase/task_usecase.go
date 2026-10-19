package usecase

import "github.com/Reddetk/Adev/internal/domain"

type TaskUseCase interface {
	CreateTask(input CreateTaskInput) (CreateTaskOutput, error)
	GetTask(id int) (*domain.Task, error)
	UpdateTask(id int, input UpdateTaskInput) error
	DeleteTask(id int) error
	ListTasks() (ListTaskOutput, error)
}
