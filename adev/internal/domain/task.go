package domain

type Task struct {
	ID          uint   `json:"id"`
	PipelineID  int    `json:"pipelineid" validate:"required,gte=0"`
	ParentID    int    `json:"parentid" validate:"required,gte=0"`
	Owner       string `json:"owner" validate:"required"`
	TaskType    string `json:"tasktype" validate:"required"`
	Status      string `json:"status" validate:"required"`
	ContractRef string `json:"contractref" validate:"required"`
}

func (t *Task) Validate() error {
	if t.PipelineID < 0 {
		return ErrInvalidTaskPipelineID
	}
	if t.ParentID < 0 {
		return ErrInvalidTaskParentID
	}
	if t.Owner == "" {
		return ErrInvalidTaskOwner
	}
	if t.TaskType == "" {
		return ErrInvalidTaskTaskType
	}
	if t.Status == "" {
		return ErrInvalidTaskStatus
	}
	if t.ContractRef == "" {
		return ErrInvalidTaskContractRef
	}
	return nil
}
