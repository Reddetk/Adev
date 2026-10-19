package worker

import (
	"encoding/json"
	"log"

	"github.com/Reddetk/Adev/internal/usecase"
)

type PipelineWorker struct {
	usecase usecase.PipelineUseCase
}

func NewPipelineWorker(uc usecase.PipelineUseCase) *PipelineWorker {
	return &PipelineWorker{usecase: uc}
}

func (w *PipelineWorker) ProcessPipelineJob(jobData []byte) error {
	var input usecase.CreatePipelineInput

	if err := json.Unmarshal(jobData, &input); err != nil {
		log.Printf("Failed to unmarshal job data: %v", err)
		return err
	}

	output, err := w.usecase.CreatePipeline(input)
	if err != nil {
		log.Printf("Failed to process pipeline job: %v", err)
		return err
	}

	log.Printf("Pipeline job completed: %v", output.Message)
	return nil
}

func (w *PipelineWorker) ProcessBatchPipelineJob(jobData []byte) error {
	var inputs []usecase.CreatePipelineInput

	if err := json.Unmarshal(jobData, &inputs); err != nil {
		log.Printf("Failed to unmarshal batch job data: %v", err)
		return err
	}

	for _, input := range inputs {
		if _, err := w.usecase.CreatePipeline(input); err != nil {
			log.Printf("Failed to process pipeline in batch: %v", err)
			// Continue processing other items
		}
	}

	log.Printf("Batch Pipeline job completed")
	return nil
}
