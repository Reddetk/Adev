package usecase

import (
	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/messages"
)

type runService struct {
	repo RunRepository
}

func NewRunService(repo RunRepository) RunUseCase {
	return &runService{repo: repo}
}

func (r *runService) CreateRun(input CreateRunInput) (CreateRunOutput, error) {
	run := domain.Run{
		PipelineID:   input.PipelineID,
		Stage:        input.Stage,
		Status:       input.Status,
		CheckpointID: input.CheckpointID,
		Attempts:     input.Attempts,
		LastError:    input.LastError,
	}

	if err := run.Validate(); err != nil {
		return CreateRunOutput{}, err
	}

	if err := r.repo.Save(&run); err != nil {
		return CreateRunOutput{}, err
	}

	return CreateRunOutput{
		ID:           run.ID,
		PipelineID:   run.PipelineID,
		Stage:        run.Stage,
		Status:       run.Status,
		CheckpointID: run.CheckpointID,
		Attempts:     run.Attempts,
		LastError:    run.LastError,
		Message:      messages.RunCreatedSuccessfully,
	}, nil
}

func (r *runService) GetRun(id int) (*domain.Run, error) {
	return r.repo.FindByID(id)
}

func (r *runService) UpdateRun(id int, input UpdateRunInput) error {
	run, err := r.repo.FindByID(id)
	if err != nil {
		return err
	}

	if input.PipelineID != nil {
		run.PipelineID = *input.PipelineID
	}
	if input.Stage != nil {
		run.Stage = *input.Stage
	}
	if input.Status != nil {
		run.Status = *input.Status
	}
	if input.CheckpointID != nil {
		run.CheckpointID = *input.CheckpointID
	}
	if input.Attempts != nil {
		run.Attempts = *input.Attempts
	}
	if input.LastError != nil {
		run.LastError = *input.LastError
	}

	return r.repo.Update(run)
}

func (r *runService) DeleteRun(id int) error {
	return r.repo.Delete(id)
}

func (r *runService) ListRuns() (ListRunOutput, error) {
	runs, err := r.repo.FindAll()
	if err != nil {
		return ListRunOutput{}, err
	}

	return ListRunOutput{
		Runs:    runs,
		Total:   len(runs),
		Message: messages.RunsListedSuccessfully,
	}, nil
}
