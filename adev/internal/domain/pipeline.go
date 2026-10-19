package domain

type Pipeline struct {
	ID             uint   `json:"id"`
	TargetSystemID int    `json:"targetsystemid" validate:"required,gte=0"`
	Route          string `json:"route" validate:"required"`
	Status         string `json:"status" validate:"required"`
	CurrentStage   string `json:"currentstage" validate:"required"`
}

func (p *Pipeline) Validate() error {
	if p.TargetSystemID < 0 {
		return ErrInvalidPipelineTargetSystemID
	}
	if p.Route == "" {
		return ErrInvalidPipelineRoute
	}
	if p.Status == "" {
		return ErrInvalidPipelineStatus
	}
	if p.CurrentStage == "" {
		return ErrInvalidPipelineCurrentStage
	}
	return nil
}
