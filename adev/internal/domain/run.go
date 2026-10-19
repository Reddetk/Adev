package domain

type Run struct {
	ID           uint   `json:"id"`
	PipelineID   int    `json:"pipelineid" validate:"required,gte=0"`
	Stage        string `json:"stage" validate:"required"`
	Status       string `json:"status" validate:"required"`
	CheckpointID string `json:"checkpointid" validate:"required"`
	Attempts     int    `json:"attempts" validate:"required,gte=0"`
	LastError    string `json:"lasterror" validate:"required"`
}

func (r *Run) Validate() error {
	if r.PipelineID < 0 {
		return ErrInvalidRunPipelineID
	}
	if r.Stage == "" {
		return ErrInvalidRunStage
	}
	if r.Status == "" {
		return ErrInvalidRunStatus
	}
	if r.CheckpointID == "" {
		return ErrInvalidRunCheckpointID
	}
	if r.Attempts < 0 {
		return ErrInvalidRunAttempts
	}
	if r.LastError == "" {
		return ErrInvalidRunLastError
	}
	return nil
}
