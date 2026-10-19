package worker

import (
	"encoding/json"
	"log"

	"github.com/Reddetk/Adev/internal/usecase"
)

type RunWorker struct {
	usecase usecase.RunUseCase
}

func NewRunWorker(uc usecase.RunUseCase) *RunWorker {
	return &RunWorker{usecase: uc}
}

func (w *RunWorker) ProcessRunJob(jobData []byte) error {
	var input usecase.CreateRunInput

	if err := json.Unmarshal(jobData, &input); err != nil {
		log.Printf("Failed to unmarshal job data: %v", err)
		return err
	}

	output, err := w.usecase.CreateRun(input)
	if err != nil {
		log.Printf("Failed to process run job: %v", err)
		return err
	}

	log.Printf("Run job completed: %v", output.Message)
	return nil
}

func (w *RunWorker) ProcessBatchRunJob(jobData []byte) error {
	var inputs []usecase.CreateRunInput

	if err := json.Unmarshal(jobData, &inputs); err != nil {
		log.Printf("Failed to unmarshal batch job data: %v", err)
		return err
	}

	for _, input := range inputs {
		if _, err := w.usecase.CreateRun(input); err != nil {
			log.Printf("Failed to process run in batch: %v", err)
			// Continue processing other items
		}
	}

	log.Printf("Batch Run job completed")
	return nil
}
