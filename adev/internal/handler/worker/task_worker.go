package worker

import (
	"encoding/json"
	"log"

	"github.com/Reddetk/Adev/internal/usecase"
)

type TaskWorker struct {
	usecase usecase.TaskUseCase
}

func NewTaskWorker(uc usecase.TaskUseCase) *TaskWorker {
	return &TaskWorker{usecase: uc}
}

func (w *TaskWorker) ProcessTaskJob(jobData []byte) error {
	var input usecase.CreateTaskInput

	if err := json.Unmarshal(jobData, &input); err != nil {
		log.Printf("Failed to unmarshal job data: %v", err)
		return err
	}

	output, err := w.usecase.CreateTask(input)
	if err != nil {
		log.Printf("Failed to process task job: %v", err)
		return err
	}

	log.Printf("Task job completed: %v", output.Message)
	return nil
}

func (w *TaskWorker) ProcessBatchTaskJob(jobData []byte) error {
	var inputs []usecase.CreateTaskInput

	if err := json.Unmarshal(jobData, &inputs); err != nil {
		log.Printf("Failed to unmarshal batch job data: %v", err)
		return err
	}

	for _, input := range inputs {
		if _, err := w.usecase.CreateTask(input); err != nil {
			log.Printf("Failed to process task in batch: %v", err)
			// Continue processing other items
		}
	}

	log.Printf("Batch Task job completed")
	return nil
}
