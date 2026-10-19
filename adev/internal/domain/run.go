package domain

type Run struct {
	ID           uint   `json:"id"`
	PipelineID   int    `json:"pipelineid" validate:"required,gte=0"`
	Stage        string `json:"stage" validate:"required"`
	Status       string `json:"status" validate:"required"`
	CheckpointID string `json:"checkpointid" validate:"required"`
	Attempts     int    `json:"attempts" validate:"gte=0"`
	LastError    string `json:"lasterror"`
}

// Validate checks the run. A successful run has no error text, so LastError is optional.
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
	return nil
}
